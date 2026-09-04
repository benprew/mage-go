/*
Package interactive provides player interfaces for the MTG engine.

It supports human players through a terminal user interface (TUI) with Go
channels. It supports computer players through configurable heuristic
strategies. The package also provides position evaluation and game-loop
runners for single-player and multiplayer sessions.

# Human Players

[HumanPlayer] embeds [mage.BasePlayer]. It sends game decisions through
buffered channels.

The game loop sends a [GameMsg] value to [HumanPlayer.ToTUI]. A [GameMsg]
contains a board snapshot, a prompt, and legal options. The TUI reads the
message, displays the game state, and collects user input. The TUI then sends a
[PriorityAction] back on [HumanPlayer.FromTUI].

Modal choices (such as mode selection, permanent selection, discard, or color
choice) use the [ChoiceRequest] and [ChoiceResponse] channel pair.

	hp := interactive.NewHumanPlayer("Alice")
	// For the SSH lobby, allocate channels first and wire them to the TUI model:
	hp := interactive.NewHumanPlayerWithChannels("Alice", toTUI, fromTUI, reqs, resps)

# AI Players

[AIPlayer] wraps [mage.BasePlayer] with an [AIStrategy] implementation. The AI
makes all decisions without channels.

You must implement three strategy methods:

  - PriorityAction — returns a spell to cast, a land to play, an ability to
    activate, or a pass.
  - Attackers — returns creature identifiers that attack this turn.
  - Blockers — returns blocker assignments to attacking creatures.

[HeuristicStrategy] is the default implementation. During its main phase, it
selects the non-instant spell with the highest [spellValue]. It plays a land if
available. It can hold instant spells until the turn of the opponent.
[HeuristicStrategy.autoSelectTargets] selects targets by using [evalCreature]
scores and [ThreatPerMana] priority.

# Personalities

[WeightedPersonality] defines continuous weights for AI decisions. Evaluation
weights (LifeWeight, BoardWeight, CardWeight, ManaWeight, TempoWeight)
configure [WeightedEvaluator]. Decision weights (Aggression, BlockThreshold,
HoldInstants, TargetFace, CurvePreference) use continuous values from 0.0
through 1.0. These weights control combat, targeting, and spell-casting
choices.

The package provides five preset configurations:

  - [AggroWeighted] — High board weight, maximum aggression, low blocking, and
    lowest spell cost first.
  - [ControlWeighted] — High life and card weights, zero aggression, holds
    instants, blocks all attackers, and highest spell cost first.
  - [MidrangeWeighted] — Balanced weights, 0.7 aggression, and highest spell
    cost first.
  - [TempoWeighted] — High tempo weight, 0.8 aggression, and 0.7 hold-instants
    weight.
  - [BurnWeighted] — Maximum aggression, zero blocking, and always targets the
    opponent.

The boolean [Personality] type remains for backward compatibility. Use
[Personality.ToWeighted] to convert [Personality] to [WeightedPersonality].
Preset variables such as [AggroPersonality] and [ControlPersonality] convert
automatically when used with [HeuristicStrategy].

Constructor helpers: [NewAIPlayer] (Midrange), [NewAggroAI], [NewControlAI],
[NewTempoAI], [NewBurnAI], and [NewWeightedAI].

# Composite Strategies

[AdaptiveStrategy] contains an aggressive strategy and a defensive strategy.
It selects between them based on [DefaultEvaluator]. When the position score is
zero or greater, it uses the aggressive strategy. When the score is negative,
it uses the defensive strategy.

[SequentialStrategy] evaluates multiple sub-strategies in order. It executes
the first sub-strategy that produces an action other than a pass.

Constructor helper: [NewAdaptiveAI] (uses AggroPersonality when ahead and
ControlPersonality when behind).

# Position Evaluation

[StateEvaluator] is a function type:

	type StateEvaluator func(g *mage.Game, playerID uuid.UUID) int

A higher score represents a better position for playerID. [DefaultEvaluator]
uses constant weights. [WeightedEvaluator] returns a [StateEvaluator]
configured by the evaluation weights of a [WeightedPersonality].

[DefaultEvaluator] adds five components:

 1. Life difference: (myLife − oppLife) × [LifeWeight]
 2. Creature board: sum of [evalCreature] scores (positive for own creatures,
    negative for opponent creatures).
 3. Non-creature permanents: CMC / [NonCreatureCMCDiv] for each permanent.
 4. Card difference: (myHand − oppHand) × [CardWeight].
 5. Mana development: own land count × [LandWeight], plus 1 per untapped land.

[evalCreature] calculates a score from base power and toughness. It includes
counters and [BasePTOverride]. It does not query continuous effects. It
applies a 2/3 multiplier for tapped creatures. It divides the score by 2 for
creatures with summoning sickness. It then adds bonuses for keywords and
abilities.

The function evaluates all 31 keywords:

  - Evasion: Flying (+4), Fear (+3), Menace (+2), Trample (+2), landwalk (+2),
    UnblockableKW (+5).
  - Combat: DoubleStrike (+4), FirstStrike (+2), Haste (+2).
  - Durability: Indestructible (+5), Hexproof (+3).
  - Utility: Vigilance (+2), Lifelink (+2).
  - Penalties: Defender (-2), DoesNotUntap (-2), MustAttack (-1).

Mana abilities add +3 (any color) or +2 (fixed color). Other activated and
triggered abilities add +1 each.

[ThreatPerMana] divides an [evalCreature] score by CMC. The AI uses this value
to prioritize lethal targets.

[StateEvaluator] accepts a [*mage.Game] directly.

# Game Loops

[RunGameLoop] runs a game between a human player and an AI player in a
goroutine. It manages the turn structure: untap, upkeep, draw, main, combat,
main, and end. It calls [GetAvailableActions] to populate the TUI menu. It
executes [PriorityAction] choices from both players. It resolves stack objects
when both players pass in sequence. It supports one undo step during the main
phase of the human player. When the game ends, the loop closes
[HumanPlayer.ToTUI].

[RunMultiplayerGameLoop] runs a game between two human players with separate
[PlayerChannels]. It follows the same turn structure. It contains no AI logic
and no undo support. It safely handles player disconnection.

# Snapshots and Available Actions

[SnapshotGameState] returns a read-only [GameState] value for one player. You
can serialize this value and send it over channels or SSH connections.

[GetAvailableActions] returns a list of legal [ActionOption] values for a
player during the current turn phase.
*/
package interactive
