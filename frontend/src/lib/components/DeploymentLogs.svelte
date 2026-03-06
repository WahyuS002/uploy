<script lang="ts">
	import { onMount, onDestroy } from 'svelte';

	interface LogEntry {
		id: number;
		created_at: string;
		output: string;
	}

	interface Props {
		deploymentId: string;
	}

	let { deploymentId }: Props = $props();

	let logs: LogEntry[] = $state([]);
	let status: string = $state('in_progress');
	let streamError: string = $state('');
	let eventSource: EventSource | null = null;

	onMount(() => {
		eventSource = new EventSource(`/api/deployments/${deploymentId}/logs`);

		eventSource.onmessage = (e) => {
			const log: LogEntry = JSON.parse(e.data);
			logs = [...logs, log];
		};

		eventSource.addEventListener('done', (e) => {
			status = (e as MessageEvent).data;
			eventSource?.close();
		});

		eventSource.addEventListener('stream-error', (e) => {
			const data = JSON.parse((e as MessageEvent).data);
			streamError = data.message;
			status = 'failed';
			eventSource?.close();
		});
	});

	onDestroy(() => {
		eventSource?.close();
	});
</script>

<div>
	<p>Status: <strong>{status}</strong></p>
	{#if streamError}
		<p style="color: red;">{streamError}</p>
	{/if}
	<div style="background: #1a1a1a; color: #fff; padding: 1rem; font-family: monospace;">
		{#each logs as log (log.id)}
			<p style="margin: 0">{log.output}</p>
		{/each}
	</div>
</div>
