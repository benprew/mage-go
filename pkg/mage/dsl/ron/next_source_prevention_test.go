package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONNextPermanentSourcePrevention(t *testing.T) {
	for _, name := range []string{"Prodigal Sorcerer", "Orcish Artillery"} {
		for _, change := range []string{"normal", "return", "phase", "phase-cycle", "control", "copy"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "LEA", name)
				if err := mage.ValidateCardDefinition(name, card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err == nil || g.StackSize() != 0 || source.Tapped {
					t.Fatal("sick activation changed state")
				}
				g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
					t.Fatal(err)
				}
				target := mage.DeclarePermanentTarget("chosen source", mage.IsCreature)
				protector := g.PutOnBattlefield(mage.NewArtifact("Protection fixture", "", mage.WithAbility(mage.Activate(mage.Costs(), mage.PreventNextDamageFromPermanent(target, mage.You, mage.PermanentFilter{}), mage.WithTargets(target)))), b.PlayerID())
				if err := g.ActivateAbilityByIndex(b.PlayerID(), protector.ID(), 0, []uuid.UUID{source.ID()}); err != nil {
					t.Fatal(err)
				}
				g.ResolveTopOfStack()
				if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				switch change {
				case "return":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					returned := g.PutOnBattlefield(card, a.PlayerID())
					g.DealDamageToPlayer(b, 1, returned.ID())
					if b.Life() != 19 {
						t.Fatal("returned source consumed shield")
					}
				case "phase":
					g.PhaseOut(source.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(source.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				clone := g.Clone()
				for _, game := range []*mage.Game{g, clone} {
					game.ResolveStack()
					want := 20
					if change == "return" {
						want = 19
					}
					amount := 1
					if name == "Orcish Artillery" {
						amount = 2
					}
					if change == "copy" {
						want -= amount
					}
					if game.GetPlayer(b.PlayerID()).Life() != want {
						t.Fatalf("recipient life=%d want=%d", game.GetPlayer(b.PlayerID()).Life(), want)
					}
					own := 20
					if name == "Orcish Artillery" {
						own = 17
						if change == "copy" {
							own = 14
						}
					}
					if game.GetPlayer(a.PlayerID()).Life() != own {
						t.Fatal("shield changed an independent simultaneous recipient")
					}
				}
			})
		}
	}
}

func TestRONNextPermanentSourcePreventionArtillerySelf(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEA", "Orcish Artillery"), a.PlayerID())
	g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if err := mage.ApplyEffect(g, mage.PreventNextDamageFromPermanent(mage.This, mage.You, mage.PermanentFilter{}), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("RON simultaneous self damage consumed shield per packet")
	}
	g.DealDamageToPlayer(a, 1, source.ID())
	if a.Life() != 19 {
		t.Fatal("RON damage did not consume shield")
	}
}
