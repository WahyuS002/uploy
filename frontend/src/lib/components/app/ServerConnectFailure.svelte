<script lang="ts">
	import { Icon } from '@steeze-ui/svelte-icon';
	import { ExclamationTriangle, ArrowLeft } from '@steeze-ui/heroicons';
	import type { ServerCreateController } from './server-create-form.svelte';

	let { controller }: { controller: ServerCreateController } = $props();

	let failure = $derived(controller.failure);
	let target = $derived(`${controller.host}:${controller.port}`);
	let account = $derived(`${controller.sshUser}@${controller.host}`);
	let keyName = $derived(controller.selectedKey?.name ?? 'this key');

	// One stage, one remedy. The three SSH failures need entirely different
	// fixes, so collapsing them into a single red line makes the user guess.
	let copy = $derived.by(() => {
		switch (failure?.code) {
			case 'unreachable':
				return {
					title: `Nothing answered at ${target}`,
					body: 'Check the address and port, and make sure a firewall or security group allows SSH from Uploy.'
				};
			case 'key_rejected':
				return {
					title: 'SSH Key Not Authorized',
					body: `${controller.host} is reachable, but ${account} hasn't authorized ${keyName} yet. Run the command above on your server to add the key, then try again.`
				};
			case 'key_invalid':
				return {
					title: `${keyName} can't be used`,
					body: 'Uploy could not read this key. Generate a new one and authorize that instead.'
				};
			case 'session_failed':
				return {
					title: 'Signed in, but commands will not run',
					body: `SSH accepted the key, but ${controller.sshUser} could not run a command. Check the account's shell and any forced-command restriction in authorized_keys.`
				};
			case 'docker_missing':
				return {
					title: 'Docker is not reachable here',
					body: `Uploy deploys with Docker. Install it on ${controller.host}, and make sure ${controller.sshUser} can run it without a password prompt.`
				};
			default:
				return { title: 'Could not connect', body: '' };
		}
	});
</script>

{#if failure}
	<div
		role="alert"
		class="flex flex-col gap-2 rounded-md border border-destructive/25 bg-destructive/5 p-3"
	>
		<p class="flex items-start gap-2 text-sm font-medium text-destructive">
			<Icon src={ExclamationTriangle} theme="outline" class="mt-px h-4 w-4 flex-none" />
			{copy.title}
		</p>

		{#if copy.body}
			<p class="text-xs leading-relaxed text-foreground">{copy.body}</p>
		{/if}

		<!-- The raw server message stays available: whoever is debugging SSH wants
		     the actual error, not only our paraphrase of it. -->
		<p class="font-mono text-[11px] leading-relaxed break-words text-muted-foreground">
			{failure.message}
		</p>

		{#if failure.belongsToTarget && controller.step === 'authorize'}
			<button
				type="button"
				onclick={controller.back}
				class="mt-0.5 inline-flex w-fit cursor-pointer items-center gap-1.5 rounded text-xs font-medium text-foreground underline decoration-border underline-offset-2 transition-colors hover:decoration-foreground focus-visible:ring-2 focus-visible:ring-ring/40 focus-visible:outline-none"
			>
				<Icon src={ArrowLeft} theme="outline" class="h-3 w-3" />
				Fix the address
			</button>
		{/if}
	</div>
{/if}
