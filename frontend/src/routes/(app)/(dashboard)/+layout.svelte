<script lang="ts">
	import Sidebar from '$lib/components/Sidebar.svelte';
	import Topbar from '$lib/components/Topbar.svelte';

	let { data, children } = $props();
</script>

<div class="flex min-h-screen bg-white">
	<Sidebar workspaceName={data.workspace?.name ?? 'Uploy'} workspaceRole={data.workspace?.role} />

	<div class="flex min-w-0 flex-1 flex-col">
		<Topbar userEmail={data.user?.email ?? ''} />
		<main class="flex-1 px-8 py-10 text-foreground">
			{@render children()}
		</main>
	</div>
</div>
