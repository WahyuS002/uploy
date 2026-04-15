<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		label: string;
		error?: string;
		children: Snippet;
	};

	let { label, error, children }: Props = $props();
</script>

<label class="flex flex-col gap-1.5">
	<span class="text-xs text-muted-foreground">{label}</span>
	{@render children()}
	{#if error}
		<p class="text-sm text-danger">{error}</p>
	{/if}
</label>
