package jumpstart

import (
	"testing"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
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

func TestCuriousObsession_BoostsAndDrawsOnHit(t *testing.T) {
	g := gametest.NewTestGame(t)
	bearID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	auraID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Curious Obsession")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain", 5)
	g.Attach(auraID, bearID)
	g.Attack(1, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
	g.AssertLife(gametest.PlayerB, 17)
}

func TestCuriousObsession_SacrificedIfDidNotAttack(t *testing.T) {
	g := gametest.NewTestGame(t)
	bearID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	auraID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Curious Obsession")
	g.Attach(auraID, bearID)
	g.StopAt(1, core.Cleanup)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Curious Obsession", 1)
}

// Duelist's Heritage: once per combat (regardless of attacker count), may have
// target attacking creature gain double strike until end of turn. Verified by
// confirming an attacker with double strike deals double damage.
func TestDuelistsHeritage_GrantsDoubleStrike(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Duelist's Heritage")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.Attack(1, gametest.PlayerA, "Grizzly Bears")
	g.ChooseTarget(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Grizzly Bears 2/2 with double strike unblocked: 2 + 2 = 4 damage.
	g.AssertLife(gametest.PlayerB, 16)
}

// Black Market accumulates charge counters when creatures die, then on the
// controller's first main phase pours {B} into the pool for each counter.
func TestBlackMarket_ChargeCountersOnCreatureDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Black Market")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Black Market", core.Charge, 1)
}

// On the controller's first (precombat) main phase, Black Market emits {B} for
// each charge counter — verified by casting a black spell using only that mana.
func TestBlackMarket_AddsBlackOnFirstMainPhase(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Black Market")
	g.AddCounters(1, core.Upkeep, gametest.PlayerA, "Black Market", core.Charge, 1)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Dark Ritual")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Dark Ritual")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Dark Ritual", 1)
}

func TestEternalThirst_GrantsLifelink(t *testing.T) {
	g := gametest.NewTestGame(t)
	bearID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	auraID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Eternal Thirst")
	g.Attach(auraID, bearID)
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Lifelink, true)
}

// Eternal Thirst's granted "Whenever a creature an opponent controls dies"
// trigger: enchanted creature gains a +1/+1 counter when an opponent's
// creature dies. Relies on LKI fallback for the dying permanent's controller.
func TestEternalThirst_CounterOnOpponentCreatureDeath(t *testing.T) {
	g := gametest.NewTestGame(t)
	bearID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	auraID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Eternal Thirst")
	g.Attach(auraID, bearID)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt", 2)
	// Two bolts (3+3=6) destroy the 3/3 Hill Giant via stacking damage.
	// Use a 3-damage source instead — easier: just kill with a single bigger spell.
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Bathe in Dragonfire")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 4)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Bathe in Dragonfire", "Hill Giant")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
	g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", core.P1P1, 1)
}

func TestVerdantEmbrace_BoostAndUpkeepSaproling(t *testing.T) {
	g := gametest.NewTestGame(t)
	bearID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	auraID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Verdant Embrace")
	g.Attach(auraID, bearID)
	g.StopAt(2, core.PostcombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 5, 5)
	// Trigger fires on each upkeep (turn 1 PlayerA upkeep + turn 2 PlayerB upkeep) = 2.
	g.AssertPermanentCount(gametest.PlayerA, "Saproling", 2)
}

func TestCelestialMantle_DoublesLifeOnCombatHit(t *testing.T) {
	g := gametest.NewTestGame(t)
	bearID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	auraID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Celestial Mantle")
	g.Attach(auraID, bearID)
	g.SetLife(gametest.PlayerA, 20)
	g.Attack(1, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 40)
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 5, 5)
}

func TestFaceOfDivinity_NoExtraKeywordsAlone(t *testing.T) {
	g := gametest.NewTestGame(t)
	bearID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	auraID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Face of Divinity")
	g.Attach(auraID, bearID)
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 4)
	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.FirstStrike, false)
	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Lifelink, false)
}

func TestFaceOfDivinity_WithSecondAuraGrantsKeywords(t *testing.T) {
	g := gametest.NewTestGame(t)
	bearID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	auraID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Face of Divinity")
	pacID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Pacifism")
	g.Attach(auraID, bearID)
	g.Attach(pacID, bearID)
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.FirstStrike, true)
	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Lifelink, true)
}

