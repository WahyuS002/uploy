<script lang="ts">
	import Sidebar from '$lib/components/Sidebar.svelte';

	let { data, children } = $props();
</script>

<div class="flex min-h-screen">
	<Sidebar workspaceName={data.workspace?.name ?? 'Uploy'} userEmail={data.user?.email ?? ''} />

	<main class="flex-1 p-6">
		{@render children()}
	</main>
</div>
