<script lang="ts">
	import DeploymentLogs from '$lib/components/DeploymentLogs.svelte';
	import { api } from '$lib/api/client';

	let deploymentId = $state<string | null>(null);

	let image = $state('nginx:latest');
	let containerName = $state('nginx-test');
	let port = $state(8080);
	let serverHost = $state('');
	let serverPort = $state(22);
	let serverUser = $state('uploy');
	let privateKey = $state('');

	async function startDeploy() {
		const { data } = await api.POST('/api/deployments', {
			body: {
				image,
				container_name: containerName,
				port,
				server: {
					host: serverHost,
					port: serverPort,
					user: serverUser,
					private_key: privateKey
				}
			}
		});
		if (data) {
			deploymentId = data.deployment_id;
		}
	}
</script>

<section>
	<h2 class="mb-4 text-xl font-bold">Deploy</h2>
	<form
		onsubmit={(e) => {
			e.preventDefault();
			startDeploy();
		}}
		class="flex max-w-md flex-col gap-2"
	>
		<fieldset class="flex flex-col gap-2 rounded border border-gray-300 p-3">
			<legend class="px-1 text-sm font-bold">Server</legend>
			<label class="flex flex-col gap-1 text-sm">
				Host
				<input
					type="text"
					bind:value={serverHost}
					required
					class="rounded border p-1"
					placeholder="103.xxx.xxx.xxx"
				/>
			</label>
			<label class="flex flex-col gap-1 text-sm">
				Port
				<input type="number" bind:value={serverPort} class="rounded border p-1" />
			</label>
			<label class="flex flex-col gap-1 text-sm">
				User
				<input type="text" bind:value={serverUser} class="rounded border p-1" />
			</label>
			<label class="flex flex-col gap-1 text-sm">
				Private Key
				<textarea
					bind:value={privateKey}
					required
					rows="5"
					class="rounded border p-1 font-mono text-xs"
					placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
				></textarea>
			</label>
		</fieldset>

		<fieldset class="flex flex-col gap-2 rounded border border-gray-300 p-3">
			<legend class="px-1 text-sm font-bold">Container</legend>
			<label class="flex flex-col gap-1 text-sm">
				Image
				<input
					type="text"
					bind:value={image}
					required
					class="rounded border p-1"
					placeholder="nginx:latest"
				/>
			</label>
			<label class="flex flex-col gap-1 text-sm">
				Container Name
				<input
					type="text"
					bind:value={containerName}
					required
					class="rounded border p-1"
					placeholder="nginx-test"
				/>
			</label>
			<label class="flex flex-col gap-1 text-sm">
				Port
				<input type="number" bind:value={port} required class="rounded border p-1" />
			</label>
		</fieldset>

		<button type="submit" class="cursor-pointer rounded-sm bg-black p-2 text-white">Deploy</button>
	</form>

	{#if deploymentId}
		<DeploymentLogs {deploymentId} />
	{/if}
</section>
