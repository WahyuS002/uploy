<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import type { PageData } from './$types';
	import StatusBadge from '$lib/components/app/StatusBadge.svelte';
	import ServerCreatePanel from '$lib/components/app/ServerCreatePanel.svelte';
	import { ServerCreateController } from '$lib/components/app/server-create-form.svelte';
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
	{#if isOwner}
		<div class="mb-8 max-w-md">
			<ServerCreatePanel controller={serverController} />
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
