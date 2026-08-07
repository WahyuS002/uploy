<script lang="ts" module>
	import { cva, type VariantProps } from 'class-variance-authority';

	export const toggleGroupRootVariants = cva('inline-flex items-center gap-0.5 rounded-md', {
		variants: {
			variant: {
				subtle: 'border border-border bg-muted p-0.5',
				segmented: 'h-7 border border-border bg-card p-0.5'
			}
		},
		defaultVariants: { variant: 'subtle' }
	});

	export const toggleGroupItemVariants = cva(
		'inline-flex cursor-pointer items-center justify-center gap-1.5 transition-colors focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none',
		{
			variants: {
				variant: {
					subtle: 'h-7 rounded-[5px] px-2 text-sm',
					segmented:
						'h-6 w-7 rounded-md text-muted-foreground hover:text-foreground data-[state=on]:bg-accent data-[state=on]:text-foreground'
				}
			},
			defaultVariants: { variant: 'subtle' }
		}
	);

	export type ToggleGroupVariant = NonNullable<
		VariantProps<typeof toggleGroupRootVariants>['variant']
	>;
</script>

<script lang="ts">
	import { ToggleGroup } from 'bits-ui';
	import { cn } from './cn.js';
	import { Icon, type IconSource } from '@steeze-ui/svelte-icon';

	type ToggleOption = { value: string; label?: string; icon?: IconSource; title?: string };

	type Props = {
		value?: string;
		onValueChange?: (value: string) => void;
		options: ToggleOption[];
		variant?: ToggleGroupVariant;
		class?: string;
	};

	let {
		value = $bindable(''),
		onValueChange,
		options,
		variant = 'subtle',
		class: className
	}: Props = $props();

	let resolvedRoot = $derived(cn(toggleGroupRootVariants({ variant }), className));
	let activeClass = $derived(
		variant === 'subtle' ? 'bg-card text-card-foreground shadow-panel' : ''
	);
	let inactiveClass = $derived(
		variant === 'subtle' ? 'text-muted-foreground hover:text-foreground' : ''
	);
</script>

<ToggleGroup.Root type="single" bind:value {onValueChange} class={resolvedRoot}>
	{#each options as option (option.value)}
		<ToggleGroup.Item
			value={option.value}
			title={option.title ?? option.label}
			class={cn(
				toggleGroupItemVariants({ variant }),
				variant === 'subtle' && (value === option.value ? activeClass : inactiveClass)
			)}
		>
			{#if option.icon}
				<Icon src={option.icon} theme="outline" class="h-3.5 w-3.5" />
			{/if}
			{#if option.label}
				<span>{option.label}</span>
			{/if}
		</ToggleGroup.Item>
	{/each}
</ToggleGroup.Root>
