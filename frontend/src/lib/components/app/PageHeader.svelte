<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		title: string;
		description?: string;
		actions?: Snippet;
	};

	let { title, description, actions }: Props = $props();
</script>

<div class="mb-6 flex items-center justify-between">
	<div>
		<h2 class="text-sm text-foreground">{title}</h2>
		{#if description}
			<p class="mt-1 text-sm text-muted-foreground">{description}</p>
		{/if}
	</div>
	{#if actions}
		{@render actions()}
	{/if}
</div>
