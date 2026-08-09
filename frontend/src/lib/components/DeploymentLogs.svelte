<script lang="ts">
	import { untrack } from 'svelte';
	import { cn } from '$lib/components/ui/cn.js';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { ChevronDown } from '@steeze-ui/heroicons';
	import type { components } from '$lib/api/v1';

	type LogEntry = components['schemas']['LogEntry'];

	interface Props {
		deploymentId: string;
		/**
		 * What the caller already knows about this deployment, when it knows
		 * anything: "in_progress", "success" or "failed".
		 *
		 * Opening a deployment replays its whole log as a catch-up, so without this
		 * a finished run arrived looking live — phases walking forward, an active
		 * status dot, the panel springing open — and only corrected itself when the
		 * replay reached the end. Every caller has the answer sitting next to the id
		 * it passes; this is only a matter of handing it over.
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
		 * section of a card, and it never auto-collapses — reading it is the only
		 * reason that panel is open.
		 */
		fill?: boolean;
	}

	let { deploymentId, deploymentStatus, onDone, flush = false, fill = false }: Props = $props();

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
		complete: 'Deployment Complete',
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
	let elapsedSeconds: number = $state(0);
	let timerInterval: ReturnType<typeof setInterval> | null = null;

	function updatePhaseFromLog(log: LogEntry) {
		if (
			log.type === 'stderr' &&
			log.phase !== 'failed' &&
			!log.output.startsWith('command failed: ')
		) {
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

	function formatElapsed(seconds: number): string {
		if (seconds < 60) return `${seconds}s`;
		const m = Math.floor(seconds / 60);
		const s = seconds % 60;
		return `${m}m ${s}s`;
	}

	let bannerStatus: 'active' | 'success' | 'error' = $derived.by(() => {
		if (status === 'success') return 'success';
		if (status === 'failed') return 'error';
		return 'active';
	});

	// Open while it matters, closed once it doesn't. A finished successful deploy
	// is the one case where nobody reads the output, and leaving 300px of log on
	// screen for it is most of what made this panel feel loud. Failures and
	// in-flight runs stay open — those you do need to read.
	let open = $state(true);
	let userToggled = $state(false);
	$effect(() => {
		if (status === 'success' && !userToggled && !fill) open = false;
	});

	function toggle() {
		userToggled = true;
		open = !open;
	}

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
		userToggled = false;

		// Seeded from what is already known rather than assumed live and corrected
		// at the end of the replay. The correction is what was visible: a finished
		// deploy opened with a pulsing dot on "Starting...", ran through every phase
		// it had been through days ago, then snapped shut.
		status = replaying ? known! : 'in_progress';
		currentPhase = replaying
			? (phaseLabels[known === 'failed' ? 'failed' : 'complete'] ?? 'Starting...')
			: 'Starting...';
		// A live run is worth watching even if the reader collapsed the last one; a
		// finished successful one is the case nobody reads, which is what the
		// auto-collapse below decides anyway — it just used to decide it a beat late,
		// after the panel had already sprung open.
		open = fill || !replaying || known !== 'success';

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
				currentPhase = 'Deployment Complete';
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
     saying it twice and spending two surfaces to do it. The card stays neutral;
     only the status mark and its label carry colour. -->
<div
	class={cn(
		'overflow-hidden',
		flush ? 'border-t border-border' : 'rounded-lg border border-border',
		fill && 'flex h-full min-h-0 flex-col'
	)}
>
	<button
		type="button"
		onclick={toggle}
		aria-expanded={open}
		class="flex w-full cursor-pointer items-center gap-2 px-3 py-2.5 text-left transition-colors hover:bg-accent/60 focus-visible:bg-accent focus-visible:outline-none"
	>
		<span class="status-mark" data-status={bannerStatus} aria-hidden="true"></span>
		<span
			class={cn(
				'min-w-0 flex-1 truncate text-[15px] font-medium',
				bannerStatus === 'success' && 'text-success',
				bannerStatus === 'error' && 'text-destructive',
				bannerStatus === 'active' && 'text-foreground'
			)}
		>
			{currentPhase}
		</span>
		<span class="flex-none text-[13px] text-muted-foreground tabular-nums">
			{formatElapsed(elapsedSeconds)}
		</span>
		<Icon
			src={ChevronDown}
			theme="outline"
			class="h-3.5 w-3.5 flex-none text-muted-foreground transition-transform duration-150 {open
				? 'rotate-180'
				: ''}"
		/>
	</button>

	{#if currentSubtext && bannerStatus !== 'success'}
		<p class="truncate border-t border-border px-3 py-2 text-[13px] text-muted-foreground">
			{currentSubtext}
		</p>
	{/if}
	{#if streamError}
		<p class="border-t border-border px-3 py-2 text-[13px] text-destructive">{streamError}</p>
	{/if}

	{#if open}
		<!-- Light, not a terminal. #171717 on a near-white achromatic panel read as
		     a foreign object dropped into the page; --muted keeps it a surface of
		     this design system and lets the one red carry real meaning. -->
		<div bind:this={logScroller} class="log-output" class:fill>
			{#each logs as log (log.order)}
				<p class:err={log.type === 'stderr'}>{log.output}</p>
			{/each}
		</div>
	{/if}
</div>

<style>
	.status-mark {
		flex: none;
		width: 0.5rem;
		height: 0.5rem;
		border-radius: 50%;
		background: var(--muted-foreground);
	}

	.status-mark[data-status='success'] {
		background: var(--success);
	}

	.status-mark[data-status='error'] {
		background: var(--destructive);
	}

	/* Only the running state pulses: motion here means "still working", so a dot
	   that keeps breathing after the deploy has landed would be lying. */
	.status-mark[data-status='active'] {
		animation: status-pulse 1.6s ease-in-out infinite;
	}

	@keyframes status-pulse {
		0%,
		100% {
			opacity: 1;
		}
		50% {
			opacity: 0.35;
		}
	}

	.log-output {
		max-height: 18rem;
		overflow-y: auto;
		border-top: 1px solid var(--border);
		background: var(--muted);
		padding: 0.625rem 0.75rem;
		font-family: var(--font-mono, ui-monospace, monospace);
		font-size: 0.75rem;
		line-height: 1.55;
		color: var(--foreground);
		/* Wrap instead of scrolling sideways: image digests are one unbroken 71-char
		   token, and in a 420px panel they were forcing a horizontal scrollbar under
		   every log view. `anywhere` is what actually breaks them; `break-word`
		   leaves an overflowing token alone. */
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}

	/* Inside the stacked panel the output has the room, so it takes it: the cap
	   comes off and the scroll happens over the full height instead of in a
	   288px window with the panel empty underneath it. */
	.log-output.fill {
		max-height: none;
		flex: 1 1 auto;
		min-height: 0;
	}

	.log-output p {
		margin: 0;
	}

	.log-output p.err {
		color: var(--destructive);
	}
</style>
