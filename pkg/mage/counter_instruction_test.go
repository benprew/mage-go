package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCounterInstructionDeclaration(t *testing.T) {
	target := DeclareSpellTarget("spell")
	instruction := CounterInstruction{Subject: target}
	for _, effect := range []ResolvingEffect{instruction.Effect(), Counter(target), Sequence(Counter(target), CompositeEffects("nested", Counter(target)))} {
		got, ok := InspectCounterInstructions(effect)
		if !ok || len(got) == 0 || got[0].Subject != target {
			t.Fatal("lost native spell declaration")
		}
		got[0].Subject = nil
		again, ok := InspectCounterInstructions(effect)
		if !ok || again[0].Subject != target {
			t.Fatal("inspection output shares mutable storage")
		}
	}
	var missing *SpellTargetBinding
	for _, ref := range []StackObjectRef{nil, missing} {
		if (CounterInstruction{Subject: ref}).Effect() != nil {
			t.Fatal("accepted missing subject")
		}
	}
	for _, effect := range []ResolvingEffect{nil, (*counterReferencedEffect)(nil), Counter(nil), &counterReferencedEffect{spell: target, ability: true}, Sequence(Counter(target), destructionInspectionTrap{})} {
		if _, ok := InspectCounterInstructions(effect); ok {
			t.Fatal("accepted incomplete tree")
		}
	}
}

func counterInstructionFixture(t *testing.T) (*Game, *StackObject, *StackObject, *SpellTargetBinding) {
	t.Helper()
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	victim := NewInstant("Victim", "")
	a.AddToHand(victim)
	if err := g.CastSpellByID(a.PlayerID(), victim.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	first := g.StackPeek()
	target := DeclareSpellTarget("spell").WithFilter(AllSpellFilters(SpellCardMatches(HasCardTypeCardFilter(core.TypeInstant))))
	card := NewInstant("Counter", "", WithAbility(Spell((CounterInstruction{Subject: target}).Effect(), WithTargets(target))))
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{first.ID}, 0); err != nil {
		t.Fatal(err)
	}
	return g, first, g.StackPeek(), target
}

func TestCounterInstructionTarget(t *testing.T) {
	for _, state := range []string{"normal", "left", "recast", "copy", "clone", "wrong type", "missing zone", "wrong zone", "foreign", "duplicate", "nil identity", "opaque", "opaque child", "bad filter count", "mana restriction", "controlled permanent"} {
		t.Run(state, func(t *testing.T) {
			g, victim, obj, target := counterInstructionFixture(t)
			want, known := victim.ID, true
			switch state {
			case "left", "recast":
				g.CounterStackObject(victim.ID)
				want = uuid.Nil
				if state == "recast" {
					p := g.GetPlayer(victim.Controller)
					p.RemoveFromGraveyard(victim.Card.ID())
					p.AddToHand(victim.Card)
					if err := g.CastSpellByID(p.PlayerID(), victim.Card.ID(), nil, 0); err != nil {
						t.Fatal(err)
					}
				}
			case "copy":
				obj = g.CopyStackObjectDirect(obj, obj.Controller, false)
			case "clone":
				g = g.Clone()
				obj = g.StackPeek()
			case "wrong type":
				target.spec.(*typedSpellTarget).contextualFilter = SpellCardMatches(HasCardTypeCardFilter(core.TypeCreature))
				want = uuid.Nil
			case "missing zone":
				obj.TargetZones = nil
				want, known = uuid.Nil, false
			case "wrong zone":
				obj.TargetZones[victim.ID] = core.ZoneHand
				want = uuid.Nil
			case "foreign":
				target = DeclareSpellTarget("spell")
				want, known = uuid.Nil, false
			case "duplicate":
				obj.TargetSpecs = append(obj.TargetSpecs, target.targetSpec())
				obj.Targets = append(obj.Targets, victim.ID)
				want, known = uuid.Nil, false
			case "nil identity":
				obj.Targets[0] = uuid.Nil
				want = uuid.Nil
			case "opaque", "opaque child":
				trap := SpellCardMatches(NewCardFilter("any card", func(Card) bool { panic("filter callback") }))
				if state == "opaque child" {
					trap = AnySpellFilter(SpellCardMatches(AnyCardFilter()), trap)
				}
				target.spec.(*typedSpellTarget).contextualFilter = trap
				want, known = uuid.Nil, false
			case "bad filter count":
				target.spec.(*typedSpellTarget).filterCount = 2
				want, known = uuid.Nil, false
			case "mana restriction":
				target.spec.(*typedSpellTarget).manaValueEqualX = true
			case "controlled permanent":
				target.spec.(*typedSpellTarget).targetsControlledPermanent = true
				want, known = uuid.Nil, false
			}
			predicate, ok := InspectSpellTarget(target)
			if !ok {
				if known {
					t.Fatal("native declaration rejected")
				}
				return
			}
			for range 2 {
				got, gotKnown := predicate.Select(g, obj)
				if got != want || gotKnown != known {
					t.Fatalf("got %v/%v want %v/%v", got, gotKnown, want, known)
				}
			}
		})
	}
}

