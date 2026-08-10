<script lang="ts">
	import { Tooltip } from 'bits-ui';
	import type { Snippet } from 'svelte';

	type Props = {
		text: string;
		children: Snippet;
		ariaLabel?: string;
		triggerClass?: string;
	};

	let { text, children, ariaLabel = text, triggerClass }: Props = $props();
</script>

<Tooltip.Provider>
	<Tooltip.Root>
		<Tooltip.Trigger aria-label={ariaLabel} class={triggerClass}>
			{@render children()}
		</Tooltip.Trigger>
		<Tooltip.Portal>
			<Tooltip.Content
				class="z-50 max-w-[min(24rem,calc(100vw-2rem))] rounded-md bg-foreground px-2.5 py-1.5 text-xs text-pretty whitespace-normal text-background shadow-md"
				sideOffset={4}
			>
				{text}
			</Tooltip.Content>
		</Tooltip.Portal>
	</Tooltip.Root>
</Tooltip.Provider>
