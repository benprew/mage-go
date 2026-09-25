package limited

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDiskTypedActivation(t *testing.T) {
	for _, sourceLeaves := range []bool{false, true} {
		name := "source remains"
		if sourceLeaves {
			name = "source leaves"
		}
		t.Run(name, func(t *testing.T) {
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			card, err := mage.CreateCard("Nevinyrral's Disk")
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			disk := g.PutOnBattlefield(card, a.PlayerID())
			ability := mage.UnwrapAbility(disk.RuntimeAbilities[0]).(mage.ActivatedAbility)
			info := mage.InspectDefinition(ability.Effects()[0])
			if info.Role != "destroy-each" || !info.Properties.Destroys || !info.Properties.Mass || len(ability.Targets()) != 0 {
				t.Fatalf("wrong destruction definition: %+v", info)
			}
			if !disk.Tapped {
				t.Fatal("Disk did not enter tapped")
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err == nil {
				t.Fatal("tapped Disk activated")
			}
			g.UntapPermanent(disk)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(card.ID()).Tapped || g.StackSize() != 1 {
				t.Fatal("activation did not pay tap cost and use the stack")
			}
			if sourceLeaves {
				g.ExilePermanent(g.FindPermanent(card.ID()))
			}
			victim := mage.NewCreature("Regenerating artifact creature", "", 2, 2)
			victim.AddType(core.TypeArtifact)
			victim.SetOwner(b.PlayerID())
			g.PutOnBattlefield(victim, b.PlayerID())
			g.AddRegenerationShield(victim.ID())
			enchantment := mage.NewEnchantment("Enchantment", "")
			enchantment.SetOwner(b.PlayerID())
			g.PutOnBattlefield(enchantment, b.PlayerID())
			g.ResolveStack()
			if g.FindPermanent(card.ID()) != nil || g.FindPermanent(enchantment.ID()) != nil {
				t.Fatal("Disk did not destroy the matching permanents")
			}
			if p := g.FindPermanent(victim.ID()); p == nil || !p.Tapped {
				t.Fatal("one shield did not save the artifact creature")
			}
			if !sourceLeaves {
				returned := g.PutOnBattlefield(card, a.PlayerID())
				g.UntapPermanent(returned)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err == nil {
					t.Fatal("activation did not spend its mana")
				}
			}
		})
	}
}
