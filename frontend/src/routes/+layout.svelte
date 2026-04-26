<script lang="ts">
	import '@fontsource-variable/inter';
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import Toaster from '$lib/components/ui/toast/Toaster.svelte';

	let { children } = $props();
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>
{@render children()}
<Toaster />
