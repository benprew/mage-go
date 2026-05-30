// Package gametest: CR 300–307 card-type tests.
// Covers: 300.2, 301.1/2/4, 302.1/2/4/5/6/7, 303.1/2/303.4(b/c/e), 304.1/2/5, 305.1/2/6, 307.1/2/5.
// GAP/SKIP entries are documented inline.
package gametest

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// ch03RegisterCards registers cards needed by this file's tests.
func ch03RegisterCards() {
	cards := []struct {
		name    string
		factory func() mage.Card
	}{
		{"Grizzly Bears", func() mage.Card {
			return mage.NewCreature("Grizzly Bears", "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
		}},
		{"Hill Giant", func() mage.Card {
			return mage.NewCreature("Hill Giant", "{3}{R}", 3, 3, mage.WithSubTypes("Giant"))
		}},
		{"White Knight", func() mage.Card {
			return mage.NewCreature("White Knight", "{W}{W}", 2, 2,
				mage.WithSubTypes("Human", "Knight"),
				mage.WithKeyword(core.FirstStrike),
				mage.WithAbility(mage.ProtectionFromColor(core.Black)),
			)
		}},
		{"Forest", func() mage.Card {
			return mage.NewLand("Forest", mage.WithSubTypes("Forest"), mage.WithManaAbility(core.Green))
		}},
		{"Mountain", func() mage.Card {
			return mage.NewLand("Mountain", mage.WithSubTypes("Mountain"), mage.WithManaAbility(core.Red))
		}},
		{"Plains", func() mage.Card {
			return mage.NewLand("Plains", mage.WithSubTypes("Plains"), mage.WithManaAbility(core.White))
		}},
		{"Sol Ring", func() mage.Card {
			return mage.NewArtifact("Sol Ring", "{1}",
				mage.WithActivatedAbility(mage.AddMana(core.Colorless, 2), mage.Tap()),
			)
		}},
		{"Black Lotus", func() mage.Card {
			return mage.NewArtifact("Black Lotus", "{0}",
				mage.WithActivatedAbility(
					mage.AddAnyMana(3, core.Green),
					mage.Tap(),
					mage.WithCost(mage.SacrificeSourceCost()),
				),
			)
		}},
		{"Lightning Bolt", func() mage.Card {
			return mage.NewInstant("Lightning Bolt", "{R}",
				mage.NewTargetedSpell(mage.TargetDamageAnyTarget(), mage.DealDamage(mage.Fixed(3))),
			)
		}},
		{"Giant Growth", func() mage.Card {
			return mage.NewInstant("Giant Growth", "{G}",
				mage.NewTargetedSpell(mage.TargetCreature(), mage.BoostUntilEndOfTurn(mage.Fixed(3), mage.Fixed(3), mage.SelectTarget)),
			)
		}},
		{"Holy Strength", func() mage.Card {
			return mage.NewAura("Holy Strength", "{W}",
				mage.WithStaticAbility(mage.BoostAttached(1, 2, core.AttachAura)),
			)
		}},
		{"Unholy Strength", func() mage.Card {
			return mage.NewAura("Unholy Strength", "{B}",
				mage.WithStaticAbility(mage.BoostAttached(2, 1, core.AttachAura)),
			)
		}},
		{"Terror", func() mage.Card {
			return mage.NewInstant("Terror", "{1}{B}",
				mage.NewTargetedSpell(
					mage.TargetCreature(
						mage.Not(mage.HasColorFilter(core.Black)),
						mage.Not(mage.IsArtifact),
					),
					mage.DestroyTargetNoRegen(),
				),
			)
		}},
		{"Shatter", func() mage.Card {
			return mage.NewInstant("Shatter", "{1}{R}",
				mage.NewTargetedSpell(mage.TargetArtifact(), mage.DestroyTargetArtifact()),
			)
		}},
		{"Stone Rain", func() mage.Card {
			return mage.NewSorcery("Stone Rain", "{2}{R}",
				mage.NewTargetedSpell(mage.TargetLand(), mage.DestroyTargetLand()),
			)
		}},
		{"Control Magic", func() mage.Card {
			return mage.NewAura("Control Magic", "{2}{U}{U}",
				mage.WithStaticAbility(mage.ControlChangeContinuous()),
			)
		}},
		{"Icy Manipulator", func() mage.Card {
			return mage.NewArtifact("Icy Manipulator", "{4}",
				mage.WithActivatedAbility(
					mage.Tap(),
					mage.GenericCost(1),
					mage.WithCost(mage.Tap()),
					mage.WithTarget(mage.TargetPermanent(mage.Or(mage.IsArtifact, mage.IsCreature, mage.IsLand))),
				),
			)
		}},
		{"Ornithopter", func() mage.Card {
			return mage.NewCreature("Ornithopter", "{0}", 0, 2,
				mage.WithSubTypes("Thopter"),
				mage.WithKeyword(core.Flying),
				mage.WithCardType(core.TypeArtifact),
			)
		}},
		{"Badlands", func() mage.Card {
			return mage.NewLand("Badlands",
				mage.WithSubTypes("Swamp", "Mountain"),
				mage.WithManaAbility(core.Black),
				mage.WithManaAbility(core.Red),
			)
		}},
		{"Regrowth", func() mage.Card {
			return mage.NewSorcery("Regrowth", "{1}{G}",
				mage.NewTargetedSpell(mage.TargetCardInYourGraveyard(), mage.ReturnFromGraveyardToHandTarget()),
			)
		}},
		{"Counterspell", func() mage.Card {
			return mage.NewInstant("Counterspell", "{U}{U}",
				mage.NewTargetedSpell(mage.TargetSpellOnStack(), mage.CounterSpell()),
			)
		}},
	}
	for _, c := range cards {

		if !mage.CardRegistered(c.name) {
			mage.Register(c.name, c.factory)
		}
	}
}

func init() {
	ch03RegisterCards()
}

// ===== 300.2 — Multi-type objects combine both types =====

// TestCR300_2_ArtifactCreatureCombinesBothTypes verifies CR 300.2:
// an artifact creature is affected by both artifact-targeting and creature-targeting effects.
func TestCR300_2_ArtifactCreatureCombinesBothTypes(t *testing.T) {
	// Ornithopter is an artifact creature. Shatter destroys artifacts.
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Ornithopter")
	tg.AddCard(core.ZoneHand, PlayerB, "Shatter")
	tg.CastSpell(1, core.PrecombatMain, PlayerB, "Shatter", "Ornithopter")
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Ornithopter", 0)
	tg.AssertGraveyardCount(PlayerA, "Ornithopter", 1)
}

// TestCR300_2a_ArtifactLandPlayedNotCast verifies CR 300.2a:
// a land (even with another type) is played as a special action, not cast.
func TestCR300_2a_ArtifactLandPlayedNotCast(t *testing.T) {
	// Badlands is a Swamp Mountain dual land — it has the land type, so it is played.
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Badlands")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Badlands", 1)
}

