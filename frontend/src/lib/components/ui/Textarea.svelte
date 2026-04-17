<script lang="ts" module>
	import { cva, type VariantProps } from 'class-variance-authority';

	export const textareaVariants = cva(
		'block w-full rounded-md border field-focus-glow border-border-input bg-surface font-normal text-foreground placeholder:text-placeholder disabled:opacity-50',
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

	export type TextareaSize = VariantProps<typeof textareaVariants>['size'];
</script>

<script lang="ts">
	import { cn } from './cn.js';
	import type { HTMLTextareaAttributes } from 'svelte/elements';

	type Props = Omit<HTMLTextareaAttributes, 'class'> & {
		class?: string;
		size?: TextareaSize;
		value?: string;
	};

	let { class: className, size, value = $bindable(), ...rest }: Props = $props();
</script>

<textarea bind:value class={cn(textareaVariants({ size }), className)} {...rest}></textarea>
