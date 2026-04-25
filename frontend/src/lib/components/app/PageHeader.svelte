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

{#if description || actions}
	<div class="mb-6 flex min-h-8 items-center justify-between gap-4">
		<div class="min-w-0">
			{#if description}
				<p class="text-sm text-muted-foreground">{description}</p>
			{/if}
		</div>
		{#if actions}
			{@render actions()}
		{/if}
	</div>
{/if}
