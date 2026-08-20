<script lang="ts">
	import { page } from '$app/state';
	import AppTopbar from '$lib/components/AppTopbar.svelte';
	import {
		provideBuilderTopbar,
		type BuilderTopbarState
	} from '$lib/components/builder-topbar-context';

	let { data, children } = $props();

	// Every route under (builder) is a full-bleed canvas with a name in the
	// topbar, so both facts come from one table rather than two lists that have
	// to be kept in step. They were two, and /projects/new/repo was added to
	// neither: it came out framed like a document page, with an empty topbar.
	const builderRoutes = [
		{ suffix: '/projects/new/image', label: 'New project / Docker Image' },
		{ suffix: '/projects/new/repo', label: 'New project / GitHub Repository' },
		{ suffix: '/projects/[id]/settings', label: 'Project settings' },
		{ suffix: '/projects/new', label: 'New project' },
		{ suffix: '/projects/[id]', label: 'Project builder' }
	];

	let routeId = $derived(page.route.id ?? '');
	let builderRoute = $derived(
		builderRoutes.find((route) => routeId.endsWith(route.suffix)) ?? null
	);
	let isCanvas = $derived(builderRoute !== null);
	let defaultLabel = $derived(builderRoute?.label ?? '');

	const topbar: BuilderTopbarState = $state({
		label: '',
		leading: null,
		action: null
	});
	provideBuilderTopbar(topbar);

	let resolvedLabel = $derived(topbar.label || defaultLabel);
</script>

<div class="flex min-h-screen flex-col bg-background">
	<AppTopbar
		userEmail={data.user?.email ?? ''}
		label={resolvedLabel}
		leading={topbar.leading ?? undefined}
		action={topbar.action ?? undefined}
	/>

	{#if isCanvas}
		<main class="mx-gutter mb-3 flex min-h-0 flex-1 overflow-hidden sm:mb-4">
			{@render children()}
		</main>
	{:else}
		<main
			class="mx-gutter mb-4 flex-1 rounded-lg border border-border bg-card px-4 py-8 text-card-foreground sm:px-8 sm:py-10 md:px-16 lg:px-32 lg:py-14 xl:px-60"
		>
			{@render children()}
		</main>
	{/if}
</div>
