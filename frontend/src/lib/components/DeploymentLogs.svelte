<script lang="ts">
	interface LogEntry {
		created_at: string;
		output: string;
	}

	interface Props {
		deploymentId: string;
	}

	let { deploymentId }: Props = $props();

	let logs: LogEntry[] = $state([]);
	let status: string = $state('in_progress');
	let after = '';

	async function fetchLogs() {
		const res = await fetch(`/api/deployments/${deploymentId}/logs?after=${encodeURIComponent(after)}`);
		const data = await res.json();

		logs = [...logs, ...data.logs];
		after = data.next_after;
		status = data.status;
	}

	// pooling
	const interval = setInterval(async () => {
		await fetchLogs();
		if (status === 'success' || status === 'failed') {
			clearInterval(interval);
		}
	}, 2000);

	fetchLogs();
</script>

<div>
	<p>Status: <strong>{status}</strong></p>
	<div style="background: #1a1a1a; color: #fff; padding: 1rem; font-family: monospace;">
		{#each logs as log (log.created_at)}
			<p style="margin: 0">{log.output}</p>
		{/each}
	</div>
</div>
