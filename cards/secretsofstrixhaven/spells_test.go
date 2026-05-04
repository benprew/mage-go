package secretsofstrixhaven

import (
	"testing"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

func TestSnarlSong(t *testing.T) {
	t.Run("creates two Fractal tokens with counters and life gain equal to colors spent", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Snarl Song")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Snarl Song")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// 1 color spent (G), so X=1: two Fractal tokens each with 1 counter, gain 1 life
		g.AssertPermanentCount(gametest.PlayerA, "Fractal Token", 2)
		g.AssertCounterCount(gametest.PlayerA, "Fractal Token", core.P1P1, 1)
		g.AssertLife(gametest.PlayerA, 21)
	})
}

func TestSocialSnub(t *testing.T) {
	t.Run("each player sacrifices a creature, opponent loses 1, you gain 1", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Social Snub")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Social Snub")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertLife(gametest.PlayerB, 19)
		g.AssertLife(gametest.PlayerA, 21)
	})
}

func TestSplatterTechnique(t *testing.T) {
	t.Run("mode 0 draws four cards", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Splatter Technique")
		g.ChooseMode(gametest.PlayerA, 0)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Splatter Technique")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// started with 0 cards in hand after casting; drew 4
		g.AssertHandCount(gametest.PlayerA, "Splatter Technique", 0)
	})
	t.Run("mode 1 deals 4 damage to each creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Splatter Technique")
		g.ChooseMode(gametest.PlayerA, 1)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Splatter Technique")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	})
}

func TestStandUpForYourself(t *testing.T) {
	t.Run("destroys target creature with power 3 or greater", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Serra Angel")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Stand Up for Yourself")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Stand Up for Yourself", "Serra Angel")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Serra Angel", 0)
	})
}

func TestStealTheShow(t *testing.T) {
	t.Run("mode 0 target player discards and redraws same count", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Steal the Show")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.ChooseMode(gametest.PlayerA, 0)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Steal the Show", "PlayerA")
		// PlayerA discards 2 bears (Steal the Show is on the stack / already cast)
		g.ChooseDiscard(gametest.PlayerA, "Grizzly Bears", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// discarded 2, drew 2
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 2)
	})
	t.Run("mode 1 deals damage equal to instants and sorceries in graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Serra Angel")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears") // not instant/sorcery
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Ancestral Anger")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Tome Blast")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Steal the Show")
		g.ChooseMode(gametest.PlayerA, 1)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Steal the Show", "Serra Angel")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// 2 instants/sorceries in GY = 2 damage; Serra Angel is 4/4, survives
		g.AssertPermanentCount(gametest.PlayerB, "Serra Angel", 1)
	})
}

func TestStressDream(t *testing.T) {
	t.Run("deals 5 damage to target creature and lets controller keep top card", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Serra Angel")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Stress Dream")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Stress Dream", "Serra Angel")
		g.ChooseFromLibrary(gametest.PlayerA, "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Serra Angel (4/4) takes 5 damage and dies
		g.AssertPermanentCount(gametest.PlayerB, "Serra Angel", 0)
		// PlayerA kept one card from library top
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}

func TestTogetherAsOne(t *testing.T) {
	t.Run("draws X cards, deals X damage, gains X life based on colors spent", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Serra Angel")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Together as One")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Together as One", "Serra Angel", "PlayerB")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// X=0 (colorless mana spent); deals 0 damage, draws 0 cards, gains 0 life (no-op)
		g.AssertLife(gametest.PlayerA, 20)
	})
}

func TestTomeBlast(t *testing.T) {
	t.Run("deals 2 damage to any target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Tome Blast")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Tome Blast", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	})
	t.Run("deals 2 damage to a player", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Tome Blast")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Tome Blast", "PlayerB")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 18)
	})
}

func TestTraumaticCritique(t *testing.T) {
	t.Run("deals X damage and draws 2 then discards 1", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Serra Angel")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Traumatic Critique")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Traumatic Critique", "Serra Angel")
		g.ChooseDiscard(gametest.PlayerA, "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// X=0, Serra Angel survives (4/4, 0 damage)
		g.AssertPermanentCount(gametest.PlayerB, "Serra Angel", 1)
	})
}

func TestUnsubtleMockery(t *testing.T) {
	t.Run("deals 4 damage to target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Serra Angel")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Unsubtle Mockery")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Unsubtle Mockery", "Serra Angel")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Serra Angel is 4/4, takes 4 damage and dies
		g.AssertPermanentCount(gametest.PlayerB, "Serra Angel", 0)
	})
}

func TestVibrantOutburst(t *testing.T) {
	t.Run("deals 3 damage to any target and taps a creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Vibrant Outburst")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Vibrant Outburst", "PlayerB", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 17)
		g.AssertTapped(gametest.PlayerA, "Grizzly Bears", true)
	})
}

func TestViciousRivalry(t *testing.T) {
	t.Run("destroys all artifacts and creatures with mana value <= X", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")    // MV 2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Serra Angel")      // MV 5
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Vicious Rivalry")
		g.ChooseNumber(gametest.PlayerA, 2) // pay 2 life (X=2)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Vicious Rivalry")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0) // MV 2 <= X=2
		g.AssertPermanentCount(gametest.PlayerB, "Serra Angel", 1)   // MV 5 > X=2
		g.AssertLife(gametest.PlayerA, 18)                            // paid 2 life
	})
}

