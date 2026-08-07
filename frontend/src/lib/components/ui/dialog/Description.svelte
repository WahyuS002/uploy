<script lang="ts">
	import { Dialog as BitsDialog } from 'bits-ui';
	import { cn } from '../cn.js';

	let { class: className, ...rest }: BitsDialog.DescriptionProps = $props();
</script>

<BitsDialog.Description
	{...rest}
	data-slot="dialog-description"
	class={cn('text-sm text-muted-foreground', className)}
/>
