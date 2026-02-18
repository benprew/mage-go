package cards

import (
	"github.com/google/uuid"
	"github.com/mage/mage"
)

func init() {
	registerAlphaEnchantments()
}

func registerAlphaEnchantments() {
	// ===== GLOBAL ENCHANTMENTS =====

	mage.Register("Crusade", func() mage.Card {
		c := mage.NewEnchantment("Crusade", "{W}{W}")
		// White creatures get +1/+1
		c.AddAbility(mage.StaticAbility(
			mage.BoostAllCreaturesIncludingSelf(1, 1, mage.HasColorFilter(mage.White)),
		))
		return c
	})

	mage.Register("Bad Moon", func() mage.Card {
		c := mage.NewEnchantment("Bad Moon", "{1}{B}")
		// Black creatures get +1/+1
		c.AddAbility(mage.StaticAbility(
			mage.BoostAllCreaturesIncludingSelf(1, 1, mage.HasColorFilter(mage.Black)),
		))
		return c
	})

	mage.Register("Orcish Oriflamme", func() mage.Card {
		c := mage.NewEnchantment("Orcish Oriflamme", "{3}{R}")
		// Attacking creatures you control get +1/+0
		c.AddAbility(mage.StaticAbility(
			mage.BoostControlledCreatures(1, 0, mage.IsAttacking),
		))
		return c
	})

	mage.Register("Castle", func() mage.Card {
		c := mage.NewEnchantment("Castle", "{3}{W}")
		// Untapped creatures you control get +0/+2
		c.AddAbility(mage.StaticAbility(
			mage.BoostControlledCreatures(0, 2, mage.IsUntapped),
		))
		return c
	})

	// ===== AURAS (CREATURE ENCHANTMENTS) =====

	// Holy Strength already registered in enchantments.go

	mage.Register("Unholy Strength", func() mage.Card {
		return mage.NewBoostAura("Unholy Strength", "{B}", 2, 1)
	})

	mage.Register("Weakness", func() mage.Card {
		return mage.NewBoostAura("Weakness", "{B}", -2, -1)
	})

	mage.Register("Holy Armor", func() mage.Card {
		return mage.NewBoostAura("Holy Armor", "{W}", 0, 2)
	})

	mage.Register("Blessing", func() mage.Card {
		c := mage.NewAura("Blessing", "{W}{W}")
		// Enchanted creature has "{W}: +1/+1 until end of turn"
		c.AddAbility(mage.StaticAbility(
			mage.GrantActivatedAbilityToAttached(
				mage.BoostUntilEndOfTurn(mage.Fixed(1), mage.Fixed(1), mage.SelectSource),
				mage.ManaCostOf("{W}"),
				mage.AttachAura,
			),
		))
		return c
	})

	mage.Register("Lance", func() mage.Card {
		c := mage.NewAura("Lance", "{W}")
		c.AddAbility(mage.StaticAbility(
			mage.GrantAbilityToAttached(mage.FirstStrike, mage.AttachAura),
		))
		return c
	})

	mage.Register("Web", func() mage.Card {
		c := mage.NewAura("Web", "{G}")
		c.AddAbility(mage.StaticAbility(
			mage.BoostAttached(0, 2, mage.AttachAura),
			mage.GrantAbilityToAttached(mage.Reach, mage.AttachAura),
		))
		return c
	})

	mage.Register("Firebreathing", func() mage.Card {
		c := mage.NewAura("Firebreathing", "{R}")
		// Enchanted creature has "{R}: +1/+0 until end of turn"
		c.AddAbility(mage.StaticAbility(
			mage.GrantActivatedAbilityToAttached(
				mage.BoostUntilEndOfTurn(mage.Fixed(1), mage.Fixed(0), mage.SelectSource),
				mage.ManaCostOf("{R}"),
				mage.AttachAura,
			),
		))
		return c
	})

	mage.Register("Flight", func() mage.Card {
		c := mage.NewAura("Flight", "{U}")
		c.AddAbility(mage.StaticAbility(
			mage.GrantAbilityToAttached(mage.Flying, mage.AttachAura),
		))
		return c
	})

	mage.Register("Jump", func() mage.Card {
		c := mage.NewInstant("Jump", "{U}")
		// Target creature gains flying until end of turn
		sa := mage.NewTargetedSpell(mage.TargetCreature(), mage.GrantKeywordUntilEndOfTurn(mage.Flying, mage.SelectTarget))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Fear", func() mage.Card {
		c := mage.NewAura("Fear", "{B}{B}")
		c.AddAbility(mage.StaticAbility(
			mage.GrantAbilityToAttached(mage.Fear, mage.AttachAura),
		))
		return c
	})

	mage.Register("Burrowing", func() mage.Card {
		c := mage.NewAura("Burrowing", "{R}")
		c.AddAbility(mage.StaticAbility(
			mage.GrantAbilityToAttached(mage.Mountainwalk, mage.AttachAura),
		))
		return c
	})

	mage.Register("Invisibility", func() mage.Card {
		c := mage.NewAura("Invisibility", "{U}{U}")
		// Enchanted creature can't be blocked except by Walls.
		c.AddAbility(mage.StaticAbility(
			mage.GrantAbilityToAttached(mage.CantBeBlockedExceptByWalls, mage.AttachAura),
		))
		return c
	})

	mage.Register("Lure", func() mage.Card {
		c := mage.NewAura("Lure", "{1}{G}{G}")
		// All creatures able to block enchanted creature do so.
		c.AddAbility(mage.StaticAbility(
			mage.GrantAbilityToAttached(mage.MustBeBlocked, mage.AttachAura),
		))
		return c
	})

	mage.Register("Paralyze", func() mage.Card {
		c := mage.NewAura("Paralyze", "{B}")
		// When Paralyze enters the battlefield, tap enchanted creature.
		// Enchanted creature doesn't untap during its controller's untap step.
		c.AddAbility(mage.EntersBattlefieldTrigger(mage.TapAttachedCreature(), false))
		c.AddAbility(mage.StaticAbility(
			mage.PreventAttachedFromUntapping(mage.AttachAura),
		))
		return c
	})

	mage.Register("Earthbind", func() mage.Card {
		c := mage.NewAura("Earthbind", "{R}")
		// Enchanted creature loses flying
		c.AddAbility(mage.StaticAbility(
			mage.RemoveKeywordFromAttached(mage.Flying, mage.AttachAura),
		))
		return c
	})

	mage.Register("Control Magic", func() mage.Card {
		c := mage.NewAura("Control Magic", "{2}{U}{U}")
		c.AddAbility(mage.StaticAbility(
			mage.ControlChangeContinuous(),
		))
		return c
	})

	// Animate Dead is registered in alpha_spells.go

	// ===== ENCHANT LAND =====

	mage.Register("Wild Growth", func() mage.Card {
		c := mage.NewAura("Wild Growth", "{G}")
		// Whenever enchanted land is tapped for mana, its controller adds {G}.
		c.AddAbility(mage.NewAttachedManaBonusAbility(mage.Green))
		return c
	})

	mage.Register("Evil Presence", func() mage.Card {
		c := mage.NewAura("Evil Presence", "{B}")
		// Enchanted land is a Swamp.
		c.AddAbility(mage.StaticAbility(
			mage.ChangeAttachedSubTypes([]string{"Swamp"}),
		))
		return c
	})

	mage.Register("Phantasmal Terrain", func() mage.Card {
		c := mage.NewAura("Phantasmal Terrain", "{U}{U}")
		// Enchanted land is the basic land type of your choice.
		// Default choice: Island.
		c.AddAbility(mage.StaticAbility(
			mage.ChangeAttachedSubTypes([]string{"Island"}),
		))
		return c
	})

	mage.Register("Psychic Venom", func() mage.Card {
		c := mage.NewAura("Psychic Venom", "{1}{U}")
		// Whenever enchanted land becomes tapped, deal 2 damage to its controller.
		c.AddAbility(mage.WhenAttachedBecomesTappedTrigger(
			mage.DealDamageToPlayers(mage.Fixed(2), mage.SelectAttachedController()), false,
		))
		return c
	})

	// ===== WARD/PROTECTION ENCHANTMENTS =====

	copColors := []struct {
		name  string
		color mage.Color
	}{
		{"Circle of Protection: Blue", mage.Blue},
		{"Circle of Protection: Green", mage.Green},
		{"Circle of Protection: Red", mage.Red},
		{"Circle of Protection: White", mage.White},
	}
	for _, cop := range copColors {
		name := cop.name
		color := cop.color
		mage.Register(name, func() mage.Card {
			c := mage.NewEnchantment(name, "{1}{W}")
			// {1}: Prevent all damage from one source of this color this turn.
			ab := mage.NewActivatedAbility(
				mage.FuncEffect(
					"prevent all damage from one source of the chosen color",
					func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						g.Effects.AddColorPrevention(controller, color)
						return nil
					}),
				mage.GenericCost(1),
			)
			c.AddAbility(ab)
			return c
		})
	}

	mage.Register("Black Ward", func() mage.Card {
		c := mage.NewAura("Black Ward", "{W}")
		c.AddAbility(mage.StaticAbility(mage.GrantProtectionToAttached(mage.Black, mage.AttachAura)))
		return c
	})

	mage.Register("Blue Ward", func() mage.Card {
		c := mage.NewAura("Blue Ward", "{W}")
		c.AddAbility(mage.StaticAbility(mage.GrantProtectionToAttached(mage.Blue, mage.AttachAura)))
		return c
	})

	mage.Register("Green Ward", func() mage.Card {
		c := mage.NewAura("Green Ward", "{W}")
		c.AddAbility(mage.StaticAbility(mage.GrantProtectionToAttached(mage.Green, mage.AttachAura)))
		return c
	})

	mage.Register("Red Ward", func() mage.Card {
		c := mage.NewAura("Red Ward", "{W}")
		c.AddAbility(mage.StaticAbility(mage.GrantProtectionToAttached(mage.Red, mage.AttachAura)))
		return c
	})

	mage.Register("White Ward", func() mage.Card {
		c := mage.NewAura("White Ward", "{W}")
		c.AddAbility(mage.StaticAbility(mage.GrantProtectionToAttached(mage.White, mage.AttachAura)))
		return c
	})

	// ===== TRIGGERED ENCHANTMENTS =====

	mage.Register("Copper Tablet", func() mage.Card {
		c := mage.NewArtifact("Copper Tablet", "{2}")
		// At the beginning of each player's upkeep, Copper Tablet deals 1 damage to that player.
		c.AddAbility(mage.BeginningOfEachUpkeepTrigger(mage.DealDamageToPlayers(mage.Fixed(1), mage.SelectActivePlayer()), false))
		return c
	})

	mage.Register("Black Vise", func() mage.Card {
		c := mage.NewArtifact("Black Vise", "{1}")
		// At the beginning of each opponent's upkeep, Black Vise deals X damage to
		// that player, where X is the number of cards in their hand minus 4, minimum 0.
		c.AddAbility(mage.BeginningOfEachUpkeepTrigger(mage.BlackViseEffect(), false))
		return c
	})

	mage.Register("Wanderlust", func() mage.Card {
		c := mage.NewAura("Wanderlust", "{2}{G}")
		// At the beginning of enchanted creature's controller's upkeep, deal 1 damage
		c.AddAbility(mage.BeginningOfAttachedControllerUpkeepTrigger(
			mage.DealDamageToPlayers(mage.Fixed(1), mage.SelectAttachedController()), false,
		))
		return c
	})

	mage.Register("Cursed Land", func() mage.Card {
		c := mage.NewAura("Cursed Land", "{2}{B}{B}")
		// At the beginning of enchanted land's controller's upkeep, deal 1 damage
		c.AddAbility(mage.BeginningOfAttachedControllerUpkeepTrigger(
			mage.DealDamageToPlayers(mage.Fixed(1), mage.SelectAttachedController()), false,
		))
		return c
	})

	mage.Register("Feedback", func() mage.Card {
		c := mage.NewAura("Feedback", "{2}{U}")
		// At the beginning of enchanted enchantment's controller's upkeep, deal 1 damage
		c.AddAbility(mage.BeginningOfAttachedControllerUpkeepTrigger(
			mage.DealDamageToPlayers(mage.Fixed(1), mage.SelectAttachedController()), false,
		))
		return c
	})

	mage.Register("Warp Artifact", func() mage.Card {
		c := mage.NewAura("Warp Artifact", "{B}{B}")
		// At the beginning of enchanted artifact's controller's upkeep, deal 1 damage
		c.AddAbility(mage.BeginningOfAttachedControllerUpkeepTrigger(
			mage.DealDamageToPlayers(mage.Fixed(1), mage.SelectAttachedController()), false,
		))
		return c
	})

	mage.Register("Karma", func() mage.Card {
		c := mage.NewEnchantment("Karma", "{2}{W}{W}")
		// At the beginning of each player's upkeep, Karma deals damage to that player
		// equal to the number of Swamps they control.
		c.AddAbility(mage.BeginningOfEachUpkeepTrigger(mage.DealDamagePerSwamp(), false))
		return c
	})

	mage.Register("Farmstead", func() mage.Card {
		c := mage.NewAura("Farmstead", "{1}{W}{W}")
		// At the beginning of your upkeep, gain 1 life
		c.AddAbility(mage.BeginningOfUpkeepTrigger(mage.GainLife(1), false))
		return c
	})

	// ===== LUCKY CHARMS =====

	mage.Register("Crystal Rod", func() mage.Card {
		return mage.NewLuckyCharm("Crystal Rod", "{1}", mage.Blue)
	})

	mage.Register("Iron Star", func() mage.Card {
		return mage.NewLuckyCharm("Iron Star", "{1}", mage.Red)
	})

	mage.Register("Ivory Cup", func() mage.Card {
		return mage.NewLuckyCharm("Ivory Cup", "{1}", mage.White)
	})

	mage.Register("Throne of Bone", func() mage.Card {
		return mage.NewLuckyCharm("Throne of Bone", "{1}", mage.Black)
	})

	mage.Register("Wooden Sphere", func() mage.Card {
		return mage.NewLuckyCharm("Wooden Sphere", "{1}", mage.Green)
	})

	mage.Register("Soul Net", func() mage.Card {
		c := mage.NewArtifact("Soul Net", "{1}")
		// Whenever a creature dies, you may pay {1}. If you do, gain 1 life.
		c.AddAbility(mage.AnyCreatureDiesTrigger(mage.GainLife(1), true))
		return c
	})

	// ===== LACE CYCLE =====

	// Lace cycle: target permanent becomes the specified color.
	laces := []struct {
		name  string
		cost  string
		color mage.Color
	}{
		{"Chaoslace", "{R}", mage.Red},
		{"Deathlace", "{B}", mage.Black},
		{"Lifelace", "{G}", mage.Green},
		{"Purelace", "{W}", mage.White},
		{"Thoughtlace", "{U}", mage.Blue},
	}
	for _, lace := range laces {
		name := lace.name
		color := lace.color
		cost := lace.cost
		mage.Register(name, func() mage.Card {
			c := mage.NewInstant(name, cost)
			sa := mage.NewTargetedSpell(mage.TargetPermanent(), mage.ChangeColorEffect(color))
			c.AddAbility(sa)
			return c
		})
	}
}
