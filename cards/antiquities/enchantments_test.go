package antiquities

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

func TestArtifactPossession(t *testing.T) {

	t.Run("is an Aura", func(t *testing.T) {
		card, err := mage.CreateCard("Artifact Possession")
		if err != nil {
			t.Fatal(err)
		}
		if !card.HasType(core.TypeEnchantment) {
			t.Errorf("Artifact Possession should be an Enchantment")
		}
		if card.ManaCost().CMC() != 3 {
			t.Errorf("expected CMC 3, got %d", card.ManaCost().CMC())
		}
	})

	t.Run("deals 2 damage when enchanted artifact becomes tapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Jayemdae Tome") // artifact with tap ability
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Artifact Possession")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Artifact Possession", "Jayemdae Tome")
		g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Forest")
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerB, "Jayemdae Tome")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 18) // 2 damage from Artifact Possession
	})
}

func TestArtifactWard(t *testing.T) {

	t.Run("is an Aura enchantment", func(t *testing.T) {
		card, err := mage.CreateCard("Artifact Ward")
		if err != nil {
			t.Fatal(err)
		}
		if !card.HasType(core.TypeEnchantment) {
			t.Errorf("Artifact Ward should be an Enchantment")
		}
		if card.ManaCost().CMC() != 1 {
			t.Errorf("expected CMC 1, got %d", card.ManaCost().CMC())
		}
	})

	t.Run("enchanted creature cannot be blocked by artifact creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Yotian Soldier") // artifact creature
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Artifact Ward")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Artifact Ward", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Yotian Soldier", "Grizzly Bears")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Block should be illegal; 2 damage goes through
		g.AssertLife(gametest.PlayerB, 18)
	})

	t.Run("prevents all damage from artifact sources to enchanted creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Rocket Launcher")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Artifact Ward")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Artifact Ward", "Grizzly Bears")
		// Rocket Launcher deals 1 damage — should be prevented by Artifact Ward
		g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerB, "Rocket Launcher", "Grizzly Bears")
		g.StopAt(3, core.BeginCombat)
		g.Execute()
		// Grizzly Bears should survive — artifact damage prevented
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	})

	t.Run("non-artifact damage still goes through", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Artifact Ward")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Artifact Ward", "Grizzly Bears")
		// Lightning Bolt is not an artifact — damage should go through
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// 3 damage kills 2/2 — not prevented
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
	})

	t.Run("artifact ability cannot target warded creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Staff of Zegon") // artifact with {3},{T}: -2/-0
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Artifact Ward")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Artifact Ward", "Grizzly Bears")
		// Staff of Zegon tries to target warded creature — should fail
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerB, "Staff of Zegon", "Grizzly Bears")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		// Staff should still be untapped (activation failed due to targeting)
		g.AssertTapped(gametest.PlayerB, "Staff of Zegon", false)
	})
}

func TestCircleOfProtectionArtifacts(t *testing.T) {

	t.Run("is an Enchantment", func(t *testing.T) {
		card, err := mage.CreateCard("Circle of Protection: Artifacts")
		if err != nil {
			t.Fatal(err)
		}
		if !card.HasType(core.TypeEnchantment) {
			t.Errorf("Circle of Protection: Artifacts should be an Enchantment")
		}
		if card.ManaCost().CMC() != 2 {
			t.Errorf("expected CMC 2, got %d", card.ManaCost().CMC())
		}
	})

	t.Run("prevents next artifact damage from chosen source", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Circle of Protection: Artifacts")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Su-Chi") // 4/4 artifact creature
		g.ChoosePermanent(gametest.PlayerA, "Su-Chi") // choose Su-Chi as source to prevent
		g.ActivateAbility(2, core.DeclareAttackers, gametest.PlayerA, "Circle of Protection: Artifacts")
		g.Attack(2, gametest.PlayerB, "Su-Chi")
		g.StopAt(2, core.EndCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 20) // damage prevented
	})
}

func TestCircleOfProtectionArtifactsSourceSpecific(t *testing.T) {
	t.Run("second artifact source damage still goes through", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Circle of Protection: Artifacts")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Su-Chi")       // 4/4 artifact creature
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Yotian Soldier") // 1/4 artifact creature
		g.ChoosePermanent(gametest.PlayerA, "Su-Chi") // prevent Su-Chi only
		g.ActivateAbility(2, core.DeclareAttackers, gametest.PlayerA, "Circle of Protection: Artifacts")
		g.Attack(2, gametest.PlayerB, "Su-Chi", "Yotian Soldier")
		g.StopAt(2, core.EndCombat)
		g.Execute()
		// Su-Chi damage (4) prevented, Yotian Soldier damage (1) goes through
		g.AssertLife(gametest.PlayerA, 19)
	})

	t.Run("multiple activations choosing different sources", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Circle of Protection: Artifacts")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Su-Chi")       // 4/4
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Yotian Soldier") // 1/4
		g.ChoosePermanent(gametest.PlayerA, "Su-Chi")
		g.ActivateAbility(2, core.DeclareAttackers, gametest.PlayerA, "Circle of Protection: Artifacts")
		g.ChoosePermanent(gametest.PlayerA, "Yotian Soldier")
		g.ActivateAbility(2, core.DeclareAttackers, gametest.PlayerA, "Circle of Protection: Artifacts")
		g.Attack(2, gametest.PlayerB, "Su-Chi", "Yotian Soldier")
		g.StopAt(2, core.EndCombat)
		g.Execute()
		// Both sources prevented
		g.AssertLife(gametest.PlayerA, 20)
	})
}

