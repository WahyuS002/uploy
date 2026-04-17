<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from './cn.js';

	type Props = {
		class?: string;
		children: Snippet;
	};

	let { class: className, children }: Props = $props();
</script>

<div
	class={cn(
		'flex items-center justify-end gap-2 rounded-b-xl border-t border-border bg-gray-50 px-5 py-3',
		className
	)}
>
	{@render children()}
</div>
