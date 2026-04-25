<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { IconSource } from '@steeze-ui/svelte-icon';
	import { topbar } from '$lib/stores/topbar.svelte';

	type Props = {
		title: string;
		icon?: IconSource;
		description?: string;
		actions?: Snippet;
	};

	let { title, icon, description, actions }: Props = $props();

	$effect(() => {
		topbar.set({ title, icon });
		return () => {
			topbar.clear();
		};
	});
</script>

{#if description}
	<div class="mb-6 flex min-h-8 items-center justify-between gap-4">
		<p class="min-w-0 text-sm text-muted-foreground">{description}</p>
		{#if actions}
			{@render actions()}
		{/if}
	</div>
{:else if actions}
	<div class="mb-6">
		{@render actions()}
	</div>
{/if}
