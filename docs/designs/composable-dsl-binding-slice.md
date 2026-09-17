# Composable DSL: saved permanent binding foundation

This is a historical slice record. See the
[implementation inventory](composable-dsl-implementation.md) for current
target, choice, loop, and payment adapters and remaining work.

This change adds the saved-permanent part of the typed-binding slice. The
state-trigger slice already exists. The complete target and saved-value slice,
card migration, and later design slices remain unfinished.

The rules reference is the repository Comprehensive Rules revision effective
February 27, 2026: CR 400.7 (new objects after zone changes), 608.2 (resolution),
701.21a (sacrifice), and 702.26 (phasing).

## Existing paths and adapter contracts

| Expression | Contract | Executor and evidence |
| --- | --- | --- |
| `BindPermanent(name)` | Immutable declaration; pointer identity, diagnostic name only. Values belong to a resolution. | `ResolutionState`; `TestPermanentBindingFrames` |
| `Snapshot(subject, binding)` | Save an incarnation at this instruction. An absent subject overwrites any old value with absence. No choices, targets, events, or game actions. | Existing `Permanent.incarnationID`; `TestPermanentBindingAbsentSnapshot`, `TestPermanentBindingSnapshotReadsBeforeWrite` |
| `From(binding)` | Resolve only the saved battlefield incarnation. An absent or phased-out object yields no subject. Phasing in restores access; leaving and returning does not. | `FindPermanent` plus incarnation check; `TestPermanentBindingLifetime` |
| `AttachedPermanent` | Read the current attachment of the source incarnation at resolution. It is absent if the source or attachment is absent or phased out. | Existing `This` resolver and attachment field; `TestPermanentBindingLifetime` |
| `Sacrifice(From(binding))` | The stored controller must still control the subject. Uses the existing sacrifice action and events. | Existing `sacrificeEffect.Apply` and `DoSacrifice`; `TestPermanentBindingLifetime` |

`ApplyEffect` opens a fresh binding frame and restores any outer frame on
return, including error returns. `Sequence`, composites, and pipelines share
that frame. `ResolutionState.Clone` copies it using immutable declaration keys
and value-only captures. `TestPermanentBindingNestedResolution` verifies that
nested resolution cannot read or erase an outer snapshot.

## Definition inspection and validation

The existing descriptor now records binding declarations, reads, and writes.
The visitor checks a read before it visits later producers. Only sequential
containers propagate assignments to later children. Branches, modes, loops,
and separate ability effects cannot export assignments to sibling expressions.
Each ability has its own diagnostic name and assignment scopes. Delayed
trigger definitions start a new scope and cannot read an outer resolution
snapshot. Explicit
branch-result merging is not implemented.

`TestPermanentBindingValidation` covers missing producers, reads before writes,
optional branches, zero-iteration loops, duplicate names, empty names, nil
bindings, and valid nested composition. Compile-fail fixtures reject a binding
declaration used directly as a subject and a condition used as a snapshot
subject. `TestSavedPermanentComposition` verifies the public DSL exports and
structural metadata.

## Retained helpers and remaining work

No legacy helper is removed in this foundation. Exact replacements are still
needed for these callers:

| Existing caller | Remaining adapter |
| --- | --- |
| Cocoon: `SnapshotAttached("host")` | Typed counter and continuous keyword effects |
| Erosion: `SnapshotAttached("land")` | Typed controller snapshot, payment, and destruction |
| Thrull Retainer: `SnapshotAttached("attached")` | Attachment capture before sacrifice cost; regeneration adapter. A resolution-time attachment lookup is not an exact substitute. |
| Sacrifice: `SacrificeGathered("t")` | Typed chosen-permanent binding and saved mana value |

Typed targets, target groups, modal scope, divided values, card and stack
references, copying and retargeting adapters, other saved-value domains,
optional-result consumers, and branch-result merging remain part of the next
work. Existing string-based helpers stay available until all their callers
have exact replacements. This change does not claim completion of the package
boundary or of the full composable DSL design.

## Verification

Passed after implementation:

- Focused binding, DSL, definition-validation, and compile-fail tests.
- `go test ./pkg/mage/... -short -count=1`.
- `go test ./... -short -count=1`.
- `go build ./...` and `go vet ./...`.
- `GOLANGCI_LINT_CACHE=/tmp/mage-composable-bindings-lint make lint`: zero issues.

Long simulations were not run.
