<script lang="ts">
	import FormField from './FormField.svelte';
	import CopyButton from './CopyButton.svelte';
	import SSHKeyCreatePanel from './SSHKeyCreatePanel.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import Alert from '$lib/components/ui/Alert.svelte';
	import CodeBlock from '$lib/components/ui/CodeBlock.svelte';
	import Dialog from '$lib/components/ui/Dialog.svelte';
	import DialogContent from '$lib/components/ui/DialogContent.svelte';
	import DialogHeader from '$lib/components/ui/DialogHeader.svelte';
	import DialogTitle from '$lib/components/ui/DialogTitle.svelte';
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
		<div class="h-10 animate-pulse rounded-lg bg-surface-muted"></div>
		<div class="h-10 animate-pulse rounded-lg bg-surface-muted"></div>
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
					<p class="text-sm text-danger">{controller.keysError}</p>
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
			<Alert tone="info">
				<p class="mb-1 text-xs font-medium">
					Public key (add to <code class="rounded bg-blue-100 px-1">~/.ssh/authorized_keys</code>
					on remote server):
				</p>
				<div class="flex items-start gap-2">
					<CodeBlock code={controller.selectedKeyPublicKey} class="flex-1 bg-white" />
					<CopyButton text={controller.selectedKeyPublicKey} defaultLabel="Copy" />
				</div>
			</Alert>
		{/if}

		{#if controller.error}
			<p class="text-sm text-danger">{controller.error}</p>
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
