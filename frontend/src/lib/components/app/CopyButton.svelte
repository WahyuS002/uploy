<script lang="ts">
	import Button from '$lib/components/ui/Button.svelte';

	type Props = {
		text: string;
		defaultLabel?: string;
		copiedLabel?: string;
		variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
		size?: 'sm' | 'md';
		class?: string;
	};

	let {
		text,
		defaultLabel = 'Copy',
		copiedLabel = 'Copied!',
		variant = 'secondary',
		size = 'sm',
		class: className
	}: Props = $props();

	let copied = $state(false);

	async function copy() {
		await navigator.clipboard.writeText(text);
		copied = true;
		setTimeout(() => (copied = false), 2000);
	}
</script>

<Button type="button" {variant} {size} class={className} onclick={copy}>
	{copied ? copiedLabel : defaultLabel}
</Button>