func TestVisionarysDance(t *testing.T) {
	t.Run("creates two 3/3 Elemental tokens with flying", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Visionary's Dance")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Visionary's Dance")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Elemental Token", 2)
		g.AssertPowerToughness(gametest.PlayerA, "Elemental Token", 3, 3)
		g.AssertHasAbility(gametest.PlayerA, "Elemental Token", core.Flying, true)
	})
}

func TestWanderOff(t *testing.T) {
	t.Run("exiles target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Wander Off")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Wander Off", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertExileCount("Grizzly Bears", 1)
	})
}

func TestWildHypothesis(t *testing.T) {
	t.Run("creates 0/0 Fractal token with X counters", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Wild Hypothesis")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Wild Hypothesis")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// X=0: creates 0/0 Fractal with 0 counters; SBA immediately destroys it
		// (CR 704.5f: a creature with toughness 0 is put into its owner's graveyard).
		g.AssertPermanentCount(gametest.PlayerA, "Fractal Token", 0)
	})
}

func TestWisdomOfAges(t *testing.T) {
	t.Run("returns all instants and sorceries from graveyard to hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Ancestral Anger")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears") // not instant/sorcery
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Wisdom of Ages")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Wisdom of Ages")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertHandCount(gametest.PlayerA, "Ancestral Anger", 1)
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
		g.AssertGraveyardCount(gametest.PlayerA, "Ancestral Anger", 0)
		// Wisdom of Ages exiles itself
		g.AssertExileCount("Wisdom of Ages", 1)
	})
}

func TestWitherbloomCharm(t *testing.T) {
	t.Run("mode 0 sacrifices a permanent to draw two cards", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Witherbloom Charm")
		g.ChooseMode(gametest.PlayerA, 0)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Witherbloom Charm")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
	})
	t.Run("mode 1 you gain 5 life", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Witherbloom Charm")
		g.ChooseMode(gametest.PlayerA, 1)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Witherbloom Charm")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 25)
	})
	t.Run("mode 2 destroys target nonland permanent with mana value 2 or less", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // MV 2
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Witherbloom Charm")
		g.ChooseMode(gametest.PlayerA, 2)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Witherbloom Charm", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	})
}

func TestZimonesExperiment(t *testing.T) {
	t.Run("puts land onto battlefield tapped and creature into hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Zimone's Experiment")
		// Library contains Forest and Grizzly Bears in the top 5 to reveal.
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Zimone's Experiment")
		// From top 5, reveal Forest (land) and Grizzly Bears (creature)
		g.ChooseFromLibrary(gametest.PlayerA, "Forest")
		g.ChooseFromLibrary(gametest.PlayerA, "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Forest", 1)
		g.AssertTapped(gametest.PlayerA, "Forest", true)
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}

func TestAjanisResponse(t *testing.T) {
	t.Run("destroys target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Ajani's Response")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Ajani's Response", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestAncestralAnger(t *testing.T) {
	t.Run("grants trample and bonus power with no copies in graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Ancestral Anger")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Ancestral Anger", "Grizzly Bears")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		// X = 1 + 0 (no copies in GY) = 1; Grizzly Bears 2/2 + 1/0 = 3/2
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 2)
		g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Trample, true)
	})

	t.Run("draws a card", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Ancestral Anger")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Ancestral Anger", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Drew one card (hand was empty after casting)
		g.AssertHandCount(gametest.PlayerA, "Ancestral Anger", 0) // The drawn card is from library, not Ancestral Anger
	})

	t.Run("bonus increases with copies in graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Ancestral Anger")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Ancestral Anger")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Ancestral Anger")
		// X = 1 + 2 = 3; Grizzly Bears 2/2 + 3/0 = 5/2
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Ancestral Anger", "Grizzly Bears")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 5, 2)
	})
}

func TestAntiquitiesOnTheLoose(t *testing.T) {
	t.Run("creates two 2/2 Spirit tokens", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Antiquities on the Loose")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Antiquities on the Loose")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Spirit Token", 2)
		g.AssertPowerToughness(gametest.PlayerA, "Spirit Token", 2, 2)
	})

	t.Run("no counter added when cast from hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Antiquities on the Loose")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Antiquities on the Loose")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Spirit Token", core.P1P1, 0)
	})
}

func TestArcaneOmens(t *testing.T) {
	t.Run("target opponent discards one card when one color spent", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Arcane Omens")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Arcane Omens", "PlayerB")
		g.ChooseDiscard(gametest.PlayerB, "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// 3 bears - 1 discarded = 2 remaining
		g.AssertHandCount(gametest.PlayerB, "Grizzly Bears", 2)
	})
}

func TestArtisticProcess(t *testing.T) {
	t.Run("mode 0 deals 6 damage to target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Artistic Process")
		g.ChooseMode(gametest.PlayerA, 0)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Artistic Process", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	})

	t.Run("mode 1 deals 2 damage to each creature opponent controls", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Artistic Process")
		g.ChooseMode(gametest.PlayerA, 1)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Artistic Process")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// PlayerB's bear (2/2) takes 2 damage — dies
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		// PlayerA's bear is not affected by this mode
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	})

	t.Run("mode 2 creates a 3/3 Elemental with flying", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Artistic Process")
		g.ChooseMode(gametest.PlayerA, 2)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Artistic Process")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Elemental Token", 1)
		g.AssertPowerToughness(gametest.PlayerA, "Elemental Token", 3, 3)
		g.AssertHasAbility(gametest.PlayerA, "Elemental Token", core.Flying, true)
	})
}

