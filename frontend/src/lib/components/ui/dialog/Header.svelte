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
	data-slot="dialog-header"
	class={cn('flex flex-col gap-1 px-5 py-4 pr-12', className)}
>
	{@render children()}
</div>
