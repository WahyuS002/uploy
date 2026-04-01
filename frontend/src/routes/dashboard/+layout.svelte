<script lang="ts">
	import Sidebar from '$lib/components/Sidebar.svelte';

	let { data, children } = $props();
</script>

<div class="flex min-h-screen bg-background">
	<Sidebar workspaceName={data.workspace?.name ?? 'Uploy'} userEmail={data.user?.email ?? ''} />

	<main class="m-4 flex-1 rounded-lg border border-border bg-surface px-5 py-2">
		{@render children()}
	</main>
</div>
