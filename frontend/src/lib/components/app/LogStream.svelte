<script lang="ts">
	import { CommandLine, ExclamationTriangle } from '@steeze-ui/heroicons';
	import { cn } from '$lib/components/ui/cn.js';
	import Button from '$lib/components/ui/Button.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';

	interface Props {
		/** SSE endpoint, without a query string — this adds `since` itself. */
		endpoint: string;
		/**
		 * What is doing the writing, for the sentences the panel says when there is
		 * nothing to read. "container" for a service, "proxy" for Traefik.
		 */
		subject?: string;
		class?: string;
	}

	let { endpoint, subject = 'container', class: className }: Props = $props();

	type Line = { output: string; type: string };

	/**
	 * How far back Docker replays before the stream catches up to live. 'all' is
	 * the absence of a range rather than a value the API knows, so it is the one
	 * option that sends no parameter — the endpoint's own default.
	 */
	const ranges = [
		{ value: 'all', label: 'All time' },
		{ value: '1h', label: 'Last hour' },
		{ value: '6h', label: 'Last 6 hours' },
		{ value: '24h', label: 'Last 24 hours' },
		{ value: '7d', label: 'Last 7 days' },
		{ value: '30d', label: 'Last 30 days' }
	];

	let range = $state('all');
	// Client-side, because the whole window is already here: 2000 lines is nothing
	// to scan per keystroke, and going back to the server would mean tearing down
	// an SSH session to answer a typo.
	let query = $state('');

	/**
	 * Four states, not two. "Streaming or not" collapsed a container that is
	 * quiet, a container that stopped, and a server nobody can reach into the
	 * same grey sentence, and those want different words and different buttons.
	 */
	type Phase = 'connecting' | 'live' | 'ended' | 'failed';

	/**
	 * A deployment's log is finite; this one is not. Somebody leaves the tab open
	 * on a chatty container and the array is the only thing that grows, so it
	 * keeps a window rather than the whole history — Docker still has the rest.
	 */
	const maxLines = 2000;

	let lines = $state<Line[]>([]);
	let streamError = $state('');
	let phase = $state<Phase>('connecting');
	let streaming = $derived(phase === 'live');
	let eventSource: EventSource | null = null;
	// Reconnecting while the previous attempt's explanation is still in flight
	// would otherwise let a stale message land on top of the new one.
	let attempt = 0;

	// Closing the socket is not a phase: whoever closes it says why, one line
	// later, and the reader needs that reason more than they need "stopped".
	function disconnect() {
		eventSource?.close();
		eventSource = null;
	}

	// 'all' is the endpoint's default, so it is expressed by not asking.
	const logUrl = (base: string, since: string) =>
		since === 'all' ? base : `${base}?since=${encodeURIComponent(since)}`;

	/**
	 * EventSource hides the response of a failed open, so the reason the backend
	 * took the trouble to send — an unreachable server, no Docker — never reaches
	 * the reader. Ask the endpoint once more, plainly, just to read it.
	 */
	async function explainOpenFailure(url: string, forAttempt: number) {
		const abort = new AbortController();
		try {
			const res = await fetch(url, { signal: abort.signal });
			// It opened this time: a blip, not a broken server. Leave the generic
			// message and let the reader hit Reconnect.
			if (res.ok) return;
			const { error } = await res.json();
			if (error && forAttempt === attempt) streamError = error;
		} catch {
			// The API itself is unreachable; the generic message already says so.
		} finally {
			// Never leave the second request streaming: it is a whole SSH session.
			abort.abort();
		}
	}

	function connect(base: string, since: string) {
		disconnect();
		lines = [];
		streamError = '';
		phase = 'connecting';
		attempt += 1;

		const url = logUrl(base, since);
		const es = new EventSource(url);
		eventSource = es;

		es.onopen = () => {
			phase = 'live';
		};

		es.onmessage = (e) => {
			const line: Line = JSON.parse(e.data);
			lines = [...lines, line].slice(-maxLines);
		};

		es.addEventListener('done', () => {
			phase = 'ended';
			streamError = `The ${subject} stopped writing output.`;
			disconnect();
		});

		es.addEventListener('stream-error', (e) => {
			phase = 'failed';
			streamError = JSON.parse((e as MessageEvent).data).message;
			disconnect();
		});

		// EventSource reconnects by itself, which is wrong here: the endpoint
		// answers 400 for a service that was never deployed and 502 for an
		// unreachable server, and retrying either just hammers it. Close and let
		// the reader decide when to try again.
		es.onerror = () => {
			const wasLive = streaming;
			disconnect();
			// 'done' and 'stream-error' land first and say something specific;
			// the socket closing behind them is not news.
			if (streamError) return;
			phase = 'failed';
			if (wasLive) {
				streamError = 'The connection dropped while the stream was running.';
				return;
			}
			streamError = 'Could not open the log stream.';
			explainOpenFailure(url, attempt);
		};
	}

	// Changing the range reconnects: how far back Docker replays is decided when
	// the stream opens, so it is not something the open stream can be asked for.
	$effect(() => {
		connect(endpoint, range);
		return disconnect;
	});

	// Case-insensitive substring, not a regex: people type `error` and `500`, and
	// a half-typed regex would throw on the way to the one they meant.
	let visible = $derived.by(() => {
		const q = query.trim().toLowerCase();
		if (!q) return lines;
		return lines.filter((l) => l.output.toLowerCase().includes(q));
	});
	let filtering = $derived(query.trim() !== '');

	// Follow the tail only from the tail. Same rule as the deployment log: pulling
	// someone back down while they are reading something further up is worse than
	// not following.
	let scroller = $state<HTMLDivElement | null>(null);
	$effect(() => {
		const count = visible.length;
		const el = scroller;
		if (!el || count === 0) return;
		const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 40;
		if (nearBottom) el.scrollTop = el.scrollHeight;
	});

	const statusLabel = $derived(
		{
			connecting: 'Connecting…',
			live: 'Streaming live output',
			ended: 'Stream ended',
			failed: 'Not streaming'
		}[phase]
	);

	/**
	 * With nothing to read, the panel is the message. A stopped container and an
	 * unreachable server are the two everyone actually hits, and neither is
	 * served by the word "empty" — so each says what happened and offers the one
	 * button that can change it.
	 */
	const empty = $derived(
		// A narrowed range that came back empty is not the same story as a quiet
		// container: nothing is wrong, the window is just too small, and the fix is
		// the control directly above rather than the Try again button.
		range !== 'all' && (phase === 'live' || phase === 'ended')
			? {
					icon: CommandLine,
					title: 'No logs in this time range',
					description: 'Nothing was written in the range you picked. Widen it to look further back.'
				}
			: {
					connecting: {
						icon: CommandLine,
						title: 'Opening the log stream',
						description: `Connecting to the server over SSH to follow the ${subject}. It takes a moment.`
					},
					live: {
						icon: CommandLine,
						title: 'No output yet',
						description: `The ${subject} has not written anything since the stream opened. Lines show up here the moment it does.`
					},
					ended: {
						icon: CommandLine,
						title: 'No logs to show',
						description: `The stream ended before any output arrived — the ${subject} may have stopped.`
					},
					failed: {
						icon: ExclamationTriangle,
						title: 'Can’t reach the logs',
						description: streamError
					}
				}[phase]
	);

	// Only a stopped stream is worth a button; connecting and live are already
	// doing the only thing that would happen if you pressed one. An empty range is
	// the exception: nothing failed, and reconnecting would find nothing again —
	// the range control above it is the button that helps.
	const canRetry = $derived(phase === 'failed' || (phase === 'ended' && range === 'all'));
