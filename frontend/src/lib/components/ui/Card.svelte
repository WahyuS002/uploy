<script lang="ts" module>
	import { cva, type VariantProps } from 'class-variance-authority';

	export const cardVariants = cva('', {
		variants: {
			variant: {
				default: 'rounded-xl border border-border bg-surface',
				interactive:
					'rounded-xl border border-border bg-surface transition-colors hover:border-border-input hover:bg-surface-muted/60',
				panel: 'rounded-lg border border-border bg-surface',
				inset: 'rounded-lg bg-surface-muted'
			}
		},
		defaultVariants: { variant: 'default' }
	});

	export type CardVariant = VariantProps<typeof cardVariants>['variant'];
</script>

<script lang="ts">
	import { cn } from './cn.js';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	type Props = Omit<HTMLAttributes<HTMLDivElement>, 'class'> & {
		variant?: CardVariant;
		class?: string;
		children: Snippet;
	};

	let { variant, class: className, children, ...rest }: Props = $props();
</script>

<div class={cn(cardVariants({ variant }), className)} {...rest}>
	{@render children()}
</div>
