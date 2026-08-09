<script lang="ts" module>
	export type PendingService = {
		id: string;
		name: string;
		image: string;
		/** False means it has never landed on a server, so deploying creates it. */
		has_deployed: boolean;
		/**
		 * How many things about this service differ from what its last deployment
		 * shipped. Counted per field, per domain and per variable — so the bar can
		 * say how much is waiting rather than how many services are involved.
		 */
		pending_change_count: number;
	};
</script>

<script lang="ts">
	import { untrack } from 'svelte';
	import { browser } from '$app/environment';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/v1';
	import Button from '$lib/components/ui/Button.svelte';
	import {
		Dialog,
		DialogContent,
		DialogHeader,
		DialogFooter,
		DialogTitle,
		DialogDescription
	} from '$lib/components/ui/dialog';
	import { Container } from 'lucide-svelte';
	import { cn } from '$lib/components/ui/cn.js';

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

	let detailsOpen = $state(false);

	// Three states, not two, because the bar leaves for two different reasons and
	// they do not mean the same thing. `hidden` is "there is nothing to say" — it
	// slides off to the right, the way it arrived. `receded` is "the dialog is
	// saying it better right now" — it shrinks toward its own top edge, so the
	// review surface reads as having come out of the bar rather than replacing it.
	let barState = $derived(
		services.length === 0 || suppressed ? 'hidden' : detailsOpen ? 'receded' : 'visible'
	);
	let visible = $derived(barState === 'visible');

	// The bar animates out rather than vanishing, so it is still on screen for
	// ~150ms after `services` empties. Rendering live props during that window
	// would flash "0 pending changes" on the way out; these hold the last real
	// values until the next time it opens. The dialog reads them too, so its list
	// does not empty out mid-close.
	let shownServices = $state<PendingService[]>([]);
	$effect(() => {
		if (services.length > 0) shownServices = services;
	});

	// The changes themselves, not the services carrying them. Editing an image and
	// adding two domains on one service is three things waiting to happen, and a
	// bar that called that "1 pending change" was answering a question nobody
	// asked — then opening a dialog that listed three.
	let count = $derived(shownServices.reduce((n, svc) => n + svc.pending_change_count, 0));
	let countLabel = $derived(count === 1 ? '1 pending change' : `${count} pending changes`);

	type PendingChanges = components['schemas']['PendingChangesResponse'];
	type ConfigChange = components['schemas']['ConfigChange'];

	// Per service id. Absent means not fetched yet, which is what the loading row
	// reads from — the dialog opens immediately and fills in, rather than holding
	// the whole surface back for a request that is usually instant.
	let diffs = $state<Record<string, PendingChanges>>({});
	let diffsFailed = $state<Record<string, boolean>>({});

	/**
	 * Fetched when the dialog opens rather than alongside the list, because the
	 * list is loaded on every canvas render and this is only ever read by someone
	 * who asked to review. One request per pending service, which is one to three
	 * in practice — the count on the bar already came from the same comparison, so
	 * this is only fetching the itemisation of a number that is already on screen.
	 */
	$effect(() => {
		if (!detailsOpen) return;
		// untrack: this fires once per opening, keyed on that alone. Reading the
		// service list or the results normally would re-enter on every response and
		// re-fetch what it had just stored.
		untrack(() => {
			// Discarded rather than reused: something edited since the last look must
			// not be reviewed against the list from before it.
			diffs = {};
			diffsFailed = {};
			for (const svc of shownServices) loadDiff(svc.id);
		});
	});

	async function loadDiff(id: string) {
		const { data } = await api.GET('/api/services/{id}/pending-changes', {
			params: { path: { id } }
		});
		if (data) {
			diffs[id] = data;
		} else {
			diffsFailed[id] = true;
		}
	}

	/**
	 * How a change reads on its row. A removed thing shows what is going away, an
	 * added one shows what arrives, and only a change that kept its identity has
	 * two sides worth putting an arrow between.
	 */
	function changeValues(change: ConfigChange): { from: string | null; to: string | null } {
		return {
			from: change.type === 'added' ? null : (change.old_value ?? null),
			to: change.type === 'removed' ? null : (change.new_value ?? null)
		};
	}

	function deployAndClose() {
		detailsOpen = false;
		onDeploy();
	}

	function selectAndClose(id: string) {
		detailsOpen = false;
		onSelect(id);
	}
	// Spelled "Enter" rather than drawn with ↵ (U+21B5): that glyph is missing
	// from most UI sans fonts, so it falls back to a different family and lands
	// off the baseline next to ⌘. ⌘ itself is safe — it ships in the system stack
	// on the only platform that shows it.
	let shortcutHint = $derived(
		browser && /Mac|iPhone|iPad|iPod/.test(navigator.platform) ? '⌘+Enter' : 'Ctrl+Enter'
	);

	function handleKeydown(event: KeyboardEvent) {
		// Works from the dialog too, not just the bar: the shortcut is advertised on
		// both Deploy buttons, so it would be a small betrayal if opening the review
		// silently disarmed it.
		if (barState === 'hidden' || deploying) return;
		if (event.key !== 'Enter' || !(event.metaKey || event.ctrlKey)) return;
		event.preventDefault();
		deployAndClose();
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<!-- The bar belongs to the canvas, so it travels with it. When the inspector
     opens, the canvas slides left by half the panel width to re-centre the nodes
     in what is left; without the same shift the bar stayed at viewport centre and
     slid under the panel. Same transform, same 280ms curve as .inspector-shift —
     the panel opening and everything on the canvas making room for it is one
     gesture, so it runs on one clock. -->
<div class="bar-shift">
	<!-- No aria-hidden: `visibility: hidden` already takes the bar out of both the
	     accessibility tree and the tab order, and layering aria-hidden on top would
	     mute the live region at exactly the moment it has something to announce. -->
	<div class="bar" data-state={barState} data-no-pan>
		<p class="count" role="status">
			<span class="tabular-nums">{count}</span>
			{count === 1 ? 'pending change' : 'pending changes'}
		</p>

		<Button
			variant="ghost"
			size="sm"
			class="h-8 px-2 text-xs text-muted-foreground"
			disabled={!visible}
			onclick={() => (detailsOpen = true)}
		>
			Details
		</Button>

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
</div>

<Dialog bind:open={detailsOpen}>
	<DialogContent class="max-w-xl">
		<DialogHeader>
			<DialogTitle>{countLabel}</DialogTitle>
			<DialogDescription>
				What your servers would have to change to match what is on the canvas.
			</DialogDescription>
		</DialogHeader>

		<!-- divide-y rather than a border on each row: the footer already draws its own
		     top edge, and a border-bottom on the last row would double it. -->
		<ul class="max-h-96 divide-y divide-border overflow-y-auto border-t border-border">
			{#each shownServices as svc (svc.id)}
				{@const diff = diffs[svc.id]}
				<li>
					<!-- The service name stays a button: it opens that service's panel so
					     you can check a change before committing to it, which is the only
					     reason to open a review surface rather than just pressing Deploy. -->
					<button
						type="button"
						onclick={() => selectAndClose(svc.id)}
						class="flex w-full cursor-pointer items-center gap-3 px-5 py-3 text-left transition-colors hover:bg-accent focus-visible:bg-accent focus-visible:outline-none"
					>
						<span
							class="grid h-8 w-8 flex-none place-content-center rounded-md bg-muted text-foreground"
						>
							<Container class="h-4 w-4" strokeWidth={1.75} />
						</span>
						<span class="min-w-0 flex-1">
							<span class="block truncate text-sm text-foreground">
								<span class="font-medium">{svc.name}</span>
								<span class="text-muted-foreground">
									{svc.has_deployed ? 'will be updated' : 'will be added'}
								</span>
							</span>
							<span class="block truncate font-mono text-[11px] text-muted-foreground">
								{svc.image}
							</span>
						</span>
					</button>

					<!-- Indented under the service rather than in a table of its own: these
					     rows belong to the name above them, and at this width a Change /
					     Current / New table would give each column about nine characters. -->
					{#if !diff}
						<p class="px-5 pb-3 pl-16 text-[13px] text-muted-foreground">
							{diffsFailed[svc.id] ? 'Could not read what changed.' : 'Reading changes…'}
						</p>
					{:else if !diff.has_baseline}
						<!-- The honest answer, not an empty list. A service Uploy has no record
						     of is genuinely pending, and showing nothing under a badge that
						     says otherwise is how a reader learns to distrust the badge. -->
						<p class="px-5 pb-3 pl-16 text-[13px] text-muted-foreground">
							{svc.has_deployed
								? 'Uploy has no record of what this service is running. Deploy to bring it in line.'
								: 'This service has never been deployed.'}
						</p>
					{:else}
						<ul class="px-5 pb-3 pl-16">
							{#each diff.changes as change (change.key)}
								{@const values = changeValues(change)}
								<li class="flex min-w-0 items-baseline gap-2 py-1 text-[13px]">
									<!-- The mark carries added / removed / changed, so the row does
									     not spend a word on it. Aria-hidden with the word restored
									     to a screen reader: +/− is a shape, not a sentence. -->
									<span
										class={cn(
											'w-3 flex-none text-center font-mono',
											change.type === 'added' && 'text-success',
											change.type === 'removed' && 'text-destructive',
											change.type === 'changed' && 'text-muted-foreground'
										)}
										aria-hidden="true"
									>
										{change.type === 'added' ? '+' : change.type === 'removed' ? '−' : '~'}
									</span>
									<span class="sr-only">{change.type}:</span>
									<span class="flex-none text-foreground">{change.label}</span>
									<span class="min-w-0 flex-1 truncate text-right font-mono text-muted-foreground">
										{#if values.from !== null && values.to !== null}
											<span class="line-through opacity-60">{values.from}</span>
											<span class="px-1" aria-hidden="true">→</span>
											<span class="text-foreground">{values.to}</span>
										{:else if values.to !== null}
											<span class="text-foreground">{values.to}</span>
										{:else}
											<span class="line-through opacity-60">{values.from}</span>
										{/if}
									</span>
								</li>
							{/each}
						</ul>
					{/if}
				</li>
			{/each}
		</ul>

		<DialogFooter>
			<Button variant="secondary" size="sm" onclick={() => (detailsOpen = false)}>Cancel</Button>
			<Button
				size="sm"
				loading={deploying}
				disabled={deploying}
				onclick={deployAndClose}
				aria-keyshortcuts="Meta+Enter Control+Enter"
			>
				Deploy changes
				<span class="shortcut" aria-hidden="true">{shortcutHint}</span>
			</Button>
		</DialogFooter>
	</DialogContent>
</Dialog>

<style>
	/* Spans the canvas so the bar's own `left: 50%` and `max-width: 100%` still
	   resolve against the same box they always did — this only moves that box.
	   pointer-events: none because it covers the whole canvas: without it the
	   wrapper would swallow every drag meant for panning. */
	.bar-shift {
		position: absolute;
		inset: 0;
		z-index: 20;
		pointer-events: none;
		transform: translate3d(calc(0px - var(--inspector-shift, 0px)), 0, 0);
		transition: transform 280ms cubic-bezier(0.23, 1, 0.32, 1);
	}

	.bar-shift > .bar {
		pointer-events: auto;
	}

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
		/* Floats over canvas content rather than sitting beside it, so it takes the
		   --shadow-float exception to the hairline-only elevation rule. */
		box-shadow: var(--shadow-float);
		cursor: default;

		/* The scale states shrink the bar toward its own top edge, so it reads as
		   folding up into the space the dialog is about to occupy rather than
		   collapsing into thin air. Pairs with `left: 50%` + the -50% translate
		   below, which is what actually centres it. */
		transform-origin: 50% 0;

		/* --- hidden: nothing to say ---------------------------------------- */
		opacity: 0;
		visibility: hidden;
		/* Arrives from the right, on the same 200ms / (0.23, 1, 0.32, 1) curve the
		   canvas uses to step between the starter panel and the add-service form —
		   so a change landing on the canvas and a step through the builder read as
		   one motion vocabulary rather than two.

		   The -50% is what centres the bar, so the 12px offset has to ride inside
		   the same translate. Splitting it across a second transform function would
		   reset the centring mid-transition. */
		transform: translate(calc(-50% + 12px), 0) scale(1);
		/* Leaves the way it came, at ~75% of the entrance: dismissal should get out
		   of the way faster than arrival asked for attention. `visibility` is held
		   until the fade finishes so the bar is never tabbable mid-flight. */
		transition:
			opacity 150ms cubic-bezier(0.23, 1, 0.32, 1),
			transform 150ms cubic-bezier(0.23, 1, 0.32, 1),
			visibility 0s linear 150ms;
	}

	/* --- visible: at rest, and the state everything returns to -------------- */
	.bar[data-state='visible'] {
		opacity: 1;
		visibility: visible;
		transform: translate(-50%, 0) scale(1);
		transition:
			opacity 200ms cubic-bezier(0.23, 1, 0.32, 1),
			transform 200ms cubic-bezier(0.23, 1, 0.32, 1),
			visibility 0s linear 0s;
	}

	/* --- receded: the review dialog is up ----------------------------------- */
	.bar[data-state='receded'] {
		opacity: 0;
		visibility: hidden;
		transform: translate(-50%, 0) scale(0.94);
		/* 140ms ease-in, matching dialog-content-out exactly: the bar folding away
		   and the dialog arriving are one gesture, so they share a clock. Ease-in
		   (accelerating out) is what makes it read as receding rather than being
		   dismissed — the return trip uses the entrance curve above instead. */
		transition:
			opacity 140ms cubic-bezier(0.4, 0, 1, 1),
			transform 140ms cubic-bezier(0.4, 0, 1, 1),
			visibility 0s linear 140ms;
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
		.bar,
		.bar[data-state='receded'] {
			transform: translate(-50%, 0) scale(1);
			transition:
				opacity 120ms linear,
				visibility 0s linear 120ms;
		}

		.bar[data-state='visible'] {
			transform: translate(-50%, 0) scale(1);
			transition:
				opacity 120ms linear,
				visibility 0s linear 0s;
		}
	}
</style>
