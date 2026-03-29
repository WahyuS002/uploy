<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { createApiClient } from '$lib/api/client';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	let isOwner = $derived(data.workspace?.role === 'owner');
	let keys = $derived(data.keys);

	let view = $state<'default' | 'import'>('default');
	let error = $state('');
	let loading = $state(false);
	let lastCreatedKey = $state<{ name: string; public_key: string } | null>(null);
	let copiedId = $state<string | null>(null);
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

	async function copyText(text: string, id: string) {
		await navigator.clipboard.writeText(text);
		copiedId = id;
		setTimeout(() => (copiedId = null), 2000);
	}
</script>

<section>
	<h2 class="mb-4 text-xl font-bold">SSH Keys</h2>

	{#if isOwner}
		{#if lastCreatedKey}
			<div class="mb-8 max-w-md rounded-lg border border-green-200 bg-green-50 p-5">
				<p class="mb-1 text-sm font-semibold text-green-800">Key created</p>
				<p class="mb-3 text-sm text-green-700">
					<span class="font-medium">{lastCreatedKey.name}</span> is ready. Add this public key to
					<code class="rounded bg-green-100 px-1 text-xs">~/.ssh/authorized_keys</code> on your remote
					server.
				</p>
				<div class="mb-3 flex items-start gap-2">
					<pre
						class="flex-1 overflow-x-auto rounded bg-white p-2 font-mono text-xs break-all whitespace-pre-wrap">{lastCreatedKey.public_key}</pre>
					<button
						type="button"
						class="shrink-0 cursor-pointer rounded border bg-white px-2 py-1 text-xs hover:bg-gray-50"
						onclick={() => copyText(lastCreatedKey!.public_key, 'created')}
					>
						{copiedId === 'created' ? 'Copied!' : 'Copy public key'}
					</button>
				</div>
				<div class="flex items-center gap-3">
					<a
						href="/dashboard/servers"
						class="rounded-sm bg-black px-3 py-1.5 text-sm text-white hover:bg-gray-800"
					>
						Go to Servers
					</a>
					<button
						type="button"
						class="cursor-pointer text-sm text-gray-500 hover:text-gray-700"
						onclick={() => (lastCreatedKey = null)}
					>
						Dismiss
					</button>
				</div>
			</div>
		{:else if view === 'import'}
			<div class="mb-8 max-w-md">
				<form
					onsubmit={(e) => {
						e.preventDefault();
						importKey();
					}}
					class="flex flex-col gap-3"
				>
					<label class="flex flex-col gap-1 text-sm">
						Name
						<input
							type="text"
							bind:value={importName}
							required
							class="rounded border p-2"
							placeholder="production-server"
						/>
					</label>
					<label class="flex flex-col gap-1 text-sm">
						Private Key
						<textarea
							bind:value={privateKey}
							required
							rows="6"
							class="rounded border p-2 font-mono text-xs"
							placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
						></textarea>
					</label>

					{#if error}
						<p class="text-sm text-red-600">{error}</p>
					{/if}

					<button
						type="submit"
						disabled={loading}
						class="cursor-pointer rounded-sm bg-black p-2 text-white disabled:opacity-50"
					>
						{loading ? 'Saving...' : 'Add SSH Key'}
					</button>
				</form>
				<button
					type="button"
					class="mt-3 cursor-pointer text-sm text-gray-500 hover:text-gray-700"
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
				<p class="mb-4 text-sm text-gray-600">
					Generate an SSH key to connect Uploy to your servers. You'll add the public key to your
					server after generating.
				</p>

				{#if error}
					<p class="mb-3 text-sm text-red-600">{error}</p>
				{/if}

				<button
					type="button"
					disabled={loading}
					class="mb-3 w-full cursor-pointer rounded-sm bg-black p-2.5 text-white disabled:opacity-50"
					onclick={quickGenerate}
				>
					{loading ? 'Generating...' : 'Generate Ed25519 Key'}
				</button>
				<button
					type="button"
					class="cursor-pointer text-sm text-gray-500 hover:text-gray-700"
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
				<h3 class="mb-3 text-sm font-semibold text-gray-700">Your keys</h3>
				<ul class="divide-y">
					{#each keys as key (key.id)}
						<li class="flex items-center justify-between py-2.5">
							<div>
								<p class="text-sm">{key.name}</p>
								<p class="text-xs text-gray-400">
									{new Date(key.created_at).toLocaleDateString()}
								</p>
							</div>
							<div class="flex items-center gap-2">
								{#if key.public_key}
									<button
										type="button"
										class="cursor-pointer rounded border px-2 py-1 text-xs hover:bg-gray-50"
										onclick={() => (expandedKeyId = expandedKeyId === key.id ? null : key.id)}
									>
										{expandedKeyId === key.id ? 'Hide' : 'Show'} public key
									</button>
									<button
										type="button"
										class="cursor-pointer rounded border px-2 py-1 text-xs hover:bg-gray-50"
										onclick={() => copyText(key.public_key, key.id)}
									>
										{copiedId === key.id ? 'Copied!' : 'Copy public key'}
									</button>
								{/if}
							</div>
						</li>
						{#if expandedKeyId === key.id && key.public_key}
							<li class="pb-3">
								<pre
									class="overflow-x-auto rounded bg-gray-50 p-2 font-mono text-xs break-all whitespace-pre-wrap">{key.public_key}</pre>
							</li>
						{/if}
					{/each}
				</ul>
			</div>
		{/if}
	{:else}
		<p class="max-w-md text-sm text-gray-500">
			Only workspace owners can manage SSH keys. Contact your workspace owner to add or generate
			keys.
		</p>
	{/if}
</section>
