<script lang="ts">
	import Button from '$lib/components/ui/Button.svelte';
	import IconButton from '$lib/components/ui/IconButton.svelte';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { Check, ClipboardDocument } from '@steeze-ui/heroicons';

	type Props = {
		text: string;
		/** Icon-only, for when the control sits on the thing it copies and a worded
		 *  button would repeat what the row already says. */
		icon?: boolean;
		defaultLabel?: string;
		copiedLabel?: string;
		variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
		size?: 'sm' | 'md';
		class?: string;
	};

	let {
		text,
		icon = false,
		defaultLabel = 'Copy',
		copiedLabel = 'Copied!',
		variant = 'secondary',
		size = 'sm',
		class: className
	}: Props = $props();

	let copied = $state(false);
	let timer: ReturnType<typeof setTimeout> | undefined;

	// Uploy is self-hosted and plenty of installs answer on http://<lan-ip>.
	// navigator.clipboard does not exist off a secure origin, so the async API on
	// its own throws on exactly the setups the server wizard is written for. The
	// deprecated selection path is the only thing that works there.
	async function write(): Promise<boolean> {
		try {
			if (navigator.clipboard?.writeText) {
				await navigator.clipboard.writeText(text);
				return true;
			}
		} catch {
			// Permission denied or a document that is not focused. Try the old way.
		}
		try {
			const field = document.createElement('textarea');
			field.value = text;
			field.readOnly = true;
			field.style.position = 'fixed';
			field.style.opacity = '0';
			document.body.appendChild(field);
			field.select();
			const ok = document.execCommand('copy');
			field.remove();
			return ok;
		} catch {
			return false;
		}
	}

	async function copy() {
		// Never flip the label on a copy that did not happen. A control reading
		// Copied over an empty clipboard is worse than one that does nothing: the
		// command is still on screen and still selectable by hand.
		if (!(await write())) return;
		copied = true;
		clearTimeout(timer);
		timer = setTimeout(() => (copied = false), 2000);
	}

	// The dialog holding this can close inside the two seconds.
	$effect(() => () => clearTimeout(timer));
</script>

{#if icon}
	<IconButton
		variant="ghost"
		size="md"
		class={className}
		aria-label={copied ? copiedLabel : defaultLabel}
		onclick={copy}
	>
		<Icon
			src={copied ? Check : ClipboardDocument}
			theme="outline"
			class={copied ? 'h-4 w-4 text-success' : 'h-4 w-4'}
		/>
	</IconButton>
	<!-- Only the icon changes here, and a name swapped under a button that
	     already has focus is not reliably announced. Say it separately. -->
	<span class="sr-only" role="status" aria-live="polite">{copied ? copiedLabel : ''}</span>
{:else}
	<Button type="button" {variant} {size} class={className} onclick={copy}>
		{copied ? copiedLabel : defaultLabel}
	</Button>
{/if}
