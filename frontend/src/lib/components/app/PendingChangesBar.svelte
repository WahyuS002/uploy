<script lang="ts" module>
	export type PendingService = {
		id: string;
		name: string;
		image: string;
	};
</script>

<script lang="ts">
	import { browser } from '$app/environment';
	import Button from '$lib/components/ui/Button.svelte';
	import { DropdownMenu } from 'bits-ui';
	import { Icon } from '@steeze-ui/svelte-icon';
	import { ChevronDown } from '@steeze-ui/heroicons';
	import { Container } from 'lucide-svelte';

	type Props = {
		/** Services with undeployed changes. Empty means nothing to show. */
		services: PendingService[];
		/**
		 * Hold the bar back even when there are pending changes — the canvas is
		 * already asking for something else (the add-service step, a dialog), and
		 * two competing primary actions on one screen is one too many.
		 */
		suppressed?: boolean;
		deploying?: boolean;
		onDeploy: () => void;
		onSelect: (id: string) => void;
	};

	let { services, suppressed = false, deploying = false, onDeploy, onSelect }: Props = $props();

	let visible = $derived(services.length > 0 && !suppressed);

	// The bar animates out rather than vanishing, so it is still on screen for
	// ~160ms after `services` empties. Rendering live props during that window
	// would flash "0 pending changes" on the way out; these hold the last real
	// values until the next time it opens.
	let shownServices = $state<PendingService[]>([]);
	$effect(() => {
		if (services.length > 0) shownServices = services;
	});

	let count = $derived(shownServices.length);
	// Spelled "Enter" rather than drawn with ↵ (U+21B5): that glyph is missing
	// from most UI sans fonts, so it falls back to a different family and lands
	// off the baseline next to ⌘. ⌘ itself is safe — it ships in the system stack
	// on the only platform that shows it.
	let shortcutHint = $derived(
		browser && /Mac|iPhone|iPad|iPod/.test(navigator.platform) ? '⌘+Enter' : 'Ctrl+Enter'
	);

	function handleKeydown(event: KeyboardEvent) {
		if (!visible || deploying) return;
		if (event.key !== 'Enter' || !(event.metaKey || event.ctrlKey)) return;
		event.preventDefault();
		onDeploy();
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<!-- No aria-hidden: `visibility: hidden` already takes the bar out of both the
     accessibility tree and the tab order, and layering aria-hidden on top would
     mute the live region at exactly the moment it has something to announce. -->
<div class="bar" data-visible={visible} data-no-pan>
	<p class="count" role="status">
		<span class="tabular-nums">{count}</span>
		{count === 1 ? 'pending change' : 'pending changes'}
	</p>

	<DropdownMenu.Root>
		<DropdownMenu.Trigger
			disabled={!visible}
			class="inline-flex h-8 cursor-pointer items-center gap-1 rounded-md px-2 text-xs font-medium text-muted-foreground transition-colors outline-none hover:bg-accent hover:text-accent-foreground focus-visible:ring-2 focus-visible:ring-ring/40 data-[state=open]:bg-accent data-[state=open]:text-accent-foreground"
		>
			Details
			<Icon src={ChevronDown} theme="outline" class="h-3 w-3" />
		</DropdownMenu.Trigger>
		<DropdownMenu.Portal>
			<DropdownMenu.Content
				align="center"
				sideOffset={8}
				class="z-50 max-h-72 w-64 overflow-y-auto rounded-lg border border-border bg-popover p-1 text-popover-foreground shadow-overlay"
			>
				<p class="px-2 py-1.5 text-[11px] font-medium text-muted-foreground">Waiting to deploy</p>
				{#each shownServices as svc (svc.id)}
					<DropdownMenu.Item
						onSelect={() => onSelect(svc.id)}
						class="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 outline-none data-highlighted:bg-accent"
					>
						<span
							class="grid h-6 w-6 flex-none place-content-center rounded bg-muted text-foreground"
						>
							<Container class="h-3 w-3" strokeWidth={1.75} />
						</span>
						<span class="min-w-0 flex-1">
							<span class="block truncate text-sm text-foreground">{svc.name}</span>
							<span class="block truncate font-mono text-[11px] text-muted-foreground">
								{svc.image}
							</span>
						</span>
					</DropdownMenu.Item>
				{/each}
			</DropdownMenu.Content>
		</DropdownMenu.Portal>
	</DropdownMenu.Root>

	<Button
		size="sm"
		class="h-8 px-3"
		disabled={!visible || deploying}
		loading={deploying}
		onclick={onDeploy}
		aria-keyshortcuts="Meta+Enter Control+Enter"
	>
		Deploy
		<!-- A hint, not a key cap: a filled chip inside a filled button is a box
		     inside a box, and at 11px it fought the label instead of supporting it.
		     aria-hidden because the button already carries the real thing in
		     aria-keyshortcuts; announcing "Deploy Command plus Enter" would just
		     say it twice, in worse words. -->
		<span class="shortcut" aria-hidden="true">{shortcutHint}</span>
	</Button>
</div>

<style>
	.bar {
		position: absolute;
		top: 0.75rem;
		left: 50%;
		z-index: 20;
		display: flex;
		align-items: center;
		gap: 0.25rem;
		/* Never wider than the canvas it floats in, however long the count gets. */
		max-width: calc(100% - 1.5rem);
		padding: 0.375rem 0.375rem 0.375rem 0.875rem;
		background: var(--card);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		/* Deliberately softer than --shadow-panel's flat ring: this floats over
		   canvas content rather than sitting beside it, so it needs a little real
		   lift. Kept under 8px blur so it stays a hairline-plus-lift, not a glow. */
		box-shadow:
			0 1px 2px rgba(3, 3, 3, 0.04),
			0 4px 8px -2px rgba(3, 3, 3, 0.06);
		cursor: default;

		opacity: 0;
		visibility: hidden;
		/* Arrives from the right, on the same 200ms / (0.23, 1, 0.32, 1) curve the
		   canvas uses to step between the starter panel and the add-service form —
		   so a change landing on the canvas and a step through the builder read as
		   one motion vocabulary rather than two.

		   The -50% is what centres the bar (it pairs with `left: 50%`), so the 12px
		   offset has to ride inside the same translate. Splitting it across a second
		   transform function would reset the centring mid-transition. */
		transform: translate(calc(-50% + 12px), 0);
		/* Leaves the way it came, at ~75% of the entrance: dismissal should get out
		   of the way faster than arrival asked for attention. `visibility` is held
		   until the fade finishes so the bar is never tabbable mid-flight. */
		transition:
			opacity 150ms cubic-bezier(0.23, 1, 0.32, 1),
			transform 150ms cubic-bezier(0.23, 1, 0.32, 1),
			visibility 0s linear 150ms;
	}

	.bar[data-visible='true'] {
		opacity: 1;
		visibility: visible;
		transform: translate(-50%, 0);
		transition:
			opacity 200ms cubic-bezier(0.23, 1, 0.32, 1),
			transform 200ms cubic-bezier(0.23, 1, 0.32, 1),
			visibility 0s linear 0s;
	}

	.count {
		font-size: 0.8125rem;
		line-height: 1.2;
		font-weight: 500;
		color: var(--muted-foreground);
		white-space: nowrap;
		padding-right: 0.375rem;
	}

	.count span {
		font-weight: 600;
		color: var(--foreground);
	}

	/* A shortcut hint is noise on a device with no modifier key, and the width it
	   costs is exactly what a 320px-wide canvas cannot spare. Same `pointer: fine`
	   gate the canvas toolbar uses. */
	.shortcut {
		display: none;
		align-items: center;
		font-size: 0.6875rem;
		font-weight: 500;
		line-height: 1;
		/* Small text tightens up optically; a hair of tracking keeps "⌘+Enter"
		   from reading as one word. */
		letter-spacing: 0.01em;
		/* Opacity rather than a fixed tint: the button fill steps through three
		   values (mint → hover → active), and a colour computed against one of
		   them would drift out of contrast on the others. At 0.65 the ink holds
		   5.6:1 on the resting fill and 5.1:1 on hover — above 4.5:1 either way,
		   which 11px text needs. */
		opacity: 0.65;
	}

	@media (pointer: fine) {
		.shortcut {
			display: inline-flex;
		}
	}

	/* The slide in from the right is what says "this arrived"; with reduce on, the
	   crossfade has to carry that alone, so it keeps a real duration rather than
	   being zeroed out. Both states pin the centred transform, which is what
	   actually removes the 12px travel — dropping `transform` from the transition
	   list alone would still let it jump. */
	@media (prefers-reduced-motion: reduce) {
		.bar {
			transform: translate(-50%, 0);
			transition:
				opacity 120ms linear,
				visibility 0s linear 120ms;
		}

		.bar[data-visible='true'] {
			transform: translate(-50%, 0);
			transition:
				opacity 120ms linear,
				visibility 0s linear 0s;
		}
	}
</style>
