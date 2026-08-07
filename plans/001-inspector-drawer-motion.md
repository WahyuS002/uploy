# 001 — Give the inspector panel a drawer entrance/exit and let its motion actually run

- **Status**: DONE (applied on `feat/ui-redesign`, uncommitted at time of writing)
- **Commit**: e8d380c
- **Severity**: HIGH
- **Category**: Purpose & frequency (1), Cohesion (7), Missed opportunities (8)
- **Estimated scope**: 1 file, ~25 lines changed

## Problem

Clicking a service node on the builder canvas opens the inspector panel. Three
things are wrong with how that reads, all in
`frontend/src/routes/(app)/(builder)/projects/[id]/+page.svelte`.

### 1. The panel has no entrance or exit at all

The `<aside>` lives inside an `{#if}` block with no transition directive, so it
is mounted fully formed on one frame and destroyed on another:

```svelte
<!-- frontend/src/routes/(app)/(builder)/projects/[id]/+page.svelte:742 — current -->
{#if selectedService}
	<aside
		class="side-panel absolute inset-0 z-30 flex min-h-0 w-full flex-col overflow-hidden rounded-xl border border-border bg-card text-card-foreground md:inset-y-0 md:right-0 md:left-auto md:w-[var(--inspector-width)] md:rounded-r-none"
	>
```

The panel is spatially connected to the node that opened it and occupies up to
960px of the viewport. Appearing with no motion gives the user nothing that
explains where it came from, and closing it snaps a large surface out of
existence.

### 2. The one animation that does exist never runs on a machine with Reduce motion

```css
/* frontend/src/routes/(app)/(builder)/projects/[id]/+page.svelte:933 — current */
@media (prefers-reduced-motion: reduce) {
	.inspector-shift {
		transition: none;
	}

	.node-badge.is-deploying {
		animation: none;
	}
}
```

The canvas re-centring slide (`.inspector-shift`) is the only motion in this
interaction, and this rule deletes it for anyone with OS-level Reduce motion
enabled — which includes this project's primary developer. The result is a
completely static, instantaneous jump.

This also contradicts the project's documented convention. From
`frontend/src/routes/layout.css:171`, the project deliberately dropped its
blanket reduced-motion override to match railway.com, its design reference:
motion runs by default, and elements opt out individually only where the
movement is genuinely disruptive (parallax, oscillation, large involuntary
travel). A 280ms drawer slide is not in that category. The `.node-badge.is-deploying`
opt-out *is* correct — it is an infinite pulse — and must stay.

### 3. Panel and canvas are one gesture on two different clocks

```css
/* frontend/src/routes/(app)/(builder)/projects/[id]/+page.svelte:822 — current */
.inspector-shift {
	transform: translate3d(calc(0px - var(--inspector-shift)), 0, 0);
	transition: transform 240ms cubic-bezier(0.77, 0, 0.175, 1);
	will-change: transform;
}
```

The panel sliding in and the canvas sliding left are physically the same event:
the drawer pushes the workspace over. They must start together, end together,
and travel on the same curve. Right now one is instant and the other is 240ms.

Note: `cubic-bezier(0.77, 0, 0.175, 1)` is a legitimate strong ease-in-out and is
not itself a bug. It is being changed only so both halves of the gesture share
one curve, and the panel half is an *entrance*, which takes ease-out.

## Target

One gesture, one clock: **280ms / `cubic-bezier(0.23, 1, 0.32, 1)`** for both the
panel and the canvas shift. That curve is the project's dominant UI easing
(`cubic-bezier(0.23, 1, 0.32, 1)` — a strong ease-out) and 280ms sits inside the
200–500ms budget for drawers.

The panel enters by sliding in from its own right edge (`translate3d(100%, 0, 0)`
→ `0`) and leaves the same way it came. No fade: a surface arriving from the
screen edge already explains itself, and cross-fading an opaque 960px card over
canvas content only muddies it.

```svelte
<!-- target: script section -->
import { quintOut } from 'svelte/easing';

// The panel and the canvas shift are one gesture — the drawer pushing the
// workspace over — so they share a duration and a curve. Percentage travel
// rather than pixels: the panel is a full-width sheet on mobile and a
// clamp(420px, 55vw, 960px) column on desktop, and both should arrive from
// exactly their own edge. quintOut is svelte/easing's match for the
// cubic-bezier(0.23, 1, 0.32, 1) the rest of the builder animates on.
const drawer = () => ({
	duration: 280,
	easing: quintOut,
	css: (_t: number, u: number) => `transform: translate3d(${u * 100}%, 0, 0)`
});
```

```svelte
<!-- target: markup -->
{#if selectedService}
	<aside
		transition:drawer
		class="side-panel absolute inset-0 z-30 flex min-h-0 w-full flex-col overflow-hidden rounded-xl border border-border bg-card text-card-foreground md:inset-y-0 md:right-0 md:left-auto md:w-[var(--inspector-width)] md:rounded-r-none"
	>
```

