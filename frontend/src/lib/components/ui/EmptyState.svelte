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

<div class={cn('flex flex-col items-center px-6 py-12 text-center', className)}>
	{#if displayIcon}
		<div
			class="mb-5 grid h-12 w-12 place-content-center rounded-xl bg-muted text-muted-foreground"
		>
			<Icon src={displayIcon} theme="outline" class="h-5 w-5" />
		</div>
	{/if}
	<h3 class="text-base font-semibold text-foreground">{title}</h3>
	{#if description}
		<p class="mt-1.5 max-w-md text-sm text-muted-foreground">{description}</p>
	{/if}
	{#if actions}
		<div class="mt-5 flex items-center gap-2">
			{@render actions()}
		</div>
	{/if}
</div>
