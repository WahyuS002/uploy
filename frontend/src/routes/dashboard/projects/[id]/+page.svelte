<script lang="ts">
	import { page } from '$app/state';
	import { replaceState } from '$app/navigation';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';
	import PageHeader from '$lib/components/app/PageHeader.svelte';
	import FormField from '$lib/components/app/FormField.svelte';
	import ResourceListItem from '$lib/components/app/ResourceListItem.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import { Squares2x2, Cube } from '@steeze-ui/heroicons';

	type ProjectResponse = components['schemas']['ProjectResponse'];
	type EnvironmentResponse = components['schemas']['EnvironmentResponse'];
	type ServiceResponse = components['schemas']['ServiceResponse'];
	type ServerResponse = components['schemas']['ServerResponse'];

	let { data }: { data: PageData } = $props();
	let canEdit = $derived(data.workspace?.role === 'owner' || data.workspace?.role === 'developer');
	let isOwner = $derived(data.workspace?.role === 'owner');

	let projectId = $derived(page.params.id as string);

	let project = $state<ProjectResponse | null>(null);
	let environments = $state<EnvironmentResponse[]>([]);
	let services = $state<ServiceResponse[]>([]);
	let servers = $state<ServerResponse[]>([]);

	let envError = $state('');
	let creatingEnv = $state(false);
	let envName = $state('');

	// Service form
	let showServiceForm = $state(false);
	let svcError = $state('');
	let creatingService = $state(false);
	let svcName = $state('');
	let svcImage = $state('nginx:latest');
	let svcContainerName = $state('');
	let svcPort = $state(8080);
	let svcServerId = $state('');
	let svcEnvironmentId = $state('');

	let serverItems = $derived(servers.map((s) => ({ value: s.id, label: `${s.name} (${s.host})` })));
	let envItems = $derived(environments.map((e) => ({ value: e.id, label: e.name })));
	let envById = $derived(new Map(environments.map((e) => [e.id, e.name])));

	async function loadProject(id: string) {
		const { data } = await api.GET('/api/projects/{id}', {
			params: { path: { id } }
		});
		return data ?? null;
	}

	async function loadEnvironments(id: string) {
		const { data } = await api.GET('/api/projects/{id}/environments', {
			params: { path: { id } }
		});
		return data ?? [];
	}

	async function loadServices(id: string) {
		const { data } = await api.GET('/api/projects/{id}/services', {
			params: { path: { id } }
		});
		return data ?? [];
	}

	async function loadServers() {
		const { data } = await api.GET('/api/servers');
		if (data) servers = data;
	}

	async function createEnvironment() {
		envError = '';
		creatingEnv = true;
		try {
			const { data, error: err } = await api.POST('/api/projects/{id}/environments', {
				params: { path: { id: projectId } },
				body: { name: envName }
			});
			if (err) {
				envError = (err as { error: string }).error;
				return;
			}
			if (data) {
				environments = [...environments, data];
				envName = '';
			}
		} catch {
			envError = 'Network error';
		} finally {
			creatingEnv = false;
		}
	}

	async function deleteEnvironment(envId: string) {
		const { error: err } = await api.DELETE('/api/projects/{id}/environments/{envId}', {
			params: { path: { id: projectId, envId } }
		});
		if (err) {
			envError = (err as { error: string }).error || 'Failed to delete environment';
			return;
		}
		const remaining = environments.filter((e) => e.id !== envId);
		environments = remaining;
		if (svcEnvironmentId === envId) {
			if (remaining.length > 0) {
				svcEnvironmentId = remaining[0].id;
			} else {
				svcEnvironmentId = '';
				closeServiceForm();
			}
		}
	}

	function openServiceForm() {
		svcError = '';
		showServiceForm = true;
		if (!svcServerId && servers.length > 0) svcServerId = servers[0].id;
		if (!svcEnvironmentId && environments.length > 0) svcEnvironmentId = environments[0].id;
	}

	function closeServiceForm() {
		showServiceForm = false;
		svcError = '';
	}

	async function createService() {
		svcError = '';
		creatingService = true;
		try {
			const { data, error: err } = await api.POST('/api/services', {
				body: {
					name: svcName,
					image: svcImage,
					container_name: svcContainerName,
					port: svcPort,
					server_id: svcServerId,
					environment_id: svcEnvironmentId,
					kind: 'application'
				}
			});
			if (err) {
				svcError = (err as { error: string }).error;
				return;
			}
			if (data) {
				services = [data, ...services];
				svcName = '';
				svcContainerName = '';
				showServiceForm = false;
			}
		} catch {
			svcError = 'Network error';
		} finally {
			creatingService = false;
		}
	}

	function applyStarter() {
		const qs = page.url.searchParams;
		const starter = qs.get('starter');
		if (starter !== 'docker-image') return;
		const serverId = qs.get('serverId') ?? '';
		const environmentId = qs.get('environmentId') ?? '';

		const serverOk = !serverId || servers.some((s) => s.id === serverId);
		const envOk = !environmentId || environments.some((e) => e.id === environmentId);
		if (!serverOk || !envOk) return;

		if (serverId) svcServerId = serverId;
		if (environmentId) svcEnvironmentId = environmentId;
		showServiceForm = true;

		// Clean URL so refresh doesn't reopen starter state
		const cleanUrl = new URL(page.url);
		cleanUrl.searchParams.delete('starter');
		cleanUrl.searchParams.delete('serverId');
		cleanUrl.searchParams.delete('environmentId');
		// eslint-disable-next-line svelte/no-navigation-without-resolve
		replaceState(cleanUrl.pathname + (cleanUrl.search ? cleanUrl.search : ''), page.state);
	}

	$effect(() => {
		const id = projectId;
		let cancelled = false;

		project = null;
		environments = [];
		services = [];
		showServiceForm = false;
		svcEnvironmentId = '';
		svcServerId = '';

		(async () => {
			const [proj, envs, svcs] = await Promise.all([
				loadProject(id),
				loadEnvironments(id),
				loadServices(id),
				loadServers()
			]);
			if (cancelled) return;
			project = proj;
			environments = envs;
			services = svcs;
			if (canEdit) applyStarter();
		})();

		return () => {
			cancelled = true;
		};
	});
