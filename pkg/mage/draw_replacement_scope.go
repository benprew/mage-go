package mage

import (
	"log"
	"maps"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// DrawReplacementWindow selects the draws that one printed rule can replace.
type DrawReplacementWindow uint8

const (
	// ControllerDrawStep includes every draw during the controller's draw step.
	ControllerDrawStep DrawReplacementWindow = iota
	// ExceptFirstDrawStepDraw excludes the first actual draw in a player's draw step.
	ExceptFirstDrawStepDraw
)

type drawReplacementRule struct {
	window DrawReplacementWindow
	player *PlayerBinding
	body   Effect
}

// ReplaceDraw replaces a matching draw with an immediate body.
// The player binding is local to the body. Generated draws exclude this rule.
func ReplaceDraw(window DrawReplacementWindow, player *PlayerBinding, body Effect) StaticRule {
	return &drawReplacementRule{window, player, body}
}
func (*drawReplacementRule) staticRule() {}
func (r *drawReplacementRule) inspectDefinition() definitionNode {
	n := definitionNode{Role: "static-draw-replacement", bindingScope: true, children: []definitionEdge{
		{"body", Sequence(&eventPlayerCapture{binding: r.player}, r.body)},
	}}
	if r.window != ControllerDrawStep && r.window != ExceptFirstDrawStepDraw {
		n.problem = "supply a supported draw window"
	}
	return n
}

func (r *drawReplacementRule) staticEffects() []ContinuousEffect {
	return []ContinuousEffect{&drawReplacementContinuous{rule: *r, ruleID: uuid.New()}}
}

type drawReplacementContinuous struct {
	effectSource
	rule   drawReplacementRule
	ruleID uuid.UUID
}

func (*drawReplacementContinuous) Text() string                 { return "replace a card draw" }
func (*drawReplacementContinuous) Properties() EffectProperties { return EffectProperties{} }
func (*drawReplacementContinuous) GetLayer() core.Layer         { return core.LayerAbility }
func (*drawReplacementContinuous) GetDuration() core.Duration   { return core.WhileOnBattlefield }
func (e *drawReplacementContinuous) IsActive(g *Game) bool {
	p := g.FindPermanent(e.sourceID)
	return p != nil && !p.PhasedOut
}
func (e *drawReplacementContinuous) inspectDefinition() definitionNode {
	return e.rule.inspectDefinition()
}
func (e *drawReplacementContinuous) cloneEffect() ContinuousEffect { c := *e; return &c }
func (e *drawReplacementContinuous) ApplyContinuous(ctx *EffectContext) error {
	source := ctx.Game.FindPermanent(e.sourceID)
	if source != nil {
		ctx.Game.effects.AddCycleReplacement(&scopedDrawReplacement{
			replacementBase: replacementBase{sourceID: e.sourceID, duration: core.WhileOnBattlefield},
			incarnation:     source.incarnationID, rule: e.rule, ruleID: e.ruleID,
		})
	}
	return nil
}

type drawReplacementKey struct{ source, incarnation, rule uuid.UUID }

type scopedDrawReplacement struct {
	replacementBase
	incarnation uuid.UUID
	rule        drawReplacementRule
	ruleID      uuid.UUID
}

func (r *scopedDrawReplacement) key() drawReplacementKey {
	return drawReplacementKey{r.sourceID, r.incarnation, r.ruleID}
}

func (r *scopedDrawReplacement) Matches(a Action, reader GameReader) bool {
	draw, ok := a.(*DrawCardAction)
	if !ok || draw.excludedRules[r.key()] {
		return false
	}
	g, ok := reader.(*Game)
	if !ok {
		return false
	}
	source := g.FindPermanent(r.sourceID)
	if source == nil {
		return false
	}
	inOwnStep := g.turns.Step() == core.Draw && g.ActivePlayerObj().PlayerID() == draw.PlayerID()
	switch r.rule.window {
	case ControllerDrawStep:
		return inOwnStep && source.ControllerID() == draw.PlayerID()
	case ExceptFirstDrawStepDraw:
		return !inOwnStep || g.turns.drawStepDraws > 0
	}
	return false
}
func (r *scopedDrawReplacement) Replace(a Action, g *Game) Action {
	if !r.IsActive(g) || !r.Matches(a, g) {
		return a
	}
	draw := a.(*DrawCardAction)
	source := g.FindPermanent(r.sourceID)
	if source == nil {
		return a
	}
	outer := g.resolution
	defer func() { g.resolution = outer }()
	g.resolution = NewResolutionState()
	g.resolution.drawReplacementRules = maps.Clone(draw.excludedRules)
	if g.resolution.drawReplacementRules == nil {
		g.resolution.drawReplacementRules = make(map[drawReplacementKey]bool)
	}
	g.resolution.drawReplacementRules[r.key()] = true
	body := Sequence(&eventPlayerCapture{binding: r.rule.player, player: draw.PlayerID()}, r.rule.body)
	if err := ApplyEffect(g, body, r.sourceID, source.ControllerID(), nil); err != nil {
		log.Printf("Draw replacement failed: %v", err)
	}
	return nil
}
func (r *scopedDrawReplacement) IsActive(g GameReader) bool {
	source := g.FindPermanent(r.sourceID)
	if source == nil || source.PhasedOut || source.incarnationID != r.incarnation {
		return false
	}
	for _, ability := range source.RuntimeAbilities {
		if holder, ok := UnwrapAbility(ability).(*StaticAbilityHolder); ok {
			for _, effect := range holder.Effects {
				if rule, ok := effect.(*drawReplacementContinuous); ok && rule.ruleID == r.ruleID {
					return true
				}
			}
		}
	}
	return false
}
func (r *scopedDrawReplacement) Clone() ReplacementEffect { c := *r; return &c }

func (g *Game) drawAction(player uuid.UUID, normal bool) *DrawCardAction {
	action := NewDrawCardAction(uuid.Nil, player, normal)
	action.excludedRules = maps.Clone(g.resolution.drawReplacementRules)
	return action
}
