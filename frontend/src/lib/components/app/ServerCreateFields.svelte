<script lang="ts">
	import FormField from './FormField.svelte';
	import PublicKeyHelper from './PublicKeyHelper.svelte';
	import SSHKeyCreatePanel from './SSHKeyCreatePanel.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import { Dialog, DialogContent, DialogHeader, DialogTitle } from '$lib/components/ui/dialog';
	import type { ServerCreateController } from './server-create-form.svelte';

	type Props = {
		controller: ServerCreateController;
	};

	let { controller }: Props = $props();

	$effect(() => {
		controller.loadKeys();
	});
</script>

{#if controller.keysLoading}
	<div class="flex flex-col gap-3">
		<div class="h-10 animate-pulse rounded-lg bg-muted"></div>
		<div class="h-10 animate-pulse rounded-lg bg-muted"></div>
	</div>
{:else}
	<div class="flex flex-col gap-3">
		<FormField label="Name">
			<Input type="text" bind:value={controller.name} required placeholder="production-server" />
		</FormField>
		<FormField label="Host">
			<Input type="text" bind:value={controller.host} required placeholder="192.168.1.100" />
		</FormField>
		<div class="grid grid-cols-2 gap-3">
			<FormField label="Port">
				<Input type="number" bind:value={controller.port} required min={1} max={65535} />
			</FormField>
			<FormField label="SSH User">
				<Input type="text" bind:value={controller.sshUser} required placeholder="root" />
			</FormField>
		</div>

		<FormField label="SSH Key">
			<Select
				items={controller.keyItems}
				bind:value={controller.selectValue}
				onValueChange={controller.handleKeySelectChange}
				disabled={!!controller.keysError}
				placeholder="Select an SSH key"
			/>
			{#if controller.keysError}
				<div class="mt-1.5 flex items-center gap-2">
					<p class="text-sm text-destructive">{controller.keysError}</p>
					<button
						type="button"
						class="cursor-pointer text-sm text-foreground underline hover:no-underline"
						onclick={() => controller.loadKeys()}
					>
						Retry
					</button>
				</div>
			{/if}
		</FormField>

		{#if controller.selectedKeyPublicKey}
			<PublicKeyHelper publicKey={controller.selectedKeyPublicKey} announce />
		{/if}

		{#if controller.error}
			<p class="text-sm text-destructive">{controller.error}</p>
		{/if}
	</div>
{/if}

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
