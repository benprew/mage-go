package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONStackAbilitySourcePrerequisite(t *testing.T) {
	for _, name := range []string{"Prodigal Sorcerer", "Rod of Ruin"} {
		for _, change := range []string{"live", "departed", "returned", "phased", "phase-cycle", "recolored", "copy"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				_, a, b := ronGame()
				chooser := &ronSourceChoicePlayer{BasePlayer: b}
				g := mage.NewGame(a, chooser)
				g.SetStep(core.PrecombatMain)
				g.SetTurn(1)
				card := repositoryCard(t, "LEA", name)
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
					t.Fatal("unpaid cast changed resources")
				}
				if name == "Prodigal Sorcerer" {
					a.ManaPool().Add(core.Blue, 1)
					a.ManaPool().Add(core.Colorless, 2)
				} else {
					a.ManaPool().Add(core.Colorless, 4)
				}
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				source := g.FindPermanent(card.ID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err == nil || g.FindPermanent(source.ID()).Tapped || g.StackSize() != 0 {
					t.Fatal("unpaid or sick activation changed resources")
				}
				g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
				if name == "Rod of Ruin" {
					a.ManaPool().Add(core.Colorless, 3)
				}
				original := g.CaptureDamageSource(source.ID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
					t.Fatal(err)
				}
				if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("ability copy failed")
				}
				switch change {
				case "departed", "returned":
					g.BouncePermanentToHand(g.FindPermanent(source.ID()))
					if change == "returned" {
						a.RemoveFromHand(card.ID())
						g.PutOnBattlefield(card, b.PlayerID())
						g.ChangeObjectColor(card.ID(), core.Red)
					}
				case "phased", "phase-cycle":
					ids := g.PhaseOut(source.ID())
					if change == "phase-cycle" {
						g.PhaseIn(ids)
					}
				case "recolored":
					g.ChangeObjectColor(source.ID(), core.Red)
				}
				set, shield := "LEA", "Circle of Protection: Blue"
				if name == "Rod of Ruin" {
					set, shield = "ATQ", "Circle of Protection: Artifacts"
				}
				filter, _ := chosenSourceFilter(t, set, shield)
				chosen := mage.BindDamageSource("stack ability source")
				fixture := mage.NewEnchantment("Stack choice fixture", "", mage.WithAbility(mage.Activate(mage.Costs(), mage.Sequence(
					mage.ChooseDamageSourceFromStackAbilities(mage.You, filter, chosen),
					mage.DealDamageFrom(mage.FromDamageSource(chosen), mage.ActivePlayer, mage.Fixed(2))))))
				if err := mage.ValidateCardDefinition(fixture.Name(), fixture); err != nil {
					t.Fatal(err)
				}
				watcher := g.PutOnBattlefield(fixture, b.PlayerID())
				if err := g.ActivateAbilityByIndex(b.PlayerID(), watcher.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				clone := g.Clone()
				if chooser.calls != 0 || a.Life() != 20 || b.Life() != 20 {
					t.Fatal("choice or damage ran before resolution")
				}
				g.ResolveTopOfStack()
				wantChoice := 1
				if name == "Prodigal Sorcerer" && change == "recolored" {
					wantChoice = 0
				}
				if chooser.calls != wantChoice || a.Life() != 20-2*wantChoice {
					t.Fatal("source query used wrong timing or filter")
				}
				if wantChoice == 1 && (!chooser.source.SameObject(original) || chooser.source.Controller() != a.PlayerID()) {
					t.Fatal("choice replaced original source with a returned card")
				}
				if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.StackSize() != g.StackSize()+1 {
					t.Fatal("resolution changed pending clone")
				}
				g.ResolveStack()
				clone.ResolveStack()
				wantLife := 19
				if change == "copy" {
					wantLife = 18
				}
				if b.Life() != wantLife || clone.GetPlayer(b.PlayerID()).Life() != wantLife || clone.GetPlayer(a.PlayerID()).Life() != a.Life() {
					t.Fatal("real RON ability or cloned choice lost its source")
				}
			})
		}
	}
}
