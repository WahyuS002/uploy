<script lang="ts">
	import { Dialog } from 'bits-ui';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { XMark } from '@steeze-ui/heroicons';
	import type { Snippet } from 'svelte';
	import IconButton from './IconButton.svelte';
	import { cn } from './cn.js';

	type Props = {
		open?: boolean;
		title: string;
		description?: string;
		dismissible?: boolean;
		class?: string;
		children: Snippet;
		footer?: Snippet;
	};

	let {
		open = $bindable(false),
		title,
		description,
		dismissible = true,
		class: className,
		children,
		footer
	}: Props = $props();

	const lockedBehavior = (value: 'close' | 'ignore') => (dismissible ? value : 'ignore');
</script>

<Dialog.Root bind:open>
	<Dialog.Portal>
		<Dialog.Overlay
			class="fixed inset-0 z-50 bg-black/40 data-[state=closed]:animate-[dialog-overlay-out_100ms_ease-in] data-[state=open]:animate-[dialog-overlay-in_100ms_ease-out]"
		/>
		<Dialog.Content
			interactOutsideBehavior={lockedBehavior('close')}
			escapeKeydownBehavior={lockedBehavior('close')}
			class={cn(
				'fixed top-[12vh] left-1/2 z-50 w-[calc(100%-2rem)] max-w-md -translate-x-1/2 rounded-2xl bg-white/20 p-1.5 shadow-lg outline-none',
				className
			)}
		>
			<div class="rounded-xl border border-border bg-surface text-foreground">
				<div class="flex items-start justify-between gap-4 px-5 pt-5 pb-3">
					<div class="flex flex-col gap-1">
						<Dialog.Title class="text-base font-semibold text-foreground">{title}</Dialog.Title>
						{#if description}
							<Dialog.Description class="text-sm text-muted-foreground">
								{description}
							</Dialog.Description>
						{/if}
					</div>
					{#if dismissible}
						<Dialog.Close>
							{#snippet child({ props })}
								<IconButton variant="ghost" aria-label="Close dialog" {...props}>
									<Icon src={XMark} theme="outline" class="h-4 w-4" />
								</IconButton>
							{/snippet}
						</Dialog.Close>
					{/if}
				</div>
				<div class="px-5 pb-5">
					{@render children()}
				</div>
				{#if footer}
					<div class="flex items-center justify-end gap-2 border-t border-border px-5 py-3">
						{@render footer()}
					</div>
				{/if}
			</div>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>

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
