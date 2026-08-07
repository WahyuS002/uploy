<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		description?: string;
		actions?: Snippet;
	};

	let { description, actions }: Props = $props();
</script>

{#if description}
	<div class="mb-6 flex min-h-8 items-center justify-between gap-4">
		<p class="min-w-0 text-sm text-muted-foreground">{description}</p>
		{#if actions}
			{@render actions()}
		{/if}
	</div>
{:else if actions}
	{@render actions()}
{/if}
