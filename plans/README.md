# Animation plans

Produced by the `improve-animations` skill. Each plan is self-contained — an
executor needs nothing from the conversation that generated it.

| # | Title | Severity | Status |
| --- | --- | --- | --- |
| [001](001-inspector-drawer-motion.md) | Give the inspector panel a drawer entrance/exit and let its motion actually run | HIGH | DONE |

## Execution order

001 has no dependencies. It is the whole of the builder-inspector motion fix —
the panel transition, the shared clock with the canvas shift, and the removal of
the reduced-motion rule that was suppressing the animation must land together or
the result is still desynced.
