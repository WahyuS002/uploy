<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';
	import PageHeader from '$lib/components/app/PageHeader.svelte';
	import { formatDate } from '$lib/format-date';
	import { serviceLogo, serviceInitial } from '$lib/service-logo';
	import {
		Button,
		EmptyState,
		SelectAction,
		SegmentedToggle,
		pillVariants
	} from '$lib/components/ui';
	import { Icon } from '@steeze-ui/svelte-icon';
	import {
		Plus,
		Squares2x2,
		ListBullet,
		Check,
		Cube,
		BarsArrowUp,
		Funnel
	} from '@steeze-ui/heroicons';

	type ServiceResponse = components['schemas']['ServiceResponse'];
	type ProjectResponse = components['schemas']['ProjectResponse'];
	type EnvironmentResponse = components['schemas']['EnvironmentResponse'];

	let { data }: { data: PageData } = $props();
	let canEdit = $derived(data.workspace?.role === 'owner' || data.workspace?.role === 'developer');

	let projects = $state<ProjectResponse[]>([]);
	let services = $state<ServiceResponse[]>([]);
	let projectEnvs = $state<Record<string, EnvironmentResponse[]>>({});
	let loading = $state(true);
	let sortBy = $state<'recent' | 'name'>('recent');
	let viewMode = $state<'grid' | 'list'>('grid');

	const sortOptions = [
		{ value: 'recent', label: 'Recent activity' },
		{ value: 'name', label: 'Name' }
	];

	let sortLabelValue = $derived(sortOptions.find((o) => o.value === sortBy)?.label ?? '');

	let sortedProjects = $derived(() => {
		const sorted = [...projects];
		if (sortBy === 'name') {
			sorted.sort((a, b) => a.name.localeCompare(b.name));
		} else {
			sorted.sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime());
		}
		return sorted;
	});

	/** Same default the builder canvas opens on, so the card previews what you land in. */
	function getProjectEnv(projectId: string): EnvironmentResponse | undefined {
		const envs = projectEnvs[projectId];
		return envs?.find((e) => e.name === 'production') ?? envs?.[0];
	}

	/** Scoped to the environment named in the footer — a count for a different env would lie. */
	function getEnvServices(projectId: string): ServiceResponse[] {
		const env = getProjectEnv(projectId);
		return services.filter(
			(s) => s.project_id === projectId && (!env || s.environment_id === env.id)
		);
	}

	const PREVIEW_TILES = 5;
	// The card is mostly canvas, so the track width sets how much of the project you
	// actually see. auto-fill (not auto-fit) keeps a card the same size whether the
	// workspace holds three projects or thirty — the grid grows, the card doesn't.
	const GRID_COLS = 'grid-cols-[repeat(auto-fill,minmax(320px,1fr))]';

	async function load() {
		loading = true;
		try {
			const [projRes, svcRes] = await Promise.all([
				api.GET('/api/projects'),
				api.GET('/api/services')
			]);
			if (projRes.data) projects = projRes.data;
			if (svcRes.data) services = svcRes.data;

			// Load environments for each project
			if (projRes.data) {
				const envResults = await Promise.all(
					projRes.data.map((p) =>
						api.GET('/api/projects/{id}/environments', {
							params: { path: { id: p.id } }
						})
					)
				);
				const envMap: Record<string, EnvironmentResponse[]> = {};
				projRes.data.forEach((p, i) => {
					if (envResults[i].data) envMap[p.id] = envResults[i].data!;
				});
				projectEnvs = envMap;
			}
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		load();
	});
</script>

<svelte:head>
	<title>Projects · Uploy</title>
</svelte:head>

