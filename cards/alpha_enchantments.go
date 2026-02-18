package cards

import "github.com/mage/mage"

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
		c := mage.NewAura("Unholy Strength", "{B}")
		c.AddAbility(mage.StaticAbility(
			mage.BoostAttached(2, 1, mage.AttachAura),
		))
		return c
	})

	mage.Register("Weakness", func() mage.Card {
		c := mage.NewAura("Weakness", "{B}")
		c.AddAbility(mage.StaticAbility(
			mage.BoostAttached(-2, -1, mage.AttachAura),
		))
		return c
	})

	mage.Register("Holy Armor", func() mage.Card {
		c := mage.NewAura("Holy Armor", "{W}")
		c.AddAbility(mage.StaticAbility(
			mage.BoostAttached(0, 2, mage.AttachAura),
		))
		return c
	})

	mage.Register("Blessing", func() mage.Card {
		c := mage.NewAura("Blessing", "{W}{W}")
		// Enchanted creature has "{W}: +1/+1 until end of turn"
		c.AddAbility(mage.StaticAbility(
			mage.GrantActivatedAbilityToAttached(
				mage.BoostSourceUntilEndOfTurn(1, 1),
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
				mage.BoostSourceUntilEndOfTurn(1, 0),
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
		sa := mage.NewSpellAbility(mage.GrantKeywordTargetUntilEndOfTurn(mage.Flying))
		sa.AddTarget(mage.TargetCreature())
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
		// Enchanted creature can't be blocked except by Walls
		// Simplified: no effect (would need custom blocking restriction)
		return c
	})

	mage.Register("Lure", func() mage.Card {
		c := mage.NewAura("Lure", "{1}{G}{G}")
		// All creatures able to block enchanted creature do so
		// Simplified: stub
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
		// Enchanted land taps for an additional G
		// Simplified: static effect on the enchantment
		return c
	})

	mage.Register("Evil Presence", func() mage.Card {
		c := mage.NewAura("Evil Presence", "{B}")
		// Enchanted land is a Swamp
		return c
	})

	mage.Register("Phantasmal Terrain", func() mage.Card {
		c := mage.NewAura("Phantasmal Terrain", "{U}{U}")
		// Enchanted land is the basic land type of your choice
		return c
	})

	mage.Register("Psychic Venom", func() mage.Card {
		c := mage.NewAura("Psychic Venom", "{1}{U}")
		// Whenever enchanted land becomes tapped, deal 2 damage to its controller
		return c
	})

	// ===== WARD/PROTECTION ENCHANTMENTS =====

	mage.Register("Circle of Protection: Blue", func() mage.Card {
		c := mage.NewEnchantment("Circle of Protection: Blue", "{1}{W}")
		// {1}: Prevent all damage from one blue source
		return c
	})

	mage.Register("Circle of Protection: Green", func() mage.Card {
		c := mage.NewEnchantment("Circle of Protection: Green", "{1}{W}")
		return c
	})

	mage.Register("Circle of Protection: Red", func() mage.Card {
		c := mage.NewEnchantment("Circle of Protection: Red", "{1}{W}")
		return c
	})

	mage.Register("Circle of Protection: White", func() mage.Card {
		c := mage.NewEnchantment("Circle of Protection: White", "{1}{W}")
		return c
	})

	mage.Register("Black Ward", func() mage.Card {
		c := mage.NewAura("Black Ward", "{W}")
		// Enchanted creature has protection from black
		return c
	})

	mage.Register("Blue Ward", func() mage.Card {
		c := mage.NewAura("Blue Ward", "{W}")
		return c
	})

	mage.Register("Green Ward", func() mage.Card {
		c := mage.NewAura("Green Ward", "{W}")
		return c
	})

	mage.Register("Red Ward", func() mage.Card {
		c := mage.NewAura("Red Ward", "{W}")
		return c
	})

	mage.Register("White Ward", func() mage.Card {
		c := mage.NewAura("White Ward", "{W}")
		return c
	})

	// ===== TRIGGERED ENCHANTMENTS =====

	mage.Register("Copper Tablet", func() mage.Card {
		c := mage.NewArtifact("Copper Tablet", "{2}")
		// At the beginning of each player's upkeep, deal 1 damage to that player
		// Simplified: upkeep trigger deals 1 to controller
		c.AddAbility(mage.BeginningOfUpkeepTrigger(mage.DealDamageToSourceController(1), false))
		return c
	})

	mage.Register("Black Vise", func() mage.Card {
		c := mage.NewArtifact("Black Vise", "{1}")
		// At the beginning of each opponent's upkeep, deal damage equal to cards in hand minus 4
		// Simplified: deal 1 on upkeep
		c.AddAbility(mage.BeginningOfUpkeepTrigger(mage.DealDamageToSourceController(1), false))
		return c
	})

	mage.Register("Wanderlust", func() mage.Card {
		c := mage.NewAura("Wanderlust", "{2}{G}")
		// At the beginning of enchanted creature's controller's upkeep, deal 1 damage
		c.AddAbility(mage.BeginningOfAttachedControllerUpkeepTrigger(
			mage.DealDamageToAttachedController(1), false,
		))
		return c
	})

	mage.Register("Cursed Land", func() mage.Card {
		c := mage.NewAura("Cursed Land", "{2}{B}{B}")
		// At the beginning of enchanted land's controller's upkeep, deal 1 damage
		c.AddAbility(mage.BeginningOfAttachedControllerUpkeepTrigger(
			mage.DealDamageToAttachedController(1), false,
		))
		return c
	})

	mage.Register("Feedback", func() mage.Card {
		c := mage.NewAura("Feedback", "{2}{U}")
		// At the beginning of enchanted enchantment's controller's upkeep, deal 1 damage
		c.AddAbility(mage.BeginningOfAttachedControllerUpkeepTrigger(
			mage.DealDamageToAttachedController(1), false,
		))
		return c
	})

	mage.Register("Warp Artifact", func() mage.Card {
		c := mage.NewAura("Warp Artifact", "{B}{B}")
		// At the beginning of enchanted artifact's controller's upkeep, deal 1 damage
		c.AddAbility(mage.BeginningOfAttachedControllerUpkeepTrigger(
			mage.DealDamageToAttachedController(1), false,
		))
		return c
	})

	mage.Register("Karma", func() mage.Card {
		c := mage.NewEnchantment("Karma", "{2}{W}{W}")
		// Upkeep: deal damage to each player equal to the number of Swamps they control
		// Simplified: deal 1 to each player on upkeep
		c.AddAbility(mage.BeginningOfUpkeepTrigger(mage.DealDamageToSourceController(1), false))
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
		c := mage.NewArtifact("Crystal Rod", "{1}")
		blue := mage.Blue
		c.AddAbility(mage.WheneverSpellCastTrigger(mage.GainLife(1), true, &blue))
		return c
	})

	mage.Register("Iron Star", func() mage.Card {
		c := mage.NewArtifact("Iron Star", "{1}")
		red := mage.Red
		c.AddAbility(mage.WheneverSpellCastTrigger(mage.GainLife(1), true, &red))
		return c
	})

	mage.Register("Ivory Cup", func() mage.Card {
		c := mage.NewArtifact("Ivory Cup", "{1}")
		white := mage.White
		c.AddAbility(mage.WheneverSpellCastTrigger(mage.GainLife(1), true, &white))
		return c
	})

	mage.Register("Throne of Bone", func() mage.Card {
		c := mage.NewArtifact("Throne of Bone", "{1}")
		black := mage.Black
		c.AddAbility(mage.WheneverSpellCastTrigger(mage.GainLife(1), true, &black))
		return c
	})

	mage.Register("Wooden Sphere", func() mage.Card {
		c := mage.NewArtifact("Wooden Sphere", "{1}")
		green := mage.Green
		c.AddAbility(mage.WheneverSpellCastTrigger(mage.GainLife(1), true, &green))
		return c
	})

	mage.Register("Soul Net", func() mage.Card {
		c := mage.NewArtifact("Soul Net", "{1}")
		// Whenever a creature dies, you may pay {1}. If you do, gain 1 life.
		c.AddAbility(mage.AnyCreatureDiesTrigger(mage.GainLife(1), true))
		return c
	})

	// ===== LACE CYCLE =====

	mage.Register("Chaoslace", func() mage.Card {
		c := mage.NewInstant("Chaoslace", "{R}")
		sa := mage.NewSpellAbility(mage.GainLife(0)) // stub
		c.AddAbility(sa)
		return c
	})

	mage.Register("Deathlace", func() mage.Card {
		c := mage.NewInstant("Deathlace", "{B}")
		sa := mage.NewSpellAbility(mage.GainLife(0)) // stub
		c.AddAbility(sa)
		return c
	})

	mage.Register("Lifelace", func() mage.Card {
		c := mage.NewInstant("Lifelace", "{G}")
		sa := mage.NewSpellAbility(mage.GainLife(0)) // stub
		c.AddAbility(sa)
		return c
	})

	mage.Register("Purelace", func() mage.Card {
		c := mage.NewInstant("Purelace", "{W}")
		sa := mage.NewSpellAbility(mage.GainLife(0)) // stub
		c.AddAbility(sa)
		return c
	})

	mage.Register("Thoughtlace", func() mage.Card {
		c := mage.NewInstant("Thoughtlace", "{U}")
		sa := mage.NewSpellAbility(mage.GainLife(0)) // stub
		c.AddAbility(sa)
		return c
	})
}
