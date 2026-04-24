<script lang="ts">
	import Sidebar from '$lib/components/Sidebar.svelte';
	import Topbar from '$lib/components/Topbar.svelte';

	let { data, children } = $props();
</script>

<div class="flex min-h-screen flex-col bg-background">
	<Topbar userEmail={data.user?.email ?? ''} />

	<div class="flex flex-1">
		<Sidebar workspaceName={data.workspace?.name ?? 'Uploy'} />

		<main class="mx-4 mb-4 flex-1 rounded-lg border border-border bg-card px-60 py-14 text-card-foreground">
			{@render children()}
		</main>
	</div>
</div>
