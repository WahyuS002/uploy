<script lang="ts">
	import CopyButton from './CopyButton.svelte';
	import CodeBlock from '$lib/components/ui/CodeBlock.svelte';
	import { cn } from '$lib/components/ui/cn.js';

	type Props = {
		publicKey: string;
		destinationPath?: string;
		description?: string;
		copyLabel?: string;
		announce?: boolean;
		class?: string;
	};

	let {
		publicKey,
		destinationPath = '~/.ssh/authorized_keys',
		description,
		copyLabel = 'Copy key',
		announce = false,
		class: className
	}: Props = $props();
</script>

<div
	class={cn(
		'overflow-hidden rounded-lg border border-border bg-card text-card-foreground',
		className
	)}
>
	{#if announce}
		<span class="sr-only" role="status" aria-live="polite">Public key ready to copy.</span>
	{/if}
	<div class="flex items-start justify-between gap-3 px-3.5 py-2.5">
		<div class="min-w-0">
			<p class="text-sm font-medium text-foreground">Public key</p>
			<p class="mt-0.5 text-xs text-muted-foreground">
				{#if description}
					{description}
				{:else}
					Add to
					<code class="rounded bg-muted px-1 py-0.5 font-mono text-[11px]">{destinationPath}</code>
					on your remote server.
				{/if}
			</p>
		</div>
		<CopyButton text={publicKey} defaultLabel={copyLabel} />
	</div>
	<div class="border-t border-border">
		<CodeBlock code={publicKey} class="rounded-none" />
	</div>
</div>