</script>

{#if project}
	<section>
		<PageHeader title={project.name} />

		<!-- Environments -->
		<div class="mb-10">
			<h3 class="mb-3 text-sm font-semibold text-foreground">Environments</h3>

			{#if canEdit}
				<form
					onsubmit={(e) => {
						e.preventDefault();
						createEnvironment();
					}}
					class="mb-4 flex items-end gap-2"
				>
					<FormField label="Environment name">
						<Input
							type="text"
							bind:value={envName}
							placeholder="e.g. staging, production"
							required
						/>
					</FormField>
					<Button type="submit" size="sm" loading={creatingEnv}>
						{creatingEnv ? 'Creating...' : 'Add Environment'}
					</Button>
				</form>
				{#if envError}
					<p class="mb-2 text-sm text-danger">{envError}</p>
				{/if}
			{/if}

			{#if environments.length === 0}
				<EmptyState
					icon={Squares2x2}
					title="No environments yet"
					description="Add an environment like staging or production before creating services."
				/>
			{:else}
				<div class="flex flex-col gap-1">
					{#each environments as env (env.id)}
						<ResourceListItem class="justify-between">
							<div>
								<span class="font-medium text-foreground">{env.name}</span>
								<span class="ml-2 font-mono text-xs text-muted-foreground">{env.id}</span>
							</div>
							{#if isOwner}
								<Button
									variant="ghost"
									size="sm"
									onclick={() => deleteEnvironment(env.id)}
									class="text-danger hover:text-red-700"
								>
									Delete
								</Button>
							{/if}
						</ResourceListItem>
					{/each}
				</div>
			{/if}
		</div>

		<!-- Services -->
		<div>
			<div class="mb-3 flex items-center justify-between">
				<h3 class="text-sm font-semibold text-foreground">Services</h3>
				{#if canEdit && environments.length > 0 && servers.length > 0 && !showServiceForm}
					<Button size="sm" onclick={openServiceForm}>Add service</Button>
				{/if}
			</div>

			{#if canEdit && showServiceForm}
				<form
					onsubmit={(e) => {
						e.preventDefault();
						createService();
					}}
					class="mb-6 flex max-w-md flex-col gap-3 rounded-xl border border-border bg-surface p-5"
				>
					<FormField label="Name">
						<Input type="text" bind:value={svcName} required />
					</FormField>
					<FormField label="Image">
						<Input type="text" bind:value={svcImage} required />
					</FormField>
					<FormField label="Container Name">
						<Input type="text" bind:value={svcContainerName} required />
					</FormField>
					<FormField label="Port">
						<Input type="number" bind:value={svcPort} required />
					</FormField>
					<FormField label="Server">
						<Select
							items={serverItems}
							bind:value={svcServerId}
							required
							placeholder="Select a server"
						/>
					</FormField>
					<FormField label="Environment">
						<Select
							items={envItems}
							bind:value={svcEnvironmentId}
							required
							placeholder="Select an environment"
						/>
					</FormField>

					{#if svcError}
						<p class="text-sm text-danger">{svcError}</p>
					{/if}

					<div class="flex items-center gap-2">
						<Button type="submit" loading={creatingService}>
							{creatingService ? 'Creating...' : 'Create Service'}
						</Button>
						<Button type="button" variant="secondary" onclick={closeServiceForm}>Cancel</Button>
					</div>
				</form>
			{/if}

			{#if services.length === 0}
				{#if environments.length === 0}
					<EmptyState
						icon={Cube}
						title="No services yet"
						description="Add an environment above before creating a service."
					/>
				{:else if servers.length === 0}
					<EmptyState
						icon={Cube}
						title="No servers available"
						description="Connect a server before creating a service."
					>
						{#snippet actions()}
							{#if isOwner}
								<Button href="/dashboard/servers" size="sm">Go to servers</Button>
							{/if}
						{/snippet}
					</EmptyState>
				{:else}
					<EmptyState
						icon={Cube}
						title="No services yet"
						description="Deploy a container to run on one of this project's environments."
					>
						{#snippet actions()}
							{#if canEdit && !showServiceForm}
								<Button size="sm" onclick={openServiceForm}>Add service</Button>
							{/if}
						{/snippet}
					</EmptyState>
				{/if}
			{:else}
				<div class="flex flex-col gap-2">
					{#each services as svc (svc.id)}
						<ResourceListItem href="/dashboard/services/{svc.id}" class="justify-between">
							<div class="min-w-0">
								<div class="truncate font-medium text-foreground">{svc.name}</div>
								<div class="truncate font-mono text-xs text-muted-foreground">
									{svc.image} &rarr; :{svc.port}
								</div>
							</div>
							{#if envById.get(svc.environment_id)}
								<Badge variant="soft">{envById.get(svc.environment_id)}</Badge>
							{/if}
						</ResourceListItem>
					{/each}
				</div>
			{/if}
		</div>
	</section>
{/if}
