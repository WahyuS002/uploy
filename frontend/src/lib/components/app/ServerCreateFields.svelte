<script lang="ts">
	import FormField from './FormField.svelte';
	import SSHKeyField from './SSHKeyField.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import type { ServerCreateController } from './server-create-form.svelte';

	type Props = {
		controller: ServerCreateController;
		/** Set false when the caller renders SSHKeyField itself (e.g. in a side rail). */
		showSshKey?: boolean;
	};

	let { controller, showSshKey = true }: Props = $props();

	$effect(() => {
		controller.loadKeys();
	});
</script>

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

	{#if showSshKey}
		<SSHKeyField {controller} />
	{/if}

	{#if controller.error}
		<p class="text-sm text-destructive">{controller.error}</p>
	{/if}
</div>
