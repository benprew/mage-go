package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONStackBoundDamageSourcePrerequisite(t *testing.T) {
	for _, name := range []string{"Prodigal Sorcerer", "Lightning Bolt"} {
		for _, change := range []string{"none", "departed", "returned", "phased", "cycle", "color", "control", "copy"} {
			if name == "Lightning Bolt" && (change == "phased" || change == "cycle" || change == "control" || change == "color") {
				continue
			}
			t.Run(name+"/"+change, func(t *testing.T) {
				_, a, b := ronGame()
				chooser := &ronSourceChoicePlayer{BasePlayer: b}
				g := mage.NewGame(a, chooser)
				g.SetStep(core.PrecombatMain)
				g.SetTurn(1)
				event, amount := mage.BindDamageSource("event"), mage.BindNumber("damage")
				observer := mage.NewEnchantment("Event observer", "", mage.WithAbility(mage.WhenEvent(mage.ControllerIsDealtDamage(amount).WithSource(event), mage.GainLifeFor(mage.You, mage.Fixed(1)))))
				if err := mage.ValidateCardDefinition(observer.Name(), observer); err != nil {
					t.Fatal(err)
				}
				g.PutOnBattlefield(observer, b.PlayerID())
				card := repositoryCard(t, "LEA", name)
				card.SetOwner(a.PlayerID())
				var saved mage.DamageSource
				var p *mage.Permanent
				if name == "Lightning Bolt" {
					a.AddToHand(card)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 {
						t.Fatal("unpaid casting rollback")
					}
					a.ManaPool().Add(core.Red, 1)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
						t.Fatal(err)
					}
					saved = g.CaptureSpellDamageSource(g.StackPeek().ID)
				} else {
					p = g.PutOnBattlefield(card, a.PlayerID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err == nil || p.Tapped || g.StackSize() != 0 {
						t.Fatal("sickness rollback")
					}
					p.RevokeBaseAttr(core.AttrSummonSick)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
						t.Fatal(err)
					}
					saved = g.CaptureDamageSource(p.ID())
				}
				if chooser.calls != 0 || b.Life() != 20 {
					t.Fatal("early effect")
				}
				g.ResolveTopOfStack()
				if g.StackSize() != 1 {
					t.Fatal("damage trigger was not placed after native resolution")
				}
				g.PutTriggersOnStack()
				got := g.StackBoundDamageSources()
				if len(got) != 1 || !got[0].SameObject(saved) {
					t.Fatal("native RON damage event capture")
				}
				if name == "Prodigal Sorcerer" {
					switch change {
					case "departed", "returned":
						g.BouncePermanentToHand(g.FindPermanent(p.ID()))
						if change == "returned" {
							a.RemoveFromHand(card.ID())
							g.PutOnBattlefield(card, b.PlayerID())
							g.ChangeObjectColor(card.ID(), core.Red)
						}
					case "phased", "cycle":
						ids := g.PhaseOut(p.ID())
						if change == "cycle" {
							g.PhaseIn(ids)
						}
					case "color":
						g.ChangeObjectColor(p.ID(), core.Red)
					case "control":
						if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ChangeController(mage.This, mage.You)), p.ID(), b.PlayerID(), nil); err != nil {
							t.Fatal(err)
						}
					}
				} else if change == "returned" {
					a.RemoveFromGraveyard(card.ID())
					a.AddToHand(card)
					a.ManaPool().Add(core.Red, 1)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
						t.Fatal(err)
					}
					g.ChangeObjectColor(g.StackPeek().ID, core.Blue)
				}
				if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				filterName := "Circle of Protection: Blue"
				if name == "Lightning Bolt" {
					filterName = "Circle of Protection: Red"
				}
				filter, _ := chosenSourceFilter(t, "LEA", filterName)
				chosen := mage.BindDamageSource("chosen")
				fixture := mage.NewArtifact("Choice fixture", "", mage.WithAbility(mage.Activate(mage.Costs(), mage.Sequence(mage.ChooseDamageSourceFromStackDamageBindings(mage.You, filter, chosen), mage.DealDamageFrom(mage.FromDamageSource(chosen), mage.ActivePlayer, mage.Fixed(2))))))
				if err := mage.ValidateCardDefinition(fixture.Name(), fixture); err != nil {
					t.Fatal(err)
				}
				watcher := g.PutOnBattlefield(fixture, b.PlayerID())
				if err := g.ActivateAbilityByIndex(b.PlayerID(), watcher.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				clone := g.Clone()
				g.ResolveTopOfStack()
				want := 1
				if change == "color" {
					want = 0
				}
				if chooser.calls != want || a.Life() != 20-2*want {
					t.Fatal("resolution choice/filter or actual damage")
				}
				if want == 1 && !chooser.source.SameObject(saved) {
					t.Fatal("returned object replaced event source")
				}
				if clone.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatal("clone changed")
				}
				g.ResolveStack()
				clone.ResolveStack()
				for _, game := range []*mage.Game{g, clone} {
					life := 20
					if name == "Lightning Bolt" {
						life = 18
						if change == "returned" {
							life = 16
						}
					}
					activeLife := 20 - 2*want
					if change == "copy" {
						activeLife++
					}
					if game.GetPlayer(a.PlayerID()).Life() != activeLife {
						t.Fatal("copied trigger controller or selected damage")
					}
					if game.GetPlayer(b.PlayerID()).Life() != life || len(game.StackBoundDamageSources()) != 0 {
						t.Fatalf("RON gameplay/lifetime: life=%d want=%d", game.GetPlayer(b.PlayerID()).Life(), life)
					}
				}
			})
		}
	}
}
