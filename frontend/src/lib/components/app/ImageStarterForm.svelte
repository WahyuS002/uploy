<script lang="ts" module>
	// The port an image listens on is a property of the image, not a decision the
	// user should have to make: nobody types `postgres:16` meaning 8080. Well-known
	// images answer for themselves; everything else falls back to 8080.
	const wellKnownPorts: Record<string, number> = {
		postgres: 5432,
		mysql: 3306,
		mariadb: 3306,
		redis: 6379,
		mongo: 27017,
		nginx: 80,
		caddy: 80,
		httpd: 80,
		traefik: 80
	};

	// 80 and 443 belong to the Uploy proxy, so an image that listens on either
	// has to be published somewhere else. Everything else is reachable on the
	// number it already answers to, which is what a database wants.
	export function defaultHostPort(containerPort: number) {
		return containerPort === 80 || containerPort === 443 ? 8080 : containerPort;
	}

	// `ghcr.io/owner/repo:tag` -> `repo`, `postgres:16` -> `postgres`. Splitting on
	// `/` before `:` is what keeps a registry port (`localhost:5000/nginx`) from
	// being mistaken for the tag.
	export function detectPort(value: string) {
		const name = value.trim().split('@')[0].split('/').pop()?.split(':')[0]?.toLowerCase() ?? '';
		return wellKnownPorts[name] ?? 8080;
	}
</script>

<script lang="ts">
	import { onMount, type Snippet } from 'svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { ArrowLeft } from '@steeze-ui/heroicons';

	type Props = {
		submitting?: boolean;
		/** Failure from the caller's request. Local validation has its own. */
		error?: string;
		submitLabel?: string;
		backLabel?: string;
		/** Rows shown under the port row — where the deploy target is spelled out. */
		details?: Snippet;
		onBack: () => void;
		onSubmit: (image: string, port: number, hostPort: number) => void;
	};

	let {
		submitting = false,
		error = '',
		submitLabel = 'Create project',
		backLabel = 'Back to starters',
		details,
		onBack,
		onSubmit
	}: Props = $props();

	// Empty on landing, not prefilled: the field is the first thing you type into,
	// and backspace-on-empty only works as "go back" if it starts empty.
	let image = $state('');
	let localError = $state('');

	let started = $derived(image.trim() !== '');

	const examples = [
		'nginx:latest',
		'redis:7-alpine',
		'postgres:16',
		'caddy:2',
		'ghcr.io/owner/repo:tag'
	];

	let port = $state(8080);

	// The port trails the image right up until it is edited, then it stops moving
	// — an auto-filled value that overwrites a deliberate one is worse than no
	// auto-fill at all.
	let portTouched = $state(false);
	$effect(() => {
		const detected = detectPort(image);
		if (!portTouched) port = detected;
	});

	// Where the service is reachable from outside. It trails the listening port
	// under the same rule, except when that port is one the proxy has claimed —
	// then it has to differ, and 8080 is the suggestion.
	let hostPort = $state(8080);
	let hostPortTouched = $state(false);
	$effect(() => {
		const suggested = defaultHostPort(port);
		if (!hostPortTouched) hostPort = suggested;
	});

	function pickExample(example: string) {
		image = example;
		portTouched = false;
		hostPortTouched = false;
	}

	// The command-palette idiom: the arrow lives in the field, and the keyboard
	// gets out the same way it came in. Backspace only leaves when there is
	// nothing left to delete, so it never eats a character.
	function onImageKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			event.preventDefault();
			onBack();
			return;
		}
		if (event.key === 'Backspace' && image === '') {
			event.preventDefault();
			onBack();
		}
	}

	let imageInput = $state<HTMLInputElement | null>(null);
	onMount(() => imageInput?.focus());

	function submit(event: SubmitEvent) {
		event.preventDefault();
		if (submitting) return;

		const trimmedImage = image.trim();
		if (!trimmedImage) {
			localError = 'Image is required';
			return;
		}
		if (!Number.isFinite(port) || port < 1 || port > 65535) {
			localError = 'Port must be between 1 and 65535';
			return;
		}
		if (!Number.isFinite(hostPort) || hostPort < 1 || hostPort > 65535) {
			localError = 'Reachable-on port must be between 1 and 65535';
			return;
		}
		if (hostPort === 80 || hostPort === 443) {
			localError = 'Ports 80 and 443 are reserved for the Uploy proxy';
			return;
		}

		localError = '';
		onSubmit(trimmedImage, port, hostPort);
	}