<!-- The service marks, shared by both views: brand logo where we know the image,
     monogram where we don't, so an unmapped image still reads as a distinct node. -->
{#snippet serviceTiles(list: ServiceResponse[], size: 'sm' | 'lg')}
	{#each list.slice(0, PREVIEW_TILES) as service (service.id)}
		{@const Logo = serviceLogo(service.image)}
		<span class="node-tile" class:node-tile-lg={size === 'lg'} title={service.image}>
			{#if Logo}
				<Logo class={size === 'lg' ? 'h-5 w-5' : 'h-4 w-4'} />
			{:else}
				{serviceInitial(service.image)}
			{/if}
		</span>
	{/each}
	{#if list.length > PREVIEW_TILES}
		<span class="node-tile text-[11px]" class:node-tile-lg={size === 'lg'}>
			+{list.length - PREVIEW_TILES}
		</span>
	{/if}
{/snippet}

<section class="flex flex-1 flex-col">
	<PageHeader>
		{#snippet actions()}
			<div
				class="flex w-full flex-wrap items-center justify-between gap-3 border-b border-border px-4 py-3"
			>
				<div class="flex flex-wrap items-center gap-1.5">
					<SelectAction.Root type="single" bind:value={sortBy} items={sortOptions}>
						<SelectAction.Trigger title="Sort projects">
							<Icon src={BarsArrowUp} theme="outline" class="h-3.5 w-3.5" />
							<span class="text-muted-foreground">Sorted by</span>
							<span>{sortLabelValue}</span>
						</SelectAction.Trigger>
						<SelectAction.Portal>
							<SelectAction.Content align="start">
								{#each sortOptions as option (option.value)}
									<SelectAction.Item value={option.value} label={option.label}>
										{#snippet children({ selected })}
											<span class="inline-flex h-3.5 w-3.5 items-center justify-center">
												{#if selected}
													<Icon src={Check} theme="outline" class="h-3 w-3" />
												{/if}
											</span>
											{option.label}
										{/snippet}
									</SelectAction.Item>
								{/each}
							</SelectAction.Content>
						</SelectAction.Portal>
					</SelectAction.Root>
					<span class="mx-1 inline-block h-4 w-px bg-border" aria-hidden="true"></span>
					<button
						type="button"
						title="Filter (coming soon)"
						disabled
						class={pillVariants({ state: 'placeholder' })}
					>
						<Icon src={Funnel} theme="outline" class="h-3.5 w-3.5" />
						<span>Filter</span>
					</button>
				</div>
				<div class="flex flex-wrap items-center gap-2">
					<SegmentedToggle.Root bind:value={viewMode}>
						<SegmentedToggle.Item value="grid" title="Grid view">
							<Icon src={Squares2x2} theme="outline" class="h-3.5 w-3.5" />
						</SegmentedToggle.Item>
						<SegmentedToggle.Item value="list" title="List view">
							<Icon src={ListBullet} theme="outline" class="h-3.5 w-3.5" />
						</SegmentedToggle.Item>
					</SegmentedToggle.Root>
					{#if canEdit}
						<Button href="/projects/new" variant="primary" size="sm">
							<Icon src={Plus} theme="outline" class="h-3.5 w-3.5" />
							Add project
						</Button>
					{/if}
				</div>
			</div>
		{/snippet}
	</PageHeader>

	<!-- Content -->
	<div class="flex flex-1 flex-col px-4 pt-4">
		{#if loading}
			<!-- Skeletons in the card's own shape: the grid doesn't reflow when data lands. -->
			<div class={viewMode === 'grid' ? 'grid gap-4 ' + GRID_COLS : 'flex flex-col gap-2'}>
				{#each [...Array(viewMode === 'grid' ? 6 : 5).keys()] as i (i)}
					{#if viewMode === 'grid'}
						<div class="flex flex-col rounded-lg border border-border bg-card">
							<div class="px-4 pt-3.5 pb-2.5">
								<div class="h-4 w-32 animate-pulse rounded bg-muted"></div>
							</div>
							<div class="mx-3 h-40 animate-pulse rounded-md bg-muted"></div>
							<div class="px-4 pt-3 pb-3.5">
								<div class="h-3 w-40 animate-pulse rounded bg-muted"></div>
							</div>
						</div>
					{:else}
						<div class="flex items-center gap-3 rounded-lg border border-border bg-card px-4 py-3">
							<div class="h-9 w-9 shrink-0 animate-pulse rounded-md bg-muted"></div>
							<div class="flex flex-col gap-1.5">
								<div class="h-4 w-40 animate-pulse rounded bg-muted"></div>
								<div class="h-3 w-28 animate-pulse rounded bg-muted"></div>
							</div>
						</div>
					{/if}
				{/each}
			</div>
		{:else if projects.length === 0}
			<EmptyState
				variant="canvas"
				icon={Cube}
				title="No projects yet"
				description={canEdit
					? 'Create your first project to get started.'
					: 'Ask a workspace owner or developer to create the first project.'}
			>
				{#snippet actions()}
					{#if canEdit}
						<Button href="/projects/new" variant="primary" size="sm">
							<Icon src={Plus} theme="outline" class="h-3.5 w-3.5" />
							Add project
						</Button>
					{/if}
				{/snippet}
			</EmptyState>
		{:else if viewMode === 'grid'}
			<div class="grid gap-4 {GRID_COLS}">
				{#each sortedProjects() as project (project.id)}
					{@const envServices = getEnvServices(project.id)}
					{@const svcCount = envServices.length}
					{@const env = getProjectEnv(project.id)}
					<!-- eslint-disable svelte/no-navigation-without-resolve -->
					<a
						href="/projects/{project.id}"
						class="group flex flex-col rounded-lg border border-border bg-card text-card-foreground transition-colors duration-150 outline-none hover:border-input focus-visible:border-input focus-visible:ring-3 focus-visible:ring-primary/30"
					>
						<div class="px-4 pt-3.5 pb-2.5">
							<h3 class="truncate text-sm font-medium text-foreground">{project.name}</h3>
						</div>
						<!-- A miniature of the canvas this link opens, dots and all. -->
						<div
							class="canvas-preview relative mx-3 flex h-40 items-center justify-center overflow-hidden rounded-md border border-border"
							aria-hidden="true"
						>
							<div class="flex flex-wrap items-center justify-center gap-2 px-3">
								{@render serviceTiles(envServices, 'lg')}
							</div>
						</div>
						<div
							class="mt-auto flex items-center gap-1.5 px-4 pt-3 pb-3.5 text-xs text-muted-foreground"
						>
							<span
								class="inline-block h-1.5 w-1.5 shrink-0 rounded-full {svcCount > 0
									? 'bg-success'
									: 'bg-input'}"
							></span>
							{#if env}
								<span class="truncate">{env.name}</span>
								<span class="text-input">&middot;</span>
							{/if}
							<span class="shrink-0">
								{svcCount === 0
									? 'No services'
									: `${svcCount} ${svcCount === 1 ? 'service' : 'services'}`}
							</span>
						</div>
					</a>
					<!-- eslint-enable svelte/no-navigation-without-resolve -->
				{/each}
			</div>
		{:else}
			<div class="flex flex-col gap-2">
				{#each sortedProjects() as project (project.id)}
					{@const envServices = getEnvServices(project.id)}
					{@const svcCount = envServices.length}
					{@const env = getProjectEnv(project.id)}
					<!-- eslint-disable svelte/no-navigation-without-resolve -->
					<a
						href="/projects/{project.id}"
						class="group flex items-center justify-between gap-4 rounded-lg border border-border bg-card px-4 py-3 text-card-foreground transition-colors duration-150 outline-none hover:border-input focus-visible:border-input focus-visible:ring-3 focus-visible:ring-primary/30"
					>
						<div class="flex min-w-0 items-center gap-3">
							<div
								class="canvas-preview flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-border"
							>
								<Icon src={Cube} theme="outline" class="h-4 w-4 text-muted-foreground" />
							</div>
							<div class="min-w-0">
								<h3 class="truncate text-sm font-medium text-foreground">{project.name}</h3>
								<p class="text-xs text-muted-foreground">
									Updated {formatDate(project.updated_at)}
								</p>
							</div>
						</div>
						<div class="flex shrink-0 items-center gap-3 text-xs text-muted-foreground">
							<div class="hidden items-center gap-1.5 sm:flex" aria-hidden="true">
								{@render serviceTiles(envServices, 'sm')}
							</div>
							<span class="flex items-center gap-1.5">
								<span
									class="inline-block h-1.5 w-1.5 rounded-full {svcCount > 0
										? 'bg-success'
										: 'bg-input'}"
								></span>
								{#if env}
									<span>{env.name}</span>
									<span class="text-input">&middot;</span>
								{/if}
								<span>
									{svcCount === 0
										? 'No services'
										: `${svcCount} ${svcCount === 1 ? 'service' : 'services'}`}
								</span>
							</span>
						</div>
					</a>
					<!-- eslint-enable svelte/no-navigation-without-resolve -->
				{/each}
			</div>
		{/if}
	</div>
</section>

<style>
	/* The same surface as the builder canvas, down to the dot pitch — this is a
	   miniature of it, so it reads from the shared tokens rather than its own
	   near-match values. */
	.canvas-preview {
		background-color: var(--canvas);
		background-image: radial-gradient(circle at 1px 1px, var(--canvas-dot) 1px, transparent 0);
		background-size: 12px 12px;
	}

	.node-tile {
		display: inline-flex;
		height: 1.75rem;
		width: 1.75rem;
		align-items: center;
		justify-content: center;
		border-radius: var(--radius-md);
		border: 1px solid var(--border);
		background-color: var(--card);
		box-shadow: 0 1px 0 rgba(17, 17, 17, 0.04);
		font-family: var(--font-mono);
		font-size: 0.75rem;
		color: var(--muted-foreground);
		transition: transform 150ms cubic-bezier(0.16, 1, 0.3, 1);
	}

	/* On the card the tiles are the subject, not a footnote: they carry the same
	   share of the preview a node does on the real canvas. The list row keeps the
	   small tile, where they sit beside text and must not outweigh it. */
	.node-tile-lg {
		height: 2.25rem;
		width: 2.25rem;
		font-size: 0.8125rem;
	}

	@media (hover: hover) {
		.group:hover .node-tile {
			transform: translateY(-1px);
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.node-tile {
			transition: none;
		}

		.group:hover .node-tile {
			transform: none;
		}
	}
</style>
