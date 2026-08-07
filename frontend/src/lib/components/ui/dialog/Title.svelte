<script lang="ts">
	import { Dialog as BitsDialog } from 'bits-ui';
	import { cn } from '../cn.js';

	let { class: className, ...rest }: BitsDialog.TitleProps = $props();
</script>

<BitsDialog.Title
	{...rest}
	data-slot="dialog-title"
	class={cn('text-base font-semibold text-foreground', className)}
/>
