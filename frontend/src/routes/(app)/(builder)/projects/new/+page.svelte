<script lang="ts">
	import { goto } from '$app/navigation';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';
	import Button from '$lib/components/ui/Button.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import StarterPanel, { type Starter } from '$lib/components/app/StarterPanel.svelte';
	import StatusBadge from '$lib/components/app/StatusBadge.svelte';
	import Spinner from '$lib/components/ui/Spinner.svelte';
	import {
		Dialog,
		DialogContent,
		DialogHeader,
		DialogFooter,
		DialogTitle,
		DialogDescription
	} from '$lib/components/ui/dialog';
	import { toast } from '$lib/components/ui/toast/toast-service.svelte.js';
	import { createCanvasPan } from '$lib/actions/canvas-pan.svelte';
	import { Icon } from '@steeze-ui/svelte-icon';
	import {
		Server,
		Minus,
		Plus,
		ArrowsPointingIn,
		Clock,
		ExclamationCircle,
		ServerStack,
		Check
	} from '@steeze-ui/heroicons';

	type ProjectResponse = components['schemas']['ProjectResponse'];
	type ServerResponse = components['schemas']['ServerResponse'];

	let { data }: { data: PageData } = $props();
	let canEdit = $derived(data.workspace?.role === 'owner' || data.workspace?.role === 'developer');
	let isOwner = $derived(data.workspace?.role === 'owner');

	let busyStarter = $state<Starter | null>(null);
	let error = $state('');

	let serverDialogOpen = $state(false);
	let servers = $state<ServerResponse[]>([]);
	let serversLoaded = $state(false);
	let serversLoading = $state(false);
	let serversError = $state('');
	let pickedServerId = $state<string | null>(null);

	// Stable sort: ready first, then everything else. Within each bucket the
	// API's existing created_at DESC ordering is preserved.
	let sortedServers = $derived(
		servers.slice().sort((a, b) => {
			const aReady = a.proxy_status === 'ready' ? 0 : 1;
			const bReady = b.proxy_status === 'ready' ? 0 : 1;
			return aReady - bReady;
		})
	);
	let pickedServer = $derived(
		pickedServerId ? (sortedServers.find((s) => s.id === pickedServerId) ?? null) : null
	);

	async function createProject(): Promise<ProjectResponse | null> {
		const { data, error: err } = await api.POST('/api/projects', {
			body: {}
		});
		if (err) {
			error = (err as { error: string }).error ?? 'Failed to create project';
			return null;
		}
		return data ?? null;
	}

	async function ensureServersLoaded() {
		if (serversLoaded || serversLoading) return;
		serversLoading = true;
		serversError = '';
		try {
			const { data, error: err } = await api.GET('/api/servers');
			if (err) {
				serversError = (err as { error: string }).error ?? 'Failed to load servers';
				return;
			}
			servers = data ?? [];
			serversLoaded = true;
		} catch {
			serversError = 'Network error';
		} finally {
			serversLoading = false;
		}
	}

	async function openServerPicker() {
		pickedServerId = null;
		serverDialogOpen = true;
		await ensureServersLoaded();
		const ordered = sortedServers;
		if (serversLoaded && ordered.length > 0) pickedServerId = ordered[0].id;
	}

	function retryLoadServers() {
		serversError = '';
		void ensureServersLoaded();
	}

	async function confirmServerPick() {
		if (!pickedServerId) return;
		serverDialogOpen = false;
		// eslint-disable-next-line svelte/no-navigation-without-resolve
		await goto(`/projects/new/image?server_id=${encodeURIComponent(pickedServerId)}`);
	}

	async function launchEmptyProject() {
		const pendingId = toast.neutral({
			title: 'Creating empty project...',
			description: 'Please wait a moment.',
			icon: { kind: 'heroicon', src: Clock }
		});

		try {
			const minHold = new Promise((resolve) => setTimeout(resolve, 2000));

			let project: ProjectResponse | null = null;
			try {
				project = await createProject();
			} catch {
				error = 'Network error';
			}

			if (!project) {
				toast.dismiss(pendingId);
				toast.error({
					title: 'Failed to create project',
					description: error || 'Please try again.',
					icon: { kind: 'heroicon', src: ExclamationCircle },
					duration: 6000
				});
				return;
			}

			await minHold;
			toast.dismiss(pendingId);

			// eslint-disable-next-line svelte/no-navigation-without-resolve
			await goto(`/projects/${project.id}`, {
				state: {
					toastFlash: {
						tone: 'success',
						title: 'Project created successfully',
						description: 'Ready to build.'
					}
				}
			});
		} finally {
			busyStarter = null;
		}
	}

	async function launch(starter: Starter) {
		if (busyStarter) return;
		error = '';
		busyStarter = starter;

		if (starter === 'empty-project') {
			await launchEmptyProject();
			return;
		}

		if (starter === 'docker-image') {
			try {
				await openServerPicker();
			} finally {
				busyStarter = null;
			}
			return;
		}

		busyStarter = null;
	}

	const pan = createCanvasPan({ bounds: 'auto' });
	const panViewport = pan.viewport;
</script>

<div
	class="canvas viewport relative flex w-full flex-1 overflow-hidden rounded-xl border border-gray-200"
	data-panning={pan.isPanning ? 'true' : 'false'}
	use:panViewport
