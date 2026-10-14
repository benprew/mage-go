package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestReturnedPermanentEntryResult(t *testing.T) {
	for _, mode := range []string{"normal", "blocked", "clear", "depart", "phase", "phase-cycle"} {
		t.Run(mode, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewLand("Guest")
			card.SetOwner(b.PlayerID())
			b.AddToGraveyard(card)
			target := DeclareGraveyardCardTarget("guest", InAnyGraveyard())
			saved := BindPermanent("returned guest")
			body := Sequence(ReturnCardToBattlefieldInto(target, You, saved, EnterTapped()), AtNext(EachEndStep, TapSubject(From(saved))))
			if mode == "clear" {
				body = Sequence(ReturnCardToBattlefieldInto(target, You, saved), ReturnCardToBattlefieldInto(target, You, saved), TapSubject(From(saved)))
			}
			spell, obj := graveyardTestSpell(a.PlayerID(), target, body)
			if err := ValidateCardDefinition("Return", spell); err != nil {
				t.Fatal(err)
			}
			obj.Targets = []uuid.UUID{card.ID()}
			g.pushStack(obj)
			if mode == "blocked" {
				g.effects.Rules.SetLandsCantEnter(true)
			}
			g.ResolveTopOfStack()
			p := g.FindPermanent(card.ID())
			if mode == "blocked" {
				if p != nil || len(b.Graveyard()) != 1 {
					t.Fatal("blocked entry moved its card")
				}
				return
			}
			if p == nil || p.ControllerID() != a.PlayerID() || p.Card.Owner() != b.PlayerID() {
				t.Fatal("entry lost ownership or control")
			}
			if mode == "clear" {
				if p.Tapped {
					t.Fatal("failed entry retained the prior result")
				}
				return
			}
			if !p.Tapped {
				t.Fatal("entry option was lost")
			}
			g.MutablePermanent(p.ID()).Tapped = false
			clone := g.Clone()
			switch mode {
			case "depart":
				g.BouncePermanentToHand(p)
				b.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			}
			for _, game := range []*Game{g, clone} {
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
			}
			if !clone.FindPermanent(card.ID()).Tapped {
				t.Fatal("clone lost the saved entry result")
			}
			if mode == "phase" {
				g.PhaseIn([]uuid.UUID{card.ID()})
			}
			if g.FindPermanent(card.ID()).Tapped != (mode != "depart" && mode != "phase") {
				t.Fatal("saved result followed the wrong visit")
			}
		})
	}
}

func TestReturnedPermanentDefinition(t *testing.T) {
	target := DeclareGraveyardCardTarget("guest")
	saved := BindPermanent("returned guest")
	for _, body := range []ResolvingEffect{
		ReturnCardToBattlefieldInto(target, You, nil),
		Sequence(TapSubject(From(saved)), ReturnCardToBattlefieldInto(target, You, saved)),
	} {
		card := NewSorcery("Invalid", "", WithAbility(Spell(body, WithTargets(target))))
		if err := ValidateCardDefinition("Invalid", card); err == nil {
			t.Fatal("invalid entry result definition accepted")
		}
	}
}

func TestReturnedPermanentEntryChoiceVisit(t *testing.T) {
	a := &numericChoicePlayer{BasePlayer: NewBasePlayer("A"), answer: 3}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	guest := numericChoiceCard()
	guest.SetOwner(a.PlayerID())
	a.AddToGraveyard(guest)
	a.during = func() {
		moved, ok := g.MoveFromGraveyard(a.PlayerID(), guest.ID(), core.ZoneHand)
		if !ok {
			t.Fatal("entry choice could not move the card")
		}
		a.AddToGraveyard(moved)
	}
	target := DeclareGraveyardCardTarget("guest")
	saved := BindPermanent("entrant")
	_, obj := graveyardTestSpell(a.PlayerID(), target, Sequence(
		ReturnCardToBattlefieldInto(target, You, saved),
		AtNext(EachEndStep, GainLifeFor(You, PowerOfWithLKI(From(saved)))),
	))
	obj.Targets = []uuid.UUID{guest.ID()}
	g.pushStack(obj)
	g.ResolveTopOfStack()
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.calls != 1 || g.FindPermanent(guest.ID()) != nil || len(a.Graveyard()) != 2 || a.Life() != 20 {
		t.Fatalf("entry result followed a changed graveyard visit: calls=%d permanent=%v graveyard=%d life=%d", a.calls, g.FindPermanent(guest.ID()), len(a.Graveyard()), a.Life())
	}
}

func TestReturnedPermanentLKIAndWaiting(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	guest := NewCreature("Guest", "", 3, 4)
	guest.SetOwner(a.PlayerID())
	a.AddToGraveyard(guest)
	target := DeclareGraveyardCardTarget("guest")
	saved := BindPermanent("entrant")
	_, obj := graveyardTestSpell(a.PlayerID(), target, Sequence(
		ReturnCardToBattlefieldInto(target, You, saved),
		AtNext(EachEndStep, Sequence(TapSubject(From(saved)), GainLifeFor(You, PowerOfWithLKI(From(saved))))),
	))
	obj.Targets = []uuid.UUID{guest.ID()}
	g.pushStack(obj)
	g.ResolveTopOfStack()
	p := g.FindPermanent(guest.ID())
	exact := g.CaptureDamageSource(p.ID())
	check := func() {
		t.Helper()
		for range 2 {
			refs := g.WaitingDelayedPermanentSources()
			if len(refs) != 1 || !refs[0].SameObject(exact) {
				t.Fatal("waiting query lost the entry visit")
			}
		}
	}
	check()
	g.PhaseOut(p.ID())
	check()
	g.PhaseIn([]uuid.UUID{p.ID()})
	g.BouncePermanentToHand(p)
	check()
	a.RemoveFromHand(guest.ID())
	returned := g.PutOnBattlefield(guest, a.PlayerID())
	g.MutablePermanent(returned.ID()).Counters[core.P1P1] = 5
	check()
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 23 || g.FindPermanent(guest.ID()).Tapped {
		t.Fatal("saved entry LKI or live action used the new visit")
	}
}
