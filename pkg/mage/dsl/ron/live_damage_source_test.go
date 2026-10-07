package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONLiveDamageSourcePrerequisite(t *testing.T) {
	for _, scenario := range []string{"live", "departed", "returned", "phased", "phase-cycle", "recolored", "shroud", "copy", "clone"} {
		t.Run(scenario, func(t *testing.T) {
			_, a, b := ronGame()
			chooser := &ronSourceChoicePlayer{BasePlayer: b}
			g := mage.NewGame(a, chooser)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			filter, _ := chosenSourceFilter(t, "LEA", "Circle of Protection: Blue")
			chosen := mage.BindDamageSource("live source")
			fixture := mage.NewEnchantment("Live choice fixture", "", mage.WithAbility(mage.WhenEvent(
				mage.ControllerIsDealtDamage(mage.BindNumber("damage")),
				mage.Sequence(mage.ChooseDamageSourceFromLiveObjects(mage.You, filter, chosen), mage.DealDamageFrom(mage.FromDamageSource(chosen), mage.ActivePlayer, mage.Fixed(2))))))
			if err := mage.ValidateCardDefinition(fixture.Name(), fixture); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(fixture, b.PlayerID())
			card := repositoryCard(t, "LEA", "Prodigal Sorcerer")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			original := g.CaptureDamageSource(source.ID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveTopOfStack()
			g.PutTriggersOnStack()
			if chooser.calls != 0 || b.Life() != 19 || g.StackSize() != 1 {
				t.Fatal("choice ran before trigger resolution")
			}
			switch scenario {
			case "departed", "returned":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				if scenario == "returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phased":
				g.PhaseOut(source.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "recolored":
				g.ChangeObjectColor(source.ID(), core.Red)
			case "shroud":
				g.MutablePermanent(source.ID()).GrantBaseAttr(core.Shroud)
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			want := 1
			switch scenario {
			case "departed", "phased", "recolored":
				want = 0
			case "copy":
				want = 2
			}
			if chooser.calls != want || a.Life() != 20-2*want || b.Life() != 19 {
				t.Fatal("live choice timing, filter or continuation")
			}
			if want > 0 {
				if !chooser.source.HasColor(core.Blue) || (chooser.source.SameObject(original) == (scenario == "returned")) {
					t.Fatal("live source identity")
				}
			}
			pending := 1
			if scenario == "copy" {
				pending = 2
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.StackSize() != pending {
				t.Fatal("choice changed pending clone")
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 20-2*want {
				t.Fatal("clone lost choice declaration")
			}
		})
	}
}

func TestRONLiveDamageSourceSpell(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEA", "Lightning Bolt")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	spell := g.StackPeek()
	filter, _ := chosenSourceFilter(t, "LEA", "Circle of Protection: Red")
	chosen := mage.BindDamageSource("live spell")
	fixture := mage.NewArtifact("Spell choice fixture", "", mage.WithAbility(mage.Activate(mage.Costs(), mage.Sequence(
		mage.ChooseDamageSourceFromLiveObjects(mage.You, filter, chosen), mage.DealDamageFrom(mage.FromDamageSource(chosen), mage.You, mage.Fixed(2))))))
	source := g.PutOnBattlefield(fixture, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.ResolveTopOfStack()
	if b.Life() != 18 || g.StackSize() != 1 || g.StackPeek().ID != spell.ID {
		t.Fatal("choice did not read live RON spell")
	}
	g.ResolveStack()
	clone.ResolveStack()
	if b.Life() != 15 || clone.GetPlayer(b.PlayerID()).Life() != 15 {
		t.Fatal("RON spell or copied choice lost damage")
	}
}
