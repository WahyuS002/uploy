<script lang="ts">
	import { page } from '$app/state';
	import { replaceState } from '$app/navigation';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';
	import FormField from '$lib/components/app/FormField.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import Dialog from '$lib/components/ui/Dialog.svelte';
	import DialogContent from '$lib/components/ui/DialogContent.svelte';
	import DialogHeader from '$lib/components/ui/DialogHeader.svelte';
	import DialogTitle from '$lib/components/ui/DialogTitle.svelte';
	import DialogFooter from '$lib/components/ui/DialogFooter.svelte';
	import { DropdownMenu } from 'bits-ui';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { ChevronDown, Cube, Plus, ServerStack, Squares2x2, Check } from '@steeze-ui/heroicons';

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
	let loaded = $state(false);

	let selectedEnvId = $state('');

	let showServiceForm = $state(false);
	let svcError = $state('');
	let creatingService = $state(false);
	let svcName = $state('');
	let svcImage = $state('nginx:latest');
	let svcContainerName = $state('');
	let svcPort = $state(8080);
	let svcServerId = $state('');

	let envDialogOpen = $state(false);
	let newEnvName = $state('');
	let envError = $state('');
	let creatingEnv = $state(false);

	let selectedEnv = $derived(environments.find((e) => e.id === selectedEnvId) ?? null);
	let envServices = $derived(
		selectedEnvId ? services.filter((s) => s.environment_id === selectedEnvId) : []
	);
	let serverItems = $derived(servers.map((s) => ({ value: s.id, label: `${s.name} (${s.host})` })));
	let serverById = $derived(new Map(servers.map((s) => [s.id, s])));

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

	function pickDefaultEnvironment(envs: EnvironmentResponse[]): string {
		if (envs.length === 0) return '';
		const prod = envs.find((e) => e.name === 'production');
		return (prod ?? envs[0]).id;
	}

	function openServiceForm() {
		if (!selectedEnvId) return;
		svcError = '';
		if (!svcServerId && servers.length > 0) svcServerId = servers[0].id;
		showServiceForm = true;
	}

	function closeServiceForm() {
		showServiceForm = false;
		svcError = '';
	}

	async function createService() {
		if (!selectedEnvId) return;
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
					environment_id: selectedEnvId,
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

	function openEnvDialog() {
		envError = '';
		newEnvName = '';
		envDialogOpen = true;
	}

	async function createEnvironment() {
		envError = '';
		creatingEnv = true;
		try {
			const { data, error: err } = await api.POST('/api/projects/{id}/environments', {
				params: { path: { id: projectId } },
				body: { name: newEnvName }
			});
			if (err) {
				envError = (err as { error: string }).error;
				return;
			}
			if (data) {
				environments = [...environments, data];
				selectedEnvId = data.id;
				envDialogOpen = false;
			}
		} catch {
			envError = 'Network error';
		} finally {
			creatingEnv = false;
		}
	}

	async function createDefaultProduction() {
		envError = '';
		creatingEnv = true;
		try {
			const { data, error: err } = await api.POST('/api/projects/{id}/environments', {
				params: { path: { id: projectId } },
				body: { name: 'production' }
			});
			if (err) {
				envError = (err as { error: string }).error ?? 'Failed to create environment';
				return;
			}
			if (data) {
				environments = [...environments, data];
				selectedEnvId = data.id;
			}
		} catch {
			envError = 'Network error';
		} finally {
			creatingEnv = false;
		}
	}

	function applyStarter() {
		const qs = page.url.searchParams;
		const starter = qs.get('starter');
		if (starter !== 'docker-image') return;
		if (!canEdit) return;
		if (!selectedEnvId) return;

		if (servers.length > 0 && !svcServerId) svcServerId = servers[0].id;
		showServiceForm = true;

		const cleanUrl = new URL(page.url);
		cleanUrl.searchParams.delete('starter');
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
		selectedEnvId = '';
		svcServerId = '';
		loaded = false;

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
			selectedEnvId = pickDefaultEnvironment(envs);
			loaded = true;
			applyStarter();
		})();

		return () => {
			cancelled = true;
		};
	});
