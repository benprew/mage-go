package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTemporaryLayerBranchDeclaration(t *testing.T) {
	target := DeclarePermanentTarget("subject", AnyPermanent)
	body := (TemporaryRampageInstruction{Subject: target, Amounts: []int{0, 2}}).Effect()
	branch := PermanentBranchInstruction{Subject: target, Filter: Not(HasRampage), Matching: body}
	effect := branch.Effect()
	got, ok := InspectPermanentBranchInstruction(effect)
	if !ok || got.Subject != target || got.Matching != body || got.Nonmatching != nil || got.Negate {
		t.Fatal("lost branch operands")
	}
	for _, invalid := range []PermanentBranchInstruction{{}, {Subject: target}, {Subject: (*PermanentTargetBinding)(nil), Matching: body}, {Subject: target, Matching: (*temporaryRampageEffect)(nil)}, {Subject: target, Matching: body, Nonmatching: (*temporaryRampageEffect)(nil)}} {
		if invalid.Effect() != nil {
			t.Fatal("invalid branch")
		}
	}
	amounts := []int{0, 2}
	rampage := (TemporaryRampageInstruction{Subject: target, Amounts: amounts}).Effect()
	amounts[1] = 8
	instruction, ok := InspectTemporaryRampageInstruction(rampage)
	if !ok || instruction.Amounts[1] != 2 {
		t.Fatal("input list aliases declaration")
	}
	instruction.Amounts[1] = 9
	again, _ := InspectTemporaryRampageInstruction(rampage)
	if again.Amounts[1] != 2 {
		t.Fatal("output list aliases declaration")
	}
	for _, invalid := range []TemporaryRampageInstruction{{}, {Subject: target}, {Amounts: []int{2}}, {Subject: target, Amounts: []int{-1}}, {Subject: (*PermanentTargetBinding)(nil), Amounts: []int{2}}} {
		if invalid.Effect() != nil {
			t.Fatal("invalid rampage")
		}
	}
	for _, opaque := range []Effect{nil, Until(core.EndOfTurn, GrantKeywordModification(target, core.Flying)), Sequence(body)} {
		if _, ok := InspectTemporaryRampageInstruction(opaque); ok {
			t.Fatal("accepted undeclared rampage")
		}
		if _, ok := InspectPermanentBranchInstruction(opaque); ok {
			t.Fatal("accepted undeclared branch")
		}
	}
}

func TestTemporaryLayerBranchInspection(t *testing.T) {
	for _, state := range []string{"alone", "prefix", "suffix", "opaque then", "opaque else", "opaque filter", "foreign condition", "foreign value", "foreign layer", "opaque predicate child", "destruction branch", "illegal", "unknown illegal", "unknown filter illegal", "duplicate", "clone", "copy", "nested", "negated", "no alternative", "false prefix"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := targetQueryFixture(t)
			layer := (TemporaryPTValueInstruction{Subject: target, Adjustments: []PTValueAdjustment{{Power: Fixed(4), Toughness: ToughnessOfWithLKI(target)}}}).Effect()
			branch := PermanentBranchInstruction{Subject: target, Filter: HasToughnessGTE(5), Matching: layer, Nonmatching: (TemporaryRampageInstruction{Subject: target, Amounts: []int{2}}).Effect()}
			known, want := true, false
			switch state {
			case "opaque then":
				branch.Matching = destructionInspectionTrap{}
				known = false
			case "opaque else":
				branch.Nonmatching = destructionInspectionTrap{}
				known = false
			case "opaque filter":
				branch.Filter = NewPermanentFilter("trap", func(*Permanent, *Game) bool { panic("filter ran") })
				known = false
			case "foreign condition":
				branch.Subject = DeclarePermanentTarget("foreign", AnyPermanent)
				known = false
			case "foreign value":
				branch.Matching = (TemporaryPTValueInstruction{Subject: target, Adjustments: []PTValueAdjustment{{Power: PowerOfWithLKI(DeclarePermanentTarget("foreign", AnyPermanent)), Toughness: Fixed(0)}}}).Effect()
				known = false
			case "foreign layer":
				branch.Nonmatching = (TemporaryRampageInstruction{Subject: DeclarePermanentTarget("foreign", AnyPermanent), Amounts: []int{2}}).Effect()
				known = false
			case "opaque predicate child":
				branch.Filter = Or(AnyPermanent, NewPermanentFilter("trap", func(*Permanent, *Game) bool { panic("filter ran") }))
				known = false
			case "destruction branch":
				branch.Nonmatching = Destroy(target)
				known = false
			case "unknown filter illegal":
				p.GrantBaseAttr(core.Shroud)
				branch.Filter = NewPermanentFilter("trap", func(*Permanent, *Game) bool { panic("filter ran") })
				known = false
			case "prefix":
				branch.Filter = AnyPermanent
			case "no alternative":
				branch.Nonmatching = nil
			case "negated":
				branch.Negate = true
			case "nested":
				branch.Matching = (PermanentBranchInstruction{Subject: target, Filter: Not(HasRampage), Matching: layer}).Effect()
			}
			effect := branch.Effect()
			obj.Effects = []Effect{effect}
			switch state {
			case "prefix", "false prefix":
				obj.Effects = []Effect{Sequence(effect, Destroy(target))}
				known = false
			case "suffix", "clone", "copy":
				obj.Effects = []Effect{Sequence(Destroy(target), effect)}
				want = true
			case "illegal":
				p.GrantBaseAttr(core.Shroud)
			case "unknown illegal":
				p.GrantBaseAttr(core.Shroud)
				obj.Effects = append(obj.Effects, destructionInspectionTrap{})
				known = false
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
				certain, k := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}})
				if certain != want || k != known {
					t.Fatalf("got %v/%v want %v/%v", certain, k, want, known)
				}
			}
			if len(g.effects.effects) != 0 {
				t.Fatal("query installed layers")
			}
		})
	}
}

func TestTemporaryLayerBranchRampageInstallation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Subject", "", 3, 6)
	p := g.PutOnBattlefield(card, a.PlayerID())
	effect := (PermanentBranchInstruction{Subject: This, Filter: Not(HasRampage), Matching: (TemporaryRampageInstruction{Subject: This, Amounts: []int{0, 2, 3}}).Effect()}).Effect()
	if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.effects.effects) != 1 || !HasRampage.Match(g.FindPermanent(p.ID()), g) {
		t.Fatal("grants did not share an installation")
	}
	clone := g.Clone()
	g.PhaseIn(g.PhaseOut(p.ID()))
	g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
	for range 3 {
		g.PutOnBattlefieldBlocking(NewCreature("Blocker", "", 1, 1), b.PlayerID(), p.ID())
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatal("rampage instances merged")
	}
	g.ResolveStack()
	if g.FindPermanent(p.ID()).CurrentPower(g) != 13 {
		t.Fatal("incorrect rampage bonus")
	}
	g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if HasRampage.Match(returned, g) {
		t.Fatal("grants followed a returned object")
	}
	if !HasRampage.Match(clone.FindPermanent(p.ID()), clone) || clone.FindPermanent(p.ID()).CurrentPower(clone) != 3 {
		t.Fatal("clone shared grant state")
	}
	clone.GetEffects().RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if HasRampage.Match(clone.FindPermanent(p.ID()), clone) {
		t.Fatal("grant survived cleanup")
	}
}
