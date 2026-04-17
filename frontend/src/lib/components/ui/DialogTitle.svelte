<script lang="ts">
	import { Dialog as BitsDialog } from 'bits-ui';
	import type { Snippet } from 'svelte';
	import { cn } from './cn.js';

	type Props = {
		class?: string;
		children: Snippet;
	};

	let { class: className, children }: Props = $props();
</script>

<BitsDialog.Title class={cn('text-base font-semibold text-foreground', className)}>
	{@render children()}
</BitsDialog.Title>