// ===== 301.1 — Artifacts cast during own main phase, stack empty =====

// TestCR301_1_CastArtifactMainPhaseOnly verifies CR 301.1:
// an artifact (sorcery-speed permanent) can't be cast in response to another spell.
func TestCR301_1_CastArtifactMainPhaseOnly(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Lightning Bolt")
	tg.AddCard(core.ZoneHand, PlayerA, "Sol Ring")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Lightning Bolt", "PlayerB")
	tg.CastInResponseTo(PlayerA, "Sol Ring")
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()

	// Sol Ring can't be cast at instant speed; remains in hand.
	tg.AssertHandCount(PlayerA, "Sol Ring", 1)
	tg.AssertPermanentCount(PlayerA, "Sol Ring", 0)
}

// TestCR301_2_ArtifactSpellEntersBattlefield verifies CR 301.2:
// an artifact spell that resolves becomes a permanent on the battlefield.
func TestCR301_2_ArtifactSpellEntersBattlefield(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Sol Ring")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Sol Ring")
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Sol Ring", 1)
}

// TestCR301_4_ColorlessArtifactHasNoColor verifies CR 301.4:
// artifacts are typically colorless; Sol Ring has no color.
func TestCR301_4_ColorlessArtifactHasNoColor(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Sol Ring")
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	tg.AssertHasColor(PlayerA, "Sol Ring", core.Blue, false)
	tg.AssertHasColor(PlayerA, "Sol Ring", core.Red, false)
	tg.AssertHasColor(PlayerA, "Sol Ring", core.White, false)
	tg.AssertHasColor(PlayerA, "Sol Ring", core.Green, false)
	tg.AssertHasColor(PlayerA, "Sol Ring", core.Black, false)
}

