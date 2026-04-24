<script lang="ts">
	import { ToggleGroup } from 'bits-ui';
	import { cn } from './cn.js';
	import { Icon, type IconSource } from '@steeze-ui/svelte-icon';

	type ToggleOption = { value: string; label?: string; icon?: IconSource; title?: string };

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
	class={cn(
		'inline-flex items-center gap-0.5 rounded-md border border-border bg-muted p-0.5',
		className
	)}
>
	{#each options as option (option.value)}
		<ToggleGroup.Item
			value={option.value}
			title={option.title ?? option.label}
			class={cn(
				'inline-flex h-7 cursor-pointer items-center gap-1.5 rounded-[5px] px-2 text-sm transition-colors focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none',
				value === option.value
					? 'bg-card text-card-foreground shadow-panel'
					: 'text-muted-foreground hover:text-foreground'
			)}
		>
			{#if option.icon}
				<Icon src={option.icon} theme="outline" class="h-4 w-4" />
			{/if}
			{#if option.label}
				<span>{option.label}</span>
			{/if}
		</ToggleGroup.Item>
	{/each}
</ToggleGroup.Root>
