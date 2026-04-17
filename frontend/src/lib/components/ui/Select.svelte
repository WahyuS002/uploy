<script lang="ts">
	import { Select } from 'bits-ui';
	import { cn } from './cn.js';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { ChevronDown, Check } from '@steeze-ui/heroicons';

	type SelectItem = { value: string; label: string; disabled?: boolean };

	type Props = {
		items: SelectItem[];
		value?: string;
		onValueChange?: (value: string) => void;
		placeholder?: string;
		disabled?: boolean;
		required?: boolean;
		name?: string;
		class?: string;
	};

	let {
		items,
		value = $bindable(''),
		onValueChange,
		placeholder = 'Select...',
		disabled = false,
		required = false,
		name,
		class: className
	}: Props = $props();

	let selectedLabel = $derived(items.find((i) => i.value === value)?.label ?? '');
</script>

<Select.Root type="single" bind:value {onValueChange} {disabled} {required} {name} {items}>
	<Select.Trigger
		class={cn(
			'inline-flex h-10 w-full cursor-pointer items-center justify-between rounded-md border field-focus-glow border-border-input bg-surface px-3 py-2 text-sm text-foreground hover:bg-surface-muted disabled:opacity-50',
			className
		)}
	>
		<span class={cn(!selectedLabel && 'text-muted-foreground')}>
			{selectedLabel || placeholder}
		</span>
		<Icon src={ChevronDown} theme="outline" class="h-3.5 w-3.5 text-muted-foreground" />
	</Select.Trigger>

	<Select.Portal>
		<Select.Content
			class="z-50 max-h-60 w-[var(--bits-select-anchor-width)] min-w-[var(--bits-select-anchor-width)] overflow-auto rounded-lg border border-border bg-surface shadow-md"
			sideOffset={4}
		>
			<Select.Viewport class="p-1">
				{#each items as item (item.value)}
					<Select.Item
						value={item.value}
						label={item.label}
						disabled={item.disabled}
						class="flex w-full animate-slide-up-fade cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm text-foreground outline-none select-none data-disabled:pointer-events-none data-disabled:opacity-50 data-highlighted:bg-surface-muted"
					>
						{#snippet children({ selected })}
							<span class="flex-1">{item.label}</span>
							<span class="inline-flex h-4 w-4 items-center justify-center">
								{#if selected}
									<Icon src={Check} theme="outline" class="h-3 w-3" />
								{/if}
							</span>
						{/snippet}
					</Select.Item>
				{/each}
			</Select.Viewport>
		</Select.Content>
	</Select.Portal>
</Select.Root>