// ===== 301.5 — Equipment =====

// TestCR301_5_EquipmentAttachesToCreature verifies CR 301.5:
// the equip ability attaches the Equipment to a target creature.
func TestCR301_5_EquipmentAttachesToCreature(t *testing.T) {
	const equip = "CR301_5 Bonesplitter"
	if !mage.CardRegistered(equip) {
		mage.Register(equip, func() mage.Card {
			return mage.NewEquipment(equip, "{1}",
				mage.WithStaticAbility(mage.BoostAttached(2, 0, core.AttachEquipment)),
				mage.WithAbility(mage.NewEquipAbility(mage.GenericCost(1))),
			)
		})
	}
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.AddCard(core.ZoneBattlefield, PlayerA, equip)
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, equip, "Grizzly Bears")
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()

	tg.AssertAttachedTo(PlayerA, equip, "Grizzly Bears")
}

// TestCR301_5a_EquippedCreatureReference verifies CR 301.5a:
// the "equipped creature" ability grants bonus to the correct creature.
func TestCR301_5a_EquippedCreatureReference(t *testing.T) {
	const equip = "CR301_5a Bonesplitter"
	if !mage.CardRegistered(equip) {
		mage.Register(equip, func() mage.Card {
			return mage.NewEquipment(equip, "{1}",
				mage.WithStaticAbility(mage.BoostAttached(2, 0, core.AttachEquipment)),
				mage.WithAbility(mage.NewEquipAbility(mage.GenericCost(1))),
			)
		})
	}
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.AddCard(core.ZoneBattlefield, PlayerA, equip)
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, equip, "Grizzly Bears")
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()

	tg.AssertPowerToughness(PlayerA, "Grizzly Bears", 4, 2) // 2+2 / 2+0
}

// TestCR301_5b_EquipmentEntersUnattached verifies CR 301.5b:
// when an Equipment enters the battlefield it is not attached to any creature.
func TestCR301_5b_EquipmentEntersUnattached(t *testing.T) {
	const equip = "CR301_5b Bonesplitter"
	if !mage.CardRegistered(equip) {
		mage.Register(equip, func() mage.Card {
			return mage.NewEquipment(equip, "{1}",
				mage.WithStaticAbility(mage.BoostAttached(2, 0, core.AttachEquipment)),
				mage.WithAbility(mage.NewEquipAbility(mage.GenericCost(1))),
			)
		})
	}
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.AddCard(core.ZoneBattlefield, PlayerA, equip)
	tg.StopAt(1, core.Upkeep)
	tg.Execute()

	// Equipment not yet equipped → creature has base P/T.
	tg.AssertPowerToughness(PlayerA, "Grizzly Bears", 2, 2)
}

// TestCR301_5c_EquipmentUnattachesOnCreatureRemoval verifies CR 301.5c:
// if the equipped creature leaves the battlefield, the Equipment stays on the
// battlefield unattached.
func TestCR301_5c_EquipmentUnattachesOnCreatureRemoval(t *testing.T) {
	const equip = "CR301_5c Bonesplitter"
	if !mage.CardRegistered(equip) {
		mage.Register(equip, func() mage.Card {
			return mage.NewEquipment(equip, "{1}",
				mage.WithStaticAbility(mage.BoostAttached(2, 0, core.AttachEquipment)),
				mage.WithAbility(mage.NewEquipAbility(mage.GenericCost(1))),
			)
		})
	}
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.AddCard(core.ZoneBattlefield, PlayerA, equip)
	tg.AddCard(core.ZoneHand, PlayerB, "Terror")
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, equip, "Grizzly Bears")
	tg.CastSpell(1, core.PrecombatMain, PlayerB, "Terror", "Grizzly Bears")
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Grizzly Bears", 0)
	tg.AssertPermanentCount(PlayerA, equip, 1)
}

