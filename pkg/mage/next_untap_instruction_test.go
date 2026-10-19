package mage

import (
	"maps"
	"reflect"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestNextUntapInstructionDeclaration(t *testing.T) {
	target := DeclarePermanentTarget("subject", IsCreature)
	subjects := MatchingPermanents(IsCreature)
	for _, instruction := range []NextUntapInstruction{
		{Subject: target, Count: 0}, {Subject: target, Count: 1},
		{Subject: target, Count: 2}, {Subjects: subjects, Count: 1},
	} {
		effect := instruction.Effect()
		got, ok := InspectNextUntapInstruction(effect)
		if !ok || got.Count != instruction.Count || got.Subject != instruction.Subject || nilDefinition(got.Subjects) != nilDefinition(instruction.Subjects) {
			t.Fatal("lost native declaration")
		}
		card := NewInstant("Lock", "", Spell(effect, WithTargets(target)))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	for _, instruction := range []NextUntapInstruction{
		{}, {Count: 1}, {Subject: target, Count: -1},
		{Subject: target, Subjects: subjects, Count: 1},
		{Subjects: subjects, Count: 0}, {Subjects: subjects, Count: 2},
		{Subject: (*PermanentTargetBinding)(nil), Count: 1},
	} {
		if instruction.Effect() != nil {
			t.Fatal("accepted malformed declaration")
		}
	}
	for _, effect := range []Effect{
		nil, (*skipNextUntapEffect)(nil), (*skipNextUntapsEffect)(nil), (*skipNextUntapEachEffect)(nil),
		&skipNextUntapEffect{}, SkipNextUntaps(target, -1), SkipNextUntapEach(nil),
		SkipNextUntapOf(target, You), Sequence(SkipNextUntap(target)), destructionInspectionTrap{},
	} {
		if _, ok := InspectNextUntapInstruction(effect); ok {
			t.Fatal("accepted a different or incomplete instruction")
		}
	}
	for _, effect := range []Effect{SkipNextUntap(target), SkipNextUntaps(target, 2), SkipNextUntapEach(subjects)} {
		if _, ok := InspectNextUntapInstruction(effect); !ok {
			t.Fatal("lost legacy native declaration")
		}
	}
}

func TestNextUntapInstructionQuery(t *testing.T) {
	for _, state := range []string{
		"alone", "nested", "prefix", "suffix", "zero prefix", "empty collection", "collection prefix", "collection suffix",
		"all illegal", "partly illegal", "phase", "cycle", "return", "foreign", "foreign collection", "opaque collection",
		"opaque prefix", "opaque suffix", "opaque all illegal", "nil suffix", "typed nil suffix", "clone", "copy",
	} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := targetQueryFixture(t)
			lock := (NextUntapInstruction{Subject: target, Count: 2}).Effect()
			obj.Effects = []Effect{lock}
			watched := []PermanentVisit{{p.ID(), p.incarnationID}}
			certain, known := false, true
			switch state {
			case "nested":
				obj.Effects = []Effect{Sequence(CompositeEffects("nested", lock))}
			case "prefix":
				obj.Effects = []Effect{lock, Destroy(target)}
				known = false
			case "suffix", "clone", "copy", "cycle":
				obj.Effects = []Effect{Destroy(target), lock}
				certain = true
			case "zero prefix":
				obj.Effects = []Effect{(NextUntapInstruction{Subject: target, Count: 0}).Effect(), Destroy(target)}
				certain = true
			case "empty collection":
				obj.Effects = []Effect{(NextUntapInstruction{Subjects: MatchingPermanents(IsCreature), Count: 1}).Effect(), Destroy(target)}
				certain = true
			case "collection prefix":
				obj.Effects = []Effect{(NextUntapInstruction{Subjects: MatchingPermanents(IsLand), Count: 1}).Effect(), Destroy(target)}
				known = false
			case "collection suffix":
				obj.Effects = []Effect{Destroy(target), (NextUntapInstruction{Subjects: MatchingPermanents(IsCreature), Count: 1}).Effect()}
				certain = true
			case "all illegal":
				p.GrantBaseAttr(core.Shroud)
				obj.Effects = []Effect{lock, Destroy(target)}
			case "partly illegal":
				p.GrantBaseAttr(core.Shroud)
				other := g.PutOnBattlefield(NewCreature("Other", "", 1, 3), obj.Controller)
				second := DeclarePermanentTarget("other", IsCreature)
				obj.TargetSpecs = append(obj.TargetSpecs, second.targetSpec())
				obj.Targets = append(obj.Targets, other.ID())
				obj.targetIncarnations[other.ID()] = other.incarnationID
				obj.TargetZones[other.ID()] = core.ZoneBattlefield
				obj.Effects = []Effect{lock, Destroy(second)}
				watched = []PermanentVisit{{other.ID(), other.incarnationID}}
				certain = true
			case "phase":
				g.PhaseOut(p.ID())
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, obj.Controller)
			case "foreign":
				obj.Effects = []Effect{Destroy(target), (NextUntapInstruction{Subject: DeclarePermanentTarget("foreign", IsCreature), Count: 1}).Effect()}
				known = false
			case "foreign collection":
				obj.Effects = []Effect{Destroy(target), (NextUntapInstruction{Subjects: DeclarePermanentTargets("foreign", 1, 1, IsCreature), Count: 1}).Effect()}
				known = false
			case "opaque collection":
				obj.Effects = []Effect{Destroy(target), (NextUntapInstruction{Subjects: MatchingPermanents(NewPermanentFilter("opaque", func(*Permanent, *Game) bool { panic("callback") })), Count: 1}).Effect()}
				known = false
			case "opaque prefix":
				obj.Effects = []Effect{destructionInspectionTrap{}, lock, Destroy(target)}
				known = false
			case "opaque suffix", "opaque all illegal":
				obj.Effects = []Effect{Destroy(target), lock, destructionInspectionTrap{}}
				known = false
				if state == "opaque all illegal" {
					p.GrantBaseAttr(core.Shroud)
				}
			case "nil suffix":
				obj.Effects = []Effect{Destroy(target), lock, nil}
				known = false
			case "typed nil suffix":
				obj.Effects = []Effect{Destroy(target), (*skipNextUntapsEffect)(nil)}
				known = false
			}
			if state == "cycle" {
				g.PhaseIn(g.PhaseOut(p.ID()))
			}
			if state == "copy" {
				obj = g.CopyStackObjectDirect(obj, g.GetOpponent(obj.Controller).PlayerID(), false)
			}
			if state == "clone" {
				g = g.Clone()
				obj = g.StackPeek()
			}
			before := g.Clone()
			for range 2 {
				c, k := g.InspectSpellDestruction(obj, watched)
				if c != certain || k != known {
					t.Fatalf("got %v/%v, want %v/%v", c, k, certain, known)
				}
			}
			if !maps.Equal(g.turns.skipNextUntap, before.turns.skipNextUntap) || !maps.Equal(g.turns.capturedUntap, before.turns.capturedUntap) || g.turns.turn != before.turns.turn || g.turns.step != before.turns.step || !reflect.DeepEqual(g.random, before.random) || len(g.effects.replacements) != len(before.effects.replacements) || g.StackSize() != before.StackSize() {
				t.Fatal("inspection changed state")
			}
		})
	}
}
