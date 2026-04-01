<script lang="ts">
	import DeploymentLogs from '$lib/components/DeploymentLogs.svelte';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';

	type ServiceResponse = components['schemas']['ServiceResponse'];
	type ProjectResponse = components['schemas']['ProjectResponse'];

	let { data }: { data: PageData } = $props();
	let canEdit = $derived(data.workspace?.role === 'owner' || data.workspace?.role === 'developer');

	let services = $state<ServiceResponse[]>([]);
	let projects = $state<ProjectResponse[]>([]);
	let selectedServiceId = $state('');
	let deploymentId = $state<string | null>(null);
	let deploying = $state(false);
	let deployError = $state('');

	async function load() {
		const [svcRes, projRes] = await Promise.all([
			api.GET('/api/services'),
			api.GET('/api/projects')
		]);
		if (svcRes.data) {
			services = svcRes.data;
			if (services.length > 0 && !selectedServiceId) {
				selectedServiceId = services[0].id;
			}
		}
		if (projRes.data) projects = projRes.data;
	}

	async function deploy() {
		deployError = '';
		deploying = true;
		try {
			const { data, error } = await api.POST('/api/deployments', {
				body: { service_id: selectedServiceId }
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

	$effect(() => { load(); });
</script>

<section>
	<h2 class="mb-4 text-xl font-bold">Quick Deploy</h2>

	{#if !canEdit}
		<p class="text-sm text-gray-500">You do not have permission to deploy.</p>
	{:else if projects.length === 0}
		<p class="text-sm text-gray-500">
			No projects yet.
			<a href="/dashboard/projects" class="text-blue-600 underline">Create a project</a> to get started.
		</p>
	{:else if services.length === 0}
		<p class="text-sm text-gray-500">
			No services yet.
			<a href="/dashboard/services" class="text-blue-600 underline">Create a service</a> in one of your projects to deploy.
		</p>
	{:else}
		<form onsubmit={(e) => { e.preventDefault(); deploy(); }} class="flex max-w-md flex-col gap-2">
			<label class="flex flex-col gap-1 text-sm">
				Service
				<select bind:value={selectedServiceId} required class="rounded border p-1">
					{#each services as svc}
						<option value={svc.id}>{svc.name} ({svc.image})</option>
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