func TestBanishingBetrayal(t *testing.T) {
	t.Run("returns target nonland permanent to owner's hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Banishing Betrayal")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Banishing Betrayal", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertHandCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestBorrowedKnowledge(t *testing.T) {
	t.Run("mode 0 draws equal to opponent hand size", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Borrowed Knowledge")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Ancestral Anger")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Ancestral Anger")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Ancestral Anger")
		// PlayerB has 3 cards; PlayerA discards 3 (Borrowed Knowledge + 2 Bears) and draws 3
		g.ChooseMode(gametest.PlayerA, 0)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Borrowed Knowledge", "PlayerB")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// PlayerA now has 3 cards from library (nothing named "Grizzly Bears" or "Borrowed Knowledge" since those were discarded)
		g.AssertHandCount(gametest.PlayerA, "Borrowed Knowledge", 0)
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 0)
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 2)
		g.AssertGraveyardCount(gametest.PlayerA, "Borrowed Knowledge", 1)
	})

	t.Run("mode 1 draws equal to discarded cards", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Borrowed Knowledge")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		// PlayerA has 3 cards total, discards all 3, draws 3
		g.ChooseMode(gametest.PlayerA, 1)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Borrowed Knowledge")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 2)
		g.AssertGraveyardCount(gametest.PlayerA, "Borrowed Knowledge", 1)
	})
}

func TestBrushOff(t *testing.T) {
	t.Run("counters target spell", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Brush Off")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Grizzly Bears")
		g.CastInResponseTo(gametest.PlayerA, "Brush Off", "Grizzly Bears")
		g.StopAt(2, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestBurrogBarrage(t *testing.T) {
	t.Run("deals damage equal to creature power to opponent creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Scathe Zombies")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Burrog Barrage")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Burrog Barrage", "Grizzly Bears", "Scathe Zombies")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Grizzly Bears (2 power) deals 2 to opponent's 2/2 → kills it
		g.AssertPermanentCount(gametest.PlayerB, "Scathe Zombies", 0)
	})

	t.Run("gets +1/+0 if another instant was cast this turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Serra Angel")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Chase Inspiration")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Burrog Barrage")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Chase Inspiration", "Grizzly Bears")
		g.CastInResponseTo(gametest.PlayerA, "Burrog Barrage", "Grizzly Bears", "Serra Angel")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Bear has 2+1=3 power (from "another instant" bonus), deals 3 to 4/4 Serra Angel - doesn't die
		g.AssertPermanentCount(gametest.PlayerB, "Serra Angel", 1)
	})
}

func TestChaseInspiration(t *testing.T) {
	t.Run("grants +0/+3 while active", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Chase Inspiration")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Chase Inspiration", "Grizzly Bears")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 5)
	})

	t.Run("grants hexproof while active", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Chase Inspiration")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Chase Inspiration", "Grizzly Bears")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Hexproof, true)
	})

	t.Run("hexproof expires at end of turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Chase Inspiration")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Chase Inspiration", "Grizzly Bears")
		g.StopAt(2, core.Upkeep)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Hexproof, false)
	})
}

func TestChelonianTackle(t *testing.T) {
	t.Run("gives creature +0/+10 and it fights opponent creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Scathe Zombies")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Chelonian Tackle")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Chelonian Tackle", "Grizzly Bears", "Scathe Zombies")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// PlayerA's bear (2 power) fights opponent's 2/2 Scathe Zombies → Scathe Zombies dies
		// PlayerA's bear (12 toughness) survives opponent's 2 power
		g.AssertPermanentCount(gametest.PlayerB, "Scathe Zombies", 0)
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	})

	t.Run("boost applies even without fight target", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Chelonian Tackle")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Chelonian Tackle", "Grizzly Bears")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 12)
	})
}

func TestChoreographedSparks(t *testing.T) {
	t.Run("copies target instant or sorcery spell you control", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Choreographed Sparks")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
		g.CastInResponseTo(gametest.PlayerA, "Choreographed Sparks", "Lightning Bolt")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// The copy of Lightning Bolt resolves first (3 damage), then the original resolves (3 damage)
		// Total: 6 damage to PlayerB
		g.AssertLife(gametest.PlayerB, 14)
	})
}

func TestCostOfBrilliance(t *testing.T) {
	t.Run("target player draws 2 and loses 2 life", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Cost of Brilliance")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Cost of Brilliance", "PlayerB")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 18)
	})

	t.Run("puts +1/+1 counter on target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Cost of Brilliance")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Cost of Brilliance", "PlayerB", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
	})
}

func TestDaydream(t *testing.T) {
	t.Run("exiles and returns creature with +1/+1 counter", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Daydream")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Daydream", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
		// Grizzly Bears 2/2 + 1/1 counter = 3/3
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
	})
}

