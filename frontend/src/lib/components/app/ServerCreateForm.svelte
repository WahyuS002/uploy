<script lang="ts">
	import { createApiClient } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import FormField from '$lib/components/app/FormField.svelte';
	import CopyButton from '$lib/components/app/CopyButton.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import Alert from '$lib/components/ui/Alert.svelte';
	import CodeBlock from '$lib/components/ui/CodeBlock.svelte';

	type SSHKeyResponse = components['schemas']['SSHKeyResponse'];
	type ServerResponse = components['schemas']['ServerResponse'];

	type Props = {
		onsuccess?: (server: ServerResponse) => void;
	};

	let { onsuccess }: Props = $props();

	const api = createApiClient();

	let keys = $state<SSHKeyResponse[]>([]);
	let keysLoading = $state(true);
	let keysError = $state('');

	let name = $state('');
	let host = $state('');
	let port = $state(22);
	let sshUser = $state('root');
	let sshKeyId = $state('');
	let error = $state('');
	let loading = $state(false);

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

	async function loadKeys() {
		keysLoading = true;
		keysError = '';
		try {
			const res = await api.GET('/api/ssh-keys');
			if (res.error) {
				keysError = (res.error as { error: string }).error ?? 'Failed to load SSH keys';
				return;
			}
			if (res.data) keys = res.data;
		} catch {
			keysError = 'Network error loading SSH keys';
		} finally {
			keysLoading = false;
		}
	}

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
			const { data, error: err } = await api.POST('/api/servers', {
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
			if (data) onsuccess?.(data);
		} catch {
			error = 'Network error, please try again';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		loadKeys();
	});
</script>

{#if keysLoading}
	<div class="flex flex-col gap-3">
		<div class="h-10 animate-pulse rounded-lg bg-surface-muted"></div>
		<div class="h-10 animate-pulse rounded-lg bg-surface-muted"></div>
	</div>
{:else if keysError}
	<Alert tone="danger">
		<p class="text-sm">{keysError}</p>
	</Alert>
{:else if keys.length === 0}
	<Alert tone="warning">
		<p class="text-sm">
			No SSH keys found. <a href="/dashboard/ssh-keys" class="underline hover:no-underline">Add an SSH key</a> first, then come back to add a server.
		</p>
	</Alert>
{:else}
	<form
		onsubmit={(e) => {
			e.preventDefault();
			createServer();
		}}
		class="flex flex-col gap-3"
	>
		<FormField label="Name">
			<Input type="text" bind:value={name} required placeholder="production-server" />
		</FormField>
		<FormField label="Host">
			<Input type="text" bind:value={host} required placeholder="192.168.1.100" />
		</FormField>
		<div class="grid grid-cols-2 gap-3">
			<FormField label="Port">
				<Input type="number" bind:value={port} required min={1} max={65535} />
			</FormField>
			<FormField label="SSH User">
				<Input type="text" bind:value={sshUser} required placeholder="root" />
			</FormField>
		</div>
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
