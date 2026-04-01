<script lang="ts">
	import { Collapsible } from 'bits-ui';
	import type { Snippet } from 'svelte';

	type Props = {
		open?: boolean;
		onOpenChange?: (open: boolean) => void;
		trigger: Snippet;
		children: Snippet;
		class?: string;
	};

	let {
		open = $bindable(false),
		onOpenChange,
		trigger,
		children,
		class: className
	}: Props = $props();
</script>

<Collapsible.Root bind:open {onOpenChange} class={className}>
	<Collapsible.Trigger>
		{@render trigger()}
	</Collapsible.Trigger>
	<Collapsible.Content>
		{@render children()}
	</Collapsible.Content>
</Collapsible.Root>
