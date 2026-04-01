<script lang="ts">
	import { cn } from './cn.js';
	import type { Snippet } from 'svelte';

	type Props = {
		message: string;
		class?: string;
		actions?: Snippet;
	};

	let { message, class: className, actions }: Props = $props();
</script>

<div class={cn('rounded-lg border-2 border-dashed py-16 text-center', className)}>
	<p class="mb-2 text-muted-foreground">{message}</p>
	{#if actions}
		{@render actions()}
	{/if}
</div>
