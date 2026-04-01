<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { createApiClient } from '$lib/api/client';
	import type { PageData } from './$types';
	import PageHeader from '$lib/components/app/PageHeader.svelte';
	import FormField from '$lib/components/app/FormField.svelte';
	import CopyButton from '$lib/components/app/CopyButton.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Textarea from '$lib/components/ui/Textarea.svelte';
	import Alert from '$lib/components/ui/Alert.svelte';
	import CodeBlock from '$lib/components/ui/CodeBlock.svelte';

	let { data }: { data: PageData } = $props();

	let isOwner = $derived(data.workspace?.role === 'owner');
	let keys = $derived(data.keys);

	let view = $state<'default' | 'import'>('default');
	let error = $state('');
	let loading = $state(false);
	let lastCreatedKey = $state<{ name: string; public_key: string } | null>(null);
	let expandedKeyId = $state<string | null>(null);

	// Import form state
	let importName = $state('');
	let privateKey = $state('');

	const api = createApiClient();

	function defaultKeyName(): string {
		const now = new Date();
		const pad = (n: number) => String(n).padStart(2, '0');
		return `Ed25519 Key ${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())}`;
	}

	async function quickGenerate() {
		error = '';
		loading = true;
		lastCreatedKey = null;
		try {
			const { data: res, error: err } = await api.POST('/api/ssh-keys/generate', {
				body: { name: defaultKeyName() }
			});
			if (err) {
				error = (err as { error: string }).error;
				return;
			}
			lastCreatedKey = { name: res.name, public_key: res.public_key };
			await invalidateAll();
		} catch {
			error = 'Network error, please try again';
		} finally {
			loading = false;
		}
	}

	async function importKey() {
		error = '';
		loading = true;
		lastCreatedKey = null;
		try {
			const { data: res, error: err } = await api.POST('/api/ssh-keys', {
				body: { name: importName, private_key: privateKey }
			});
			if (err) {
				error = (err as { error: string }).error;
				return;
			}
			lastCreatedKey = { name: res.name, public_key: res.public_key };
			importName = '';
			privateKey = '';
			view = 'default';
			await invalidateAll();
		} catch {
			error = 'Network error, please try again';
		} finally {
			loading = false;
		}
	}
</script>

<section>
	<PageHeader title="SSH Keys" />

	{#if isOwner}
		{#if lastCreatedKey}
			<Alert tone="success" class="mb-8 max-w-md p-5">
				<p class="mb-1 text-sm font-semibold">Key created</p>
				<p class="mb-3 text-sm">
					<span class="font-medium">{lastCreatedKey.name}</span> is ready. Add this public key to
					<code class="rounded bg-green-100 px-1 text-xs">~/.ssh/authorized_keys</code> on your remote
					server.
				</p>
				<div class="mb-3 flex items-start gap-2">
					<CodeBlock code={lastCreatedKey.public_key} class="flex-1 bg-white" />
					<CopyButton text={lastCreatedKey.public_key} defaultLabel="Copy public key" />
				</div>
				<div class="flex items-center gap-3">
					<Button href="/dashboard/servers" size="sm">Go to Servers</Button>
					<button
						type="button"
						class="cursor-pointer text-sm text-muted-foreground hover:text-foreground"
						onclick={() => (lastCreatedKey = null)}
					>
						Dismiss
					</button>
				</div>
			</Alert>
		{:else if view === 'import'}
			<div class="mb-8 max-w-md">
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
			</div>
		{:else}
			<div class="mb-8 max-w-md">
				<p class="mb-4 text-sm text-muted-foreground">
					Generate an SSH key to connect Uploy to your servers. You'll add the public key to your
					server after generating.
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
			</div>
		{/if}

		{#if keys.length > 0}
			<div class="max-w-2xl">
				<h3 class="mb-3 text-sm font-semibold text-foreground">Your keys</h3>
				<ul class="divide-y divide-border">
					{#each keys as key (key.id)}
						<li class="flex items-center justify-between py-2.5">
							<div>
								<p class="text-sm text-foreground">{key.name}</p>
								<p class="text-xs text-muted-foreground">
									{new Date(key.created_at).toLocaleDateString()}
								</p>
							</div>
							<div class="flex items-center gap-2">
								{#if key.public_key}
									<Button
										variant="secondary"
										size="sm"
										onclick={() => (expandedKeyId = expandedKeyId === key.id ? null : key.id)}
									>
										{expandedKeyId === key.id ? 'Hide' : 'Show'} public key
									</Button>
									<CopyButton text={key.public_key} defaultLabel="Copy public key" />
								{/if}
							</div>
						</li>
						{#if expandedKeyId === key.id && key.public_key}
							<li class="pb-3">
								<CodeBlock code={key.public_key} />
							</li>
						{/if}
					{/each}
				</ul>
			</div>
		{/if}
	{:else}
		<p class="max-w-md text-sm text-muted-foreground">
			Only workspace owners can manage SSH keys. Contact your workspace owner to add or generate
			keys.
		</p>
	{/if}
</section>
