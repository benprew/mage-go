package antiquities

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// ===== INSTANTS =====

func TestArtifactBlast(t *testing.T) {
	t.Run("counters artifact spell", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Ornithopter")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Artifact Blast")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Ornithopter")
		g.CastInResponseTo(gametest.PlayerA, "Artifact Blast")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Ornithopter", 0)
		g.AssertGraveyardCount(gametest.PlayerB, "Ornithopter", 1)
	})

	t.Run("costs {R}", func(t *testing.T) {
		card, err := mage.CreateCard("Artifact Blast")
		if err != nil {
			t.Fatal(err)
		}
		if card.ManaCost().CMC() != 1 {
			t.Errorf("expected CMC 1, got %d", card.ManaCost().CMC())
		}
	})
}

func TestCrumble(t *testing.T) {
	t.Run("destroys artifact and controller gains life equal to CMC", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Su-Chi") // CMC 4 artifact creature
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Crumble")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Crumble", "Su-Chi")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Su-Chi", 0)
		g.AssertLife(gametest.PlayerB, 24) // gains 4 life (CMC of Su-Chi)
	})

	t.Run("cannot regenerate destroyed artifact", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Clay Statue") // has regen
		g.ActivateAbility(2, core.Upkeep, gametest.PlayerB, "Clay Statue") // set up regen shield
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Crumble")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerA, "Crumble", "Clay Statue")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Clay Statue", 0) // can't regen
	})
}

func TestHurkyrlsRecall(t *testing.T) {
	t.Run("returns all artifacts target player owns to hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Ornithopter")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Yotian Soldier")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // not an artifact
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Hurkyl's Recall")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Hurkyl's Recall", "PlayerB")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Ornithopter", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Yotian Soldier", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1) // stays
		g.AssertHandCount(gametest.PlayerB, "Ornithopter", 1)
		g.AssertHandCount(gametest.PlayerB, "Yotian Soldier", 1)
	})
}

func TestReversePolarity(t *testing.T) {
	t.Run("costs {W}{W}", func(t *testing.T) {
		card, err := mage.CreateCard("Reverse Polarity")
		if err != nil {
			t.Fatal(err)
		}
		if card.ManaCost().CMC() != 2 {
			t.Errorf("expected CMC 2, got %d", card.ManaCost().CMC())
		}
	})

	t.Run("gains life equal to twice artifact damage dealt this turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Su-Chi") // 4/4 artifact creature
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Reverse Polarity")
		g.Attack(2, gametest.PlayerB, "Su-Chi")
		// After taking 4 damage from artifact, cast Reverse Polarity
		g.CastSpell(2, core.PostcombatMain, gametest.PlayerA, "Reverse Polarity")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// 4 artifact damage → gain 2*4 = 8 life → net: 20-4+8 = 24
		g.AssertLife(gametest.PlayerA, 24)
	})
}

// ===== SORCERIES =====

func TestDetonate(t *testing.T) {
	t.Run("destroys artifact and deals X damage to controller", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Ornithopter") // CMC 0
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Detonate")
		g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Detonate", 0, "Ornithopter")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Ornithopter", 0)
		g.AssertLife(gametest.PlayerB, 20) // X=0, no damage
	})

	t.Run("deals X damage equal to mana value", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Jalum Tome") // CMC 3
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Detonate")
		g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Detonate", 3, "Jalum Tome")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Jalum Tome", 0)
		g.AssertLife(gametest.PlayerB, 17) // 3 damage
	})
}

func TestDrafnasRestoration(t *testing.T) {
	t.Run("costs {U}", func(t *testing.T) {
		card, err := mage.CreateCard("Drafna's Restoration")
		if err != nil {
			t.Fatal(err)
		}
		if card.ManaCost().CMC() != 1 {
			t.Errorf("expected CMC 1, got %d", card.ManaCost().CMC())
		}
	})

	t.Run("puts artifact cards from graveyard on top of library", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Ornithopter")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Yotian Soldier")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Drafna's Restoration")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Drafna's Restoration", "PlayerA")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Ornithopter", 0)
		g.AssertGraveyardCount(gametest.PlayerA, "Yotian Soldier", 0)
	})
}

func TestReconstruction(t *testing.T) {
	t.Run("returns artifact from graveyard to hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Ornithopter")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Reconstruction")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Reconstruction", "Ornithopter")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Ornithopter", 0)
		g.AssertHandCount(gametest.PlayerA, "Ornithopter", 1)
	})
}

func TestShatterstorm(t *testing.T) {
	t.Run("destroys all artifacts", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ornithopter")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Yotian Soldier")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Su-Chi")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // not an artifact
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Shatterstorm")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Shatterstorm")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Ornithopter", 0)
		g.AssertPermanentCount(gametest.PlayerA, "Yotian Soldier", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Su-Chi", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1) // non-artifact stays
	})
}

func TestTransmuteArtifact(t *testing.T) {
	t.Run("costs {U}{U}", func(t *testing.T) {
		card, err := mage.CreateCard("Transmute Artifact")
		if err != nil {
			t.Fatal(err)
		}
		if card.ManaCost().CMC() != 2 {
			t.Errorf("expected CMC 2, got %d", card.ManaCost().CMC())
		}
	})

	t.Run("free when found artifact CMC <= sacrificed CMC", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Su-Chi")  // CMC 4, sacrifice this
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Jalum Tome")  // CMC 3, find this
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Transmute Artifact")
		g.ChooseFromLibrary(gametest.PlayerA, "Jalum Tome")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Transmute Artifact")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Su-Chi", 0)           // sacrificed
		g.AssertPermanentCount(gametest.PlayerA, "Jalum Tome", 1)       // on battlefield for free
	})

	t.Run("free when found artifact CMC equals sacrificed CMC", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Jalum Tome") // CMC 3, sacrifice
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Staff of Zegon") // CMC 4... need CMC 3
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Coral Helm")     // CMC 3, find this
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Transmute Artifact")
		g.ChooseFromLibrary(gametest.PlayerA, "Coral Helm")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Transmute Artifact")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Jalum Tome", 0)   // sacrificed
		g.AssertPermanentCount(gametest.PlayerA, "Coral Helm", 1)   // on battlefield
	})

	t.Run("more expensive goes to graveyard when cannot pay difference", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ornithopter") // CMC 0, sacrifice
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Su-Chi")          // CMC 4, find this
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Transmute Artifact")
		g.ChooseFromLibrary(gametest.PlayerA, "Su-Chi")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Transmute Artifact")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Ornithopter", 0)    // sacrificed
		g.AssertPermanentCount(gametest.PlayerA, "Su-Chi", 0)         // NOT on battlefield
		g.AssertGraveyardCount(gametest.PlayerA, "Su-Chi", 1)         // went to graveyard
	})
}