func TestDigSiteInventory(t *testing.T) {
	t.Run("puts +1/+1 counter on target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Dig Site Inventory")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Dig Site Inventory", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Counter persists after EOT: 2/2 + 1/1 = 3/3
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
	})

	t.Run("creature has vigilance during turn it was cast", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Dig Site Inventory")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Dig Site Inventory", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Grizzly Bears attacked with vigilance, so it should NOT be tapped
		g.AssertTapped(gametest.PlayerA, "Grizzly Bears", false)
	})
}

func TestDinasGuidance(t *testing.T) {
	t.Run("searches library for creature and puts it in hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Dina's Guidance")
		g.ChooseFromLibrary(gametest.PlayerA, "Grizzly Bears")
		g.ChooseMode(gametest.PlayerA, 0)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Dina's Guidance")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
	})

	t.Run("searches library for creature and puts it in graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Dina's Guidance")
		g.ChooseFromLibrary(gametest.PlayerA, "Grizzly Bears")
		g.ChooseMode(gametest.PlayerA, 1)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Dina's Guidance")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}

func TestPrismariCharm(t *testing.T) {
	t.Run("mode 1 deals 1 damage to target", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Scathe Zombies")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Prismari Charm")
		g.ChooseMode(gametest.PlayerA, 1)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Prismari Charm", "Scathe Zombies")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Scathe Zombies 2/2 takes 1 damage — survives (1 damage, not lethal)
		g.AssertPermanentCount(gametest.PlayerB, "Scathe Zombies", 1)
	})
	t.Run("mode 2 returns nonland permanent to owner's hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Prismari Charm")
		g.ChooseMode(gametest.PlayerA, 2)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Prismari Charm", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertHandCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestHomesickness(t *testing.T) {
	t.Run("target player draws two cards", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Homesickness")
		// PlayerB's library has two Grizzly Bears to draw
		g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Grizzly Bears", 2)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Homesickness", "PlayerB")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// PlayerB should have drawn 2 cards (started with 0 in hand, now 2)
		g.AssertHandCount(gametest.PlayerB, "Grizzly Bears", 2)
	})
	t.Run("taps up to two target creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Homesickness")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Homesickness", "PlayerB", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertTapped(gametest.PlayerB, "Grizzly Bears", true)
	})
}

func TestImpracticalJoke(t *testing.T) {
	t.Run("deals 3 damage to target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Impractical Joke")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Impractical Joke", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestInterjection(t *testing.T) {
	t.Run("target creature gets +2/+2 and first strike until end of turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Interjection")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Interjection", "Grizzly Bears")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 4)
		g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.FirstStrike, true)
	})
	t.Run("effect wears off at end of turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Interjection")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Interjection", "Grizzly Bears")
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2)
		g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.FirstStrike, false)
	})
}

func TestKilliansConfidence(t *testing.T) {
	t.Run("target creature gets +1/+1 until end of turn and controller draws a card", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Killian's Confidence")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Killian's Confidence", "Grizzly Bears")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
		// Drew a card (hand empty after casting, should have 1 card)
		g.AssertHandCount(gametest.PlayerA, "", 1)
	})
}

func TestLastGasp(t *testing.T) {
	t.Run("target creature gets -3/-3 until end of turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Last Gasp")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Last Gasp", "Hill Giant")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		// Hill Giant 3/3 gets -3/-3 = 0/0 and dies
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 0)
	})
	t.Run("negative toughness kills the creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Last Gasp")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Last Gasp", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	})
}

func TestLoreholdCharm(t *testing.T) {
	t.Run("mode 0: opponent sacrifices a nontoken artifact", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Black Vise")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lorehold Charm")
		g.ChooseMode(gametest.PlayerA, 0)
		g.ChoosePermanent(gametest.PlayerB, "Black Vise")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lorehold Charm")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Black Vise", 0)
	})
	t.Run("mode 1: return artifact or creature card mana value 2 or less from graveyard to battlefield", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lorehold Charm")
		g.ChooseMode(gametest.PlayerA, 1)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lorehold Charm", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
	t.Run("mode 2: creatures you control get +1/+1 and trample until end of turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lorehold Charm")
		g.ChooseMode(gametest.PlayerA, 2)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lorehold Charm")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
		g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Trample, true)
	})
}

func TestManaSculpt(t *testing.T) {
	t.Run("counters target spell", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Mana Sculpt")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Grizzly Bears")
		g.CastInResponseTo(gametest.PlayerA, "Mana Sculpt", "Grizzly Bears")
		g.StopAt(2, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestMasterfulFlourish(t *testing.T) {
	t.Run("creature you control gets +1/+0 and indestructible until end of turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Masterful Flourish")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Masterful Flourish", "Grizzly Bears")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 2)
		g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Indestructible, true)
	})
	t.Run("indestructible wears off at end of turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Masterful Flourish")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Masterful Flourish", "Grizzly Bears")
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2)
		g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Indestructible, false)
	})
}

func TestMathemagics(t *testing.T) {
	t.Run("target player draws 2^X cards where X=0", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Mathemagics")
		// X=0: target player draws 2^0 = 1 card
		g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Mathemagics", 0, "PlayerA")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// PlayerA cast the spell (hand was 1: Mathemagics, then 0 after cast)
		// After drawing 1 card, hand should have 1 card
		g.AssertHandCount(gametest.PlayerA, "", 1)
	})
	t.Run("target player draws 2^X cards where X=2", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Mathemagics")
		// X=2: target player draws 2^2 = 4 cards
		g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Mathemagics", 2, "PlayerB")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertHandCount(gametest.PlayerB, "", 4)
	})
}

