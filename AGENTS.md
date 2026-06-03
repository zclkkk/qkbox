# Agent Guidelines

## No Intermediate Complexity

Do not introduce workarounds, shims, or stopgap solutions that will be removed or
refactored by a later Phase. If a problem will naturally resolve when a future Phase
is implemented, leave it as-is.

Conversely, if current code would impose extra cost on future Phases, remove it
even if the current Phase becomes less polished.

Imperfect intermediate states are acceptable. Accumulated technical debt is not.
