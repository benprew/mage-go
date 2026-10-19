package mage

import "github.com/google/uuid"

// TargetPowerLessThanOrEqualToSourcePower compares current power values.
// An absent source supplies its retained power at resolution.
func TargetPowerLessThanOrEqualToSourcePower() PermanentTargetOption {
	return targetPowerLESourceOption{}
}

type targetPowerLESourceOption struct{}

func (targetPowerLESourceOption) permanentTargetOption() {}
func (targetPowerLESourceOption) applyPermanentTarget(t *PermanentTarget) {
	t.powerLESourcePower = true
}
func (targetPowerLESourceOption) inspectDefinition() definitionNode {
	return definitionNode{Role: "target-source-power", Text: "power no greater than source power", battlefieldSource: true}
}

// TargetToughnessLessThanSourcePower checks current target toughness against
// source power. An absent source uses its retained power at resolution.
func TargetToughnessLessThanSourcePower() PermanentTargetOption {
	return targetToughnessLessThanSourcePowerOption{}
}

type targetToughnessLessThanSourcePowerOption struct{}

func (targetToughnessLessThanSourcePowerOption) permanentTargetOption() {}
func (targetToughnessLessThanSourcePowerOption) applyPermanentTarget(t *PermanentTarget) {
	t.toughnessLessThanSourcePower = true
}
func (targetToughnessLessThanSourcePowerOption) inspectDefinition() definitionNode {
	return definitionNode{Role: "target-source-power", Text: "toughness below source power", battlefieldSource: true}
}

func targetSourcePower(card Card, g *Game) (int, bool) {
	if card == nil {
		return 0, false
	}
	if retained, ok := card.(targetSourceLKI); ok && retained.powerKnown {
		return retained.power, true
	}
	if retained, ok := card.(targetSourceLKI); ok {
		if retained.spell || retained.incarnation == uuid.Nil {
			return 0, false
		}
		if p := g.FindPermanent(card.ID()); p != nil && !p.PhasedOut && p.incarnationID == retained.incarnation {
			return p.CurrentPower(g), true
		}
		return 0, false
	}
	if p := g.FindPermanent(card.ID()); p != nil && !p.PhasedOut {
		return p.CurrentPower(g), true
	}
	return 0, false
}

func sourceRelativeTargetMatches(permanent *Permanent, source Card, g *Game, toughnessLess, powerLE bool) (matches, known bool) {
	if !toughnessLess && !powerLE {
		return true, true
	}
	power, known := targetSourcePower(source, g)
	if !known {
		return false, false
	}
	return (!toughnessLess || permanent.CurrentToughness(g) < power) &&
		(!powerLE || permanent.CurrentPower(g) <= power), true
}

// TargetManaValueEqualX checks current mana value against the announced X.
func TargetManaValueEqualX() PermanentTargetOption { return targetManaValueEqualXOption{} }

type targetManaValueEqualXOption struct{}

func (targetManaValueEqualXOption) permanentTargetOption()                  {}
func (targetManaValueEqualXOption) applyPermanentTarget(t *PermanentTarget) { t.manaValueEqualX = true }
func (targetManaValueEqualXOption) inspectDefinition() definitionNode {
	return definitionNode{Role: "target-mana-value", Text: "mana value equals announced X"}
}

func permanentTargetManaValueMatches(permanent *Permanent, x int) bool {
	return permanent.Card.ManaCost().CMC() == x
}

// DeclareSpellTargetManaValueX checks a spell's mana value against the announced X.
// The target spell's own X contributes to its mana value.
func DeclareSpellTargetManaValueX(name string, filters ...CardFilter) *SpellTargetBinding {
	target := DeclareSpellTarget(name, filters...)
	target.spec.(*typedSpellTarget).manaValueEqualX = true
	return target
}

func (g *Game) possibleTargetsWithX(target Target, controller uuid.UUID, source Card, x int) []uuid.UUID {
	// Target queries must not change the current resolution frame.
	query := *g
	query.resolution.SetX(x)
	return target.Possible(controller, source, &query)
}
