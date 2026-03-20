<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { createApiClient } from '$lib/api/client';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	let isOwner = $derived(data.workspace?.role === 'owner');
	let keys = $derived(data.keys);
	let name = $state('');
	let privateKey = $state('');
	let error = $state('');
	let loading = $state(false);

	const api = createApiClient();

	async function createKey() {
		error = '';
		loading = true;
		try {
			const { error: err } = await api.POST('/api/ssh-keys', {
				body: { name, private_key: privateKey }
			});
			if (err) {
				error = (err as { error: string }).error;
				return;
			}
			name = '';
			privateKey = '';
			await invalidateAll();
		} catch {
			error = 'Network error, please try again';
		} finally {
			loading = false;
		}
	}
</script>

<section>
	<h2 class="mb-4 text-xl font-bold">SSH Keys</h2>

	{#if isOwner}
		<form
			onsubmit={(e) => {
				e.preventDefault();
				createKey();
			}}
			class="mb-8 flex max-w-md flex-col gap-2"
		>
			<label class="flex flex-col gap-1 text-sm">
				Name
				<input
					type="text"
					bind:value={name}
					required
					class="rounded border p-1"
					placeholder="production-server"
				/>
			</label>
			<label class="flex flex-col gap-1 text-sm">
				Private Key
				<textarea
					bind:value={privateKey}
					required
					rows="6"
					class="rounded border p-1 font-mono text-xs"
					placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
				></textarea>
			</label>

			{#if error}
				<p class="text-sm text-red-600">{error}</p>
			{/if}

			<button
				type="submit"
				disabled={loading}
				class="cursor-pointer rounded-sm bg-black p-2 text-white disabled:opacity-50"
			>
				{loading ? 'Saving...' : 'Add SSH Key'}
			</button>
		</form>
	{/if}

	{#if keys.length > 0}
		<table class="w-full max-w-2xl text-left text-sm">
			<thead>
				<tr class="border-b text-gray-500">
					<th class="pb-2 font-medium">Name</th>
					<th class="pb-2 font-medium">ID</th>
					<th class="pb-2 font-medium">Created</th>
				</tr>
			</thead>
			<tbody>
				{#each keys as key (key.id)}
					<tr class="border-b">
						<td class="py-2">{key.name}</td>
						<td class="py-2 font-mono text-xs text-gray-500">{key.id}</td>
						<td class="py-2 text-gray-500">{new Date(key.created_at).toLocaleDateString()}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{:else}
		<p class="text-sm text-gray-500">No SSH keys saved yet.</p>
	{/if}
</section>
