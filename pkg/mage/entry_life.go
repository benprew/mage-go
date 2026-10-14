package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// PayLifeUpToAsEnters saves a legal life payment before battlefield entry.
// The entry controller chooses. Return requires a new payment.
func PayLifeUpToAsEnters(maximum ResolutionValue) Ability {
	return &entryLifeAbility{BaseAbility: BaseAbility{id: uuid.New(), abilityType: core.AbilityStatic}, maximum: maximum}
}

type entryLifeAbility struct {
	BaseAbility
	maximum ResolutionValue
}

func (a *entryLifeAbility) inspectDefinition() definitionNode {
	return definitionNode{Role: "entry-life-payment", Text: "pay life as enters", battlefieldSource: true, children: []definitionEdge{{"maximum", a.maximum}}}
}

func (a *entryLifeAbility) applyEntry(g *Game, p *Permanent) {
	p.EntryLifePaid = 0
	player := g.GetPlayer(p.ControllerID())
	if player == nil {
		return
	}
	ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: p.ControllerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
	maximum := max(0, min(a.maximum.resolveValue(ctx), player.Life()))
	amount := validNumericChoice(player, 0, maximum)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: p.ControllerID(), SourceID: p.ID(), Costs: []Cost{LifePayCost(amount)}})
	if err != nil || tx.commit() != nil {
		return
	}
	p.EntryLifePaid = amount
}

type entryLifeReplacement struct {
	replacementBase
	ability *entryLifeAbility
}

func (*entryLifeReplacement) IsActive(GameReader) bool { return true }
func (*entryLifeReplacement) Text() string             { return "pay life as enters" }
func (*entryLifeReplacement) Matches(action Action, _ GameReader) bool {
	_, ok := action.(*battlefieldEntryAction)
	return ok
}
func (r *entryLifeReplacement) Clone() ReplacementEffect { c := *r; return &c }
func (r *entryLifeReplacement) Replace(action Action, g *Game) Action {
	r.ability.applyEntry(g, action.(*battlefieldEntryAction).permanent)
	return action
}

// PTEqualsEntryLifePaid defines P/T from this visit's saved life payment.
// Cards in other zones have no entry result and supply zero.
func PTEqualsEntryLifePaid() ContinuousEffect {
	return &countCharacteristicPT{power: true, toughness: true, entryLife: true}
}

// EntryLifePaidOf reads a current or last-known entry payment.
// Returned objects cannot supply the saved visit.
func EntryLifePaidOf(subject PermanentRef) ResolutionValue { return entryLifePaidValue{subject} }

type entryLifePaidValue struct{ subject PermanentRef }

func (entryLifePaidValue) Text() string { return "life paid as enters" }
func (v entryLifePaidValue) inspectDefinition() definitionNode {
	return permanentInformationNode(v.subject, "permanent-value", v.Text())
}
func (v entryLifePaidValue) resolveValue(ctx *EffectContext) int {
	p, lki := readPermanentInformation(v.subject, ctx)
	if lki != nil {
		p = lki.Snapshot
	}
	if p == nil {
		return 0
	}
	return p.EntryLifePaid
}
