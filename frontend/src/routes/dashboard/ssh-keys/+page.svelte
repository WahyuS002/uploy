<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import type { components } from '$lib/api/v1';
	import type { PageData } from './$types';
	import PageHeader from '$lib/components/app/PageHeader.svelte';
	import CopyButton from '$lib/components/app/CopyButton.svelte';
	import SSHKeyCreatePanel from '$lib/components/app/SSHKeyCreatePanel.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Alert from '$lib/components/ui/Alert.svelte';
	import CodeBlock from '$lib/components/ui/CodeBlock.svelte';

	type SSHKeyResponse = components['schemas']['SSHKeyResponse'];

	let { data }: { data: PageData } = $props();

	let isOwner = $derived(data.workspace?.role === 'owner');
	let keys = $derived(data.keys);

	let lastCreatedKey = $state<{ name: string; public_key: string } | null>(null);
	let expandedKeyId = $state<string | null>(null);

	async function handleKeyCreated(key: SSHKeyResponse) {
		lastCreatedKey = { name: key.name, public_key: key.public_key };
		await invalidateAll();
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
		{:else}
			<div class="mb-8 max-w-md">
				<SSHKeyCreatePanel onsuccess={handleKeyCreated} />
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
