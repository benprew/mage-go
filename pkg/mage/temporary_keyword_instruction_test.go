package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTemporaryKeywordInstructionDeclaration(t *testing.T) {
	target := DeclarePermanentTarget("subject", AnyPermanent)
	values := []core.Attr{core.Flying, core.FirstStrike}
	instruction := TemporaryKeywordInstruction{Subject: target, Keywords: values}
	effect := instruction.Effect()
	values[0] = core.Trample
	got, ok := InspectTemporaryKeywordInstruction(effect)
	if !ok || got.Subject != target || got.Keywords[0] != core.Flying {
		t.Fatal("lost or shared declaration")
	}
	got.Keywords[0] = core.Trample
	again, _ := InspectTemporaryKeywordInstruction(effect)
	if again.Keywords[0] != core.Flying {
		t.Fatal("inspection aliases declaration")
	}
	for _, instruction := range []TemporaryKeywordInstruction{
		{}, {Subject: target},
		{Subject: target, Keywords: []core.Attr{0}},
		{Subject: target, Keywords: []core.Attr{core.AttrIsCreature}},
		{Subject: target, Keywords: []core.Attr{core.Attr(999)}},
		{Subject: target, Keywords: []core.Attr{core.CantRegenerate}},
		{Subject: target, Keywords: []core.Attr{core.AttrAttackAsThoughHaste}},
		{Subject: target, Keywords: []core.Attr{core.AttrPrepared}},
		{Keywords: []core.Attr{core.Flying}},
		{Subject: target, Subjects: MatchingPermanents(AnyPermanent), Keywords: []core.Attr{core.Flying}},
		{Subject: (*PermanentTargetBinding)(nil), Keywords: []core.Attr{core.Flying}},
	} {
		if instruction.Effect() != nil {
			t.Fatal("accepted invalid declaration")
		}
	}
	for _, effect := range []Effect{nil, (*temporaryKeywordEffect)(nil), &temporaryKeywordEffect{}, Until(core.EndOfTurn, GrantKeywordModification(target, core.Flying)), Sequence(effect, destructionInspectionTrap{})} {
		if _, ok := InspectTemporaryKeywordInstruction(effect); ok {
			t.Fatal("accepted an undeclared tree")
		}
	}
}

func TestTemporaryKeywordInstructionOrder(t *testing.T) {
	for _, plural := range []bool{false, true} {
		for _, state := range []string{"alone", "prefix", "suffix", "unknown suffix", "all illegal", "unknown all illegal", "illegal prefix", "foreign", "duplicate", "opaque", "clone", "copy", "bound missing"} {
			t.Run(state+map[bool]string{false: "/target", true: "/collection"}[plural], func(t *testing.T) {
				g, p, obj, target := targetQueryFixture(t)
				other := g.PutOnBattlefield(NewLand("Other"), obj.Controller)
				otherTarget := DeclarePermanentTarget("other", HasCardTypeFilter(core.TypeLand))
				obj.TargetSpecs = append(obj.TargetSpecs, otherTarget.targetSpec())
				obj.Targets = append(obj.Targets, other.ID())
				obj.TargetZones[other.ID()] = core.ZoneBattlefield
				obj.targetIncarnations[other.ID()] = other.incarnationID
				instruction := TemporaryKeywordInstruction{Subject: target, Keywords: []core.Attr{core.Flying}}
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
				case "bound missing":
					instruction.Subject = nil
					instruction.Subjects = PermanentsControlledBy(DeclarePlayerTarget("missing"), AnyPermanent)
					known = false
				case "opaque":
					instruction.Subject = This
					instruction.Subjects = nil
					known = false
				}
				effect := instruction.Effect()
				obj.Effects = []Effect{effect}
				switch state {
				case "prefix":
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

func TestTemporaryKeywordInstructionInstallation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Subject", "", 4, 4)
	p := g.PutOnBattlefield(card, a.PlayerID())
	effect := (TemporaryKeywordInstruction{Subject: This, Keywords: []core.Attr{core.Flying, core.FirstStrike}}).Effect()
	if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !p.HasKeyword(core.Flying) || !p.HasKeyword(core.FirstStrike) || len(g.effects.effects) != 1 {
		t.Fatal("adjustments did not share one installation")
	}
	clone := g.Clone()
	g.PhaseIn(g.PhaseOut(p.ID()))
	if !p.HasKeyword(core.Flying) {
		t.Fatal("phasing lost the installation")
	}
	g.RemoveFromBattlefield(p)
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if returned.HasKeyword(core.Flying) {
		t.Fatal("installation followed a new visit")
	}
	cp := clone.FindPermanent(p.ID())
	if !cp.HasKeyword(core.Flying) {
		t.Fatal("clone followed original movement")
	}
	clone.GetEffects().RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	cp = clone.FindPermanent(p.ID())
	if cp.HasKeyword(core.Flying) || cp.HasKeyword(core.FirstStrike) {
		t.Fatal("cleanup kept the installation")
	}
}
