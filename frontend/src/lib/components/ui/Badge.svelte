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
				{ variant: 'soft', tone: 'neutral', class: 'bg-muted text-muted-foreground' },
				{ variant: 'soft', tone: 'info', class: 'bg-info-muted text-info' },
				{ variant: 'soft', tone: 'success', class: 'bg-success-muted text-success' },
				{ variant: 'soft', tone: 'warning', class: 'bg-warning-muted text-warning' },
				{ variant: 'soft', tone: 'danger', class: 'bg-destructive/10 text-destructive' },
				{ variant: 'outline', tone: 'neutral', class: 'border-border text-muted-foreground' },
				{ variant: 'outline', tone: 'info', class: 'border-info/25 text-info' },
				{ variant: 'outline', tone: 'success', class: 'border-success/25 text-success' },
				{ variant: 'outline', tone: 'warning', class: 'border-warning/25 text-warning' },
				{ variant: 'outline', tone: 'danger', class: 'border-destructive/25 text-destructive' }
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
