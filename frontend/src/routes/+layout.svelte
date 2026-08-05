<script lang="ts">
	import '@fontsource-variable/inter';
	import './layout.css';
	import Toaster from '$lib/components/ui/toast/Toaster.svelte';

	let { children } = $props();
</script>

{@render children()}
<Toaster />
