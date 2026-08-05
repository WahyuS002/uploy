<script lang="ts">
	import { onMount } from 'svelte';
	import { goto, invalidateAll } from '$app/navigation';
	import { navigating, page } from '$app/state';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
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

	type ServerResponse = components['schemas']['ServerResponse'];

	let { data }: { data: PageData } = $props();
	let canEdit = $derived(data.workspace?.role === 'owner' || data.workspace?.role === 'developer');

	// Only a step in from the starter panel animates. A direct URL load has no
	// previous position to move from, so the motion would be explaining nothing.
	// Captured once at init rather than derived: SvelteKit clears `navigating`
	// as soon as the navigation settles, and a reactive read would strip the
	// class out from under the animation while it is still running.
	const steppedIn = navigating.from?.url.pathname === '/projects/new';

	let serverId = $derived(page.url.searchParams.get('server_id') ?? '');

	let server = $derived<ServerResponse | null>(data.servers.find((s) => s.id === serverId) ?? null);

	// Empty on landing, not prefilled: the field is the first thing you type into,
	// and backspace-on-empty only works as "go back" if it starts empty.
	let image = $state('');
	let submitting = $state(false);
	let error = $state('');

	let started = $derived(image.trim() !== '');

	// The port an image listens on is a property of the image, not a decision the
	// user should have to make: nobody types `postgres:16` meaning 8080. Well-known
	// images answer for themselves; everything else falls back to 8080.
	const wellKnownPorts: Record<string, number> = {
		postgres: 5432,
		mysql: 3306,
		mariadb: 3306,
		redis: 6379,
		mongo: 27017,
		nginx: 80,
		caddy: 80,
		httpd: 80,
		traefik: 80
	};

	// `ghcr.io/owner/repo:tag` -> `repo`, `postgres:16` -> `postgres`. Splitting on
	// `/` before `:` is what keeps a registry port (`localhost:5000/nginx`) from
	// being mistaken for the tag.
	function detectPort(value: string) {
		const name = value.trim().split('@')[0].split('/').pop()?.split(':')[0]?.toLowerCase() ?? '';
		return wellKnownPorts[name] ?? 8080;
	}

	const examples = [
		'nginx:latest',
		'redis:7-alpine',
		'postgres:16',
		'caddy:2',
		'ghcr.io/owner/repo:tag'
	];

	let port = $state(8080);

	// The port trails the image right up until it is edited, then it stops moving
	// — an auto-filled value that overwrites a deliberate one is worse than no
	// auto-fill at all.
	let portTouched = $state(false);
	$effect(() => {
		const detected = detectPort(image);
		if (!portTouched) port = detected;
	});

	function pickExample(example: string) {
		image = example;
		portTouched = false;
	}

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

	// The command-palette idiom: the arrow lives in the field, and the keyboard
	// gets out the same way it came in. Backspace only leaves when there is
	// nothing left to delete, so it never eats a character.
	function onImageKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			event.preventDefault();
			goBack();
			return;
		}
		if (event.key === 'Backspace' && image === '') {
			event.preventDefault();
			goBack();
		}
	}

	// Works because servers now arrive with the page: the form is in the very
	// first render, so the input is already bound when onMount fires. While the
	// list was fetched client-side, mount landed on the spinner branch and this
	// focused nothing.
	let imageInput = $state<HTMLInputElement | null>(null);
	onMount(() => imageInput?.focus());

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
				{:else if data.serversError}
					<div class="w-full max-w-105" data-no-pan>
						<EmptyState
							icon={ServerStack}
							title="Couldn't load servers"
							description={data.serversError}
						>
							{#snippet actions()}
								<Button type="button" size="sm" loading={retrying} onclick={retryLoadServers}>
									Retry
								</Button>
								<Button href="/projects/new" variant="secondary" size="sm">
									<Icon src={ArrowLeft} theme="outline" class="h-3.5 w-3.5" />
									Back to starters
								</Button>
							{/snippet}
						</EmptyState>
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
							class="card overflow-hidden rounded-xl border border-border bg-card text-card-foreground"
						>
							<div class="relative p-2">
								<button
									type="button"
									onclick={goBack}
									aria-label="Back to starters"
									class="absolute top-1/2 left-4 z-10 grid h-5 w-5 -translate-y-1/2 cursor-pointer place-content-center rounded text-muted-foreground transition-colors hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
								>
									<Icon src={ArrowLeft} theme="outline" class="h-3.5 w-3.5" />
								</button>
								<!-- No border/shadow overrides here: Input already carries
								     field-focus-glow, which is this design system's answer to the
								     brand-coloured focus ring — --mint-deep border plus a soft
								     brand halo. Suppressing it was the reason focus read as dead. -->
								<Input
									type="text"
									bind:value={image}
									onkeydown={onImageKeydown}
									bind:ref={imageInput}
									placeholder="nginx:latest"
									aria-label="Docker image"
									autocomplete="off"
									autocapitalize="off"
									spellcheck={false}
									required
									class="pl-9"
								/>
							</div>

							{#if !started}
								<p class="border-t border-border/70 px-4 py-2.5 text-xs text-muted-foreground">
									Any public registry works — Docker Hub, GHCR, GCR, Quay.
								</p>

								<div class="border-t border-border/70 py-1">
									<p class="px-4 py-1.5 text-xs text-muted-foreground">Examples</p>
									{#each examples as example (example)}
										<button
											type="button"
											onclick={() => pickExample(example)}
											class="flex w-full cursor-pointer items-center justify-between gap-3 px-4 py-1.5 text-left transition-colors hover:bg-accent focus-visible:bg-accent focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none focus-visible:ring-inset"
										>
											<span class="truncate font-mono text-xs text-foreground">{example}</span>
											<span class="font-mono text-[11px] text-muted-foreground"
												>:{detectPort(example)}</span
											>
										</button>
									{/each}
								</div>
							{:else}
								<div class="flex flex-col gap-4 border-t border-border/70 p-4">
									<!-- Only the port is a control, so only the port gets the recessed
									     field surface. The previous pass gave both rows the same bordered
									     box and the input stopped reading as editable. The target is a
									     plain line: nothing to click, so nothing that looks clickable. -->
									<div class="flex flex-col gap-2.5">
										<label class="flex items-center justify-between gap-3">
											<span class="text-xs text-muted-foreground">Listens on port</span>
											<Input
												type="number"
												size="sm"
												bind:value={port}
												oninput={() => (portTouched = true)}
												min={1}
												max={65535}
												required
												class="w-24"
											/>
										</label>

										<div class="flex items-baseline justify-between gap-3 text-xs">
											<span class="flex-none text-muted-foreground">Deploys to</span>
											<span class="flex min-w-0 items-baseline gap-2">
												<span class="truncate font-medium text-foreground">{server.name}</span>
												<span class="flex-none font-mono text-[11px] text-muted-foreground">
													{server.host}:{server.port}
												</span>
											</span>
										</div>
									</div>

									{#if error}
										<p class="text-sm text-destructive">{error}</p>
									{/if}

									<Button type="submit" size="sm" loading={submitting} disabled={submitting}>
										{submitting ? 'Creating...' : 'Create project'}
									</Button>
								</div>
							{/if}
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

	/* Forward step from the starter panel, so the form enters from the right.
	   Applied to .stage, not .world — .world owns the pan transform. */
	/* Forward step from the starter panel, so the form enters from the right. One
	   node here, so no stagger — but the same 200ms curve as the nodes it
	   replaces, and as the dialog (Content.svelte). Applied to .stage, not
	   .world: .world owns the pan transform. `backwards` holds the start offset
	   on the first frame instead of flashing the end state. */
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
