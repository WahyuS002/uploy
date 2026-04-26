<script lang="ts">
	import type { PageData } from './$types';
	import ServiceWorkspace from '$lib/components/app/ServiceWorkspace.svelte';

	let { data }: { data: PageData } = $props();
	let canEdit = $derived(data.workspace?.role === 'owner' || data.workspace?.role === 'developer');
</script>

{#if data.service}
	<div class="mx-auto w-full max-w-4xl px-4 py-6">
		<header class="mb-4">
			<h2 class="text-xl font-semibold text-foreground">{data.service.name}</h2>
			<p class="mt-1 text-sm text-muted-foreground">{data.service.image}</p>
		</header>
		<div class="rounded-xl border border-border bg-card">
			<ServiceWorkspace service={data.service} {canEdit} class="h-[640px]" />
		</div>
	</div>
{/if}
