<script lang="ts">
	import { Dialog as BitsDialog } from 'bits-ui';
	import { cn } from '../cn.js';

	let { class: className, ...rest }: BitsDialog.TriggerProps = $props();
</script>

<BitsDialog.Trigger {...rest} data-slot="dialog-trigger" class={cn(className)} />
