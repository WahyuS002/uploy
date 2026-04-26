<script lang="ts">
	import { page } from '$app/state';
	import { replaceState } from '$app/navigation';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';
	import FormField from '$lib/components/app/FormField.svelte';
	import ServiceWorkspace from '$lib/components/app/ServiceWorkspace.svelte';
	import StarterPanel, { type Starter } from '$lib/components/app/StarterPanel.svelte';
	import { toast } from '$lib/components/ui/toast/toast-service.svelte.js';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import Spinner from '$lib/components/ui/Spinner.svelte';
	import {
		Dialog,
		DialogContent,
		DialogHeader,
		DialogFooter,
		DialogTitle
	} from '$lib/components/ui/dialog';
	import { createCanvasPan } from '$lib/actions/canvas-pan.svelte';
	import { useBuilderTopbar } from '$lib/components/builder-topbar-context';
	import { DropdownMenu } from 'bits-ui';
	import { Icon } from '@steeze-ui/svelte-icon';
	import {
		ChevronDown,
		Cube,
		Plus,
		ServerStack,
		Squares2x2,
		Check,
		CheckCircle,
		XMark,
		Minus,
		ArrowsPointingIn
	} from '@steeze-ui/heroicons';
	import { Container } from 'lucide-svelte';

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
	let selectedServiceId = $state<string | null>(null);

	let serviceDialogOpen = $state(false);
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
		selectedEnvId
			? services
					.filter((s) => s.environment_id === selectedEnvId)
					.slice()
					.sort((a, b) => a.created_at.localeCompare(b.created_at))
			: []
	);
	let serverItems = $derived(servers.map((s) => ({ value: s.id, label: `${s.name} (${s.host})` })));
	let serverById = $derived(new Map(servers.map((s) => [s.id, s])));
	let selectedService = $derived(
		selectedServiceId ? (services.find((s) => s.id === selectedServiceId) ?? null) : null
	);

	async function loadProject(id: string) {
		const { data } = await api.GET('/api/projects/{id}', { params: { path: { id } } });
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

	function selectEnvironment(id: string) {
		if (id === selectedEnvId) return;
		selectedEnvId = id;
		selectedServiceId = null;
	}

	function openServiceDialog() {
		if (!selectedEnvId) return;
		svcError = '';
		svcName = '';
		svcContainerName = '';
		svcImage = 'nginx:latest';
		svcPort = 8080;
		if (!svcServerId && servers.length > 0) svcServerId = servers[0].id;
		serviceDialogOpen = true;
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
				services = [...services, data];
				selectedServiceId = data.id;
				serviceDialogOpen = false;
				pan.recenter();
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
				selectEnvironment(data.id);
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
				selectEnvironment(data.id);
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

		openServiceDialog();

		const cleanUrl = new URL(page.url);
		cleanUrl.searchParams.delete('starter');
		// eslint-disable-next-line svelte/no-navigation-without-resolve
		replaceState(cleanUrl.pathname + (cleanUrl.search ? cleanUrl.search : ''), page.state);
	}

	function consumeToastFlash() {
		const flash = page.state.toastFlash;
		if (!flash) return;
		toast.show({
			tone: flash.tone ?? 'success',
			title: flash.title,
			description: flash.description,
			icon: { kind: 'heroicon', src: CheckCircle },
			duration: 4500
		});
		const nextState = { ...page.state };
		delete nextState.toastFlash;
		// eslint-disable-next-line svelte/no-navigation-without-resolve
		replaceState(page.url.pathname + page.url.search, nextState);
	}

	function handleStarterSelect(starter: Starter) {
		if (starter === 'docker-image') openServiceDialog();
	}

	const pan = createCanvasPan({ bounds: 'auto' });
	const panViewport = pan.viewport;

	$effect(() => {
		const id = projectId;
		let cancelled = false;

		project = null;
		environments = [];
		services = [];
		serviceDialogOpen = false;
		selectedEnvId = '';
		selectedServiceId = null;
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
			if (proj) consumeToastFlash();
			applyStarter();
		})();

		return () => {
			cancelled = true;
		};
	});

	const topbar = useBuilderTopbar();

	$effect(() => {
		if (!topbar) return;
		topbar.label = '';
		topbar.leading = leadingSnippet;
		topbar.action = canEdit && selectedEnvId ? actionSnippet : null;
		return () => {
			topbar.label = '';
			topbar.leading = null;
			topbar.action = null;
		};
	});

	function nodeColumnsFor(count: number): number {
		if (count <= 1) return 1;
		if (count <= 4) return 2;
		if (count <= 9) return 3;
		return 4;
	}
