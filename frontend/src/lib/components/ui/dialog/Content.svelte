<script lang="ts">
	import { Dialog as BitsDialog } from 'bits-ui';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { XMark } from '@steeze-ui/heroicons';
	import IconButton from '../IconButton.svelte';
	import { cn } from '../cn.js';

	type Props = BitsDialog.ContentProps & {
		showCloseButton?: boolean;
		dismissible?: boolean;
	};

	let {
		class: className,
		showCloseButton = true,
		dismissible = true,
		interactOutsideBehavior,
		escapeKeydownBehavior,
		children,
		...rest
	}: Props = $props();

	let interact = $derived<'close' | 'ignore'>(
		dismissible
			? ((interactOutsideBehavior as 'close' | 'ignore' | undefined) ?? 'close')
			: 'ignore'
	);
	let escape = $derived<'close' | 'ignore'>(
		dismissible ? ((escapeKeydownBehavior as 'close' | 'ignore' | undefined) ?? 'close') : 'ignore'
	);
</script>

<BitsDialog.Portal>
	<BitsDialog.Overlay
		data-slot="dialog-overlay"
		class="fixed inset-0 z-50 bg-foreground/25 backdrop-blur-[1px] data-[state=closed]:animate-[dialog-overlay-out_120ms_ease-in] data-[state=open]:animate-[dialog-overlay-in_120ms_ease-out]"
	/>
	<BitsDialog.Content
		{...rest}
		data-slot="dialog-content"
		interactOutsideBehavior={interact}
		escapeKeydownBehavior={escape}
		class={cn(
			'fixed inset-0 z-50 m-auto h-fit w-[calc(100%-2rem)] max-w-md rounded-xl border border-border bg-popover text-popover-foreground shadow-overlay outline-none data-[state=closed]:animate-[dialog-content-out_140ms_ease-in] data-[state=open]:animate-[dialog-content-in_160ms_ease-out]',
			className
		)}
	>
		<div class="relative">
			{#if children}{@render children()}{/if}
			{#if showCloseButton && dismissible}
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
	@keyframes -global-dialog-content-in {
		from {
			opacity: 0;
			scale: 0.98;
			translate: 0 -4px;
		}
		to {
			opacity: 1;
			scale: 1;
			translate: 0 0;
		}
	}
	@keyframes -global-dialog-content-out {
		from {
			opacity: 1;
			scale: 1;
			translate: 0 0;
		}
		to {
			opacity: 0;
			scale: 0.98;
			translate: 0 -4px;
		}
	}
</style>
