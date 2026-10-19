package mage

import (
	"math"
	"slices"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTemporaryColorInstructionDeclaration(t *testing.T) {
	subjects := DeclarePermanentTargets("subjects", 1, 2, AnyPermanent)
	colors := []core.Color{core.Red, core.Blue}
	effect := (TemporaryColorInstruction{Subjects: subjects, Colors: colors}).Effect()
	colors[0] = core.Green
	got, ok := InspectTemporaryColorInstruction(effect)
	if !ok || got.Subjects != subjects || got.Colors[0] != core.Red {
		t.Fatal("lost or shared declaration")
	}
	got.Colors[0] = core.Black
	again, _ := InspectTemporaryColorInstruction(effect)
	if again.Colors[0] != core.Red {
		t.Fatal("inspection aliases declaration")
	}
	for _, instruction := range []TemporaryColorInstruction{
		{}, {Subjects: (*PermanentTargetsBinding)(nil)},
		{Subjects: subjects, Colors: []core.Color{core.Colorless}},
		{Subjects: subjects, Colors: []core.Color{core.Color(999)}},
	} {
		if instruction.Effect() != nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	if (TemporaryColorInstruction{Subjects: subjects}).Effect() == nil {
		t.Fatal("colorless declaration rejected")
	}
	for _, effect := range []Effect{nil, (*temporaryColorEffect)(nil), &temporaryColorEffect{}, Until(core.EndOfTurn, SetEachColors(subjects, core.Red)), Sequence(effect, destructionInspectionTrap{})} {
		if _, ok := InspectTemporaryColorInstruction(effect); ok {
			t.Fatal("undeclared tree accepted")
		}
	}
}

func TestTemporaryColorInstructionQuery(t *testing.T) {
	for _, state := range []string{"alone", "prefix", "suffix", "empty prefix", "unknown suffix", "unknown all illegal", "all illegal", "opaque", "foreign", "clone", "copy", "colorless"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := targetQueryFixture(t)
			subjects := MatchingPermanents(AnyPermanent)
			colors := []core.Color{core.Red}
			known, want := true, false
			switch state {
			case "empty prefix":
				subjects = MatchingPermanents(IsCreature)
			case "opaque":
				subjects = MatchingPermanents(NewPermanentFilter("trap", func(*Permanent, *Game) bool { panic("filter ran") }))
				known = false
			case "foreign":
				subjects = DeclarePermanentTargets("foreign", 1, 2, AnyPermanent)
				known = false
			case "colorless":
				colors = nil
			}
			effect := (TemporaryColorInstruction{Subjects: subjects, Colors: colors}).Effect()
			obj.Effects = []Effect{effect}
			switch state {
			case "prefix", "empty prefix":
				obj.Effects = []Effect{Sequence(effect, Destroy(target))}
				known = state == "empty prefix"
				want = known
			case "suffix", "clone", "copy":
				obj.Effects = []Effect{Sequence(Destroy(target), effect)}
				want = true
			case "unknown suffix", "unknown all illegal":
				obj.Effects = []Effect{Sequence(Destroy(target), effect, destructionInspectionTrap{})}
				known = false
			}
			if state == "all illegal" || state == "unknown all illegal" {
				p.GrantBaseAttr(core.Shroud)
			}
			if state == "clone" {
				g = g.Clone()
				obj = g.StackPeek()
			}
			if state == "copy" {
				obj = g.CopyStackObjectDirect(obj, obj.Controller, false)
			}
			for range 2 {
				certain, k := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}})
				if certain != want || k != known {
					t.Fatalf("got %v/%v want %v/%v", certain, k, want, known)
				}
			}
			if len(g.GetEffects().effects) != 0 || len(p.Colors()) != 0 {
				t.Fatal("query changed state")
			}
		})
	}
}

func TestTemporaryColorInstructionInstallation(t *testing.T) {
	for _, colors := range [][]core.Color{nil, {core.Red, core.Green}} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		card := NewCreature("Subject", "{W}{U}", 2, 3)
		p := g.PutOnBattlefield(card, a.PlayerID())
		effect := (TemporaryColorInstruction{Subjects: MatchingPermanents(IsCreature), Colors: colors}).Effect()
		if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		late := g.PutOnBattlefield(NewCreature("Late", "{U}", 1, 1), b.PlayerID())
		if !slices.Equal(p.Colors(), colors) || !slices.Equal(late.Colors(), []core.Color{core.Blue}) || len(g.effects.effects) != 1 {
			t.Fatal("incorrect capture or installation")
		}
		clone := g.Clone()
		g.PhaseIn(g.PhaseOut(p.ID()))
		if !slices.Equal(g.FindPermanent(p.ID()).Colors(), colors) {
			t.Fatal("phase cycle lost colors")
		}
		g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
		returned := g.PutOnBattlefield(card, a.PlayerID())
		if !slices.Equal(returned.Colors(), []core.Color{core.White, core.Blue}) {
			t.Fatal("colors followed return")
		}
		if !slices.Equal(clone.FindPermanent(p.ID()).Colors(), colors) {
			t.Fatal("clone lost captured colors")
		}
		clone.GetEffects().RemoveEndOfTurn()
		clone.ApplyContinuousEffects()
		if !slices.Equal(clone.FindPermanent(p.ID()).Colors(), []core.Color{core.White, core.Blue}) {
			t.Fatal("cleanup retained colors")
		}
	}
}

func TestTemporaryColorInstructionOneOrMoreTargets(t *testing.T) {
	for _, state := range []string{"normal", "too few", "too many", "duplicate", "opaque", "phase", "missing visit"} {
		t.Run(state, func(t *testing.T) {
			g, p, q, obj, binding := pluralQueryFixture(t, 2)
			binding.spec = &PermanentTarget{BaseTarget: BaseTarget{min: 1, max: math.MaxInt}, Filters: []PermanentFilter{IsCreature}}
			switch state {
			case "too few":
				obj.Targets = nil
				obj.TargetSpecs = nil
			case "too many":
				binding.spec.(*PermanentTarget).max = 1
			case "duplicate":
				obj.Targets[1] = obj.Targets[0]
			case "opaque":
				binding.spec.(*PermanentTarget).Filters = append(binding.spec.(*PermanentTarget).Filters, NewPermanentFilter("trap", func(*Permanent, *Game) bool { panic("filter ran") }))
			case "phase":
				g.PhaseOut(p.ID())
			case "missing visit":
				delete(obj.targetIncarnations, p.ID())
			}
			predicate, ok := InspectPermanentTargets(binding)
			if state == "opaque" {
				if ok {
					t.Fatal("opaque target accepted")
				}
				return
			}
			if !ok {
				t.Fatal("one-or-more declaration unknown")
			}
			got, known := predicate.Select(g, obj)
			wantKnown := state == "normal" || state == "phase"
			wantCount := 2
			if state == "phase" {
				wantCount = 1
			}
			if known != wantKnown || known && len(got) != wantCount {
				t.Fatalf("selection %v/%v", got, known)
			}
			if state == "normal" && (got[0].ID != p.ID() || got[1].ID != q.ID()) {
				t.Fatal("selection order changed")
			}
		})
	}
}
