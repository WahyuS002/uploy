<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';
	import PageHeader from '$lib/components/app/PageHeader.svelte';
	import FormField from '$lib/components/app/FormField.svelte';
	import ResourceListItem from '$lib/components/app/ResourceListItem.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';

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

	let serverItems = $derived(servers.map((s) => ({ value: s.id, label: `${s.name} (${s.host})` })));
	let projectItems = $derived(projects.map((p) => ({ value: p.id, label: p.name })));
	let envItems = $derived(environments.map((e) => ({ value: e.id, label: e.name })));

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
	<PageHeader title="Services" />

	{#if canEdit && projects.length === 0}
		<!-- eslint-disable svelte/no-navigation-without-resolve -->
		<p class="mb-6 text-sm text-muted-foreground">
			You need a project and environment before creating a service.
			<a href="/dashboard/projects" class="text-accent underline">Create a project</a> first.
		</p>
		<!-- eslint-enable svelte/no-navigation-without-resolve -->
	{:else if canEdit}
		<!-- Create form -->
		<form
			onsubmit={(e) => {
				e.preventDefault();
				createService();
			}}
			class="mb-6 flex max-w-md flex-col gap-3"
		>
			<FormField label="Name">
				<Input type="text" bind:value={name} required />
			</FormField>
			<FormField label="Image">
				<Input type="text" bind:value={image} required />
			</FormField>
			<FormField label="Container Name">
				<Input type="text" bind:value={containerName} required />
			</FormField>
			<FormField label="Port">
				<Input type="number" bind:value={port} required />
			</FormField>
			<FormField label="Server">
				<Select
					items={serverItems}
					bind:value={selectedServerId}
					required
					placeholder="Select a server"
				/>
			</FormField>
			<FormField label="Project">
				<Select
					items={projectItems}
					bind:value={selectedProjectId}
					onValueChange={() => loadEnvironments()}
					required
					placeholder="Select a project"
				/>
			</FormField>
			<FormField label="Environment">
				<Select
					items={envItems}
					bind:value={selectedEnvironmentId}
					required
					disabled={environments.length === 0}
					placeholder="Select an environment"
				/>
			</FormField>

			{#if error}
				<p class="text-sm text-danger">{error}</p>
			{/if}

			<Button
				type="submit"
				loading={creating}
				disabled={servers.length === 0 || environments.length === 0}
			>
				{creating ? 'Creating...' : 'Create Service'}
			</Button>
		</form>
	{/if}

	<!-- Service list -->
	{#if services.length === 0}
		<p class="text-sm text-muted-foreground">No services yet.</p>
	{:else}
		<div class="flex flex-col gap-2">
			{#each services as svc (svc.id)}
				<ResourceListItem href="/dashboard/services/{svc.id}">
					<div>
						<div class="font-bold text-foreground">{svc.name}</div>
						<div class="text-sm text-muted-foreground">{svc.image} &rarr; :{svc.port}</div>
					</div>
				</ResourceListItem>
			{/each}
		</div>
	{/if}
</section>
