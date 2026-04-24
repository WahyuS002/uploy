<script lang="ts" module>
	import { cva, type VariantProps } from 'class-variance-authority';

	export const inputVariants = cva(
		'block w-full rounded-md border field-focus-glow border-input bg-background font-normal text-foreground placeholder:text-muted-foreground disabled:opacity-50',
		{
			variants: {
				size: {
					sm: 'px-2.5 py-1.5 text-xs',
					md: 'px-3 py-2 text-sm'
				}
			},
			defaultVariants: { size: 'md' }
		}
	);

	export type InputSize = VariantProps<typeof inputVariants>['size'];
</script>

<script lang="ts">
	import { cn } from './cn.js';
	import type { HTMLInputAttributes } from 'svelte/elements';

	type Props = Omit<HTMLInputAttributes, 'class' | 'size'> & {
		class?: string;
		size?: InputSize;
		nativeSize?: number;
		value?: HTMLInputAttributes['value'];
	};

	let { class: className, size, nativeSize, value = $bindable(), ...rest }: Props = $props();
</script>

<input bind:value size={nativeSize} class={cn(inputVariants({ size }), className)} {...rest} />
