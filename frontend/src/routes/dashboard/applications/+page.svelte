<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';

	type ApplicationResponse = components['schemas']['ApplicationResponse'];
	type ServerResponse = components['schemas']['ServerResponse'];

	let { data }: { data: PageData } = $props();
	let canEdit = $derived(data.workspace?.role === 'owner' || data.workspace?.role === 'developer');

	let apps = $state<ApplicationResponse[]>([]);
	let servers = $state<ServerResponse[]>([]);
	let error = $state('');
	let creating = $state(false);

	// Form state
	let name = $state('');
	let image = $state('nginx:latest');
	let containerName = $state('');
	let port = $state(80);
	let selectedServerId = $state('');
	let fqdn = $state('');

	async function load() {
		const [appsRes, serversRes] = await Promise.all([
			api.GET('/api/applications'),
			api.GET('/api/servers')
		]);
		if (appsRes.data) apps = appsRes.data;
		if (serversRes.data) {
			servers = serversRes.data;
			if (servers.length > 0 && !selectedServerId) {
				selectedServerId = servers[0].id;
			}
		}
	}

	async function createApp() {
		error = '';
		creating = true;
		try {
			const { data, error: err } = await api.POST('/api/applications', {
				body: {
					name,
					image,
					container_name: containerName,
					port,
					server_id: selectedServerId,
					...(fqdn.trim() ? { fqdn: fqdn.trim() } : {})
				}
			});
			if (err) {
				error = (err as { error: string }).error;
				return;
			}
			if (data) {
				apps = [data, ...apps];
				name = '';
				containerName = '';
				fqdn = '';
			}
		} catch {
			error = 'Network error';
		} finally {
			creating = false;
		}
	}

	$effect(() => {
		load();
	});
</script>

<section>
	<h2 class="mb-4 text-xl font-bold">Applications</h2>

	{#if canEdit}
		<!-- Create form -->
		<form
			onsubmit={(e) => { e.preventDefault(); createApp(); }}
			class="mb-6 flex max-w-md flex-col gap-2"
		>
			<label class="flex flex-col gap-1 text-sm">
				Name
				<input type="text" bind:value={name} required class="rounded border p-1" />
			</label>
			<label class="flex flex-col gap-1 text-sm">
				Image
				<input type="text" bind:value={image} required class="rounded border p-1" />
			</label>
			<label class="flex flex-col gap-1 text-sm">
				Container Name
				<input type="text" bind:value={containerName} required class="rounded border p-1" />
			</label>
			<label class="flex flex-col gap-1 text-sm">
				Port
				<input type="number" bind:value={port} required class="rounded border p-1" />
			</label>
			<label class="flex flex-col gap-1 text-sm">
				Domain (optional)
				<input
					type="text"
					bind:value={fqdn}
					class="rounded border p-1"
					placeholder="myapp.example.com"
				/>
				<span class="text-xs text-gray-400">
					Leave empty for direct IP:port access. Set domain for HTTPS proxy routing.
				</span>
			</label>
			<label class="flex flex-col gap-1 text-sm">
				Server
				<select bind:value={selectedServerId} required class="rounded border p-1">
					{#each servers as server}
						<option value={server.id}>{server.name} ({server.host})</option>
					{/each}
				</select>
			</label>

			{#if error}
				<p class="text-sm text-red-600">{error}</p>
			{/if}

			<button
				type="submit"
				disabled={servers.length === 0 || creating}
				class="cursor-pointer rounded-sm bg-black p-2 text-white disabled:cursor-not-allowed disabled:opacity-50"
			>
				{creating ? 'Creating...' : 'Create Application'}
			</button>
		</form>
	{/if}

	<!-- App list -->
	{#if apps.length === 0}
		<p class="text-sm text-gray-500">No applications yet.</p>
	{:else}
		<div class="flex flex-col gap-2">
			{#each apps as app}
				<a
					href="/dashboard/applications/{app.id}"
					class="rounded border p-3 hover:bg-gray-50"
				>
					<div class="font-bold">{app.name}</div>
					<div class="text-sm text-gray-600">{app.image} → :{app.port}</div>
				</a>
			{/each}
		</div>
	{/if}
</section>
