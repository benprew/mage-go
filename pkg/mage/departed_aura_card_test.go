package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDepartedAuraCard(t *testing.T) {
	for _, state := range []string{"sba", "aura-first", "host-first", "new-visit", "manual-death", "bounce", "clone-pending", "copy", "phase", "host-return", "aura-return", "exile-host", "removed", "control", "exile-aura", "regenerate-host", "token", "removed-after", "reattach"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			hostCard := NewCreature("Host", "", 2, 2)
			hostCard.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			binding := BindCard("departed aura")
			card := NewAura("Watcher", "", WithCastTarget(TargetPermanent(AnyPermanent)), WithAbility(WhenEvent(AttachedPermanentDies().WithDepartedAuraCard(binding), ReturnCardToHand(FromCard(binding)))))
			card.SetOwner(b.PlayerID())
			aura := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(aura.ID(), host.ID())
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			if state == "removed" {
				g.AddContinuousEffect(RemoveAllAbilities(aura.ID(), core.Indefinite))
			}
			if state == "control" {
				g.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: aura.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			if state == "phase" {
				g.PhaseOut(aura.ID())
			}
			if state == "regenerate-host" {
				g.AddRegenerationShield(host.ID())
			}
			if state == "exile-aura" {
				if err := ApplyEffect(g, ExileIfWouldDieThisTurn(This), aura.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			if state == "token" {
				g.MutablePermanent(aura.ID()).IsToken = true
			}
			switch state {
			case "aura-first":
				g.DestroyPermanents([]*Permanent{aura, host})
			case "exile-host":
				g.ExilePermanent(host)
			case "host-first":
				g.DestroyPermanents([]*Permanent{host, aura})
			default:
				g.DestroyPermanent(host)
			}
			if state == "removed-after" {
				g.AddContinuousEffect(RemoveAllAbilities(aura.ID(), core.Indefinite))
			}
			if state == "reattach" {
				g.Attach(aura.ID(), g.PutOnBattlefield(NewCreature("Other", "", 2, 2), b.PlayerID()).ID())
			}
			if state == "manual-death" {
				g.DestroyPermanent(g.FindPermanent(aura.ID()))
			}
			if state == "bounce" {
				g.BouncePermanentToHand(g.FindPermanent(aura.ID()))
				b.RemoveFromHand(card.ID())
				b.AddToGraveyard(card)
			}
			if state == "clone-pending" {
				clone := g.Clone()
				clone.CheckStateBasedActions()
				clone.PutTriggersOnStack()
				clone.ResolveStack()
				if len(clone.GetPlayer(b.PlayerID()).Hand()) != 1 || len(b.Hand()) != 0 {
					t.Fatal("pending capture shares clone state")
				}
			}
			if state == "host-return" {
				b.RemoveFromGraveyard(hostCard.ID())
				g.PutOnBattlefield(hostCard, b.PlayerID())
			}
			g.CheckStateBasedActions()
			if state == "aura-return" {
				b.RemoveFromGraveyard(card.ID())
				returned := g.PutOnBattlefield(card, b.PlayerID())
				g.Attach(returned.ID(), g.PutOnBattlefield(NewCreature("New host", "", 2, 2), b.PlayerID()).ID())
				g.DestroyPermanent(returned)
			}
			if state == "new-visit" {
				b.RemoveFromGraveyard(card.ID())
				b.AddToGraveyard(card)
			}
			g.PutTriggersOnStack()
			if state == "copy" {
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			want := 1
			if state == "new-visit" || state == "manual-death" || state == "bounce" || state == "phase" || state == "exile-host" || state == "removed" || state == "aura-return" || state == "exile-aura" || state == "regenerate-host" || state == "token" || state == "reattach" {
				want = 0
			}
			if len(b.Hand()) != want || len(a.Hand()) != 0 || len(clone.GetPlayer(b.PlayerID()).Hand()) != want {
				t.Fatalf("hand=%d want=%d", len(b.Hand()), want)
			}
		})
	}
}

func TestDepartedAuraCardInvalid(t *testing.T) {
	binding := BindCard("aura")
	for _, pattern := range []creatureDiesPattern{AttachedPermanentDies().WithDepartedAuraCard(nil), AttachedPermanentDies().WithCard(binding).WithDepartedAuraCard(binding), PermanentDies().WithDepartedAuraCard(binding), AttachedPermanentDies().WithDepartedAuraCard(binding).WithDepartedAuraCard(binding)} {
		card := NewAura("Invalid", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid capture accepted")
		}
	}
}

func TestDepartedAuraCardInstances(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	hostCard := NewCreature("Host", "", 2, 2)
	hostCard.SetOwner(b.PlayerID())
	host := g.PutOnBattlefield(hostCard, b.PlayerID())
	cards := make([]Card, 0, 2)
	for range 2 {
		binding := BindCard("departed Aura")
		card := NewAura("Watcher", "", WithCastTarget(TargetPermanent(AnyPermanent)), WithAbility(WhenEvent(AttachedPermanentDies().WithDepartedAuraCard(binding), ReturnCardToHand(FromCard(binding)))))
		card.SetOwner(a.PlayerID())
		aura := g.PutOnBattlefield(card, b.PlayerID())
		g.Attach(aura.ID(), host.ID())
		cards = append(cards, card)
	}
	g.DestroyPermanent(host)
	g.CheckStateBasedActions()
	a.RemoveFromGraveyard(cards[0].ID())
	a.AddToGraveyard(cards[0])
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack=%d", g.StackSize())
	}
	g.ResolveStack()
	if len(a.Hand()) != 1 || a.Hand()[0].ID() != cards[1].ID() {
		t.Fatal("Aura captures share a card visit")
	}
}
