<script lang="ts">
	import type { Snippet } from 'svelte';
	import ServerCreateFields from './ServerCreateFields.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import { cn } from '$lib/components/ui/cn.js';
	import type { ServerCreateController } from './server-create-form.svelte';

	type Props = {
		controller: ServerCreateController;
		submitLabel?: string;
		class?: string;
		fieldsClass?: string;
		actionsClass?: string;
		actionsLeading?: Snippet;
		/** Right-hand rail. When given, the fields sit in a two-column layout and the
		 * rail owns the SSH key concern instead of the form body. */
		aside?: Snippet;
	};

	let {
		controller,
		submitLabel = 'Add Server',
		class: className,
		fieldsClass,
		actionsClass,
		actionsLeading,
		aside
	}: Props = $props();
</script>

<form
	onsubmit={(e) => {
		e.preventDefault();
		controller.createServer();
	}}
	class={cn('flex flex-col', aside ? '' : 'gap-3', className)}
>
	<div
		class={cn(
			aside
				? 'grid items-start gap-5 sm:grid-cols-[minmax(0,1fr)_minmax(0,16rem)]'
				: 'flex flex-col gap-3',
			fieldsClass
		)}
	>
		<ServerCreateFields {controller} showSshKey={!aside} />
		{#if aside}
			<div class="min-w-0 self-stretch">{@render aside()}</div>
		{/if}
	</div>
	<div class={cn('flex items-center gap-2', actionsClass)}>
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
				Check connection
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