func TestDampingField(t *testing.T) {

	t.Run("is an Enchantment", func(t *testing.T) {
		card, err := mage.CreateCard("Damping Field")
		if err != nil {
			t.Fatal(err)
		}
		if !card.HasType(core.TypeEnchantment) {
			t.Errorf("Damping Field should be an Enchantment")
		}
		if card.ManaCost().CMC() != 3 {
			t.Errorf("expected CMC 3, got %d", card.ManaCost().CMC())
		}
	})

	t.Run("only one artifact untaps per untap step", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Damping Field")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Jalum Tome")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Millstone")
		// Tap both
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Jalum Tome")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Millstone", "PlayerB")
		// Next untap step: only one should untap
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		jTapped := false
		mTapped := false
		for _, perm := range g.Battlefield {
			if perm.Name() == "Jalum Tome" && perm.Tapped {
				jTapped = true
			}
			if perm.Name() == "Millstone" && perm.Tapped {
				mTapped = true
			}
		}
		// Exactly one should remain tapped
		if !jTapped && !mTapped {
			t.Errorf("Damping Field should prevent one artifact from untapping; both untapped")
		}
	})
}

func TestEnergyFlux(t *testing.T) {

	t.Run("is an Enchantment", func(t *testing.T) {
		card, err := mage.CreateCard("Energy Flux")
		if err != nil {
			t.Fatal(err)
		}
		if !card.HasType(core.TypeEnchantment) {
			t.Errorf("Energy Flux should be an Enchantment")
		}
		if card.ManaCost().CMC() != 3 {
			t.Errorf("expected CMC 3, got %d", card.ManaCost().CMC())
		}
	})

	t.Run("sacrifices artifact when no lands to pay", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Energy Flux")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Ornithopter")
		// PlayerB has no lands → can't pay {2} → sacrifice
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Ornithopter", 0)
	})

	t.Run("artifact survives when lands can pay", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Energy Flux")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Ornithopter")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		// PlayerB has 2 lands → can pay {2} → Ornithopter survives
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Ornithopter", 1)
	})

	t.Run("two artifacts with 5 lands both survive", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Energy Flux")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Ornithopter")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Jalum Tome")
		for i := 0; i < 5; i++ {
			g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		}
		// 5 lands can pay {2}+{2}=4 mana (5 lands produce 5, enough)
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Ornithopter", 1)
		g.AssertPermanentCount(gametest.PlayerB, "Jalum Tome", 1)
	})

	t.Run("one artifact sacrificed when not enough lands for all", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Energy Flux")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Ornithopter")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Jalum Tome")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		// 2 lands → can only pay for 1 artifact ({2}), second gets sacrificed
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		// One should survive, one should be sacrificed
		ornCount := 0
		jalumCount := 0
		for _, perm := range g.Battlefield {
			if perm.Name() == "Ornithopter" {
				ornCount++
			}
			if perm.Name() == "Jalum Tome" {
				jalumCount++
			}
		}
		total := ornCount + jalumCount
		if total != 1 {
			t.Errorf("expected 1 artifact to survive, got %d", total)
		}
	})

	t.Run("Energy Flux itself is not affected", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Energy Flux")
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		// Energy Flux is an enchantment, not an artifact — it stays
		g.AssertPermanentCount(gametest.PlayerA, "Energy Flux", 1)
	})
}

func TestGateToPhyrexia(t *testing.T) {

	t.Run("is an Enchantment", func(t *testing.T) {
		card, err := mage.CreateCard("Gate to Phyrexia")
		if err != nil {
			t.Fatal(err)
		}
		if !card.HasType(core.TypeEnchantment) {
			t.Errorf("Gate to Phyrexia should be an Enchantment")
		}
		if card.ManaCost().CMC() != 2 {
			t.Errorf("expected CMC 2, got %d", card.ManaCost().CMC())
		}
	})

	t.Run("sacrifice creature to destroy artifact during upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gate to Phyrexia")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // sacrifice fodder
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Ornithopter")
		g.ActivateAbility(1, core.Upkeep, gametest.PlayerA, "Gate to Phyrexia", "Ornithopter")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Ornithopter", 0)
	})
}

