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
						'rounded-full border border-border bg-white text-muted-foreground hover:bg-surface-muted hover:text-foreground',
					ghost: 'rounded-full text-muted-foreground hover:bg-surface-muted hover:text-foreground'
				},
				size: {
					sm: 'h-6 w-6',
					md: 'h-8 w-8'
				}
			},
			defaultVariants: { variant: 'default', size: 'sm' }
		}
	);

	type Props = Omit<HTMLButtonAttributes, 'class'> & {
		variant?: VariantProps<typeof iconButtonVariants>['variant'];
		size?: VariantProps<typeof iconButtonVariants>['size'];
		class?: string;
		children: Snippet;
	};

	let { variant, size, class: className, children, ...rest }: Props = $props();
</script>

<button type="button" class={cn(iconButtonVariants({ variant, size }), className)} {...rest}>
	{@render children()}
</button>
