<script lang="ts">
	import { untrack } from 'svelte';
	import { cn } from '$lib/components/ui/cn.js';
	import { formatDuration } from '$lib/format-date';
	import type { components } from '$lib/api/v1';

	type LogEntry = components['schemas']['LogEntry'];

	interface Props {
		deploymentId: string;
		/**
		 * What the caller already knows about this deployment, when it knows
		 * anything: "in_progress", "success" or "failed".
		 *
		 * Opening a deployment replays its whole log as a catch-up, so without this
		 * a finished run arrived looking live — phases walking forward and an active
		 * status dot — and only corrected itself when the replay reached the end.
		 * Every caller has the answer sitting next to the id it passes; this is only
		 * a matter of handing it over.
		 *
		 * Omit it when genuinely unknown, and the component assumes a live run,
		 * which is the safe guess for a deployment that was just started.
		 */
		deploymentStatus?: string;
		onDone?: (status: string) => void;
		/**
		 * Drops the card's own border and radius so it can sit inside another card
		 * as its bottom section — the deployment header and its output are one
		 * object, and two nested outlines say otherwise.
		 */
		flush?: boolean;
		/**
		 * The output *is* the surface it sits on (the stacked deployment panel), so
		 * it takes the full height instead of the 288px slab it uses when it is one
		 * section of a card.
		 */
		fill?: boolean;
		/**
		 * The run's duration in seconds — live while it streams, frozen on its own
		 * last log once it ends. Bindable because the clock can only live here: the
		 * first and last log lines are what it is measured between, and nothing
		 * outside this component sees them. The panel around it is where the
		 * duration is read.
		 */
		elapsedSeconds?: number;
	}

	let {
		deploymentId,
		deploymentStatus,
		onDone,
		flush = false,
		fill = false,
		elapsedSeconds = $bindable(0)
	}: Props = $props();

	// Whether the run being streamed had already finished before this subscription
	// opened. Set once per subscription, and read by the log handler to keep a
	// replay from narrating itself as progress. A plain variable rather than
	// state: it changes only when the stream is rebuilt, and the handlers that
	// read it belong to that same stream.
	let replaying = false;

	let logs: LogEntry[] = $state([]);
	let status: string = $state('in_progress');
	let streamError: string = $state('');

	const phaseLabels: Record<string, string> = {
		connect: 'Connecting to Server',
		pull_image: 'Pulling Image',
		proxy_setup: 'Setting Up Reverse Proxy',
		stop_container: 'Stopping Existing Container',
		start_container: 'Starting Application',
		tls_cert: 'Waiting for TLS Certificates',
		complete: 'Deployment Logs',
		failed: 'Deployment Failed'
	};

	let currentPhase: string = $state('Starting...');
	let currentSubtext: string = $state('');
	let lastErrorReason: string = $state('');
	// The deployment's own clock, set from its first log rather than from mount.
	// Reopening a finished deployment used to start counting at the moment you
	// opened it, so a day-old success reported "1414m 44s" — the age of the
	// deployment, not how long it took.
	let startTime: number | null = $state(null);
	let lastLogTime: number = $state(Date.now());
	let timerInterval: ReturnType<typeof setInterval> | null = null;

	function updatePhaseFromLog(log: LogEntry) {
		if (log.type === 'stderr' && log.phase !== 'failed' && !log.output.startsWith('command failed: ')) {
			lastErrorReason = log.output;
		}

		const logTime = new Date(log.created_at).getTime();
		startTime ??= logTime;
		lastLogTime = logTime;

		// The timestamps above are still wanted while replaying — they are what the
		// final duration is measured from. The phase is not: walking the header
		// through a run that ended days ago is the whole of what made a replay look
		// like a deployment.
		if (replaying) return;

		if (!log.phase) return;

		const label = phaseLabels[log.phase];
		if (!label) return;

		currentPhase = label;
		currentSubtext = log.output;
	}

	function startTimer() {
		if (timerInterval) return;
		timerInterval = setInterval(() => {
			if (startTime === null) return;
			elapsedSeconds = Math.max(0, Math.floor((Date.now() - startTime) / 1000));
		}, 1000);
	}

	// Freeze on the run's own duration, not on however long ago it ran.
	function freezeElapsed() {
		stopTimer();
		if (startTime !== null) {
			elapsedSeconds = Math.max(0, Math.floor((lastLogTime - startTime) / 1000));
		}
	}

	function stopTimer() {
		if (timerInterval) {
			clearInterval(timerInterval);
			timerInterval = null;
		}
	}

	function formatLogTimestamp(timestamp: string): string {
		const date = new Date(timestamp);
		if (Number.isNaN(date.getTime())) return '';
		const pad = (value: number) => String(value).padStart(2, '0');
		return `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}.${String(date.getMilliseconds()).padStart(3, '0')}`;
	}

	let bannerStatus: 'active' | 'success' | 'error' = $derived.by(() => {
		if (status === 'success') return 'success';
		if (status === 'failed') return 'error';
		return 'active';
	});

	// Follow the tail while streaming, but only if the reader is already at the
	// bottom — yanking the view back down while someone is reading an earlier
	// error is worse than not following at all.
	let logScroller = $state<HTMLDivElement | null>(null);
	$effect(() => {
		const count = logs.length;
		const el = logScroller;
		if (!el || count === 0) return;
		const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 40;
		if (nearBottom) el.scrollTop = el.scrollHeight;
	});

	/**
	 * One stream per deployment, torn down and reopened when the id changes.
	 *
	 * This used to be onMount, which runs once — but the service panel reuses this
	 * component across deploys rather than remounting it, so a second deploy went
	 * on listening to the first. That stream had already closed, so the output
	 * stayed frozen on the previous run's last line, `done` never fired again, and
	 * the header above it sat on "in progress" while the log underneath said
	 * Complete.
	 */
	$effect(() => {
		const id = deploymentId;

		// untracked: only the id may rebuild the stream. The status changes from
		// in_progress to success while a live deploy is being watched, and reading
		// it normally would tear down and re-open the very stream that reported it.
		const known = untrack(() => deploymentStatus);
		replaying = known === 'success' || known === 'failed';

		// None of this may survive into the next deployment: a phase left over from
		// the last run, or its clock still counting, reads as progress on this one.
		logs = [];
		streamError = '';
		currentSubtext = '';
		lastErrorReason = '';
		startTime = null;
		lastLogTime = Date.now();
		elapsedSeconds = 0;

		// Seeded from what is already known rather than assumed live and corrected
		// at the end of the replay. The correction is what was visible: a finished
		// deploy opened with a pulsing dot on "Starting..." and ran through every
		// phase it had been through days ago.
		status = replaying ? known! : 'in_progress';
		currentPhase = replaying
			? (phaseLabels[known === 'failed' ? 'failed' : 'complete'] ?? 'Starting...')
			: 'Starting...';

		// No clock for a run that already stopped: it counts from the deployment's
		// first log to *now*, which is how a two-day-old deploy once reported
		// "1414m 44s". The real duration arrives with freezeElapsed on `done`.
		if (!replaying) startTimer();

		// Local to this run, so the handlers below close the stream they belong to
		// rather than whichever one is current by the time they fire.
		const source = new EventSource(`/api/deployments/${id}/logs`);

		source.onmessage = (e) => {
			const log: LogEntry = JSON.parse(e.data);
			logs = [...logs, log];
			updatePhaseFromLog(log);
		};

		source.addEventListener('done', (e) => {
			status = (e as MessageEvent).data;
			if (status === 'success') {
				currentPhase = 'Deployment Logs';
				currentSubtext = 'deployment success';
			} else if (status === 'failed') {
				currentPhase = 'Deployment Failed';
				if (lastErrorReason) {
					currentSubtext = lastErrorReason;
				}
			}
			freezeElapsed();
			onDone?.(status);
			source.close();
		});

		source.addEventListener('stream-error', (e) => {
			const data = JSON.parse((e as MessageEvent).data);
			streamError = data.message;
			status = 'failed';
			currentPhase = 'Deployment Failed';
			if (lastErrorReason) {
				currentSubtext = lastErrorReason;
			}
			freezeElapsed();
			onDone?.('failed');
			source.close();
		});

		return () => {
			stopTimer();
			source.close();
		};
	});