</script>

<form
	onsubmit={submit}
	class="card overflow-hidden rounded-xl border border-border bg-card text-card-foreground"
>
	<div class="relative p-2">
		<button
			type="button"
			onclick={onBack}
			aria-label={backLabel}
			class="absolute top-1/2 left-4 z-10 grid h-5 w-5 -translate-y-1/2 cursor-pointer place-content-center rounded text-muted-foreground transition-colors hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
		>
			<Icon src={ArrowLeft} theme="outline" class="h-3.5 w-3.5" />
		</button>
		<!-- No border/shadow overrides here: Input already carries
		     field-focus-glow, which is this design system's answer to the
		     brand-coloured focus ring — --mint-deep border plus a soft
		     brand halo. Suppressing it was the reason focus read as dead. -->
		<Input
			type="text"
			bind:value={image}
			onkeydown={onImageKeydown}
			bind:ref={imageInput}
			placeholder="nginx:latest"
			aria-label="Docker image"
			autocomplete="off"
			autocapitalize="off"
			spellcheck={false}
			required
			class="pl-9"
		/>
	</div>

	{#if !started}
		<p class="border-t border-border/70 px-4 py-2.5 text-xs text-muted-foreground">
			Supports any public registry (Docker Hub, GHCR, GCR, Quay).
		</p>

		<div class="border-t border-border/70 py-1">
			<p class="px-4 py-1.5 text-xs text-muted-foreground">Examples</p>
			{#each examples as example (example)}
				<button
					type="button"
					onclick={() => pickExample(example)}
					class="flex w-full cursor-pointer items-center justify-between gap-3 px-4 py-1.5 text-left transition-colors hover:bg-accent focus-visible:bg-accent focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none focus-visible:ring-inset"
				>
					<span class="truncate font-mono text-xs text-foreground">{example}</span>
					<span class="font-mono text-[11px] text-muted-foreground">:{detectPort(example)}</span>
				</button>
			{/each}
		</div>
	{:else}
		<div class="flex flex-col gap-4 border-t border-border/70 p-4">
			<!-- Only the port is a control, so only the port gets the recessed
			     field surface. The previous pass gave both rows the same bordered
			     box and the input stopped reading as editable. The target is a
			     plain line: nothing to click, so nothing that looks clickable. -->
			<div class="flex flex-col gap-2.5">
				<label class="flex items-center justify-between gap-3">
					<span class="text-xs text-muted-foreground">Listens on port</span>
					<Input
						type="number"
						size="sm"
						bind:value={port}
						oninput={() => (portTouched = true)}
						min={1}
						max={65535}
						required
						class="w-24"
					/>
				</label>

				<label class="flex items-center justify-between gap-3">
					<span class="text-xs text-muted-foreground">Reachable on port</span>
					<Input
						type="number"
						size="sm"
						bind:value={hostPort}
						oninput={() => (hostPortTouched = true)}
						min={1}
						max={65535}
						required
						class="w-24"
					/>
				</label>

				{@render details?.()}
			</div>

			{#if localError || error}
				<p class="text-sm text-destructive">{localError || error}</p>
			{/if}

			<Button type="submit" size="sm" loading={submitting} disabled={submitting}>
				{submitting ? 'Creating...' : submitLabel}
			</Button>
		</div>
	{/if}
</form>

<style>
	.card {
		box-shadow: var(--shadow-panel);
	}
</style>
