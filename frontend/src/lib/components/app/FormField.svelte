<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		label: string;
		error?: string;
		children: Snippet;
	};

	let { label, error, children }: Props = $props();
</script>

<div class="flex flex-col gap-1">
	<label class="text-sm font-medium text-foreground">
		{label}
		{@render children()}
	</label>
	{#if error}
		<p class="text-sm text-danger">{error}</p>
	{/if}
</div>
