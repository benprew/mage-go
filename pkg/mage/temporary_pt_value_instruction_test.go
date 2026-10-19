package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTemporaryPTValueInstructionDeclaration(t *testing.T) {
	target := DeclarePermanentTarget("subject", AnyPermanent)
	adjustments := []PTValueAdjustment{{Power: ValueAtResolution(XValue()), Toughness: Fixed(0)}}
	effect := (TemporaryPTValueInstruction{Subject: target, Adjustments: adjustments}).Effect()
	adjustments[0].Power = Fixed(99)
	got, ok := InspectTemporaryPTValueInstruction(effect)
	if !ok || got.Subject != target || !pureFixedOrXAmount(got.Adjustments[0].Power) {
		t.Fatal("lost native operands")
	}
	if _, ok := got.Adjustments[0].Power.(legacyResolutionValue); !ok {
		t.Fatal("input list aliases declaration")
	}
	got.Adjustments[0].Power = Fixed(88)
	again, _ := InspectTemporaryPTValueInstruction(effect)
	if _, ok := again.Adjustments[0].Power.(legacyResolutionValue); !ok {
		t.Fatal("inspection aliases declaration")
	}
	for _, instruction := range []TemporaryPTValueInstruction{
		{}, {Subject: target}, {Adjustments: adjustments},
		{Subject: (*PermanentTargetBinding)(nil), Adjustments: adjustments},
		{Subject: target, Adjustments: []PTValueAdjustment{{Power: Fixed(1)}}},
		{Subject: target, Adjustments: []PTValueAdjustment{{Toughness: Fixed(1)}}},
		{Subject: target, Adjustments: []PTValueAdjustment{{Power: (*damageInspectionValueTrap)(nil), Toughness: Fixed(1)}}},
	} {
		if instruction.Effect() != nil {
			t.Fatal("accepted incomplete declaration")
		}
	}
	for _, candidate := range []Effect{nil, (*temporaryPTValueEffect)(nil), Until(core.EndOfTurn, ModifyPT(target, Fixed(1), Fixed(1))), Sequence(effect, destructionInspectionTrap{})} {
		if _, ok := InspectTemporaryPTValueInstruction(candidate); ok {
			t.Fatal("accepted undeclared tree")
		}
	}
}

func TestTemporaryPTValueInstructionInstallation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Subject", "", 3, 6)
	p := g.PutOnBattlefield(card, a.PlayerID())
	effect := (TemporaryPTValueInstruction{Subject: This, Adjustments: []PTValueAdjustment{
		{Power: PowerOfWithLKI(This), Toughness: Fixed(0)},
		{Power: Fixed(1), Toughness: SubtractValues(Fixed(1), ToughnessOfWithLKI(This))},
	}}).Effect()
	if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if p.CurrentPower(g) != 7 || p.CurrentToughness(g) != 1 || len(g.effects.effects) != 1 {
		t.Fatal("adjustments did not capture together in one installation")
	}
	clone := g.Clone()
	g.PhaseIn(g.PhaseOut(p.ID()))
	if g.FindPermanent(p.ID()).CurrentPower(g) != 7 {
		t.Fatal("phase cycle lost installation")
	}
	g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if returned.CurrentPower(g) != 3 || returned.CurrentToughness(g) != 6 {
		t.Fatal("layer followed another visit")
	}
	if clone.FindPermanent(p.ID()).CurrentPower(clone) != 7 {
		t.Fatal("clone followed original movement")
	}
	clone.GetEffects().RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if clone.FindPermanent(p.ID()).CurrentPower(clone) != 3 || clone.FindPermanent(p.ID()).CurrentToughness(clone) != 6 {
		t.Fatal("layer survived cleanup")
	}
}

