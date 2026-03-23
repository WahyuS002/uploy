<script lang="ts">
	import DeploymentLogs from '$lib/components/DeploymentLogs.svelte';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';

	type ApplicationResponse = components['schemas']['ApplicationResponse'];

	let { data }: { data: PageData } = $props();
	let canEdit = $derived(data.workspace?.role === 'owner' || data.workspace?.role === 'developer');

	let apps = $state<ApplicationResponse[]>([]);
	let selectedAppId = $state('');
	let deploymentId = $state<string | null>(null);
	let deploying = $state(false);
	let deployError = $state('');

	async function loadApps() {
		const { data } = await api.GET('/api/applications');
		if (data) {
			apps = data;
			if (apps.length > 0 && !selectedAppId) {
				selectedAppId = apps[0].id;
			}
		}
	}

	async function deploy() {
		deployError = '';
		deploying = true;
		try {
			const { data, error } = await api.POST('/api/deployments', {
				body: { application_id: selectedAppId }
			});
			if (error) {
				deployError = (error as { error: string }).error;
				return;
			}
			if (data) deploymentId = data.deployment_id;
		} catch {
			deployError = 'Network error';
		} finally {
			deploying = false;
		}
	}

	$effect(() => { loadApps(); });
</script>

<section>
	<h2 class="mb-4 text-xl font-bold">Quick Deploy</h2>

	{#if !canEdit}
		<p class="text-sm text-gray-500">You do not have permission to deploy.</p>
	{:else if apps.length === 0}
		<p class="text-sm text-gray-500">
			No applications.
			<a href="/dashboard/applications" class="text-blue-600 underline">Create one</a> first.
		</p>
	{:else}
		<form onsubmit={(e) => { e.preventDefault(); deploy(); }} class="flex max-w-md flex-col gap-2">
			<label class="flex flex-col gap-1 text-sm">
				Application
				<select bind:value={selectedAppId} required class="rounded border p-1">
					{#each apps as app}
						<option value={app.id}>{app.name} ({app.image})</option>
					{/each}
				</select>
			</label>

			{#if deployError}
				<p class="text-sm text-red-600">{deployError}</p>
			{/if}

			<button
				type="submit"
				disabled={deploying}
				class="cursor-pointer rounded-sm bg-black p-2 text-white disabled:opacity-50"
			>
				{deploying ? 'Deploying...' : 'Deploy'}
			</button>
		</form>
	{/if}

	{#if deploymentId}
		<DeploymentLogs {deploymentId} />
	{/if}
</section>
