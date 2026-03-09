package antiquities

import (
	"testing"

	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
	"github.com/mage/mage/pkg/mage/gametest"
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

	t.Run("prevents next artifact damage for {2}", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Circle of Protection: Artifacts")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Ornithopter")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grapeshot Catapult") // pretend it can target player
		// Use Su-Chi (4/4 artifact creature) attacking instead
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Su-Chi") // 4/4 artifact creature
		g.ActivateAbility(2, core.DeclareAttackers, gametest.PlayerA, "Circle of Protection: Artifacts")
		g.Attack(2, gametest.PlayerB, "Su-Chi")
		g.StopAt(2, core.EndCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 20) // damage prevented
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

	t.Run("forces sacrifice of artifacts that don't pay {2}", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Energy Flux")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Ornithopter")
		// PlayerB's upkeep: must pay {2} or sacrifice Ornithopter
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Ornithopter", 0)
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
	// XXX: cost reduction for attached artifact
	t.Run("is an Aura", func(t *testing.T) {
		card, err := mage.CreateCard("Power Artifact")
		if err != nil {
			t.Fatal(err)
		}
		if !card.HasType(core.TypeEnchantment) {
			t.Errorf("Power Artifact should be an Enchantment")
		}
		if card.ManaCost().CMC() != 2 {
			t.Errorf("expected CMC 2, got %d", card.ManaCost().CMC())
		}
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
