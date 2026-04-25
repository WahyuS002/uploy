<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';
	import PageHeader from '$lib/components/app/PageHeader.svelte';
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
				class="flex w-full flex-wrap items-center justify-between gap-3 border-b border-border py-3"
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
							Add record
						</Button>
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
