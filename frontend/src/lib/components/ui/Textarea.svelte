<script lang="ts">
	import { cn } from './cn.js';
	import type { HTMLTextareaAttributes } from 'svelte/elements';

	type Props = Omit<HTMLTextareaAttributes, 'class'> & { class?: string; value?: string };

	let { class: className, value = $bindable(), ...rest }: Props = $props();
</script>

<textarea
	bind:value
	class={cn(
		'rounded-lg border border-border bg-surface px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-ring focus:ring-1 focus:ring-ring focus:outline-none disabled:opacity-50',
		className
	)}
	{...rest}
></textarea>
