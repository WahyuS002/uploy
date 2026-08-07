<script lang="ts" module>
	import { cva } from 'class-variance-authority';

	/* The one place the "pick one of these" treatment is defined. Both the dot
	   colour and the row policy live here on purpose: the row must NOT fill on
	   checked, because the hover fill is the same --accent and the two states
	   would render identically. */
	export const radioListVariants = cva('divide-y divide-border overflow-y-auto');

	export const radioListItemVariants = cva(
		'flex cursor-pointer items-center gap-2.5 px-3 py-2.5 transition-colors hover:bg-accent'
	);

	export const radioListInputVariants = cva(
		'form-radio h-3.5 w-3.5 flex-none border-input text-primary-deep focus:ring-2 focus:ring-ring/40 focus:ring-offset-0'
	);
</script>

<script lang="ts" generics="T extends { id: string }">
	import type { Snippet } from 'svelte';
	import { cn } from './cn.js';

	type Props = {
		items: T[];
		/** id of the checked item */
		value?: string;
		/** Radio group name. Must be unique per rendered group — pass $props.id(). */
		name: string;
		ariaLabel?: string;
		ariaLabelledby?: string;
		onChange: (id: string) => void;
		/** Row content, rendered after the radio. Give the growing part flex-1. */
		children: Snippet<[T]>;
		class?: string;
	};

	let {
		items,
		value = '',
		name,
		ariaLabel,
		ariaLabelledby,
		onChange,
		children,
		class: className
	}: Props = $props();
</script>

<div
	role="radiogroup"
	aria-label={ariaLabel}
	aria-labelledby={ariaLabelledby}
	class={cn(radioListVariants(), className)}
>
	{#each items as item (item.id)}
		<label class={radioListItemVariants()}>
			<input
				type="radio"
				{name}
				value={item.id}
				checked={value === item.id}
				onchange={() => onChange(item.id)}
				class={radioListInputVariants()}
			/>
			{@render children(item)}
		</label>
	{/each}
</div>