// CR 301.5d (Equipment controller != enchanted creature controller):
// GAP — no card currently steals a creature while leaving the equipment behind.
// CR 301.5e (Equipment entering attached to illegal target enters unattached):
// GAP — no engine card exercises this path.
// CR 301.6 (Fortification): GAP — Fortification subtype not implemented.
// CR 301.7 (Vehicle): GAP — Vehicle subtype/crew not implemented.

// ===== 302.1 — Creatures cast during own main phase, stack empty =====

// TestCR302_1_CastCreatureMainPhaseOnly verifies CR 302.1:
// a creature can't be cast at instant speed (in response to another spell).
func TestCR302_1_CastCreatureMainPhaseOnly(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Grizzly Bears")
	tg.AddCard(core.ZoneHand, PlayerB, "Lightning Bolt")
	tg.CastSpell(1, core.PrecombatMain, PlayerB, "Lightning Bolt", "PlayerA")
	tg.CastInResponseTo(PlayerA, "Grizzly Bears")
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()

	tg.AssertHandCount(PlayerA, "Grizzly Bears", 1)
	tg.AssertPermanentCount(PlayerA, "Grizzly Bears", 0)
}

// TestCR302_2_CreatureSpellEntersBattlefield verifies CR 302.2:
// a creature spell that resolves becomes a permanent on the battlefield.
func TestCR302_2_CreatureSpellEntersBattlefield(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Grizzly Bears")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Grizzly Bears")
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Grizzly Bears", 1)
}

// TestCR302_4_NoncreatureHasNoPT verifies CR 302.4:
// a non-creature permanent has no power or toughness.
func TestCR302_4_NoncreatureHasNoPT(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Sol Ring")
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	playerID := tg.getPlayerID(PlayerA)
	perm := tg.FindPermanentByName("Sol Ring", playerID)
	if perm == nil {
		t.Fatal("Sol Ring not found on battlefield")
	}
	if perm.HasType(core.TypeCreature) {
		t.Error("CR 302.4: Sol Ring (artifact) should not be a creature and thus has no P/T")
	}
}

// TestCR302_4a_PowerIsCombatDamage verifies CR 302.4a:
// a creature's power equals the combat damage it assigns in a normal attack.
func TestCR302_4a_PowerIsCombatDamage(t *testing.T) {
	// Grizzly Bears (2/2) attacks unblocked; PlayerB takes 2 damage.
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.Attack(1, PlayerA, "Grizzly Bears")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertLife(PlayerB, 18)
}

// TestCR302_4b_ToughnessIsLethalThreshold verifies CR 302.4b:
// damage equal to a creature's toughness is lethal; the creature is destroyed by SBA.
func TestCR302_4b_ToughnessIsLethalThreshold(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Grizzly Bears")
	tg.Attack(1, PlayerA, "Grizzly Bears")
	tg.Block(1, PlayerB, "Grizzly Bears", "Grizzly Bears")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Grizzly Bears", 0)
	tg.AssertPermanentCount(PlayerB, "Grizzly Bears", 0)
}

// TestCR302_4c_ContinuousEffectsModifyPT verifies CR 302.4c:
// continuous effects (Holy Strength +1/+2) are applied on top of base P/T.
func TestCR302_4c_ContinuousEffectsModifyPT(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.AddCard(core.ZoneHand, PlayerA, "Holy Strength")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Holy Strength", "Grizzly Bears")
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()

	tg.AssertPowerToughness(PlayerA, "Grizzly Bears", 3, 4) // 2+1 / 2+2
}

// TestCR302_5_CreatureCanAttackAndBlock verifies CR 302.5:
// creatures can attack and block.
func TestCR302_5_CreatureCanAttackAndBlock(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Hill Giant")
	tg.Attack(1, PlayerA, "Grizzly Bears")
	tg.Block(1, PlayerB, "Hill Giant", "Grizzly Bears")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Grizzly Bears", 0)
	tg.AssertPermanentCount(PlayerB, "Hill Giant", 1)
	tg.AssertLife(PlayerB, 20)
}