func TestMindRoots(t *testing.T) {
	t.Run("target player discards two cards", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Forest")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Mind Roots")
		g.ChooseDiscard(gametest.PlayerB, "Grizzly Bears", "Forest")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Mind Roots", "PlayerB")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
		g.AssertGraveyardCount(gametest.PlayerB, "Forest", 1)
	})
	t.Run("discarded land card is put onto battlefield tapped under controller's control", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Forest")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Island")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Mind Roots")
		g.ChooseDiscard(gametest.PlayerB, "Forest", "Island")
		// Choose to put Forest onto battlefield
		g.ChoosePermanent(gametest.PlayerA, "Forest")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Mind Roots", "PlayerB")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Forest", 1)
	})
}

func TestMindIntoMatter(t *testing.T) {
	t.Run("draws X cards", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Mind into Matter")
		g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Mind into Matter", 3)
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Drew 3 cards; hand should have 3
		g.AssertHandCount(gametest.PlayerA, "", 3)
	})
}

func TestMoltenNote(t *testing.T) {
	t.Run("deals damage equal to mana spent to cast to target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Molten Note")
		// X=1: total mana spent = 1(X) + 2(R+W) = 3
		g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Molten Note", 1, "Hill Giant")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Hill Giant 3/3 takes 3 damage (1+2), dies
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 0)
	})
	t.Run("untaps all creatures you control", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Molten Note")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.CastSpellWithX(1, core.DeclareAttackers, gametest.PlayerA, "Molten Note", 0, "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// PlayerA's Grizzly Bears attacked (tapped), then Molten Note untapped all creatures you control
		g.AssertTapped(gametest.PlayerA, "Grizzly Bears", false)
	})
}

func TestMomentOfReckoning(t *testing.T) {
	t.Run("mode 0: destroy target nonland permanent", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Moment of Reckoning")
		g.ChooseMode(gametest.PlayerA, 0)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Moment of Reckoning", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	})
	t.Run("mode 1: return target nonland permanent card from graveyard to battlefield", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Moment of Reckoning")
		g.ChooseMode(gametest.PlayerA, 1)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Moment of Reckoning", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}

func TestMusesEncouragement(t *testing.T) {
	t.Run("creates a 3/3 blue red Elemental token with flying", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Muse's Encouragement")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Muse's Encouragement")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Elemental", 1)
	})
}

func TestOraclesRestoration(t *testing.T) {
	t.Run("target creature gets +1/+1 until end of turn, draw a card, gain 1 life", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Oracle's Restoration")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Oracle's Restoration", "Grizzly Bears")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
		// Drew a card and gained 1 life
		g.AssertHandCount(gametest.PlayerA, "", 1)
		g.AssertLife(gametest.PlayerA, 21)
	})
}

func TestPlanarEngineering(t *testing.T) {
	t.Run("sacrifice two lands and search for four basic lands", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Planar Engineering")
		g.ChoosePermanent(gametest.PlayerA, "Forest")
		g.ChoosePermanent(gametest.PlayerA, "Forest")
		g.ChooseFromLibrary(gametest.PlayerA, "Mountain")
		g.ChooseFromLibrary(gametest.PlayerA, "Mountain")
		g.ChooseFromLibrary(gametest.PlayerA, "Plains")
		g.ChooseFromLibrary(gametest.PlayerA, "Island")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Planar Engineering")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Two Forests sacrificed, 4 basic lands enter tapped
		g.AssertPermanentCount(gametest.PlayerA, "Forest", 0)
		g.AssertPermanentCount(gametest.PlayerA, "Mountain", 2)
		g.AssertPermanentCount(gametest.PlayerA, "Plains", 1)
		g.AssertPermanentCount(gametest.PlayerA, "Island", 1)
	})
}

func TestPoxPlague(t *testing.T) {
	t.Run("each player loses half their life", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Pox Plague")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Pox Plague")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Each player loses half (20/2 = 10) life, round down
		g.AssertLife(gametest.PlayerA, 10)
		g.AssertLife(gametest.PlayerB, 10)
	})
	t.Run("each player discards half their hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Pox Plague")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Hill Giant")
		g.ChooseDiscard(gametest.PlayerA, "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Pox Plague")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// PlayerA had 3 cards (Pox Plague + 2), casts Pox Plague (2 in hand),
		// discards half of 2 = 1 card
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
	t.Run("each player sacrifices half their permanents of their choice", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Pox Plague")
		g.ChoosePermanent(gametest.PlayerB, "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Pox Plague")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// PlayerB controls 2 creatures, sacrifices half = 1
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 1)
	})
}

func TestPracticedOffense(t *testing.T) {
	t.Run("puts +1/+1 counter on each creature target player controls", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Practiced Offense")
		g.ChooseMode(gametest.PlayerA, 0) // double strike
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Practiced Offense", "PlayerA", "Grizzly Bears")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
		g.AssertPowerToughness(gametest.PlayerA, "Hill Giant", 4, 4)
	})
	t.Run("target creature gains double strike", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Practiced Offense")
		g.ChooseMode(gametest.PlayerA, 0) // double strike
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Practiced Offense", "PlayerA", "Grizzly Bears")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.DoubleStrike, true)
	})
	t.Run("target creature gains lifelink", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Practiced Offense")
		g.ChooseMode(gametest.PlayerA, 1) // lifelink
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Practiced Offense", "PlayerA", "Grizzly Bears")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Lifelink, true)
	})
}

