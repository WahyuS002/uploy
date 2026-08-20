<script lang="ts">
	import { goto, invalidateAll } from '$app/navigation';
	import { navigating, page } from '$app/state';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';
	import Button from '$lib/components/ui/Button.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import RepoStarterForm from '$lib/components/app/RepoStarterForm.svelte';
	import { createCanvasPan } from '$lib/actions/canvas-pan.svelte';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { ArrowLeft, Minus, Plus, ServerStack, ArrowsPointingIn } from '@steeze-ui/heroicons';

	type ServerResponse = components['schemas']['ServerResponse'];
	type AnalysisResponse = components['schemas']['AnalyzeSourceResponse'];

	let { data }: { data: PageData } = $props();
	let canEdit = $derived(data.workspace?.role === 'owner' || data.workspace?.role === 'developer');
	const steppedIn = navigating.from?.url.pathname === '/projects/new';
	let serverId = $derived(page.url.searchParams.get('server_id') ?? '');
	let server = $derived<ServerResponse | null>(data.servers.find((s) => s.id === serverId) ?? null);
	let analysis = $state<AnalysisResponse | null>(null);
	let analyzing = $state(false);
	let submitting = $state(false);
	let error = $state('');
	let retrying = $state(false);

	async function retryLoadServers() {
		retrying = true;
		try {
			await invalidateAll();
		} finally {
			retrying = false;
		}
	}

	function goBack() {
		// eslint-disable-next-line svelte/no-navigation-without-resolve
		void goto('/projects/new');
	}

	function changeRepository() {
		analysis = null;
		error = '';
	}

	async function analyze(repoUrl: string, branch: string) {
		analyzing = true;
		error = '';
		try {
			const { data: result, error: err } = await api.POST('/api/source/analyze', {
				body: { repo_url: repoUrl, branch }
			});
			if (err || !result) {
				error = (err as { error?: string } | undefined)?.error ?? 'Could not analyze repository';
				return;
			}
			analysis = result;
		} catch {
			error = 'Network error';
		} finally {
			analyzing = false;
		}
	}

	async function create(values: {
		name: string;
		containerName: string;
		containerPort: number;
		branch: string;
	}) {
		if (submitting || !server || !analysis) return;
		submitting = true;
		error = '';
		try {
			const projectResult = await api.POST('/api/projects', { body: {} });
			if (projectResult.error || !projectResult.data) {
				error =
					(projectResult.error as { error?: string } | undefined)?.error ??
					'Could not create project';
				return;
			}
			const project = projectResult.data;
			const environmentsResult = await api.GET('/api/projects/{id}/environments', {
				params: { path: { id: project.id } }
			});
			const environment =
				environmentsResult.data?.find((item) => item.name === 'production') ??
				environmentsResult.data?.[0];
			if (environmentsResult.error || !environment) {
				error = 'Could not prepare the project environment';
				return;
			}
			const serviceResult = await api.POST('/api/services/from-source', {
				body: {
					name: values.name,
					container_name: values.containerName,
					container_port: values.containerPort,
					server_id: server.id,
					environment_id: environment.id,
					repo_url: `https://github.com/${analysis.owner}/${analysis.name}`,
					branch: values.branch
				}
			});
			if (serviceResult.error || !serviceResult.data) {
				error =
					(serviceResult.error as { error?: string } | undefined)?.error ??
					'Could not create service';
				if (serviceResult.response?.status === 422) analysis = null;
				return;
			}
			// eslint-disable-next-line svelte/no-navigation-without-resolve
			await goto(`/projects/${project.id}`, {
				state: {
					toastFlash: {
						tone: 'success',
						title: 'Service created successfully',
						description: `Imported ${analysis.owner}/${analysis.name}.`
					}
				}
			});
		} catch {
			error = 'Network error';
		} finally {
			submitting = false;
		}
	}

	const pan = createCanvasPan({ bounds: 'auto' });
	const panViewport = pan.viewport;
</script>

