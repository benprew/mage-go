package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// DrawInsteadOfLifeGain replaces positive life gain with that many card draws.
// The subject uses the current source controller. Each draw can be replaced.
func DrawInsteadOfLifeGain(subject PlayerRef) ContinuousEffect {
	return &lifeGainDrawContinuous{subject: subject, ruleID: uuid.New()}
}

type lifeGainDrawContinuous struct {
	effectSource
	subject PlayerRef
	ruleID  uuid.UUID
}

func (*lifeGainDrawContinuous) Text() string                 { return "draw cards instead of gaining life" }
func (*lifeGainDrawContinuous) Properties() EffectProperties { return EffectProperties{} }
func (*lifeGainDrawContinuous) GetLayer() core.Layer         { return core.LayerAbility }
func (*lifeGainDrawContinuous) GetDuration() core.Duration   { return core.WhileOnBattlefield }
func (e *lifeGainDrawContinuous) IsActive(g *Game) bool      { return g.FindPermanent(e.SourceID()) != nil }
func (e *lifeGainDrawContinuous) inspectDefinition() definitionNode {
	return definitionNode{Role: "static-life-gain-draw-replacement", Text: e.Text(), battlefieldSource: true, children: []definitionEdge{{"subject", e.subject}}}
}
func (e *lifeGainDrawContinuous) cloneEffect() ContinuousEffect { c := *e; return &c }
func (e *lifeGainDrawContinuous) ApplyContinuous(ctx *EffectContext) error {
	source := ctx.Game.FindPermanent(e.SourceID())
	if source != nil && !source.PhasedOut {
		ctx.Game.effects.AddCycleReplacement(&lifeGainDrawReplacement{
			replacementBase: replacementBase{sourceID: source.ID(), duration: core.WhileOnBattlefield},
			incarnation:     source.incarnationID, ruleID: e.ruleID, subject: e.subject,
		})
	}
	return nil
}

type lifeGainDrawReplacement struct {
	replacementBase
	incarnation, ruleID uuid.UUID
	subject             PlayerRef
}

func (r *lifeGainDrawReplacement) context(g GameReader) *EffectContext {
	source := g.FindPermanent(r.sourceID)
	if source == nil || source.PhasedOut || source.incarnationID != r.incarnation {
		return nil
	}
	for _, ability := range source.RuntimeAbilities {
		if holder, ok := UnwrapAbility(ability).(*StaticAbilityHolder); ok {
			for _, effect := range holder.Effects {
				if rule, ok := effect.(*lifeGainDrawContinuous); ok && rule.ruleID == r.ruleID {
					return &EffectContext{Game: g.(*Game), SourceID: source.ID(), Controller: source.ControllerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
				}
			}
		}
	}
	return nil
}
func (r *lifeGainDrawReplacement) IsActive(g GameReader) bool { return r.context(g) != nil }
func (r *lifeGainDrawReplacement) Matches(action Action, g GameReader) bool {
	gain, ok := action.(*LifeGainAction)
	if !ok || gain.Amount() <= 0 {
		return false
	}
	ctx := r.context(g)
	if ctx == nil {
		return false
	}
	player, ok := r.subject.resolvePlayer(ctx)
	return ok && player == gain.PlayerID()
}
func (r *lifeGainDrawReplacement) Replace(action Action, g *Game) Action {
	if !r.Matches(action, g) {
		return action
	}
	gain := action.(*LifeGainAction)
	player := g.GetPlayer(gain.PlayerID())
	if player == nil {
		return action
	}
	for range gain.Amount() {
		g.PlayerDrawCard(player)
	}
	return nil
}
func (r *lifeGainDrawReplacement) Clone() ReplacementEffect { c := *r; return &c }
func (*lifeGainDrawReplacement) Text() string               { return "draw cards instead of gaining life" }
