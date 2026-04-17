<script lang="ts">
	import { cn } from './cn.js';
	import type { HTMLInputAttributes } from 'svelte/elements';

	type Props = Omit<HTMLInputAttributes, 'class'> & {
		class?: string;
		value?: HTMLInputAttributes['value'];
	};

	let { class: className, value = $bindable(), ...rest }: Props = $props();
</script>

<input
	bind:value
	class={cn(
		'block w-full rounded-md border field-focus-glow border-border-input bg-surface px-3 py-2 text-sm font-normal text-foreground placeholder:text-placeholder disabled:opacity-50',
		className
	)}
	{...rest}
/>
