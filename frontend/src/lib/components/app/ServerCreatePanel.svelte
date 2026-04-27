<script lang="ts">
	import type { Snippet } from 'svelte';
	import ServerCreateFields from './ServerCreateFields.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import type { ServerCreateController } from './server-create-form.svelte';

	type Props = {
		controller: ServerCreateController;
		submitLabel?: string;
		actionsClass?: string;
		actionsLeading?: Snippet;
	};

	let { controller, submitLabel = 'Add Server', actionsClass, actionsLeading }: Props = $props();
</script>

<form
	onsubmit={(e) => {
		e.preventDefault();
		controller.createServer();
	}}
	class="flex flex-col gap-3"
>
	<ServerCreateFields {controller} />
	<div class="flex items-center gap-2 {actionsClass ?? ''}">
		{#if actionsLeading}
			{@render actionsLeading()}
		{/if}
		<Button
			type="button"
			variant="secondary"
			disabled={!controller.canCheckConnection}
			onclick={controller.checkConnection}
		>
			{#if controller.checking}
				Checking...
			{:else if controller.isVerified}
				Connected
			{:else}
				Check Connection
			{/if}
		</Button>
		<Button
			type="submit"
			loading={controller.loading}
			disabled={!controller.isVerified || !!controller.keysError}
		>
			{controller.loading ? 'Saving...' : submitLabel}
		</Button>
	</div>
</form>
