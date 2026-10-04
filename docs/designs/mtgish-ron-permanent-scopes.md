# RON unary permanent reference scopes

Status: P0-SCOPE-PERMANENT-UNARY complete. Other permanent scopes remain partial.

## Exact supported shapes and contexts

`DestroyPermanent/1`, `DestroyPermanentNoRegen/1`, `TapPermanent/1`,
`UntapPermanent/1`, `RegeneratePermanent/1`, and `SacrificePermanent/1` accept
one symbol reference in an `ActionList/1`. The list belongs to `SpellActions/1`,
`Activated/2`, or the existing supported `TriggerA/2` contexts. Costs retain
their separate compiler and payment contracts.

| Symbol | Scope | Engine reference |
| --- | --- | --- |
| `Ref_TargetPermanent` | One `TargetPermanent/1` in a single-element `Targeted/2` declaration | Fresh `DeclarePermanentTarget` for this action instance |
| `ThisPermanent` | Permanent source in an activation or trigger | `mage.This`, with the resolving source incarnation |
| `HostPermanent` | Aura source in an activation or trigger | `mage.AttachedPermanent` |

The target filter retains existing supported forms. Unbound, wrong-type,
numbered, event, result, and tuple-form references are rejected at compilation.
A non-Aura cannot read a host. An instant or sorcery cannot read a permanent
source. A declaration in another rule cannot supply a reference. Nested and
multiple target declarations remain rejected.

`permanentTargetPlan` has immutable lexical identity. `permanentReferencePlan`
holds a source/host kind or that target identity. `instanceBindings.permanents`
maps the identity to a fresh engine declaration per factory/action build.
The plan stores no chosen objects. Engine definition inspection sees the target
read and detects missing declarations. Copies and clones use the existing stack
binding services; there is no mutable chosen state in the compiler plan.

## Execution and reuse

Lowering uses `mage.Destroy`, `DestroyWithoutRegeneration`, `TapSubject`,
`UntapPermanent`, `Regenerate`, and `Sacrifice`. The shared engine owns target
legality, source capture, attachments, regeneration, destruction replacements,
zone changes, cost transactions, and clone isolation. No engine rule changed.

Targets retain announcement-time incarnation identity. An illegal or phased-out
target is absent; phase-in before resolution retains identity. Source actions
cannot follow a returned card. Host reads use the current attachment of a live
source at resolution. A departed source uses its retained attachment. A returned
source cannot redirect its old ability to its new attachment, and a returned or
phased-out host is absent. Thrull Retainer uses this retained attachment after
its sacrifice cost. Sacrifice acts only on a subject the resolving controller
controls. Regeneration can act on the other player's creature.

Relevant contracts are CR 113.7a, 400.7, 608.2b/c/h/n, and 702.26b/d/g. These
adapters introduce no saved characteristic query, simultaneous batch, or new
replacement ordering. They preserve the existing engine services for those rules.

## Boundaries

Damage sources/recipients, spell and union targets, temporary layer selectors,
static host lowering, event bindings, numbered groups, saved results, wrappers,
and player life actions remain separate slices. Production registry cutover and
Go-card/DSL removal gates remain open. This document establishes compiler
behavior only; it does not expand the engine API.

## Complete-record audit

The full nested inventory was inspected for all six constructor names, including
cost and unknown-role occurrences. Costs were distinguished from actions.
Complete position-free records were read for the following 92 cards with direct
action occurrences (identical reprints share those records):

Ali Baba, Ball Lightning, Basalt Monolith, Blight, Clay Statue, Cocoon, Crumble, Dance of Many, Dandân, Death Ward, Deep Spawn, Delif's Cube, Demonic Hordes, Desert Twister, Detonate, Diabolic Machine, Disenchant, Disharmony, Divine Offering, Draconian Cylix, Drop of Honey, Drowned, Drudge Skeletons, Dwarven Demolition Team, Ebony Horse, Elephant Graveyard, Elvish Scout, Exorcist, Fasting, Feral Thallid, Fissure, Flood, Ghost Ship, Giant Shark, Goblin Chirurgeon, Goblin Digging Team, Goblins of the Flarg, Hand of Justice, Homarid Shaman, Homarid Warrior, Horror of Horrors, Ice Storm, Icy Manipulator, Island Fish Jasconius, Jandor's Saddlebags, Jihad, King Suleiman, Kudzu, Ley Druid, Living Wall, Mana Vortex, Maze of Ith, Merchant Ship, Merfolk Assassin, Miracle Worker, Niall Silvain, Northern Paladin, Paralyze, Phyrexian Gremlins, Pirate Ship, Ragnar, Ramses Overdark, Regeneration, Relic Barrier, Royal Assassin, Savaen Elves, Scavenger Folk, Scavenging Ghoul, Sea Serpent, Seasinger, Sedge Troll, Serendib Djinn, Shatter, Sinkhole, Spinal Villain, Stone Rain, Strip Mine, Tangle Kelp, Telekinesis, Terror, Tetsuo Umezawa, The Abyss, The Brute, Thrull Retainer, Tunnel, Uthden Troll, Venarian Gold, Vodalian Knights, Walking Dead, Wall of Bone, Wall of Brambles, Will-o'-the-Wisp.

43 of these unique cards compile after this slice:

Ali Baba, Clay Statue, Death Ward, Desert Twister, Diabolic Machine, Disenchant, Drowned, Drudge Skeletons, Dwarven Demolition Team, Elephant Graveyard, Exorcist, Fissure, Flood, Ghost Ship, Goblin Digging Team, Homarid Shaman, Ice Storm, Icy Manipulator, Jandor's Saddlebags, King Suleiman, Ley Druid, Living Wall, Niall Silvain, Northern Paladin, Ragnar, Regeneration, Relic Barrier, Royal Assassin, Scavenger Folk, Shatter, Sinkhole, Spinal Villain, Stone Rain, Strip Mine, Terror, The Brute, Thrull Retainer, Tunnel, Uthden Troll, Walking Dead, Wall of Bone, Wall of Brambles, Will-o'-the-Wisp.

Other records remain rejected for surrounding rules, filters, costs, events,
or later instructions. Reading their action shapes does not establish support
for the entire card. No RON source was corrected.