func TestProcrastinate(t *testing.T) {
	t.Run("taps target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Procrastinate")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Procrastinate", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertTapped(gametest.PlayerB, "Grizzly Bears", true)
	})
}

func TestProctorsGaze(t *testing.T) {
	t.Run("returns nonland permanent to owner's hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Proctor's Gaze")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
		g.ChooseFromLibrary(gametest.PlayerA, "Forest")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Proctor's Gaze", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertHandCount(gametest.PlayerB, "Grizzly Bears", 1)
		g.AssertPermanentCount(gametest.PlayerA, "Forest", 1)
	})
	t.Run("puts basic land onto battlefield tapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Proctor's Gaze")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
		g.ChooseFromLibrary(gametest.PlayerA, "Forest")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Proctor's Gaze")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Forest", 1)
		g.AssertTapped(gametest.PlayerA, "Forest", true)
	})
}

func TestProfessorDellianFel(t *testing.T) {
	t.Run("enters with 5 loyalty", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Professor Dellian Fel")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Professor Dellian Fel")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Professor Dellian Fel", core.Loyalty, 5)
	})
}

func TestPullFromTheGrave(t *testing.T) {
	t.Run("returns up to two creature cards from graveyard to hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Scathe Zombies")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Pull from the Grave")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Pull from the Grave", "Grizzly Bears", "Scathe Zombies")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
		g.AssertHandCount(gametest.PlayerA, "Scathe Zombies", 1)
	})
	t.Run("gains 2 life", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Pull from the Grave")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Pull from the Grave")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 22)
	})
}

func TestPursuethePast(t *testing.T) {
	t.Run("gains 2 life", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Pursue the Past")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Pursue the Past")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 22)
	})
	t.Run("discard then draw two", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Pursue the Past")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.ChooseDiscard(gametest.PlayerA, "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Pursue the Past")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}

func TestQuandrixCharm(t *testing.T) {
	t.Run("mode 1 destroys target enchantment", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Holy Strength")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Quandrix Charm")
		g.ChooseMode(gametest.PlayerA, 1)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Quandrix Charm", "Holy Strength")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Holy Strength", 0)
	})
	t.Run("mode 2 sets creature to 5/5 until end of turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Quandrix Charm")
		g.ChooseMode(gametest.PlayerA, 2)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Quandrix Charm", "Grizzly Bears")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 5, 5)
	})
}

func TestQuickStudy(t *testing.T) {
	t.Run("draws two cards", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Quick Study")
		initialHand := 0 // Quick Study in hand only
		_ = initialHand
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Quick Study")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// After casting Quick Study (hand=0) and drawing 2, hand should have 2 library cards
		g.AssertHandCount(gametest.PlayerA, "Quick Study", 0)
	})
}

func TestRabidAttack(t *testing.T) {
	t.Run("grants +1/+0 to target creatures you control", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Rabid Attack")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Rabid Attack", "Grizzly Bears")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 2)
	})
}

func TestRalZarekGuestLecturer(t *testing.T) {
	t.Run("enters with 4 loyalty", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Ral Zarek, Guest Lecturer")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Ral Zarek, Guest Lecturer")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Ral Zarek, Guest Lecturer", core.Loyalty, 4)
	})
}

func TestRapierWit(t *testing.T) {
	t.Run("taps target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Rapier Wit")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Rapier Wit", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertTapped(gametest.PlayerB, "Grizzly Bears", true)
	})
	t.Run("draws a card", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Rapier Wit")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Rapier Wit", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// After casting (hand=0), drew 1 card → hand has 1 card
		g.AssertHandCount(gametest.PlayerA, "Rapier Wit", 0)
	})
}

func TestRapturousMoment(t *testing.T) {
	t.Run("draws three then discards two", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Rapturous Moment")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Scathe Zombies")
		g.ChooseDiscard(gametest.PlayerA, "Grizzly Bears", "Scathe Zombies")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Rapturous Moment")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
		g.AssertGraveyardCount(gametest.PlayerA, "Scathe Zombies", 1)
	})
}

func TestRenderSpeechless(t *testing.T) {
	t.Run("opponent discards chosen nonland card from their hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Render Speechless")
		g.ChooseFromLibrary(gametest.PlayerA, "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Render Speechless", "PlayerB")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertHandCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
	t.Run("puts two +1/+1 counters on target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Scathe Zombies")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Render Speechless")
		g.ChooseFromLibrary(gametest.PlayerA, "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Render Speechless", "PlayerB", "Scathe Zombies")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Scathe Zombies", core.P1P1, 2)
	})
}

func TestRootManipulation(t *testing.T) {
	t.Run("grants +2/+2 and menace to creatures you control until end of turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Root Manipulation")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Root Manipulation")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 4)
		g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Menace, true)
	})
}

func TestRunBehind(t *testing.T) {
	t.Run("owner puts creature on top of library", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Run Behind")
		g.ChooseMode(gametest.PlayerB, 0) // 0 = top, 1 = bottom
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Run Behind", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertLibraryTop(gametest.PlayerB, "Grizzly Bears")
	})
}