```css
/* target: styles */
.inspector-shift {
	transform: translate3d(calc(0px - var(--inspector-shift)), 0, 0);
	transition: transform 280ms cubic-bezier(0.23, 1, 0.32, 1);
	will-change: transform;
}

@media (prefers-reduced-motion: reduce) {
	.node-badge.is-deploying {
		animation: none;
	}
}
```

## Repo conventions to follow

- **Easing**: the builder's motion vocabulary is `cubic-bezier(0.23, 1, 0.32, 1)`.
  Exemplars: `frontend/src/routes/(app)/(builder)/projects/[id]/+page.svelte:865`
  (`.step-enter`), `frontend/src/routes/(app)/(builder)/projects/new/+page.svelte:360`,
  and `frontend/src/lib/components/app/PendingChangesBar.svelte:247`. Do not
  introduce a new curve.
- **Curves are written inline in each component's `<style>` block.** This project
  has no `--ease-*` token layer; `frontend/src/routes/layout.css:150` defines only
  `--animate-slide-up-fade`. Do not create an easing token layer as part of this
  plan.
- **Reduced motion is opt-in per element, never blanket.** See the comment at
  `frontend/src/routes/layout.css:171` and the worked example at
  `frontend/src/lib/components/app/PendingChangesBar.svelte:307-327`, where the
  bar keeps a real 120ms fade under `reduce` and only drops its 12px travel.
- **Svelte 5 runes syntax.** `svelte/transition` and `svelte/easing` are already
  used in `frontend/src/lib/components/ui/toast/Toaster.svelte:3`.
- **Formatting**: prettier with `prettier-plugin-svelte`, wide print width — match
  the surrounding file, do not reflow unrelated lines.

## Steps

1. In `frontend/src/routes/(app)/(builder)/projects/[id]/+page.svelte`, add
   `import { quintOut } from 'svelte/easing';` to the existing import block at the
   top of `<script lang="ts">` (alongside the other `$lib` and package imports
   around lines 1–25).

2. In the same `<script>` block, after the `$derived` declarations (i.e. after
   the `selectedService` derivation near line 84), add the `drawer` transition
   function exactly as written in the Target section above, comment included.

3. At line 743, add `transition:drawer` as the first attribute on the `<aside>`,
   before `class`. Change nothing else about that element — the class list, the
   `md:` breakpoints, and the `--inspector-width` var all stay exactly as they are.

4. At line 824, change
   `transition: transform 240ms cubic-bezier(0.77, 0, 0.175, 1);` to
   `transition: transform 280ms cubic-bezier(0.23, 1, 0.32, 1);`.

5. At lines 933-936, delete the `.inspector-shift { transition: none; }` rule from
   inside the `@media (prefers-reduced-motion: reduce)` block, leaving only the
   `.node-badge.is-deploying { animation: none; }` rule. Do not delete the media
   query itself.

## Boundaries

- Do NOT touch any file other than
  `frontend/src/routes/(app)/(builder)/projects/[id]/+page.svelte`.
- Do NOT change the panel's markup, class list, layout, or the
  `--inspector-width` / `--inspector-shift` values. Motion only.
- Do NOT add an opacity fade, a scale, a blur, or a backdrop to the panel.
- Do NOT add a new dependency — `svelte/transition` and `svelte/easing` ship with
  Svelte.
- Do NOT reintroduce a global or blanket `prefers-reduced-motion` rule anywhere.
- Do NOT touch the `.node-badge.is-deploying` reduced-motion opt-out.
- Do NOT run prettier across the repo; format only the lines you touched.
- If a step's cited code does not match what you find (drift since commit
  `e8d380c`), STOP and report rather than improvising.

## Verification

- **Mechanical**: from `frontend/`, run `pnpm check`. Expect no new errors
  attributable to this file (a pre-existing baseline of unrelated warnings is
  acceptable — compare before and after). Then `pnpm lint`; expect it to pass.
- **Feel check**: `pnpm dev`, open a project with at least one service node, and:
  - Click a node. The panel slides in from the right edge of the window; at no
    point is a partially-rendered panel visible sitting in place before it moves.
  - The canvas content starts sliding left on the **same frame** the panel starts
    moving, and both come to rest on the same frame. If one lands visibly before
    the other, the durations have drifted apart.
  - Click the ✕ to close. The panel exits back out the right edge and the canvas
    returns; the panel is not still on screen after the canvas has finished.
  - Click a different node while the panel is open. The panel stays put (it is
    not remounted) and only the content swaps — no re-slide.
  - In DevTools → Animations, set playback to 10% and confirm the panel never
    starts at `translateX(0)` and jumps, and never overshoots past its own edge.
  - In DevTools → Rendering, tick **Emulate CSS prefers-reduced-motion: reduce**
    and repeat. The drawer must still animate — this is the project's Railway
    convention, and it is the specific regression this plan exists to undo. The
    deploying badge pulse must still be frozen.
- **Done when**: opening and closing the inspector reads as one continuous 280ms
  push, both under default settings and under emulated `reduce`.
