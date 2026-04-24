<script lang="ts">
	import { goto } from '$app/navigation';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';
	import Button from '$lib/components/ui/Button.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import { createCanvasPan } from '$lib/actions/canvas-pan.svelte';
	import { Icon, type IconSource } from '@steeze-ui/svelte-icon';
	import {
		Server,
		CodeBracket,
		CircleStack,
		RectangleStack,
		ChevronRight,
		MagnifyingGlass,
		Minus,
		Plus,
		ArrowsPointingIn
	} from '@steeze-ui/heroicons';
	import { Archive, Container, SquareTerminal } from 'lucide-svelte';

	type ProjectResponse = components['schemas']['ProjectResponse'];

	type Starter = 'empty-project' | 'docker-image';

	let { data }: { data: PageData } = $props();
	let canEdit = $derived(data.workspace?.role === 'owner' || data.workspace?.role === 'developer');

	let busyStarter = $state<Starter | null>(null);
	let error = $state('');
	let promptDraft = $state('');

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

	async function launch(starter: Starter) {
		if (busyStarter) return;
		error = '';
		busyStarter = starter;
		try {
			const project = await createProject();
			if (!project) return;
			const target =
				starter === 'docker-image'
					? `/projects/${project.id}?starter=docker-image`
					: `/projects/${project.id}`;
			// eslint-disable-next-line svelte/no-navigation-without-resolve
			await goto(target);
		} catch {
			error = 'Network error';
		} finally {
			busyStarter = null;
		}
	}

	type StarterRow = {
		id: string;
		title: string;
		icon?: IconSource;
		lucide?: typeof Container;
		interactive: boolean;
		showsChevron: boolean;
		starter?: Starter;
	};

	const rows: StarterRow[] = [
		{
			id: 'github',
			title: 'GitHub Repository',
			icon: CodeBracket,
			interactive: false,
			showsChevron: true
		},
		{
			id: 'database',
			title: 'Database',
			icon: CircleStack,
			interactive: false,
			showsChevron: true
		},
		{
			id: 'template',
			title: 'Template',
			icon: RectangleStack,
			interactive: false,
			showsChevron: true
		},
		{
			id: 'docker-image',
			title: 'Docker Image',
			lucide: Container,
			interactive: true,
			showsChevron: true,
			starter: 'docker-image'
		},
		{
			id: 'bucket',
			title: 'Bucket',
			lucide: Archive,
			interactive: false,
			showsChevron: false
		},
		{
			id: 'empty-project',
			title: 'Empty Project',
			lucide: SquareTerminal,
			interactive: true,
			showsChevron: false,
			starter: 'empty-project'
		}
	];

	let filteredRows = $derived.by(() => {
		const q = promptDraft.trim().toLowerCase();
		if (!q) return rows;
		return rows.filter((row) => row.title.toLowerCase().includes(q));
	});

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
		style="background-size: {20 * pan.scale}px {20 * pan.scale}px; background-position: {pan.x -
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
						<section
							class="panel overflow-hidden rounded-lg border border-border bg-card text-card-foreground"
							aria-label="Start a new project"
						>
							<div class="border-b border-border/70 px-2 py-1.5">
								<Input
									bind:value={promptDraft}
									placeholder="What would you like to create?"
									class="border-transparent! bg-transparent! px-2 py-1.5 text-sm focus:shadow-none!"
								/>
							</div>

							<div class="py-1">
								{#if filteredRows.length === 0}
									<div class="flex items-center gap-2.5 px-4 py-3 text-sm text-muted-foreground">
										<Icon src={MagnifyingGlass} theme="outline" class="h-3.5 w-3.5 flex-none" />
										<span>No starters match "{promptDraft}"</span>
									</div>
								{:else}
									<ul>
										{#each filteredRows as row (row.id)}
											{@const busy = row.starter != null && busyStarter === row.starter}
											{@const pending = busyStarter !== null}
											{@const gridCols =
												row.showsChevron || !row.interactive
													? 'grid-cols-[auto_1fr_auto]'
													: 'grid-cols-[auto_1fr]'}
											{#if row.interactive}
												<li>
													<button
														type="button"
														onclick={() => row.starter && launch(row.starter)}
														disabled={pending}
														class="grid w-full cursor-pointer items-center gap-x-3 px-3 py-2 text-left text-sm text-foreground transition-colors hover:bg-accent hover:text-accent-foreground disabled:cursor-not-allowed disabled:hover:bg-transparent {gridCols}"
													>
														<span class="text-muted-foreground/70">
															{#if row.lucide}
																{@const LucideIcon = row.lucide}
																<LucideIcon class="h-4 w-4" strokeWidth={1.75} />
															{:else if row.icon}
																<Icon src={row.icon} theme="outline" class="h-4 w-4" />
															{/if}
														</span>
														<span class="truncate">{row.title}</span>
														{#if busy}
															<svg
																class="h-3.5 w-3.5 animate-spin text-muted-foreground/70"
																xmlns="http://www.w3.org/2000/svg"
																fill="none"
																viewBox="0 0 24 24"
															>
																<circle
																	class="opacity-25"
																	cx="12"
																	cy="12"
																	r="10"
																	stroke="currentColor"
																	stroke-width="4"
																/>
																<path
																	class="opacity-75"
																	fill="currentColor"
																	d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
																/>
															</svg>
														{:else if row.showsChevron}
															<Icon
																src={ChevronRight}
																theme="outline"
																class="h-3.5 w-3.5 text-muted-foreground/70"
															/>
														{/if}
													</button>
												</li>
											{:else}
												<li
													class="grid cursor-default items-center gap-x-3 px-3 py-2 text-sm text-muted-foreground {gridCols}"
												>
													<span class="text-muted-foreground/70">
														{#if row.lucide}
															{@const LucideIcon = row.lucide}
															<LucideIcon class="h-4 w-4" strokeWidth={1.75} />
														{:else if row.icon}
															<Icon src={row.icon} theme="outline" class="h-4 w-4" />
														{/if}
													</span>
													<span class="truncate">{row.title}</span>
													<span
														class="rounded-sm bg-secondary px-1.5 py-0.5 text-[10px] font-medium tracking-wide text-muted-foreground/70 uppercase"
													>
														Soon
													</span>
												</li>
											{/if}
										{/each}
									</ul>
								{/if}
							</div>
						</section>

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

<style>
	.canvas {
		background-color: var(--muted);
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
		background-image: radial-gradient(circle at 1px 1px, rgba(26, 27, 30, 0.06) 1px, transparent 0);
		background-size: 20px 20px;
		pointer-events: none;
	}

	.world {
		transform-origin: center center;
		will-change: transform;
	}

	.panel {
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