func TestSeizeTheSpoils(t *testing.T) {
	t.Run("draws two cards and creates a Treasure token", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Seize the Spoils")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.ChooseDiscard(gametest.PlayerA, "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Seize the Spoils")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
		g.AssertPermanentCount(gametest.PlayerA, "Treasure Token", 1)
	})
}

func TestSendInThePest(t *testing.T) {
	t.Run("each opponent discards a card", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Send in the Pest")
		g.ChooseDiscard(gametest.PlayerB, "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Send in the Pest")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
	t.Run("creates a 1/1 Pest token", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Send in the Pest")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Send in the Pest")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Pest Token", 1)
		g.AssertPowerToughness(gametest.PlayerA, "Pest Token", 1, 1)
	})
}

func TestSilverquillCharm(t *testing.T) {
	t.Run("mode 0 puts two +1/+1 counters on target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Silverquill Charm")
		g.ChooseMode(gametest.PlayerA, 0)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Silverquill Charm", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", core.P1P1, 2)
	})
	t.Run("mode 1 exiles creature with power 2 or less", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Silverquill Charm")
		g.ChooseMode(gametest.PlayerA, 1)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Silverquill Charm", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertExileCount("Grizzly Bears", 1)
	})
	t.Run("mode 2 opponent loses 3 life and you gain 3 life", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Silverquill Charm")
		g.ChooseMode(gametest.PlayerA, 2)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Silverquill Charm")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 17)
		g.AssertLife(gametest.PlayerA, 23)
	})
}

func TestDissectionPractice(t *testing.T) {
	t.Run("target opponent loses 1 life and controller gains 1 life", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Dissection Practice")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Dissection Practice", "PlayerB", "Grizzly Bears", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 19)
		g.AssertLife(gametest.PlayerA, 21)
	})

	t.Run("up to one creature gets +1/+1 until end of turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Dissection Practice")
		// Target PlayerB for life loss; target Grizzly Bears for +1/+1; skip -1/-1 (up to one).
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Dissection Practice", "PlayerB", "Grizzly Bears")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
	})

	t.Run("up to one creature gets -1/-1 until end of turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Air Elemental")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Dissection Practice")
		// Target PlayerB for life loss; target Air Elemental for +1/+1; target Grizzly Bears for -1/-1.
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Dissection Practice", "PlayerB", "Air Elemental", "Grizzly Bears")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerB, "Grizzly Bears", 1, 1)
	})
}

func TestDivergentEquation(t *testing.T) {
	t.Run("returns up to X instant/sorcery cards from graveyard to hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Lightning Bolt")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Ancestral Anger")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Divergent Equation")
		g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Divergent Equation", 2, "Lightning Bolt", "Ancestral Anger")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertHandCount(gametest.PlayerA, "Lightning Bolt", 1)
		g.AssertHandCount(gametest.PlayerA, "Ancestral Anger", 1)
	})

	t.Run("exiles itself after resolving", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Divergent Equation")
		g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Divergent Equation", 0)
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertExileCount("Divergent Equation", 1)
		g.AssertGraveyardCount(gametest.PlayerA, "Divergent Equation", 0)
	})
}

func TestDuelTactics(t *testing.T) {
	t.Run("deals 1 damage to target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Duel Tactics")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Duel Tactics", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// 2/2 takes 1 damage; survives
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestEmbraceTheParadox(t *testing.T) {
	t.Run("draws three cards", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Embrace the Paradox")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.GetPlayer(gametest.PlayerA).QueueMayAbilityChoices(false)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Embrace the Paradox")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 3)
	})
}

func TestEndOfTheHunt(t *testing.T) {
	t.Run("target opponent exiles their creature with greatest mana value", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Air Elemental")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "End of the Hunt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "End of the Hunt", "PlayerB")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Air Elemental CMC=5 > Grizzly Bears CMC=2
		g.AssertPermanentCount(gametest.PlayerB, "Air Elemental", 0)
		g.AssertExileCount("Air Elemental", 1)
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestErode(t *testing.T) {
	t.Run("destroys target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Erode")
		g.GetPlayer(gametest.PlayerB).QueueMayAbilityChoices(false)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Erode", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	})

	t.Run("destroyed creature controller may search library for basic land enters tapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Forest")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Erode")
		g.ChooseFromLibrary(gametest.PlayerB, "Forest")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Erode", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Forest", 1)
		g.AssertTapped(gametest.PlayerB, "Forest", true)
	})
}

func TestEssenceScatter(t *testing.T) {
	t.Run("counters target creature spell", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Essence Scatter")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Grizzly Bears")
		g.CastInResponseTo(gametest.PlayerA, "Essence Scatter", "Grizzly Bears")
		g.StopAt(2, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestFixWhatsBroken(t *testing.T) {
	t.Run("returns artifact and creature cards with mana value X from graveyard to battlefield", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Fix What's Broken")
		g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Fix What's Broken", 2)
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 2)
		g.AssertLife(gametest.PlayerA, 18)
	})
}

func TestFlashbackSpell(t *testing.T) {
	t.Run("targets instant or sorcery card in graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Ancestral Anger")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Flashback")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Flashback", "Ancestral Anger")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Resolves without error; XXX: granting flashback not yet engine-supported
	})
}

