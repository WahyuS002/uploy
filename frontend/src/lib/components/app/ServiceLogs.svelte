<script lang="ts">
	import { cn } from '$lib/components/ui/cn.js';
	import Button from '$lib/components/ui/Button.svelte';

	interface Props {
		serviceId: string;
		class?: string;
	}

	let { serviceId, class: className }: Props = $props();

	type Line = { output: string; type: string };

	/**
	 * A deployment's log is finite; this one is not. Somebody leaves the tab open
	 * on a chatty container and the array is the only thing that grows, so it
	 * keeps a window rather than the whole history — Docker still has the rest.
	 */
	const maxLines = 2000;

	let lines = $state<Line[]>([]);
	let streamError = $state('');
	let streaming = $state(false);
	let eventSource: EventSource | null = null;
	// Reconnecting while the previous attempt's explanation is still in flight
	// would otherwise let a stale message land on top of the new one.
	let attempt = 0;

	function disconnect() {
		eventSource?.close();
		eventSource = null;
		streaming = false;
	}

	/**
	 * EventSource hides the response of a failed open, so the reason the backend
	 * took the trouble to send — an unreachable server, no Docker — never reaches
	 * the reader. Ask the endpoint once more, plainly, just to read it.
	 */
	async function explainOpenFailure(id: string, forAttempt: number) {
		const abort = new AbortController();
		try {
			const res = await fetch(`/api/services/${id}/logs`, { signal: abort.signal });
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

	function connect(id: string) {
		disconnect();
		lines = [];
		streamError = '';
		attempt += 1;

		const es = new EventSource(`/api/services/${id}/logs`);
		eventSource = es;

		es.onopen = () => {
			streaming = true;
		};

		es.onmessage = (e) => {
			const line: Line = JSON.parse(e.data);
			lines = [...lines, line].slice(-maxLines);
		};

		es.addEventListener('done', () => {
			streamError = 'The log stream ended — the container may have stopped.';
			disconnect();
		});

		es.addEventListener('stream-error', (e) => {
			streamError = JSON.parse((e as MessageEvent).data).message;
			disconnect();
		});

		// EventSource reconnects by itself, which is wrong here: the endpoint
		// answers 400 for a service that was never deployed and 502 for an
		// unreachable server, and retrying either just hammers it. Close and let
		// the reader decide when to try again.
		es.onerror = () => {
			const wasStreaming = streaming;
			disconnect();
			if (streamError) return;
			if (wasStreaming) {
				streamError = 'Lost connection to the log stream.';
				return;
			}
			streamError = 'Could not open the log stream.';
			explainOpenFailure(id, attempt);
		};
	}

	$effect(() => {
		connect(serviceId);
		return disconnect;
	});

	// Follow the tail only from the tail. Same rule as the deployment log: pulling
	// someone back down while they are reading something further up is worse than
	// not following.
	let scroller = $state<HTMLDivElement | null>(null);
	$effect(() => {
		const count = lines.length;
		const el = scroller;
		if (!el || count === 0) return;
		const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 40;
		if (nearBottom) el.scrollTop = el.scrollHeight;
	});
</script>

<div
	class={cn(
		'flex h-full min-h-0 flex-col overflow-hidden rounded-lg border border-border',
		className
	)}
>
	<div class="flex flex-none items-center gap-2 border-b border-border px-3 py-2.5">
		<span class="status-mark" data-streaming={streaming} aria-hidden="true"></span>
		<span class="min-w-0 flex-1 truncate text-[13px] text-muted-foreground">
			{#if streaming}
				Streaming live output
			{:else if streamError}
				Not streaming
			{:else}
				Connecting…
			{/if}
		</span>
		{#if !streaming}
			<Button variant="ghost" size="sm" onclick={() => connect(serviceId)}>Reconnect</Button>
		{/if}
	</div>

	{#if streamError}
		<p class="flex-none border-b border-border px-3 py-2 text-[13px] text-muted-foreground">
			{streamError}
		</p>
	{/if}

	<div bind:this={scroller} class="log-output">
		{#each lines as line, i (i)}
			<p class:err={line.type === 'stderr'}>{line.output}</p>
		{:else}
			<p class="empty">
				{streaming ? 'Waiting for output…' : 'No output.'}
			</p>
		{/each}
	</div>
</div>

<style>
	.status-mark {
		flex: none;
		width: 0.5rem;
		height: 0.5rem;
		border-radius: 50%;
		background: var(--muted-foreground);
	}

	/* Pulsing means "output is still arriving". A stopped stream holds still. */
	.status-mark[data-streaming='true'] {
		background: var(--success);
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

	.log-output p.empty {
		color: var(--muted-foreground);
	}

	@media (prefers-reduced-motion: reduce) {
		.status-mark[data-streaming='true'] {
			animation: none;
		}
	}
</style>
