<script lang="ts">
	import { goto, invalidateAll } from '$app/navigation';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';
	import Button from '$lib/components/ui/Button.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import StarterPanel, { type Starter } from '$lib/components/app/StarterPanel.svelte';
	import ServerNode from '$lib/components/app/ServerNode.svelte';
	import ServerConnectWizard from '$lib/components/app/ServerConnectWizard.svelte';
	import { ServerCreateController } from '$lib/components/app/server-create-form.svelte';
	import {
		Dialog,
		DialogContent,
		DialogHeader,
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
		ExclamationCircle
	} from '@steeze-ui/heroicons';

	type ProjectResponse = components['schemas']['ProjectResponse'];

	let { data }: { data: PageData } = $props();
	let canEdit = $derived(data.workspace?.role === 'owner' || data.workspace?.role === 'developer');
	let isOwner = $derived(data.workspace?.role === 'owner');

	let busyStarter = $state<Starter | null>(null);
	let error = $state('');

	let serverDialogOpen = $state(false);
	let refreshingServers = $state(false);

	// Stable sort: ready first, then everything else. Within each bucket the
	// API's existing created_at DESC ordering is preserved.
	let sortedServers = $derived(
		data.servers.slice().sort((a, b) => {
			const aReady = a.proxy_status === 'ready' ? 0 : 1;
			const bReady = b.proxy_status === 'ready' ? 0 : 1;
			return aReady - bReady;
		})
	);

	// Holds an explicit pick only. The active target falls back to the first ready
	// server, derived rather than assigned in an $effect: an effect would run
	// after the first paint, so the Docker Image row would pop in a frame late.
	let pickedServerId = $state('');
	let activeServerId = $derived(
		(sortedServers.some((s) => s.id === pickedServerId) ? pickedServerId : sortedServers[0]?.id) ??
			''
	);

	const serverController = new ServerCreateController({
		onSuccess: (created) => {
			void (async () => {
				try {
					// Refresh before selecting, so the node never points at a server
					// that is not in its own list yet.
					await invalidateAll();
					pickedServerId = created.id;
				} finally {
					serverDialogOpen = false;
				}
			})();
		}
	});

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

	async function retryLoadServers() {
		refreshingServers = true;
		try {
			await invalidateAll();
		} finally {
			refreshingServers = false;
		}
	}

	function openServerWizard() {
		serverDialogOpen = true;
		void serverController.loadKeys();
	}

	$effect(() => {
		if (!serverDialogOpen) {
			serverController.reset();
		}
	});

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
				if (activeServerId) {
					// eslint-disable-next-line svelte/no-navigation-without-resolve
					await goto(`/projects/new/image?server_id=${encodeURIComponent(activeServerId)}`);
				}
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
	class="canvas viewport relative flex w-full flex-1 overflow-hidden rounded-xl border border-border"
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
					<div class="flex w-full max-w-105 flex-col items-center" data-no-pan>
						<ServerNode
							servers={sortedServers}
							value={activeServerId}
							onValueChange={(id) => (pickedServerId = id)}
							loading={refreshingServers}
							error={data.serversError}
							canConnect={isOwner}
							onConnect={openServerWizard}
							onRetry={retryLoadServers}
						/>

						<span class="edge" aria-hidden="true"></span>

						<StarterPanel
							{busyStarter}
							title="Deploy to it"
							enabled={{ 'docker-image': activeServerId !== '', 'empty-project': true }}
							onSelect={launch}
						/>

						{#if error}
							<p class="mt-2 self-start text-sm text-destructive">{error}</p>
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
		class="inset-y-auto top-[18vh] mt-0 mb-0 w-[min(92vw,32rem)] max-w-none overflow-hidden rounded-2xl"
	>
		<DialogHeader class="border-b border-border px-5 pt-4 pr-12 pb-3">
			<DialogTitle class="text-sm">Connect a server</DialogTitle>
			<DialogDescription class="text-xs">
				Add SSH credentials so Uploy can deploy here.
			</DialogDescription>
		</DialogHeader>

		<ServerConnectWizard
			controller={serverController}
			bodyClass="max-h-[min(65vh,32rem)] overflow-y-auto px-5 pt-4 pb-5"
			actionsClass="rounded-none border-t border-border px-5 py-3"
		/>
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

	/* The connector between the server node and the starter node. Without it the
	   two panels read as stacked modals; with it they read as a graph, which is
	   what the canvas is claiming to be. Drawn at --input, not --border: the
	   hairline disappears against the dotted background. */
	.edge {
		position: relative;
		width: 1px;
		height: 1.25rem;
		background: var(--input);
	}

	.edge::before,
	.edge::after {
		content: '';
		position: absolute;
		left: 50%;
		width: 5px;
		height: 5px;
		border-radius: 9999px;
		background: var(--card);
		border: 1px solid var(--input);
		translate: -50% 0;
	}

	.edge::before {
		top: -3px;
	}

	.edge::after {
		bottom: -3px;
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
