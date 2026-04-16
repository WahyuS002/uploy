<script lang="ts">
	import { createApiClient } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import FormField from '$lib/components/app/FormField.svelte';
	import CopyButton from '$lib/components/app/CopyButton.svelte';
	import SSHKeyCreatePanel from '$lib/components/app/SSHKeyCreatePanel.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import Alert from '$lib/components/ui/Alert.svelte';
	import CodeBlock from '$lib/components/ui/CodeBlock.svelte';
	import Dialog from '$lib/components/ui/Dialog.svelte';

	type SSHKeyResponse = components['schemas']['SSHKeyResponse'];
	type ServerResponse = components['schemas']['ServerResponse'];

	const CREATE_NEW = '__create_new__';

	type Props = {
		onsuccess?: (server: ServerResponse) => void;
	};

	let { onsuccess }: Props = $props();

	const api = createApiClient();

	let keys = $state<SSHKeyResponse[]>([]);
	let keysLoading = $state(true);
	let keysError = $state('');
	let sshKeyDialogOpen = $state(false);

	let name = $state('');
	let host = $state('');
	let port = $state(22);
	let sshUser = $state('root');
	let sshKeyId = $state('');
	let selectValue = $state('');
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
		host.trim() !== '' && sshUser.trim() !== '' && sshKeyId !== '' && !checking && !keysError
	);

	let keyItems = $derived([
		{ value: CREATE_NEW, label: 'Create new SSH key' },
		...keys.map((k) => ({ value: k.id, label: k.name }))
	]);

	function handleKeySelectChange(value: string) {
		if (value === CREATE_NEW) {
			sshKeyDialogOpen = true;
			selectValue = sshKeyId;
			return;
		}
		selectValue = value;
		sshKeyId = value;
	}

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

	async function handleKeyCreated(key: SSHKeyResponse) {
		sshKeyDialogOpen = false;
		await loadKeys();
		sshKeyId = key.id;
		selectValue = key.id;
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
			selectValue = '';
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
			<Select
				items={keyItems}
				bind:value={selectValue}
				onValueChange={handleKeySelectChange}
				disabled={!!keysError}
				placeholder="Select an SSH key"
			/>
			{#if keysError}
				<div class="mt-1.5 flex items-center gap-2">
					<p class="text-sm text-danger">{keysError}</p>
					<button
						type="button"
						class="cursor-pointer text-sm text-foreground underline hover:no-underline"
						onclick={loadKeys}
					>
						Retry
					</button>
				</div>
			{/if}
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
			<Button type="submit" {loading} disabled={!isVerified || !!keysError}>
				{loading ? 'Saving...' : 'Add Server'}
			</Button>
		</div>
	</form>
{/if}

<Dialog bind:open={sshKeyDialogOpen} title="Create SSH key">
	<SSHKeyCreatePanel onsuccess={handleKeyCreated} />
</Dialog>
