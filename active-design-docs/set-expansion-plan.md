# MTG Set Expansion Plan: Through Odyssey Block (+ Onslaught Stretch Goal)

## Current State

**Implemented sets:** Alpha (LEA), Arabian Nights (ARN), Antiquities (ATQ), Legends (LEG, partial)

**Engine capabilities:** 50+ pre-built effects, 20+ triggered ability constructors, 7-layer continuous effect system, 40+ keyword attrs, protection, banding, rampage, cumulative upkeep, coin flip, X spells, modal spells, regeneration, damage prevention/redirection, control change, clone/copy, counterspells, token creation, tutors, cantrips, mana conversion, card counters.

---

## Block-by-Block Breakdown

### 1. LEGENDS / THE DARK / FALLEN EMPIRES (Early Expansions)

| Set | Cards | Implementable Now | After Key Gaps |
|-----|-------|-------------------|----------------|
| Legends (LEG) | 310 | ~56% | ~70% |
| The Dark (DRK) | 119 | ~50% | ~70% |
| Fallen Empires (FEM) | 102 unique | ~54% | ~75% |
| **Total** | **531** | **~54%** | **~72%** |

**Already partially implemented:** Legends has card shells for all 310 cards (correct types, costs, subtypes) but zero functional abilities wired up.

**New keywords:** Rampage (already supported), Bands with Others (already supported)

**Key engine gaps needed:**

| Gap | Cards Unlocked | Effort | Notes |
|-----|----------------|--------|-------|
| `SacrificeMatchingCost(filter)` | ~15 | Low | Goblin Grenade, Horror of Horrors, etc. |
| "Attacks and isn't blocked" trigger (`EvtUnblocked`) | ~8 | Low | Farrel's Zealot, Necrite, Ophidian pattern |
| `CounterSpellUnlessPay(amount)` | ~8 | Low | Force Spike, Nether Void |
| Poison counters (player state + 10-poison SBA) | ~3 | Medium | Marsh Viper, Serpent Generator |
| `TapMatchingCreaturesCost(filter, n)` | ~4 | Low-Med | Hand of Justice, Vodalian War Machine |
| Mill effect `MillCards(n)` | ~3 now, many later | Low | Foundational for later sets |
| Hand reveal / choose from hand | ~5 | Low-Med | Amnesia, Rag Man |
| Exiled-card association ("exiled with" tracking) | ~4 | Medium | Knowledge Vault, Safe Haven |
| Blood Moon (land type override + mana ability) | 1 | Medium | Iconic card |

**Hardest cards:** Chains of Mephistopheles (draw replacement branching), Sylvan Library (track drawn cards), Takklemaggot (aura type-change on death), All Hallow's Eve (triggers from exile), Eureka (alternating player choices), Season of the Witch (track who attacked + who could attack), High Tide (mana production trigger).

**Overall difficulty: Medium-Hard.** Legends is massive with many bespoke designs. The Dark and Fallen Empires are smaller and more focused. The key blockers are small, high-ROI additions (sacrifice-matching, unblocked trigger, counter-unless-pay).

---

### 2. ICE AGE BLOCK (Ice Age / Homelands / Alliances)

| Set | Cards | Implementable Now | After Key Gaps |
|-----|-------|-------------------|----------------|
| Ice Age (ICE) | 383 | ~58% | ~75% |
| Homelands (HML) | 140 | ~72% | ~85% |
| Alliances (ALL) | 199 | ~57% | ~73% |
| **Total** | **722** | **~60%** | **~76%** |

**New keywords:** Cumulative Upkeep (already supported), Snow supertype (already exists)

**Key engine gaps needed:**

| Gap | Cards Unlocked | Effort | Notes |
|-----|----------------|--------|-------|
| Delayed draw at next upkeep | ~33 | Low | Ice Age cantrip pattern. `RegisterDelayedTrigger` on `EvtUpkeep` |
| Snow-permanent filter `HasSuperTypeFilter(Snow)` | ~18 | Low | All "if you control a snow land" cards |
| Alternative casting costs (pitch spells) | 6 | High | Force of Will, Contagion, Pyrokinesis. Needs alt-cost framework |
| `SacrificeMatchingCost(filter)` (same as above) | ~9 | Low | Zuran Orb, Glacial Crevasses |
| Divided damage distribution | ~7 | Medium | Fire Covenant, Pyrokinesis |
| Library manipulation (put cards from hand on top) | ~8 | Low-Med | Brainstorm, Portent |
| Non-mana cumulative upkeep | ~7 | Low-Med | Extend CU to arbitrary costs/effects |
| Zone-based abilities (hand/graveyard) | ~4 | Medium | Elvish Spirit Guide (mana from hand), Death Spark |
| Poison counters (same as above) | ~2 | Medium | Swamp Mosquito |

