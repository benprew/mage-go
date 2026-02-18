package cards

import "github.com/mage/mage"

func init() {
	registerAlphaSpells()
}

func registerAlphaSpells() {
	// ===== WHITE SPELLS =====

	mage.Register("Swords to Plowshares", func() mage.Card {
		c := mage.NewInstant("Swords to Plowshares", "{W}")
		sa := mage.NewTargetedSpell(mage.TargetCreature(), mage.ExileTargetCreatureGainLife())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Disenchant", func() mage.Card {
		c := mage.NewInstant("Disenchant", "{1}{W}")
		sa := mage.NewTargetedSpell(mage.TargetArtifactOrEnchantment(), mage.DestroyTargetPermanent())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Healing Salve", func() mage.Card {
		c := mage.NewInstant("Healing Salve", "{W}")
		// Target player gains 3 life (one of two modes, simplified)
		sa := mage.NewTargetedSpell(mage.TargetPlayer(), mage.GainLifeTarget(3))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Armageddon", func() mage.Card {
		c := mage.NewSorcery("Armageddon", "{3}{W}")
		sa := mage.NewSpellAbility(mage.DestroyAllLands())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Balance", func() mage.Card {
		// Complex card - stub
		c := mage.NewSorcery("Balance", "{1}{W}")
		sa := mage.NewSpellAbility(mage.GainLife(0)) // stub
		c.AddAbility(sa)
		return c
	})

	mage.Register("Resurrection", func() mage.Card {
		c := mage.NewSorcery("Resurrection", "{2}{W}{W}")
		sa := mage.NewTargetedSpell(mage.TargetCreatureInYourGraveyard(), mage.ReturnFromGraveyardToBattlefield())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Reverse Damage", func() mage.Card {
		c := mage.NewInstant("Reverse Damage", "{1}{W}{W}")
		sa := mage.NewSpellAbility(mage.GainLife(0)) // stub - prevent damage is complex
		c.AddAbility(sa)
		return c
	})

	mage.Register("Death Ward", func() mage.Card {
		c := mage.NewInstant("Death Ward", "{W}")
		sa := mage.NewSpellAbility(mage.GainLife(0)) // stub - regenerate target
		c.AddAbility(sa)
		return c
	})

	mage.Register("Guardian Angel", func() mage.Card {
		c := mage.NewInstant("Guardian Angel", "{X}{W}")
		// Prevent the next X damage that would be dealt to any target this turn
		sa := mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.PreventXDamageToTarget())
		c.AddAbility(sa)
		return c
	})

	// ===== BLUE SPELLS =====

	mage.Register("Ancestral Recall", func() mage.Card {
		c := mage.NewInstant("Ancestral Recall", "{U}")
		sa := mage.NewTargetedSpell(mage.TargetPlayer(), mage.DrawCardsTarget(3))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Braingeyser", func() mage.Card {
		c := mage.NewSorcery("Braingeyser", "{X}{U}{U}")
		sa := mage.NewTargetedSpell(mage.TargetPlayer(), mage.DrawXCards())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Counterspell", func() mage.Card {
		c := mage.NewInstant("Counterspell", "{U}{U}")
		sa := mage.NewTargetedSpell(mage.TargetSpellOnStack(), mage.CounterSpell())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Unsummon", func() mage.Card {
		c := mage.NewInstant("Unsummon", "{U}")
		sa := mage.NewTargetedSpell(mage.TargetCreature(), mage.ReturnToHandTarget())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Spell Blast", func() mage.Card {
		c := mage.NewInstant("Spell Blast", "{X}{U}")
		// Counter target spell with mana value X
		sa := mage.NewTargetedSpell(mage.TargetSpellOnStack(), mage.CounterSpellIfXMeetsCMC())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Power Sink", func() mage.Card {
		c := mage.NewInstant("Power Sink", "{X}{U}")
		// Counter target spell unless its controller pays {X}
		sa := mage.NewTargetedSpell(mage.TargetSpellOnStack(), mage.PowerSinkEffect())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Blue Elemental Blast", func() mage.Card {
		c := mage.NewInstant("Blue Elemental Blast", "{U}")
		// Counter target red spell (only counters if the spell is red)
		sa := mage.NewTargetedSpell(mage.TargetSpellOnStack(), mage.CounterSpellIfColor(mage.Red))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Twiddle", func() mage.Card {
		c := mage.NewInstant("Twiddle", "{U}")
		// You may tap or untap target artifact, creature, or land
		sa := mage.NewTargetedSpell(mage.TargetPermanent(), mage.TapOrUntapTarget())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Timetwister", func() mage.Card {
		// Each player shuffles their hand and graveyard into library, then draws 7
		c := mage.NewSorcery("Timetwister", "{2}{U}")
		sa := mage.NewSpellAbility(mage.ShuffleGraveyardIntoLibraryAndDraw(7))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Time Walk", func() mage.Card {
		c := mage.NewSorcery("Time Walk", "{1}{U}")
		sa := mage.NewSpellAbility(mage.ExtraTurn())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Sleight of Mind", func() mage.Card {
		c := mage.NewInstant("Sleight of Mind", "{U}")
		sa := mage.NewSpellAbility(mage.GainLife(0)) // stub
		c.AddAbility(sa)
		return c
	})

	mage.Register("Stasis", func() mage.Card {
		c := mage.NewEnchantment("Stasis", "{1}{U}")
		// Stub - nobody untaps is very complex
		return c
	})

	// ===== BLACK SPELLS =====

	mage.Register("Dark Ritual", func() mage.Card {
		c := mage.NewInstant("Dark Ritual", "{B}")
		sa := mage.NewSpellAbility(mage.AddMana(mage.Black, 3))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Terror", func() mage.Card {
		c := mage.NewInstant("Terror", "{1}{B}")
		sa := mage.NewTargetedSpell(mage.TargetCreature(
			mage.Not(mage.HasColorFilter(mage.Black)),
			mage.Not(mage.IsArtifact),
		), mage.DestroyTarget())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Raise Dead", func() mage.Card {
		c := mage.NewSorcery("Raise Dead", "{B}")
		sa := mage.NewTargetedSpell(mage.TargetCreatureInYourGraveyard(), mage.ReturnFromGraveyardToHandTarget())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Demonic Tutor", func() mage.Card {
		c := mage.NewSorcery("Demonic Tutor", "{1}{B}")
		sa := mage.NewSpellAbility(mage.SearchLibraryToHand())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Drain Life", func() mage.Card {
		c := mage.NewSorcery("Drain Life", "{X}{1}{B}")
		sa := mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DrainXLife())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Mind Twist", func() mage.Card {
		c := mage.NewSorcery("Mind Twist", "{X}{B}")
		sa := mage.NewTargetedSpell(mage.TargetPlayer(), mage.DiscardXCards())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Sinkhole", func() mage.Card {
		return mage.NewLandDestruction("Sinkhole", "{B}{B}")
	})

	mage.Register("Animate Dead", func() mage.Card {
		c := mage.NewAura("Animate Dead", "{1}{B}")
		// Return target creature card from a graveyard to the battlefield.
		// Animate Dead attaches to it. Enchanted creature gets -1/-0.
		sa := mage.NewSpellAbility(mage.ReturnFromGraveyardToBattlefield())
		c.AddAbility(sa)
		c.AddAbility(mage.StaticAbility(
			mage.BoostAttached(-1, 0, mage.AttachAura),
		))
		return c
	})

	mage.Register("Pestilence", func() mage.Card {
		c := mage.NewEnchantment("Pestilence", "{2}{B}{B}")
		// {B}: Deal 1 damage to each creature and each player
		ab := mage.NewActivatedAbility(
			mage.CompositeEffects("deal 1 damage to each creature and each player",
				mage.DealDamageToAllCreatures(1, nil),
				mage.DealDamageToEachPlayer(1),
			),
			mage.ManaCostOf("{B}"),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Red Elemental Blast", func() mage.Card {
		c := mage.NewInstant("Red Elemental Blast", "{R}")
		// Counter target blue spell (only counters if the spell is blue)
		sa := mage.NewTargetedSpell(mage.TargetSpellOnStack(), mage.CounterSpellIfColor(mage.Blue))
		c.AddAbility(sa)
		return c
	})

	// ===== RED SPELLS =====

	// Lightning Bolt already registered in spells.go

	mage.Register("Fireball", func() mage.Card {
		c := mage.NewSorcery("Fireball", "{X}{R}")
		sa := mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealXDamage())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Disintegrate", func() mage.Card {
		c := mage.NewSorcery("Disintegrate", "{X}{R}")
		sa := mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealXDamage())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Earthquake", func() mage.Card {
		c := mage.NewSorcery("Earthquake", "{X}{R}")
		// Deal X damage to each creature without flying and each player
		sa := mage.NewSpellAbility(mage.CompositeEffects(
			"deal X damage to each creature without flying and each player",
			mage.DealXDamageToAllCreatures(mage.NotHasKeywordFilter(mage.Flying)),
			mage.DealXDamageToEachPlayer(),
		))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Shatter", func() mage.Card {
		c := mage.NewInstant("Shatter", "{1}{R}")
		sa := mage.NewTargetedSpell(mage.TargetArtifact(), mage.DestroyTargetArtifact())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Stone Rain", func() mage.Card {
		return mage.NewLandDestruction("Stone Rain", "{2}{R}")
	})

	mage.Register("Flashfires", func() mage.Card {
		c := mage.NewSorcery("Flashfires", "{3}{R}")
		// Destroy all Plains
		sa := mage.NewSpellAbility(mage.DestroyAllMatching(
			mage.HasSubType("Plains"),
			"destroy all Plains",
		))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Wheel of Fortune", func() mage.Card {
		c := mage.NewSorcery("Wheel of Fortune", "{2}{R}")
		// Each player discards their hand, then draws seven cards
		sa := mage.NewSpellAbility(mage.DiscardHandAndDraw(7))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Fork", func() mage.Card {
		// Copy target instant or sorcery spell
		c := mage.NewInstant("Fork", "{R}{R}")
		sa := mage.NewSpellAbility(mage.GainLife(0)) // stub
		c.AddAbility(sa)
		return c
	})

	mage.Register("Berserk", func() mage.Card {
		c := mage.NewInstant("Berserk", "{G}")
		// Double target creature's power until end of turn, destroy it at end of turn
		sa := mage.NewTargetedSpell(mage.TargetCreature(), mage.CompositeEffects(
			"Target creature's power is doubled. Destroy it at end of turn.",
			mage.DoubleTargetPower(),
			mage.DestroyTargetAtEndOfTurn(),
		))
		c.AddAbility(sa)
		return c
	})

	// ===== GREEN SPELLS =====

	// Giant Growth already registered in spells.go

	mage.Register("Regrowth", func() mage.Card {
		c := mage.NewSorcery("Regrowth", "{1}{G}")
		sa := mage.NewTargetedSpell(mage.TargetCardInYourGraveyard(), mage.ReturnFromGraveyardToHandTarget())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Hurricane", func() mage.Card {
		c := mage.NewSorcery("Hurricane", "{X}{G}")
		// Deal X damage to each creature with flying and each player
		sa := mage.NewSpellAbility(mage.CompositeEffects(
			"deal X damage to each creature with flying and each player",
			mage.DealXDamageToAllCreatures(mage.HasKeywordFilter(mage.Flying)),
			mage.DealXDamageToEachPlayer(),
		))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Tranquility", func() mage.Card {
		c := mage.NewSorcery("Tranquility", "{2}{G}")
		sa := mage.NewSpellAbility(mage.DestroyAllEnchantments())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Tsunami", func() mage.Card {
		c := mage.NewSorcery("Tsunami", "{3}{G}")
		// Destroy all Islands
		sa := mage.NewSpellAbility(mage.DestroyAllMatching(
			mage.HasSubType("Island"),
			"destroy all Islands",
		))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Ice Storm", func() mage.Card {
		return mage.NewLandDestruction("Ice Storm", "{2}{G}")
	})

	mage.Register("Stream of Life", func() mage.Card {
		c := mage.NewSorcery("Stream of Life", "{X}{G}")
		sa := mage.NewTargetedSpell(mage.TargetPlayer(), mage.GainXLife())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Fog", func() mage.Card {
		c := mage.NewInstant("Fog", "{G}")
		sa := mage.NewSpellAbility(mage.PreventAllCombatDamage())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Channel", func() mage.Card {
		// Complex card - stub
		c := mage.NewSorcery("Channel", "{G}{G}")
		sa := mage.NewSpellAbility(mage.GainLife(0)) // stub
		c.AddAbility(sa)
		return c
	})

	mage.Register("Howl from Beyond", func() mage.Card {
		c := mage.NewInstant("Howl from Beyond", "{X}{B}")
		// Target creature gets +X/+0 until end of turn
		sa := mage.NewTargetedSpell(mage.TargetCreature(), mage.BoostTargetXUntilEndOfTurn(true, false))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Righteousness", func() mage.Card {
		c := mage.NewInstant("Righteousness", "{W}")
		sa := mage.NewTargetedSpell(mage.TargetCreature(), mage.BoostTargetUntilEndOfTurn(7, 7))
		c.AddAbility(sa)
		return c
	})

	// ===== COLORLESS SPELLS =====

	mage.Register("Chaos Orb", func() mage.Card {
		// Physical dexterity card - not implementable
		c := mage.NewArtifact("Chaos Orb", "{2}")
		return c
	})
}