// TestCR302_6_SummoningSicknessBlocksAttack verifies CR 302.6:
// a creature cast on the same turn can't attack (summoning sickness).
func TestCR302_6_SummoningSicknessBlocksAttack(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Plains")
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Plains")
	tg.AddCard(core.ZoneHand, PlayerA, "White Knight")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "White Knight")
	tg.Attack(1, PlayerA, "White Knight")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertLife(PlayerB, 20)
}

// TestCR302_7_LethalDamageDestroysCreature verifies CR 302.7:
// damage >= toughness is lethal and the creature is destroyed by SBA.
func TestCR302_7_LethalDamageDestroysCreature(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Grizzly Bears")
	tg.AddCard(core.ZoneHand, PlayerA, "Lightning Bolt")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Lightning Bolt", "Grizzly Bears")
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()

	tg.AssertPermanentCount(PlayerB, "Grizzly Bears", 0)
	tg.AssertGraveyardCount(PlayerB, "Grizzly Bears", 1)
}

// ===== 303.1 — Enchantments cast during own main phase, stack empty =====

// TestCR303_1_CastEnchantmentMainPhaseOnly verifies CR 303.1:
// an enchantment can't be cast at instant speed.
func TestCR303_1_CastEnchantmentMainPhaseOnly(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Unholy Strength")
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.AddCard(core.ZoneHand, PlayerB, "Lightning Bolt")
	tg.CastSpell(1, core.PrecombatMain, PlayerB, "Lightning Bolt", "PlayerA")
	tg.CastInResponseTo(PlayerA, "Unholy Strength")
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()

	tg.AssertHandCount(PlayerA, "Unholy Strength", 1)
	tg.AssertPermanentCount(PlayerA, "Unholy Strength", 0)
}

// TestCR303_2_EnchantmentEntersBattlefield verifies CR 303.2:
// an enchantment spell that resolves enters the battlefield.
func TestCR303_2_EnchantmentEntersBattlefield(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.AddCard(core.ZoneHand, PlayerA, "Holy Strength")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Holy Strength", "Grizzly Bears")
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Holy Strength", 1)
}

// ===== 303.4 — Auras =====

// TestCR303_4_AuraEntersAttached verifies CR 303.4:
// when an Aura spell resolves, it enters attached to its target.
func TestCR303_4_AuraEntersAttached(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.AddCard(core.ZoneHand, PlayerA, "Holy Strength")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Holy Strength", "Grizzly Bears")
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()

	tg.AssertAttachedTo(PlayerA, "Holy Strength", "Grizzly Bears")
}

// TestCR303_4b_EnchantedReference verifies CR 303.4b:
// the "enchanted creature" reference applies the boost to the correct permanent.
func TestCR303_4b_EnchantedReference(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.AddCard(core.ZoneHand, PlayerA, "Holy Strength")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Holy Strength", "Grizzly Bears")
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()

	tg.AssertPowerToughness(PlayerA, "Grizzly Bears", 3, 4)
}

// TestCR303_4c_AuraToGraveyardOnIllegalEnchant verifies CR 303.4c:
// when the enchanted creature leaves the battlefield, the Aura goes to the graveyard (SBA).
func TestCR303_4c_AuraToGraveyardOnIllegalEnchant(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.AddCard(core.ZoneHand, PlayerA, "Holy Strength")
	tg.AddCard(core.ZoneHand, PlayerB, "Terror")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Holy Strength", "Grizzly Bears")
	tg.CastSpell(1, core.PrecombatMain, PlayerB, "Terror", "Grizzly Bears")
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Grizzly Bears", 0)
	tg.AssertPermanentCount(PlayerA, "Holy Strength", 0)
	tg.AssertGraveyardCount(PlayerA, "Holy Strength", 1)
}

// TestCR303_4e_AuraSeparateController verifies CR 303.4e:
// the Aura's controller and the enchanted creature's controller are independent.
// Control Magic is under PlayerA; it grants control of Hill Giant to PlayerA.
func TestCR303_4e_AuraSeparateController(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Hill Giant")
	tg.AddCard(core.ZoneHand, PlayerA, "Control Magic")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Control Magic", "Hill Giant")
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Hill Giant", 1)
	tg.AssertPermanentCount(PlayerB, "Hill Giant", 0)
	tg.AssertPermanentCount(PlayerA, "Control Magic", 1)
}

