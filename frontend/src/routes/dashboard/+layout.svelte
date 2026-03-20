<script lang="ts">
	let { data, children } = $props();

	async function handleLogout() {
		await fetch('/api/auth/logout', { method: 'POST' });
		window.location.href = '/login';
	}
</script>

<div class="min-h-screen">
	<nav class="flex items-center justify-between border-b px-6 py-3">
		<div class="flex items-center gap-4">
			<a href="/dashboard" class="text-lg font-bold">Uploy</a>
			<a href="/dashboard/ssh-keys" class="text-sm text-gray-600 hover:text-black">SSH Keys</a>
			<a href="/dashboard/servers" class="text-sm text-gray-600 hover:text-black">Servers</a>
			<span class="text-sm text-gray-500">{data.workspace?.name}</span>
		</div>
		<div class="flex items-center gap-4">
			<span class="text-sm text-gray-600">{data.user?.email}</span>
			<button
				onclick={handleLogout}
				class="cursor-pointer rounded border px-3 py-1 text-sm hover:bg-gray-50"
			>
				Sign out
			</button>
		</div>
	</nav>

	<main class="p-6">
		{@render children()}
	</main>
</div>
