<script lang="ts">
	import { Icon, type IconSource } from '@steeze-ui/svelte-icon';
	import { cn } from './cn.js';
	import type { Snippet } from 'svelte';

	type Props = {
		icons?: IconSource[];
		icon?: IconSource;
		title: string;
		description?: string;
		class?: string;
		actions?: Snippet;
	};

	let { icons, icon, title, description, class: className, actions }: Props = $props();
	let displayIcon = $derived(icon ?? icons?.[0]);
</script>

<div class={cn('flex flex-col items-center px-6 py-16 text-center', className)}>
	{#if displayIcon}
		<div
			class="mb-6 grid h-14 w-14 place-content-center rounded-2xl border border-border bg-surface"
		>
			<Icon src={displayIcon} theme="outline" class="h-6 w-6 text-foreground" />
		</div>
	{/if}
	<h3 class="text-lg font-semibold text-foreground">{title}</h3>
	{#if description}
		<p class="mt-2 max-w-md text-sm text-muted-foreground">{description}</p>
	{/if}
	{#if actions}
		<div class="mt-6 flex items-center gap-3">
			{@render actions()}
		</div>
	{/if}
</div>
