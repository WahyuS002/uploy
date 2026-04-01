<script lang="ts">
	import { Tooltip } from 'bits-ui';
	import type { Snippet } from 'svelte';

	type Props = {
		text: string;
		children: Snippet;
	};

	let { text, children }: Props = $props();
</script>

<Tooltip.Root>
	<Tooltip.Trigger>
		{@render children()}
	</Tooltip.Trigger>
	<Tooltip.Portal>
		<Tooltip.Content
			class="z-50 rounded-md bg-foreground px-2.5 py-1.5 text-xs text-surface shadow-md"
			sideOffset={4}
		>
			{text}
		</Tooltip.Content>
	</Tooltip.Portal>
</Tooltip.Root>
