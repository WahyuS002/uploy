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

<BitsDialog.Description class={cn('text-sm text-muted-foreground', className)}>
	{@render children()}
</BitsDialog.Description>