func TestCounterInstructionSpell(t *testing.T) {
	for _, state := range []string{"alone", "prefix", "suffix", "illegal prefix", "unknown suffix", "foreign binding", "all illegal", "uncounterable prefix"} {
		t.Run(state, func(t *testing.T) {
			g, victim, obj, target := counterInstructionFixture(t)
			land := g.PutOnBattlefield(NewLand("Land"), obj.Controller)
			watched := []PermanentVisit{{land.ID(), land.incarnationID}}
			destroy := DestroyEach(MatchingPermanents(IsLand))
			certain, known := false, true
			switch state {
			case "prefix":
				obj.Effects = []Effect{Sequence(Counter(target), destroy)}
				known = false
			case "uncounterable prefix":
				victim.Card = NewInstant("Uncounterable", "", WithUncounterable())
				obj.Effects = []Effect{Sequence(Counter(target), destroy)}
				known = false
			case "suffix":
				obj.Effects = []Effect{Sequence(destroy, Counter(target))}
				certain = true
			case "illegal prefix":
				g.CounterStackObject(victim.ID)
				other := DeclarePlayerTarget("player")
				obj.TargetSpecs = append(obj.TargetSpecs, other.targetSpec())
				obj.Targets = append(obj.Targets, obj.Controller)
				obj.Effects = []Effect{Sequence(Counter(target), destroy)}
				certain = true
			case "unknown suffix":
				obj.Effects = []Effect{Sequence(destroy, Counter(target), destructionInspectionTrap{})}
				known = false
			case "foreign binding":
				obj.Effects = []Effect{Sequence(destroy, Counter(DeclareSpellTarget("spell")))}
				known = false
			case "all illegal":
				g.CounterStackObject(victim.ID)
				obj.Effects = []Effect{Sequence(Counter(target), destroy)}
			}
			for _, game := range []*Game{g, g.Clone()} {
				for range 2 {
					got, gotKnown := game.InspectSpellDestruction(game.StackPeek(), watched)
					if got != certain || gotKnown != known {
						t.Fatalf("got %v/%v want %v/%v", got, gotKnown, certain, known)
					}
				}
				if game.FindPermanent(land.ID()) == nil {
					t.Fatal("inspection destroyed land")
				}
			}
		})
	}
}

func TestCounterInstructionFilterContract(t *testing.T) {
	for _, test := range []struct {
		name   string
		filter SpellFilter
		want   bool
	}{
		{"any", SpellCardMatches(AnyCardFilter()), true},
		{"type", SpellCardMatches(HasCardTypeCardFilter(core.TypeInstant)), true},
		{"not", NotSpellFilter(SpellCardMatches(HasCardTypeCardFilter(core.TypeArtifact))), true},
		{"or", AnySpellFilter(SpellCardMatches(HasCardTypeCardFilter(core.TypeArtifact)), SpellCardMatches(HasCardTypeCardFilter(core.TypeInstant))), true},
		{"and", AllSpellFilters(SpellCardMatches(HasCardTypeCardFilter(core.TypeArtifact)), SpellCardMatches(HasCardTypeCardFilter(core.TypeInstant))), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			g, victim, obj, target := counterInstructionFixture(t)
			target.spec.(*typedSpellTarget).contextualFilter = test.filter
			predicate, ok := InspectSpellTarget(target)
			if !ok {
				t.Fatal("native Boolean tree rejected")
			}
			got, known := predicate.Select(g, obj)
			if !known || (got == victim.ID) != test.want {
				t.Fatal("pure and native filters differ")
			}
			if test.filter.matches(g, obj.Controller, obj.Card, victim) != test.want {
				t.Fatal("runtime filter differs")
			}
		})
	}
	for _, filter := range []SpellFilter{{}, AllSpellFilters(), AnySpellFilter(), NotSpellFilter(SpellFilter{}), SpellWithSingleTarget(), SpellControlledByYou(), SpellCardMatches(NewCardFilter("any", func(Card) bool { panic("filter callback") }))} {
		target := DeclareSpellTarget("spell").WithFilter(filter)
		if _, ok := InspectSpellTarget(target); ok {
			t.Fatal("unsupported filter accepted")
		}
	}
	g, victim, obj, target := counterInstructionFixture(t)
	filter := HasCardTypeCardFilter(core.TypeInstant)
	filter.fn = func(Card) bool { panic("native callback") }
	target.spec.(*typedSpellTarget).contextualFilter = SpellCardMatches(filter)
	predicate, ok := InspectSpellTarget(target)
	if !ok {
		t.Fatal("native plan rejected")
	}
	if got, known := predicate.Select(g, obj); !known || got != victim.ID {
		t.Fatal("native plan not used")
	}
	victim.IsAbility = true
	if got, known := predicate.Select(g, obj); !known || got != uuid.Nil {
		t.Fatal("ability accepted as spell")
	}
}
