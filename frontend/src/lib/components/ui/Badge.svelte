<script lang="ts" module>
	import { cva, type VariantProps } from 'class-variance-authority';

	export const badgeVariants = cva(
		'inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium',
		{
			variants: {
				variant: {
					soft: '',
					outline: 'border bg-transparent'
				},
				tone: {
					neutral: '',
					info: '',
					success: '',
					warning: '',
					danger: ''
				}
			},
			compoundVariants: [
				{ variant: 'soft', tone: 'neutral', class: 'bg-surface-muted text-muted-foreground' },
				{ variant: 'soft', tone: 'info', class: 'bg-accent-muted text-accent' },
				{ variant: 'soft', tone: 'success', class: 'bg-success-muted text-success' },
				{ variant: 'soft', tone: 'warning', class: 'bg-warning-muted text-warning' },
				{ variant: 'soft', tone: 'danger', class: 'bg-danger-muted text-danger' },
				{ variant: 'outline', tone: 'neutral', class: 'border-border text-muted-foreground' },
				{ variant: 'outline', tone: 'info', class: 'border-accent/25 text-accent' },
				{ variant: 'outline', tone: 'success', class: 'border-success/25 text-success' },
				{ variant: 'outline', tone: 'warning', class: 'border-warning/25 text-warning' },
				{ variant: 'outline', tone: 'danger', class: 'border-danger/25 text-danger' }
			],
			defaultVariants: {
				variant: 'soft',
				tone: 'neutral'
			}
		}
	);

	export type BadgeTone = VariantProps<typeof badgeVariants>['tone'];
	export type BadgeVariant = VariantProps<typeof badgeVariants>['variant'];
</script>

<script lang="ts">
	import { cn } from './cn.js';
	import type { Snippet } from 'svelte';

	type Props = {
		tone?: BadgeTone;
		variant?: BadgeVariant;
		class?: string;
		children: Snippet;
	};

	let { tone, variant, class: className, children }: Props = $props();
</script>

<span class={cn(badgeVariants({ tone, variant }), className)}>
	{@render children()}
</span>
