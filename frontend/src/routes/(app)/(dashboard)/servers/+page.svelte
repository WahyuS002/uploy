<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import type { PageData } from './$types';
	import PageHeader from '$lib/components/app/PageHeader.svelte';
	import StatusBadge from '$lib/components/app/StatusBadge.svelte';
	import ServerCreateFields from '$lib/components/app/ServerCreateFields.svelte';
	import { ServerCreateController } from '$lib/components/app/server-create-form.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import { Server } from '@steeze-ui/heroicons';

	let { data }: { data: PageData } = $props();

	let isOwner = $derived(data.workspace?.role === 'owner');
	let servers = $derived(data.servers);

	async function handleServerCreated() {
		await invalidateAll();
	}

	const serverController = new ServerCreateController({ onSuccess: handleServerCreated });
</script>

<section>
	<PageHeader title="Servers" icon={Server} />

	{#if isOwner}
		<div class="mb-8 max-w-md">
			<form
				onsubmit={(e) => {
					e.preventDefault();
					serverController.createServer();
				}}
				class="flex flex-col gap-3"
			>
				<ServerCreateFields controller={serverController} />
				<div class="flex gap-2">
					<Button
						type="button"
						variant="secondary"
						disabled={!serverController.canCheckConnection}
						onclick={serverController.checkConnection}
					>
						{#if serverController.checking}
							Checking...
						{:else if serverController.isVerified}
							Connected
						{:else}
							Check Connection
						{/if}
					</Button>
					<Button
						type="submit"
						loading={serverController.loading}
						disabled={!serverController.isVerified || !!serverController.keysError}
					>
						{serverController.loading ? 'Saving...' : 'Add Server'}
					</Button>
				</div>
			</form>
		</div>
	{/if}

	{#if servers.length > 0}
		<table class="w-full max-w-2xl text-left text-sm">
			<thead>
				<tr class="border-b border-border text-muted-foreground">
					<th class="pb-2 font-medium">Name</th>
					<th class="pb-2 font-medium">Host</th>
					<th class="pb-2 font-medium">User</th>
					<th class="pb-2 font-medium">Proxy</th>
					<th class="pb-2 font-medium">Created</th>
				</tr>
			</thead>
			<tbody>
				{#each servers as server (server.id)}
					<tr class="border-b border-border">
						<td class="py-2 text-foreground">{server.name}</td>
						<td class="py-2 font-mono text-xs text-muted-foreground">{server.host}:{server.port}</td
						>
						<td class="py-2 text-muted-foreground">{server.ssh_user}</td>
						<td class="py-2">
							<StatusBadge status={server.proxy_status} />
							{#if server.proxy_last_error}
								<p class="mt-0.5 text-xs text-destructive" title={server.proxy_last_error}>
									{server.proxy_last_error.length > 50
										? server.proxy_last_error.slice(0, 50) + '...'
										: server.proxy_last_error}
								</p>
							{/if}
						</td>
						<td class="py-2 text-muted-foreground"
							>{new Date(server.created_at).toLocaleDateString()}</td
						>
					</tr>
				{/each}
			</tbody>
		</table>
	{:else}
		<EmptyState
			icon={Server}
			title="No servers registered yet"
			description="Connect your first server to start deploying services to your own infrastructure."
		/>
	{/if}
</section>
