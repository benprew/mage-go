package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestComposableTargetUntapPartialAndLifetime(t *testing.T) {
	for _, change := range []string{"none", "return", "phase", "phase cycle", "absent"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
			for _, c := range []Card{first, second} {
				g.PutOnBattlefield(c, b.PlayerID())
				g.TapPermanent(g.FindPermanent(c.ID()))
			}
			x, y := DeclarePermanentTarget("first", IsCreature), DeclarePermanentTarget("second", IsCreature)
			ability := Spell(Sequence(UntapPermanent(x), UntapPermanent(y)), WithTargets(x, y))
			card := NewInstant("Untap", "", WithAbility(ability))
			obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID(), second.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
			g.pushStack(obj)
			switch change {
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
				g.PutOnBattlefield(first, b.PlayerID())
				g.TapPermanent(g.FindPermanent(first.ID()))
			case "phase":
				g.PhaseOut(first.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(first.ID()))
			case "absent":
				g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
			}
			clone := g.Clone()
			clone.ResolveStack()
			if change == "phase" {
				clone.PhaseIn([]uuid.UUID{first.ID()})
			}
			wantTapped := change == "return" || change == "phase" || change == "absent"
			if (clone.FindPermanent(first.ID()) != nil && clone.FindPermanent(first.ID()).Tapped != wantTapped) || clone.FindPermanent(second.ID()).Tapped {
				t.Fatalf("incorrect target binding or legality: first=%+v second tapped=%v", clone.FindPermanent(first.ID()), clone.FindPermanent(second.ID()).Tapped)
			}
			if !g.FindPermanent(second.ID()).Tapped {
				t.Fatal("clone changed original")
			}
		})
	}
}

func TestComposableRandomUntapBinding(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	for _, c := range []Card{first, second} {
		g.PutOnBattlefield(c, a.PlayerID())
		g.TapPermanent(g.FindPermanent(c.ID()))
	}
	saved := BindPermanent("random")
	body := Sequence(ChooseRandomPermanent(MatchingPermanents(And(IsCreature, IsTapped)), saved), UntapPermanent(From(saved)))
	card := NewSorcery("Random", "", WithAbility(Spell(body)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.SetRandomResults([]int{1, 0, 7})
	clone := g.Clone()
	if err := ApplyEffect(clone, body, card.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if clone.FindPermanent(second.ID()).Tapped || !clone.FindPermanent(first.ID()).Tapped || !g.FindPermanent(second.ID()).Tapped {
		t.Fatal("random choice or clone isolation failed")
	}
	for range 3 {
		if err := ApplyEffect(g, body, card.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if g.FindPermanent(first.ID()).Tapped || g.FindPermanent(second.ID()).Tapped || g.RandIntn(10) != 7 {
		t.Fatal("empty choice consumed randomness or reused a binding")
	}
}

func TestComposableLegacyTriggerTypedUntap(t *testing.T) {
	target := DeclarePermanentTarget("creature", IsCreature)
	trigger := NewTriggered(EvtUpkeep, false, UntapPermanent(target)).WithTargets(target)
	card := NewCreature("Trigger", "", 1, 1, WithAbility(trigger))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(card, a.PlayerID())
	g.TapPermanent(g.FindPermanent(card.ID()))
	g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
	resolvePendingTriggers(g)
	if g.FindPermanent(card.ID()).Tapped {
		t.Fatal("trigger lost typed target")
	}
}

func TestComposableTargetUntapCopyRetarget(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	for _, c := range []Card{first, second} {
		g.PutOnBattlefield(c, b.PlayerID())
		g.TapPermanent(g.FindPermanent(c.ID()))
	}
	target := DeclarePermanentTarget("creature", IsCreature)
	ability := Spell(UntapPermanent(target), WithTargets(target))
	card := NewInstant("Untap", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	a.chooseQueue = [][]uuid.UUID{{second.ID()}}
	copied := g.CopySpellOnStack(card.ID(), a.PlayerID(), true)
	g.ResolveStackObject(copied)
	if g.FindPermanent(second.ID()).Tapped || !g.FindPermanent(first.ID()).Tapped {
		t.Fatal("copy used original binding")
	}
	g.ResolveStackObject(obj)
	if g.FindPermanent(first.ID()).Tapped {
		t.Fatal("copy changed original binding")
	}
}

func TestComposableRandomUntapValidationAndAbsence(t *testing.T) {
	saved := BindPermanent("chosen")
	for _, body := range []Effect{
		Sequence(UntapPermanent(From(saved)), ChooseRandomPermanent(MatchingPermanents(IsCreature), saved)),
		Sequence(ChooseRandomPermanent(nil, saved), UntapPermanent(From(saved))),
		ChooseRandomPermanent(MatchingPermanents(IsCreature), nil),
		UntapPermanent(DeclarePermanentTarget("undeclared", IsCreature)),
	} {
		card := NewSorcery("Invalid", "", WithAbility(Spell(body)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid binding")
		}
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Creature", "", 1, 1)
	g.PutOnBattlefield(c, a.PlayerID())
	g.TapPermanent(g.FindPermanent(c.ID()))
	body := Sequence(Snapshot(This, saved), ChooseRandomPermanent(MatchingPermanents(IsLand), saved), UntapPermanent(From(saved)))
	if err := ApplyEffect(g, body, c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(c.ID()).Tapped {
		t.Fatal("empty random choice retained previous binding")
	}
}