func TestLightningDiadem_BoostAttached(t *testing.T) {
	g := gametest.NewTestGame(t)
	bearID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	auraID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lightning Diadem")
	g.Attach(auraID, bearID)
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 4)
}

func TestVastwoodZendikon_AnimatesEnchantedLand(t *testing.T) {
	g := gametest.NewTestGame(t)
	forestID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	auraID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Vastwood Zendikon")
	g.Attach(auraID, forestID)
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Forest", 6, 4)
}

func TestNewHorizons_GrantsCounterAndManaAbility(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "New Horizons")
	g.ChooseTarget(gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "New Horizons", "Forest")
	g.StopAt(1, core.EndCombat)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", core.P1P1, 1)
}

// Sarkhan's Unsealing: casting a 4-power creature triggers 4-damage-to-any-target.
func TestSarkhansUnsealing_Power4Trigger(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sarkhan's Unsealing")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 5)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 5)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Serra Angel")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Hill Giant")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Serra Angel")
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 16)
}

// Sarkhan's Unsealing: casting a 7+ power creature deals 4 to opponent and
// each creature they control.
func TestSarkhansUnsealing_Power7Trigger(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sarkhan's Unsealing")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 4)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Force of Nature")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Force of Nature")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 16)
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
}

// Blessed Sanctuary: nontoken creature ETB creates a 2/2 white Unicorn token.
func TestBlessedSanctuary_NontokenETBCreatesUnicorn(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Blessed Sanctuary")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 4)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Unicorn", 1)
}

// Token ETB does NOT trigger Blessed Sanctuary's nontoken-ETB clause.
// Cast Sporemound after Sanctuary is on the battlefield — Sporemound enters
// (nontoken: 1 Unicorn), then plays a Forest so Sporemound's landfall makes a
// Saproling token. Sanctuary must NOT mint a second Unicorn for the token.
func TestBlessedSanctuary_TokenETBDoesNotTrigger(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Blessed Sanctuary")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 5)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Sporemound")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Forest")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Sporemound")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Saproling", 1)
	// Exactly one Unicorn (from nontoken Sporemound ETB), zero from the Saproling token.
	g.AssertPermanentCount(gametest.PlayerA, "Unicorn", 1)
}

// Noncombat damage to controller is prevented.
func TestBlessedSanctuary_PreventsBoltToController(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Blessed Sanctuary")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
	g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "PlayerA")
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 20)
}

// Noncombat damage to a creature you control is prevented.
func TestBlessedSanctuary_PreventsBoltToYourCreature(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Blessed Sanctuary")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
	g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Grizzly Bears")
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
}

// Combat damage is NOT prevented.
func TestBlessedSanctuary_DoesNotPreventCombatDamage(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Blessed Sanctuary")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.Attack(2, gametest.PlayerB, "Hill Giant")
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 17)
}

// Damage to opponent's creatures is NOT prevented.
func TestBlessedSanctuary_DoesNotPreventDamageToOpponentCreatures(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Blessed Sanctuary")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
}

func TestPathOfBravery_BoostsAtFullLifeAndGainsOnAttack(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Path of Bravery")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	// Life is 20 = starting life → +1/+1 active.
	g.Attack(1, gametest.PlayerA, "Grizzly Bears", "Hill Giant")
	g.StopAt(1, core.EndStep)
	g.Execute()
	// Two attackers → gain 2 life → 22.
	g.AssertLife(gametest.PlayerA, 22)
	// Boost still active during postcombat.
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 3, 3)
	g.AssertPowerToughness(gametest.PlayerA, "Hill Giant", 4, 4)
}

func TestPathOfBravery_BoostInactiveBelowStartingLife(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Path of Bravery")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.SetLife(gametest.PlayerA, 19)
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2)
}

func TestAjanisChosen_NonAuraEnchantmentCreatesCatToken(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ajani's Chosen")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 5)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Black Market")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Black Market")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Black Market", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Cat", 1)
	g.AssertPowerToughness(gametest.PlayerA, "Cat", 2, 2)
}