func TestTemporaryPTValueInstructionInspection(t *testing.T) {
	for _, amount := range []string{"fixed", "x", "power", "toughness", "mana", "arithmetic", "foreign value", "opaque value", "opaque legacy", "invalid arithmetic"} {
		for _, order := range []string{"alone", "prefix", "suffix", "unknown suffix", "all illegal", "partial illegal", "foreign subject", "duplicate", "clone", "copy"} {
			t.Run(amount+"/"+order, func(t *testing.T) {
				g, p, obj, target := targetQueryFixture(t)
				other := g.PutOnBattlefield(NewLand("Other"), obj.Controller)
				otherTarget := DeclarePermanentTarget("other", HasCardTypeFilter(core.TypeLand))
				obj.TargetSpecs = append(obj.TargetSpecs, otherTarget.targetSpec())
				obj.Targets = append(obj.Targets, other.ID())
				obj.TargetZones[other.ID()] = core.ZoneBattlefield
				obj.targetIncarnations[other.ID()] = other.incarnationID
				value := ResolutionValue(Fixed(0))
				known, want := true, false
				switch amount {
				case "x":
					value = ValueAtResolution(XValue())
				case "power":
					value = PowerOfWithLKI(target)
				case "toughness":
					value = ToughnessOfWithLKI(target)
				case "mana":
					value = ManaValueOf(target)
				case "arithmetic":
					value = MultiplyValues(Fixed(-1), SubtractValues(ToughnessOfWithLKI(target), Fixed(1)))
				case "foreign value":
					value = PowerOfWithLKI(DeclarePermanentTarget("foreign", AnyPermanent))
					known = false
				case "opaque value":
					value = damageInspectionValueTrap{}
					known = false
				case "opaque legacy":
					value = ValueAtResolution(targetQueryValueTrap{})
					known = false
				case "invalid arithmetic":
					value = arithmeticValue{left: Fixed(1), right: Fixed(1), operation: 255}
					known = false
				}
				instruction := TemporaryPTValueInstruction{Subject: target, Adjustments: []PTValueAdjustment{{Power: value, Toughness: Fixed(0)}}}
				if order == "foreign subject" {
					instruction.Subject = DeclarePermanentTarget("foreign", AnyPermanent)
					known = false
				}
				effect := instruction.Effect()
				obj.Effects = []Effect{effect}
				switch order {
				case "prefix":
					obj.Effects = []Effect{Sequence(effect, Destroy(otherTarget))}
					known = false
				case "suffix", "clone", "copy":
					obj.Effects = []Effect{Sequence(Destroy(otherTarget), effect)}
					want = known
				case "unknown suffix":
					obj.Effects = []Effect{Sequence(Destroy(otherTarget), effect, destructionInspectionTrap{})}
					known = false
				case "all illegal", "partial illegal":
					p.GrantBaseAttr(core.Shroud)
					obj.Effects = []Effect{Sequence(effect, Destroy(otherTarget))}
					if order == "all illegal" {
						other.GrantBaseAttr(core.Shroud)
					} else {
						want = known
					}
				case "duplicate":
					obj.TargetSpecs = append(obj.TargetSpecs, target.targetSpec())
					obj.Targets = append(obj.Targets, p.ID())
					known = false
				}
				if order == "clone" {
					g = g.Clone()
					obj = g.StackPeek()
				}
				if order == "copy" {
					obj = g.CopyStackObjectDirect(obj, g.GetOpponent(obj.Controller).PlayerID(), false)
				}
				for range 2 {
					certain, k := g.InspectSpellDestruction(obj, []PermanentVisit{{other.ID(), other.incarnationID}})
					if certain != want || k != known {
						t.Fatalf("got %v/%v want %v/%v", certain, k, want, known)
					}
				}
				if len(g.effects.effects) != 0 || g.FindPermanent(other.ID()) == nil {
					t.Fatal("query changed state")
				}
			})
		}
	}
}

type targetQueryValueTrap struct{}

func (targetQueryValueTrap) Text() string { panic("value text callback ran") }
func (targetQueryValueTrap) Resolve(GameReader, uuid.UUID, uuid.UUID, []uuid.UUID) int {
	panic("legacy value callback ran")
}
