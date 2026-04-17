<script lang="ts" module>
	import { cva, type VariantProps } from 'class-variance-authority';

	export const buttonVariants = cva(
		'inline-flex cursor-pointer items-center justify-center gap-2 rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-50',
		{
			variants: {
				variant: {
					primary: 'bg-primary text-primary-foreground hover:bg-primary/90',
					secondary: 'border border-border bg-surface text-foreground hover:bg-surface-muted',
					subtle: 'bg-surface-muted text-foreground hover:bg-surface-sunken',
					ghost: 'text-foreground hover:bg-surface-muted',
					danger: 'bg-danger text-white hover:bg-danger/90'
				},
				size: {
					xs: 'h-7 gap-1.5 px-2.5 text-xs',
					sm: 'h-8 px-3 text-xs',
					md: 'h-10 px-4 py-2'
				}
			},
			defaultVariants: {
				variant: 'primary',
				size: 'md'
			}
		}
	);

	export type ButtonVariant = VariantProps<typeof buttonVariants>['variant'];
	export type ButtonSize = VariantProps<typeof buttonVariants>['size'];
</script>

<script lang="ts">
	import { cn } from './cn.js';
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes, HTMLAnchorAttributes } from 'svelte/elements';

	type BaseProps = {
		variant?: ButtonVariant;
		size?: ButtonSize;
		loading?: boolean;
		class?: string;
		children: Snippet;
	};

	type AsButton = BaseProps & Omit<HTMLButtonAttributes, 'class'> & { href?: never };
	type AsAnchor = BaseProps & Omit<HTMLAnchorAttributes, 'class'> & { href: string };

	type Props = AsButton | AsAnchor;

	let {
		variant,
		size,
		loading = false,
		class: className,
		children,
		href,
		...rest
	}: Props = $props();
</script>

{#if href}
	<!-- eslint-disable svelte/no-navigation-without-resolve -->
	<a
		{href}
		class={cn(buttonVariants({ variant, size }), className)}
		{...rest as HTMLAnchorAttributes}
	>
		{@render children()}
	</a>
	<!-- eslint-enable svelte/no-navigation-without-resolve -->
{:else}
	<button
		class={cn(buttonVariants({ variant, size }), className)}
		disabled={loading || (rest as HTMLButtonAttributes).disabled}
		{...rest as HTMLButtonAttributes}
	>
		{#if loading}
			<svg
				class="h-4 w-4 animate-spin"
				xmlns="http://www.w3.org/2000/svg"
				fill="none"
				viewBox="0 0 24 24"
			>
				<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
				<path
					class="opacity-75"
					fill="currentColor"
					d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
				/>
			</svg>
		{/if}
		{@render children()}
	</button>
{/if}
