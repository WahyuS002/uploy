<script lang="ts">
	import { page } from '$app/state';
	import BuilderTopbar from '$lib/components/BuilderTopbar.svelte';

	let { data, children } = $props();

	let label = $derived.by(() => {
		const id = page.route.id ?? '';
		if (id.endsWith('/projects/new')) return 'New project';
		if (id.endsWith('/projects/[id]')) return 'Project builder';
		return '';
	});
</script>

<div class="flex min-h-screen flex-col bg-background">
	<BuilderTopbar userEmail={data.user?.email ?? ''} {label} />

	<main class="mx-4 mb-4 flex-1 rounded-lg border border-border bg-surface px-60 py-14">
		{@render children()}
	</main>
</div>
