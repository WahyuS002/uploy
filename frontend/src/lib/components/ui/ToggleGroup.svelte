<script lang="ts">
	import { ToggleGroup } from 'bits-ui';
	import { cn } from './cn.js';
	import type { Component } from 'svelte';

	type ToggleOption = { value: string; label: string; icon?: Component };

	type Props = {
		value?: string;
		onValueChange?: (value: string) => void;
		options: ToggleOption[];
		class?: string;
	};

	let { value = $bindable(''), onValueChange, options, class: className }: Props = $props();
</script>

<ToggleGroup.Root
	type="single"
	bind:value
	{onValueChange}
	class={cn('inline-flex items-center gap-1', className)}
>
	{#each options as option (option.value)}
		<ToggleGroup.Item
			value={option.value}
			class={cn(
				'inline-flex cursor-pointer items-center gap-1.5 rounded-md px-2.5 py-1 text-sm transition-colors focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none',
				value === option.value
					? 'bg-gray-200 text-foreground'
					: 'text-muted-foreground hover:text-foreground'
			)}
		>
			{#if option.icon}
				{@const Icon = option.icon}
				<Icon size={16} />
			{/if}
			{option.label}
		</ToggleGroup.Item>
	{/each}
</ToggleGroup.Root>