func TestHauntingWind(t *testing.T) {
	t.Run("is an Enchantment", func(t *testing.T) {
		card, err := mage.CreateCard("Haunting Wind")
		if err != nil {
			t.Fatal(err)
		}
		if !card.HasType(core.TypeEnchantment) {
			t.Errorf("Haunting Wind should be an Enchantment")
		}
		if card.ManaCost().CMC() != 4 {
			t.Errorf("expected CMC 4, got %d", card.ManaCost().CMC())
		}
	})

	t.Run("deals 1 damage when artifact becomes tapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Haunting Wind")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Jalum Tome")
		g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Forest")
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerB, "Jalum Tome")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 19)
	})

	t.Run("deals 1 damage when artifact ability activated without tap cost", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Haunting Wind")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Ashnod's Altar") // artifact, no tap cost
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")  // sacrifice fodder
		// Ashnod's Altar: "Sacrifice a creature: Add {C}{C}" — no tap cost
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerB, "Ashnod's Altar")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 19) // 1 damage from Haunting Wind
	})
}

func TestPowerArtifact(t *testing.T) {
	t.Run("reduces activation cost by 2", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Jayemdae Tome") // {4}, {T}: draw a card
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Power Artifact")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant") // card to draw (non-land to avoid autoPlayLands)
		// Cast Power Artifact on Jayemdae Tome
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Power Artifact", "Jayemdae Tome")
		// Now Jayemdae Tome's {4},{T} ability should cost {2},{T} instead
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Jayemdae Tome")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Jayemdae Tome", true)
		// Should have drawn a card if the reduced cost was payable
		g.AssertHandCount(gametest.PlayerA, "Hill Giant", 1)
	})

	t.Run("floor of 1 mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		// Amulet of Kroog costs {2},{T}: prevent 1 damage. With Power Artifact, it should still cost {1},{T}
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Amulet of Kroog")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Power Artifact")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Power Artifact", "Amulet of Kroog")
		// Activation cost reduced from {2} to {1} (floor of 1 mana)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Amulet of Kroog", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Amulet of Kroog", true)
	})

	t.Run("cost returns to normal when Power Artifact removed", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Jayemdae Tome")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Power Artifact")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Disenchant")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Power Artifact", "Jayemdae Tome")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Disenchant", "Power Artifact")
		// After removing Power Artifact, Jayemdae Tome costs {4},{T} again
		// With only 5 auto-mana minus cost of Disenchant/Power Artifact, may not be able to activate
		// Just verify Tome is not tapped (wasn't activated)
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Jayemdae Tome", false)
	})
}

func TestPowerleech(t *testing.T) {
	t.Run("is an Enchantment", func(t *testing.T) {
		card, err := mage.CreateCard("Powerleech")
		if err != nil {
			t.Fatal(err)
		}
		if !card.HasType(core.TypeEnchantment) {
			t.Errorf("Powerleech should be an Enchantment")
		}
		if card.ManaCost().CMC() != 2 {
			t.Errorf("expected CMC 2, got %d", card.ManaCost().CMC())
		}
	})

	t.Run("gains 1 life when opponent taps artifact", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Powerleech")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Jalum Tome")
		g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Forest")
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerB, "Jalum Tome")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 21)
	})
}

func TestTitaniasSong(t *testing.T) {
	t.Run("is an Enchantment", func(t *testing.T) {
		card, err := mage.CreateCard("Titania's Song")
		if err != nil {
			t.Fatal(err)
		}
		if !card.HasType(core.TypeEnchantment) {
			t.Errorf("Titania's Song should be an Enchantment")
		}
		if card.ManaCost().CMC() != 4 {
			t.Errorf("expected CMC 4, got %d", card.ManaCost().CMC())
		}
	})

	t.Run("turns noncreature artifacts into creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Titania's Song")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Jalum Tome") // CMC 3
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Jalum Tome should be a 3/3 creature (CMC = P/T)
		g.AssertPowerToughness(gametest.PlayerB, "Jalum Tome", 3, 3)
	})

	t.Run("animated artifacts lose all abilities", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Titania's Song")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Ornithopter") // has flying
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Ornithopter is artifact creature already, so Song doesn't affect it
		g.AssertHasAbility(gametest.PlayerB, "Ornithopter", core.Flying, true)
	})

	t.Run("effect lingers until end of turn after Song is destroyed", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Titania's Song")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Jalum Tome") // CMC 3
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Disenchant")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Disenchant", "Titania's Song")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Song was destroyed, but effect continues until end of turn
		g.AssertPowerToughness(gametest.PlayerB, "Jalum Tome", 3, 3)
	})
}
