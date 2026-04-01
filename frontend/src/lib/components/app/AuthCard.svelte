<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		title: string;
		children: Snippet;
		footer?: Snippet;
	};

	let { title, children, footer }: Props = $props();
</script>

<div class="flex min-h-screen items-center justify-center">
	<div class="w-full max-w-sm">
		<h1 class="mb-6 text-2xl font-bold text-foreground">{title}</h1>
		{@render children()}
		{#if footer}
			<div class="mt-4 text-center text-sm text-muted-foreground">
				{@render footer()}
			</div>
		{/if}
	</div>
</div>
