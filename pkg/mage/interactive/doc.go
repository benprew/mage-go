/*
Package interactive provides the interactive player layer for the MTG engine:
human players driven by a TUI over channels, computer players driven by
pluggable heuristic strategies, position evaluation, and game-loop runners
for both single-player (human vs. AI) and multiplayer (human vs. human over SSH)
sessions.

# Human Players

[HumanPlayer] wraps [mage.BasePlayer] and routes every game decision through a
pair of buffered channels. The game loop writes a [GameMsg] (board snapshot +
prompt + legal options) to [HumanPlayer.ToTUI]; the TUI reads it, renders the
state, collects input, and sends a [PriorityAction] back on [HumanPlayer.FromTUI].
Modal choices (mode selection, permanent selection, discard, color choice) use a
separate [ChoiceRequest]/[ChoiceResponse] channel pair.

	hp := interactive.NewHumanPlayer("Alice")
	// For the SSH lobby, allocate channels first and wire them to the TUI model:
	hp := interactive.NewHumanPlayerWithChannels("Alice", toTUI, fromTUI, reqs, resps)

# AI Players

[AIPlayer] wraps [mage.BasePlayer] with a pluggable [AIStrategy] that makes all
decisions without channels. Three strategy hooks must be implemented:

  - PriorityAction — called when the AI has priority; returns what to cast,
    which land to play, which ability to activate, or pass.
  - Attackers — returns the IDs of creatures to attack with this turn.
  - Blockers — returns blocker→attacker assignments.

[HeuristicStrategy] is the built-in implementation. On its main-phase turn it
picks the highest-[spellValue] non-instant from the castable spells, plays a
land if one is available, and optionally holds instants for the opponent's turn.
Target selection in [HeuristicStrategy.autoSelectTargets] uses [evalCreature]
scores and lethal-first [ThreatPerMana] ordering.

# Personalities

[Personality] parameters tune [HeuristicStrategy] behavior. Five presets are
provided:

  - [AggroPersonality] — cheapest spells first, attack with everything,
    block only large threats (power ≥ 3).
  - [ControlPersonality] — most-expensive spells first, hold instants,
    attack only when profitable, block everything.
  - [MidrangePersonality] — most-expensive spells first, attack aggressively,
    block large threats.
  - [TempoPersonality] — cheapest spells first, hold instants, attack
    aggressively, block medium threats.
  - [BurnPersonality] — cheapest spells first, never block, always aim damage
    at the opponent's face.

Constructor helpers: [NewAIPlayer] (Midrange), [NewAggroAI], [NewControlAI],
[NewTempoAI], [NewBurnAI].

# Composite Strategies

[AdaptiveStrategy] wraps an Aggressive and a Defensive sub-strategy and switches
between them based on [DefaultEvaluator]: when the AI's position score is ≥ 0
it plays aggressively; when behind it plays defensively.

[SequentialStrategy] tries each sub-strategy in order and uses the first one
that produces a non-pass priority action (or non-empty attacker/blocker list).

Constructor: [NewAdaptiveAI] (AggroPersonality when ahead, ControlPersonality
when behind).

# Position Evaluation

[StateEvaluator] is a function type:

	type StateEvaluator func(g GameReader, playerID uuid.UUID) int

Higher scores are better for playerID. [DefaultEvaluator] is the standard
implementation; it sums five components:

  1. Life advantage: (myLife − oppLife) × [LifeWeight]
  2. Creature board: for each creature, [evalCreature] score — positive for own,
     negative for opponent's.
  3. Non-creature permanents: CMC / [NonCreatureCMCDiv] per permanent.
  4. Card advantage: (myHand − oppHand) × [CardWeight].
  5. Mana development: own land count × [LandWeight], plus +1 per untapped land.

[evalCreature] scores a creature from its base P/T (via counters and
[BasePTOverride], without querying continuous effects), applies a ×2/3 discount
for tapped creatures and ÷2 for summoning-sick creatures, then adds keyword and
ability bonuses. All 31 keywords are weighted — evasion (Flying +4, Fear +3,
Menace/Trample/landwalk +2, UnblockableKW +5), combat (DoubleStrike +4,
FirstStrike/Haste +2), durability (Indestructible +5, Hexproof +3), utility
(Vigilance/Lifelink +2), and drawbacks (Defender/DoesNotUntap −2, MustAttack −1).
Mana abilities score +3 (any-color) or +2 (fixed); other activated and triggered
abilities score +1 each.

[ThreatPerMana] divides an [evalCreature] score by CMC and is used for lethal
target prioritisation.

[GameReader] is the narrow interface [DefaultEvaluator] accepts, allowing the
evaluator to be tested independently of a full [mage.Game]:

	type GameReader interface {
	    GetPlayer(uuid.UUID) mage.Player
	    GetOpponent(uuid.UUID) mage.Player
	    FilterBattlefield(mage.PermanentFilter) []*mage.Permanent
	    CountBattlefield(mage.PermanentFilter) int
	}

# Game Loops

[RunGameLoop] drives a single human-vs-AI game in its own goroutine. It owns the
turn structure (untap → upkeep → draw → main → combat → main → end), calls
[GetAvailableActions] to populate the TUI's action menu, executes [PriorityAction]
decisions from both sides, resolves the stack on consecutive passes, and supports
one level of undo during the human's main phase. The loop closes [HumanPlayer.ToTUI]
on game over so the TUI can detect termination.

[RunMultiplayerGameLoop] drives a human-vs-human game over two independent
[PlayerChannels], one per player. It follows the same turn structure but has no AI
logic, no undo, and handles player disconnection gracefully.

# Snapshots and Available Actions

[SnapshotGameState] produces a [GameState] value — a fully serialisable, read-only
view of the current position from one player's perspective — suitable for sending
over the TUI channel or SSH connection.

[GetAvailableActions] returns the legal [ActionOption] list for a player at a
given point in the turn (main-phase vs. priority window).
*/
package interactive