<svelte:head>
	<title>Import a GitHub repository · Uploy</title>
</svelte:head>

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
			class:stage-enter={steppedIn}
		>
			<div
				class="world flex w-full items-center justify-center"
				style="transform: translate3d({pan.x}px, {pan.y}px, 0) scale({pan.scale});"
			>
				{#if !canEdit}
					<div class="w-full max-w-105" data-no-pan>
						<EmptyState
							icon={ServerStack}
							title="You don't have permission to create projects"
							description="Ask a workspace owner or developer to import a repository, or request a role change."
						>
							{#snippet actions()}<Button href="/projects" variant="secondary" size="sm"
									>Back to projects</Button
								>{/snippet}
						</EmptyState>
					</div>
				{:else if !serverId}
					<div class="w-full max-w-105" data-no-pan>
						<EmptyState
							icon={ServerStack}
							title="Pick a server first"
							description="Start by choosing the server this repository should deploy to."
						>
							{#snippet actions()}<Button href="/projects/new" variant="secondary" size="sm"
									><Icon src={ArrowLeft} theme="outline" class="h-3.5 w-3.5" />Back to starters</Button
								>{/snippet}
						</EmptyState>
					</div>
				{:else if data.serversError}
					<div class="w-full max-w-105" data-no-pan>
						<EmptyState
							icon={ServerStack}
							title="Couldn't load servers"
							description={data.serversError}
						>
							{#snippet actions()}<Button
									type="button"
									size="sm"
									loading={retrying}
									onclick={retryLoadServers}>Retry</Button
								><Button href="/projects/new" variant="secondary" size="sm"
									><Icon src={ArrowLeft} theme="outline" class="h-3.5 w-3.5" />Back to starters</Button
								>{/snippet}
						</EmptyState>
					</div>
				{:else if !server}
					<div class="w-full max-w-105" data-no-pan>
						<EmptyState
							icon={ServerStack}
							title="Server not found"
							description="That server is no longer available in this workspace. Pick a different one to continue."
						>
							{#snippet actions()}<Button href="/projects/new" variant="secondary" size="sm"
									><Icon src={ArrowLeft} theme="outline" class="h-3.5 w-3.5" />Back to starters</Button
								>{/snippet}
						</EmptyState>
					</div>
				{:else}
					<div class="w-full max-w-120" data-no-pan>
						<RepoStarterForm
							serverName={server.name}
							serverHost={`${server.host}:${server.port}`}
							{analysis}
							{submitting}
							{analyzing}
							{error}
							onBack={goBack}
							onChangeRepository={changeRepository}
							onAnalyze={analyze}
							onCreate={create}
						/>
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
			aria-label="Zoom out"><Icon src={Minus} theme="outline" class="h-3.5 w-3.5" /></button
		>
		<button
			type="button"
			class="tool-btn zoom-label"
			onclick={() => pan.resetZoom()}
			aria-label="Reset zoom to 100%">{Math.round(pan.scale * 100)}%</button
		>
		<button
			type="button"
			class="tool-btn"
			onclick={() => pan.zoomIn()}
			disabled={pan.scale >= 1.4}
			aria-label="Zoom in"><Icon src={Plus} theme="outline" class="h-3.5 w-3.5" /></button
		>
		<span class="divider" aria-hidden="true"></span>
		<button
			type="button"
			class="tool-btn"
			onclick={() => pan.recenter()}
			aria-label="Recenter canvas"
			><Icon src={ArrowsPointingIn} theme="outline" class="h-3.5 w-3.5" /></button
		>
	</div>
</div>

<style>
	.canvas {
		background-color: var(--canvas);
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
	.stage-enter {
		animation: stage-in-from-right 200ms cubic-bezier(0.23, 1, 0.32, 1) backwards;
	}
	@keyframes stage-in-from-right {
		from {
			opacity: 0;
			transform: translateX(12px);
		}
		to {
			opacity: 1;
			transform: translateX(0);
		}
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