**Hardest cards:** Necropotence (skip draw + exile discard + end-step hand fill), Pox (proportional everything), Dance of the Dead (zone-changing enchant-graveyard aura), Illusions of Grandeur (20 life gain/loss), Ice Cauldron (store card+mana), Enduring Renewal (play with hand revealed + draw/death replacements), Force of Will (pitch spell framework), Helm of Obedience (repeating mill loop), Lim-Dul's Vault (iterative top-5 selection).

**Overall difficulty: Hard.** The delayed-cantrip pattern is high-ROI (33 cards). The alt-cost framework for Alliances pitch spells is the biggest infrastructure investment but pays forward through every subsequent block.

---

### 3. MIRAGE BLOCK (Mirage / Visions / Weatherlight)

| Set | Cards | Implementable Now | After Key Gaps |
|-----|-------|-------------------|----------------|
| Mirage (MIR) | 351 | ~62% | ~78% |
| Visions (VIS) | 167 | ~63% | ~80% |
| Weatherlight (WTH) | 167 | ~69% | ~82% |
| **Total** | **685** | **~64%** | **~80%** |

**New keywords:** FLANKING, PHASING

**Key engine gaps needed:**

| Gap | Cards Unlocked | Effort | Notes |
|-----|----------------|--------|-------|
| **Flanking** (attr + combat hook) | ~18 | Low | -1/-1 to non-flanking blockers. ~20 lines in combat step |
| **Phasing** (permanent state + untap step + visibility) | ~28 | High | Phased-out = invisible to everything. Major engine change |
| `SacrificeMatchingCost(filter)` (same) | ~15 | Low | "Sacrifice a Forest/Swamp/land" costs |
| Delayed cantrip (same as Ice Age) | ~13 | Low | Already needed for Ice Age block |
| "Attacks and isn't blocked" trigger (same) | ~10 | Low | Ophidian, Bone Dancer, Pygmy Hippo |
| "Becomes blocked" trigger `EvtBecameBlocked` | ~8 | Low | Brushwagg, Goblin Elite Infantry |
| Flash keyword | ~6 | Low | Benalish Knight, King Cheetah |
| Extra combat phase | 1 | Medium | Relentless Assault |
| Alternative costs (sacrifice permanents for mana) | ~3 | Medium | Fireblast, Spinning Darkness |

**Hardest cards:** Phyrexian Dreadnought (multi-sacrifice ETB), Taniwha (all lands phase out), Celestial Dawn (multi-layer color/type/mana override), Tombstone Stairwell (world enchantment + CU + per-player tokens + EOT destroy), Doomsday (exile library, choose 5), Necromancy (flash-aura reanimate), Sands of Time (untap step replacement), Teferi's Puzzle Box (draw step replacement), Bone Dancer (graveyard-order-matters).

**Overall difficulty: 8/10.** Flanking is easy and clean. Phasing is the single largest infrastructure investment in this block -- it touches battlefield visibility, combat, targeting, untap step, and aura/equipment handling. Most other gaps are shared with earlier blocks.

---

### 4. TEMPEST BLOCK (Tempest / Stronghold / Exodus)

| Set | Cards | Implementable Now | After Key Gaps |
|-----|-------|-------------------|----------------|
| Tempest (TMP) | 345 | ~62% | ~78% |
| Stronghold (STH) | 143 | ~65% | ~80% |
| Exodus (EXO) | 143 | ~58% | ~73% |
| **Total** | **631** | **~62%** | **~77%** |

**New keywords:** SHADOW, BUYBACK

**Key engine gaps needed:**

| Gap | Cards Unlocked | Effort | Notes |
|-----|----------------|--------|-------|
| **Shadow** (attr + combat restriction) | ~24 + 8 ref | Low | Symmetric evasion. Analogous to Fear/Menace in combat code |
| **Buyback** (optional cost + return to hand) | ~28 | Medium | Spell resolution hook. Various buyback costs (mana, life, sacrifice, discard) |
| `SacrificeLandCost()` | ~5 | Low | Constant Mists, Scorched Earth |
| Conditional targeting ("more X than you") | ~10 | Medium | Oath cycle, Keeper cycle |
| Sliver activated ability grants | ~8 | Low-Med | Extend `GrantActivatedAbilityToAll` |
| En-Kor damage redirection | 5 | Medium | Repeatable self-to-other redirection |
| Licids (creature <-> aura transformation) | 12 | Very High | Type mutation + attachment dynamics. Defer. |

