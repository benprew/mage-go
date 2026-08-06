# Fallen Empires — Remaining Work

This inventory includes unimplemented and partially implemented cards. A reprint is considered implemented when its current Oracle behavior is fully registered by another card package.

## Unimplemented cards

### Artifacts

- **Delif's Cone**
- **Delif's Cube**

### Creatures

- **Farrelite Priest**
- **Hand of Justice**
- **Icatian Skirmishers**
- **Seasinger**
- **Vodalian War Machine**
- **Derelor**
- **Ebon Praetor**
- **Initiates of the Ebon Hand**
- **Thrull Wizard**
- **Dwarven Armorer**
- **Goblin Chirurgeon**
- **Goblin Flotilla**
- **Orcish Spy**
- **Elvish Scout**
- **Thelonite Druid**

### Enchantments

- **Farrel's Mantle**
- **Goblin Kites**
- **Heroism**
- **Merseine**
- **Raiding Party**
- **Thelon's Chant**
- **Thelon's Curse**
- **Tidal Flats**
- **Tidal Influence**
- **Tourach's Chant**
- **Tourach's Gate**

### Land

- **Rainbow Vale**

### Instants and sorceries

- **Dwarven Catapult**
- **High Tide**
- **Soul Exchange**
- **Spore Cloud**

## Confirmed engine blockers

- **Dwarven Armorer** needs a `+0/+1` counter type. The existing counter model has `+1/+0`, `+1/+1`, and several other P/T counters, but not `+0/+1`.
- Costs such as “Sacrifice a Goblin” or “Sacrifice a creature” need to permit sacrificing the ability's source when it matches. `SacrificeMatchingCost` currently excludes the source, blocking exact implementations of **Goblin Chirurgeon**, **Thelonite Druid**, and similar abilities.
- **Hand of Justice** needs a cost that taps three matching untapped permanents; `TapMatchingCost` currently selects only one.
- **Derelor** needs a colored `{B}` spell-cost increase. The existing color-filtered cost modifier adds generic mana instead.

Other cards remain pending because their complete Oracle behavior needs multi-object choices, duration tracking, combat-specific prevention or assignment changes, conditional mana payments, library inspection, or delayed control changes not currently exposed by the card DSL.
