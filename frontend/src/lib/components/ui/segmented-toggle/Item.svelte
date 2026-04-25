<script lang="ts">
	import { ToggleGroup } from 'bits-ui';
	import type { Snippet } from 'svelte';
	import { cn } from '../cn.js';
	import { toggleGroupItemVariants } from '../ToggleGroup.svelte';

	type Props = {
		value: string;
		title?: string;
		disabled?: boolean;
		class?: string;
		children?: Snippet;
	};

	let { value, title, disabled, class: className, children }: Props = $props();
</script>

<ToggleGroup.Item
	{value}
	{title}
	{disabled}
	class={cn(toggleGroupItemVariants({ variant: 'segmented' }), className)}
>
	{@render children?.()}
</ToggleGroup.Item>