**Hardest cards:** Humility (remove all abilities + set all P/T to 1/1, layer nightmare), Living Death (mass exile+sacrifice+reanimate), Recurring Nightmare (sac+return as enchantment activated ability), Mind Over Matter (discard-to-tap/untap), Survival of the Fittest (discard creature to tutor creature), Dream Halls (alt cost: discard same-color), Aluren (free flash casting for CMC<=3), Ensnaring Bridge (hand-size attack restriction), Volrath's Shapeshifter (graveyard-top text copying), Coffin Queen (graveyard reanimation with untap condition).

**Overall difficulty: Medium-High.** Shadow and Buyback are the two big wins -- implementing them alone jumps coverage from 62% to 70%. Licids should be deferred (very high complexity, only 12 cards total across the block).

---

### 5. URZA'S BLOCK (Urza's Saga / Urza's Legacy / Urza's Destiny)

| Set | Cards | Implementable Now | After Key Gaps |
|-----|-------|-------------------|----------------|
| Urza's Saga (USG) | 330 | ~48% | ~68% |
| Urza's Legacy (ULG) | 143 | ~55% | ~72% |
| Urza's Destiny (UDS) | 143 | ~53% | ~70% |
| **Total** | **616** | **~51%** | **~70%** |

**New keywords:** CYCLING, ECHO

**Key engine gaps needed:**

| Gap | Cards Unlocked | Effort | Notes |
|-----|----------------|--------|-------|
| **Cycling** (hand-zone ability) | ~51 | High | New ability zone concept. `WithCycling(cost)` CardOption |
| **Echo** (one-shot sacrifice-unless-pay at upkeep) | ~35 | Low-Med | Track "entered since last upkeep." Builds on existing CU |
| Dynamic mana production `AddDynamicMana(color, ValueSource)` | ~8 | Low | Tolarian Academy, Gaea's Cradle, Serra's Sanctum |
| Search library to battlefield `SearchLibraryToBattlefield(filter)` | ~8 | Low-Med | Tinker, Crop Rotation, Defense of the Heart |
| Sleeping enchantments (enchantment -> creature type change) | ~22 | Medium | Opal/Hidden/Veiled/Lurking cycles |
| Hand reveal / choose from hand | ~12 | Low-Med | Duress, Persecute, Ostracize |
| Man-land animation (temporary, keep land type) | 5 | Low | Faerie Conclave, Treetop Village cycle |
| Free spells (untap lands on resolution) | ~10 | Low | `UntapLands(n)` FuncEffect. Palinchron, Frantic Search |

**Hardest cards:** Yawgmoth's Will (play from graveyard + exile replacement), Opalescence (all enchantments become creatures), Replenish (mass reanimate enchantments), Time Spiral (shuffle hand+graveyard into library, draw 7, untap 6), Sneak Attack (cheat from hand + delayed sacrifice), Tolarian Academy (tap for blue per artifact), Academy Rector (death-search enchantment to battlefield), Contamination (mana replacement).

**Overall difficulty: High.** Cycling is the big infrastructure investment. Echo is moderate and pays off well (35 cards). The sleeping-enchantment type-change pattern is unique to this block and needs its own solution.

---

### 6. MASQUES BLOCK (Mercadian Masques / Nemesis / Prophecy)

| Set | Cards | Implementable Now | After Key Gaps |
|-----|-------|-------------------|----------------|
| Mercadian Masques (MMQ) | 350 | ~45% | ~70% |
| Nemesis (NEM) | 143 | ~40% | ~68% |
| Prophecy (PCY) | 144 | ~38% | ~65% |
| **Total** | **637** | **~42%** | **~69%** |

**New keywords:** FADING

**Key engine gaps needed:**