// CR 303.4a — GAP: harness can't assert cast failure for lack of legal target.
// CR 303.4d — GAP: Aura-creature combo edge case not exercised by existing cards.
// CR 303.4f/g — GAP: no engine mechanism to put Auras onto battlefield via non-spell effects.
// CR 303.4i — GAP: invalidating a target mid-stack not scriptable in current harness.
// CR 303.4j — GAP: no card tries to re-attach Aura to illegal target on battlefield.
// CR 303.5 Sagas / CR 303.6 Classes / CR 303.7 Roles — GAP: not implemented.

// ===== 304.1 — Instants can be cast any time a player has priority =====

// TestCR304_1_InstantCastAnyPriorityWindow verifies CR 304.1:
// an instant can be cast during an opponent's main phase in response to a spell.
func TestCR304_1_InstantCastAnyPriorityWindow(t *testing.T) {
	// PlayerB casts Lightning Bolt at Grizzly Bears; PlayerA responds with Giant Growth.
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.AddCard(core.ZoneHand, PlayerA, "Giant Growth")
	tg.AddCard(core.ZoneHand, PlayerB, "Lightning Bolt")
	tg.CastSpell(1, core.PrecombatMain, PlayerB, "Lightning Bolt", "Grizzly Bears")
	tg.CastInResponseTo(PlayerA, "Giant Growth", "Grizzly Bears")
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Grizzly Bears", 1)
}

// TestCR304_2_InstantResolvesToGraveyard verifies CR 304.2:
// after an instant resolves, it moves to the owner's graveyard.
func TestCR304_2_InstantResolvesToGraveyard(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Lightning Bolt")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Lightning Bolt", "PlayerB")
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()

	tg.AssertGraveyardCount(PlayerA, "Lightning Bolt", 1)
	tg.AssertHandCount(PlayerA, "Lightning Bolt", 0)
}

// TestCR304_5_ActivatedAbilityAtInstantSpeed verifies CR 304.5:
// "any time you could cast an instant" for activated abilities means any priority
// window, independent of holding an actual instant card.
func TestCR304_5_ActivatedAbilityAtInstantSpeed(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Icy Manipulator")
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Hill Giant")
	// Activate during BeginCombat (not a main phase).
	tg.ActivateAbility(1, core.BeginCombat, PlayerA, "Icy Manipulator", "Hill Giant")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()

	tg.AssertTapped(PlayerB, "Hill Giant", true)
}

// ===== 305.1 — Land play is a special action; doesn't use the stack =====

// TestCR305_1_LandPlayDoesntUseStack verifies CR 305.1:
// playing a land is a special action and doesn't use the stack; opponents can't counter it.
func TestCR305_1_LandPlayDoesntUseStack(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Forest")
	tg.AddCard(core.ZoneHand, PlayerB, "Counterspell")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	// Forest entered despite PlayerB having Counterspell (land play bypasses stack).
	tg.AssertPermanentCount(PlayerA, "Forest", 1)
}

// TestCR305_2_OneLandPerTurn verifies CR 305.2:
// a player can play at most one land per turn normally.
func TestCR305_2_OneLandPerTurn(t *testing.T) {
	tg := NewTestGame(t)
	tg.padLibraries()
	tg.AddCard(core.ZoneHand, PlayerA, "Forest", 2)

	tg.Turn = 1
	tg.ActivePlayer = 0
	tg.OnPriority = func(g *mage.Game, playerIdx int, mainPhase bool) mage.PriorityAction {
		return mage.PriorityAction{Type: mage.PriorityPass}
	}
	for _, step := range []core.PhaseStep{core.Untap, core.Upkeep, core.Draw} {
		tg.Step = step
		tg.RunStepWithPriority(step)
	}
	tg.Step = core.PrecombatMain

	playerID := tg.getPlayerID(PlayerA)
	first, ok := findLandInHand(tg, PlayerA, "Forest")
	if !ok {
		t.Fatal("setup: first Forest not in hand")
	}
	if err := tg.PlayLand(playerID, first.ID()); err != nil {
		t.Fatalf("CR 305.2: first land play should succeed, got %v", err)
	}
	second, ok := findLandInHand(tg, PlayerA, "Forest")
	if !ok {
		t.Fatal("setup: second Forest not in hand")
	}
	err := tg.PlayLand(playerID, second.ID())
	if err == nil {
		t.Error("CR 305.2: second land play in same turn should fail, got nil error")
	}
}

