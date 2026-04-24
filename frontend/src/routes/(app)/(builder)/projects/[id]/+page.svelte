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
	<div class="flex w-full justify-center">
		<section
			class="w-full max-w-3xl rounded-2xl border border-border bg-card p-6 text-card-foreground shadow-sm sm:p-8"
		>
			<header class="flex flex-wrap items-center justify-between gap-3">
				<div class="min-w-0">
					<h2 class="truncate text-xl font-semibold text-foreground">{project.name}</h2>
					<p class="mt-1 text-sm text-muted-foreground">
						Manage environments and services for this project.
					</p>
				</div>
				<div class="flex items-center gap-2">
					{#if environments.length > 0}
						<DropdownMenu.Root>
							<DropdownMenu.Trigger
								class="inline-flex h-8 items-center gap-1.5 rounded-md border border-border bg-card px-2.5 text-xs font-medium text-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
							>
								<Icon src={Squares2x2} theme="outline" class="h-3.5 w-3.5 text-muted-foreground" />
								<span>{selectedEnv?.name ?? 'Select environment'}</span>
								<Icon src={ChevronDown} theme="outline" class="h-3 w-3 text-muted-foreground" />
							</DropdownMenu.Trigger>
							<DropdownMenu.Portal>
								<DropdownMenu.Content
									align="end"
									sideOffset={6}
									class="z-50 min-w-48 overflow-hidden rounded-lg border border-border bg-popover p-1 text-popover-foreground shadow-overlay"
								>
									{#each environments as env (env.id)}
										<DropdownMenu.Item
											onSelect={() => (selectedEnvId = env.id)}
											class="flex cursor-pointer items-center justify-between gap-2 rounded-md px-2 py-1.5 text-sm text-foreground outline-none data-highlighted:bg-accent data-highlighted:text-accent-foreground"
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
											class="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm text-muted-foreground outline-none data-highlighted:bg-accent data-highlighted:text-accent-foreground"
										>
											<Icon src={Plus} theme="outline" class="h-3.5 w-3.5" />
											<span>New environment</span>
										</DropdownMenu.Item>
									{/if}
								</DropdownMenu.Content>
							</DropdownMenu.Portal>
						</DropdownMenu.Root>
					{/if}

					{#if canEdit && selectedEnvId && !showServiceForm}
						<Button size="sm" onclick={openServiceForm}>
							<Icon src={Plus} theme="outline" class="h-3.5 w-3.5" />
							Add service
						</Button>
					{/if}
				</div>
			</header>

			<div class="mt-6">
				{#if !loaded}
					<div class="flex flex-col gap-2">
						<div class="h-14 animate-pulse rounded-lg bg-muted"></div>
						<div class="h-14 animate-pulse rounded-lg bg-muted"></div>
					</div>
				{:else if environments.length === 0}
					<div class="rounded-xl border border-dashed border-border bg-muted/30 p-8">
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
							<p class="mt-2 text-center text-sm text-destructive">{envError}</p>
						{/if}
					</div>
				{:else}
					{#if canEdit && showServiceForm}
						<form
							onsubmit={(e) => {
								e.preventDefault();
								createService();
							}}
							class="mb-4 flex w-full flex-col gap-3 rounded-xl border border-border bg-muted/40 p-5"
						>
							<div class="flex items-baseline justify-between">
								<h3 class="text-sm font-semibold text-foreground">Add service</h3>
								<span class="text-xs text-muted-foreground">in {selectedEnv?.name}</span>
							</div>
							{#if servers.length === 0}
								<div
									class="rounded-md border border-dashed border-border bg-card p-3 text-xs text-muted-foreground"
								>
									You need to connect a server before you can deploy a service.
									{#if isOwner}
										<div class="mt-2">
											<Button href="/servers" size="xs" variant="secondary">
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
								<p class="text-sm text-destructive">{svcError}</p>
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
							<div class="rounded-xl border border-dashed border-border bg-muted/30 p-8">
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
							</div>
						{/if}
					{:else}
						<ul class="flex flex-col divide-y divide-border rounded-xl border border-border">
							{#each envServices as svc (svc.id)}
								{@const srv = serverById.get(svc.server_id)}
								<li>
									<!-- eslint-disable svelte/no-navigation-without-resolve -->
									<a
										href="/services/{svc.id}"
										class="group flex items-center gap-3 px-4 py-3 transition-colors first:rounded-t-xl last:rounded-b-xl hover:bg-accent/60 hover:text-accent-foreground"
									>
										<div
											class="grid h-9 w-9 flex-none place-content-center rounded-md bg-muted"
										>
											<Icon src={Cube} theme="outline" class="h-4 w-4 text-foreground" />
										</div>
										<div class="min-w-0 flex-1">
											<div class="truncate font-medium text-foreground">{svc.name}</div>
											<div class="truncate font-mono text-xs text-muted-foreground">
												{svc.image}
											</div>
										</div>
										<div
											class="hidden flex-none items-center gap-3 text-xs text-muted-foreground sm:flex"
										>
											<span>Port {svc.port}</span>
											{#if srv}
												<span class="max-w-32 truncate">{srv.name}</span>
											{/if}
										</div>
									</a>
									<!-- eslint-enable svelte/no-navigation-without-resolve -->
								</li>
							{/each}
						</ul>
					{/if}
				{/if}
			</div>
		</section>
	</div>

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
						<p class="mt-2 text-sm text-destructive">{envError}</p>
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
