import { cva, type VariantProps } from 'class-variance-authority';

export const pillVariants = cva(
	'inline-flex h-7 cursor-pointer items-center gap-1.5 rounded-lg px-2.5 text-sm font-medium tracking-[-0.01em] outline-none transition-colors focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:ring-offset-background disabled:cursor-not-allowed disabled:opacity-75',
	{
		variants: {
			state: {
				applied: 'border border-border bg-card text-foreground hover:bg-accent',
				placeholder:
					'border border-dashed border-input bg-transparent text-muted-foreground hover:bg-accent disabled:hover:bg-transparent'
			}
		},
		defaultVariants: { state: 'applied' }
	}
);

export type PillState = VariantProps<typeof pillVariants>['state'];
