package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestActivationHistory(t *testing.T) {
	for _, change := range []string{"pending", "countered", "copy", "clone", "return", "phase", "control", "reset", "unpaid"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			history := BindActivationHistory("pump")
			card := NewCreature("History", "", 2, 5, WithAbility(NewAction(ActionActivated, WithCost(ManaCostOf("{R}")), WithActivationHistory(history), WithEffect(IfActivationsAtLeast(history, 4, GainLifeFor(You, Fixed(1)))))))
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			a.ManaPool().Add(Red, 4)
			for range 3 {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			}
			if change != "pending" && change != "unpaid" {
				g.ResolveStack()
			}
			if a.Life() != 20 {
				t.Fatal("threshold before fourth activation")
			}
			clone := g.Clone()
			payer := a
			switch change {
			case "countered":
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.GetStack().Pop()
				a.ManaPool().Add(Red, 1)
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(card.ID())
				p = g.PutOnBattlefield(card, a.PlayerID())
			case "phase":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
				payer = b
				b.ManaPool().Add(Red, 1)
			case "reset":
				g.trackers.Turn.ResetForNewTurn(b.PlayerID())
			case "unpaid":
				a.ManaPool().Clear()
			}
			err := g.ActivateAbilityByIndex(payer.PlayerID(), p.ID(), 0, nil)
			if change == "unpaid" {
				if err == nil {
					t.Fatal("unpaid activation")
				}
				g.ResolveStack()
				if a.Life() != 20 {
					t.Fatal("failed payment changed activation history")
				}
				a.ManaPool().Add(Red, 1)
				if err = g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if change == "copy" {
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
			}
			g.ResolveStack()
			want := 21
			if change == "return" || change == "reset" {
				want = 20
			}
			if change == "pending" {
				want = 24
			}
			if payer.Life() != want {
				t.Fatalf("life %d want %d", payer.Life(), want)
			}
			if change == "copy" && b.Life() != 21 {
				t.Fatal("copy lost original history")
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("clone changed")
			}
			if err := clone.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			clone.ResolveStack()
			wantClone := 21
			if change == "pending" || change == "unpaid" {
				wantClone = 24
			}
			if clone.GetPlayer(a.PlayerID()).Life() != wantClone {
				t.Fatal("clone lost history")
			}
		})
	}
}

func TestActivationHistoryDefinition(t *testing.T) {
	h := BindActivationHistory("history")
	for _, options := range [][]ActionOption{
		{WithEffect(IfActivationsAtLeast(h, 4, GainLifeFor(You, Fixed(1))))},
		{WithActivationHistory(nil)},
		{WithActivationHistory(BindActivationHistory(""))},
		{WithActivationHistory(h), WithActivationHistory(h)},
		{WithActivationHistory(h), WithEffect(IfActivationsAtLeast(nil, 4, GainLifeFor(You, Fixed(1))))},
		{WithActivationHistory(h), WithEffect(IfActivationsAtLeast(h, -1, GainLifeFor(You, Fixed(1))))},
	} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(NewAction(ActionActivated, options...)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid history accepted")
		}
	}
	card := NewCreature("Valid", "", 1, 1, WithAbility(NewAction(ActionActivated, WithActivationHistory(h), WithEffect(IfActivationsAtLeast(h, 4, GainLifeFor(You, Fixed(1)))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestActivationHistoryIndependentDeclarations(t *testing.T) {
	h := BindActivationHistory("history")
	action := func() *ActionDefinition {
		return NewAction(ActionActivated, WithActivationHistory(h), WithEffect(IfActivationsAtLeast(h, 2, GainLifeFor(You, Fixed(1)))))
	}
	card := NewCreature("Invalid", "", 1, 1, WithAbility(action()), WithAbility(action()))
	if ValidateCardDefinition(card.Name(), card) == nil {
		t.Fatal("history shared by two abilities")
	}
}

func TestActivationHistorySeparateAbilitiesAndSacrificeCost(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first, second := BindActivationHistory("first"), BindActivationHistory("second")
	card := NewCreature("Source", "", 1, 3,
		WithAbility(NewAction(ActionActivated, WithActivationHistory(first), WithEffect(IfActivationsAtLeast(first, 2, GainLifeFor(You, Fixed(1)))))),
		WithAbility(NewAction(ActionActivated, WithActivationHistory(second), WithCost(SacrificeSourceCost()), WithEffect(IfActivationsAtLeast(second, 1, GainLifeFor(You, Fixed(2)))))))
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 22 || g.FindPermanent(p.ID()) != nil {
		t.Fatal("sacrifice cost lost original history or abilities share history")
	}
}

func TestActivationHistoryRepeatedDeclaration(t *testing.T) {
	h := BindActivationHistory("history")
	action := NewAction(ActionActivated, WithActivationHistory(h), WithEffect(IfActivationsAtLeast(h, 2, GainLifeFor(You, Fixed(1)))))
	card := NewCreature("Invalid", "", 1, 1, WithAbility(action), WithAbility(action))
	if ValidateCardDefinition(card.Name(), card) == nil {
		t.Fatal("history declaration used twice")
	}
}
