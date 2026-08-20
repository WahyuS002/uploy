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
		submitLabel = 'Connect server',
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
							? 'bg-success-fill'
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
						Authorize on {controller.host}
					</h3>
					{#if controller.authorizeCommand}
						<p class="mt-0.5 text-xs text-muted-foreground">
							Run as <span class="font-medium text-foreground">{controller.sshUser}</span>:
						</p>
					{:else}
						<p class="mt-0.5 text-xs text-muted-foreground">
							{controller.selectedKey?.name ?? 'This key'} has no public key stored.
						</p>
					{/if}
				</div>

				{#if controller.authorizeCommand}
					<!-- One object rather than a block with a button parked under it. The
					     command is something you copy, so the control that copies it sits
					     on the same row and the whole step collapses to a single strip.

					     It scrolls instead of wrapping: the key alone is 380-odd
					     characters, and wrapped it would be the tallest thing in the
					     dialog for a string nobody reads. -->
					<div class="flex items-center gap-1 rounded-md border border-border bg-muted p-1">
						<div class="relative min-w-0 flex-1">
							<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
							<!-- The rule is about not putting <div>s in the tab order. This is the
							     opposite case: a region that clips its content has to be reachable
							     without a mouse, or the tail of the command is keyboard-only
							     unreachable. WCAG 2.1.1 wants the tab stop here. -->
							<pre
								tabindex="0"
								role="group"
								aria-label="Authorize command"
								class="command overflow-x-auto px-2 py-1.5 font-mono text-[11px] leading-relaxed text-foreground">{controller.authorizeCommand}</pre>
							<span
								aria-hidden="true"
								class="fade pointer-events-none absolute inset-y-0 right-0 w-8"
							></span>
						</div>
						<CopyButton
							icon
							text={controller.authorizeCommand}
							defaultLabel="Copy command"
							copiedLabel="Command copied"
							class="flex-none"
						/>
					</div>
				{/if}

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

<style>
	/* The scrollbar is the loudest thing in a 40px strip, and it says nothing the
	   fade at the end does not already say. Dragging, shift-scroll and the arrow
	   keys all still move it; only the chrome is gone. */
	.command {
		scrollbar-width: none;
	}

	.command::-webkit-scrollbar {
		display: none;
	}

	/* A scroll region has to be reachable without a mouse, and a tab stop with no
	   visible focus is a trap. The outline sits inside so it does not overlap the
	   strip's own border. */
	.command:focus-visible {
		border-radius: var(--radius-sm);
		outline: 2px solid var(--ring);
		outline-offset: -2px;
	}

	.fade {
		background: linear-gradient(to right, transparent, var(--muted));
	}
</style>
