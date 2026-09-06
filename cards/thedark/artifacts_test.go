package thedark

import (
	"testing"

	_ "github.com/benprew/mage-go/cards/legends"
	_ "github.com/benprew/mage-go/cards/limited"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestBarlsCage(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Barl's Cage")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")

	// Turn 2 is Player B's turn. Player B attacks with Grizzly Bears, tapping it.
	g.Attack(2, gametest.PlayerB, "Grizzly Bears")

	// Turn 3 (Player A): Activate Barl's Cage targeting Grizzly Bears
	g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Barl's Cage", "Grizzly Bears")

	// Turn 4 (Player B): Grizzly Bears should not untap during untap step
	g.StopAt(4, core.PrecombatMain)
	g.Execute()

	g.AssertTapped(gametest.PlayerB, "Grizzly Bears", true)
}

func TestBoneFlute(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bone Flute")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")

	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Bone Flute")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()

	// Bears 2/2 -> 1/2, Giant 3/3 -> 2/3
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 1, 2)
	g.AssertPowerToughness(gametest.PlayerB, "Hill Giant", 2, 3)
}

func TestBookOfRass(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Book of Rass")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")

	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Book of Rass")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()

	g.AssertLife(gametest.PlayerA, 18)
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestDarkSphere(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Dark Sphere")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 6)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Fireball")

	// Player A casts Fireball with X=5 targeting Player B.
	// Player B activates Dark Sphere in response targeting Fireball.
	// Fireball deals 5 damage. Half of 5 is 2.5, rounded down is 2 prevented. So 3 damage dealt.
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Fireball", 5, "PlayerB")
	g.ActivateInResponseTo(gametest.PlayerB, "Dark Sphere", "Fireball")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()

	g.AssertLife(gametest.PlayerB, 17) // 20 - (5 - 2) = 17
	g.AssertGraveyardCount(gametest.PlayerB, "Dark Sphere", 1)
}

func TestFountainOfYouth(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fountain of Youth")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)

	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Fountain of Youth")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()

	g.AssertLife(gametest.PlayerA, 21)
}

func TestLivingArmor(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Living Armor")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant") // CMC 4

	// Hill Giant is 3/3, CMC 4. Living Armor puts 4 +0/+1 counters on it.
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Living Armor", "Hill Giant")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()

	g.AssertPowerToughness(gametest.PlayerA, "Hill Giant", 3, 7)
	g.AssertGraveyardCount(gametest.PlayerA, "Living Armor", 1)
}

func TestReflectingMirror(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Reflecting Mirror")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island", 2) // Lightning Bolt is CMC 1, X = 2 * 1 = 2
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")

	// Player A casts Lightning Bolt targeting Player B
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	// Player B activates Reflecting Mirror targeting Lightning Bolt, paying X=2, new target Player A
	g.ActivateInResponseToWithX(gametest.PlayerB, "Reflecting Mirror", 2, "Lightning Bolt", "PlayerA")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()

	// Player A should take the 3 damage
	g.AssertLife(gametest.PlayerA, 17)
	g.AssertLife(gametest.PlayerB, 20)
}

func TestRunesword(t *testing.T) {
	t.Run("buff and damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Runesword")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")

		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.ActivateAbility(1, core.DeclareAttackers, gametest.PlayerA, "Runesword", "Grizzly Bears")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()

		// Bears is 2/2 + 2/0 = 4/2 attacking unblocked, deals 4 to Player B
		g.AssertLife(gametest.PlayerB, 16)
		// Bears did not leave the battlefield, Runesword still on BF
		g.AssertPermanentCount(gametest.PlayerA, "Runesword", 1)
	})

	t.Run("leaves battlefield triggers sacrifice", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Runesword")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")

		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.ActivateAbility(1, core.DeclareAttackers, gametest.PlayerA, "Runesword", "Grizzly Bears")
		// Player B burns Bears
		g.CastSpell(1, core.PostcombatMain, gametest.PlayerB, "Lightning Bolt", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()

		// Runesword sacrificed because Bears left the battlefield
		g.AssertGraveyardCount(gametest.PlayerA, "Runesword", 1)
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}

func TestSkullOfOrm(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Skull of Orm")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 5)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Holy Strength")

	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Skull of Orm", "Holy Strength")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()

	g.AssertHandCount(gametest.PlayerA, "Holy Strength", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Holy Strength", 0)
}

func TestStandingStones(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Standing Stones")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Healing Salve") // requires {W}

	// Turn 1 PrecombatMain: Activate Standing Stones for White mana, paying 1 life and 1 mana from Forest
	g.ChooseManaColor(gametest.PlayerA, core.White)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Standing Stones")
	// Cast Healing Salve with the generated White mana
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Healing Salve", "PlayerA")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()

	// Started at 20, paid 1 life to Standing Stones = 19, gained 3 from Salve = 22
	g.AssertLife(gametest.PlayerA, 22)
}

func TestStoneCalendar(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Stone Calendar")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")

	// Grizzly Bears costs {1}{G}. With Stone Calendar it costs {G}.
	// We only have 1 Forest, so without Stone Calendar we couldn't cast it.
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()

	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestTormodsCrypt(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tormod's Crypt")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerB, "Grizzly Bears", 3)

	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Tormod's Crypt", "PlayerB")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()

	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 0)
	g.AssertExileCount("Grizzly Bears", 3)
}

func TestTowerOfCoireall(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tower of Coireall")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Wood")

	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Tower of Coireall", "Grizzly Bears")
	g.Attack(1, gametest.PlayerA, "Grizzly Bears")
	g.Block(1, gametest.PlayerB, "Wall of Wood", "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()

	// Wall of Wood cannot block Grizzly Bears
	g.AssertLife(gametest.PlayerB, 18)
}

func TestWandOfIth(t *testing.T) {
	t.Run("reveals land and pays life", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wand of Ith")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Forest")

		// Player B chooses to pay 1 life to not discard
		g.GetPlayer(gametest.PlayerB).QueueMayAbilityChoices(true)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Wand of Ith", "PlayerB")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()

		g.AssertLife(gametest.PlayerB, 19)
		g.AssertHandCount(gametest.PlayerB, "Forest", 1)
	})

	t.Run("reveals nonland and discards", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wand of Ith")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Hill Giant") // CMC 4

		// Player B refuses to pay 4 life
		g.GetPlayer(gametest.PlayerB).QueueMayAbilityChoices(false)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Wand of Ith", "PlayerB")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()

		g.AssertLife(gametest.PlayerB, 20)
		g.AssertHandCount(gametest.PlayerB, "Hill Giant", 0)
		g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
	})
}

func TestWarBarge(t *testing.T) {
	t.Run("gains islandwalk", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "War Barge")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")

		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "War Barge", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Hill Giant", "Grizzly Bears")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()

		// Islandwalk prevents blocking
		g.AssertLife(gametest.PlayerB, 18)
	})

	t.Run("leaves battlefield destroys creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "War Barge")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains", 2)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Disenchant")

		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "War Barge", "Grizzly Bears")
		g.CastSpell(1, core.PostcombatMain, gametest.PlayerB, "Disenchant", "War Barge")
		g.StopAt(1, core.EndStep)
		g.Execute()

		// Grizzly Bears destroyed when War Barge leaves battlefield
		g.AssertGraveyardCount(gametest.PlayerA, "War Barge", 1)
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}
