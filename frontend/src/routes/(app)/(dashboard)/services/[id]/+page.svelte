<script lang="ts">
	import { goto } from '$app/navigation';
	import type { PageData } from './$types';
	import ServiceWorkspace from '$lib/components/app/ServiceWorkspace.svelte';
	import DeploymentPanel from '$lib/components/app/DeploymentPanel.svelte';
	import type { components } from '$lib/api/v1';

	type DeploymentResponse = components['schemas']['DeploymentResponse'];

	let { data }: { data: PageData } = $props();

	// No floating inspector to stack on here, so the log panel pins itself to the
	// viewport instead — same surface, same entrance, one plane above the page.
	let openDeployment = $state<DeploymentResponse | null>(null);
	let canEdit = $derived(data.workspace?.role === 'owner' || data.workspace?.role === 'developer');
	let isOwner = $derived(data.workspace?.role === 'owner');

	// This page *is* the service, so deleting it has to leave. The project canvas
	// picks the flash up the same way project creation does.
	function goToProject() {
		if (!data.service) return;
		// eslint-disable-next-line svelte/no-navigation-without-resolve
		void goto(`/projects/${data.service.project_id}`, {
			state: { toastFlash: { tone: 'success', title: 'Service deleted' } }
		});
	}
</script>

<svelte:head>
	<title>{data.service ? `${data.service.name} · Uploy` : 'Service · Uploy'}</title>
</svelte:head>

{#if data.service}
	<div class="mx-auto w-full max-w-4xl px-4 py-6">
		<header class="mb-4">
			<h2 class="text-xl font-semibold text-foreground">{data.service.name}</h2>
			<p class="mt-1 text-sm text-muted-foreground">{data.service.image}</p>
		</header>
		<div class="rounded-xl border border-border bg-card">
			<ServiceWorkspace
				service={data.service}
				{canEdit}
				{isOwner}
				bind:openDeployment
				onDeleted={goToProject}
				class="h-[640px]"
			/>
		</div>
	</div>

	{#if openDeployment}
		<DeploymentPanel
			deployment={openDeployment}
			serviceName={data.service.name}
			onClose={() => (openDeployment = null)}
			class="fixed inset-4 z-40 md:inset-y-4 md:right-4 md:left-auto md:w-[min(46rem,calc(100vw-2rem))]"
		/>
	{/if}
{/if}
