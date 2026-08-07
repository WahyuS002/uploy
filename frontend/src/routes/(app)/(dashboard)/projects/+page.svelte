<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';
	import { serviceLogo, serviceInitial } from '$lib/service-logo';
	import { Button, EmptyState } from '$lib/components/ui';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { Plus, Cube } from '@steeze-ui/heroicons';

	type ServiceResponse = components['schemas']['ServiceResponse'];
	type ProjectResponse = components['schemas']['ProjectResponse'];
	type EnvironmentResponse = components['schemas']['EnvironmentResponse'];

	let { data }: { data: PageData } = $props();
	let canEdit = $derived(data.workspace?.role === 'owner' || data.workspace?.role === 'developer');

	let projects = $state<ProjectResponse[]>([]);
	let services = $state<ServiceResponse[]>([]);
	let projectEnvs = $state<Record<string, EnvironmentResponse[]>>({});
	let loading = $state(true);

	let sortedProjects = $derived(
		[...projects].sort(
			(a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
		)
	);

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

<div class="mb-5 flex items-center justify-between gap-4">
	<h1 class="text-xl font-semibold tracking-tight text-foreground">All projects</h1>
	{#if canEdit}
		<Button href="/projects/new" variant="primary" size="sm">
			<Icon src={Plus} theme="outline" class="h-3.5 w-3.5" />
			Add project
		</Button>
	{/if}
</div>

{#if loading}
	<!-- Skeletons in the card's own shape: the grid doesn't reflow when data lands. -->
	<div class="grid gap-4 {GRID_COLS}">
		{#each [...Array(6).keys()] as i (i)}
			<div class="flex flex-col rounded-lg border border-border bg-card">
				<div class="px-4 pt-3.5 pb-2.5">
					<div class="h-4 w-32 animate-pulse rounded bg-muted"></div>
				</div>
				<div class="mx-3 h-40 animate-pulse rounded-md bg-muted"></div>
				<div class="px-4 pt-3 pb-3.5">
					<div class="h-3 w-40 animate-pulse rounded bg-muted"></div>
				</div>
			</div>
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
{:else}
	<div class="grid gap-4 {GRID_COLS}">
		{#each sortedProjects as project (project.id)}
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
				<!-- A miniature of the canvas this link opens, dots and all. The service
				     marks are brand logos where we know the image and monograms where we
				     don't, so an unmapped image still reads as a distinct node. -->
				<div
					class="canvas-preview relative mx-3 flex h-40 items-center justify-center overflow-hidden rounded-md border border-border"
					aria-hidden="true"
				>
					<div class="flex flex-wrap items-center justify-center gap-2 px-3">
						{#each envServices.slice(0, PREVIEW_TILES) as service (service.id)}
							{@const Logo = serviceLogo(service.image)}
							<span class="node-tile" title={service.image}>
								{#if Logo}
									<Logo class="h-5 w-5" />
								{:else}
									{serviceInitial(service.image)}
								{/if}
							</span>
						{/each}
						{#if envServices.length > PREVIEW_TILES}
							<span class="node-tile">+{envServices.length - PREVIEW_TILES}</span>
						{/if}
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
{/if}

<style>
	/* The same surface as the builder canvas, down to the dot pitch — this is a
	   miniature of it, so it reads from the shared tokens rather than its own
	   near-match values. */
	.canvas-preview {
		background-color: var(--canvas);
		background-image: radial-gradient(circle at 1px 1px, var(--canvas-dot) 1px, transparent 0);
		background-size: 12px 12px;
	}

	/* The tiles are the subject of the preview, not a footnote: they carry the same
	   share of it a node does on the real canvas. */
	.node-tile {
		display: inline-flex;
		height: 2.25rem;
		width: 2.25rem;
		align-items: center;
		justify-content: center;
		border-radius: var(--radius-md);
		border: 1px solid var(--border);
		background-color: var(--card);
		box-shadow: 0 1px 0 rgba(17, 17, 17, 0.04);
		font-family: var(--font-mono);
		font-size: 0.8125rem;
		color: var(--muted-foreground);
		transition: transform 150ms cubic-bezier(0.16, 1, 0.3, 1);
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
