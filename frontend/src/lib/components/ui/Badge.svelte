<script lang="ts" module>
	import { cva, type VariantProps } from 'class-variance-authority';

	export const badgeVariants = cva(
		'inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium',
		{
			variants: {
				tone: {
					neutral: 'bg-gray-100 text-gray-700',
					info: 'bg-blue-100 text-blue-700',
					success: 'bg-green-100 text-green-700',
					warning: 'bg-yellow-100 text-yellow-700',
					danger: 'bg-red-100 text-red-700'
				}
			},
			defaultVariants: { tone: 'neutral' }
		}
	);

	export type BadgeTone = VariantProps<typeof badgeVariants>['tone'];
</script>

<script lang="ts">
	import { cn } from './cn.js';
	import type { Snippet } from 'svelte';

	type Props = {
		tone?: BadgeTone;
		class?: string;
		children: Snippet;
	};

	let { tone, class: className, children }: Props = $props();
</script>

<span class={cn(badgeVariants({ tone }), className)}>
	{@render children()}
</span>
