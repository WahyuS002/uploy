<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';
	import PageHeader from '$lib/components/app/PageHeader.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import FormField from '$lib/components/app/FormField.svelte';
	import ToggleGroup from '$lib/components/ui/ToggleGroup.svelte';
	import { Select } from 'bits-ui';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { Plus, Squares2x2, ListBullet, Check, Cube, Server } from '@steeze-ui/heroicons';
	import { ListFilter } from 'lucide-svelte';

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

	// Create project
	let showCreateForm = $state(false);
	let newProjectName = $state('');
	let creating = $state(false);
	let createError = $state('');

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

	async function createProject() {
		createError = '';
		creating = true;
		try {
			const { data, error } = await api.POST('/api/projects', {
				body: { name: newProjectName }
			});
			if (error) {
				createError = (error as { error: string }).error;
				return;
			}
			if (data) {
				projects = [data, ...projects];
				newProjectName = '';
				showCreateForm = false;
			}
		} catch {
			createError = 'Network error';
		} finally {
			creating = false;
		}
	}

	$effect(() => {
		load();
	});
</script>

<section>
	<PageHeader title="Projects">
		{#snippet actions()}
			<div class="flex items-center gap-2">
				<Select.Root type="single" bind:value={sortBy} items={sortOptions}>
					<Select.Trigger
						class="inline-flex h-8 cursor-pointer items-center gap-1.5 rounded-md border border-border bg-surface px-2.5 text-sm text-muted-foreground transition-colors hover:bg-surface-muted hover:text-foreground"
					>
						<ListFilter class="h-4 w-4" strokeWidth={1.75} />
					</Select.Trigger>
					<Select.Portal>
						<Select.Content
							class="z-50 min-w-[160px] rounded-lg border border-border bg-surface p-1 shadow-md"
							sideOffset={4}
						>
							<Select.Viewport>
								{#each sortOptions as option (option.value)}
									<Select.Item
										value={option.value}
										label={option.label}
										class="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm text-foreground outline-none select-none data-[highlighted]:bg-surface-muted"
									>
										{#snippet children({ selected })}
											<span class="inline-flex h-4 w-4 items-center justify-center">
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
				<ToggleGroup
					bind:value={viewMode}
					options={[
						{ value: 'grid', icon: Squares2x2, title: 'Grid view' },
						{ value: 'list', icon: ListBullet, title: 'List view' }
					]}
				/>
				{#if canEdit}
					<Button size="sm" onclick={() => (showCreateForm = !showCreateForm)}>
						<Icon src={Plus} theme="outline" class="h-4 w-4" />
						New
					</Button>
				{/if}
			</div>
		{/snippet}
	</PageHeader>

	<!-- Create project form -->
	{#if showCreateForm}
		<Card class="mb-6 bg-surface-muted p-4">
			<form
				onsubmit={(e) => {
					e.preventDefault();
					createProject();
				}}
				class="flex items-end gap-3"
			>
				<FormField label="Project name">
					<Input type="text" bind:value={newProjectName} placeholder="my-project" required />
				</FormField>
				<Button type="submit" size="sm" loading={creating}>
					{creating ? 'Creating...' : 'Create'}
				</Button>
				<Button
					type="button"
					variant="secondary"
					size="sm"
					onclick={() => {
						showCreateForm = false;
						createError = '';
					}}
				>
					Cancel
				</Button>
			</form>
			{#if createError}
				<p class="mt-2 text-sm text-danger">{createError}</p>
			{/if}
		</Card>
	{/if}

	<!-- Content -->
	{#if loading}
		{#if viewMode === 'grid'}
			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
				{#each [0, 1, 2, 3, 4, 5] as i (i)}
					<div class="overflow-hidden rounded-xl border border-border bg-surface">
						<div class="px-4 pt-4 pb-3">
							<div class="h-4 w-32 animate-pulse rounded bg-surface-muted"></div>
						</div>
						<div class="mx-4 mb-3 h-28 animate-pulse rounded-lg bg-surface-muted"></div>
						<div class="flex items-center gap-2 border-t border-border px-4 py-3">
							<div class="h-2 w-2 animate-pulse rounded-full bg-surface-muted"></div>
							<div class="h-3 w-20 animate-pulse rounded bg-surface-muted"></div>
						</div>
					</div>
				{/each}
			</div>
		{:else}
			<div class="flex flex-col gap-2">
				{#each [0, 1, 2, 3, 4] as i (i)}
					<div
						class="flex items-center justify-between rounded-lg border border-border bg-surface px-4 py-3"
					>
						<div class="flex items-center gap-3">
							<div class="h-9 w-9 animate-pulse rounded-lg bg-surface-muted"></div>
							<div class="flex flex-col gap-1.5">
								<div class="h-4 w-32 animate-pulse rounded bg-surface-muted"></div>
								<div class="h-3 w-24 animate-pulse rounded bg-surface-muted"></div>
							</div>
						</div>
						<div class="h-3 w-20 animate-pulse rounded bg-surface-muted"></div>
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
					href="/dashboard/projects/{project.id}"
					class="group overflow-hidden rounded-xl border border-border bg-surface transition-shadow hover:shadow-md"
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
					href="/dashboard/projects/{project.id}"
					class="flex items-center justify-between rounded-lg border border-border bg-surface px-4 py-3 transition-shadow hover:shadow-md"
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