</script>

{#snippet leadingSnippet()}
	<div class="flex min-w-0 items-center gap-2">
		<span class="truncate text-sm font-semibold text-foreground">
			{project?.name ?? 'Project'}
		</span>
		{#if environments.length > 0}
			<span class="text-muted-foreground/60">/</span>
			<DropdownMenu.Root>
				<DropdownMenu.Trigger
					class="inline-flex h-7 items-center gap-1.5 rounded-md border border-border bg-card px-2 text-xs font-medium text-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
				>
					<Icon src={Squares2x2} theme="outline" class="h-3.5 w-3.5 text-muted-foreground" />
					<span>{selectedEnv?.name ?? 'Select environment'}</span>
					<Icon src={ChevronDown} theme="outline" class="h-3 w-3 text-muted-foreground" />
				</DropdownMenu.Trigger>
				<DropdownMenu.Portal>
					<DropdownMenu.Content
						align="start"
						sideOffset={6}
						class="z-50 min-w-48 overflow-hidden rounded-lg border border-border bg-popover p-1 text-popover-foreground shadow-overlay"
					>
						{#each environments as env (env.id)}
							<DropdownMenu.Item
								onSelect={() => selectEnvironment(env.id)}
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
	</div>
{/snippet}

{#snippet actionSnippet()}
	<Button size="sm" onclick={openServiceDialog}>
		<Icon src={Plus} theme="outline" class="h-3.5 w-3.5" />
		Add service
	</Button>
{/snippet}

<div class="relative flex min-h-0 w-full flex-1 gap-3">
	<div
		class="canvas viewport relative flex min-h-0 flex-1 overflow-hidden rounded-xl border border-gray-200"
		data-panning={pan.isPanning ? 'true' : 'false'}
		use:panViewport
	>
		<div
			class="canvas-bg"
			aria-hidden="true"
			style="background-size: {12 * pan.scale}px {12 * pan.scale}px; background-position: {pan.x -
				pan.scale}px {pan.y - pan.scale}px;"
		></div>

		<div
			class="scroll-area relative z-10 flex min-h-0 w-full flex-1 overflow-x-hidden overflow-y-auto"
		>
			<div
				class="stage m-auto flex min-h-full w-full items-center justify-center px-4 py-8 sm:px-6 sm:py-12"
			>
				<div
					class="world flex w-full items-center justify-center"
					style="transform: translate3d({pan.x}px, {pan.y}px, 0) scale({pan.scale});"
				>
					{#if !loaded}
						<div data-no-pan class="flex items-center justify-center">
							<Spinner class="h-6 w-6 text-muted-foreground" />
						</div>
					{:else if !project}
						<div class="w-full max-w-105" data-no-pan>
							<EmptyState
								icon={Cube}
								title="Project not found"
								description="It may have been deleted or you might not have access to it."
							>
								{#snippet actions()}
									<Button href="/projects" variant="secondary" size="sm">Back to projects</Button>
								{/snippet}
							</EmptyState>
						</div>
					{:else if environments.length === 0}
						<div class="w-full max-w-md" data-no-pan>
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
						</div>
					{:else if envServices.length === 0}
						<div class="flex w-full max-w-105 flex-col gap-2" data-no-pan>
							{#if canEdit}
								<StarterPanel
									enabled={{ 'docker-image': true }}
									placeholder="What would you like to deploy?"
									onSelect={handleStarterSelect}
								/>
							{:else}
								<EmptyState
									icon={Cube}
									title="No services in {selectedEnv?.name}"
									description="A workspace owner or developer can deploy a service to this environment."
								/>
							{/if}
						</div>
					{:else}
						{@const cols = nodeColumnsFor(envServices.length)}
						<div
							class="grid gap-4"
							style="grid-template-columns: repeat({cols}, minmax(220px, 240px));"
							data-no-pan
						>
							{#each envServices as svc (svc.id)}
								{@const srv = serverById.get(svc.server_id)}
								{@const isSelected = svc.id === selectedServiceId}
								<button
									type="button"
									onclick={() => (selectedServiceId = svc.id)}
									class="service-node group flex flex-col gap-2 rounded-lg border bg-card p-3 text-left text-card-foreground transition-shadow hover:shadow-md {isSelected
										? 'border-foreground shadow-md'
										: 'border-border'}"
								>
									<div class="flex items-center gap-2">
										<span
											class="grid h-7 w-7 flex-none place-content-center rounded-md bg-muted text-foreground"
										>
											<Container class="h-3.5 w-3.5" strokeWidth={1.75} />
										</span>
										<div class="min-w-0 flex-1">
											<div class="truncate text-sm font-semibold text-foreground">{svc.name}</div>
											<div class="truncate font-mono text-[11px] text-muted-foreground">
												{svc.image}
											</div>
										</div>
									</div>
									<div class="flex items-center justify-between text-[11px] text-muted-foreground">
										<span>Port {svc.port}</span>
										{#if srv}
											<span class="max-w-32 truncate">{srv.name}</span>
										{/if}
									</div>
								</button>
							{/each}
						</div>
					{/if}
				</div>
			</div>
		</div>

		<div class="toolbar" data-no-pan aria-label="Canvas controls">
			<button
				type="button"
				class="tool-btn"
				onclick={() => pan.zoomOut()}
				disabled={pan.scale <= 0.8}
				aria-label="Zoom out"
			>
				<Icon src={Minus} theme="outline" class="h-3.5 w-3.5" />
			</button>
			<button
				type="button"
				class="tool-btn zoom-label"
				onclick={() => pan.resetZoom()}
				aria-label="Reset zoom to 100%"
			>
				{Math.round(pan.scale * 100)}%
			</button>
			<button
				type="button"
				class="tool-btn"
				onclick={() => pan.zoomIn()}
				disabled={pan.scale >= 1.4}
				aria-label="Zoom in"
			>
				<Icon src={Plus} theme="outline" class="h-3.5 w-3.5" />
			</button>
			<span class="divider" aria-hidden="true"></span>
			<button
				type="button"
				class="tool-btn"
				onclick={() => pan.recenter()}
				aria-label="Recenter canvas"
			>
				<Icon src={ArrowsPointingIn} theme="outline" class="h-3.5 w-3.5" />
			</button>
		</div>
	</div>

	{#if selectedService}
		<aside
			class="side-panel absolute inset-0 z-30 flex min-h-0 w-full flex-col overflow-hidden rounded-xl border border-border bg-card text-card-foreground md:relative md:inset-auto md:z-auto md:w-[420px] md:max-w-[440px] md:min-w-[320px] md:flex-none"
		>
			<header class="flex items-center justify-between gap-2 border-b border-border px-4 py-3">
				<div class="min-w-0">
					<div class="truncate text-sm font-semibold text-foreground">{selectedService.name}</div>
					<div class="truncate font-mono text-[11px] text-muted-foreground">
						{selectedService.image}
					</div>
				</div>
				<button
					type="button"
					onclick={() => (selectedServiceId = null)}
					class="grid h-7 w-7 cursor-pointer place-content-center rounded-md text-muted-foreground hover:bg-accent hover:text-accent-foreground"
					aria-label="Close service panel"
				>
					<Icon src={XMark} theme="outline" class="h-4 w-4" />
				</button>
			</header>
			<div class="min-h-0 flex-1 overflow-hidden">
				<ServiceWorkspace service={selectedService} {canEdit} />
			</div>
		</aside>
	{/if}
</div>

<Dialog bind:open={serviceDialogOpen}>
	<DialogContent>
		<form
			onsubmit={(e) => {
				e.preventDefault();
				createService();
			}}
		>
			<DialogHeader>
				<DialogTitle>Add service</DialogTitle>
			</DialogHeader>
			<div class="px-5 pb-5">
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
					<div class="flex flex-col gap-3">
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
						<p class="text-xs text-muted-foreground">
							Deploys to <span class="font-medium text-foreground">{selectedEnv?.name}</span>
						</p>
					</div>
				{/if}

				{#if svcError}
					<p class="mt-3 text-sm text-destructive">{svcError}</p>
				{/if}
			</div>
			<DialogFooter>
				<Button
					type="button"
					variant="secondary"
					size="sm"
					onclick={() => (serviceDialogOpen = false)}
				>
					Cancel
				</Button>
				{#if servers.length > 0}
					<Button type="submit" size="sm" loading={creatingService}>
						{creatingService ? 'Creating...' : 'Create service'}
					</Button>
				{/if}
			</DialogFooter>
		</form>
	</DialogContent>
</Dialog>

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
					<Input type="text" bind:value={newEnvName} placeholder="e.g. staging, preview" required />
				</FormField>
				{#if envError}
					<p class="mt-2 text-sm text-destructive">{envError}</p>
				{/if}
			</div>
			<DialogFooter>
				<Button type="button" variant="secondary" size="sm" onclick={() => (envDialogOpen = false)}>
					Cancel
				</Button>
				<Button type="submit" size="sm" loading={creatingEnv}>
					{creatingEnv ? 'Creating...' : 'Create'}
				</Button>
			</DialogFooter>
		</form>
	</DialogContent>
</Dialog>

<style>
	.canvas {
		background-color: var(--background);
		box-shadow: var(--shadow-panel);
	}

	@media (pointer: fine) {
		.viewport {
			cursor: grab;
		}

		.viewport[data-panning='true'] {
			cursor: grabbing;
			user-select: none;
		}
	}

	.canvas-bg {
		position: absolute;
		inset: 0;
		background-image: radial-gradient(
			circle at 1px 1px,
			rgba(26, 27, 30, 0.125) 1px,
			transparent 0
		);
		background-size: 12px 12px;
		pointer-events: none;
	}

	.world {
		transform-origin: center center;
		will-change: transform;
	}

	.service-node {
		box-shadow: 0 1px 0 rgba(17, 17, 17, 0.04);
		cursor: pointer;
	}

	.side-panel {
		box-shadow: var(--shadow-panel);
	}

	.toolbar {
		display: none;
	}

	@media (pointer: fine) {
		.toolbar {
			position: absolute;
			bottom: 0.75rem;
			left: 0.75rem;
			z-index: 20;
			display: inline-flex;
			align-items: center;
			gap: 0.125rem;
			padding: 0.25rem;
			background: var(--card);
			border: 1px solid var(--border);
			border-radius: var(--radius-md);
			box-shadow: var(--shadow-panel);
			cursor: default;
		}

		.tool-btn {
			display: inline-flex;
			align-items: center;
			justify-content: center;
			min-width: 1.75rem;
			height: 1.75rem;
			padding: 0 0.375rem;
			border-radius: var(--radius-sm);
			font-size: 0.75rem;
			font-variant-numeric: tabular-nums;
			color: var(--muted-foreground);
			background: transparent;
			cursor: pointer;
			transition:
				background-color 120ms ease-out,
				color 120ms ease-out;
		}

		.tool-btn:hover:not(:disabled) {
			background: var(--accent);
			color: var(--accent-foreground);
		}

		.tool-btn:focus-visible {
			outline: none;
			box-shadow: 0 0 0 2px var(--ring);
			color: var(--foreground);
		}

		.tool-btn:disabled {
			opacity: 0.4;
			cursor: not-allowed;
		}

		.zoom-label {
			min-width: 2.5rem;
		}

		.divider {
			width: 1px;
			height: 1rem;
			margin: 0 0.125rem;
			background: var(--border);
		}
	}
</style>
