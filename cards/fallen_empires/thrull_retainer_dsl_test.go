package fallen_empires

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestThrullRetainerTypedActivation(t *testing.T) {
	for _, scenario := range []string{"normal", "shroud", "wrong controller", "tapped", "source return", "host return", "phased host", "copy clone"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
			host := g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
			other := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
			aura := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Thrull Retainer")
			auraCard, hostCard := g.FindPermanent(aura).Card, g.FindPermanent(host).Card
			g.Attach(aura, host)
			g.ApplyContinuousEffects()
			if scenario == "shroud" {
				g.FindPermanent(host).GrantBaseAttr(core.Shroud)
			}
			if scenario == "tapped" {
				g.FindPermanent(aura).Tapped = true
			}
			actor := a
			if scenario == "wrong controller" {
				actor = b
			}
			err := g.ActivateAbilityByIndex(actor.PlayerID(), aura, 1, nil)
			if scenario == "wrong controller" {
				if err == nil || g.FindPermanent(aura) == nil || g.StackSize() != 0 {
					t.Fatal("invalid activation paid sacrifice")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(aura) != nil || g.StackSize() != 1 || len(g.StackPeek().Targets) != 0 || g.HasRegenerationShield(host) {
				t.Fatal("incorrect sacrifice, target, or shield timing")
			}
			g.AssertGraveyardCount(gametest.PlayerA, "Thrull Retainer", 1)
			wantHost := true
			switch scenario {
			case "source return":
				a.RemoveFromGraveyard(aura)
				g.PutOnBattlefield(auraCard, a.PlayerID())
				g.Attach(aura, other)
			case "host return":
				g.RemoveFromBattlefield(g.FindPermanent(host))
				g.PutOnBattlefield(hostCard, b.PlayerID())
				wantHost = false
			case "phased host":
				g.PhaseOut(host)
				wantHost = false
			case "copy clone":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				clone := g.Clone()
				clone.ResolveStack()
				if !clone.HasRegenerationShield(host) || g.HasRegenerationShield(host) || g.StackSize() != 2 {
					t.Fatal("clone lost capture or changed original")
				}
			}
			g.ResolveStack()
			if g.HasRegenerationShield(host) != wantHost || g.HasRegenerationShield(other) || g.HasRegenerationShield(aura) {
				t.Fatal("wrong shield recipient")
			}
			if !wantHost {
				return
			}
			p := g.FindPermanent(host)
			p.Damage = 1
			g.DestroyPermanent(p)
			p = g.FindPermanent(host)
			if p == nil || !p.Tapped || p.Damage != 0 {
				t.Fatal("shield did not replace destruction")
			}
			if scenario == "copy clone" {
				if !g.HasRegenerationShield(host) {
					t.Fatal("copy did not install its own shield")
				}
				g.DestroyPermanent(p)
			}
			if g.HasRegenerationShield(host) {
				t.Fatal("shield not consumed")
			}
		})
	}
}

func TestThrullRetainerTypedDefinition(t *testing.T) {
	card, err := mage.CreateCard("Thrull Retainer")
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	var capture, regenerate bool
	var visit func(mage.DefinitionMetadata)
	visit = func(n mage.DefinitionMetadata) {
		capture = capture || n.Role == "activation-permanent-capture"
		regenerate = regenerate || n.Properties.Regenerates
		for _, child := range n.Children {
			visit(child)
		}
	}
	visit(mage.InspectDefinition(card.Abilities()[1]))
	if !capture || !regenerate {
		t.Fatalf("capture=%v regeneration=%v", capture, regenerate)
	}
}
