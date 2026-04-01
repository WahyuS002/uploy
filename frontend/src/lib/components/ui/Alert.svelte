<script lang="ts">
	import { cn } from './cn.js';
	import { cva, type VariantProps } from 'class-variance-authority';
	import type { Snippet } from 'svelte';

	const alertVariants = cva('rounded-lg border p-3 text-sm', {
		variants: {
			tone: {
				neutral: 'border-border bg-surface-muted text-foreground',
				info: 'border-blue-200 bg-blue-50 text-blue-800',
				success: 'border-green-200 bg-green-50 text-green-800',
				warning: 'border-yellow-300 bg-yellow-50 text-yellow-700',
				danger: 'border-red-200 bg-red-50 text-red-600'
			}
		},
		defaultVariants: { tone: 'neutral' }
	});

	type Props = {
		tone?: VariantProps<typeof alertVariants>['tone'];
		class?: string;
		children: Snippet;
	};

	let { tone, class: className, children }: Props = $props();

	let ariaRole = $derived(tone === 'danger' || tone === 'warning' ? 'alert' : 'status');
</script>

<div role={ariaRole} class={cn(alertVariants({ tone }), className)}>
	{@render children()}
</div>
