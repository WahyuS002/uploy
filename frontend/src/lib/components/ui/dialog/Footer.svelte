<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';
	import { cn } from '../cn.js';

	type Props = HTMLAttributes<HTMLDivElement> & {
		children: Snippet;
	};

	let { class: className, children, ...rest }: Props = $props();
</script>

<div
	{...rest}
	data-slot="dialog-footer"
	class={cn(
		'flex items-center justify-end gap-2 rounded-b-xl border-t border-border bg-muted px-5 py-3',
		className
	)}
>
	{@render children()}
</div>
