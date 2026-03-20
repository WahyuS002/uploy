<script lang="ts">
	import DeploymentLogs from '$lib/components/DeploymentLogs.svelte';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';

	type ServerResponse = components['schemas']['ServerResponse'];

	let deploymentId = $state<string | null>(null);
	let servers = $state<ServerResponse[]>([]);
	let selectedServerId = $state('');
	let serverLoadError = $state<string | null>(null);
	let deployError = $state('');
	let deploying = $state(false);

	let image = $state('nginx:latest');
	let containerName = $state('nginx-test');
	let port = $state(8080);

	async function loadServers() {
		try {
			const { data, error } = await api.GET('/api/servers');
			if (error) {
				serverLoadError = (error as { error: string }).error;
				return;
			}
			if (data) {
				servers = data;
				if (servers.length > 0 && !selectedServerId) {
					selectedServerId = servers[0].id;
				}
			}
		} catch {
			serverLoadError = 'Network error, please try again';
		}
	}

	async function startDeploy() {
		deployError = '';
		deploying = true;
		try {
			const { data, error } = await api.POST('/api/deployments', {
				body: {
					image,
					container_name: containerName,
					port,
					server_id: selectedServerId
				}
			});
			if (error) {
				deployError = (error as { error: string }).error;
				return;
			}
			if (data) {
				deploymentId = data.deployment_id;
			}
		} catch {
			deployError = 'Network error, please try again';
		} finally {
			deploying = false;
		}
	}

	$effect(() => {
		loadServers();
	});
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
			{#if serverLoadError}
				<p class="text-sm text-red-600">{serverLoadError}</p>
			{:else if servers.length === 0}
				<p class="text-sm text-gray-500">
					No servers registered.
					<a href="/dashboard/servers" class="text-blue-600 underline">Add a server</a> first.
				</p>
			{:else}
				<label class="flex flex-col gap-1 text-sm">
					Select Server
					<select bind:value={selectedServerId} required class="rounded border p-1">
						{#each servers as server}
							<option value={server.id}>{server.name} ({server.host})</option>
						{/each}
					</select>
				</label>
			{/if}
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

		{#if deployError}
			<p class="text-sm text-red-600">{deployError}</p>
		{/if}

		<button
			type="submit"
			disabled={servers.length === 0 || !!serverLoadError || deploying}
			class="cursor-pointer rounded-sm bg-black p-2 text-white disabled:cursor-not-allowed disabled:opacity-50"
		>
			{deploying ? 'Deploying...' : 'Deploy'}
		</button>
	</form>

	{#if deploymentId}
		<DeploymentLogs {deploymentId} />
	{/if}
</section>
