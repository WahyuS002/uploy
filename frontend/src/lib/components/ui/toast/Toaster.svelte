<script lang="ts">
	import { onDestroy } from 'svelte';
	import { fly, fade } from 'svelte/transition';
	import { browser } from '$app/environment';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { XMark, CheckCircle, ExclamationCircle } from '@steeze-ui/heroicons';
	import Spinner from '../Spinner.svelte';
	import { toastStore, TOAST_QUEUE_LIMIT, type Toast } from './toast-service.svelte.js';

	const COLLAPSED_PEEK = 10;
	const EXPANDED_GAP = 8;

	let expanded = $state(false);
	let heights = $state<Record<string, number>>({});
	let region = $state<HTMLDivElement | null>(null);

	type TimerEntry = {
		timeoutId: ReturnType<typeof setTimeout> | null;
		remaining: number;
		startedAt: number;
		createdAt: number;
	};
	const timers: Record<string, TimerEntry> = {};

	let visible = $derived(toastStore.items.slice().reverse().slice(0, TOAST_QUEUE_LIMIT));

	function clearTimer(id: string) {
		const entry = timers[id];
		if (entry?.timeoutId) clearTimeout(entry.timeoutId);
		delete timers[id];
	}

	function scheduleDismiss(id: string, remaining: number) {
		const entry = timers[id];
		if (!entry) return;
		entry.startedAt = Date.now();
		entry.timeoutId = setTimeout(() => toastStore.dismiss(id), remaining);
	}

	function startTimer(id: string, duration: number, createdAt: number) {
		clearTimer(id);
		timers[id] = { timeoutId: null, remaining: duration, startedAt: Date.now(), createdAt };
		if (!expanded) scheduleDismiss(id, duration);
	}

	function pauseAll() {
		const now = Date.now();
		for (const id of Object.keys(timers)) {
			const entry = timers[id];
			if (entry.timeoutId) {
				clearTimeout(entry.timeoutId);
				entry.remaining = Math.max(entry.remaining - (now - entry.startedAt), 0);
				entry.timeoutId = null;
			}
		}
	}

	function resumeAll() {
		for (const id of Object.keys(timers)) {
			const entry = timers[id];
			if (!entry.timeoutId && entry.remaining > 0) {
				scheduleDismiss(id, entry.remaining);
			}
		}
	}

	$effect(() => {
		if (!browser) return;
		const byId = new Map(visible.map((t) => [t.id, t]));

		for (const id of Object.keys(timers)) {
			const t = byId.get(id);
			if (!t || t.duration === 0 || timers[id].createdAt !== t.createdAt) {
				clearTimer(id);
			}
		}

		for (const t of visible) {
			if (t.duration > 0 && !timers[t.id]) {
				startTimer(t.id, t.duration, t.createdAt);
			}
		}
	});

	$effect(() => {
		if (!browser) return;
		if (expanded) pauseAll();
		else resumeAll();
	});

	onDestroy(() => {
		for (const id of Object.keys(timers)) clearTimer(id);
	});

	function transformFor(index: number): string {
		if (expanded) {
			let y = 0;
			for (let i = 0; i < index; i++) {
				const id = visible[i]?.id;
				const h = id ? (heights[id] ?? 64) : 64;
				y -= h + EXPANDED_GAP;
			}
			return `translateY(${y}px) scale(1)`;
		}
		const y = -index * COLLAPSED_PEEK;
		const scale = Math.max(1 - index * 0.05, 0.9);
		return `translateY(${y}px) scale(${scale})`;
	}

	function opacityFor(index: number): number {
		if (expanded) return 1;
		if (index === 0) return 1;
		if (index === 1) return 0.85;
		return 0.7;
	}

	function isInteractive(index: number): boolean {
		return expanded || index === 0;
	}

	function handlePointerEnter(event: PointerEvent) {
		if (event.pointerType === 'mouse' || event.pointerType === 'pen') {
			expanded = true;
		}
	}

	function handlePointerLeave(event: PointerEvent) {
		if (event.pointerType === 'mouse' || event.pointerType === 'pen') {
			expanded = false;
		}
	}

	function handleFocusIn() {
		expanded = true;
	}

	function handleFocusOut(event: FocusEvent) {
		const next = event.relatedTarget as Node | null;
		if (!next || !region || !region.contains(next)) {
			expanded = false;
		}
	}

	function toneAccentClass(tone: Toast['tone']): string {
		switch (tone) {
			case 'success':
				return 'tone-success';
			case 'error':
				return 'tone-error';
			default:
				return 'tone-neutral';
		}
	}

	function fallbackIconFor(tone: Toast['tone']) {
		if (tone === 'success') return CheckCircle;
		if (tone === 'error') return ExclamationCircle;
		return null;
	}
</script>

