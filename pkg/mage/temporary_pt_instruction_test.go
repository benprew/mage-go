package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTemporaryPTInstructionDeclaration(t *testing.T) {
	target := DeclarePermanentTarget("subject", AnyPermanent)
	values := []PTAdjustment{{3, 3}, {-1, 0}}
	instruction := TemporaryPTInstruction{Subject: target, Adjustments: values}
	effect := instruction.Effect()
	values[0].Power = 99
	got, ok := InspectTemporaryPTInstruction(effect)
	if !ok || got.Subject != target || got.Adjustments[0].Power != 3 {
		t.Fatal("lost or shared declaration")
	}
	got.Adjustments[0].Power = 99
	again, _ := InspectTemporaryPTInstruction(effect)
	if again.Adjustments[0].Power != 3 {
		t.Fatal("inspection aliases declaration")
	}
	for _, instruction := range []TemporaryPTInstruction{
		{}, {Subject: target}, {Adjustments: []PTAdjustment{{1, 1}}},
		{Subject: target, Subjects: MatchingPermanents(AnyPermanent), Adjustments: []PTAdjustment{{1, 1}}},
		{Subject: (*PermanentTargetBinding)(nil), Adjustments: []PTAdjustment{{1, 1}}},
	} {
		if instruction.Effect() != nil {
			t.Fatal("accepted invalid declaration")
		}
	}
	for _, effect := range []Effect{nil, (*temporaryPTEffect)(nil), Until(core.EndOfTurn, ModifyPT(target, Fixed(1), Fixed(1))), Sequence(effect, destructionInspectionTrap{})} {
		if _, ok := InspectTemporaryPTInstruction(effect); ok {
			t.Fatal("accepted an undeclared tree")
		}
	}
}

func TestTemporaryPTInstructionOrder(t *testing.T) {
	for _, plural := range []bool{false, true} {
		for _, state := range []string{"alone", "prefix", "suffix", "unknown suffix", "all illegal", "unknown all illegal", "illegal prefix", "foreign", "duplicate", "opaque", "clone", "copy", "zero"} {
			t.Run(state+map[bool]string{false: "/target", true: "/collection"}[plural], func(t *testing.T) {
				g, p, obj, target := targetQueryFixture(t)
				other := g.PutOnBattlefield(NewLand("Other"), obj.Controller)
				otherTarget := DeclarePermanentTarget("other", HasCardTypeFilter(core.TypeLand))
				obj.TargetSpecs = append(obj.TargetSpecs, otherTarget.targetSpec())
				obj.Targets = append(obj.Targets, other.ID())
				obj.TargetZones[other.ID()] = core.ZoneBattlefield
				obj.targetIncarnations[other.ID()] = other.incarnationID
				instruction := TemporaryPTInstruction{Subject: target, Adjustments: []PTAdjustment{{1, 1}}}
				if plural {
					instruction.Subject = nil
					instruction.Subjects = MatchingPermanents(HasCardTypeFilter(core.TypeLand))
				}
				known, want := true, false
				switch state {
				case "foreign":
					if plural {
						instruction.Subjects = MatchingPermanents(NewPermanentFilter("trap", func(*Permanent, *Game) bool { panic("filter ran") }))
					} else {
						instruction.Subject = DeclarePermanentTarget("foreign", AnyPermanent)
					}
					known = false
				case "opaque":
					instruction.Subject = This
					instruction.Subjects = nil
					known = false
				case "zero":
					instruction.Adjustments[0] = PTAdjustment{}
				}
				effect := instruction.Effect()
				obj.Effects = []Effect{effect}
				switch state {
				case "prefix", "zero":
					obj.Effects = []Effect{Sequence(effect, Destroy(otherTarget))}
					known = false
				case "suffix", "clone", "copy":
					obj.Effects = []Effect{Sequence(Destroy(otherTarget), effect)}
					want = true
				case "unknown suffix":
					obj.Effects = []Effect{Sequence(Destroy(otherTarget), effect, destructionInspectionTrap{})}
					known = false
				case "all illegal", "unknown all illegal":
					p.GrantBaseAttr(core.Shroud)
					other.GrantBaseAttr(core.Shroud)
					obj.Effects = []Effect{Sequence(effect, Destroy(otherTarget))}
					if state == "unknown all illegal" {
						obj.Effects = append(obj.Effects, destructionInspectionTrap{})
						known = false
					}
				case "illegal prefix":
					p.GrantBaseAttr(core.Shroud)
					if plural {
						instruction.Subjects = MatchingPermanents(HasCardTypeFilter(core.TypeCreature))
						effect = instruction.Effect()
					}
					obj.Effects = []Effect{Sequence(effect, Destroy(otherTarget))}
					want = true
				case "duplicate":
					obj.TargetSpecs = append(obj.TargetSpecs, target.targetSpec())
					obj.Targets = append(obj.Targets, p.ID())
					known = false
				}
				if state == "clone" {
					g = g.Clone()
					obj = g.StackPeek()
				}
				if state == "copy" {
					obj = g.CopyStackObjectDirect(obj, g.GetOpponent(obj.Controller).PlayerID(), false)
				}
				for range 2 {
					certain, k := g.InspectSpellDestruction(obj, []PermanentVisit{{other.ID(), other.incarnationID}})
					if certain != want || k != known {
						t.Fatalf("got %v/%v want %v/%v", certain, k, want, known)
					}
				}
				if len(g.GetEffects().effects) != 0 {
					t.Fatal("query installed effects")
				}
			})
		}
	}
}

func TestTemporaryPTInstructionInstallation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Subject", "", 4, 4)
	p := g.PutOnBattlefield(card, a.PlayerID())
	effect := (TemporaryPTInstruction{Subject: This, Adjustments: []PTAdjustment{{3, 3}, {-1, -2}}}).Effect()
	if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if p.CurrentPower(g) != 6 || p.CurrentToughness(g) != 5 || len(g.effects.effects) != 1 {
		t.Fatal("adjustments did not share one installation")
	}
	clone := g.Clone()
	g.PhaseIn(g.PhaseOut(p.ID()))
	if p.CurrentPower(g) != 6 {
		t.Fatal("phasing lost the installation")
	}
	g.RemoveFromBattlefield(p)
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if returned.CurrentPower(g) != 4 {
		t.Fatal("installation followed a new visit")
	}
	cp := clone.FindPermanent(p.ID())
	if cp.CurrentPower(clone) != 6 {
		t.Fatal("clone followed original movement")
	}
	clone.GetEffects().RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	cp = clone.FindPermanent(p.ID())
	if cp.CurrentPower(clone) != 4 || cp.CurrentToughness(clone) != 4 {
		t.Fatalf("cleanup kept the installation: %d/%d, effects=%d", cp.CurrentPower(clone), cp.CurrentToughness(clone), len(clone.effects.effects))
	}
}
