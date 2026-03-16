<script lang="ts">
	import DeploymentLogs from '$lib/components/DeploymentLogs.svelte';

	let output = $state('Klik tombol untuk lihat container yang sedang berjalan');
	let deploymentId = $state<string | null>(null);

	let image = $state('nginx:latest');
	let containerName = $state('nginx-test');
	let port = $state(8080);
	let serverHost = $state('');
	let serverPort = $state(22);
	let serverUser = $state('uploy');
	let privateKey = $state('');

	async function fetchDockerPs() {
		const res = await fetch('/api/docker/ps');
		output = await res.text();
	}

	async function startDeploy() {
		const res = await fetch('/api/deployments', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				image,
				container_name: containerName,
				port,
				server: {
					host: serverHost,
					port: serverPort,
					user: serverUser,
					private_key: privateKey
				}
			})
		});
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
	<form
		onsubmit={(e) => {
			e.preventDefault();
			startDeploy();
		}}
		class="flex flex-col gap-2 max-w-md"
	>
		<fieldset class="flex flex-col gap-2 border border-gray-300 p-3 rounded">
			<legend class="font-bold text-sm px-1">Server</legend>
			<label class="flex flex-col gap-1 text-sm">
				Host
				<input type="text" bind:value={serverHost} required class="border p-1 rounded" placeholder="103.xxx.xxx.xxx" />
			</label>
			<label class="flex flex-col gap-1 text-sm">
				Port
				<input type="number" bind:value={serverPort} class="border p-1 rounded" />
			</label>
			<label class="flex flex-col gap-1 text-sm">
				User
				<input type="text" bind:value={serverUser} class="border p-1 rounded" />
			</label>
			<label class="flex flex-col gap-1 text-sm">
				Private Key
				<textarea bind:value={privateKey} required rows="5" class="border p-1 rounded font-mono text-xs" placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"></textarea>
			</label>
		</fieldset>

		<fieldset class="flex flex-col gap-2 border border-gray-300 p-3 rounded">
			<legend class="font-bold text-sm px-1">Container</legend>
			<label class="flex flex-col gap-1 text-sm">
				Image
				<input type="text" bind:value={image} required class="border p-1 rounded" placeholder="nginx:latest" />
			</label>
			<label class="flex flex-col gap-1 text-sm">
				Container Name
				<input type="text" bind:value={containerName} required class="border p-1 rounded" placeholder="nginx-test" />
			</label>
			<label class="flex flex-col gap-1 text-sm">
				Port
				<input type="number" bind:value={port} required class="border p-1 rounded" />
			</label>
		</fieldset>

		<button type="submit" class="cursor-pointer rounded-sm bg-black p-2 text-white">Deploy</button>
	</form>

	{#if deploymentId}
		<DeploymentLogs {deploymentId} />
	{/if}
</section>
