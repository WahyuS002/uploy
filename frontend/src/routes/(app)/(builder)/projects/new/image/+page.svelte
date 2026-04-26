<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import FormField from '$lib/components/app/FormField.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import Spinner from '$lib/components/ui/Spinner.svelte';
	import { createCanvasPan } from '$lib/actions/canvas-pan.svelte';
	import { Icon } from '@steeze-ui/svelte-icon';
	import {
		Server,
		ServerStack,
		Minus,
		Plus,
		ArrowsPointingIn,
		ArrowLeft
	} from '@steeze-ui/heroicons';
	import { Container } from 'lucide-svelte';

	type ServerResponse = components['schemas']['ServerResponse'];

	let { data }: { data: PageData } = $props();
	let canEdit = $derived(data.workspace?.role === 'owner' || data.workspace?.role === 'developer');

	let serverId = $derived(page.url.searchParams.get('server_id') ?? '');

	let servers = $state<ServerResponse[]>([]);
	let serversLoaded = $state(false);
	let serversError = $state('');
	let serversLoadToken = 0;
	let server = $derived<ServerResponse | null>(
		serversLoaded ? (servers.find((s) => s.id === serverId) ?? null) : null
	);

	let image = $state('nginx:latest');
	let port = $state(8080);
	let submitting = $state(false);
	let error = $state('');

	const examples = [
		'nginx:latest',
		'redis:7-alpine',
		'postgres:16',
		'caddy:2',
		'ghcr.io/owner/repo:tag'
	];

	async function loadServers() {
		const token = ++serversLoadToken;
		serversError = '';
		serversLoaded = false;
		try {
			const { data, error: err } = await api.GET('/api/servers');
			if (token !== serversLoadToken) return;
			if (err) {
				serversError = (err as { error: string }).error ?? 'Failed to load servers';
				return;
			}
			servers = data ?? [];
			serversLoaded = true;
		} catch {
			if (token !== serversLoadToken) return;
			serversError = 'Network error';
		}
	}

	$effect(() => {
		loadServers();
		return () => {
			serversLoadToken++;
		};
	});

	function pickExample(value: string) {
		image = value;
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (submitting || !server) return;

		const trimmedImage = image.trim();
		if (!trimmedImage) {
			error = 'Image is required';
			return;
		}
		if (!Number.isFinite(port) || port < 1 || port > 65535) {
			error = 'Port must be between 1 and 65535';
			return;
		}

		error = '';
		submitting = true;
		try {
			const { data: result, error: err } = await api.POST('/api/projects/from-image', {
				body: {
					server_id: server.id,
					image: trimmedImage,
					port
				}
			});
			if (err || !result) {
				error = (err as { error: string } | undefined)?.error ?? 'Failed to create project';
				return;
			}
			// eslint-disable-next-line svelte/no-navigation-without-resolve
			await goto(`/projects/${result.project.id}`, {
				state: {
					toastFlash: {
						tone: 'success',
						title: 'Project created successfully',
						description: `Deploying ${result.service.name} to ${server.name}.`
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
				{:else if !serverId}
					<div class="w-full max-w-105" data-no-pan>
						<EmptyState
							icon={ServerStack}
							title="Pick a server first"
							description="Start by choosing the server this Docker image should deploy to."
						>
							{#snippet actions()}
								<Button href="/projects/new" variant="secondary" size="sm">
									<Icon src={ArrowLeft} theme="outline" class="h-3.5 w-3.5" />
									Back to starters
								</Button>
							{/snippet}
						</EmptyState>
					</div>
				{:else if serversError}
					<div class="w-full max-w-105" data-no-pan>
						<EmptyState icon={ServerStack} title="Couldn't load servers" description={serversError}>
							{#snippet actions()}
								<Button type="button" size="sm" onclick={loadServers}>Retry</Button>
								<Button href="/projects/new" variant="secondary" size="sm">
									<Icon src={ArrowLeft} theme="outline" class="h-3.5 w-3.5" />
									Back to starters
								</Button>
							{/snippet}
						</EmptyState>
					</div>
				{:else if !serversLoaded}
					<div data-no-pan class="flex items-center justify-center">
						<Spinner class="h-6 w-6 text-muted-foreground" />
					</div>
				{:else if !server}
					<div class="w-full max-w-105" data-no-pan>
						<EmptyState
							icon={ServerStack}
							title="Server not found"
							description="That server is no longer available in this workspace. Pick a different one to continue."
						>
							{#snippet actions()}
								<Button href="/projects/new" variant="secondary" size="sm">
									<Icon src={ArrowLeft} theme="outline" class="h-3.5 w-3.5" />
									Back to starters
								</Button>
							{/snippet}
						</EmptyState>
					</div>
				{:else}
					<div class="w-full max-w-105" data-no-pan>
						<form
							onsubmit={submit}
							class="card flex flex-col gap-4 rounded-xl border border-gray-200 bg-card p-5 text-card-foreground"
						>
							<header class="flex items-start gap-3">
								<span
									class="grid h-9 w-9 flex-none place-content-center rounded-md bg-muted text-foreground"
								>
									<Container class="h-4 w-4" strokeWidth={1.75} />
								</span>
								<div class="min-w-0">
									<h1 class="text-sm font-semibold text-foreground">Deploy a Docker image</h1>
									<p class="text-xs text-muted-foreground">
										We'll create the project, a production environment, and your service in one
										step.
									</p>
								</div>
							</header>

							<div class="flex items-end gap-2">
								<div class="flex-1">
									<FormField label="Docker image">
										<Input
											type="text"
											bind:value={image}
											placeholder="nginx:latest"
											autocomplete="off"
											required
										/>
									</FormField>
								</div>
								<div class="w-24">
									<FormField label="Port">
										<Input type="number" bind:value={port} min={1} max={65535} required />
									</FormField>
								</div>
							</div>

							<p class="text-xs text-muted-foreground">
								Supports Docker Hub, GHCR, GCR, Quay, and other public registries.
							</p>

							<div class="flex flex-col gap-1.5">
								<span class="text-[11px] font-medium tracking-wide text-muted-foreground uppercase">
									Try one of these
								</span>
								<div class="flex flex-wrap gap-1.5">
									{#each examples as example (example)}
										<button
											type="button"
											onclick={() => pickExample(example)}
											class="cursor-pointer rounded-md border border-border bg-card px-2 py-1 font-mono text-[11px] text-muted-foreground transition-colors hover:border-foreground/30 hover:bg-accent hover:text-foreground"
										>
											{example}
										</button>
									{/each}
								</div>
							</div>

							<div
								class="flex items-center justify-between rounded-md border border-border bg-muted/40 px-3 py-2 text-xs"
							>
								<div class="flex items-center gap-2 text-muted-foreground">
									<Icon src={ServerStack} theme="outline" class="h-3.5 w-3.5" />
									<span
										>Deploys to <span class="font-medium text-foreground">{server.name}</span></span
									>
								</div>
								<span class="font-mono text-[11px] text-muted-foreground">
									{server.host}:{server.port}
								</span>
							</div>

							{#if error}
								<p class="text-sm text-destructive">{error}</p>
							{/if}

							<div class="flex items-center justify-between gap-2">
								<Button href="/projects/new" variant="secondary" size="sm" type="button">
									<Icon src={ArrowLeft} theme="outline" class="h-3.5 w-3.5" />
									Back
								</Button>
								<Button type="submit" size="sm" loading={submitting} disabled={submitting}>
									{submitting ? 'Creating...' : 'Create project'}
								</Button>
							</div>
						</form>
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

	.card {
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
