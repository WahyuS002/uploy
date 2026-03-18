<script lang="ts">
	import { onMount } from 'svelte';

	onMount(async () => {
		const res = await fetch('/api/auth/me');
		if (res.ok) {
			window.location.href = '/dashboard';
		} else {
			window.location.href = '/login';
		}
	});
</script>

<div class="flex min-h-screen items-center justify-center">
	<p class="text-gray-500">Loading...</p>
</div>
