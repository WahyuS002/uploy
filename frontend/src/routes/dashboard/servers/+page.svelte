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
	let copied = $state(false);

	// Check connection state
	let checking = $state(false);
	let verified = $state<{ host: string; port: number; sshUser: string; sshKeyId: string } | null>(
		null
	);

	let selectedKeyPublicKey = $derived(keys.find((k) => k.id === sshKeyId)?.public_key ?? '');

	let isVerified = $derived(
		verified !== null &&
			verified.host === host &&
			verified.port === port &&
			verified.sshUser === sshUser &&
			verified.sshKeyId === sshKeyId
	);

	let canCheckConnection = $derived(
		host.trim() !== '' && sshUser.trim() !== '' && sshKeyId !== '' && !checking
	);

	const api = createApiClient();

	async function checkConnection() {
		error = '';
		checking = true;
		verified = null;
		try {
			const { error: err } = await api.POST('/api/servers/check-connection', {
				body: { host, port, ssh_user: sshUser, ssh_key_id: sshKeyId }
			});
			if (err) {
				error = (err as { error: string }).error;
				return;
			}
			verified = { host, port, sshUser, sshKeyId };
		} catch {
			error = 'Network error, please try again';
		} finally {
			checking = false;
		}
	}

	async function createServer() {
		if (!isVerified) return;
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
			verified = null;
			await invalidateAll();
		} catch {
			error = 'Network error, please try again';
		} finally {
			loading = false;
		}
	}

	async function copyPublicKey(text: string) {
		await navigator.clipboard.writeText(text);
		copied = true;
		setTimeout(() => (copied = false), 2000);
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

			{#if selectedKeyPublicKey}
				<div class="rounded border border-blue-200 bg-blue-50 p-3">
					<p class="mb-1 text-xs font-medium text-blue-800">
						Public key (add to <code class="rounded bg-blue-100 px-1">~/.ssh/authorized_keys</code>
						on remote server):
					</p>
					<div class="flex items-start gap-2">
						<pre
							class="flex-1 overflow-x-auto rounded bg-white p-2 font-mono text-xs break-all whitespace-pre-wrap">{selectedKeyPublicKey}</pre>
						<button
							type="button"
							class="shrink-0 cursor-pointer rounded border px-2 py-1 text-xs hover:bg-gray-50"
							onclick={() => copyPublicKey(selectedKeyPublicKey)}
						>
							{copied ? 'Copied!' : 'Copy'}
						</button>
					</div>
				</div>
			{/if}

			{#if error}
				<p class="text-sm text-red-600">{error}</p>
			{/if}

			<div class="flex gap-2">
				<button
					type="button"
					disabled={!canCheckConnection}
					class="cursor-pointer rounded-sm border border-black px-3 py-2 text-sm disabled:cursor-not-allowed disabled:opacity-50"
					onclick={checkConnection}
				>
					{#if checking}
						Checking...
					{:else if isVerified}
						Connected
					{:else}
						Check Connection
					{/if}
				</button>
				<button
					type="submit"
					disabled={loading || !isVerified}
					class="cursor-pointer rounded-sm bg-black px-3 py-2 text-white disabled:cursor-not-allowed disabled:opacity-50"
				>
					{loading ? 'Saving...' : 'Add Server'}
				</button>
			</div>
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
							<span
								class="rounded px-2 py-0.5 text-xs font-medium"
								class:bg-green-100={server.proxy_status === 'ready'}
								class:text-green-700={server.proxy_status === 'ready'}
								class:bg-red-100={server.proxy_status === 'port_conflict' ||
									server.proxy_status === 'degraded'}
								class:text-red-700={server.proxy_status === 'port_conflict' ||
									server.proxy_status === 'degraded'}
								class:bg-yellow-100={server.proxy_status === 'not_configured'}
								class:text-yellow-700={server.proxy_status === 'not_configured'}
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
						</td>
						<td class="py-2 text-gray-500">{new Date(server.created_at).toLocaleDateString()}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{:else}
		<p class="text-sm text-gray-500">No servers registered yet.</p>
	{/if}
</section>
