package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// LoseLifeAsEnters causes the entry controller to lose life before entry.
// Each entry reads its amount after earlier replacements.
func LoseLifeAsEnters(amount ResolutionValue) Ability {
	return &entryLifeLossAbility{BaseAbility: BaseAbility{id: uuid.New(), abilityType: core.AbilityStatic}, amount: amount}
}

type entryLifeLossAbility struct {
	BaseAbility
	amount ResolutionValue
}

func (a *entryLifeLossAbility) inspectDefinition() definitionNode {
	return definitionNode{Role: "entry-life-loss", Text: "lose life as enters", battlefieldSource: true, children: []definitionEdge{{"amount", a.amount}}}
}

type entryLifeLossReplacement struct {
	replacementBase
	ability *entryLifeLossAbility
}

func (*entryLifeLossReplacement) IsActive(GameReader) bool { return true }
func (*entryLifeLossReplacement) Text() string             { return "lose life as enters" }
func (*entryLifeLossReplacement) Matches(action Action, _ GameReader) bool {
	_, ok := action.(*battlefieldEntryAction)
	return ok
}
func (r *entryLifeLossReplacement) Clone() ReplacementEffect { c := *r; return &c }
func (r *entryLifeLossReplacement) Replace(action Action, g *Game) Action {
	p := action.(*battlefieldEntryAction).permanent
	if player := g.GetPlayer(p.ControllerID()); player != nil {
		ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: p.ControllerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
		g.PlayerLoseLife(player, r.ability.amount.resolveValue(ctx))
	}
	return action
}
