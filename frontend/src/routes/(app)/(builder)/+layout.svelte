<script lang="ts">
	import { page } from '$app/state';
	import BuilderTopbar from '$lib/components/BuilderTopbar.svelte';
	import {
		provideBuilderTopbar,
		type BuilderTopbarState
	} from '$lib/components/builder-topbar-context';

	let { data, children } = $props();

	let routeId = $derived(page.route.id ?? '');
	let isCanvas = $derived(routeId.endsWith('/projects/new') || routeId.endsWith('/projects/[id]'));

	let defaultLabel = $derived.by(() => {
		if (routeId.endsWith('/projects/new')) return 'New project';
		if (routeId.endsWith('/projects/[id]')) return 'Project builder';
		return '';
	});

	const topbar: BuilderTopbarState = $state({
		label: '',
		leading: null,
		action: null
	});
	provideBuilderTopbar(topbar);

	let resolvedLabel = $derived(topbar.label || defaultLabel);
</script>

<div class="flex min-h-screen flex-col bg-white">
	<BuilderTopbar
		userEmail={data.user?.email ?? ''}
		label={resolvedLabel}
		leading={topbar.leading ?? undefined}
		action={topbar.action ?? undefined}
	/>

	{#if isCanvas}
		<main class="mx-3 mb-3 flex min-h-0 flex-1 overflow-hidden sm:mx-4 sm:mb-4">
			{@render children()}
		</main>
	{:else}
		<main
			class="mx-4 mb-4 flex-1 rounded-lg border border-border bg-card px-4 py-8 text-card-foreground sm:px-8 sm:py-10 md:px-16 lg:px-32 lg:py-14 xl:px-60"
		>
			{@render children()}
		</main>
	{/if}
</div>
