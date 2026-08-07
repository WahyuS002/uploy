<script lang="ts">
	import FormField from './FormField.svelte';
	import ServerConnectFailure from './ServerConnectFailure.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import type { ServerCreateController } from './server-create-form.svelte';

	let { controller }: { controller: ServerCreateController } = $props();

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

	<!-- An unreachable host is this step's problem, so its diagnosis is shown
	     here rather than back on the step that reported it. -->
	{#if controller.failure?.belongsToTarget}
		<ServerConnectFailure {controller} />
	{/if}
</div>