<div
	bind:this={region}
	class="toaster-region"
	role="region"
	aria-label="Notifications"
	aria-live="polite"
	data-expanded={expanded ? 'true' : 'false'}
	onpointerenter={handlePointerEnter}
	onpointerleave={handlePointerLeave}
	onfocusin={handleFocusIn}
	onfocusout={handleFocusOut}
>
	<div class="toaster-stack">
		{#each visible as toast, index (toast.id)}
			{@const fallback = fallbackIconFor(toast.tone)}
			<div
				class="toast-slot"
				style="transform: {transformFor(index)}; opacity: {opacityFor(index)}; z-index: {100 -
					index};"
				data-front={index === 0 ? 'true' : 'false'}
				inert={!isInteractive(index)}
			>
				<div
					class="toast-enter"
					bind:clientHeight={heights[toast.id]}
					in:fly={{ y: 18, duration: 240 }}
					out:fade={{ duration: 160 }}
				>
					<div class="toast {toneAccentClass(toast.tone)}">
						<span class="toast-icon" aria-hidden="true">
							{#if toast.icon?.kind === 'spinner'}
								<Spinner class="h-4 w-4 text-muted-foreground" />
							{:else if toast.icon?.kind === 'heroicon'}
								<Icon src={toast.icon.src} theme="outline" class="h-4 w-4" />
							{:else if toast.icon?.kind === 'lucide'}
								{@const LucideIcon = toast.icon.component}
								<LucideIcon class="h-4 w-4" strokeWidth={1.75} />
							{:else if fallback}
								<Icon src={fallback} theme="outline" class="h-4 w-4" />
							{/if}
						</span>
						<div class="toast-body">
							<div class="toast-title">{toast.title}</div>
							{#if toast.description}
								<div class="toast-description">{toast.description}</div>
							{/if}
						</div>
						{#if toast.dismissible}
							<button
								type="button"
								class="toast-close"
								onclick={() => toastStore.dismiss(toast.id)}
								aria-label="Dismiss notification"
							>
								<Icon src={XMark} theme="outline" class="h-3.5 w-3.5" />
							</button>
						{/if}
					</div>
				</div>
			</div>
		{/each}
	</div>
</div>

<style>
	.toaster-region {
		position: fixed;
		right: 3rem;
		bottom: 3rem;
		z-index: 80;
		width: min(22rem, calc(100vw - 2rem));
		pointer-events: none;
	}

	.toaster-stack {
		position: relative;
		height: 0;
	}

	.toast-slot {
		position: absolute;
		right: 0;
		bottom: 0;
		width: 100%;
		transform-origin: bottom right;
		transition:
			transform 280ms cubic-bezier(0.22, 1, 0.36, 1),
			opacity 200ms ease-out;
		pointer-events: auto;
	}

	.toast-slot[inert] {
		pointer-events: none;
	}

	.toast-enter {
		display: block;
	}

	.toast {
		display: grid;
		grid-template-columns: auto 1fr auto;
		align-items: start;
		gap: 0.625rem;
		padding: 0.6rem 0.75rem;
		background: var(--card);
		color: var(--card-foreground);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		box-shadow:
			0 1px 2px rgba(17, 17, 17, 0.06),
			0 8px 24px -12px rgba(17, 17, 17, 0.18);
	}

	.toast.tone-success {
		border-color: color-mix(in srgb, var(--success) 22%, var(--border));
	}

	.toast.tone-success .toast-icon {
		color: var(--success);
	}

	.toast.tone-error {
		border-color: color-mix(in srgb, var(--destructive) 28%, var(--border));
	}

	.toast.tone-error .toast-icon {
		color: var(--destructive);
	}

	.toast-icon {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 1.25rem;
		height: 1.25rem;
		margin-top: 0.05rem;
		color: var(--muted-foreground);
	}

	.toast-body {
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 0.125rem;
	}

	.toast-title {
		font-size: 0.8125rem;
		font-weight: 600;
		line-height: 1.25;
		color: var(--foreground);
	}

	.toast-description {
		font-size: 0.75rem;
		line-height: 1.35;
		color: var(--muted-foreground);
	}

	.toast-close {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 1.25rem;
		height: 1.25rem;
		margin: -0.125rem -0.125rem 0 0;
		border-radius: var(--radius-sm);
		color: var(--muted-foreground);
		background: transparent;
		cursor: pointer;
		transition:
			background-color 120ms ease-out,
			color 120ms ease-out;
	}

	.toast-close:hover {
		background: var(--accent);
		color: var(--foreground);
	}

	.toast-close:focus-visible {
		outline: none;
		box-shadow: 0 0 0 2px var(--ring);
	}

	@media (hover: none) {
		.toaster-region {
			pointer-events: none;
		}
		.toast-slot:not([data-front='true']) {
			pointer-events: none;
		}
	}
</style>
