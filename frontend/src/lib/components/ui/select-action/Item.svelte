<script lang="ts">
	import { Select } from 'bits-ui';
	import type { ComponentProps, Snippet } from 'svelte';
	import { cn } from '../cn.js';
	import { selectMenuItemVariants } from '../Select.svelte';

	type ChildPayload = { selected: boolean; highlighted: boolean };
	type Props = Omit<ComponentProps<typeof Select.Item>, 'class' | 'children'> & {
		class?: string;
		children?: Snippet<[ChildPayload]>;
	};

	let { class: className, children: itemChildren, ...rest }: Props = $props();
</script>

<Select.Item class={cn(selectMenuItemVariants(), className)} {...rest}>
	{#snippet children(payload)}
		{@render itemChildren?.(payload)}
	{/snippet}
</Select.Item>
