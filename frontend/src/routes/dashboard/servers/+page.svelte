<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { createApiClient } from '$lib/api/client';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	let isOwner = $derived(data.workspace?.role === 'owner');
	let servers = $derived(data.servers);
	let keys = $derived(data.keys);
	let name = $state('');
	let host = $state('');
	let port = $state(22);
	let sshUser = $state('root');
	let sshKeyId = $state('');
	let error = $state('');
	let loading = $state(false);

	const api = createApiClient();

	async function createServer() {
		error = '';
		loading = true;
		try {
			const { error: err } = await api.POST('/api/servers', {
				body: { name, host, port, ssh_user: sshUser, ssh_key_id: sshKeyId }
			});
			if (err) {
				error = (err as { error: string }).error;
				return;
			}
			name = '';
			host = '';
			port = 22;
			sshUser = 'root';
			sshKeyId = '';
			await invalidateAll();
		} catch {
			error = 'Network error, please try again';
		} finally {
			loading = false;
		}
	}
</script>

<section>
	<h2 class="mb-4 text-xl font-bold">Servers</h2>

	{#if isOwner}
		<form
			onsubmit={(e) => {
				e.preventDefault();
				createServer();
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
				Host
				<input
					type="text"
					bind:value={host}
					required
					class="rounded border p-1"
					placeholder="192.168.1.100"
				/>
			</label>
			<label class="flex flex-col gap-1 text-sm">
				Port
				<input
					type="number"
					bind:value={port}
					required
					class="rounded border p-1"
					min="1"
					max="65535"
				/>
			</label>
			<label class="flex flex-col gap-1 text-sm">
				SSH User
				<input
					type="text"
					bind:value={sshUser}
					required
					class="rounded border p-1"
					placeholder="root"
				/>
			</label>
			<label class="flex flex-col gap-1 text-sm">
				SSH Key
				<select bind:value={sshKeyId} required class="rounded border p-1">
					<option value="" disabled>Select an SSH key</option>
					{#each keys as key (key.id)}
						<option value={key.id}>{key.name}</option>
					{/each}
				</select>
			</label>

			{#if error}
				<p class="text-sm text-red-600">{error}</p>
			{/if}

			<button
				type="submit"
				disabled={loading}
				class="cursor-pointer rounded-sm bg-black p-2 text-white disabled:opacity-50"
			>
				{loading ? 'Testing & saving...' : 'Add Server'}
			</button>
		</form>
	{/if}

	{#if servers.length > 0}
		<table class="w-full max-w-2xl text-left text-sm">
			<thead>
				<tr class="border-b text-gray-500">
					<th class="pb-2 font-medium">Name</th>
					<th class="pb-2 font-medium">Host</th>
					<th class="pb-2 font-medium">User</th>
					<th class="pb-2 font-medium">Proxy</th>
					<th class="pb-2 font-medium">Created</th>
				</tr>
			</thead>
			<tbody>
				{#each servers as server (server.id)}
					<tr class="border-b">
						<td class="py-2">{server.name}</td>
						<td class="py-2 font-mono text-xs text-gray-500">{server.host}:{server.port}</td>
						<td class="py-2 text-gray-500">{server.ssh_user}</td>
						<td class="py-2">
							{#if server.proxy_status === 'not_configured' && server.proxy_mode === 'none'}
								<span class="rounded bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600">
									direct only
								</span>
							{:else}
								<span
									class="rounded px-2 py-0.5 text-xs font-medium"
									class:bg-green-100={server.proxy_status === 'ready'}
									class:text-green-700={server.proxy_status === 'ready'}
									class:bg-red-100={server.proxy_status === 'port_conflict' || server.proxy_status === 'degraded'}
									class:text-red-700={server.proxy_status === 'port_conflict' || server.proxy_status === 'degraded'}
									class:bg-yellow-100={server.proxy_status === 'tls_pending' || server.proxy_status === 'not_configured'}
									class:text-yellow-700={server.proxy_status === 'tls_pending' || server.proxy_status === 'not_configured'}
								>
									{server.proxy_status.replace('_', ' ')}
								</span>
								{#if server.proxy_last_error}
									<p class="mt-0.5 text-xs text-red-500" title={server.proxy_last_error}>
										{server.proxy_last_error.length > 50
											? server.proxy_last_error.slice(0, 50) + '...'
											: server.proxy_last_error}
									</p>
								{/if}
							{/if}
						</td>
						<td class="py-2 text-gray-500"
							>{new Date(server.created_at).toLocaleDateString()}</td
						>
					</tr>
				{/each}
			</tbody>
		</table>
	{:else}
		<p class="text-sm text-gray-500">No servers registered yet.</p>
	{/if}
</section>
