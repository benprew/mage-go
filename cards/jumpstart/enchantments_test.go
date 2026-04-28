package jumpstart

import (
	"testing"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

func TestPacifism(t *testing.T) {
	t.Run("enchanted creature can't attack", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Pacifism")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Pacifism", "Hill Giant")
		g.Attack(3, gametest.PlayerA, "Hill Giant")
		g.StopAt(3, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 20)
	})
}

func TestSkyTether(t *testing.T) {
	t.Run("removes flying and grants defender", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Air Elemental")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Sky Tether")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Sky Tether", "Air Elemental")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Air Elemental", core.Flying, false)
		g.AssertHasAbility(gametest.PlayerA, "Air Elemental", core.Defender, true)
	})
}

func TestMarkOfTheVampire(t *testing.T) {
	t.Run("boosts +2/+2 and grants lifelink", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Mark of the Vampire")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Mark of the Vampire", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 4)
		g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Lifelink, true)
	})
}

func TestPhyrexianReclamation(t *testing.T) {
	t.Run("returns target creature card from graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Phyrexian Reclamation")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Phyrexian Reclamation", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
		g.AssertLife(gametest.PlayerA, 18)
	})
}

func TestZombieInfestation(t *testing.T) {
	t.Run("discard two cards to make a zombie token", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Zombie Infestation")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears", 2)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Zombie Infestation")
		g.ChooseDiscard(gametest.PlayerA, "Grizzly Bears", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Zombie", 1)
	})
}

func TestBarrageOfExpendables(t *testing.T) {
	t.Run("sacrifice creature to deal 1 damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Barrage of Expendables")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Barrage of Expendables", "PlayerB")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 19)
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
	})
}

func TestMakeshiftMunitions(t *testing.T) {
	t.Run("sacrifice artifact to deal 1 damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Makeshift Munitions")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Makeshift Munitions", "PlayerB")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 19)
	})
}

func TestForcedWorship(t *testing.T) {
	t.Run("enchanted creature can't attack", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Forced Worship")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Forced Worship", "Hill Giant")
		g.Attack(3, gametest.PlayerA, "Hill Giant")
		g.StopAt(3, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 20)
	})

	t.Run("activated ability returns aura to hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Forced Worship")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Forced Worship", "Hill Giant")
		g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Forced Worship")
		g.StopAt(3, core.EndStep)
		g.Execute()
		g.AssertHandCount(gametest.PlayerA, "Forced Worship", 1)
	})
}

func TestCathersCrusade(t *testing.T) {
	t.Run("creature ETB places +1/+1 on each creature you control", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Cathars' Crusade")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Hill Giant")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Hill Giant")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", core.P1P1, 2)
		g.AssertCounterCount(gametest.PlayerA, "Hill Giant", core.P1P1, 1)
	})
}

func TestKnightlyValor(t *testing.T) {
	t.Run("creates a Knight token and boosts enchanted creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Knightly Valor")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Knightly Valor", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 4)
		g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Vigilance, true)
		g.AssertPermanentCount(gametest.PlayerA, "Knight", 1)
	})
}

func TestNarcolepsy(t *testing.T) {
	t.Run("taps enchanted creature each upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Narcolepsy")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Narcolepsy", "Grizzly Bears")
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Grizzly Bears", true)
	})
}

func TestWaterknot(t *testing.T) {
	t.Run("taps enchanted creature on ETB and prevents untap", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Waterknot")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Waterknot", "Grizzly Bears")
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Grizzly Bears", true)
	})
}

func TestStabWound(t *testing.T) {
	t.Run("boosts -2/-2 and damages controller each upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Stab Wound")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Stab Wound", "Hill Giant")
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 18)
	})
}

func TestDeathsApproach(t *testing.T) {
	t.Run("reduces P/T by creature cards in graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerB, "Grizzly Bears", 2)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Death's Approach")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Death's Approach", "Hill Giant")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerB, "Hill Giant", 1, 1)
	})
}

func TestZendikarsRoil(t *testing.T) {
	t.Run("creates Elemental token when land enters", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Zendikar's Roil")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Forest")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Elemental", 1)
	})
}

func TestPresenceOfGond(t *testing.T) {
	t.Run("grants tap activated ability to create Elf Warrior token", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Presence of Gond")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Presence of Gond", "Grizzly Bears")
		g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(3, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Elf Warrior", 1)
	})
}

func TestFeralInvocation(t *testing.T) {
	t.Run("boosts +2/+2", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Feral Invocation")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Feral Invocation", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 4)
	})

	t.Run("flash: castable on opponent's turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Feral Invocation")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest", 3)
		g.CastSpell(1, core.BeginCombat, gametest.PlayerB, "Feral Invocation", "Grizzly Bears")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertAttachedTo(gametest.PlayerA, "Feral Invocation", "Grizzly Bears")
	})
}

func TestIndomitableWill(t *testing.T) {
	t.Run("boosts +1/+2", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Indomitable Will")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Indomitable Will", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 4)
	})

	t.Run("flash: castable on opponent's turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Indomitable Will")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains", 2)
		g.CastSpell(1, core.BeginCombat, gametest.PlayerB, "Indomitable Will", "Grizzly Bears")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertAttachedTo(gametest.PlayerA, "Indomitable Will", "Grizzly Bears")
	})
}

func TestLawmagesBinding(t *testing.T) {
	t.Run("enchanted creature can't attack", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lawmage's Binding")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 1)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lawmage's Binding", "Hill Giant")
		g.Attack(2, gametest.PlayerB, "Hill Giant")
		g.StopAt(2, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 20)
	})

	t.Run("flash: castable on opponent's turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lawmage's Binding")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains", 2)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island", 1)
		g.CastSpell(1, core.BeginCombat, gametest.PlayerB, "Lawmage's Binding", "Hill Giant")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertAttachedTo(gametest.PlayerA, "Lawmage's Binding", "Hill Giant")
	})
}

func TestExquisiteBlood_GainsOnOpponentLifeLoss(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Exquisite Blood")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 17)
	g.AssertLife(gametest.PlayerA, 23)
}

func TestExquisiteBlood_NoGainOnOwnLifeLoss(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Exquisite Blood")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerA")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 17)
}

func TestBranchingEvolution_DoublesP1P1Counters(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Branching Evolution")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bloodbond Vampire")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Healing Salve")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Healing Salve", "PlayerA")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Bloodbond Vampire would normally get 1 +1/+1 counter on life gain.
	// Branching Evolution doubles to 2.
	g.AssertCounterCount(gametest.PlayerA, "Bloodbond Vampire", core.P1P1, 2)
}

func TestBranchingEvolution_DoesNotDoubleOpponentCounters(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Branching Evolution")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Bloodbond Vampire")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Healing Salve")
	g.CastSpell(1, core.PostcombatMain, gametest.PlayerB, "Healing Salve", "PlayerB")
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerB, "Bloodbond Vampire", core.P1P1, 1)
}

func TestCradleOfVitality_PutCountersWhenPaying(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Cradle of Vitality")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	// Plains for Healing Salve {W}; two more for the may-pay {1}{W}.
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Healing Salve")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Healing Salve", "PlayerA")
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 23)
	// Healing Salve gains 3 life, so 3 +1/+1 counters on the bears.
	g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", core.P1P1, 3)
}

func TestCradleOfVitality_DeclinePayment(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Cradle of Vitality")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Healing Salve")
	tpA := g.GetPlayer(gametest.PlayerA)
	tpA.QueueMayAbilityChoices(false)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Healing Salve", "PlayerA")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 23)
	g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", core.P1P1, 0)
}