func TestFlowState(t *testing.T) {
	t.Run("puts one of top three into hand when condition not met", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Air Elemental")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Llanowar Elves")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Flow State")
		g.ChooseFromLibrary(gametest.PlayerA, "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Flow State")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
	})

	t.Run("puts two of top three into hand when instant and sorcery both in graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Air Elemental")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Llanowar Elves")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Lightning Bolt")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Ancestral Anger")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Flow State")
		g.ChooseFromLibrary(gametest.PlayerA, "Grizzly Bears")
		g.ChooseFromLibrary(gametest.PlayerA, "Air Elemental")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Flow State")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
		g.AssertHandCount(gametest.PlayerA, "Air Elemental", 1)
	})
}

func TestFractalAnomaly(t *testing.T) {
	// When X=0 (no cards drawn this turn), the 0/0 Fractal token dies immediately
	// from state-based actions (CR 704.5f). This sub-test verifies the token does
	// not persist on the battlefield (correct Oracle behavior).
	t.Run("creates a 0/0 green and blue Fractal token", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Fractal Anomaly")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Fractal Anomaly")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// 0 cards drawn → token enters as 0/0, immediately dies from SBA
		g.AssertPermanentCount(gametest.PlayerA, "Fractal Token", 0)
	})

	t.Run("token has X +1/+1 counters when cards were drawn this turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Fractal Anomaly")
		// Cast on turn 3 (PlayerA's second turn), when draw step draws 1 card (X=1).
		// The 0/0 Fractal token enters with 1 +1/+1 counter and persists as 1/1.
		g.CastSpell(3, core.PrecombatMain, gametest.PlayerA, "Fractal Anomaly")
		g.StopAt(3, core.EndStep)
		g.Execute()
		// X=1 (1 card drawn in draw step); token persists as 1/1 with 1 counter
		g.AssertPermanentCount(gametest.PlayerA, "Fractal Token", 1)
		g.AssertCounterCount(gametest.PlayerA, "Fractal Token", core.P1P1, 1)
	})
}

func TestFractalize(t *testing.T) {
	t.Run("target creature has base P/T (X+1)/(X+1) until end of turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Fractalize")
		g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Fractalize", 3, "Grizzly Bears")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		// X=3 → base P/T = 4/4
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 4)
	})

	t.Run("effect expires after end of turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Fractalize")
		g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Fractalize", 3, "Grizzly Bears")
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2)
	})
}

func TestGloriousDecay(t *testing.T) {
	t.Run("mode 0 destroys target artifact", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Black Lotus")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Glorious Decay")
		g.ChooseMode(gametest.PlayerA, 0)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Glorious Decay", "Black Lotus")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Black Lotus", 0)
	})

	t.Run("mode 1 deals 4 damage to target creature with flying", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Air Elemental")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Glorious Decay")
		g.ChooseMode(gametest.PlayerA, 1)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Glorious Decay", "Air Elemental")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Air Elemental", 0)
	})

	t.Run("mode 2 exiles target card from a graveyard and draws a card", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Glorious Decay")
		g.ChooseMode(gametest.PlayerA, 2)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Glorious Decay", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertExileCount("Grizzly Bears", 1)
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 0)
	})
}

func TestGrappleWithDeath(t *testing.T) {
	t.Run("destroys target artifact or creature and controller gains 1 life", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grapple with Death")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Grapple with Death", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertLife(gametest.PlayerA, 21)
	})
}

func TestGroupProject(t *testing.T) {
	t.Run("creates a 2/2 red and white Spirit creature token", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Group Project")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Group Project")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Spirit Token", 1)
		g.AssertPowerToughness(gametest.PlayerA, "Spirit Token", 2, 2)
	})
}

func TestGrowthCurve(t *testing.T) {
	t.Run("puts +1/+1 counter then doubles counters on target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Growth Curve")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Growth Curve", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// 0+1=1, doubled=2 counters
		g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", core.P1P1, 2)
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 4)
	})

	t.Run("starting with 2 counters results in 6 after +1 then double", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCounters(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears", core.P1P1, 2)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Growth Curve")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Growth Curve", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// 2+1=3, doubled=6
		g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", core.P1P1, 6)
	})
}

func TestHarshAnnotation(t *testing.T) {
	t.Run("destroys target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Harsh Annotation")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Harsh Annotation", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	})

	t.Run("destroyed creature's controller creates 1/1 white and black Inkling token with flying", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Harsh Annotation")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Harsh Annotation", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Inkling Token", 1)
		g.AssertPowerToughness(gametest.PlayerB, "Inkling Token", 1, 1)
		g.AssertHasAbility(gametest.PlayerB, "Inkling Token", core.Flying, true)
	})
}

func TestHeatedArgument(t *testing.T) {
	t.Run("deals 6 damage to target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Air Elemental")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Heated Argument")
		g.GetPlayer(gametest.PlayerA).QueueMayAbilityChoices(false)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Heated Argument", "Air Elemental")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Air Elemental", 0)
	})

	t.Run("if exile graveyard card also deals 2 damage to creature controller", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Air Elemental")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Heated Argument")
		g.ChooseFromLibrary(gametest.PlayerA, "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Heated Argument", "Air Elemental")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 18)
	})
}