func TestAjanisChosen_AuraOnExistingCreatureMayReattachToToken(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ajani's Chosen")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Pacifism")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Pacifism", "Hill Giant")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Cat", 1)
	g.AssertAttachedTo(gametest.PlayerA, "Pacifism", "Cat")
}

func TestAjanisChosen_UnattachedAuraAttachesToCatToken(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ajani's Chosen")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	pacifismCard, err := mage.CreateCard("Pacifism")
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	playerAID := g.GetPlayer(gametest.PlayerA).PlayerID()
	pacifismCard.SetOwner(playerAID)
	g.PutOnBattlefield(pacifismCard, playerAID)
	g.PutTriggersOnStack()
	g.ResolveStack()

	g.AssertPermanentCount(gametest.PlayerA, "Cat", 1)
	g.AssertAttachedTo(gametest.PlayerA, "Pacifism", "Cat")
}

func TestParasiticImplant(t *testing.T) {
	t.Run("upkeep: enchanted creature's controller sacrifices it; aura controller gets a 1/1 colorless Phyrexian Myr artifact creature token", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		bearID := g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		auraID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Parasitic Implant")
		g.Attach(auraID, bearID)
		g.StopAt(3, core.Upkeep)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
		g.AssertPermanentCount(gametest.PlayerA, "Myr", 1)
		g.AssertPowerToughness(gametest.PlayerA, "Myr", 1, 1)
		g.AssertHasAbility(gametest.PlayerA, "Myr", core.Flying, false)
		g.AssertGraveyardCount(gametest.PlayerA, "Parasitic Implant", 1)

		pid := g.GetPlayer(gametest.PlayerA).PlayerID()
		myr := g.FindPermanentByName("Myr", pid)
		if myr == nil {
			t.Fatalf("Myr token not on battlefield")
		}
		if !myr.HasType(core.TypeArtifact) || !myr.HasType(core.TypeCreature) {
			t.Fatalf("Myr token types: got %v, want Artifact Creature", myr.Card.Types())
		}
		hasPhyrexian := false
		hasMyr := false
		for _, st := range myr.Card.SubTypes() {
			if st == "Phyrexian" {
				hasPhyrexian = true
			}
			if st == "Myr" {
				hasMyr = true
			}
		}
		if !hasPhyrexian || !hasMyr {
			t.Fatalf("Myr token subtypes: got %v, want [Phyrexian Myr]", myr.Card.SubTypes())
		}
		if cs := myr.Colors(); len(cs) != 0 {
			t.Fatalf("Myr token should be colorless, got %v", cs)
		}
	})

	t.Run("no host: aura is detached and falls off; no token", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Parasitic Implant")
		g.StopAt(2, core.Upkeep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Myr", 0)
	})
}

// TestRhysticStudy_OpponentDeclinesPay verifies that when an opponent casts
// a spell and declines/cannot pay {1}, the controller draws a card.
func TestRhysticStudy_OpponentDeclinesPay(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rhystic Study")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
	g.SetLife(gametest.PlayerA, 20)
	tpB := g.GetPlayer(gametest.PlayerB)
	tpB.QueueMayAbilityChoices(false)
	g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "PlayerA")
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Plains", 1)
	g.AssertLife(gametest.PlayerA, 17)
}

// Lawmage's Binding: enchanted creature's non-mana activated abilities can't be
// activated.
func TestLawmagesBinding_BlocksActivatedAbilities(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Prodigal Sorcerer")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lawmage's Binding")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 1)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lawmage's Binding", "Prodigal Sorcerer")
	g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerB, "Prodigal Sorcerer", "PlayerA")
	g.StopAt(2, core.EndStep)
	g.Execute()
	// Activation blocked: PlayerA still at 20 life.
	g.AssertLife(gametest.PlayerA, 20)
}

// Assault Formation: a 0/4 Wall attacking deals damage equal to its toughness.
func TestAssaultFormation_AssignsToughnessAsDamage(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Assault Formation")
	// Wall of Wood: 0/3 with defender. Use {G} to remove defender, then attack.
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Wood")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Assault Formation", "Wall of Wood")
	g.Attack(3, gametest.PlayerA, "Wall of Wood")
	g.StopAt(3, core.EndStep)
	g.Execute()
	// 0/3 wall now assigns 3 damage instead of 0.
	g.AssertLife(gametest.PlayerB, 17)
}
