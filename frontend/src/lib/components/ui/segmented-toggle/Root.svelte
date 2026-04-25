<script lang="ts">
	import { ToggleGroup } from 'bits-ui';
	import type { Snippet } from 'svelte';
	import { cn } from '../cn.js';
	import { toggleGroupRootVariants } from '../ToggleGroup.svelte';

	type Props = {
		value?: string;
		onValueChange?: (value: string) => void;
		disabled?: boolean;
		class?: string;
		children?: Snippet;
	};

	let {
		value = $bindable(''),
		onValueChange,
		disabled,
		class: className,
		children
	}: Props = $props();
</script>

<ToggleGroup.Root
	type="single"
	bind:value
	{onValueChange}
	{disabled}
	class={cn(toggleGroupRootVariants({ variant: 'segmented' }), className)}
>
	{@render children?.()}
</ToggleGroup.Root>