</script>

{#if project}
	<section class="flex flex-col gap-6">
		<header class="flex flex-wrap items-center justify-between gap-3">
			<div class="flex items-center gap-3">
				<h2 class="text-xl font-medium text-foreground">{project.name}</h2>
				{#if environments.length > 0}
					<DropdownMenu.Root>
						<DropdownMenu.Trigger
							class="inline-flex h-8 items-center gap-1.5 rounded-md border border-border bg-surface px-2.5 text-xs font-medium text-foreground transition-colors hover:bg-surface-muted"
						>
							<Icon src={Squares2x2} theme="outline" class="h-3.5 w-3.5 text-muted-foreground" />
							<span>{selectedEnv?.name ?? 'Select environment'}</span>
							<Icon src={ChevronDown} theme="outline" class="h-3 w-3 text-muted-foreground" />
						</DropdownMenu.Trigger>
						<DropdownMenu.Portal>
							<DropdownMenu.Content
								align="start"
								sideOffset={6}
								class="z-50 min-w-48 overflow-hidden rounded-lg border border-border bg-surface p-1 shadow-overlay"
							>
								{#each environments as env (env.id)}
									<DropdownMenu.Item
										onSelect={() => (selectedEnvId = env.id)}
										class="flex cursor-pointer items-center justify-between gap-2 rounded-md px-2 py-1.5 text-sm text-foreground outline-none data-highlighted:bg-surface-muted"
									>
										<span>{env.name}</span>
										{#if env.id === selectedEnvId}
											<Icon src={Check} theme="outline" class="h-3.5 w-3.5 text-foreground" />
										{/if}
									</DropdownMenu.Item>
								{/each}
								{#if canEdit}
									<DropdownMenu.Separator class="my-1 h-px bg-border" />
									<DropdownMenu.Item
										onSelect={openEnvDialog}
										class="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm text-muted-foreground outline-none data-highlighted:bg-surface-muted data-highlighted:text-foreground"
									>
										<Icon src={Plus} theme="outline" class="h-3.5 w-3.5" />
										<span>New environment</span>
									</DropdownMenu.Item>
								{/if}
							</DropdownMenu.Content>
						</DropdownMenu.Portal>
					</DropdownMenu.Root>
				{/if}
			</div>

			{#if canEdit && selectedEnvId && !showServiceForm}
				<Button size="sm" onclick={openServiceForm}>
					<Icon src={Plus} theme="outline" class="h-3.5 w-3.5" />
					Add service
				</Button>
			{/if}
		</header>

		{#if !loaded}
			<div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
				<div class="h-28 animate-pulse rounded-xl bg-surface-muted"></div>
				<div class="h-28 animate-pulse rounded-xl bg-surface-muted"></div>
			</div>
		{:else if environments.length === 0}
			<div class="rounded-xl border border-dashed border-border bg-surface-muted/30 p-8">
				<EmptyState
					icon={Squares2x2}
					title="This project has no environments yet"
					description="Create a production environment to start adding services. You'll be able to add more environments later."
				>
					{#snippet actions()}
						{#if canEdit}
							<Button size="sm" onclick={createDefaultProduction} loading={creatingEnv}>
								{creatingEnv ? 'Creating...' : 'Create production environment'}
							</Button>
						{/if}
					{/snippet}
				</EmptyState>
				{#if envError}
					<p class="mt-2 text-center text-sm text-danger">{envError}</p>
				{/if}
			</div>
		{:else}
			<div
				class="relative min-h-90 rounded-xl border border-border bg-surface-muted/20 bg-[radial-gradient(circle_at_1px_1px,var(--color-border)_1px,transparent_0)] bg-size-[18px_18px] p-6"
			>
				{#if canEdit && showServiceForm}
					<form
						onsubmit={(e) => {
							e.preventDefault();
							createService();
						}}
						class="mb-6 flex w-full max-w-md flex-col gap-3 rounded-xl border border-border bg-surface p-5 shadow-sm"
					>
						<div class="flex items-baseline justify-between">
							<h3 class="text-sm font-semibold text-foreground">Add service</h3>
							<span class="text-xs text-muted-foreground">in {selectedEnv?.name}</span>
						</div>
						{#if servers.length === 0}
							<div
								class="rounded-md border border-dashed border-border bg-surface-muted/40 p-3 text-xs text-muted-foreground"
							>
								You need to connect a server before you can deploy a service.
								{#if isOwner}
									<div class="mt-2">
										<Button href="/dashboard/servers" size="xs" variant="secondary">
											<Icon src={ServerStack} theme="outline" class="h-3 w-3" />
											Connect a server
										</Button>
									</div>
								{/if}
							</div>
						{:else}
							<FormField label="Name">
								<Input type="text" bind:value={svcName} required />
							</FormField>
							<FormField label="Image">
								<Input type="text" bind:value={svcImage} required />
							</FormField>
							<FormField label="Container name">
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
						{/if}

						{#if svcError}
							<p class="text-sm text-danger">{svcError}</p>
						{/if}

						<div class="flex items-center gap-2">
							{#if servers.length > 0}
								<Button type="submit" size="sm" loading={creatingService}>
									{creatingService ? 'Creating...' : 'Create service'}
								</Button>
							{/if}
							<Button type="button" size="sm" variant="secondary" onclick={closeServiceForm}>
								Cancel
							</Button>
						</div>
					</form>
				{/if}

				{#if envServices.length === 0}
					{#if !showServiceForm}
						<EmptyState
							icon={Cube}
							title="No services in {selectedEnv?.name}"
							description="Deploy a container to this environment to get started."
						>
							{#snippet actions()}
								{#if canEdit}
									<Button size="sm" onclick={openServiceForm}>
										<Icon src={Plus} theme="outline" class="h-3.5 w-3.5" />
										Add service
									</Button>
								{/if}
							{/snippet}
						</EmptyState>
					{/if}
				{:else}
					<div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
						{#each envServices as svc (svc.id)}
							{@const srv = serverById.get(svc.server_id)}
							<!-- eslint-disable svelte/no-navigation-without-resolve -->
							<a
								href="/dashboard/services/{svc.id}"
								class="group flex flex-col gap-2 rounded-xl border border-border bg-surface p-4 shadow-sm transition-all hover:border-foreground/40 hover:shadow-md"
							>
								<div class="flex items-start justify-between gap-2">
									<div class="flex min-w-0 items-center gap-2">
										<div class="grid h-8 w-8 place-content-center rounded-md bg-surface-muted">
											<Icon src={Cube} theme="outline" class="h-4 w-4 text-foreground" />
										</div>
										<span class="truncate font-medium text-foreground">{svc.name}</span>
									</div>
								</div>
								<div class="truncate font-mono text-xs text-muted-foreground">
									{svc.image}
								</div>
								<div class="flex items-center justify-between text-xs text-muted-foreground">
									<span>Port {svc.port}</span>
									{#if srv}
										<span class="truncate">{srv.name}</span>
									{/if}
								</div>
							</a>
							<!-- eslint-enable svelte/no-navigation-without-resolve -->
						{/each}
					</div>
				{/if}
			</div>
		{/if}
	</section>

	<Dialog bind:open={envDialogOpen}>
		<DialogContent>
			<form
				onsubmit={(e) => {
					e.preventDefault();
					createEnvironment();
				}}
			>
				<DialogHeader>
					<DialogTitle>New environment</DialogTitle>
				</DialogHeader>
				<div class="px-5 pb-5">
					<FormField label="Environment name">
						<Input
							type="text"
							bind:value={newEnvName}
							placeholder="e.g. staging, preview"
							required
						/>
					</FormField>
					{#if envError}
						<p class="mt-2 text-sm text-danger">{envError}</p>
					{/if}
				</div>
				<DialogFooter>
					<Button
						type="button"
						variant="secondary"
						size="sm"
						onclick={() => (envDialogOpen = false)}
					>
						Cancel
					</Button>
					<Button type="submit" size="sm" loading={creatingEnv}>
						{creatingEnv ? 'Creating...' : 'Create'}
					</Button>
				</DialogFooter>
			</form>
		</DialogContent>
	</Dialog>
{/if}