| Gap | Cards Unlocked | Effort | Notes |
|-----|----------------|--------|-------|
| Alternative cost framework (consolidated) | ~49 | High | Block-defining. Pitch, bounce-land, sacrifice-land, tap-creature, life-pay, free-if-condition |
| **Fading** (fade counters + upkeep remove + sacrifice) | ~15 | Low-Med | `WithFading(n)`. Clean counter-based mechanic |
| Search library to battlefield + CMC filter | ~20 | Medium | Rebel/Mercenary search chains |
| `EvtBecameBlocked` trigger (same) | ~20 | Low | Heavy "becomes blocked" theme in Masques |
| `SacrificeLandCost()` (same, critical for PCY) | ~25 | Low | Prophecy's land-sacrifice theme |
| Rhystic / "unless any player pays" framework | ~19 | Medium | Rhystic Study, Rhystic Lightning |
| Flash keyword (same) | ~8 | Low | Several auras and creatures |
| Mana value filters `HasManaValueLTE(n)` | ~10 | Low | CMC-based targeting/searching |
| Exile tracking with return-on-leaves | ~6 | Medium | Parallax Wave/Tide/Nexus |
| "No untapped lands" condition | ~10 | Low | Prophecy's Scoria Cat, Spur Grappler |

**Hardest cards:** Tangle Wire (complex tap-N escalating), Parallax Wave (exile-tracking + return), Rishadan Port (opponent's land tap), Lin Sivvi (X-cost Rebel search), Rising Waters (Stasis-like), Saproling Burst (fading + token P/T linked to counters), Accumulated Knowledge (cross-graveyard name counting), Chimeric Idol (animate + tap-all-lands cost).

**Overall difficulty: High.** The alternative cost framework is the critical unlock for this block -- 49 cards depend on it, and it was already needed for Alliances. Fading is clean and focused. The Rebel/Mercenary search chain is a unique pattern needing search-to-battlefield with CMC filtering.

---

### 7. INVASION BLOCK (Invasion / Planeshift / Apocalypse)

| Set | Cards | Implementable Now | After Key Gaps |
|-----|-------|-------------------|----------------|
| Invasion (INV) | 337 | ~38% | ~65% |
| Planeshift (PLS) | 143 | ~40% | ~68% |
| Apocalypse (APC) | 143 | ~42% | ~70% |
| **Total** | **623** | **~39%** | **~67%** |

**New keywords:** KICKER, DOMAIN

**Key engine gaps needed:**

| Gap | Cards Unlocked | Effort | Notes |
|-----|----------------|--------|-------|
| **Kicker** (single, optional additional cost) | ~40 | High | `WithKicker(cost)`, `WasKicked()` check. Block-defining |
| **Multi-kicker** ("and/or" kicker, Battlemages/Volvers) | ~11 | Medium | Extension of kicker for 2 independent kicker costs |
| **Domain** (`CountBasicLandTypes` ValueSource) | ~20 | Low | Count distinct {Plains,Island,Swamp,Mountain,Forest} subtypes |
| **Split cards** (dual-face spells) | 10 | High | Two spells on one card, choice during casting |
| Non-mana kicker costs (life, sacrifice, bounce) | ~8 | Medium | Extends kicker with arbitrary Cost objects |
| "Can't be countered" flag | ~6 | Low | `AttrCantBeCountered` |
| Search basic land to battlefield | ~5 | Low | `SearchBasicLandToBattlefield()` |
| Gating (ETB bounce by color) | ~12 | Low | FuncEffect-implementable |
| Fact or Fiction / pile separation | ~4 | High | New player choice interface |
| Scry | 1 now, many later | Low | `Scry(n)` effect. Foundational |

**Hardest cards:** Pernicious Deed (X-cost sacrifice to destroy CMC<=X), Spiritmonger (regen + color change + grows), Coalition Victory (alternate win condition), Phyrexian Altar (sacrifice for any-color mana), Captain Sisay (tutor legendary to hand), Fact or Fiction (pile division), Flagbearers (targeting restriction), Necravolver/Degavolver (multi-kicked creatures with complex abilities).

**Overall difficulty: High.** Kicker is the defining mechanic and needs solid infrastructure. Domain is trivial. Split cards need their own card representation system. The heavy multicolor theme works fine with existing mana cost parsing.

---

### 8. ODYSSEY BLOCK (Odyssey / Torment / Judgment)

| Set | Cards | Implementable Now | After Key Gaps |
|-----|-------|-------------------|----------------|
| Odyssey (ODY) | 345 | ~42% | ~72% |
| Torment (TOR) | 138 | ~38% | ~68% |
| Judgment (JUD) | 140 | ~40% | ~70% |
| **Total** | **623** | **~41%** | **~70%** |

**New keywords:** FLASHBACK, THRESHOLD, MADNESS

**Key engine gaps needed:**

| Gap | Cards Unlocked | Effort | Notes |
|-----|----------------|--------|-------|
| **Threshold** (`WhileThreshold()` ActiveCondition) | ~75 | Low | Check `len(graveyard) >= 7`. Simplest new mechanic, highest card count |
| **Flashback** (cast from graveyard + exile) | ~52 | High | Alternative casting zone. `WithFlashback(cost)`. Complex flashback costs (sacrifice, tap, exile) |
| **Madness** (discard replacement -> cast at madness cost) | ~10 | High | Hooks into discard pipeline. `WithMadness(cost)` |
| Mill effect `MillCards(n, selector)` | ~13 | Low | Trivial to implement, broadly needed |
| Nightmare ETB/LTB exile-return pattern | ~20 | Medium | Faceless Butcher, Mesmeric Fiend. Exiled-card tracking |
| Incarnations (graveyard-based continuous effects) | ~7 | Medium | `SourceInGraveyard` ActiveCondition + land-type check |
| Lhurgoyf P/T (count card type across ALL graveyards) | ~5 | Low | `PTEqualsAllGraveyardCount(filter)` |
| `SacrificeLandCost()` (same) | ~5 | Low | Already needed |
| Punisher/opponent-choice ("you choose: X or Y") | ~8 | Medium | Browbeat, Blazing Salvo, Book Burning |
| Wishes (search outside the game) | 5 | Medium | Sideboard/exile zone concept |
| Phantom damage prevention (counter removal replacement) | 6 | Medium | Per-creature damage replacement effect |
| Discard triggers `EvtCardDiscarded` | ~5 | Low | Spirit Cairn, Confessor, Pitchstone Wall |

**Hardest cards:** Mirari (copy spells for {3}), Upheaval (bounce everything), Wild Mongrel (discard to pump + color change), Psychatog (exile graveyard + discard for power), Cabal Coffers (mana per Swamp), Astral Slide (cycling trigger exile-return -- needs cycling too), Wonder/Anger/Genesis/Glory (graveyard-based ability grants), Wormfang Manta (skip turn), Battle of Wits (alternate win at 200+ library).

**Overall difficulty: Very High.** Three major new mechanics. Threshold is easy (75 cards!). Flashback is the big infrastructure investment. Madness is architecturally complex but only 10 cards. The graveyard-matters theme pervades every card and pushes existing graveyard infrastructure hard.

---

### 9. ONSLAUGHT BLOCK (Stretch Goal: Onslaught / Legions / Scourge)

| Set | Cards | Implementable Now | After Key Gaps |
|-----|-------|-------------------|----------------|
| Onslaught (ONS) | 335 | ~28% | ~60% |
| Legions (LGN) | 145 | ~30% | ~62% |
| Scourge (SCG) | 143 | ~30% | ~58% |
| **Total** | **623** | **~29%** | **~60%** |

**New keywords:** MORPH, STORM, AMPLIFY, PROVOKE, LANDCYCLING, DOUBLE STRIKE (first set appearance)

**Key engine gaps needed:**

| Gap | Cards Unlocked | Effort | Notes |
|-----|----------------|--------|-------|
| **Morph** (face-down casting + turn-face-up) | ~115 | Very High | Block-defining. Face-down 2/2 for {3}, morph cost to flip. Engine has `FaceDown` field but no morph casting |
| **Cycling** (same as Urza's, already needed) | ~60 | High | Plus cycling triggers (`EvtCycled`) and landcycling variant |
| Creature-type-matters (`ChooseCreatureType` + tribal counting) | ~70 | Medium | Heavy tribal block |
| **Storm** (copy spell N times on cast) | 12 | Medium | Spells-cast-this-turn counter + auto-copy |
| **Provoke** (force block + untap) | 8 | Medium | Attack trigger + forced-block declaration |
| **Amplify** (reveal from hand for +1/+1 counters) | 9 | Medium | Reveal-from-hand ETB mechanic |
| Flash keyword (same, already needed) | ~2 | Low | Quick Sliver, Caller of the Claw |
| Fight mechanic | 1 now, many later | Low | Foundational for future sets |

**Overall difficulty: Very High.** Morph alone is 115 cards and requires significant infrastructure (face-down representation, morph cost storage, turn-face-up as special action, morph triggers). Storm is iconic but only 12 cards. The tribal density is extreme.

---

## Consolidated Engine Feature Roadmap

Here's every new engine feature needed, ordered by cumulative impact across all blocks:

### Tier 1: Highest ROI (unlock 100+ cards each, needed by multiple blocks)

| Feature | Blocks Using It | Total Cards | Effort |
|---------|----------------|-------------|--------|
| **Alternative casting cost framework** | ALL (pitch), MIR (Fireblast), USG, MMQ (heavy), INV (kicker), ODY (flashback/madness) | 150+ | High |
| **Cycling** (hand-zone ability) | USG, ONS | ~110 | High |
| **Flashback** (cast from graveyard) | ODY block | ~52 | High |
| **Kicker** (optional additional cost) | INV block | ~55 | High |
| **Threshold** (graveyard count condition) | ODY block | ~75 | Low |

### Tier 2: High ROI (unlock 20-50 cards each)

| Feature | Blocks Using It | Total Cards | Effort |
|---------|----------------|-------------|--------|
| **Shadow** (evasion keyword) | TMP block | ~32 | Low |
| **Buyback** (return spell to hand) | TMP block | ~28 | Medium |
| **Echo** (one-shot upkeep sacrifice) | USG block | ~35 | Low-Med |
| **Flanking** (combat -1/-1 trigger) | MIR block | ~18 | Low |
| **Fading** (fade counters + sacrifice) | MMQ block | ~15 | Low-Med |
| Delayed draw at next upkeep (Ice Age cantrip) | ICE, MIR | ~46 | Low |
| `SacrificeMatchingCost(filter)` / `SacrificeLandCost()` | LEG, ICE, MIR, TMP, USG, MMQ, PCY, ODY | ~80+ | Low |
| `EvtUnblocked` / "attacks and isn't blocked" trigger | LEG, DRK, FEM, MIR, WTH, EXO, MMQ | ~30+ | Low |
| `EvtBecameBlocked` trigger | MIR, TMP, MMQ, ONS | ~30+ | Low |
| **Domain** (count basic land types) | INV block | ~20 | Low |
| Nightmare ETB/LTB exile-return pattern | ODY block | ~20 | Medium |

### Tier 3: Medium ROI (unlock 5-20 cards each)

| Feature | Blocks Using It | Total Cards | Effort |
|---------|----------------|-------------|--------|
| `CounterSpellUnlessPay(amount)` | LEG, ICE, MIR, TMP, all later | ~15+ | Low |
| Flash keyword (`AttrFlash`) | MIR, TMP, MMQ, ODY, INV, ONS | ~20 | Low |
| Mill effect `MillCards(n)` | LEG forward, heavy in ODY | ~20 | Low |
| `SearchLibraryToBattlefield(filter)` | USG, MMQ, INV, ODY | ~30 | Low-Med |
| Snow-permanent filter | ICE block | ~18 | Low |
| Hand reveal / choose from hand | LEG, ICE, USG, MMQ | ~20 | Low-Med |
| Divided damage distribution | ICE, ALL, MIR | ~10 | Medium |
| Scry | INV, all later | ~5 now | Low |
| "Can't be countered" flag | INV, ONS | ~8 | Low |
| Mana value filters `HasManaValueLTE/GTE` | MMQ, INV, ONS | ~15 | Low |
| Dynamic mana production (count-based) | USG, MMQ | ~10 | Low |
| Poison counters | LEG, DRK, ICE, HML, future | ~5 now | Medium |
| `ChooseCreatureType` | USG, MMQ, INV, ONS | ~15 | Low |

### Tier 4: Block-Specific or Deferred

| Feature | Block | Cards | Effort | Notes |
|---------|-------|-------|--------|-------|
| **Phasing** | MIR | ~28 | Very High | Major game state change |
| **Morph** | ONS | ~115 | Very High | Stretch goal block |
| **Madness** | ODY (TOR) | ~10 | High | Complex discard pipeline hook |
| **Storm** | ONS (SCG) | 12 | Medium | Stretch goal |
| **Split cards** | INV, APC | 10 | High | Dual-face representation |
| Sleeping enchantments | USG | ~22 | Medium | Type change: enchantment -> creature |
| Licids | TMP | 12 | Very High | Defer indefinitely |
| En-Kor redirection | TMP (STH) | 5 | Medium | Niche |
| Incarnations (GY continuous effects) | ODY (JUD) | 7 | Medium | `SourceInGraveyard` condition |
| Rebel/Mercenary search chains | MMQ | ~20 | Medium | Search-to-BF + CMC filter |
| Rhystic ("unless any player pays") | MMQ (PCY) | ~19 | Medium | Player choice framework |

---

## Recommended Implementation Order

### Phase 1: Quick Wins (unlock ~200 cards across all blocks)
Low effort, high impact. Can be done before tackling any specific block:

1. `SacrificeMatchingCost(filter)` / `SacrificeLandCost()` -- unlocks 80+ cards
2. `EvtUnblocked` trigger ("attacks and isn't blocked") -- unlocks 30+ cards
3. `EvtBecameBlocked` trigger -- unlocks 30+ cards
4. `CounterSpellUnlessPay(amount)` -- unlocks 15+ cards
5. `MillCards(n, selector)` -- unlocks 20+ cards, foundational
6. Flash keyword (`AttrFlash`) -- unlocks 20 cards
7. Delayed draw at next upkeep -- unlocks 46 cards (Ice Age + Mirage cantrips)
8. `SearchLibraryToBattlefield(filter)` -- unlocks 30+ cards
9. Scry(n) effect -- unlocks few now, foundational

### Phase 2: Block-Defining Keywords (unlock ~250 more cards)
Medium effort, each unlocks a specific block:

1. **Shadow** (Low effort) -- unlocks Tempest block
2. **Flanking** (Low effort) -- unlocks Mirage block
3. **Echo** (Low-Med effort) -- unlocks Urza's block
4. **Fading** (Low-Med effort) -- unlocks Masques block
5. **Buyback** (Medium effort) -- unlocks Tempest block
6. **Threshold** (Low effort) -- unlocks Odyssey block
7. **Domain** (Low effort) -- unlocks Invasion block

### Phase 3: Major Infrastructure (unlock ~300 more cards)
High effort, each is a significant engine change:

1. **Alternative casting cost framework** -- the biggest single unlock. Needed for pitch spells (ALL), free spells (MMQ), kicker, flashback, madness
2. **Cycling** (hand-zone ability) -- Urza's + Onslaught blocks
3. **Kicker** (optional additional cost) -- Invasion block
4. **Flashback** (cast from graveyard) -- Odyssey block

### Phase 4: Polish & Deferred
The remaining gaps that are either very complex or block-specific:

- Phasing (complex, 28 cards)
- Split cards (10 cards)
- Madness (complex, 10 cards)
- Sleeping enchantments (22 cards)
- Rebel/Mercenary search chains (20 cards)
- Rhystic framework (19 cards)
- Morph (stretch goal, 115 cards)
- Storm (stretch goal, 12 cards)
- Licids (defer indefinitely)

---

## Card Counts Summary

| Block | Sets | Total Cards | Implementable Now | After Phase 1-2 | After Phase 3 |
|-------|------|-------------|-------------------|-----------------|---------------|
| Early Expansions | LEG, DRK, FEM | 531 | ~54% | ~72% | ~80% |
| Ice Age | ICE, HML, ALL | 722 | ~60% | ~76% | ~82% |
| Mirage | MIR, VIS, WTH | 685 | ~64% | ~80% | ~85% |
| Tempest | TMP, STH, EXO | 631 | ~62% | ~77% | ~82% |
| Urza's | USG, ULG, UDS | 616 | ~51% | ~65% | ~78% |
| Masques | MMQ, NEM, PCY | 637 | ~42% | ~60% | ~75% |
| Invasion | INV, PLS, APC | 623 | ~39% | ~55% | ~72% |
| Odyssey | ODY, TOR, JUD | 623 | ~41% | ~55% | ~75% |
| **Through Odyssey** | **18 sets** | **5,068** | **~52%** | **~68%** | **~79%** |
| *Onslaught (stretch)* | *ONS, LGN, SCG* | *623* | *~29%* | *~40%* | *~65%* |
| **Through Onslaught** | **21 sets** | **5,691** | **~49%** | **~65%** | **~77%** |

**Bottom line:** With Phase 1 quick wins (~1 week of work), we jump from 52% to 68% across 5,000+ cards. Adding Phase 2 keywords (~2-3 weeks) gets to 79%. Phase 3 infrastructure (~4-6 weeks) pushes to 85%+, leaving only the gnarliest individual cards and deferred mechanics.
