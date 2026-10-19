package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestGraveyardHandInstructionDeclaration(t *testing.T) {
	target := DeclareGraveyardCardTarget("card")
	instruction := GraveyardHandInstruction{Subject: target}
	effect := instruction.Effect()
	got, ok := InspectGraveyardHandInstruction(effect)
	if !ok || got.Subject != target {
		t.Fatal("lost native subject")
	}
	var missing *CardTargetBinding
	for _, subject := range []CardRef{nil, missing} {
		if (GraveyardHandInstruction{Subject: subject}).Effect() != nil {
			t.Fatal("accepted missing subject")
		}
	}
	for _, effect := range []ResolvingEffect{nil, (*returnCardToHandEffect)(nil), ReturnCardToHand(nil), Sequence(instruction.Effect(), destructionInspectionTrap{})} {
		if _, ok := InspectGraveyardHandInstruction(effect); ok {
			t.Fatal("accepted incomplete instruction")
		}
	}
}

func graveyardHandInspectionFixture(t *testing.T) (*Game, *BasePlayer, *CardTargetBinding, Card, *StackObject) {
	t.Helper()
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	victim := NewCreature("Victim", "", 1, 1)
	victim.SetOwner(a.PlayerID())
	a.AddToGraveyard(victim)
	target := DeclareGraveyardCardTarget("card", GraveyardCardFilter(HasCardTypeCardFilter(core.TypeCreature)))
	_, obj := graveyardTestSpell(a.PlayerID(), target, (GraveyardHandInstruction{Subject: target}).Effect())
	obj.Targets = []uuid.UUID{victim.ID()}
	g.pushStack(obj)
	return g, a, target, victim, obj
}

func TestGraveyardHandInstructionInspection(t *testing.T) {
	for _, state := range []string{"normal", "absent", "returned", "foreign", "missing zone", "missing visit", "opaque", "opaque child", "optional", "mana bound", "duplicate", "clone", "copy"} {
		t.Run(state, func(t *testing.T) {
			g, a, target, victim, obj := graveyardHandInspectionFixture(t)
			wantKnown := true
			switch state {
			case "absent", "returned":
				card, _ := a.RemoveFromGraveyard(victim.ID())
				if state == "returned" {
					a.AddToGraveyard(card)
				} else {
					a.AddToHand(card)
				}
			case "foreign":
				obj.Effects = []Effect{ReturnCardToHand(DeclareGraveyardCardTarget("card"))}
				wantKnown = false
			case "missing zone":
				obj.TargetZones = nil
				wantKnown = false
			case "missing visit":
				obj.targetIncarnations = nil
				wantKnown = false
			case "opaque", "opaque child":
				trap := NewCardFilter("trap", func(Card) bool { panic("inspection ran card callback") })
				if state == "opaque child" {
					trap = OrCardFilter(AnyCardFilter(), trap)
				}
				target.spec.(*GraveyardCardTarget).Filters = []CardFilter{trap}
				wantKnown = false
			case "optional":
				target.spec.(*GraveyardCardTarget).min = 0
				wantKnown = false
			case "mana bound":
				target.spec = &manaValueBoundGraveyardTarget{inner: target.spec, maximum: FixedTargetNumber(2), optionCount: 1}
				wantKnown = false
			case "duplicate":
				obj.TargetSpecs = append(obj.TargetSpecs, obj.TargetSpecs[0])
				obj.Targets = append(obj.Targets, victim.ID())
				wantKnown = false
			case "clone":
				g = g.Clone()
				obj = g.StackPeek()
			case "copy":
				obj = g.CopyStackObjectDirect(obj, obj.Controller, false)
			}
			for range 2 {
				if certain, known := g.InspectSpellDestruction(obj, nil); certain || known != wantKnown {
					t.Fatalf("certainty=%v/%v want=false/%v", certain, known, wantKnown)
				}
			}
		})
	}
}

