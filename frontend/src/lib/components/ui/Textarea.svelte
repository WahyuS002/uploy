<script lang="ts">
	import { cn } from './cn.js';
	import type { HTMLTextareaAttributes } from 'svelte/elements';

	type Props = Omit<HTMLTextareaAttributes, 'class'> & { class?: string; value?: string };

	let { class: className, value = $bindable(), ...rest }: Props = $props();
</script>

<textarea
	bind:value
	class={cn(
		'block w-full rounded-md border field-focus-glow border-border-input bg-surface px-3 py-2 text-sm font-normal text-foreground placeholder:text-placeholder disabled:opacity-50',
		className
	)}
	{...rest}
></textarea>