>
	<div
		class="canvas-bg"
		aria-hidden="true"
		style="background-size: {12 * pan.scale}px {12 * pan.scale}px; background-position: {pan.x -
			pan.scale}px {pan.y - pan.scale}px;"
	></div>

	<div class="scroll-area relative z-10 flex w-full overflow-x-hidden overflow-y-auto">
		<div
			class="stage m-auto flex min-h-full w-full items-center justify-center px-4 py-8 sm:px-6 sm:py-12"
		>
			<div
				class="world flex w-full items-center justify-center"
				style="transform: translate3d({pan.x}px, {pan.y}px, 0) scale({pan.scale});"
			>
				{#if !canEdit}
					<div class="w-full max-w-105" data-no-pan>
						<EmptyState
							icon={Server}
							title="You don't have permission to create projects"
							description="Ask a workspace owner or developer to create a project, or request a role change."
						>
							{#snippet actions()}
								<Button href="/projects" variant="secondary" size="sm">Back to projects</Button>
							{/snippet}
						</EmptyState>
					</div>
				{:else}
					<div class="flex w-full max-w-105 flex-col gap-2" data-no-pan>
						<StarterPanel
							{busyStarter}
							enabled={{ 'docker-image': true, 'empty-project': true }}
							onSelect={launch}
						/>

						{#if error}
							<p class="text-sm text-destructive">{error}</p>
						{/if}
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

<Dialog bind:open={serverDialogOpen}>
	<DialogContent
		showCloseButton={false}
		class="inset-y-auto top-[18vh] mt-0 mb-0 w-[min(92vw,560px)] max-w-none overflow-hidden rounded-2xl"
	>
		<DialogHeader class="border-b border-border px-5 pt-4 pr-5 pb-3">
			<DialogTitle class="text-sm">Pick a server</DialogTitle>
			<DialogDescription class="text-xs">
				Choose where to deploy this Docker image. You can connect more servers from the Servers
				page.
			</DialogDescription>
		</DialogHeader>

		<div class="max-h-[min(60vh,440px)] overflow-y-auto p-2">
			{#if serversLoading && !serversLoaded}
				<div class="flex items-center justify-center py-10">
					<Spinner class="h-5 w-5 text-muted-foreground" />
				</div>
			{:else if serversError}
				<div
					class="m-1 rounded-lg border border-destructive/20 bg-destructive/5 p-4 text-sm text-destructive"
				>
					<div class="text-sm font-semibold">Couldn't load servers</div>
					<div class="mt-1 text-xs text-destructive/80">{serversError}</div>
					<div class="mt-3">
						<Button type="button" size="xs" variant="secondary" onclick={retryLoadServers}>
							Retry
						</Button>
					</div>
				</div>
			{:else if sortedServers.length === 0}
				<div
					class="m-1 flex flex-col items-center gap-2 rounded-lg border border-dashed border-border bg-card px-4 py-8 text-center"
				>
					<span
						class="grid h-9 w-9 place-content-center rounded-full bg-muted text-muted-foreground"
						aria-hidden="true"
					>
						<Icon src={ServerStack} theme="outline" class="h-4 w-4" />
					</span>
					<div class="text-sm font-medium text-foreground">No servers connected yet</div>
					<p class="max-w-xs text-xs text-muted-foreground">
						{#if isOwner}
							Connect a server before you can deploy a Docker image to this workspace.
						{:else}
							Ask a workspace owner to connect a server before deploying.
						{/if}
					</p>
					{#if isOwner}
						<div class="mt-1">
							<Button href="/servers" size="xs" variant="secondary">
								<Icon src={ServerStack} theme="outline" class="h-3 w-3" />
								Add server
							</Button>
						</div>
					{/if}
				</div>
			{:else}
				<ul class="flex flex-col gap-1" role="listbox" aria-label="Servers">
					{#each sortedServers as server (server.id)}
						{@const selected = server.id === pickedServerId}
						<li>
							<button
								type="button"
								role="option"
								aria-selected={selected}
								onclick={() => (pickedServerId = server.id)}
								class="grid w-full cursor-pointer grid-cols-[auto_1fr_auto_auto] items-center gap-3 rounded-md px-2.5 py-2 text-left transition-colors {selected
									? 'bg-accent text-accent-foreground'
									: 'hover:bg-accent/70 hover:text-accent-foreground'}"
							>
								<span
									class="grid h-7 w-7 flex-none place-content-center rounded-md bg-muted text-foreground"
									aria-hidden="true"
								>
									<Icon src={ServerStack} theme="outline" class="h-3.5 w-3.5" />
								</span>
								<span class="min-w-0">
									<span class="block truncate text-sm font-medium text-foreground">
										{server.name}
									</span>
									<span class="block truncate font-mono text-[11px] text-muted-foreground">
										{server.host}:{server.port}
									</span>
								</span>
								<StatusBadge status={server.proxy_status} />
								<span class="grid h-4 w-4 place-content-center text-foreground" aria-hidden="true">
									{#if selected}
										<Icon src={Check} theme="outline" class="h-3.5 w-3.5" />
									{/if}
								</span>
							</button>
						</li>
					{/each}
				</ul>
			{/if}
		</div>

		<DialogFooter class="rounded-none bg-muted/40 px-4">
			<Button
				type="button"
				variant="secondary"
				size="sm"
				onclick={() => (serverDialogOpen = false)}
			>
				Cancel
			</Button>
			<Button type="button" size="sm" onclick={confirmServerPick} disabled={!pickedServer}>
				Continue
			</Button>
		</DialogFooter>
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
