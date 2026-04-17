<script lang="ts">
	import { Dialog as BitsDialog } from 'bits-ui';
	import type { Snippet } from 'svelte';

	type Props = {
		open?: boolean;
		children: Snippet;
	};

	let { open = $bindable(false), children }: Props = $props();
</script>

<BitsDialog.Root bind:open>
	{@render children()}
</BitsDialog.Root>
