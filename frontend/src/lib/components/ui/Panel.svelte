<script lang="ts">
	import { cn } from './cn.js';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	type Props = Omit<HTMLAttributes<HTMLElement>, 'class' | 'title'> & {
		title?: string;
		description?: string;
		class?: string;
		headerClass?: string;
		bodyClass?: string;
		header?: Snippet;
		actions?: Snippet;
		footer?: Snippet;
		children: Snippet;
	};

	let {
		title,
		description,
		class: className,
		headerClass,
		bodyClass,
		header,
		actions,
		footer,
		children,
		...rest
	}: Props = $props();

	let hasHeader = $derived(Boolean(header || title || description || actions));
</script>

<section class={cn('rounded-lg border border-border bg-surface', className)} {...rest}>
	{#if hasHeader}
		<header
			class={cn(
				'flex items-start justify-between gap-3 border-b border-border px-4 py-3',
				headerClass
			)}
		>
			<div class="flex min-w-0 flex-col gap-0.5">
				{#if header}
					{@render header()}
				{:else}
					{#if title}
						<h3 class="text-sm font-semibold text-foreground">{title}</h3>
					{/if}
					{#if description}
						<p class="text-xs text-muted-foreground">{description}</p>
					{/if}
				{/if}
			</div>
			{#if actions}
				<div class="flex shrink-0 items-center gap-2">
					{@render actions()}
				</div>
			{/if}
		</header>
	{/if}
	<div class={cn('px-4 py-3', bodyClass)}>
		{@render children()}
	</div>
	{#if footer}
		<footer class="border-t border-border bg-surface-muted px-4 py-3">
			{@render footer()}
		</footer>
	{/if}
</section>
