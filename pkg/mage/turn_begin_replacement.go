package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type beginTurnAction struct{ playerID uuid.UUID }

func (*beginTurnAction) ActionSource() uuid.UUID { return uuid.Nil }
func (a *beginTurnAction) PlayerID() uuid.UUID   { return a.playerID }

// MaySkipYourTurnToUntap replaces the controller's turn if the player agrees.
// The saved permanent untaps first in the next turn that occurs (CR 614.10b).
func MaySkipYourTurnToUntap(subject PermanentRef, condition StateCondition) StaticRule {
	return turnSkipUntapRule{subject: subject, condition: condition}
}

type turnSkipUntapRule struct {
	subject   PermanentRef
	condition StateCondition
}

func (turnSkipUntapRule) staticRule() {}
func (r turnSkipUntapRule) inspectDefinition() definitionNode {
	n := definitionNode{Role: "static-rule", Text: "may skip your turn to untap a permanent", battlefieldSource: true, children: []definitionEdge{{"subject", r.subject}, {"condition", r.condition}}}
	switch r.subject.(type) {
	case sourceRef, attachedPermanentRef:
	default:
		n.problem = "supply This or AttachedPermanent"
	}
	return n
}
func (r turnSkipUntapRule) staticEffects() []ContinuousEffect {
	return []ContinuousEffect{&turnSkipUntapEffect{funcContinuousEffect: FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil }).(*funcContinuousEffect), rule: r, ruleID: uuid.New()}}
}

type turnSkipUntapEffect struct {
	*funcContinuousEffect
	rule   turnSkipUntapRule
	ruleID uuid.UUID
}

func (e *turnSkipUntapEffect) inspectDefinition() definitionNode { return e.rule.inspectDefinition() }
func (e *turnSkipUntapEffect) cloneEffect() ContinuousEffect {
	c := *e
	inner := *e.funcContinuousEffect
	c.funcContinuousEffect = &inner
	return &c
}
func (e *turnSkipUntapEffect) ApplyContinuous(ctx *EffectContext) error {
	source := ctx.Game.FindPermanent(e.SourceID())
	if source != nil {
		ctx.Game.effects.AddCycleReplacement(&turnSkipUntapReplacement{replacementBase: replacementBase{sourceID: source.ID(), duration: core.WhileOnBattlefield}, incarnation: source.incarnationID, rule: e.rule, ruleID: e.ruleID})
	}
	return nil
}

type turnSkipUntapReplacement struct {
	replacementBase
	incarnation, ruleID uuid.UUID
	rule                turnSkipUntapRule
}

func (r *turnSkipUntapReplacement) IsActive(g GameReader) bool {
	source := g.FindPermanent(r.sourceID)
	if source == nil || source.PhasedOut || source.incarnationID != r.incarnation {
		return false
	}
	for _, ability := range source.RuntimeAbilities {
		if holder, ok := UnwrapAbility(ability).(*StaticAbilityHolder); ok {
			for _, effect := range holder.Effects {
				if candidate, ok := effect.(*turnSkipUntapEffect); ok && candidate.ruleID == r.ruleID {
					return true
				}
			}
		}
	}
	return false
}
func (r *turnSkipUntapReplacement) Matches(action Action, g GameReader) bool {
	a, ok := action.(*beginTurnAction)
	if !ok || !r.IsActive(g) {
		return false
	}
	source := g.FindPermanent(r.sourceID)
	return source.ControllerID() == a.playerID && r.rule.condition.checkState(g, StateContext{SourceID: source.ID(), SourceIncarnation: source.incarnationID, ControllerID: source.ControllerID()})
}
func (r *turnSkipUntapReplacement) Replace(action Action, g *Game) Action {
	if !r.Matches(action, g) {
		return action
	}
	source := g.FindPermanent(r.sourceID)
	subject := resolveStatePermanent(g, StateContext{SourceID: source.ID(), SourceIncarnation: source.incarnationID, ControllerID: source.ControllerID()}, r.rule.subject)
	if !g.GetPlayer(source.ControllerID()).ChooseMayAbility("skip this turn to untap " + source.Name()) {
		return action
	}
	if subject != nil {
		g.turns.pendingTurnUntaps = append(g.turns.pendingTurnUntaps, capturedPermanent{subject.ID(), subject.incarnationID})
	}
	return nil
}
func (r *turnSkipUntapReplacement) Clone() ReplacementEffect { c := *r; return &c }
func (*turnSkipUntapReplacement) Text() string               { return "may skip this turn to untap" }

type scheduledTurnSkipReplacement struct{ replacementBase }

func (*scheduledTurnSkipReplacement) IsActive(GameReader) bool { return true }
func (*scheduledTurnSkipReplacement) Matches(action Action, g GameReader) bool {
	a, ok := action.(*beginTurnAction)
	return ok && g.(*Game).turns.Schedule().SkipNextTurnFor[a.playerID] > 0
}
func (r *scheduledTurnSkipReplacement) Replace(action Action, g *Game) Action {
	if !r.Matches(action, g) {
		return action
	}
	g.turns.ConsumeTurnSkip(action.(*beginTurnAction).playerID)
	return nil
}
func (r *scheduledTurnSkipReplacement) Clone() ReplacementEffect { c := *r; return &c }
func (*scheduledTurnSkipReplacement) Text() string               { return "skip the next turn" }

// TryBeginTurn applies turn replacements before any action of the new turn.
// Drivers must use this before they run a turn. Repeated calls are safe.
func (g *Game) TryBeginTurn() bool {
	player := g.ActivePlayerObj().PlayerID()
	if g.turns.attemptedTurn == g.CurrentTurn() && g.turns.attemptedTurnPlayer == player {
		return !g.turns.turnSkipped
	}
	g.turns.attemptedTurn = g.CurrentTurn()
	g.turns.attemptedTurnPlayer = player
	g.ApplyContinuousEffects()
	action := &beginTurnAction{playerID: player}
	result, _ := g.effects.prepareReplacements(action, g, &scheduledTurnSkipReplacement{})
	g.turns.turnSkipped = result == nil
	if result == nil {
		return false
	}
	g.BeginActiveTurn()
	return true
}

func (g *Game) applyPendingTurnUntaps() {
	pending := g.turns.pendingTurnUntaps
	g.turns.pendingTurnUntaps = nil
	for _, saved := range pending {
		subject := g.FindPermanent(saved.id)
		if subject != nil && subject.incarnationID == saved.incarnation {
			g.UntapPermanent(subject)
		}
	}
}
