<script lang="ts">
	import DeploymentLogs from '$lib/components/DeploymentLogs.svelte';

	let output = $state('Klik tombol untuk lihat container yang sedang berjalan');
	let deploymentId = $state<string | null>(null);

	async function fetchDockerPs() {
		const res = await fetch('/api/docker/ps');
		output = await res.text();
	}

	async function startDeploy() {
		const res = await fetch('/api/deployments', { method: 'POST' });
		const data = await res.json();
		deploymentId = data.deployment_id;
	}
</script>

<h1>Hello World (uploy)</h1>

<section>
	<h2>Docker PS</h2>
	<button onclick={fetchDockerPs} class="cursor-pointer rounded-sm bg-black p-2 text-white"
		>Refresh</button
	>
	<pre>{output}</pre>
</section>

<section>
	<h2>Deploy</h2>
	<button onclick={startDeploy} class="cursor-pointer rounded-sm bg-black p-2 text-white"
		>Deploy nginx:latest</button
	>

	{#if deploymentId}
		<DeploymentLogs {deploymentId} />
	{/if}
</section>
