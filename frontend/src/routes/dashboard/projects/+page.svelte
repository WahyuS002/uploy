<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';

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
	<!-- Header -->
	<div class="mb-6 flex items-center justify-between">
		<h2 class="text-sm">Projects</h2>
		{#if canEdit}
			<button
				onclick={() => (showCreateForm = !showCreateForm)}
				class="flex cursor-pointer items-center gap-1.5 rounded-lg bg-black px-4 py-2 text-sm font-medium text-white hover:bg-gray-800"
			>
				<svg
					xmlns="http://www.w3.org/2000/svg"
					class="h-4 w-4"
					viewBox="0 0 20 20"
					fill="currentColor"
				>
					<path
						fill-rule="evenodd"
						d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z"
						clip-rule="evenodd"
					/>
				</svg>
				New
			</button>
		{/if}
	</div>

	<!-- Create project form -->
	{#if showCreateForm}
		<div class="mb-6 rounded-lg border bg-gray-50 p-4">
			<form
				onsubmit={(e) => {
					e.preventDefault();
					createProject();
				}}
				class="flex items-end gap-3"
			>
				<label class="flex flex-1 flex-col gap-1 text-sm font-medium">
					Project name
					<input
						type="text"
						bind:value={newProjectName}
						placeholder="my-project"
						required
						class="rounded-lg border px-3 py-2 text-sm"
					/>
				</label>
				<button
					type="submit"
					disabled={creating}
					class="cursor-pointer rounded-lg bg-black px-4 py-2 text-sm font-medium text-white disabled:opacity-50"
				>
					{creating ? 'Creating...' : 'Create'}
				</button>
				<button
					type="button"
					onclick={() => {
						showCreateForm = false;
						createError = '';
					}}
					class="cursor-pointer rounded-lg border px-4 py-2 text-sm hover:bg-gray-100"
				>
					Cancel
				</button>
			</form>
			{#if createError}
				<p class="mt-2 text-sm text-red-600">{createError}</p>
			{/if}
		</div>
	{/if}

	<!-- Toolbar -->
	{#if !loading}
		<div class="mb-4 flex items-center justify-between text-sm text-gray-500">
			<div class="flex items-center gap-2">
				<span>{projects.length} {projects.length === 1 ? 'Project' : 'Projects'}</span>
				<span class="text-gray-300">|</span>
				<label class="flex items-center gap-1">
					Sort By:
					<select
						bind:value={sortBy}
						class="cursor-pointer border-none bg-transparent p-0 text-sm text-gray-500 focus:ring-0"
					>
						<option value="recent">Recent Activity</option>
						<option value="name">Name</option>
					</select>
				</label>
			</div>
			<div class="flex items-center gap-1">
				<button
					onclick={() => (viewMode = 'grid')}
					class="cursor-pointer rounded p-1 {viewMode === 'grid'
						? 'bg-gray-200 text-black'
						: 'text-gray-400 hover:text-gray-600'}"
					title="Grid view"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						class="h-5 w-5"
						viewBox="0 0 20 20"
						fill="currentColor"
					>
						<path
							d="M5 3a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2V5a2 2 0 00-2-2H5zM5 11a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2v-2a2 2 0 00-2-2H5zM11 5a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V5zM11 13a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z"
						/>
					</svg>
				</button>
				<button
					onclick={() => (viewMode = 'list')}
					class="cursor-pointer rounded p-1 {viewMode === 'list'
						? 'bg-gray-200 text-black'
						: 'text-gray-400 hover:text-gray-600'}"
					title="List view"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						class="h-5 w-5"
						viewBox="0 0 20 20"
						fill="currentColor"
					>
						<path
							fill-rule="evenodd"
							d="M3 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z"
							clip-rule="evenodd"
						/>
					</svg>
				</button>
			</div>
		</div>
	{/if}

	<!-- Content -->
	{#if loading}
		<div class="flex items-center gap-2 py-12 text-sm text-gray-400">
			<svg
				class="h-4 w-4 animate-spin"
				xmlns="http://www.w3.org/2000/svg"
				fill="none"
				viewBox="0 0 24 24"
			>
				<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
				></circle>
				<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
				></path>
			</svg>
			Loading projects...
		</div>
	{:else if projects.length === 0}
		<div class="rounded-lg border-2 border-dashed py-16 text-center">
			<p class="mb-2 text-gray-500">No projects yet</p>
			{#if canEdit}
				<button
					onclick={() => (showCreateForm = true)}
					class="cursor-pointer text-sm font-medium text-black underline hover:no-underline"
				>
					Create your first project
				</button>
			{/if}
		</div>
	{:else if viewMode === 'grid'}
		<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each sortedProjects() as project (project.id)}
				{@const svcCount = getProjectServiceCount(project.id)}
				{@const firstEnv = getProjectFirstEnv(project.id)}
				<a
					href="/dashboard/projects/{project.id}"
					class="group overflow-hidden rounded-xl border bg-white transition-shadow hover:shadow-md"
				>
					<div class="px-4 pt-4 pb-3">
						<h3 class="font-semibold text-gray-900 group-hover:text-black">{project.name}</h3>
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
					<div class="flex items-center gap-2 border-t px-4 py-3 text-xs text-gray-500">
						{#if firstEnv}
							<span class="flex items-center gap-1">
								<span
									class="inline-block h-2 w-2 rounded-full {svcCount > 0
										? 'bg-green-500'
										: 'bg-gray-300'}"
								></span>
								{firstEnv.name}
							</span>
							<span class="text-gray-300">&middot;</span>
						{/if}
						<span>{svcCount} {svcCount === 1 ? 'service' : 'services'}</span>
					</div>
				</a>
			{/each}
		</div>
	{:else}
		<div class="flex flex-col gap-2">
			{#each sortedProjects() as project (project.id)}
				{@const svcCount = getProjectServiceCount(project.id)}
				{@const firstEnv = getProjectFirstEnv(project.id)}
				<a
					href="/dashboard/projects/{project.id}"
					class="flex items-center justify-between rounded-lg border bg-white px-4 py-3 transition-shadow hover:shadow-md"
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
							<h3 class="font-semibold text-gray-900">{project.name}</h3>
							<p class="text-xs text-gray-400">
								Updated {new Date(project.updated_at).toLocaleDateString()}
							</p>
						</div>
					</div>
					<div class="flex items-center gap-3 text-xs text-gray-500">
						{#if firstEnv}
							<span class="flex items-center gap-1">
								<span
									class="inline-block h-2 w-2 rounded-full {svcCount > 0
										? 'bg-green-500'
										: 'bg-gray-300'}"
								></span>
								{firstEnv.name}
							</span>
							<span class="text-gray-300">&middot;</span>
						{/if}
						<span>{svcCount} {svcCount === 1 ? 'service' : 'services'}</span>
					</div>
				</a>
			{/each}
		</div>
	{/if}
</section>
