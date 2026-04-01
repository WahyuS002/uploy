<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';

	type ServiceResponse = components['schemas']['ServiceResponse'];
	type ServerResponse = components['schemas']['ServerResponse'];
	type ProjectResponse = components['schemas']['ProjectResponse'];
	type EnvironmentResponse = components['schemas']['EnvironmentResponse'];

	let { data }: { data: PageData } = $props();
	let canEdit = $derived(data.workspace?.role === 'owner' || data.workspace?.role === 'developer');

	let services = $state<ServiceResponse[]>([]);
	let servers = $state<ServerResponse[]>([]);
	let projects = $state<ProjectResponse[]>([]);
	let environments = $state<EnvironmentResponse[]>([]);
	let error = $state('');
	let creating = $state(false);

	// Form state
	let name = $state('');
	let image = $state('nginx:latest');
	let containerName = $state('');
	let port = $state(8080);
	let selectedServerId = $state('');
	let selectedProjectId = $state('');
	let selectedEnvironmentId = $state('');

	async function load() {
		const [servicesRes, serversRes, projectsRes] = await Promise.all([
			api.GET('/api/services'),
			api.GET('/api/servers'),
			api.GET('/api/projects')
		]);
		if (servicesRes.data) services = servicesRes.data;
		if (serversRes.data) {
			servers = serversRes.data;
			if (servers.length > 0 && !selectedServerId) {
				selectedServerId = servers[0].id;
			}
		}
		if (projectsRes.data) {
			projects = projectsRes.data;
			if (projects.length > 0 && !selectedProjectId) {
				selectedProjectId = projects[0].id;
				await loadEnvironments();
			}
		}
	}

	async function loadEnvironments() {
		if (!selectedProjectId) {
			environments = [];
			selectedEnvironmentId = '';
			return;
		}
		const { data } = await api.GET('/api/projects/{id}/environments', {
			params: { path: { id: selectedProjectId } }
		});
		if (data) {
			environments = data;
			if (environments.length > 0) {
				selectedEnvironmentId = environments[0].id;
			} else {
				selectedEnvironmentId = '';
			}
		}
	}

	async function createService() {
		error = '';
		creating = true;
		try {
			const { data, error: err } = await api.POST('/api/services', {
				body: {
					name,
					image,
					container_name: containerName,
					port,
					server_id: selectedServerId,
					environment_id: selectedEnvironmentId,
					kind: 'application'
				}
			});
			if (err) {
				error = (err as { error: string }).error;
				return;
			}
			if (data) {
				services = [data, ...services];
				name = '';
				containerName = '';
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
	<h2 class="mb-4 text-xl font-bold">Services</h2>

	{#if canEdit && projects.length === 0}
		<p class="mb-6 text-sm text-gray-500">
			You need a project and environment before creating a service.
			<a href="/dashboard/projects" class="text-blue-600 underline">Create a project</a> first.
		</p>
	{:else if canEdit}
		<!-- Create form -->
		<form
			onsubmit={(e) => { e.preventDefault(); createService(); }}
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
				Server
				<select bind:value={selectedServerId} required class="rounded border p-1">
					{#each servers as server}
						<option value={server.id}>{server.name} ({server.host})</option>
					{/each}
				</select>
			</label>
			<label class="flex flex-col gap-1 text-sm">
				Project
				<select bind:value={selectedProjectId} onchange={() => loadEnvironments()} required class="rounded border p-1">
					{#each projects as project}
						<option value={project.id}>{project.name}</option>
					{/each}
				</select>
			</label>
			<label class="flex flex-col gap-1 text-sm">
				Environment
				<select bind:value={selectedEnvironmentId} required class="rounded border p-1" disabled={environments.length === 0}>
					{#each environments as env}
						<option value={env.id}>{env.name}</option>
					{/each}
				</select>
			</label>

			{#if error}
				<p class="text-sm text-red-600">{error}</p>
			{/if}

			<button
				type="submit"
				disabled={servers.length === 0 || environments.length === 0 || creating}
				class="cursor-pointer rounded-sm bg-black p-2 text-white disabled:cursor-not-allowed disabled:opacity-50"
			>
				{creating ? 'Creating...' : 'Create Service'}
			</button>
		</form>
	{/if}

	<!-- Service list -->
	{#if services.length === 0}
		<p class="text-sm text-gray-500">No services yet.</p>
	{:else}
		<div class="flex flex-col gap-2">
			{#each services as svc}
				<a
					href="/dashboard/services/{svc.id}"
					class="rounded border p-3 hover:bg-gray-50"
				>
					<div class="font-bold">{svc.name}</div>
					<div class="text-sm text-gray-600">{svc.image} &rarr; :{svc.port}</div>
				</a>
			{/each}
		</div>
	{/if}
</section>
