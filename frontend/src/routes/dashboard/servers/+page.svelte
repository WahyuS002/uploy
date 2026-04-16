<script lang="ts">
	import { goto, invalidateAll } from '$app/navigation';
	import { page } from '$app/state';
	import { createApiClient } from '$lib/api/client';
	import type { PageData } from './$types';
	import PageHeader from '$lib/components/app/PageHeader.svelte';
	import FormField from '$lib/components/app/FormField.svelte';
	import StatusBadge from '$lib/components/app/StatusBadge.svelte';
	import CopyButton from '$lib/components/app/CopyButton.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import Alert from '$lib/components/ui/Alert.svelte';
	import CodeBlock from '$lib/components/ui/CodeBlock.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import { Server } from '@steeze-ui/heroicons';

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

	let keyItems = $derived(keys.map((k) => ({ value: k.id, label: k.name })));

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

	function safeReturnTo(): string | null {
		const raw = page.url.searchParams.get('returnTo');
		if (!raw) return null;
		if (!raw.startsWith('/') || raw.startsWith('//')) return null;
		return raw;
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
			const returnTo = safeReturnTo();
			if (returnTo) {
				// eslint-disable-next-line svelte/no-navigation-without-resolve
				await goto(returnTo);
				return;
			}
			await invalidateAll();
		} catch {
			error = 'Network error, please try again';
		} finally {
			loading = false;
		}
	}
</script>

<section>
	<PageHeader title="Servers" />

	{#if isOwner}
		<form
			onsubmit={(e) => {
				e.preventDefault();
				createServer();
			}}
			class="mb-8 flex max-w-md flex-col gap-3"
		>
			<FormField label="Name">
				<Input type="text" bind:value={name} required placeholder="production-server" />
			</FormField>
			<FormField label="Host">
				<Input type="text" bind:value={host} required placeholder="192.168.1.100" />
			</FormField>
			<FormField label="Port">
				<Input type="number" bind:value={port} required min={1} max={65535} />
			</FormField>
			<FormField label="SSH User">
				<Input type="text" bind:value={sshUser} required placeholder="root" />
			</FormField>
			<FormField label="SSH Key">
				<Select items={keyItems} bind:value={sshKeyId} required placeholder="Select an SSH key" />
			</FormField>

			{#if selectedKeyPublicKey}
				<Alert tone="info">
					<p class="mb-1 text-xs font-medium">
						Public key (add to <code class="rounded bg-blue-100 px-1">~/.ssh/authorized_keys</code>
						on remote server):
					</p>
					<div class="flex items-start gap-2">
						<CodeBlock code={selectedKeyPublicKey} class="flex-1 bg-white" />
						<CopyButton text={selectedKeyPublicKey} defaultLabel="Copy" />
					</div>
				</Alert>
			{/if}

			{#if error}
				<p class="text-sm text-danger">{error}</p>
			{/if}

			<div class="flex gap-2">
				<Button
					type="button"
					variant="secondary"
					disabled={!canCheckConnection}
					onclick={checkConnection}
				>
					{#if checking}
						Checking...
					{:else if isVerified}
						Connected
					{:else}
						Check Connection
					{/if}
				</Button>
				<Button type="submit" {loading} disabled={!isVerified}>
					{loading ? 'Saving...' : 'Add Server'}
				</Button>
			</div>
		</form>
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
								<p class="mt-0.5 text-xs text-danger" title={server.proxy_last_error}>
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
