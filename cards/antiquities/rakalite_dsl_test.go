package antiquities

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRakaliteDelayedSource(t *testing.T) {
	for _, change := range []string{"none", "return before resolution", "return before end step", "return on delayed stack", "absent", "phased", "phase cycle", "control", "activate in end step", "illegal target"} {
		t.Run(change, func(t *testing.T) {
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			card, err := mage.CreateCard("Rakalite")
			if err != nil {
				t.Fatal(err)
			}
			card.SetOwner(b.PlayerID())
			g.PutOnBattlefield(card, a.PlayerID())
			target := mage.NewCreature("Recipient", "", 4, 4)
			target.SetOwner(b.PlayerID())
			g.PutOnBattlefield(target, b.PlayerID())
			if change == "activate in end step" {
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			}
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			if g.StackSize() != 1 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("activation must use the stack and pay two mana")
			}
			returnSource := func() {
				g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
				g.PutOnBattlefield(card, a.PlayerID())
			}
			if change == "return before resolution" {
				returnSource()
			}
			if change == "illegal target" {
				g.RemoveFromBattlefield(g.FindPermanent(target.ID()))
			}
			g.ResolveStack()
			if change == "illegal target" {
				if len(g.DelayedTriggers()) != 0 {
					t.Fatal("illegal target created a delayed trigger")
				}
				return
			}
			if len(g.DelayedTriggers()) != 1 {
				t.Fatal("expected one future end-step trigger")
			}
			g.DealDamageToPermanent(g.FindPermanent(target.ID()), 2, card.ID())
			if g.FindPermanent(target.ID()).Damage != 1 {
				t.Fatal("shield did not prevent exactly one damage")
			}
			switch change {
			case "return before end step":
				returnSource()
			case "absent":
				g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
			case "phased":
				g.PhaseOut(card.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(card.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: target.ID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || g.StackPeek().Controller != a.PlayerID() || len(g.StackPeek().TargetSpecs) != 0 {
				t.Fatal("delayed trigger must retain its controller and have no target")
			}
			if change == "return on delayed stack" {
				returnSource()
			}
			clone := g.Clone()
			clone.ResolveStack()
			wantHand := 0
			if change == "none" || change == "phase cycle" || change == "control" || change == "activate in end step" {
				wantHand = 1
			}
			if got := len(clone.GetPlayer(b.PlayerID()).Hand()); got != wantHand {
				t.Fatalf("owner hand = %d, want %d", got, wantHand)
			}
			if len(clone.GetPlayer(a.PlayerID()).Hand()) != 0 || len(b.Hand()) != 0 {
				t.Fatal("return used controller's hand or changed the original game")
			}
			if change != "absent" && change != "phased" && wantHand == 0 && clone.FindPermanent(card.ID()) == nil {
				t.Fatal("delayed return followed a new incarnation")
			}
			clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			clone.PutTriggersOnStack()
			if clone.StackSize() != 0 || len(clone.DelayedTriggers()) != 0 {
				t.Fatal("delayed trigger repeated")
			}
		})
	}
}

func TestRakaliteTargetsAndPayment(t *testing.T) {
	for _, kind := range []string{"own creature", "opposing creature", "own player", "opposing player", "planeswalker", "artifact"} {
		t.Run(kind, func(t *testing.T) {
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			card, err := mage.CreateCard("Rakalite")
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			ability := mage.UnwrapAbility(source.RuntimeAbilities[0]).(mage.ActivatedAbility)
			if info := mage.InspectDefinition(ability.Effects()[0]); info.Role != "sequence" {
				t.Fatalf("expected typed sequence, got %s", info.Role)
			}
			targetID := b.PlayerID()
			if kind == "own player" {
				targetID = a.PlayerID()
			}
			if kind == "own creature" || kind == "opposing creature" || kind == "planeswalker" || kind == "artifact" {
				var target mage.Card = mage.NewCreature("Recipient", "", 4, 4)
				if kind == "planeswalker" {
					target = mage.NewPlaneswalker("Recipient", "", 4)
				}
				if kind == "artifact" {
					target = mage.NewArtifact("Recipient", "")
				}
				owner := b.PlayerID()
				if kind == "own creature" {
					owner = a.PlayerID()
				}
				target.SetOwner(owner)
				g.PutOnBattlefield(target, owner)
				targetID = target.ID()
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, []uuid.UUID{targetID}); err == nil {
				t.Fatal("activation accepted insufficient mana")
			}
			if a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
				t.Fatal("failed payment changed mana or stack")
			}
			a.ManaPool().Add(core.Colorless, 1)
			err = g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, []uuid.UUID{targetID})
			if kind == "artifact" {
				if err == nil || a.ManaPool().TotalMana() != 2 {
					t.Fatal("accepted invalid target or spent mana")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if player := g.GetPlayer(targetID); player != nil {
				g.DealDamageToPlayer(player, 2, card.ID())
				if player.Life() != 19 {
					t.Fatal("player shield failed")
				}
			} else {
				amount := 2
				if kind == "planeswalker" {
					amount = 1
				}
				g.DealDamageToPermanent(g.FindPermanent(targetID), amount, card.ID())
				recipient := g.FindPermanent(targetID)
				if recipient.Damage != amount-1 {
					t.Fatal("permanent shield failed")
				}
				if kind == "planeswalker" && recipient.Counters[core.Loyalty] != 4 {
					t.Fatal("prevented damage changed loyalty")
				}
			}
		})
	}
}
