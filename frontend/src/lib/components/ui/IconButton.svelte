<script lang="ts">
	import { cn } from './cn.js';
	import { cva, type VariantProps } from 'class-variance-authority';
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';

	const iconButtonVariants = cva(
		'inline-grid cursor-pointer place-content-center transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50',
		{
			variants: {
				variant: {
					default:
						'rounded-full border border-border bg-card text-muted-foreground hover:bg-accent hover:text-accent-foreground',
					ghost: 'rounded-full text-muted-foreground hover:bg-accent hover:text-accent-foreground'
				},
				size: {
					sm: 'h-6 w-6',
					md: 'h-8 w-8'
				},
				selected: {
					true: '',
					false: ''
				}
			},
			compoundVariants: [
				{
					variant: 'default',
					selected: true,
					class: 'border-input bg-accent text-accent-foreground'
				},
				{ variant: 'ghost', selected: true, class: 'bg-accent text-accent-foreground' }
			],
			defaultVariants: { variant: 'default', size: 'sm', selected: false }
		}
	);

	type Props = Omit<HTMLButtonAttributes, 'class'> & {
		variant?: VariantProps<typeof iconButtonVariants>['variant'];
		size?: VariantProps<typeof iconButtonVariants>['size'];
		selected?: boolean;
		class?: string;
		children: Snippet;
	};

	let { variant, size, selected = false, class: className, children, ...rest }: Props = $props();
</script>

<button
	type="button"
	aria-pressed={selected || undefined}
	class={cn(iconButtonVariants({ variant, size, selected }), className)}
	{...rest}
>
	{@render children()}
</button>
