<script lang="ts">
	import { page } from '$app/state';
	import { replaceState } from '$app/navigation';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';
	import FormField from '$lib/components/app/FormField.svelte';
	import ServiceWorkspace from '$lib/components/app/ServiceWorkspace.svelte';
	import StarterPanel, { type Starter } from '$lib/components/app/StarterPanel.svelte';
	import ImageStarterForm from '$lib/components/app/ImageStarterForm.svelte';
	import PendingChangesBar from '$lib/components/app/PendingChangesBar.svelte';
	import { toast } from '$lib/components/ui/toast/toast-service.svelte.js';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import { Dialog, DialogContent, DialogHeader, DialogFooter, DialogTitle } from '$lib/components/ui/dialog';
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
	import { quintOut } from 'svelte/easing';
	import type { TransitionConfig } from 'svelte/transition';

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

	// Adding a service is a step on the canvas, not a dialog — the same image-first
	// flow as /projects/new/image. Everything else (name, container name) is
	// derived from the image server-side, so there is nothing left to fill in.
	let addingService = $state(false);
	// Latches on the first step into the form and stays on, so the canvas has a
	// direction to come back from. Without it the very first paint of a project
	// would slide in from the left, explaining a step that never happened.
	let steppedIntoForm = $state(false);
	let svcError = $state('');
	let creatingService = $state(false);
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
	let selectedService = $derived(selectedServiceId ? (services.find((s) => s.id === selectedServiceId) ?? null) : null);

	// The panel and the canvas shift are one gesture — the drawer pushing the
	// workspace over — so they share a duration and a curve. Percentage travel
	// rather than pixels: the panel is a full-width sheet on mobile and a
	// clamp(420px, 55vw, 960px) column on desktop, and both should arrive from
	// exactly their own edge. quintOut is svelte/easing's match for the
	// cubic-bezier(0.23, 1, 0.32, 1) the rest of the builder animates on.
	const drawer: (node: Element) => TransitionConfig = () => ({
		duration: 280,
		easing: quintOut,
		// Its own width *plus* the 12px it floats off the right edge, so it clears
		// the frame completely instead of leaving a sliver parked at the gap.
		css: (_t: number, u: number) => `transform: translate3d(calc(${u} * (100% + 0.75rem)), 0, 0)`
	});

	// Services whose deployment we have already fired. The API keeps reporting
	// them as pending until the deployment actually succeeds, so without this the
	// bar would slide straight back in on the next poll.
	let deployingIds = $state<Set<string>>(new Set());
	let deploying = $state(false);
	// serviceId -> the deployment the bar started for it, so the side panel can
	// stream its logs. Without this a bar deploy runs completely invisibly: the
	// panel only ever knew about deployments it started itself.
	let barDeploymentIds = $state<Record<string, string>>({});

	let unfiredPending = $derived(envServices.filter((s) => s.has_pending_changes && !deployingIds.has(s.id)));
	// One primary action at a time: while the canvas is asking for an image, or a
	// dialog is up, the bar stays out of the way rather than competing with it.
	let barSuppressed = $derived(!canEdit || addingService || envDialogOpen);

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

	function startAddService() {
		if (!selectedEnvId) return;
		svcError = '';
		if (!svcServerId && servers.length > 0) svcServerId = servers[0].id;
		steppedIntoForm = true;
		addingService = true;
	}

	async function createService(image: string, port: number) {
		if (!selectedEnvId || creatingService) return;
		svcError = '';
		creatingService = true;
		try {
			const { data, error: err } = await api.POST('/api/services', {
				body: {
					// Blank name and container name: the API derives both from the
					// image, the same way POST /api/projects/from-image does.
					name: '',
					image,
					container_name: '',
					port,
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
				addingService = false;
				pan.recenter();
			}
		} catch {
			svcError = 'Network error';
		} finally {
			creatingService = false;
		}
	}

	function removeService(id: string) {
		services = services.filter((s) => s.id !== id);
		if (selectedServiceId === id) selectedServiceId = null;
		// A deleted service never leaves the "deploying" set on its own — the poll
		// only keeps ids it still sees as pending, and a gone service reports nothing.
		deployingIds = new Set([...deployingIds].filter((x) => x !== id));
		const rest = { ...barDeploymentIds };
		delete rest[id];
		barDeploymentIds = rest;
		toast.show({ tone: 'success', title: 'Service deleted', duration: 4000 });
	}

	// ponytail: fixed 3s poll with a hard cap, because a deployment's only precise
	// completion signal is the per-deployment SSE stream and subscribing to N of
	// them to dim N badges is not worth it. The cap is what keeps this honest: a
	// deployment that fails or outlives it releases its service, and the change
	// reappears as pending instead of staying silently hidden. Swap in the SSE
	// subscription if the badge ever needs to be exact.
	let pollTimer: ReturnType<typeof setInterval> | null = null;
	let pollDeadline = 0;

	function stopWatchingDeployments() {
		if (pollTimer === null) return;
		clearInterval(pollTimer);
		pollTimer = null;
	}

	function watchDeployments() {
		pollDeadline = Date.now() + 120_000;
		if (pollTimer !== null) return;
		pollTimer = setInterval(async () => {
			const svcs = await loadServices(projectId);
			services = svcs;
			const expired = Date.now() > pollDeadline;
			const stillPending = new Set(svcs.filter((s) => s.has_pending_changes).map((s) => s.id));
			deployingIds = expired ? new Set() : new Set([...deployingIds].filter((id) => stillPending.has(id)));
			if (deployingIds.size === 0) stopWatchingDeployments();
		}, 3000);
	}

	async function deployPending() {
		if (deploying || unfiredPending.length === 0) return;
		deploying = true;
		const ids = unfiredPending.map((s) => s.id);
		try {
			const results = await Promise.all(ids.map((id) => api.POST('/api/deployments', { body: { service_id: id } })));
			const started: string[] = [];
			const nextDeploymentIds = { ...barDeploymentIds };
			ids.forEach((id, i) => {
				const dep = results[i].data;
				if (results[i].error || !dep) return;
				started.push(id);
				nextDeploymentIds[id] = dep.deployment_id;
			});
			if (started.length === 0) {
				toast.show({
					tone: 'error',
					title: ids.length === 1 ? 'Could not start the deployment' : 'Could not start deployments',
					description: (results[0]?.error as { error: string } | undefined)?.error,
					duration: 6000
				});
				return;
			}
			barDeploymentIds = nextDeploymentIds;
			deployingIds = new Set([...deployingIds, ...started]);
			watchDeployments();
			// Deploying one service from the bar hides the bar and leaves only a badge
			// on the node, so the logs are a click away with nothing pointing at them.
			// Opening the panel puts the thing the user just asked for in front of
			// them. With several services there is no non-arbitrary one to pick, so
			// the badges carry it instead.
			if (started.length === 1) selectedServiceId = started[0];
			toast.show({
				tone: 'success',
				title: started.length === 1 ? 'Deploying 1 service' : `Deploying ${started.length} services`,
				description: started.length < ids.length ? `${ids.length - started.length} could not be started.` : undefined,
				icon: { kind: 'heroicon', src: CheckCircle },
				duration: 4500
			});
		} catch {
			toast.show({ tone: 'error', title: 'Network error', duration: 6000 });
		} finally {
			deploying = false;
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

		startAddService();

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
		if (starter === 'docker-image') startAddService();
	}

	const pan = createCanvasPan({ bounds: 'auto' });
	const panViewport = pan.viewport;

	$effect(() => {
		const id = projectId;
		let cancelled = false;

		project = null;
		environments = [];
		services = [];
		addingService = false;
		steppedIntoForm = false;
		selectedEnvId = '';
		selectedServiceId = null;
		svcServerId = '';
		loaded = false;
		stopWatchingDeployments();
		deployingIds = new Set();
		barDeploymentIds = {};

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
			stopWatchingDeployments();
		};
	});

	const topbar = useBuilderTopbar();

	$effect(() => {
		if (!topbar) return;
		topbar.label = '';
		topbar.leading = leadingSnippet;
		// The canvas owns "add a service" whenever it can say it better: the
		// StarterPanel in the empty state, the form itself once you're in it. The
		// topbar only takes over when the canvas has nodes on it and no longer
		// offers the action anywhere.
		topbar.action = canEdit && selectedEnvId && !addingService && envServices.length > 0 ? actionSnippet : null;
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

<svelte:head>
	<title>{project ? `${project.name} · Uploy` : 'Project · Uploy'}</title>
</svelte:head>

{#snippet leadingSnippet()}
	<div class="flex min-w-0 items-center gap-2">
		{#if !loaded}
			<!-- A bar, not the literal word "Project": the fallback label reads as the
			     real name until it swaps, which is a content flash, not a load. -->
			<div class="h-4 w-28 animate-pulse rounded bg-muted"></div>
		{:else}
			<span class="truncate text-sm font-semibold text-foreground">
				{project?.name ?? 'Project'}
			</span>
		{/if}
		{#if environments.length > 0}
			<span class="text-muted-foreground/60">/</span>
			<DropdownMenu.Root>
				<!-- Borderless, like the workspace switcher and every other control in this
				     bar. A bordered white pill on a near-white topbar reads as an input
				     sitting inside a text breadcrumb; the chevron already says "menu". The
				     background has to persist on open, or the trigger goes flat the moment
				     the pointer moves into its own menu. -->
				<DropdownMenu.Trigger
					class="-ml-1 inline-flex h-7 cursor-pointer items-center gap-1.5 rounded-md px-2 text-xs font-medium text-foreground transition-colors outline-none hover:bg-accent hover:text-accent-foreground focus-visible:ring-2 focus-visible:ring-ring/40 data-[state=open]:bg-accent data-[state=open]:text-accent-foreground"
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
	<Button size="sm" onclick={startAddService}>
		<Icon src={Plus} theme="outline" class="h-3.5 w-3.5" />
		Add service
	</Button>
{/snippet}

<div
	class="builder-shell relative flex min-h-0 w-full flex-1 gap-3"
	data-inspector-open={selectedService ? 'true' : undefined}
>
	<div
		class="canvas viewport relative flex min-h-0 flex-1 overflow-hidden rounded-xl border border-border"
		data-panning={pan.isPanning ? 'true' : 'false'}
		use:panViewport
	>
		<div
			class="canvas-bg"
			aria-hidden="true"
			style="background-size: {12 * pan.scale}px {12 * pan.scale}px; background-position: {pan.x - pan.scale}px {pan.y -
				pan.scale}px;"
		></div>

		<div class="scroll-area relative z-10 flex min-h-0 w-full flex-1 overflow-x-hidden overflow-y-auto">
			<div
				class="stage inspector-shift m-auto flex min-h-full w-full items-center justify-center px-4 py-8 sm:px-6 sm:py-12"
			>
				<div
					class="world flex w-full items-center justify-center"
					style="transform: translate3d({pan.x}px, {pan.y}px, 0) scale({pan.scale});"
				>
					{#if !loaded}
						<!-- Ghost nodes in the real node geometry, so the canvas resolves in
						     place instead of swapping a centred spinner for a grid. bg-card
						     with muted bars inside, not bare muted blocks: --muted and
						     --canvas are the same value out here, so the node surface is what
						     carries the shape. Two, because that is the honest guess — enough
						     to say "services are coming", not enough to promise a count. -->
						<div
							class="grid gap-4"
							style="grid-template-columns: repeat(2, minmax(220px, 240px));"
							data-no-pan
							role="status"
							aria-label="Loading project"
						>
							{#each [0, 1] as i (i)}
								<div class="skeleton-node flex flex-col gap-2 rounded-lg border border-border bg-card p-3">
									<div class="flex items-center gap-2">
										<div class="h-7 w-7 flex-none animate-pulse rounded-md bg-muted"></div>
										<div class="flex min-w-0 flex-1 flex-col gap-1">
											<div class="h-3.5 w-24 animate-pulse rounded bg-muted"></div>
											<div class="h-2.5 w-32 animate-pulse rounded bg-muted"></div>
										</div>
									</div>
									<div class="flex h-4 items-center justify-between">
										<div class="h-2.5 w-12 animate-pulse rounded bg-muted"></div>
										<div class="h-2.5 w-16 animate-pulse rounded bg-muted"></div>
									</div>
								</div>
							{/each}
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
							<!-- bg-card, not a muted tint: the canvas is now that same muted value,
							     so a tint over it would fill with nothing. Dashed edge still says
							     "placeholder"; the white surface says "this is a node". -->
							<div class="rounded-xl border border-dashed border-border bg-card p-8">
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
					{:else if addingService}
						<div class="step-enter w-full max-w-105" data-no-pan>
							{#if servers.length === 0}
								<EmptyState
									icon={ServerStack}
									title="Connect a server first"
									description="Uploy needs a server with SSH access before it can deploy a service."
								>
									{#snippet actions()}
										{#if isOwner}
											<Button href="/servers" size="sm">Connect a server</Button>
										{/if}
										<Button type="button" variant="secondary" size="sm" onclick={() => (addingService = false)}>
											Cancel
										</Button>
									{/snippet}
								</EmptyState>
							{:else}
								<ImageStarterForm
									submitting={creatingService}
									error={svcError}
									submitLabel="Create service"
									backLabel="Back to canvas"
									onBack={() => (addingService = false)}
									onSubmit={createService}
								>
									{#snippet details()}
										<div class="flex items-center justify-between gap-3">
											<span class="text-xs text-muted-foreground">Server</span>
											<Select items={serverItems} bind:value={svcServerId} size="sm" class="w-56" />
										</div>

										<div class="flex items-baseline justify-between gap-3 text-xs">
											<span class="flex-none text-muted-foreground">Deploys to</span>
											<span class="truncate font-medium text-foreground">{selectedEnv?.name}</span>
										</div>
									{/snippet}
								</ImageStarterForm>
							{/if}
						</div>
					{:else if envServices.length === 0}
						<div class="flex w-full max-w-105 flex-col gap-2" class:step-back={steppedIntoForm} data-no-pan>
							{#if canEdit}
								<StarterPanel enabled={{ 'docker-image': true }} title="Add a service" onSelect={handleStarterSelect} />
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
							class:step-back={steppedIntoForm}
							style="grid-template-columns: repeat({cols}, minmax(220px, 240px));"
							data-no-pan
						>
							{#each envServices as svc (svc.id)}
								{@const srv = serverById.get(svc.server_id)}
								{@const isSelected = svc.id === selectedServiceId}
								{@const isDeploying = deployingIds.has(svc.id)}
								<!-- A node with undeployed changes is a plan, not a running thing, so
								     its edge goes dashed — the same "not real yet" grammar the empty
								     environment placeholder already uses on this canvas. Selection
								     still wins the border outright: that is the state the user is
								     actively driving, and two dashed-vs-solid signals on one edge
								     would just cancel each other out. -->
								{@const isPending = svc.has_pending_changes && !isDeploying}
								<!-- items-stretch is load-bearing, not decoration: the UA stylesheet sets
								     align-items: flex-start on <button>, so as a flex column its rows
								     shrink to their own content and the justify-between further down has
								     no free space left to distribute — which is what ran "Port 8080"
								     straight into the server name. -->
								<button
									type="button"
									onclick={() => (selectedServiceId = svc.id)}
									class="service-node group flex flex-col items-stretch gap-2 rounded-lg border bg-card p-3 text-left text-card-foreground transition-shadow hover:shadow-md {isSelected
										? 'border-foreground shadow-md'
										: isPending
											? 'border-dashed border-foreground/25'
											: 'border-border'}"
								>
									<div class="flex items-center gap-2">
										<span class="grid h-7 w-7 flex-none place-content-center rounded-md bg-muted text-foreground">
											<Container class="h-3.5 w-3.5" strokeWidth={1.75} />
										</span>
										<div class="min-w-0 flex-1">
											<div class="truncate text-sm font-semibold text-foreground">{svc.name}</div>
											<div class="truncate font-mono text-[11px] text-muted-foreground">
												{svc.image}
											</div>
										</div>
										{#if isDeploying}
											<span class="node-badge is-deploying">Deploying</span>
										{:else if isPending}
											<!-- "New" and "Modified" both name what the change *is*, matching the
											     review dialog's "will be added" / "will be updated". "Pending"
											     named the queue instead, which read as a different axis from
											     "New" and said nothing you could act on. -->
											<span class="node-badge">{svc.has_deployed ? 'Modified' : 'New'}</span>
										{/if}
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

		<!-- Chrome, not canvas content: it sits outside .world so it holds its place
		     while the canvas pans and zooms underneath it. -->
		<PendingChangesBar
			services={unfiredPending}
			suppressed={barSuppressed}
			{deploying}
			onDeploy={deployPending}
			onSelect={(id) => (selectedServiceId = id)}
		/>

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
			<button type="button" class="tool-btn zoom-label" onclick={() => pan.resetZoom()} aria-label="Reset zoom to 100%">
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
			<button type="button" class="tool-btn" onclick={() => pan.recenter()} aria-label="Recenter canvas">
				<Icon src={ArrowsPointingIn} theme="outline" class="h-3.5 w-3.5" />
			</button>
		</div>
	</div>

	{#if selectedService}
		<aside
			transition:drawer
			class="side-panel absolute inset-0 z-30 flex min-h-0 w-full flex-col overflow-hidden rounded-xl border border-border bg-card text-card-foreground md:inset-y-3 md:right-3 md:left-auto md:w-[var(--inspector-width)]"
		>
			<!-- px-5, matching the tab row and content below it: the title used to sit a
			     notch left of everything it introduced. The icon chip is the canvas
			     node's own, so the panel reads as that node opened up rather than as a
			     separate screen. The image line is gone — it leads the Deployments tab
			     and has its own row in Settings, and three restatements of one string in
			     a 420px column is two too many. -->
			<header class="flex items-center justify-between gap-2 border-b border-border px-5 py-4">
				<div class="flex min-w-0 items-center gap-3">
					<span class="grid h-9 w-9 flex-none place-content-center rounded-md bg-muted text-foreground">
						<Container class="h-4.5 w-4.5" strokeWidth={1.75} />
					</span>
					<h2 class="truncate text-2xl font-semibold tracking-[-0.01em] text-foreground">
						{selectedService.name}
					</h2>
				</div>
				<button
					type="button"
					onclick={() => (selectedServiceId = null)}
					class="grid h-11 w-11 cursor-pointer place-content-center rounded-md text-muted-foreground hover:bg-accent hover:text-accent-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-card focus-visible:outline-none md:h-8 md:w-8"
					aria-label="Close service panel"
				>
					<Icon src={XMark} theme="outline" class="h-4.5 w-4.5" />
				</button>
			</header>
			<div class="min-h-0 flex-1 overflow-hidden">
				<ServiceWorkspace
					service={selectedService}
					{canEdit}
					{isOwner}
					externalDeploymentId={barDeploymentIds[selectedService.id] ?? null}
					onDeleted={removeService}
				/>
			</div>
		</aside>
	{/if}
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
					<Input type="text" bind:value={newEnvName} placeholder="e.g. staging, preview" required />
				</FormField>
				{#if envError}
					<p class="mt-2 text-sm text-destructive">{envError}</p>
				{/if}
			</div>
			<DialogFooter>
				<Button type="button" variant="secondary" size="sm" onclick={() => (envDialogOpen = false)}>Cancel</Button>
				<Button type="submit" size="sm" loading={creatingEnv}>
					{creatingEnv ? 'Creating...' : 'Create'}
				</Button>
			</DialogFooter>
		</form>
	</DialogContent>
</Dialog>

<style>
	.canvas {
		background-color: var(--canvas);
	}

	.builder-shell {
		--inspector-width: clamp(420px, 55vw, 960px);
		--inspector-shift: 0px;
	}

	.inspector-shift {
		transform: translate3d(calc(0px - var(--inspector-shift)), 0, 0);
		transition: transform 280ms cubic-bezier(0.23, 1, 0.32, 1);
		will-change: transform;
	}

	@media (min-width: 768px) {
		/* Half the panel, plus half the 12px it floats off the edge — what the
		   canvas has to give up is the panel *and* its gap, so the remaining area
		   only stays centred if the shift accounts for both. */
		.builder-shell[data-inspector-open='true'] {
			--inspector-shift: calc(clamp(210px, 27.5vw, 480px) + 6px);
		}
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
		background-image: radial-gradient(circle at 1px 1px, var(--canvas-dot) 1px, transparent 0);
		background-size: 12px 12px;
		pointer-events: none;
	}

	.world {
		transform-origin: center center;
		will-change: transform;
	}

	/* Stepping into the add-service form pushes the canvas left, so the form
	   enters from the right; coming back, the canvas returns from the left. Same
	   200ms curve as /projects/new/image and the dialog, so the whole builder
	   shares one motion vocabulary. Applied to the branch wrapper, not .world —
	   .world owns the pan transform. `backwards` holds the start offset on the
	   first frame instead of flashing the end state. */
	.step-enter {
		animation: step-in-from-right 200ms cubic-bezier(0.23, 1, 0.32, 1) backwards;
	}

	.step-back {
		animation: step-in-from-left 200ms cubic-bezier(0.23, 1, 0.32, 1) backwards;
	}

	@keyframes step-in-from-right {
		from {
			opacity: 0;
			transform: translateX(12px);
		}
		to {
			opacity: 1;
			transform: translateX(0);
		}
	}

	@keyframes step-in-from-left {
		from {
			opacity: 0;
			transform: translateX(-12px);
		}
		to {
			opacity: 1;
			transform: translateX(0);
		}
	}

	.service-node,
	.skeleton-node {
		box-shadow: 0 1px 0 rgba(17, 17, 17, 0.04);
	}

	.service-node {
		cursor: pointer;
	}

	.node-badge {
		flex: none;
		align-self: flex-start;
		padding: 0.0625rem 0.375rem;
		border-radius: var(--radius-sm);
		background: var(--muted);
		color: var(--muted-foreground);
		font-size: 0.625rem;
		font-weight: 500;
		line-height: 1.4;
		letter-spacing: 0.01em;
	}

	/* Deploying is the one node state that is genuinely mid-flight, so it is the
	   one that gets motion. Opacity only — a pulsing background on a chip this
	   small reads as a rendering bug. */
	.node-badge.is-deploying {
		animation: badge-pulse 1.6s ease-in-out infinite;
	}

	@keyframes badge-pulse {
		0%,
		100% {
			opacity: 1;
		}
		50% {
			opacity: 0.55;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.node-badge.is-deploying {
			animation: none;
		}
	}

	/* Floats over the canvas rather than sitting beside it, so it takes the
	   --shadow-float exception. The lift stays a whisper on purpose: the panel is
	   #fff on a #f6f6f6 canvas with a hairline border, and that value step is
	   already most of the separation — the shadow only has to say which plane it
	   is on. */
	.side-panel {
		box-shadow: var(--shadow-float);
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
