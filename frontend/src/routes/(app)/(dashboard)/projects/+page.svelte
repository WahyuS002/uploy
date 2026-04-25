<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';
	import PageHeader from '$lib/components/app/PageHeader.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import { Select, ToggleGroup } from 'bits-ui';
	import { Icon } from '@steeze-ui/svelte-icon';
	import {
		Plus,
		Squares2x2,
		ListBullet,
		Check,
		Cube,
		Server,
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

	function getProjectServiceCount(projectId: string): number {
		return services.filter((s) => s.project_id === projectId).length;
	}

	function getProjectFirstEnv(projectId: string): EnvironmentResponse | undefined {
		return projectEnvs[projectId]?.[0];
	}

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

<section>
	<PageHeader title="Projects" icon={Squares2x2}>
		{#snippet actions()}
			<div
				class="toolbar flex w-full flex-wrap items-center justify-between gap-3 border-b border-border px-4 py-3"
			>
				<div class="flex flex-wrap items-center gap-1.5">
					<Select.Root type="single" bind:value={sortBy} items={sortOptions}>
						<Select.Trigger class="sort-pill" title="Sort projects">
							<Icon src={BarsArrowUp} theme="outline" class="pill-icon" />
							<span class="label-prefix">Sorted by</span>
							<span>{sortLabelValue}</span>
						</Select.Trigger>
						<Select.Portal>
							<Select.Content class="menu" sideOffset={6} align="start">
								<Select.Viewport>
									{#each sortOptions as option (option.value)}
										<Select.Item value={option.value} label={option.label} class="menu-item">
											{#snippet children({ selected })}
												<span class="check-slot">
													{#if selected}
														<Icon src={Check} theme="outline" class="h-3 w-3" />
													{/if}
												</span>
												{option.label}
											{/snippet}
										</Select.Item>
									{/each}
								</Select.Viewport>
							</Select.Content>
						</Select.Portal>
					</Select.Root>
					<span class="divider" aria-hidden="true"></span>
					<button type="button" class="filter-pill" title="Filter (coming soon)" disabled>
						<Icon src={Funnel} theme="outline" class="pill-icon" />
						<span>Filter</span>
					</button>
				</div>
				<div class="flex flex-wrap items-center gap-2">
					<ToggleGroup.Root type="single" bind:value={viewMode} class="view-shell">
						<ToggleGroup.Item value="grid" title="Grid view" class="view-item">
							<Icon src={Squares2x2} theme="outline" class="pill-icon" />
						</ToggleGroup.Item>
						<ToggleGroup.Item value="list" title="List view" class="view-item">
							<Icon src={ListBullet} theme="outline" class="pill-icon" />
						</ToggleGroup.Item>
					</ToggleGroup.Root>
					{#if canEdit}
						<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
						<a href="/projects/new" class="cta-blue">
							<Icon src={Plus} theme="outline" class="pill-icon" />
							<span>Add record</span>
						</a>
					{/if}
				</div>
			</div>
		{/snippet}
	</PageHeader>

	<!-- Content -->
	{#if loading}
		{#if viewMode === 'grid'}
			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
				{#each [0, 1, 2, 3, 4, 5] as i (i)}
					<div class="overflow-hidden rounded-xl border border-border bg-card text-card-foreground">
						<div class="px-4 pt-4 pb-3">
							<div class="h-4 w-32 animate-pulse rounded bg-muted"></div>
						</div>
						<div class="mx-4 mb-3 h-28 animate-pulse rounded-lg bg-muted"></div>
						<div class="flex items-center gap-2 border-t border-border px-4 py-3">
							<div class="h-2 w-2 animate-pulse rounded-full bg-muted"></div>
							<div class="h-3 w-20 animate-pulse rounded bg-muted"></div>
						</div>
					</div>
				{/each}
			</div>
		{:else}
			<div class="flex flex-col gap-2">
				{#each [0, 1, 2, 3, 4] as i (i)}
					<div
						class="flex items-center justify-between rounded-lg border border-border bg-card px-4 py-3 text-card-foreground"
					>
						<div class="flex items-center gap-3">
							<div class="h-9 w-9 animate-pulse rounded-lg bg-muted"></div>
							<div class="flex flex-col gap-1.5">
								<div class="h-4 w-32 animate-pulse rounded bg-muted"></div>
								<div class="h-3 w-24 animate-pulse rounded bg-muted"></div>
							</div>
						</div>
						<div class="h-3 w-20 animate-pulse rounded bg-muted"></div>
					</div>
				{/each}
			</div>
		{/if}
	{:else if projects.length === 0}
		<EmptyState
			icons={[Squares2x2, Cube, Server, Plus]}
			title="No projects yet"
			description="Create your first project to start organizing services, environments, and deployments."
		/>
	{:else if viewMode === 'grid'}
		<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each sortedProjects() as project (project.id)}
				{@const svcCount = getProjectServiceCount(project.id)}
				{@const firstEnv = getProjectFirstEnv(project.id)}
				<!-- eslint-disable svelte/no-navigation-without-resolve -->
				<a
					href="/projects/{project.id}"
					class="group overflow-hidden rounded-xl border border-border bg-card text-card-foreground transition-shadow hover:shadow-md"
				>
					<div class="px-4 pt-4 pb-3">
						<h3 class="font-semibold text-foreground group-hover:text-black">{project.name}</h3>
					</div>
					<div
						class="relative mx-4 mb-3 flex h-28 items-center justify-center rounded-lg bg-gray-900"
						style="background-image: radial-gradient(circle, rgba(255,255,255,0.05) 1px, transparent 1px); background-size: 12px 12px;"
					>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							class="h-8 w-8 text-gray-600"
							viewBox="0 0 20 20"
							fill="currentColor"
						>
							<path
								fill-rule="evenodd"
								d="M2 5a2 2 0 012-2h12a2 2 0 012 2v10a2 2 0 01-2 2H4a2 2 0 01-2-2V5zm3.293 1.293a1 1 0 011.414 0l3 3a1 1 0 010 1.414l-3 3a1 1 0 01-1.414-1.414L7.586 10 5.293 7.707a1 1 0 010-1.414zM11 12a1 1 0 100 2h3a1 1 0 100-2h-3z"
								clip-rule="evenodd"
							/>
						</svg>
					</div>
					<div
						class="flex items-center gap-2 border-t border-border px-4 py-3 text-xs text-muted-foreground"
					>
						{#if firstEnv}
							<span class="flex items-center gap-1">
								<span
									class="inline-block h-2 w-2 rounded-full {svcCount > 0
										? 'bg-success'
										: 'bg-gray-300'}"
								></span>
								{firstEnv.name}
							</span>
							<span class="text-gray-300">&middot;</span>
						{/if}
						<span>{svcCount} {svcCount === 1 ? 'service' : 'services'}</span>
					</div>
				</a>
				<!-- eslint-enable svelte/no-navigation-without-resolve -->
			{/each}
		</div>
	{:else}
		<div class="flex flex-col gap-2">
			{#each sortedProjects() as project (project.id)}
				{@const svcCount = getProjectServiceCount(project.id)}
				{@const firstEnv = getProjectFirstEnv(project.id)}
				<!-- eslint-disable svelte/no-navigation-without-resolve -->
				<a
					href="/projects/{project.id}"
					class="flex items-center justify-between rounded-lg border border-border bg-card px-4 py-3 text-card-foreground transition-shadow hover:shadow-md"
				>
					<div class="flex items-center gap-3">
						<div class="flex h-9 w-9 items-center justify-center rounded-lg bg-gray-900">
							<svg
								xmlns="http://www.w3.org/2000/svg"
								class="h-4 w-4 text-gray-400"
								viewBox="0 0 20 20"
								fill="currentColor"
							>
								<path
									fill-rule="evenodd"
									d="M2 5a2 2 0 012-2h12a2 2 0 012 2v10a2 2 0 01-2 2H4a2 2 0 01-2-2V5zm3.293 1.293a1 1 0 011.414 0l3 3a1 1 0 010 1.414l-3 3a1 1 0 01-1.414-1.414L7.586 10 5.293 7.707a1 1 0 010-1.414zM11 12a1 1 0 100 2h3a1 1 0 100-2h-3z"
									clip-rule="evenodd"
								/>
							</svg>
						</div>
						<div>
							<h3 class="font-semibold text-foreground">{project.name}</h3>
							<p class="text-xs text-muted-foreground">
								Updated {new Date(project.updated_at).toLocaleDateString()}
							</p>
						</div>
					</div>
					<div class="flex items-center gap-3 text-xs text-muted-foreground">
						{#if firstEnv}
							<span class="flex items-center gap-1">
								<span
									class="inline-block h-2 w-2 rounded-full {svcCount > 0
										? 'bg-success'
										: 'bg-gray-300'}"
								></span>
								{firstEnv.name}
							</span>
							<span class="text-gray-300">&middot;</span>
						{/if}
						<span>{svcCount} {svcCount === 1 ? 'service' : 'services'}</span>
					</div>
				</a>
				<!-- eslint-enable svelte/no-navigation-without-resolve -->
			{/each}
		</div>
	{/if}
</section>

<style>
	.toolbar :global(.pill-icon) {
		width: 14px;
		height: 14px;
		flex: none;
	}

	.toolbar :global(.sort-pill) {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		height: 28px;
		padding: 0 10px;
		border: 0;
		border-radius: 8px;
		background: #f6f7f8;
		box-shadow: inset 0 0 0 1px rgba(17, 17, 17, 0.06);
		color: #1a1b1e;
		font-size: 14px;
		line-height: 20px;
		font-weight: 500;
		letter-spacing: -0.01em;
		cursor: pointer;
		transition: background 120ms ease;
	}

	.toolbar :global(.sort-pill:hover) {
		background: #eeeff1;
	}

	.toolbar :global(.sort-pill .label-prefix) {
		color: #72757a;
		font-weight: 500;
	}

	.toolbar :global(.sort-pill:focus-visible) {
		outline: 2px solid var(--ring);
		outline-offset: 2px;
	}

	.toolbar :global(.filter-pill) {
		position: relative;
		display: inline-flex;
		align-items: center;
		gap: 6px;
		height: 28px;
		padding: 0 10px;
		border: 0;
		border-radius: 8px;
		background: transparent;
		color: #72757a;
		font-size: 14px;
		line-height: 20px;
		font-weight: 500;
		letter-spacing: -0.01em;
		cursor: pointer;
		transition: background 120ms ease;
	}

	.toolbar :global(.filter-pill::before) {
		content: '';
		position: absolute;
		inset: 0;
		border-radius: 8px;
		border: 1px dashed #d6d8dc;
		pointer-events: none;
	}

	.toolbar :global(.filter-pill:hover) {
		background: #eeeff1;
	}

	.toolbar :global(.filter-pill:disabled) {
		cursor: not-allowed;
		opacity: 0.75;
	}

	.toolbar :global(.filter-pill:disabled:hover) {
		background: transparent;
	}

	.toolbar :global(.filter-pill:focus-visible) {
		outline: 2px solid var(--ring);
		outline-offset: 2px;
	}

	.toolbar .divider {
		display: inline-block;
		width: 1px;
		height: 18px;
		background: #e3e5e8;
		margin: 0 2px;
	}

	.toolbar :global(.view-shell) {
		display: inline-flex;
		align-items: center;
		gap: 2px;
		height: 28px;
		padding: 2px;
		background: #ffffff;
		border-radius: 8px;
		box-shadow:
			0 1px 2px rgba(17, 17, 17, 0.04),
			0 0 0 1px rgba(17, 17, 17, 0.05);
	}

	.toolbar :global(.view-item) {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 24px;
		border: 0;
		border-radius: 6px;
		background: transparent;
		color: #9ca0a6;
		cursor: pointer;
		transition:
			color 120ms ease,
			background 120ms ease,
			box-shadow 120ms ease;
	}

	.toolbar :global(.view-item:hover) {
		color: #1a1b1e;
	}

	.toolbar :global(.view-item[data-state='on']) {
		color: #1a1b1e;
		background: #f6f7f8;
		box-shadow: inset 0 0 0 1px rgba(17, 17, 17, 0.06);
	}

	.toolbar :global(.view-item:focus-visible) {
		outline: 2px solid var(--ring);
		outline-offset: 2px;
	}

	.toolbar :global(.cta-blue) {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		height: 28px;
		padding: 0 12px 0 10px;
		border-radius: 8px;
		background: #2a7cf2;
		color: #ffffff;
		font-size: 14px;
		line-height: 20px;
		font-weight: 500;
		letter-spacing: -0.01em;
		text-decoration: none;
		cursor: pointer;
		box-shadow:
			0 1px 0 rgba(17, 17, 17, 0.04),
			0 1px 2px rgba(42, 124, 242, 0.35),
			inset 0 1px 0 rgba(255, 255, 255, 0.18);
		transition: background 120ms ease;
	}

	.toolbar :global(.cta-blue:hover) {
		background: #1f6cdc;
	}

	.toolbar :global(.cta-blue:focus-visible) {
		outline: 2px solid var(--ring);
		outline-offset: 2px;
	}

	:global(.menu) {
		z-index: 50;
		min-width: 200px;
		padding: 4px;
		background: #ffffff;
		border-radius: 8px;
		box-shadow:
			0 0 0 1px rgba(17, 17, 17, 0.05),
			0 12px 32px -16px rgba(17, 17, 17, 0.18),
			0 2px 6px rgba(17, 17, 17, 0.04);
	}

	:global(.menu-item) {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 6px 8px;
		border-radius: 6px;
		font-size: 14px;
		line-height: 20px;
		font-weight: 500;
		letter-spacing: -0.01em;
		color: #1a1b1e;
		cursor: pointer;
		outline: none;
		user-select: none;
	}

	:global(.menu-item[data-highlighted]) {
		background: #f2f4f7;
	}

	:global(.menu-item .check-slot) {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 14px;
		height: 14px;
		color: #1a1b1e;
	}
</style>
