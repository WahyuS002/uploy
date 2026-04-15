<script lang="ts">
	import { page } from '$app/stores';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';
	import PageHeader from '$lib/components/app/PageHeader.svelte';
	import FormField from '$lib/components/app/FormField.svelte';
	import ResourceListItem from '$lib/components/app/ResourceListItem.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import { Squares2x2 } from '@steeze-ui/heroicons';

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
		<PageHeader title={project.name} />

		<div class="mb-6">
			<h3 class="mb-2 text-lg font-bold text-foreground">Environments</h3>

			{#if canEdit}
				<form
					onsubmit={(e) => {
						e.preventDefault();
						createEnvironment();
					}}
					class="mb-4 flex items-end gap-2"
				>
					<FormField label="Environment name">
						<Input
							type="text"
							bind:value={envName}
							placeholder="e.g. staging, production"
							required
						/>
					</FormField>
					<Button type="submit" size="sm" loading={creating}>
						{creating ? 'Creating...' : 'Add Environment'}
					</Button>
				</form>
				{#if error}
					<p class="mb-2 text-sm text-danger">{error}</p>
				{/if}
			{/if}

			{#if environments.length === 0}
				<EmptyState
					icon={Squares2x2}
					title="No environments yet"
					description="Add an environment like staging or production to group your services."
				/>
			{:else}
				<div class="flex flex-col gap-1">
					{#each environments as env (env.id)}
						<ResourceListItem class="justify-between">
							<div>
								<span class="font-bold text-foreground">{env.name}</span>
								<span class="ml-2 text-xs text-muted-foreground">{env.id}</span>
							</div>
							{#if isOwner}
								<Button
									variant="ghost"
									size="sm"
									onclick={() => deleteEnvironment(env.id)}
									class="text-danger hover:text-red-700"
								>
									Delete
								</Button>
							{/if}
						</ResourceListItem>
					{/each}
				</div>
			{/if}
		</div>
	</section>
{/if}
