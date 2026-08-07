<script lang="ts">
	import { Icon, type IconSource } from '@steeze-ui/svelte-icon';
	import { cn } from './cn.js';
	import type { Snippet } from 'svelte';

	type Variant = 'default' | 'canvas';

	type Props = {
		icons?: IconSource[];
		icon?: IconSource;
		title: string;
		description?: string;
		variant?: Variant;
		class?: string;
		actions?: Snippet;
	};

	let {
		icons,
		icon,
		title,
		description,
		variant = 'default',
		class: className,
		actions
	}: Props = $props();

	let displayIcon = $derived(icon ?? icons?.[0]);
</script>

{#if variant === 'canvas'}
	<div
		class={cn(
			'flex w-full flex-1 flex-col items-center justify-center px-6 pt-6 pb-20 text-center',
			className
		)}
	>
		<div class="relative mb-5 h-[112px] w-[172px]">
			<svg
				xmlns="http://www.w3.org/2000/svg"
				viewBox="0 0 215 140"
				preserveAspectRatio="xMidYMid meet"
				fill="none"
				aria-hidden="true"
				focusable="false"
				class="pointer-events-none absolute inset-0 h-full w-full text-border"
			>
				<path
					d="M64 0L64 140"
					stroke="currentColor"
					stroke-width="0.8"
					stroke-miterlimit="10"
					stroke-dasharray="3 3"
				/>
				<path
					d="M151 0L151 140"
					stroke="currentColor"
					stroke-width="0.8"
					stroke-miterlimit="10"
					stroke-dasharray="3 3"
				/>
				<path
					d="M215 24H0"
					stroke="currentColor"
					stroke-width="0.8"
					stroke-miterlimit="10"
					stroke-dasharray="3 3"
				/>
				<path
					d="M215 50H0"
					stroke="currentColor"
					stroke-width="0.8"
					stroke-miterlimit="10"
					stroke-dasharray="3 3"
				/>
				<path
					d="M215 88H0"
					stroke="currentColor"
					stroke-width="0.8"
					stroke-miterlimit="10"
					stroke-dasharray="3 3"
				/>
				<path
					d="M215 114H0"
					stroke="currentColor"
					stroke-width="0.8"
					stroke-miterlimit="10"
					stroke-dasharray="3 3"
				/>
				<path d="M199 0L199 140" stroke="currentColor" stroke-width="0.8" stroke-miterlimit="10" />
				<path d="M16 0L16 140" stroke="currentColor" stroke-width="0.8" stroke-miterlimit="10" />
				<path d="M0 16L215 16" stroke="currentColor" stroke-width="0.8" stroke-miterlimit="10" />
				<path d="M0 124L215 124" stroke="currentColor" stroke-width="0.8" stroke-miterlimit="10" />
			</svg>
			{#if displayIcon}
				<div class="absolute inset-0 grid place-items-center">
					<Icon src={displayIcon} theme="outline" class="h-10 w-10 text-muted-foreground" />
				</div>
			{/if}
		</div>
		<h3 class="text-lg font-semibold tracking-[-0.01em] text-foreground">{title}</h3>
		{#if description}
			<p class="mt-2 max-w-md text-sm leading-relaxed text-muted-foreground">{description}</p>
		{/if}
		{#if actions}
			<div class="mt-6 flex items-center gap-2">
				{@render actions()}
			</div>
		{/if}
	</div>
{:else}
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
{/if}
