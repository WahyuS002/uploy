<script lang="ts">
	import { goto } from '$app/navigation';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';
	import Button from '$lib/components/ui/Button.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import { Icon, type IconSource } from '@steeze-ui/svelte-icon';
	import {
		Server,
		CodeBracket,
		CircleStack,
		RectangleStack,
		ChevronRight,
		MagnifyingGlass
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
</script>

<div class="canvas relative flex w-full flex-1 overflow-hidden rounded-xl border border-border">
	<div class="canvas-bg" aria-hidden="true"></div>
	<div class="canvas-glow" aria-hidden="true"></div>

	<div class="relative z-10 flex w-full overflow-y-auto">
		<div
			class="m-auto flex min-h-full w-full items-center justify-center px-4 py-8 sm:px-6 sm:py-12"
		>
			{#if !canEdit}
				<div class="w-full max-w-105">
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
				<div class="flex w-full max-w-105 flex-col gap-3">
					<section
						class="palette overflow-hidden rounded-xl border border-border bg-surface"
						aria-label="Start a new project"
					>
						<div class="p-1.5 pb-1">
							<Input
								bind:value={promptDraft}
								placeholder="What would you like to create?"
								class="rounded-md p-3 text-base sm:text-sm"
							/>
						</div>

						<div class="py-2">
							{#if filteredRows.length === 0}
								<div
									class="mx-2 flex items-center gap-3 rounded-lg px-3 py-4 text-sm text-muted-foreground"
								>
									<Icon src={MagnifyingGlass} theme="outline" class="h-4 w-4 flex-none" />
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
											<li class="mx-2">
												<button
													type="button"
													onclick={() => row.starter && launch(row.starter)}
													disabled={pending}
													class="grid w-full cursor-pointer items-center gap-x-4 rounded-lg p-3 text-left text-sm text-muted-foreground transition-colors hover:bg-surface-muted hover:text-foreground disabled:cursor-not-allowed disabled:hover:bg-transparent disabled:hover:text-muted-foreground {gridCols}"
												>
													<span class="text-lg leading-none">
														{#if row.lucide}
															{@const LucideIcon = row.lucide}
															<LucideIcon class="h-[1em] w-[1em]" strokeWidth={1.5} />
														{:else if row.icon}
															<Icon src={row.icon} theme="outline" class="h-[1em] w-[1em]" />
														{/if}
													</span>
													<span class="truncate">{row.title}</span>
													{#if busy}
														<svg
															class="h-4 w-4 animate-spin"
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
														<Icon src={ChevronRight} theme="outline" class="h-4 w-4" />
													{/if}
												</button>
											</li>
										{:else}
											<li
												class="mx-2 grid cursor-default items-center gap-x-4 rounded-lg p-3 text-sm text-muted-foreground {gridCols}"
											>
												<span class="text-lg leading-none">
													{#if row.lucide}
														{@const LucideIcon = row.lucide}
														<LucideIcon class="h-[1em] w-[1em]" strokeWidth={1.5} />
													{:else if row.icon}
														<Icon src={row.icon} theme="outline" class="h-[1em] w-[1em]" />
													{/if}
												</span>
												<span class="truncate">{row.title}</span>
												<span
													class="rounded-md bg-surface-muted px-1.5 py-0.5 text-[10px] font-medium tracking-wide text-muted-foreground uppercase"
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
						<p class="text-sm text-danger">{error}</p>
					{/if}
				</div>
			{/if}
		</div>
	</div>
</div>

<style>
	.canvas {
		background-color: var(--color-surface);
		box-shadow:
			inset 0 1px 0 rgba(255, 255, 255, 0.6),
			var(--shadow-panel);
	}

	.canvas-bg {
		position: absolute;
		inset: 0;
		background-image: radial-gradient(circle at 1px 1px, rgba(26, 27, 30, 0.08) 1px, transparent 0);
		background-size: 20px 20px;
		background-position: -1px -1px;
		mask-image: radial-gradient(
			ellipse 80% 70% at 50% 45%,
			rgba(0, 0, 0, 0.9) 0%,
			rgba(0, 0, 0, 0.6) 55%,
			transparent 100%
		);
		-webkit-mask-image: radial-gradient(
			ellipse 80% 70% at 50% 45%,
			rgba(0, 0, 0, 0.9) 0%,
			rgba(0, 0, 0, 0.6) 55%,
			transparent 100%
		);
		pointer-events: none;
	}

	.canvas-glow {
		position: absolute;
		inset: 0;
		background:
			radial-gradient(
				circle at 50% 38%,
				rgba(62, 99, 221, 0.08) 0%,
				rgba(62, 99, 221, 0.03) 30%,
				transparent 60%
			),
			linear-gradient(180deg, rgba(247, 247, 245, 0) 0%, rgba(238, 237, 233, 0.4) 100%);
		pointer-events: none;
	}

	.palette {
		box-shadow:
			0 1px 0 rgba(255, 255, 255, 0.8) inset,
			0 20px 48px -24px rgba(17, 17, 17, 0.22),
			0 4px 12px -6px rgba(17, 17, 17, 0.08);
	}
</style>
