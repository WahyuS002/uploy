<script lang="ts">
	import { Dialog as BitsDialog } from 'bits-ui';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { XMark } from '@steeze-ui/heroicons';
	import type { Snippet } from 'svelte';
	import IconButton from './IconButton.svelte';
	import { cn } from './cn.js';

	type Props = {
		dismissible?: boolean;
		class?: string;
		children: Snippet;
	};

	let { dismissible = true, class: className, children }: Props = $props();

	const lockedBehavior = (value: 'close' | 'ignore') => (dismissible ? value : 'ignore');
</script>

<BitsDialog.Portal>
	<BitsDialog.Overlay
		class="fixed inset-0 z-50 bg-foreground/25 data-[state=closed]:animate-[dialog-overlay-out_120ms_ease-in] data-[state=open]:animate-[dialog-overlay-in_120ms_ease-out]"
	/>
	<BitsDialog.Content
		interactOutsideBehavior={lockedBehavior('close')}
		escapeKeydownBehavior={lockedBehavior('close')}
		class={cn(
			'fixed top-1/2 left-1/2 z-50 w-[calc(100%-2rem)] max-w-md -translate-x-1/2 -translate-y-1/2 rounded-xl border border-border bg-surface text-foreground shadow-overlay outline-none',
			className
		)}
	>
		<div class="relative">
			{@render children()}
			{#if dismissible}
				<div class="absolute top-2.5 right-2.5">
					<BitsDialog.Close>
						{#snippet child({ props })}
							<IconButton variant="ghost" aria-label="Close dialog" {...props}>
								<Icon src={XMark} theme="outline" class="h-4 w-4" />
							</IconButton>
						{/snippet}
					</BitsDialog.Close>
				</div>
			{/if}
		</div>
	</BitsDialog.Content>
</BitsDialog.Portal>

<style>
	@keyframes -global-dialog-overlay-in {
		from {
			opacity: 0;
		}
		to {
			opacity: 1;
		}
	}
	@keyframes -global-dialog-overlay-out {
		from {
			opacity: 1;
		}
		to {
			opacity: 0;
		}
	}
</style>
