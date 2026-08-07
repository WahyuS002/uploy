<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import type { PageData } from './$types';
	import PublicKeyHelper from '$lib/components/app/PublicKeyHelper.svelte';
	import StatusBadge from '$lib/components/app/StatusBadge.svelte';
	import ServerConnectWizard from '$lib/components/app/ServerConnectWizard.svelte';
	import { ServerCreateController } from '$lib/components/app/server-create-form.svelte';
	import { formatDate } from '$lib/format-date';
	import { Button, EmptyState } from '$lib/components/ui';
	import { Dialog, DialogContent, DialogHeader, DialogTitle } from '$lib/components/ui/dialog';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { ChevronRight, Key, Plus, Server } from '@steeze-ui/heroicons';

	let { data }: { data: PageData } = $props();

	let isOwner = $derived(data.workspace?.role === 'owner');
	let servers = $derived(data.servers ?? []);
	let keysById = $derived(new Map((data.keys ?? []).map((k) => [k.id, k])));

	let dialogOpen = $state(false);
	let expandedServerId = $state<string | null>(null);

	const serverController = new ServerCreateController({
		onSuccess: async () => {
			dialogOpen = false;
			await invalidateAll();
		}
	});

	function openCreate() {
		serverController.reset();
		dialogOpen = true;
	}
</script>

<svelte:head>
	<title>Servers · Uploy</title>
</svelte:head>

<section class="flex flex-1 flex-col">
	<div class="mb-5 flex items-center justify-between gap-4">
		<div class="min-w-0">
			<h1 class="text-xl font-semibold tracking-tight text-foreground">All servers</h1>
			{#if servers.length > 0}
				<p class="mt-1 text-sm text-muted-foreground">
					{servers.length}
					{servers.length === 1 ? 'server' : 'servers'} reachable over SSH.
				</p>
			{/if}
		</div>
		{#if isOwner}
			<Button variant="primary" size="sm" onclick={openCreate}>
				<Icon src={Plus} theme="outline" class="h-3.5 w-3.5" />
				Add server
			</Button>
		{/if}
	</div>

	<div class="flex flex-1 flex-col">
		{#if servers.length > 0}
			<div class="overflow-x-auto">
				<table class="w-full min-w-[44rem] text-left text-sm">
					<thead>
						<tr class="border-b border-border text-xs text-muted-foreground">
							<th scope="col" class="pb-2 font-medium">Server</th>
							<th scope="col" class="pb-2 font-medium">SSH Access</th>
							<th scope="col" class="pb-2 font-medium">Proxy Status</th>
							<th scope="col" class="pb-2 font-medium">Created</th>
						</tr>
					</thead>
					<tbody>
						{#each servers as server (server.id)}
							{@const key = keysById.get(server.ssh_key_id)}
							{@const expanded = expandedServerId === server.id}
							<tr class="border-b border-border">
								<td class="py-2.5 align-top">
									<p class="font-medium text-foreground">{server.name}</p>
									<p class="mt-0.5 font-mono text-xs text-muted-foreground">
										{server.host}:{server.port}
									</p>
								</td>
								<td class="py-2.5 align-top">
									<p class="font-mono text-xs text-muted-foreground">{server.ssh_user}</p>
									{#if key?.public_key}
										<button
											type="button"
											id="public-key-toggle-{server.id}"
											aria-expanded={expanded}
											aria-controls="public-key-{server.id}"
											onclick={() => (expandedServerId = expanded ? null : server.id)}
											class="mt-1 inline-flex cursor-pointer items-center gap-1 rounded text-xs text-foreground underline decoration-border underline-offset-2 transition-colors hover:decoration-foreground focus-visible:ring-2 focus-visible:ring-ring/40 focus-visible:outline-none"
										>
											<Icon src={Key} theme="outline" class="h-3 w-3 text-muted-foreground" />
											{key.name}
										</button>
									{:else}
										<p class="mt-1 inline-flex items-center gap-1 text-xs text-muted-foreground">
											<Icon src={Key} theme="outline" class="h-3 w-3" />
											{key?.name ?? 'Key unavailable'}
										</p>
									{/if}
								</td>
								<td class="py-2.5 align-top">
									<StatusBadge status={server.proxy_status} />
									{#if server.proxy_last_error}
										<p class="mt-0.5 text-xs text-destructive" title={server.proxy_last_error}>
											{server.proxy_last_error.length > 50
												? server.proxy_last_error.slice(0, 50) + '...'
												: server.proxy_last_error}
										</p>
									{/if}
								</td>
								<td class="py-2.5 align-top text-muted-foreground">
									{formatDate(server.created_at)}
								</td>
							</tr>
							{#if expanded && key?.public_key}
								<tr class="border-b border-border">
									<td colspan="4" class="pt-1 pb-3">
										<div id="public-key-{server.id}" class="max-w-2xl">
											<PublicKeyHelper
												publicKey={key.public_key}
												description="Add to ~/.ssh/authorized_keys for {server.ssh_user} on {server.host}."
											/>
										</div>
									</td>
								</tr>
							{/if}
						{/each}
					</tbody>
				</table>
			</div>
		{:else}
			<EmptyState
				variant="canvas"
				icon={Server}
				title="No servers connected yet"
				description={isOwner
					? 'Connect your first server to start deploying services to your own infrastructure. You can generate an SSH key along the way.'
					: 'Ask a workspace owner to connect a server before deploying.'}
			>
				{#snippet actions()}
					{#if isOwner}
						<Button variant="primary" size="sm" onclick={openCreate}>
							<Icon src={Plus} theme="outline" class="h-3.5 w-3.5" />
							Add server
						</Button>
					{/if}
				{/snippet}
			</EmptyState>
		{/if}
	</div>
</section>

<Dialog bind:open={dialogOpen}>
	<DialogContent class="w-[min(92vw,32rem)] max-w-none overflow-hidden">
		<DialogHeader class="border-b border-border px-5 pt-4 pr-12 pb-3">
			<DialogTitle class="flex items-center gap-2 text-sm">
				<span class="font-medium text-muted-foreground">Servers</span>
				<Icon src={ChevronRight} theme="outline" class="h-3.5 w-3.5 text-muted-foreground" />
				<span
					class="grid h-6 w-6 place-content-center rounded-full border border-border bg-muted"
					aria-hidden="true"
				>
					<Icon src={Server} theme="outline" class="h-3.5 w-3.5 text-muted-foreground" />
				</span>
				New server
			</DialogTitle>
		</DialogHeader>
		<ServerConnectWizard
			controller={serverController}
			bodyClass="max-h-[min(65vh,32rem)] overflow-y-auto px-5 pt-4 pb-5"
			actionsClass="rounded-b-xl border-t border-border px-5 py-3"
		/>
	</DialogContent>
</Dialog>
