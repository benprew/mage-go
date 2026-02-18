package cards

import "github.com/mage/mage"

func init() {
	registerAlphaArtifacts()
}

func registerAlphaArtifacts() {
	// ===== MOX CYCLE =====

	moxen := []struct {
		name  string
		color mage.Color
	}{
		{"Mox Pearl", mage.White},
		{"Mox Sapphire", mage.Blue},
		{"Mox Jet", mage.Black},
		{"Mox Ruby", mage.Red},
		{"Mox Emerald", mage.Green},
	}

	for _, m := range moxen {
		name := m.name
		color := m.color
		mage.Register(name, func() mage.Card {
			c := mage.NewArtifact(name, "{0}")
			c.AddAbility(mage.NewManaAbility(color))
			return c
		})
	}

	// ===== MANA ARTIFACTS =====

	mage.Register("Black Lotus", func() mage.Card {
		c := mage.NewArtifact("Black Lotus", "{0}")
		// {T}, Sacrifice: Add 3 mana of any one color
		// Default choice: Green (not hardcoded to Black)
		ab := mage.NewActivatedAbility(
			mage.AddAnyMana(3, mage.Green),
			mage.TapSourceCost(),
		
			mage.WithCost(mage.SacrificeSourceCost()),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Sol Ring", func() mage.Card {
		c := mage.NewArtifact("Sol Ring", "{1}")
		// {T}: Add {C}{C}
		ab := mage.NewActivatedAbility(
			mage.AddMana(mage.Colorless, 2),
			mage.TapSourceCost(),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Basalt Monolith", func() mage.Card {
		c := mage.NewArtifact("Basalt Monolith", "{3}")
		c.SetIntrinsicDoesNotUntap(true) // doesn't untap during untap step
		// {T}: Add {C}{C}{C}
		ab := mage.NewActivatedAbility(
			mage.AddMana(mage.Colorless, 3),
			mage.TapSourceCost(),
		)
		c.AddAbility(ab)
		// {3}: Untap Basalt Monolith
		untap := mage.NewActivatedAbility(
			mage.UntapSource(),
			mage.GenericCost(3),
		)
		c.AddAbility(untap)
		return c
	})

	mage.Register("Mana Vault", func() mage.Card {
		c := mage.NewArtifact("Mana Vault", "{1}")
		c.SetIntrinsicDoesNotUntap(true) // doesn't untap during untap step
		// {T}: Add {C}{C}{C}
		ab := mage.NewActivatedAbility(
			mage.AddMana(mage.Colorless, 3),
			mage.TapSourceCost(),
		)
		c.AddAbility(ab)
		// At the beginning of your upkeep, deal 1 damage to you
		c.AddAbility(mage.BeginningOfUpkeepTrigger(mage.DealDamageToSourceController(1), false))
		return c
	})

	// ===== UTILITY ARTIFACTS =====

	mage.Register("Jayemdae Tome", func() mage.Card {
		c := mage.NewArtifact("Jayemdae Tome", "{4}")
		// {4}, {T}: Draw a card
		ab := mage.NewActivatedAbility(
			mage.DrawCards(1),
			mage.GenericCost(4),
		
			mage.WithCost(mage.TapSourceCost()),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Disrupting Scepter", func() mage.Card {
		c := mage.NewArtifact("Disrupting Scepter", "{3}")
		// {3}, {T}: Target player discards a card
		ab := mage.NewActivatedAbility(
			mage.DiscardCards(1),
			mage.GenericCost(3),
		
			mage.WithCost(mage.TapSourceCost()),
		
			mage.WithTarget(mage.TargetPlayer()),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Icy Manipulator", func() mage.Card {
		c := mage.NewArtifact("Icy Manipulator", "{4}")
		// {1}, {T}: Tap target permanent
		ab := mage.NewActivatedAbility(
			mage.TapTarget(),
			mage.GenericCost(1),
		
			mage.WithCost(mage.TapSourceCost()),
		
			mage.WithTarget(mage.TargetPermanent()),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Rod of Ruin", func() mage.Card {
		c := mage.NewArtifact("Rod of Ruin", "{4}")
		// {3}, {T}: Deal 1 damage to any target
		ab := mage.NewActivatedAbility(
			mage.DealDamage(1),
			mage.GenericCost(3),
		
			mage.WithCost(mage.TapSourceCost()),
		
			mage.WithTarget(mage.TargetAnyTarget()),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("The Hive", func() mage.Card {
		c := mage.NewArtifact("The Hive", "{5}")
		// {5}, {T}: Create a 1/1 colorless Insect artifact creature token with flying named Wasp
		// Simplified: draw a card (token creation is complex)
		ab := mage.NewActivatedAbility(
			mage.GainLife(1), // stub
			mage.GenericCost(5),
		
			mage.WithCost(mage.TapSourceCost()),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Winter Orb", func() mage.Card {
		c := mage.NewArtifact("Winter Orb", "{2}")
		// Players can't untap more than one land during their untap step
		// Stub - complex
		return c
	})

	mage.Register("Meekstone", func() mage.Card {
		c := mage.NewArtifact("Meekstone", "{1}")
		// Creatures with power 3 or greater don't untap during their controller's untap step
		return c
	})

	mage.Register("Howling Mine", func() mage.Card {
		c := mage.NewArtifact("Howling Mine", "{2}")
		// At the beginning of each player's draw step, that player draws an additional card
		return c
	})

	mage.Register("Forcefield", func() mage.Card {
		c := mage.NewArtifact("Forcefield", "{3}")
		// {1}: The next time an unblocked creature would deal combat damage to you, prevent all but 1
		return c
	})

	// ===== GAUNTLET =====

	mage.Register("Gauntlet of Might", func() mage.Card {
		c := mage.NewArtifact("Gauntlet of Might", "{4}")
		// Red creatures get +1/+1
		c.AddAbility(mage.StaticAbility(
			mage.BoostAllCreaturesIncludingSelf(1, 1, mage.HasColorFilter(mage.Red)),
		))
		// Whenever a Mountain is tapped for mana, its controller adds an additional {R}
		c.AddAbility(mage.NewManaBonusAbility(mage.HasSubType("Mountain"), mage.Red))
		return c
	})

	// ===== STEAL ARTIFACT =====

	mage.Register("Steal Artifact", func() mage.Card {
		c := mage.NewAura("Steal Artifact", "{2}{U}{U}")
		c.AddAbility(mage.StaticAbility(
			mage.ControlChangeContinuous(),
		))
		return c
	})

	mage.Register("Copy Artifact", func() mage.Card {
		c := mage.NewArtifact("Copy Artifact", "{1}{U}")
		// Copy - stub
		return c
	})

	// ===== MISC ARTIFACTS =====

	mage.Register("Ankh of Mishra", func() mage.Card {
		c := mage.NewArtifact("Ankh of Mishra", "{2}")
		// Whenever a land enters the battlefield, deal 2 damage to that land's controller
		c.AddAbility(mage.WheneverLandEntersBattlefieldTrigger(
			mage.DealDamageToEventController(2), false,
		))
		return c
	})

	mage.Register("Jade Monolith", func() mage.Card {
		c := mage.NewArtifact("Jade Monolith", "{4}")
		return c
	})

	mage.Register("Jade Statue", func() mage.Card {
		c := mage.NewArtifact("Jade Statue", "{4}")
		return c
	})

	mage.Register("Glasses of Urza", func() mage.Card {
		c := mage.NewArtifact("Glasses of Urza", "{1}")
		return c
	})

	mage.Register("Helm of Chatzuk", func() mage.Card {
		c := mage.NewArtifact("Helm of Chatzuk", "{1}")
		return c
	})

	mage.Register("Sunglasses of Urza", func() mage.Card {
		c := mage.NewArtifact("Sunglasses of Urza", "{3}")
		return c
	})

	mage.Register("Kormus Bell", func() mage.Card {
		c := mage.NewArtifact("Kormus Bell", "{4}")
		return c
	})

	mage.Register("Cyclopean Tomb", func() mage.Card {
		c := mage.NewArtifact("Cyclopean Tomb", "{4}")
		return c
	})

	mage.Register("Illusionary Mask", func() mage.Card {
		c := mage.NewArtifact("Illusionary Mask", "{2}")
		return c
	})

	mage.Register("Nevinyrral's Disk", func() mage.Card {
		c := mage.NewArtifact("Nevinyrral's Disk", "{4}")
		c.SetEntersTapped(true)
		// {1}, {T}: Destroy all artifacts, creatures, and enchantments
		ab := mage.NewActivatedAbility(
			mage.CompositeEffects("destroy all artifacts, creatures, and enchantments",
				mage.DestroyAllCreatures(),
				mage.DestroyAllEnchantments(),
				mage.DestroyAllMatching(mage.IsArtifact, "destroy all artifacts"),
			),
			mage.GenericCost(1),
		
			mage.WithCost(mage.TapSourceCost()),
		)
		c.AddAbility(ab)
		return c
	})

	// ===== DEATHGRIP / LIFEFORCE =====

	mage.Register("Deathgrip", func() mage.Card {
		c := mage.NewEnchantment("Deathgrip", "{B}{B}")
		// {B}{B}: Counter target green spell (only counters if green)
		ab := mage.NewActivatedAbility(
			mage.CounterSpellIfColor(mage.Green),
			mage.ManaCostOf("{B}{B}"),
		
			mage.WithTarget(mage.TargetSpellOnStack()),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Lifeforce", func() mage.Card {
		c := mage.NewEnchantment("Lifeforce", "{G}{G}")
		// {G}{G}: Counter target black spell (only counters if black)
		ab := mage.NewActivatedAbility(
			mage.CounterSpellIfColor(mage.Black),
			mage.ManaCostOf("{G}{G}"),
		
			mage.WithTarget(mage.TargetSpellOnStack()),
		)
		c.AddAbility(ab)
		return c
	})

	// ===== LIVING LANDS / INSTILL ENERGY =====

	mage.Register("Living Lands", func() mage.Card {
		c := mage.NewEnchantment("Living Lands", "{3}{G}")
		// All Forests are 1/1 creatures - stub
		return c
	})

	mage.Register("Instill Energy", func() mage.Card {
		c := mage.NewAura("Instill Energy", "{G}")
		c.AddAbility(mage.StaticAbility(
			mage.GrantAbilityToAttached(mage.Haste, mage.AttachAura),
		))
		return c
	})

	mage.Register("Mana Flare", func() mage.Card {
		c := mage.NewEnchantment("Mana Flare", "{2}{R}")
		// Whenever a player taps a land for mana, that land produces one additional mana
		// Stub - complex trigger
		return c
	})

	mage.Register("Sacrifice", func() mage.Card {
		c := mage.NewInstant("Sacrifice", "{B}")
		// As an additional cost, sacrifice a creature. Add mana equal to that creature's CMC.
		// Stub
		sa := mage.NewSpellAbility(mage.AddMana(mage.Black, 1))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Word of Command", func() mage.Card {
		c := mage.NewInstant("Word of Command", "{B}{B}")
		// Very complex - stub
		sa := mage.NewSpellAbility(mage.GainLife(0))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Camouflage", func() mage.Card {
		c := mage.NewInstant("Camouflage", "{G}")
		sa := mage.NewSpellAbility(mage.GainLife(0)) // stub
		c.AddAbility(sa)
		return c
	})

	mage.Register("Raging River", func() mage.Card {
		c := mage.NewEnchantment("Raging River", "{R}{R}")
		// Complex - stub
		return c
	})

	mage.Register("Natural Selection", func() mage.Card {
		c := mage.NewInstant("Natural Selection", "{G}")
		sa := mage.NewSpellAbility(mage.GainLife(0)) // stub
		c.AddAbility(sa)
		return c
	})

	mage.Register("Lich", func() mage.Card {
		c := mage.NewEnchantment("Lich", "{B}{B}{B}{B}")
		// Very complex - stub
		return c
	})

	mage.Register("Island Sanctuary", func() mage.Card {
		c := mage.NewEnchantment("Island Sanctuary", "{1}{W}")
		// Stub
		return c
	})

	mage.Register("Power Surge", func() mage.Card {
		c := mage.NewEnchantment("Power Surge", "{R}{R}")
		return c
	})

	mage.Register("Mana Short", func() mage.Card {
		c := mage.NewInstant("Mana Short", "{2}{U}")
		sa := mage.NewSpellAbility(mage.GainLife(0)) // stub
		c.AddAbility(sa)
		return c
	})

	mage.Register("Drain Power", func() mage.Card {
		c := mage.NewSorcery("Drain Power", "{U}{U}")
		sa := mage.NewSpellAbility(mage.GainLife(0)) // stub
		c.AddAbility(sa)
		return c
	})

	mage.Register("Simulacrum", func() mage.Card {
		c := mage.NewInstant("Simulacrum", "{1}{B}")
		sa := mage.NewSpellAbility(mage.GainLife(0)) // stub
		c.AddAbility(sa)
		return c
	})

	mage.Register("Blaze of Glory", func() mage.Card {
		c := mage.NewInstant("Blaze of Glory", "{W}")
		sa := mage.NewSpellAbility(mage.GainLife(0)) // stub
		c.AddAbility(sa)
		return c
	})

	mage.Register("False Orders", func() mage.Card {
		c := mage.NewInstant("False Orders", "{R}")
		sa := mage.NewSpellAbility(mage.GainLife(0)) // stub
		c.AddAbility(sa)
		return c
	})

	mage.Register("Lifetap", func() mage.Card {
		c := mage.NewEnchantment("Lifetap", "{U}{U}")
		return c
	})

	mage.Register("Conversion", func() mage.Card {
		c := mage.NewEnchantment("Conversion", "{2}{W}{W}")
		return c
	})

	mage.Register("Gloom", func() mage.Card {
		c := mage.NewEnchantment("Gloom", "{2}{B}")
		return c
	})

	mage.Register("Magnetic Mountain", func() mage.Card {
		c := mage.NewEnchantment("Magnetic Mountain", "{1}{R}{R}")
		return c
	})

	mage.Register("Consecrate Land", func() mage.Card {
		c := mage.NewAura("Consecrate Land", "{W}")
		return c
	})

	mage.Register("Fastbond", func() mage.Card {
		c := mage.NewEnchantment("Fastbond", "{G}")
		return c
	})

	mage.Register("Kudzu", func() mage.Card {
		c := mage.NewAura("Kudzu", "{1}{G}{G}")
		return c
	})

	mage.Register("Regeneration", func() mage.Card {
		c := mage.NewAura("Regeneration", "{1}{G}")
		return c
	})

	mage.Register("Siren's Call", func() mage.Card {
		c := mage.NewInstant("Siren's Call", "{U}")
		sa := mage.NewSpellAbility(mage.GainLife(0)) // stub
		c.AddAbility(sa)
		return c
	})

	mage.Register("Magical Hack", func() mage.Card {
		c := mage.NewInstant("Magical Hack", "{U}")
		sa := mage.NewSpellAbility(mage.GainLife(0)) // stub
		c.AddAbility(sa)
		return c
	})
}