func TestGraveyardHandInstructionOrder(t *testing.T) {
	for _, state := range []string{"prefix", "suffix", "illegal prefix", "foreign suffix", "opaque suffix"} {
		t.Run(state, func(t *testing.T) {
			g, a, target, victim, obj := graveyardHandInspectionFixture(t)
			land := g.PutOnBattlefield(NewLand("Land"), a.PlayerID())
			destroy := DestroyEach(MatchingPermanents(IsLand))
			returnEffect := ReturnCardToHand(target)
			obj.Effects = []Effect{Sequence(returnEffect, destroy)}
			want, known := false, false
			switch state {
			case "suffix":
				obj.Effects = []Effect{Sequence(destroy, returnEffect)}
				want, known = true, true
			case "illegal prefix":
				a.RemoveFromGraveyard(victim.ID())
				obj.Effects = append(obj.Effects, GainLifeFor(You, Fixed(1)))
				known = true
				player := DeclarePlayerTarget("player")
				obj.TargetSpecs = append(obj.TargetSpecs, player.targetSpec())
				obj.Targets = append(obj.Targets, a.PlayerID())
				obj.TargetZones[a.PlayerID()] = core.ZoneAny
				want = true
			case "foreign suffix":
				obj.Effects = []Effect{Sequence(destroy, ReturnCardToHand(DeclareGraveyardCardTarget("card")))}
			case "opaque suffix":
				obj.Effects = []Effect{Sequence(destroy, destructionInspectionTrap{})}
			}
			if got, gotKnown := g.InspectSpellDestruction(obj, []PermanentVisit{{land.ID(), land.incarnationID}}); got != want || gotKnown != known {
				t.Fatalf("certainty=%v/%v want=%v/%v", got, gotKnown, want, known)
			}
			if g.FindPermanent(land.ID()) == nil {
				t.Fatal("inspection destroyed land")
			}
		})
	}
}

func TestGraveyardHandInstructionTargetContract(t *testing.T) {
	for _, state := range []string{"own", "other graveyard", "any graveyard", "wrong type", "wrong zone", "missing declaration", "typed callback", "zero predicate"} {
		t.Run(state, func(t *testing.T) {
			g, a, target, victim, obj := graveyardHandInspectionFixture(t)
			want := victim.ID()
			switch state {
			case "other graveyard", "any graveyard":
				card, _ := a.RemoveFromGraveyard(victim.ID())
				card.SetOwner(g.players[1].PlayerID())
				g.players[1].AddToGraveyard(card)
				obj.targetIncarnations[victim.ID()] = g.graveyardVisit(victim.ID())
				if state == "any graveyard" {
					target.spec = TargetCardInAnyGraveyard(HasCardTypeCardFilter(core.TypeCreature))
				} else {
					want = uuid.Nil
				}
			case "wrong type":
				target.spec.(*GraveyardCardTarget).Filters = []CardFilter{HasCardTypeCardFilter(core.TypeLand)}
				want = uuid.Nil
			case "wrong zone":
				obj.TargetZones[victim.ID()] = core.ZoneHand
				want = uuid.Nil
			case "typed callback":
				target.spec.(*GraveyardCardTarget).Filters[0].fn = func(Card) bool { panic("native callback ran") }
			}
			predicate, ok := InspectGraveyardTarget(target)
			if !ok {
				t.Fatal("native declaration unknown")
			}
			known := true
			if state == "missing declaration" {
				obj.TargetSpecs = nil
				obj.Targets = nil
				want = uuid.Nil
				known = false
			}
			if state == "zero predicate" {
				predicate = GraveyardTargetPredicate{}
				want = uuid.Nil
				known = false
			}
			before := g.Clone()
			for range 2 {
				id, gotKnown := predicate.Select(g, obj)
				if id != want || gotKnown != known {
					t.Fatalf("selected=%v/%v want=%v/%v", id, gotKnown, want, known)
				}
			}
			if len(a.Hand()) != 0 || g.StackSize() != before.StackSize() || g.graveyardVisit(victim.ID()) != before.graveyardVisit(victim.ID()) {
				t.Fatal("query changed zones or stack")
			}
			if state == "any graveyard" {
				g.ResolveTopOfStack()
				if len(a.Hand()) != 0 || len(g.players[1].Hand()) != 1 {
					t.Fatal("return used graveyard holder instead of owner")
				}
			}
		})
	}
}
