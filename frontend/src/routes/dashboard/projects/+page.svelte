<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';

	type ProjectResponse = components['schemas']['ProjectResponse'];

	let { data }: { data: PageData } = $props();
	let canEdit = $derived(data.workspace?.role === 'owner' || data.workspace?.role === 'developer');

	let projects = $state<ProjectResponse[]>([]);
	let error = $state('');
	let creating = $state(false);
	let name = $state('');

	async function load() {
		const { data } = await api.GET('/api/projects');
		if (data) projects = data;
	}

	async function createProject() {
		error = '';
		creating = true;
		try {
			const { data, error: err } = await api.POST('/api/projects', {
				body: { name }
			});
			if (err) {
				error = (err as { error: string }).error;
				return;
			}
			if (data) {
				projects = [data, ...projects];
				name = '';
			}
		} catch {
			error = 'Network error';
		} finally {
			creating = false;
		}
	}

	$effect(() => { load(); });
</script>

<section>
	<h2 class="mb-4 text-xl font-bold">Projects</h2>

	{#if canEdit}
		<form
			onsubmit={(e) => { e.preventDefault(); createProject(); }}
			class="mb-6 flex max-w-md items-end gap-2"
		>
			<label class="flex flex-1 flex-col gap-1 text-sm">
				Name
				<input type="text" bind:value={name} required class="rounded border p-1" />
			</label>
			<button
				type="submit"
				disabled={creating}
				class="cursor-pointer rounded-sm bg-black px-4 py-1.5 text-sm text-white disabled:opacity-50"
			>
				{creating ? 'Creating...' : 'Create Project'}
			</button>
		</form>
		{#if error}
			<p class="mb-4 text-sm text-red-600">{error}</p>
		{/if}
	{/if}

	{#if projects.length === 0}
		<p class="text-sm text-gray-500">No projects yet.</p>
	{:else}
		<div class="flex flex-col gap-2">
			{#each projects as project}
				<a
					href="/dashboard/projects/{project.id}"
					class="rounded border p-3 hover:bg-gray-50"
				>
					<div class="font-bold">{project.name}</div>
					<div class="text-xs text-gray-400">
						Created {new Date(project.created_at).toLocaleDateString()}
					</div>
				</a>
			{/each}
		</div>
	{/if}
</section>
