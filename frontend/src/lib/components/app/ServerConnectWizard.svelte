<script lang="ts">
	import type { Snippet } from 'svelte';
	import ServerCreateFields from './ServerCreateFields.svelte';
	import SSHKeyField from './SSHKeyField.svelte';
	import ServerConnectFailure from './ServerConnectFailure.svelte';
	import CopyButton from './CopyButton.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import { cn } from '$lib/components/ui/cn.js';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { ArrowLeft, Check } from '@steeze-ui/heroicons';
	import type { ServerCreateController } from './server-create-form.svelte';

	type Props = {
		controller: ServerCreateController;
		submitLabel?: string;
		class?: string;
		bodyClass?: string;
		actionsClass?: string;
		/** Rendered at the start of the footer on the first step only, so a host
		 *  flow can offer its own way back out of the wizard. */
		actionsLeading?: Snippet;
	};

	let {
		controller,
		submitLabel = 'Connect and add server',
		class: className,
		bodyClass,
		actionsClass,
		actionsLeading
	}: Props = $props();

	const steps = [
		{ id: 'target', label: 'Target' },
		{ id: 'authorize', label: 'Authorize' }
	] as const;

	let stepIndex = $derived(steps.findIndex((s) => s.id === controller.step));
</script>

<form
	onsubmit={(e) => {
		e.preventDefault();
		if (controller.step === 'target') controller.advance();
		else controller.createServer();
	}}
	class={cn('flex min-h-0 flex-col', className)}
>
	<div class={cn('flex min-h-0 flex-col gap-4', bodyClass)}>
		<!-- Three states, three treatments. A marker that renders "done" and
		     "current" the same way stops answering the only question it exists to
		     answer. Each step owns a full-width rail rather than a stub connector,
		     so the row reads as progress across the dialog instead of two badges
		     floating in the corner. Done is tint-and-text (the tone every other
		     success surface here uses); current is the solid obsidian mark. -->
		<ol class="flex items-stretch gap-3 text-xs" aria-label="Progress">
			{#each steps as step, i (step.id)}
				{@const state = i === stepIndex ? 'current' : i < stepIndex ? 'done' : 'todo'}
				<li
					class="flex min-w-0 flex-1 flex-col gap-2"
					aria-current={state === 'current' ? 'step' : undefined}
				>
					<span class="flex items-center gap-2">
						<span
							class="grid h-5 w-5 flex-none place-content-center rounded-full text-[11px] font-medium tabular-nums transition-colors duration-150 {state ===
							'done'
								? 'bg-success-muted text-success'
								: state === 'current'
									? 'bg-foreground text-background'
									: 'border border-input text-muted-foreground'}"
						>
							{#if state === 'done'}
								<Icon src={Check} theme="outline" class="h-3 w-3" />
							{:else}
								{i + 1}
							{/if}
						</span>
						<span
							class="min-w-0 truncate {state === 'current'
								? 'font-medium text-foreground'
								: state === 'done'
									? 'text-foreground'
									: 'text-muted-foreground'}"
						>
							{step.label}
						</span>
						<!-- Colour alone must not carry "done": name the state for AT too. -->
						<span class="sr-only">
							{state === 'done'
								? 'completed'
								: state === 'current'
									? 'current step'
									: 'not started'}
						</span>
					</span>
					<span
						aria-hidden="true"
						class="h-0.5 rounded-full transition-colors duration-150 {state === 'done'
							? 'bg-success'
							: state === 'current'
								? 'bg-foreground'
								: 'bg-border'}"
					></span>
				</li>
			{/each}
		</ol>

		{#if controller.step === 'target'}
			<div class="flex flex-col gap-4">
				<ServerCreateFields {controller} />
				<SSHKeyField {controller} />
			</div>
		{:else}
			<div class="flex flex-col gap-3">
				<div>
					<h3 class="text-sm font-medium text-foreground">
						Authorize SSH key on {controller.host}
					</h3>
					{#if controller.authorizeCommand}
						<p class="mt-1 text-xs leading-relaxed text-muted-foreground">
							Run this command on the server as <span class="font-medium text-foreground"
								>{controller.sshUser}</span
							>:
						</p>
					{:else}
						<p class="mt-1 text-xs leading-relaxed text-muted-foreground">
							{controller.selectedKey?.name ?? 'This key'} has no public key stored. Authorize it on the
							server, then continue.
						</p>
					{/if}
				</div>

				{#if controller.authorizeCommand}
					<div class="flex flex-col gap-2">
						<pre
							class="overflow-x-auto rounded-md border border-border bg-muted px-3 py-2.5 font-mono text-[11px] leading-relaxed text-foreground">{controller.authorizeCommand}</pre>
						<CopyButton
							text={controller.authorizeCommand}
							defaultLabel="Copy command"
							copiedLabel="Copied"
							class="w-fit"
						/>
					</div>
				{/if}

				<p class="text-xs text-muted-foreground">
					Already authorized this key? Click Connect to continue.
				</p>

				<ServerConnectFailure {controller} />
			</div>
		{/if}
	</div>

	<div class={cn('flex items-center gap-2', actionsClass)}>
		{#if controller.step === 'target'}
			{#if actionsLeading}
				{@render actionsLeading()}
			{/if}
			<span class="flex-1"></span>
			<Button type="submit" disabled={!controller.canAdvance}>Continue</Button>
		{:else}
			<Button type="button" variant="ghost" size="sm" onclick={controller.back}>
				<Icon src={ArrowLeft} theme="outline" class="h-3.5 w-3.5" />
				Back
			</Button>
			<span class="flex-1"></span>
			<Button type="submit" loading={controller.loading}>
				{controller.loading ? 'Connecting...' : submitLabel}
			</Button>
		{/if}
	</div>
</form>
