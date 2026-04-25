<script lang="ts">
	import { Select } from 'bits-ui';
	import type { ComponentProps } from 'svelte';
	import { cn } from '../cn.js';
	import { selectMenuVariants } from '../Select.svelte';

	type Props = Omit<ComponentProps<typeof Select.Content>, 'class'> & {
		class?: string;
	};

	let { class: className, children, sideOffset = 6, ...rest }: Props = $props();
</script>

<Select.Content class={cn(selectMenuVariants(), className)} {sideOffset} {...rest}>
	<Select.Viewport>
		{@render children?.()}
	</Select.Viewport>
</Select.Content>
