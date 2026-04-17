<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from './cn.js';

	type Props = {
		class?: string;
		children: Snippet;
	};

	let { class: className, children }: Props = $props();
</script>

<div class={cn('flex flex-col gap-1 px-5 py-4 pr-12', className)}>
	{@render children()}
</div>
