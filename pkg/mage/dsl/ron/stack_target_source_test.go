package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONStackTargetSourcePrerequisite(t *testing.T) {
	for _, name := range []string{"Giant Growth", "Lightning Bolt", "Prodigal Sorcerer"} {
		for _, change := range []string{"live", "departed", "returned", "phased", "phase-cycle", "recolored", "shroud", "copy"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				_, a, b := ronGame()
				chooser := &ronSourceChoicePlayer{BasePlayer: b}
				g := mage.NewGame(a, chooser)
				g.SetStep(core.PrecombatMain)
				g.SetTurn(1)
				targetCard := mage.NewCreature("Target", "{U}", 4, 10)
				targetCard.SetOwner(b.PlayerID())
				target := g.PutOnBattlefield(targetCard, b.PlayerID())
				original := g.CaptureDamageSource(target.ID())
				card := repositoryCard(t, "LEA", name)
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				targets := []uuid.UUID{target.ID()}
				if name == "Prodigal Sorcerer" {
					targets = nil
				}
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
					t.Fatal("unpaid casting changed resources")
				}
				color := core.Green
				if name == "Lightning Bolt" {
					color = core.Red
				}
				if name == "Prodigal Sorcerer" {
					color = core.Blue
					a.ManaPool().Add(core.Colorless, 2)
				}
				a.ManaPool().Add(color, 1)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err != nil {
					t.Fatal(err)
				}
				if name == "Prodigal Sorcerer" {
					g.ResolveStack()
					source := g.FindPermanent(card.ID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err == nil || g.FindPermanent(source.ID()).Tapped || g.StackSize() != 0 {
						t.Fatal("sick activation paid costs")
					}
					g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
						t.Fatal(err)
					}
				}
				if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				switch change {
				case "departed", "returned":
					g.BouncePermanentToHand(g.FindPermanent(target.ID()))
					if change == "returned" {
						b.RemoveFromHand(targetCard.ID())
						g.PutOnBattlefield(targetCard, a.PlayerID())
						g.ChangeObjectColor(targetCard.ID(), core.Red)
					}
				case "phased", "phase-cycle":
					ids := g.PhaseOut(target.ID())
					if change == "phase-cycle" {
						g.PhaseIn(ids)
					}
				case "recolored":
					g.ChangeObjectColor(target.ID(), core.Red)
				case "shroud":
					g.MutablePermanent(target.ID()).GrantBaseAttr(core.Shroud)
				}
				filter, _ := chosenSourceFilter(t, "LEA", "Circle of Protection: Blue")
				chosen := mage.BindDamageSource("stack target")
				fixture := mage.NewEnchantment("Target choice fixture", "", mage.WithAbility(mage.Activate(mage.Costs(), mage.Sequence(
					mage.ChooseDamageSourceFromStackTargets(mage.You, filter, chosen),
					mage.DealDamageFrom(mage.FromDamageSource(chosen), mage.ActivePlayer, mage.Fixed(2))))))
				if err := mage.ValidateCardDefinition(fixture.Name(), fixture); err != nil {
					t.Fatal(err)
				}
				watcher := g.PutOnBattlefield(fixture, b.PlayerID())
				if err := g.ActivateAbilityByIndex(b.PlayerID(), watcher.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				clone := g.Clone()
				if chooser.calls != 0 || a.Life() != 20 {
					t.Fatal("choice ran before resolution")
				}
				g.ResolveTopOfStack()
				wantChoice := 1
				if change == "recolored" {
					wantChoice = 0
				}
				if chooser.calls != wantChoice || a.Life() != 20-2*wantChoice {
					t.Fatal("target choice lost timing or current filter")
				}
				if wantChoice == 1 && (!chooser.source.SameObject(original) || chooser.source.Controller() != b.PlayerID()) {
					t.Fatal("target choice used returned object")
				}
				if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.StackSize() != g.StackSize()+1 {
					t.Fatal("choice changed pending clone")
				}
				g.ResolveStack()
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).Life() != a.Life() || len(g.StackTargetPermanentSources()) != 0 {
					t.Fatal("clone resolution or reference lifetime failed")
				}
				p := g.FindPermanentIncludingPhased(target.ID())
				legal := change == "live" || change == "phase-cycle" || change == "recolored" || change == "copy"
				if (p == nil) != (change == "departed") {
					t.Fatal("RON resolution changed target presence")
				}
				if name == "Giant Growth" {
					want := 4
					if legal {
						want = 7
					}
					if change == "copy" {
						want = 10
					}
					if p != nil && p.CurrentPower(g) != want {
						t.Fatalf("RON growth power: got %d, want %d", p.CurrentPower(g), want)
					}
				} else {
					want := 0
					if legal {
						want = 1
						if name == "Lightning Bolt" {
							want = 3
						}
						if change == "copy" {
							want *= 2
						}
					}
					if p != nil && p.Damage != want {
						t.Fatalf("RON damage: got %d, want %d", p.Damage, want)
					}
				}
			})
		}
	}
}
