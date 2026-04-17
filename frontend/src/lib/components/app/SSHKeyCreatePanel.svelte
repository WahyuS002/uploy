<script lang="ts">
	import { createApiClient } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import FormField from '$lib/components/app/FormField.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Textarea from '$lib/components/ui/Textarea.svelte';

	type SSHKeyResponse = components['schemas']['SSHKeyResponse'];

	type Props = {
		onsuccess?: (key: SSHKeyResponse) => void;
	};

	let { onsuccess }: Props = $props();

	const api = createApiClient();

	let view = $state<'default' | 'import'>('default');
	let error = $state('');
	let loading = $state(false);

	// Import form state
	let importName = $state('');
	let privateKey = $state('');

	function defaultKeyName(): string {
		const now = new Date();
		const pad = (n: number) => String(n).padStart(2, '0');
		return `Ed25519 Key ${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())}`;
	}

	async function quickGenerate() {
		error = '';
		loading = true;
		try {
			const { data: res, error: err } = await api.POST('/api/ssh-keys/generate', {
				body: { name: defaultKeyName() }
			});
			if (err) {
				error = (err as { error: string }).error;
				return;
			}
			if (res) onsuccess?.(res);
		} catch {
			error = 'Network error, please try again';
		} finally {
			loading = false;
		}
	}

	async function importKey() {
		error = '';
		loading = true;
		try {
			const { data: res, error: err } = await api.POST('/api/ssh-keys', {
				body: { name: importName, private_key: privateKey }
			});
			if (err) {
				error = (err as { error: string }).error;
				return;
			}
			importName = '';
			privateKey = '';
			view = 'default';
			if (res) onsuccess?.(res);
		} catch {
			error = 'Network error, please try again';
		} finally {
			loading = false;
		}
	}
</script>

{#if view === 'import'}
	<form
		onsubmit={(e) => {
			e.preventDefault();
			importKey();
		}}
		class="flex flex-col gap-3"
	>
		<FormField label="Name">
			<Input type="text" bind:value={importName} required placeholder="production-server" />
		</FormField>
		<FormField label="Private Key">
			<Textarea
				bind:value={privateKey}
				required
				rows={6}
				class="font-mono text-xs"
				placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
			/>
		</FormField>

		{#if error}
			<p class="text-sm text-danger">{error}</p>
		{/if}

		<Button type="submit" {loading} class="w-full">
			{loading ? 'Saving...' : 'Add SSH Key'}
		</Button>
	</form>
	<button
		type="button"
		class="mt-3 cursor-pointer text-sm text-muted-foreground hover:text-foreground"
		onclick={() => {
			view = 'default';
			error = '';
		}}
	>
		Back to quick setup
	</button>
{:else}
	<p class="mb-4 text-sm text-muted-foreground">
		Generate an SSH key to connect Uploy to your servers. You'll add the public key to your server
		after generating.
	</p>

	{#if error}
		<p class="mb-3 text-sm text-danger">{error}</p>
	{/if}

	<Button {loading} class="mb-3 w-full" onclick={quickGenerate}>
		{loading ? 'Generating...' : 'Generate Ed25519 Key'}
	</Button>
	<button
		type="button"
		class="cursor-pointer text-sm text-muted-foreground hover:text-foreground"
		onclick={() => {
			view = 'import';
			error = '';
		}}
	>
		I have my own private key
	</button>
{/if}