</script>

<!-- One surface, not two. The status row and the output belong to the same
     deployment, so stacking a tinted banner on top of a separate black slab was
     saying it twice and spending two surfaces to do it. -->
<div
	class={cn(
		'overflow-hidden',
		flush ? 'border-t border-border' : 'rounded-lg border border-border',
		fill && 'flex h-full min-h-0 flex-col'
	)}
>
	<div class="flex w-full items-center gap-2 px-3 py-2.5">
		{#if bannerStatus !== 'success'}
			<span
				class={cn(
					'h-2 w-2 flex-none rounded-full bg-muted-foreground',
					bannerStatus === 'error' && 'bg-destructive',
					bannerStatus === 'active' && 'motion-safe:animate-pulse'
				)}
				aria-hidden="true"
			></span>
		{/if}
		<span
			class={cn(
				'min-w-0 flex-1 truncate text-[14px] font-medium text-foreground',
				bannerStatus === 'error' && 'text-destructive'
			)}
		>
			{currentPhase}
		</span>
		<span class="flex-none text-[13px] text-muted-foreground tabular-nums">
			{formatDuration(elapsedSeconds)}
		</span>
	</div>

	{#if currentSubtext && bannerStatus !== 'success'}
		<p class="truncate border-t border-border px-3 py-2 text-[13px] text-muted-foreground">
			{currentSubtext}
		</p>
	{/if}
	{#if streamError}
		<p class="border-t border-border px-3 py-2 text-[13px] text-destructive">{streamError}</p>
	{/if}

	<!-- Vercel-style rows: fixed timestamp column, preformatted output, and a
	     restrained red surface for stderr rather than a terminal treatment.

	     Lines wrap rather than run off the side. The panel is one inspector wide,
	     so a docker pull line or a stack trace put the whole log behind a
	     horizontal scrollbar — and the timestamp column scrolled away with it,
	     which is exactly the thing you want to keep in view while reading. -->
	<div
		bind:this={logScroller}
		class={cn(
			'max-h-72 overflow-y-auto border-t border-border bg-card font-mono text-xs leading-5 text-foreground sm:text-sm',
			fill && 'max-h-none min-h-0 flex-1'
		)}
		role="log"
		aria-live="off"
		aria-label="Deployment output"
	>
		{#each logs as log (log.order)}
			<div
				class={cn(
					'flex w-full cursor-default border-l-2 border-l-transparent text-foreground select-text hover:bg-accent',
					log.type === 'stderr' && 'bg-destructive/10 text-destructive hover:bg-destructive/15'
				)}
			>
				<time
					class="relative inline-flex w-28 flex-none items-start overflow-hidden py-0.5 pl-2 whitespace-nowrap text-muted-foreground tabular-nums select-none sm:w-32 sm:pl-4"
					datetime={log.created_at}
				>
					{formatLogTimestamp(log.created_at)}
				</time>
				<div class="flex min-w-0 flex-1 flex-col">
					<!-- pre-wrap, not pre: the log's own spacing and indentation is kept,
					     but a line that outruns the panel breaks instead of widening it.
					     wrap-anywhere on top of that, because the lines that overflow are
					     usually one unbroken token — an image digest, a URL, a container
					     id — with no space to break at. -->
					<p class="m-0 py-0.5 pr-3 pl-1 wrap-anywhere whitespace-pre-wrap sm:pr-6 sm:pl-3">
						{log.output}
					</p>
				</div>
			</div>
		{/each}
	</div>
</div>
