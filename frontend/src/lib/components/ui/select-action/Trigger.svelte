<script lang="ts">
	import { Select } from 'bits-ui';
	import type { ComponentProps } from 'svelte';
	import { cn } from '../cn.js';
	import { pillVariants } from '../pillVariants.js';

	type Props = Omit<ComponentProps<typeof Select.Trigger>, 'class'> & {
		class?: string;
	};

	let { class: className, children, ...rest }: Props = $props();
</script>

<Select.Trigger class={cn(pillVariants({ state: 'applied' }), className)} {...rest}>
	{@render children?.()}
</Select.Trigger>
