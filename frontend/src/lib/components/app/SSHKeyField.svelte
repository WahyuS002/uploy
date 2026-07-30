<script lang="ts">
	import CopyButton from './CopyButton.svelte';
	import SSHKeyCreatePanel from './SSHKeyCreatePanel.svelte';
	import { Dialog, DialogContent, DialogHeader, DialogTitle } from '$lib/components/ui/dialog';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { Plus } from '@steeze-ui/heroicons';
	import type { ServerCreateController } from './server-create-form.svelte';

	let { controller }: { controller: ServerCreateController } = $props();

	let selectedKey = $derived(controller.keys.find((k) => k.id === controller.sshKeyId));
</script>

<fieldset class="flex h-full min-w-0 flex-col gap-1.5">
	<legend class="mb-1.5 text-xs text-muted-foreground">SSH key</legend>

	{#if controller.keysError}
		<div class="rounded-lg border border-destructive/20 bg-destructive/5 px-3 py-2.5">
			<p class="text-xs text-destructive">{controller.keysError}</p>
			<button
				type="button"
				class="mt-1.5 cursor-pointer text-xs text-foreground underline hover:no-underline"
				onclick={() => controller.loadKeys()}
			>
				Retry
			</button>
		</div>
	{:else if controller.keysLoading && !controller.keysLoaded}
		<div class="h-10 animate-pulse rounded-lg bg-muted"></div>
	{:else}
		<div class="overflow-hidden rounded-lg border border-border bg-card">
			{#if controller.keys.length > 0}
				<div class="max-h-52 divide-y divide-border overflow-y-auto">
					{#each controller.keys as key (key.id)}
						<label
							title={key.name}
							class="flex cursor-pointer items-center gap-2.5 px-3 py-2.5 transition-colors hover:bg-accent has-checked:bg-accent"
						>
							<input
								type="radio"
								name="ssh-key"
								value={key.id}
								checked={controller.sshKeyId === key.id}
								onchange={() => (controller.sshKeyId = key.id)}
								class="form-radio h-3.5 w-3.5 flex-none border-input text-foreground focus:ring-2 focus:ring-ring/40 focus:ring-offset-0"
							/>
							<span class="min-w-0 flex-1 truncate text-sm text-foreground">{key.name}</span>
						</label>
					{/each}
				</div>
			{:else}
				<p class="px-3 py-2.5 text-xs text-muted-foreground">
					No key in this workspace yet. Uploy needs one to reach the server.
				</p>
			{/if}
			<button
				type="button"
				onclick={() => (controller.sshKeyDialogOpen = true)}
				class="flex w-full cursor-pointer items-center gap-2 border-t border-border px-3 py-2.5 text-left text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:ring-2 focus-visible:ring-ring/40 focus-visible:outline-none"
			>
				<Icon src={Plus} theme="outline" class="h-3.5 w-3.5 flex-none" />
				Generate a new key
			</button>
		</div>

		{#if selectedKey?.public_key}
			<div class="mt-1 flex flex-col gap-1.5">
				<CopyButton text={selectedKey.public_key} defaultLabel="Copy public key" class="w-full" />
				<p class="text-xs text-muted-foreground">
					Add it to ~/.ssh/authorized_keys on the server, then check the connection.
				</p>
			</div>
		{/if}
	{/if}
</fieldset>

<Dialog bind:open={controller.sshKeyDialogOpen}>
	<DialogContent>
		<DialogHeader>
			<DialogTitle>Create SSH key</DialogTitle>
		</DialogHeader>
		<div class="px-5 pb-5">
			<SSHKeyCreatePanel onsuccess={controller.handleKeyCreated} />
		</div>
	</DialogContent>
</Dialog>
