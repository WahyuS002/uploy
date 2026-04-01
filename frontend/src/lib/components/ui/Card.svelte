<script lang="ts">
	import { cn } from './cn.js';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	type Props = Omit<HTMLAttributes<HTMLDivElement>, 'class'> & {
		variant?: 'default' | 'interactive';
		class?: string;
		children: Snippet;
	};

	let { variant = 'default', class: className, children, ...rest }: Props = $props();
</script>

<div
	class={cn(
		'rounded-xl border border-border bg-surface',
		variant === 'interactive' && 'transition-shadow hover:shadow-md',
		className
	)}
	{...rest}
>
	{@render children()}
</div>
