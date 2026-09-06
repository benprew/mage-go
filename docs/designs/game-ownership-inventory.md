# Game Field & Method Ownership Inventory

This document represents the Phase 0 deliverable of the Game Subsystem Refactor (`docs/game-subsystem-refactor.md`). It inventories all fields and responsibilities of `mage.Game`, assigning each to its authoritative owner subsystem or aggregate coordination.

## Field Inventory & Target Subsystems

| Field | Type | Target Subsystem | Notes / Lifetimes |
|---|---|---|---|
| `currentX` | `int` | `ResolutionState` | X value of resolving spell or activated ability |
| `currentMode` | `int` | `ResolutionState` | Chosen mode (0-indexed) for resolving modal spell |
| `currentEventAmount` | `int` | `ResolutionState` | Triggering event amount (e.g. damage) |
| `currentEventSourceID` | `uuid.UUID` | `ResolutionState` | Source ID of triggering event |
| `resolvingCard` | `Card` | `ResolutionState` | Card currently resolving from stack |
| `resolvingColorSourceID` | `uuid.UUID` | `ResolutionState` | Source ID for resolving color override |
| `resolvingColorOverride` | `*[]Color` | `ResolutionState` | Color override during stack resolution |
| `resolvingCastZone` | `Zone` | `ResolutionState` | Zone resolving spell was cast from |
| `resolvingCastContext` | `*CastContext` | `ResolutionState` | Cast-time snapshot (CR 608.2g) |
| `resolvingTargets` | `[]uuid.UUID` | `ResolutionState` | Targets of resolving spell/ability |
| `resolvingDamageDistribution`| `map[uuid.UUID]int` | `ResolutionState` | Divided damage distribution |
| `resolvingCounterDistribution`| `map[uuid.UUID]int` | `ResolutionState` | Counter distribution |
| `lastSacrificedID` | `uuid.UUID` | `ResolutionState` | Most recently sacrificed permanent ID |
| `lastCostReveal` | `Card` | `ResolutionState` | Most recently revealed card for cost |
| `lastExiledCard` | `Card` | `ResolutionState` | Most recently exiled card for cost/effect |
| `discardCountThisTurn` | `map[uuid.UUID]int` | `TrackerSystem.Turn` | Turn tracker: discards |
| `lifeGainedThisTurn` | `map[uuid.UUID]int` | `TrackerSystem.Turn` | Turn tracker: life gained |
| `permDamageReceivedThisTurn`| `map[uuid.UUID]int` | `TrackerSystem.Turn` | Turn tracker: damage received |
| `attackedOrBlockedThisTurn` | `map[uuid.UUID]bool` | `TrackerSystem.Turn` | Turn tracker: creature attacked/blocked |
| `playerCastSpellThisTurn` | `map[uuid.UUID]bool` | `TrackerSystem.Turn` | Turn tracker: player cast spell |
| `playerAttackedThisTurn` | `map[uuid.UUID]bool` | `TrackerSystem.Turn` | Turn tracker: player attacked |
| `cardsDrawnThisTurn` | `map[uuid.UUID]int` | `TrackerSystem.Turn` | Turn tracker: cards drawn |
| `cardsLeftGraveyardThisTurn`| `map[uuid.UUID]int` | `TrackerSystem.Turn` | Turn tracker: cards left graveyard |
| `cardsPutIntoExileThisTurn` | `int` | `TrackerSystem.Turn` | Turn tracker: exile count |
| `exileZoneChangesPending` | `map[uuid.UUID]int` | `TrackerSystem.Turn` | Turn tracker: pending exile count |
| `landsPlayedThisTurn` | `int` | `TrackerSystem.Turn` | Turn tracker: land play count |
| `extraLandPlaysThisTurn` | `map[uuid.UUID]int` | `TrackerSystem.Turn` | Turn tracker: extra land plays |
| `optionalCostPaid` | `map[uuid.UUID]bool`| `TrackerSystem.Turn` | Turn tracker: optional costs paid |
| `damageTakenThisTurn` | `map[uuid.UUID]int` | `TrackerSystem.Turn` | Turn tracker: player damage |
| `artifactDamageTakenThisTurn`| `map[uuid.UUID]int` | `TrackerSystem.Turn` | Turn tracker: artifact damage |
| `attackedThisTurn` | `map[uuid.UUID]bool`| `TrackerSystem.Turn` | Turn tracker: attackers |
| `attackedLastTurn` | `map[uuid.UUID]map[uuid.UUID]bool` | `TrackerSystem.Turn` | Turn tracker: previous turn attackers |
| `blockedThisTurn` | `map[uuid.UUID][]uuid.UUID` | `TrackerSystem.Turn` | Turn tracker: blocker assignments |
| `instantsCastThisTurn` | `map[uuid.UUID]int` | `TrackerSystem.Turn` | Turn tracker: instants cast |
| `sorceriesCastThisTurn` | `map[uuid.UUID]int` | `TrackerSystem.Turn` | Turn tracker: sorceries cast |
| `creatureDeathsThisTurn` | `int` | `TrackerSystem.Turn` | Turn tracker: total creature deaths |
| `untappedLandsAtTurnStart` | `map[uuid.UUID]int` | `TrackerSystem.Turn` | Turn tracker: untapped lands snapshot |
| `timesTargetedThisTurn` | `map[uuid.UUID]int` | `TrackerSystem.Turn` | Turn tracker: target counts |
| `cleanupPriorityRounds` | `int` | `TrackerSystem.Turn` | Turn tracker: cleanup priority rounds |
| `duelSpellsCastByColor` | `map[uuid.UUID]map[Color]int` | `TrackerSystem.Duel` | Duel tracker: spells by color |
| `duelSpellsCastByType` | `map[uuid.UUID]map[CardType]int` | `TrackerSystem.Duel` | Duel tracker: spells by card type |
| `duelLandsPlayed` | `map[uuid.UUID]int` | `TrackerSystem.Duel` | Duel tracker: total lands played |
| `duelAttackersDeclared` | `map[uuid.UUID]int` | `TrackerSystem.Duel` | Duel tracker: total attackers declared |
| `duelCreatureDeaths` | `map[uuid.UUID]int` | `TrackerSystem.Duel` | Duel tracker: own creature deaths |
| `duelNonCombatDamage` | `map[uuid.UUID]int` | `TrackerSystem.Duel` | Duel tracker: noncombat damage to opponent |
| `coinFlipResults` | `[]bool` | `RandomSource` | Scripted coin flip queue |
| `randomResults` | `[]int` | `RandomSource` | Scripted random int queue |
| `pendingTriggers` | `[]*pendingTrigger`| `TriggerSystem` | Pending triggers awaiting stack |
| `armedStateTriggers` | `map[stateTriggerKey]bool` | `TriggerSystem` | CR 603.8 state triggers arming state |
| `delayedTriggers` | `[]*DelayedTrigger`| `TriggerSystem` | Active delayed triggers |
| `manaScratch` | `[]manaSourceInfo` | `ManaSystem` | Scratch buffer for mana solver |
| `damageDealtBy` | `map[uuid.UUID]map[uuid.UUID]bool` | `DamageSystem` | Damage tracking: target -> sources |
| `damageDealtToPlayersByPermanent` | `map[uuid.UUID]map[uuid.UUID]bool` | `DamageSystem` | Long-term damage tracking |
| `damageDealtToPermanentsByPermanent` | `map[uuid.UUID]map[uuid.UUID]bool` | `DamageSystem` | Long-term damage tracking |
| `combatDamageThisStep` | `map[uuid.UUID]map[uuid.UUID]int` | `DamageSystem` | Per-step combat damage aggregation |
| `combatDamageSourcesThisStep` | `map[uuid.UUID]map[uuid.UUID]map[uuid.UUID]int` | `DamageSystem` | Per-step damage breakdown |
| `resolvingCombatDamage` | `bool` | `DamageSystem` | Combat damage execution flag |
| `onDamageDealt` | `func(...)` | `DamageSystem` | Damage callback |
| `battlefield` | `[]*Permanent` | `ZoneSystem` | Battlefield permanents |
| `battlefieldShared` | `bool` | `ZoneSystem` | Copy-on-write clone flag |
| `battlefieldSliceShared` | `bool` | `ZoneSystem` | Copy-on-write slice header flag |
| `ownedPermanents` | `map[uuid.UUID]struct{}` | `ZoneSystem` | Copy-on-write private permanents |
| `exile` | `[]ExiledCard` | `ZoneSystem` | Exile zone storage & visibility |
| `enteringPermanent` | `*Permanent` | `ZoneSystem` | Permanent entering battlefield (ETB) |
| `lki` | `map[uuid.UUID]*PermanentLKI` | `ZoneSystem` | Last-Known-Information map |
| `castFromExilePermissions` | `[]CastableFromExilePermission` | `ZoneSystem` | Exile permissions |
| `exileInsteadCards` | `map[uuid.UUID]uuid.UUID` | `ZoneSystem` | Exile replacement tags |
| `turn` | `int` | `TurnSystem` | Turn counter |
| `step` | `PhaseStep` | `TurnSystem` | Current step / phase |
| `activePlayer` | `int` | `TurnSystem` | Active player index |
| `extraTurns` | `[]uuid.UUID` | `TurnSystem` | Extra turns queue |
| `schedule` | `*TurnSchedule` | `TurnSystem` | Turn phase/step schedule & skips |
| `skipNextUntap` | `map[uuid.UUID]int` | `TurnSystem` | Per-permanent untap skip counters |
| `players` | `[]Player` | `Game` Coordinator | Aggregate root player list |
| `anteEnabled`, `originalOwners`, `anteResult`, `anteSettled` | Various | `Game` Ante Domain | Ante rules orchestration |
| `stack` | `*Stack` | `Game` Coordinator | Stack aggregate |
| `combat` | `*Combat` | `Game` Coordinator | Combat aggregate |
| `effects` | `*EffectManager` | `Game` Coordinator | Continuous/replacement effects |
| `layer2Controllers` | `map[uuid.UUID]uuid.UUID` | `Game` Coordinator | Layer 2 control continuous effects |
| `onPriority`, `afterPriorityAction`, `beforeStackResolve` | `func(...)` | `Game` Presentation | Presentation callbacks |
| `stopped` | `bool` | `Game` Coordinator | Game flow control |
| `millModifiers`, `lifeGainModifiers` | `[]modifier` | `Game` Coordinator | Continuous amount modifiers |
| `customState` | `map[string]any` | `Game` Extensibility | Arbitrary set metadata |

---

## Phase Plan & Boundaries (Completed)

1. **Phase 1: ResolutionState** [Done]: Extract `resolution_state.go` with scoped Begin/End lifecycle and deep cloning.
2. **Phase 2: TrackerSystem** [Done]: Extract `tracker_system.go` with `TurnTrackers` and `DuelTrackers`.
3. **Phase 3: RandomSource** [Done]: Extract `random_source.go` with deterministic RNG queue.
4. **Phase 4: TriggerSystem** [Done]: Extract `trigger_system.go` with pending, delayed, and armed state triggers.
5. **Phase 5: ManaSystem** [Done]: Extract `mana_system.go` with scratch state, discovery, and payment planning.
6. **Phase 6: DamageSystem** [Done]: Extract `damage_system.go` with aggregation, history, and prevention.
7. **Phase 7: ZoneSystem** [Done]: Extract `zone_system.go` with battlefield COW, exile, and LKI.
8. **Phase 8: TurnSystem** [Done]: Extract `turn_system.go` with schedule, active player, and extra turns.
9. **Phase 9: Cleanup & Facades** [Done]: Shrink compatibility surfaces, split `GameReader` into domain readers, update documentation and architectural models.
