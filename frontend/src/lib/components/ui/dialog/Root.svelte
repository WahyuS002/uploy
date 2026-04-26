<script lang="ts">
	import { Dialog as BitsDialog } from 'bits-ui';

	let { open = $bindable(false), ...rest }: BitsDialog.RootProps = $props();
</script>

<BitsDialog.Root bind:open {...rest} />
