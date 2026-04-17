<script lang="ts">
	import { Dialog as BitsDialog } from 'bits-ui';
	import type { Snippet } from 'svelte';

	type Props = {
		class?: string;
		children?: Snippet;
		child?: Snippet<[{ props: Record<string, unknown> }]>;
	};

	let { class: className, children, child: asChild }: Props = $props();
</script>

{#if asChild}
	<BitsDialog.Trigger>
		{#snippet child(args)}
			{@render asChild(args)}
		{/snippet}
	</BitsDialog.Trigger>
{:else}
	<BitsDialog.Trigger class={className}>
		{#if children}{@render children()}{/if}
	</BitsDialog.Trigger>
{/if}
