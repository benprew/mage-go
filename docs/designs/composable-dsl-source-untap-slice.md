# Source-untap action slice

Status: complete. This family is part of later action slice 2. Other action
families remain open. Rules references: CR 400.7 (new objects), 608.2h
(resolution information), and 702.26 (phasing).

## Removed exports and semantic parity

| Removed export | Canonical expression | Existing executor |
| --- | --- | --- |
| `mage.UntapSource`, `dsl.UntapSource` | `dsl.Untap(dsl.This)` | Typed `UntapPermanent` adapter, target untap executor, `Game.UntapPermanent` |

The private source-only executor is removed. The typed subject uses the stored
source incarnation. A source that leaves and returns cannot be untapped by its
old ability. An absent or phased-out source has no effect. Control changes and
phasing out and back in preserve the reference. Positional targets do not select
the subject. Stun counters and untap events retain the existing action behavior.
The definition stores no mutable resolution state; cloned games act independently.

The source-return regression failed with the old constructor before migration.
The existing typed adapter passes it without a new engine execution path.
Diagnostics now include the subject, and metadata identifies untapping as a
benefit, consistent with the existing target-untap instruction. Registration
checks the subject and rejects a spell-only source context.

## Caller inventory

| File | Caller | Clause retained |
| --- | --- | --- |
| `cards/limited/artifacts.go` | Basalt Monolith | {3}: Untap Basalt Monolith |
| `cards/limited/enchantments.go` | Instill Energy | Untap the host of the granted ability; existing cost and once-per-turn limit |
| `cards/antiquities/creatures.go` | Colossus of Sardia | {9}: Untap this creature; existing upkeep restriction |
| `cards/jumpstart/creatures.go` | Chained Brute | Untap this creature; existing mana, sacrifice, and turn restrictions |
| `cards/thedark/creatures.go` | Eater of the Dead | Untap this creature after the existing conditional exile instruction |
| `pkg/mage/tapped_for_mana_test.go` | Non-mana activation fixture | Tap cost followed by source untap |

All production expressions compile through `dsl`. These are expression
migrations, not a new audit of unrelated clauses in those cards. Existing costs,
conditions, grants, and ability builders remain. Tutorial references now use the
canonical expression.

## Evidence

- `TestComposableSourceUntapLifetime`: pending trigger resolution after source
  absence, return, phasing, and control changes; cloned stack isolation.
- `TestComposableSourceUntapStunAndEvents`: stun replacement, exactly one untap
  event on a state change, and an unrelated positional target stays tapped.
- `TestComposableSourceUntapValidation`: subject diagnostics, benefit metadata,
  nil subject rejection, and spell-only source rejection.
- `TestTypedSourceUntapComposition`: public activation, trigger, sequence,
  saved-value, and modal composition.
- `untap_player.go` and `untap_collection.go`: compile-time subject rejection.
- Existing Basalt Monolith, Instill Energy, Colossus of Sardia, Eater of the Dead,
  tapped-for-mana, and complete affected card-package tests.

## Remaining families

`UntapTarget` and `UntapTargetStep` retain positional-target callers and are not
removed. `TapAttachedCreature` remains: migration needs a separate audit of
last-known attachment information when the Aura leaves before its trigger
resolves (CR 608.2h). The current `AttachedPermanent` adapter reads a live source;
this slice does not certify that behavior for departed Auras. No damage, target,
attachment, or replacement subsystem is rewritten here.

## Verification

- Focused source-untap, tap-for-mana, public composition, and compile-fail tests passed.
- `go test ./pkg/mage/... -short -count=1` passed.
- `go test ./... -short -count=1` and `go test ./... -count=1` passed, including
  registered-definition validation, all card packages, and long AI simulations.
- `go build ./...` and `go vet ./...` passed.
- `make lint` passed with zero issues. The sandbox blocked optional cache
  writes. The unrelated Secrets of Strixhaven import-format change was reverted.
- `git diff --check` passed. No Go callers or definitions of the removed helper remain.
