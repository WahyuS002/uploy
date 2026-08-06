<script lang="ts">
	import { goto } from '$app/navigation';
	import type { PageData } from './$types';
	import ServiceWorkspace from '$lib/components/app/ServiceWorkspace.svelte';

	let { data }: { data: PageData } = $props();
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
				onDeleted={goToProject}
				class="h-[640px]"
			/>
		</div>
	</div>
{/if}