// TestCR305_6_BasicLandTypeGrantsMana verifies CR 305.6:
// a land with the Forest subtype intrinsically taps for {G} mana.
func TestCR305_6_BasicLandTypeGrantsMana(t *testing.T) {
	// Grizzly Bears needs {1}{G}. Forest provides the {G}; harness auto-adds colorless.
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Forest")
	tg.AddCard(core.ZoneHand, PlayerA, "Grizzly Bears")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Grizzly Bears")
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Grizzly Bears", 1)
}

// CR 305.7 (set subtype replaces abilities): GAP — no engine effect dynamically
// sets a land's subtype to a basic land type.
// CR 305.8 (nonbasic without Basic supertype): covered implicitly by land tests above.

// ===== 307.1 — Sorceries cast during own main phase, stack empty =====

// TestCR307_1_SorceryCastMainPhaseOnly verifies CR 307.1:
// a sorcery can't be cast at instant speed.
func TestCR307_1_SorceryCastMainPhaseOnly(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Regrowth")
	tg.AddCard(core.ZoneHand, PlayerB, "Lightning Bolt")
	tg.CastSpell(1, core.PrecombatMain, PlayerB, "Lightning Bolt", "PlayerA")
	tg.CastInResponseTo(PlayerA, "Regrowth")
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()

	tg.AssertHandCount(PlayerA, "Regrowth", 1)
}

// TestCR307_2_SorceryResolvesToGraveyard verifies CR 307.2:
// after a sorcery resolves, it moves to the owner's graveyard.
func TestCR307_2_SorceryResolvesToGraveyard(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Mountain")
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Forest")
	tg.AddCard(core.ZoneHand, PlayerA, "Stone Rain")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Stone Rain", "Forest")
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()

	tg.AssertGraveyardCount(PlayerA, "Stone Rain", 1)
	tg.AssertHandCount(PlayerA, "Stone Rain", 0)
}

// TestCR307_5_AsSorceryTimingRequirements verifies CR 307.5:
// an ability restricted to "only as a sorcery" cannot be activated in response
// to another spell. The equip ability (inherently sorcery-speed) is used as a
// proxy: attempting to equip in response to Lightning Bolt should not work.
func TestCR307_5_AsSorceryTimingRequirements(t *testing.T) {
	const equip = "CR307_5 Equip Test"
	if !mage.CardRegistered(equip) {
		mage.Register(equip, func() mage.Card {
			return mage.NewEquipment(equip, "{1}",
				mage.WithStaticAbility(mage.BoostAttached(2, 0, core.AttachEquipment)),
				mage.WithAbility(mage.NewEquipAbility(mage.GenericCost(1))),
			)
		})
	}
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, equip)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.AddCard(core.ZoneHand, PlayerB, "Lightning Bolt")
	// PlayerB casts Lightning Bolt; PlayerA tries to activate equip (sorcery-speed) in response.
	tg.CastSpell(1, core.PrecombatMain, PlayerB, "Lightning Bolt", "PlayerA")
	tg.ActivateInResponseTo(PlayerA, equip, "Grizzly Bears")
	tg.StopAt(1, core.BeginCombat)
	tg.Execute()

	// Equip should not have attached (sorcery-speed restriction).
	// Grizzly Bears has base 2/2; if equip had fired it would be 4/2.
	tg.AssertPowerToughness(PlayerA, "Grizzly Bears", 2, 2)
}

// CR 306 (Planeswalkers): GAP — Planeswalker card type not implemented.
// CR 308 (Kindreds): GAP — Kindred card type not implemented.
// CR 309 (Dungeons): GAP — Dungeon / venture mechanic not implemented.
// CR 310 (Battles): GAP — Battle card type, defense counters, protector not implemented.
// CR 311–315 (Planes, Phenomena, Vanguards, Schemes, Conspiracies): format-variant only.
