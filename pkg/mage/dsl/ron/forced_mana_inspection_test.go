package ron

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONForcedManaInspection(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, state := range []string{"normal", "activated body", "empty", "phased", "phase cycle", "returned", "control", "copy", "countered"} {
			t.Run(set+"/"+state, func(t *testing.T) {
				a := &ronForcedManaPlayer{Player: mage.NewBasePlayer("A"), color: core.White}
				b := &ronForcedManaPlayer{Player: mage.NewBasePlayer("B"), color: core.Green}
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				land := g.PutOnBattlefield(mage.NewLand("Choice", mage.WithManaAbility(core.Red), mage.WithManaAbility(core.AnyColor)), b.PlayerID())
				b.ManaPool().AddRestricted(core.Colorless, 2, core.ArtifactSpellsOnly{})
				card := repositoryCard(t, set, "Drain Power")
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				a.ManaPool().Add(core.Blue, 2)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
					t.Fatal(err)
				}
				want := 3
				if state == "activated body" {
					g.RemoveFromBattlefield(land)
					land = g.PutOnBattlefield(mage.NewLand("Activated", mage.WithAbility(mage.Activate(mage.Costs(mage.TapCost(mage.This)), mage.AddMana(core.Green, 2)))), b.PlayerID())
					want = 4
				}
				switch state {
				case "empty":
					b.ManaPool().Clear()
					g.TapPermanent(land)
					want = 0
				case "phased":
					g.PhaseOut(land.ID())
					want = 2
				case "phase cycle":
					g.PhaseIn(g.PhaseOut(land.ID()))
				case "returned":
					g.BouncePermanentToHand(land)
					b.RemoveFromHand(land.Card.ID())
					g.PutOnBattlefield(land.Card, b.PlayerID())
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: land.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
					g.ApplyContinuousEffects()
					want = 2
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				beforeA, beforeB := a.ManaPool().SnapshotPool(), b.ManaPool().SnapshotPool()
				for _, game := range []*mage.Game{g, g.Clone()} {
					for range 2 {
						if certain, known := game.InspectSpellDestruction(game.StackPeek(), nil); certain || !known {
							t.Fatal("forced mana tree is unknown")
						}
					}
				}
				if a.modes != 0 || b.modes != 0 || !reflect.DeepEqual(beforeA, a.ManaPool().SnapshotPool()) || !reflect.DeepEqual(beforeB, b.ManaPool().SnapshotPool()) {
					t.Fatal("inspection executed decisions or changed mana")
				}
				if state == "countered" {
					g.CounterSpellOnStack(g.StackPeek().ID)
					if b.ManaPool().TotalMana() != 2 || g.FindPermanent(land.ID()).Tapped {
						t.Fatal("countered spell ran")
					}
					return
				}
				clone := g.Clone()
				g.ResolveStack()
				clone.ResolveStack()
				for _, game := range []*mage.Game{g, clone} {
					pool := game.GetPlayer(a.PlayerID()).ManaPool()
					if pool.TotalMana() != want || game.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != 0 {
						t.Fatalf("mana=%d want=%d", pool.TotalMana(), want)
					}
					if want >= 2 && pool.CanPay(core.ManaCost{Generic: want}, nil) {
						t.Fatal("transfer lost restrictions")
					}
					if pool.TotalProducedThisTurn() != 2 {
						t.Fatal("transfer produced mana")
					}
				}
			})
		}
	}
}
