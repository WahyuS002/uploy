<script lang="ts">
	import { cn } from './cn.js';
	import type { Snippet } from 'svelte';
	import type { HTMLLabelAttributes } from 'svelte/elements';

	type Props = Omit<HTMLLabelAttributes, 'class'> & {
		class?: string;
		children: Snippet;
	};

	let { class: className, children, ...rest }: Props = $props();
</script>

<label class={cn('flex flex-col gap-1 text-sm font-medium text-foreground', className)} {...rest}>
	{@render children()}
</label>
