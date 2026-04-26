<script lang="ts">
	import { Dialog as BitsDialog } from 'bits-ui';
	import { cn } from '../cn.js';

	let { class: className, ...rest }: BitsDialog.CloseProps = $props();
</script>

<BitsDialog.Close {...rest} data-slot="dialog-close" class={cn(className)} />
