# 002 — Make the stacked deployment drawer arrive from the top-right while the inspector recedes down-left

- **Status**: DONE (applied on `feat/ui-redesign`, uncommitted at time of writing).
  The motion values below shipped unchanged, but the layer they animate has since
  moved: the log panel is now `frontend/src/lib/components/app/DeploymentPanel.svelte`,
  a page-level peer of the inspector that owns the `stack` transition itself,
  rather than an overlay rendered inside `ServiceWorkspace`. The `.is-stacked`
  recede on `.side-panel` is unchanged.
- **Commit**: 388b815
- **Severity**: MEDIUM
- **Category**: Physicality & origin (3), Cohesion & tokens (7)
- **Estimated scope**: 2 files, ~30 lines changed

## Problem

Clicking a row under **Previous** in the Deployments tab opens a second panel
over the inspector, holding that deployment's logs. It currently enters on the
same purely-horizontal path as the inspector itself:

```svelte
<!-- frontend/src/lib/components/app/ServiceWorkspace.svelte:89 — current -->
const stack: (node: Element) => TransitionConfig = () => ({
	duration: 240,
	easing: quintOut,
	css: (_t: number, u: number) => `transform: translate3d(${u * 100}%, 0, 0)`
});
```

Two problems follow from that.

1. **The two levels are indistinguishable.** The inspector enters from the right
   edge (`+page.svelte:94`, `translate3d(calc(u * (100% + 0.75rem)), 0, 0)`), and
   so does the layer that stacks on top of it. Same origin, same axis, so nothing
   in the motion says the second panel is a level deeper rather than a
   replacement for the first.

2. **The panel underneath does not react.** It sits perfectly still while another
   surface slides over it, which reads as an unrelated overlay dropped on the
   page rather than as one card stacking onto another. Nothing establishes the
   depth relationship that the 16px sliver of inspector left visible at
   `left-4` is trying to imply.

## Target

One gesture, two surfaces, one clock — the same doctrine plan 001 applied to the
inspector and the canvas shift.

- The inspector (`aside.side-panel`) **recedes down-left** by `translate3d(-10px, 8px, 0)`.
- The stacked layer **arrives from the top-right** — off the right edge *and*
  24px high — and lands on the inspector's new resting plane, so its own travel
  is down-and-left too.
