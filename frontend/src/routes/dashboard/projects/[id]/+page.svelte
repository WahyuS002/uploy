<script lang="ts">
	import { page } from '$app/stores';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';

	type ProjectResponse = components['schemas']['ProjectResponse'];
	type EnvironmentResponse = components['schemas']['EnvironmentResponse'];

	let { data }: { data: PageData } = $props();
	let canEdit = $derived(data.workspace?.role === 'owner' || data.workspace?.role === 'developer');
	let isOwner = $derived(data.workspace?.role === 'owner');

	let project = $state<ProjectResponse | null>(null);
	let environments = $state<EnvironmentResponse[]>([]);
	let error = $state('');
	let creating = $state(false);
	let envName = $state('');

	const projectId = $page.params.id as string;

	async function loadProject() {
		const { data } = await api.GET('/api/projects/{id}', {
			params: { path: { id: projectId } }
		});
		if (data) project = data;
	}

	async function loadEnvironments() {
		const { data } = await api.GET('/api/projects/{id}/environments', {
			params: { path: { id: projectId } }
		});
		if (data) environments = data;
	}

	async function createEnvironment() {
		error = '';
		creating = true;
		try {
			const { data, error: err } = await api.POST('/api/projects/{id}/environments', {
				params: { path: { id: projectId } },
				body: { name: envName }
			});
			if (err) {
				error = (err as { error: string }).error;
				return;
			}
			if (data) {
				environments = [...environments, data];
				envName = '';
			}
		} catch {
			error = 'Network error';
		} finally {
			creating = false;
		}
	}

	async function deleteEnvironment(envId: string) {
		const { error: err } = await api.DELETE('/api/projects/{id}/environments/{envId}', {
			params: { path: { id: projectId, envId } }
		});
		if (err) {
			error = (err as { error: string }).error || 'Failed to delete environment';
			return;
		}
		environments = environments.filter((e) => e.id !== envId);
	}

	$effect(() => {
		loadProject();
		loadEnvironments();
	});
</script>

{#if project}
	<section>
		<h2 class="mb-4 text-xl font-bold">{project.name}</h2>

		<div class="mb-6">
			<h3 class="mb-2 text-lg font-bold">Environments</h3>

			{#if canEdit}
				<form
					onsubmit={(e) => { e.preventDefault(); createEnvironment(); }}
					class="mb-4 flex items-end gap-2"
				>
					<label class="flex flex-1 flex-col gap-1 text-sm">
						<span>Environment name</span>
						<input
							type="text"
							bind:value={envName}
							placeholder="e.g. staging, production"
							required
							class="rounded border p-1 text-sm"
						/>
					</label>
					<button
						type="submit"
						disabled={creating}
						class="cursor-pointer rounded-sm bg-black px-3 py-1.5 text-sm text-white disabled:opacity-50"
					>
						{creating ? 'Creating...' : 'Add Environment'}
					</button>
				</form>
				{#if error}
					<p class="mb-2 text-sm text-red-600">{error}</p>
				{/if}
			{/if}

			{#if environments.length === 0}
				<p class="text-sm text-gray-500">No environments yet.</p>
			{:else}
				<div class="flex flex-col gap-1">
					{#each environments as env}
						<div class="flex items-center justify-between rounded border p-3">
							<div>
								<span class="font-bold">{env.name}</span>
								<span class="ml-2 text-xs text-gray-400">{env.id}</span>
							</div>
							{#if isOwner}
								<button
									onclick={() => deleteEnvironment(env.id)}
									class="cursor-pointer text-sm text-red-500 hover:text-red-700"
								>
									Delete
								</button>
							{/if}
						</div>
					{/each}
				</div>
			{/if}
		</div>
	</section>
{/if}
