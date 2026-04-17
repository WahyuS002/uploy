<script lang="ts">
	import { cn } from './cn.js';
	import { cva, type VariantProps } from 'class-variance-authority';
	import type { Snippet } from 'svelte';

	const alertVariants = cva('rounded-lg border p-3 text-sm', {
		variants: {
			tone: {
				neutral: 'border-border bg-surface-muted text-foreground',
				info: 'border-accent/20 bg-accent-muted text-accent',
				success: 'border-success/20 bg-success-muted text-success',
				warning: 'border-warning/25 bg-warning-muted text-warning',
				danger: 'border-danger/20 bg-danger-muted text-danger'
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