- Both run **260ms** on **`cubic-bezier(0.23, 1, 0.32, 1)`** (`quintOut`, the
  project's ease-out curve). Because the layer is a child of the aside, the two
  transforms compose: the layer's arc is its own diagonal plus the recede.

```ts
/* target — frontend/src/lib/components/app/ServiceWorkspace.svelte */
const stack: (node: Element) => TransitionConfig = () => ({
	duration: 260,
	easing: quintOut,
	css: (_t: number, u: number) =>
		`transform: translate3d(calc(${u} * (100% + 1rem)), ${u * -24}px, 0)`
});
```

```css
/* target — frontend/src/routes/(app)/(builder)/projects/[id]/+page.svelte */
.side-panel {
	box-shadow: var(--shadow-float);
	transition: transform 260ms cubic-bezier(0.23, 1, 0.32, 1);
}

.side-panel.is-stacked {
	transform: translate3d(-10px, 8px, 0);
}
```

Reverse playback is the exit: removing `.is-stacked` returns the inspector on the
same curve while the layer leaves back up and to the right.

## Repo conventions to follow

- No easing CSS variables exist. Curves are written inline as
  `cubic-bezier(0.23, 1, 0.32, 1)` in CSS and as `quintOut` from `svelte/easing`
  in transition functions — they are the same curve. Exemplar:
  `frontend/src/routes/(app)/(builder)/projects/[id]/+page.svelte:94` (the
  inspector `drawer` transition) and `:822` (`.inspector-shift`).
- Travel is expressed as `translate3d(...)` percentages of the moving element
  where the element's own width is the distance, plus a fixed rem/px for the gap
  it must clear. Same exemplar.
- No blanket `prefers-reduced-motion` override in this project
  (`frontend/src/routes/layout.css:171`); individual elements opt out only for
  genuinely disruptive motion — infinite pulses, parallax. A 260ms drawer slide
  does not qualify and must NOT get an opt-out.
- Elevation is a hairline ring; `--shadow-float` is the documented exception for
  surfaces that sit *over* content. The stacked layer already uses it.

## Steps

1. In `frontend/src/lib/components/app/ServiceWorkspace.svelte`, add a bindable
   prop so the host can react to the layer being open. In the `Props` type, next
   to `class?: string;`, add:

   ```ts
   /**
    * True while a past deployment's log panel is stacked over this workspace.
    * The host owns the surface underneath, so it is the only thing that can
    * make it recede — bound rather than internal for that reason.
    */
   stacked?: boolean;
   ```

   and in the destructuring add `stacked = $bindable(false),`.

2. In the same file, keep `openDeployment` as the single source of truth and
   mirror it into the prop:

   ```svelte
   $effect(() => {
   	stacked = openDeployment !== null;
   });
   ```

   Place it directly after the `openDeployment` declaration (currently line 87).

3. In the same file, replace the `stack` transition body (currently lines 91–95)
   with the target above: `duration: 260`, and
   `css: (_t, u) => \`transform: translate3d(calc(${u} * (100% + 1rem)), ${u * -24}px, 0)\``.
   `1rem` clears the `left-4` sliver so the layer starts fully off the panel;
   `-24px` is the height it falls from.

4. In the same file, reset the prop when the service changes. The existing
   `$effect` that resets panel state already sets `openDeployment = null;` — no
   extra line is needed if step 2's effect is in place. Confirm that is true and
   change nothing else there.

5. In `frontend/src/routes/(app)/(builder)/projects/[id]/+page.svelte`, add
   `let inspectorStacked = $state(false);` next to the other panel state (near
   `let selectedServiceId`), bind it on the component:

   ```svelte
   <ServiceWorkspace
   	service={selectedService}
   	{canEdit}
   	{isOwner}
   	bind:stacked={inspectorStacked}
   	externalDeploymentId={barDeploymentIds[selectedService.id] ?? null}
   	onDeleted={removeService}
   />
   ```

   and put the class on the aside: `class:is-stacked={inspectorStacked}`.

6. In the same file's `<style>` block, extend the existing `.side-panel` rule
   (currently at `:964`) with the transition and add the `.is-stacked` rule, both
   exactly as written in **Target**. Leave the existing `box-shadow` line alone.

## Boundaries

- Do NOT touch `frontend/src/routes/(app)/(dashboard)/services/[id]/+page.svelte`.
  There the workspace sits in a static card in normal page flow, not a floating
  panel, and sliding it down-left would just detach it from its own heading. The
  layer's own entrance still applies there; the recede is builder-only by design.
- Do NOT add a `prefers-reduced-motion` opt-out for either transform.
- Do NOT change the inspector's own `drawer` transition (`+page.svelte:94`) or
  `.inspector-shift`.
- Do NOT add scale, blur, or opacity to the recede. Scaling live text
  re-rasterizes it mid-transition, and the scrim already carries the dimming.
- Do NOT add dependencies.
- If a step does not match the code you find, STOP and report rather than
  improvising.

## Verification

- **Mechanical**: from `frontend/`, `pnpm check` → 0 errors, and
  `npx eslint src` → no output.
- **Feel check**: open a project, click a service node, open **Deployments**, and
  click a row under **Previous**.
  - The inspector must start moving on the same frame as the log panel, and both
    must stop on the same frame — no second beat.
  - The log panel must visibly come from above the panel's top-right corner, not
    straight in from the right.
  - Closing (X or the scrim) must play the whole thing in reverse; clicking a
    different **Previous** row while the layer is open must not re-run the
    entrance, only swap the log stream.
  - Throttle to 4× slow motion in devtools to confirm the two transforms share a
    curve, then check at 1× that nothing overshoots the 16px sliver.
