<script lang="ts">
	import { Check, CircleCheck, CircleX, Copy, LoaderCircle, Search } from 'lucide-svelte';
	import { untrack } from 'svelte';
	import { dev } from '$app/environment';
	import { cn } from '$lib/components/ui/cn.js';
	import IconButton from '$lib/components/ui/IconButton.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import { formatDuration } from '$lib/format-date';
	import type { components } from '$lib/api/v1';

	type LogEntry = components['schemas']['LogEntry'];
	type BannerStatus = 'active' | 'success' | 'error';
	type CompactState = {
		status: BannerStatus;
		title: string;
		detail: string;
		elapsedSeconds: number;
	};

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
		compact?: boolean;
		previewState?: BannerStatus;
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
		compact = false,
		previewState,
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

	function formatLogTimestamp(timestamp: string): string {
		const date = new Date(timestamp);
		if (Number.isNaN(date.getTime())) return '';
		const pad = (value: number) => String(value).padStart(2, '0');
		return `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}.${String(date.getMilliseconds()).padStart(3, '0')}`;
	}

	let bannerStatus: BannerStatus = $derived.by(() => {
		if (status === 'success') return 'success';
		if (status === 'failed') return 'error';
		return 'active';
	});
	let compactTitle = $derived(
		bannerStatus === 'active'
			? 'Deployment in progress'
			: bannerStatus === 'success'
				? 'Deployment successful'
				: 'Deployment failed'
	);
	let compactDetail = $derived.by(() => {
		if (bannerStatus === 'active') {
			return currentPhase === 'Starting...' ? 'Preparing deployment...' : currentPhase;
		}
		if (bannerStatus === 'success') return 'Service is live.';
		return currentSubtext || streamError || 'Open logs to review the failure.';
	});
	let showStatePreview = $derived(dev && previewState !== undefined);
	const statePreview: Record<BannerStatus, CompactState> = {
		active: {
			status: 'active',
			title: 'Deployment in progress',
			detail: 'Pulling Image',
			elapsedSeconds: 8
		},
		error: {
			status: 'error',
			title: 'Deployment failed',
			detail: 'Rolling proxy is not ready; upgrade it from the Servers page.',
			elapsedSeconds: 9
		},
		success: {
			status: 'success',
			title: 'Deployment successful',
			detail: 'Service is live.',
			elapsedSeconds: 12
		}
	};
	let compactState: CompactState = $derived.by(() => {
		if (showStatePreview && previewState) return statePreview[previewState];
		return {
			status: bannerStatus,
			title: compactTitle,
			detail: compactDetail,
			elapsedSeconds
		};
	});
	let query = $state('');
	let visibleLogs = $derived.by(() => {
		const normalizedQuery = query.trim().toLowerCase();
		if (!normalizedQuery) return logs;
		return logs.filter((log) => log.output.toLowerCase().includes(normalizedQuery));
	});
	let filtering = $derived(query.trim() !== '');
	let searchInput = $state<HTMLInputElement | null>(null);
	let copied = $state(false);

	async function copyLogs() {
		await navigator.clipboard.writeText(
			logs.map((log) => `${formatLogTimestamp(log.created_at)} ${log.output}`).join('\n')
		);
		copied = true;
		setTimeout(() => (copied = false), 2000);
	}

	function focusLogSearch(event: KeyboardEvent) {
		if (compact) return;
		if (event.key.toLowerCase() !== 'f' || !(event.metaKey || event.ctrlKey)) return;
		event.preventDefault();
		searchInput?.focus();
		searchInput?.select();
	}

	// Follow the tail while streaming, but only if the reader is already at the
	// bottom — yanking the view back down while someone is reading an earlier
	// error is worse than not following at all.
	let logScroller = $state<HTMLDivElement | null>(null);
	$effect(() => {
		const count = visibleLogs.length;
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
		if (showStatePreview && compact) return;

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

<svelte:window onkeydown={focusLogSearch} />

{#snippet compactBanner(state: CompactState, attached: boolean, hidden: boolean, announce: boolean)}
	<div
		class={cn(
			'flex items-center gap-3 px-3 py-3',
			hidden && 'hidden',
			attached ? 'border-t' : 'rounded-lg border',
			state.status === 'active' && 'border-[#5b709c]/25 bg-[#f5f7fb] text-[#5b709c]',
			state.status === 'success' && 'border-[#43946e]/25 bg-[#f4faf8] text-[#43946e]',
			state.status === 'error' && 'border-[#a65353]/20 bg-[#fdf7f7] text-[#a65353]'
		)}
		role={announce ? 'status' : undefined}
		aria-live={announce ? 'polite' : undefined}
		aria-atomic={announce ? 'true' : undefined}
		data-deployment-state={state.status}
	>
		{#if state.status === 'success'}
			<CircleCheck class="h-5 w-5 flex-none text-[#43946e]" strokeWidth={1.75} aria-hidden="true" />
		{:else if state.status === 'active'}
			<LoaderCircle
				class="h-5 w-5 flex-none animate-[spin_2s_linear_infinite] text-[#5b709c]"
				strokeWidth={1.75}
				aria-hidden="true"
			/>
		{:else}
			<CircleX class="h-5 w-5 flex-none text-[#a65353]" strokeWidth={1.75} aria-hidden="true" />
		{/if}
		<div class="min-w-0 flex-1">
			<p class="truncate text-[14px] font-medium">{state.title}</p>
			{#if state.status !== 'success'}
				<p
					class={cn(
						'mt-0.5 truncate text-[13px]',
						state.status === 'active' && 'text-[#5b709c]/80',
						state.status === 'error' && 'text-[#a65353]/80'
					)}
				>
					{state.detail}
				</p>
			{/if}
		</div>
		{#if state.status !== 'success'}
			<span class="flex-none text-[13px] tabular-nums opacity-75">
				{formatDuration(state.elapsedSeconds)}
			</span>
		{/if}
	</div>
{/snippet}

{@render compactBanner(compactState, flush, !compact, !showStatePreview)}

<!-- One surface, not two. The status row and the output belong to the same
     deployment, so stacking a tinted banner on top of a separate black slab was
     saying it twice and spending two surfaces to do it. -->
<div
	class={cn(
		'overflow-hidden',
		compact && 'hidden',
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
					bannerStatus === 'active' && 'animate-pulse'
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
	<div class="flex items-center gap-2 border-t border-border px-3 py-2">
		<IconButton
			variant="ghost"
			size="sm"
			disabled={logs.length === 0}
			aria-label={copied ? 'Deployment logs copied' : 'Copy deployment logs'}
			title={copied ? 'Copied' : 'Copy deployment logs'}
			onclick={copyLogs}
		>
			{#if copied}
				<Check class="size-3.5" aria-hidden="true" />
			{:else}
				<Copy class="size-3.5" aria-hidden="true" />
			{/if}
		</IconButton>
		<span class="flex-none text-[13px] text-muted-foreground tabular-nums">
			{#if filtering}
				{visibleLogs.length.toLocaleString()} of {logs.length.toLocaleString()} lines
			{:else}
				{logs.length.toLocaleString()} {logs.length === 1 ? 'line' : 'lines'}
			{/if}
		</span>
		<div class="relative ml-auto min-w-0 flex-1 sm:max-w-72">
			<Search
				class="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-muted-foreground"
				aria-hidden="true"
			/>
			<Input
				type="search"
				size="sm"
				bind:value={query}
				bind:ref={searchInput}
				placeholder="Find in logs"
				aria-label="Find in deployment logs"
				class="h-8 min-w-0 pr-10 pl-8"
			/>
			<kbd
				class="pointer-events-none absolute top-1/2 right-2 -translate-y-1/2 rounded border border-border bg-card px-1 py-0.5 font-sans text-[10px] text-muted-foreground"
			>
				⌘F
			</kbd>
		</div>
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
		{#if visibleLogs.length === 0 && filtering}
			<p class="px-4 py-3 font-sans text-[13px] text-muted-foreground">No matching log lines.</p>
		{:else}
			{#each visibleLogs as log (log.order)}
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
		{/if}
	</div>
</div>
