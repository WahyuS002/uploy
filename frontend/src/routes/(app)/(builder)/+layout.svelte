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

	<main class="relative flex-1 overflow-hidden">
		<div
			class="absolute inset-0 bg-[radial-gradient(circle_at_1px_1px,var(--color-border)_1px,transparent_0)] bg-size-[18px_18px] opacity-60"
			aria-hidden="true"
		></div>
		<div
			class="pointer-events-none absolute inset-x-0 bottom-0 h-64 bg-gradient-to-t from-background to-transparent"
			aria-hidden="true"
		></div>
		<div class="relative mx-auto flex w-full max-w-5xl flex-col px-4 py-10 sm:px-6 lg:py-14">
			{@render children()}
		</div>
	</main>
</div>