</script>

<div
	class={cn(
		'flex h-full min-h-0 flex-col overflow-hidden rounded-lg border border-border',
		className
	)}
>
	<div class="flex flex-none items-center gap-2.5 border-b border-border px-3 py-2.5">
		<span class="status-mark" data-phase={phase} aria-hidden="true"></span>
		<span role="status" class="min-w-0 flex-1 truncate text-[13px] text-muted-foreground">
			{statusLabel}
		</span>
		{#if lines.length > 0}
			<!-- While filtering the count is the answer to what was typed, so it counts
			     matches and keeps the total for scale. -->
			<span class="flex-none text-[12px] text-muted-foreground tabular-nums">
				{#if filtering}
					{visible.length.toLocaleString()} of {lines.length.toLocaleString()}
				{:else}
					{lines.length.toLocaleString()}
					{lines.length === 1 ? 'line' : 'lines'}
				{/if}
			</span>
			{#if !streaming}
				<Button variant="ghost" size="sm" onclick={() => connect(endpoint, range)}>
					Reconnect
				</Button>
			{/if}
		{/if}
	</div>

	<!-- Always present, never conditional on there being lines: an empty panel is
	     exactly when someone reaches for the range control, and a row that appears
	     with the first line would move the output the moment it arrived. -->
	<div class="flex flex-none items-center gap-2 border-b border-border px-3 py-2">
		<!-- type=search for the clear button the platform already draws. -->
		<Input
			type="search"
			size="sm"
			bind:value={query}
			placeholder="Filter logs"
			aria-label="Filter {subject} output"
			class="min-w-0 flex-1"
		/>
		<Select
			size="sm"
			items={ranges}
			bind:value={range}
			class="w-32 flex-none"
			aria-label="Time range"
		/>
	</div>

	{#if lines.length === 0}
		<!-- One block for all four states, spinner in the icon's place while we
		     wait: whichever answer arrives, nothing on screen moves.

		     Column, not row: a flex item cannot shrink below its content here, so a
		     short panel scrolls from the top instead of clipping the frame. -->
		<div class="flex min-h-0 flex-1 flex-col overflow-y-auto">
			{#if canRetry}
				<EmptyState
					variant="canvas"
					icon={empty.icon}
					title={empty.title}
					description={empty.description}
					class="pb-10"
				>
					{#snippet actions()}
						<Button variant="secondary" size="sm" onclick={() => connect(endpoint, range)}>
							Try again
						</Button>
					{/snippet}
				</EmptyState>
			{:else}
				<EmptyState
					variant="canvas"
					icon={empty.icon}
					title={empty.title}
					description={empty.description}
					busy={phase === 'connecting'}
					class="pb-10"
				/>
			{/if}
		</div>
	{:else if visible.length === 0}
		<!-- Lines arrived, the filter hides all of them. Its own state rather than
		     the ones above: nothing is wrong with the stream, and offering Try again
		     here would point at the wrong thing entirely. -->
		<div class="flex min-h-0 flex-1 flex-col overflow-y-auto">
			<EmptyState
				variant="canvas"
				icon={CommandLine}
				title="No matching lines"
				description="None of the {lines.length.toLocaleString()} lines held so far contain “{query.trim()}”."
				class="pb-10"
			>
				{#snippet actions()}
					<Button variant="secondary" size="sm" onclick={() => (query = '')}>Clear filter</Button>
				{/snippet}
			</EmptyState>
		</div>
	{:else}
		<!-- aria-live off, deliberately: role="log" is polite by default and a
		     chatty container would read every line aloud forever. The header's
		     role="status" announces the state changes that actually matter. -->
		<div
			bind:this={scroller}
			class="log-output"
			role="log"
			aria-live="off"
			aria-label="{subject} output"
		>
			{#each visible as line, i (i)}
				<p class:err={line.type === 'stderr'}>{line.output}</p>
			{/each}
		</div>

		<!-- Under the output, not above it: this happened after the last line, and
		     that is where the eye already is when the stream dies. -->
		{#if streamError}
			<p class="flex-none border-t border-border px-3 py-2 text-[13px] text-muted-foreground">
				{streamError}
			</p>
		{/if}
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

	/* Pulsing means "something is still happening" — output arriving, or a
	   connection being made. A stopped stream holds still, and a failed one
	   holds still in the colour of the problem. */
	.status-mark[data-phase='live'] {
		background: var(--success);
		animation: status-pulse 1.6s ease-in-out infinite;
	}

	.status-mark[data-phase='connecting'] {
		animation: status-pulse 1.6s ease-in-out infinite;
	}

	.status-mark[data-phase='failed'] {
		background: var(--destructive);
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
		flex: 1 1 auto;
		min-height: 0;
		overflow-y: auto;
		background: var(--muted);
		padding: 0.625rem 0.75rem;
		font-family: var(--font-mono, ui-monospace, monospace);
		font-size: 0.75rem;
		line-height: 1.55;
		color: var(--foreground);
		/* Same reason as the deployment log: one unbroken 71-char digest would
		   otherwise force a horizontal scrollbar across the whole panel. */
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}

	.log-output p {
		margin: 0;
	}

	.log-output p.err {
		color: var(--destructive);
	}
</style>
