<script lang="ts">
	import SSHKeyCreatePanel from './SSHKeyCreatePanel.svelte';
	import RadioList from '$lib/components/ui/RadioList.svelte';
	import { Dialog, DialogContent, DialogHeader, DialogTitle } from '$lib/components/ui/dialog';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { Plus } from '@steeze-ui/heroicons';
	import type { ServerCreateController } from './server-create-form.svelte';

	let { controller }: { controller: ServerCreateController } = $props();

	// Was a hardcoded "ssh-key": two wizards mounted at once would have shared one
	// radio group and fought over the checked state.
	const groupName = $props.id();
</script>

<fieldset class="flex min-w-0 flex-col gap-1.5">
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
				<RadioList
					items={controller.keys}
					value={controller.sshKeyId}
					name={groupName}
					ariaLabel="SSH key"
					onChange={(id) => (controller.sshKeyId = id)}
					class="max-h-52"
				>
					{#snippet children(key)}
						<span class="min-w-0 flex-1 truncate text-sm text-foreground" title={key.name}>
							{key.name}
						</span>
					{/snippet}
				</RadioList>
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
