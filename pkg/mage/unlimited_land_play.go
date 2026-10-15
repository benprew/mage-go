package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// AllowUnlimitedLandPlaysFor removes one live player's land play limit.
// Timing, zones and prohibitions still apply.
func AllowUnlimitedLandPlaysFor(subject PlayerRef) ContinuousEffect {
	return &unlimitedLandPlayEffect{subject: subject}
}

type unlimitedLandPlayEffect struct {
	effectSource
	subject PlayerRef
}

func (*unlimitedLandPlayEffect) GetLayer() core.Layer                 { return core.LayerAbility }
func (*unlimitedLandPlayEffect) GetDuration() core.Duration           { return core.WhileOnBattlefield }
func (e *unlimitedLandPlayEffect) IsActive(g *Game) bool              { return g.FindPermanent(e.SourceID()) != nil }
func (*unlimitedLandPlayEffect) Text() string                         { return "play any number of lands during your turn" }
func (*unlimitedLandPlayEffect) Properties() EffectProperties         { return EffectProperties{} }
func (*unlimitedLandPlayEffect) ApplyContinuous(*EffectContext) error { return nil }
func (e *unlimitedLandPlayEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "unlimited-land-plays", Text: e.Text(), battlefieldSource: true, children: []definitionEdge{{"subject", e.subject}}}
}
func (e *unlimitedLandPlayEffect) cloneEffect() ContinuousEffect { clone := *e; return &clone }

// HasUnlimitedLandPlays reads a player's live land play permission.
// The permission does not change the number of lands already played.
func (g *Game) HasUnlimitedLandPlays(playerID uuid.UUID) bool {
	if g.GetPlayer(playerID) == nil {
		return false
	}
	if g.effects.Rules.UnlimitedLandPlays {
		return true
	}
	for _, effect := range g.effects.effects {
		inner := effect
		if printed, ok := inner.(*printedAbilityContinuousEffect); ok {
			inner = printed.ContinuousEffect
		}
		rule, ok := inner.(*unlimitedLandPlayEffect)
		if !ok || !effect.IsActive(g) {
			continue
		}
		source := g.FindPermanent(rule.SourceID())
		if source == nil || source.FaceDown {
			continue
		}
		frame := &EffectContext{Game: g, SourceID: source.ID(), Controller: source.ControllerID(), sourceIncarnation: source.incarnationID, sourceCaptured: true}
		id, ok := rule.subject.resolvePlayer(frame)
		if ok && id == playerID {
			return true
		}
	}
	return false
}
