<script lang="ts">
	import { cn } from '$lib/components/ui/cn.js';
	import type { Snippet } from 'svelte';

	type Props = {
		href?: string;
		class?: string;
		children: Snippet;
	};

	let { href, class: className, children }: Props = $props();
</script>

{#if href}
	<!-- eslint-disable svelte/no-navigation-without-resolve -->
	<a
		{href}
		class={cn(
			'flex items-center gap-3 rounded-lg border border-border bg-card p-3 text-card-foreground transition-shadow hover:shadow-md',
			className
		)}
	>
		{@render children()}
	</a>
	<!-- eslint-enable svelte/no-navigation-without-resolve -->
{:else}
	<div
		class={cn(
			'flex items-center gap-3 rounded-lg border border-border bg-card p-3 text-card-foreground',
			className
		)}
	>
		{@render children()}
	</div>
{/if}
