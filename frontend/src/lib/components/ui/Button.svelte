<script lang="ts" module>
	import { cva, type VariantProps } from 'class-variance-authority';

	export const buttonVariants = cva(
		'inline-flex cursor-pointer items-center justify-center gap-2 rounded-md text-sm font-medium tracking-[-0.01em] transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-50',
		{
			variants: {
				variant: {
					primary:
						'bg-[#1a1b1e] text-white shadow-[0_1px_0_rgba(17,17,17,0.04),0_1px_2px_rgba(17,17,17,0.22),inset_0_0_0_1px_#3a3d42,inset_0_1px_0_rgba(255,255,255,0.06)] hover:bg-[#2c3037] focus-visible:bg-[#2c3037] active:bg-[#45494f] active:shadow-[0_1px_0_rgba(17,17,17,0.04),0_1px_2px_rgba(17,17,17,0.18),inset_0_0_0_1px_#45494f,inset_0_1px_0_rgba(255,255,255,0.04)]',
					secondary:
						'border border-border bg-card text-card-foreground hover:bg-accent hover:text-accent-foreground',
					ghost: 'text-foreground hover:bg-accent hover:text-accent-foreground',
					destructive: 'bg-destructive text-destructive-foreground hover:bg-destructive/90'
				},
				size: {
					default: 'h-10 px-4 py-2',
					sm: 'h-8 px-3 text-xs'
				}
			},
			defaultVariants: { variant: 'primary', size: 'default' }
		}
	);

	type CanonicalVariant = NonNullable<VariantProps<typeof buttonVariants>['variant']>;
	type CanonicalSize = NonNullable<VariantProps<typeof buttonVariants>['size']>;

	export type ButtonVariant = CanonicalVariant | 'subtle' | 'danger';
	export type ButtonSize = CanonicalSize | 'md' | 'xs';

	const variantAlias: Record<string, CanonicalVariant> = {
		subtle: 'secondary',
		danger: 'destructive'
	};
	const sizeAlias: Record<string, CanonicalSize> = {
		md: 'default',
		xs: 'sm'
	};

	function resolveVariant(v: ButtonVariant | undefined): CanonicalVariant {
		if (!v) return 'primary';
		return variantAlias[v] ?? (v as CanonicalVariant);
	}
	function resolveSize(s: ButtonSize | undefined): CanonicalSize {
		if (!s) return 'default';
		return sizeAlias[s] ?? (s as CanonicalSize);
	}
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

	let resolvedClass = $derived(
		cn(buttonVariants({ variant: resolveVariant(variant), size: resolveSize(size) }), className)
	);
</script>

{#if href}
	<!-- eslint-disable svelte/no-navigation-without-resolve -->
	<a {href} class={resolvedClass} {...rest as HTMLAnchorAttributes}>
		{@render children()}
	</a>
	<!-- eslint-enable svelte/no-navigation-without-resolve -->
{:else}
	<button
		class={resolvedClass}
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
