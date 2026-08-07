<script lang="ts" module>
	import { cva, type VariantProps } from 'class-variance-authority';

	export const dataRowVariants = cva(
		'flex w-full items-center gap-3 border-b border-border px-3 text-sm text-foreground transition-colors last:border-b-0',
		{
			variants: {
				density: {
					comfortable: 'py-3',
					dense: 'py-2'
				},
				interactive: {
					true: 'cursor-pointer hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
					false: ''
				},
				selected: {
					true: 'bg-accent text-accent-foreground',
					false: ''
				}
			},
			defaultVariants: {
				density: 'comfortable',
				interactive: false,
				selected: false
			}
		}
	);

	export type DataRowDensity = VariantProps<typeof dataRowVariants>['density'];
</script>

<script lang="ts">
	import { cn } from './cn.js';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes, HTMLAnchorAttributes } from 'svelte/elements';

	type BaseProps = {
		density?: DataRowDensity;
		selected?: boolean;
		class?: string;
		children: Snippet;
	};

	type AsDiv = BaseProps & Omit<HTMLAttributes<HTMLDivElement>, 'class'> & { href?: never };
	type AsAnchor = BaseProps & Omit<HTMLAnchorAttributes, 'class'> & { href: string };

	type Props = AsDiv | AsAnchor;

	let { density, selected = false, class: className, children, href, ...rest }: Props = $props();

	let interactive = $derived(Boolean(href));
</script>

{#if href}
	<!-- eslint-disable svelte/no-navigation-without-resolve -->
	<a
		{href}
		class={cn(dataRowVariants({ density, selected, interactive: true }), className)}
		{...rest as HTMLAnchorAttributes}
	>
		{@render children()}
	</a>
	<!-- eslint-enable svelte/no-navigation-without-resolve -->
{:else}
	<div
		class={cn(dataRowVariants({ density, selected, interactive }), className)}
		{...rest as HTMLAttributes<HTMLDivElement>}
	>
		{@render children()}
	</div>
{/if}
