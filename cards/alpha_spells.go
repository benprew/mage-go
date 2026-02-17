package cards

import "github.com/mage/mage"

func init() {
	registerAlphaSpells()
}

func registerAlphaSpells() {
	// ===== WHITE SPELLS =====

	mage.Register("Swords to Plowshares", func() mage.Card {
		c := mage.NewInstant("Swords to Plowshares", "{W}")
		sa := mage.NewSpellAbility(mage.ExileTargetCreatureGainLife())
		sa.AddTarget(mage.TargetCreature())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Disenchant", func() mage.Card {
		c := mage.NewInstant("Disenchant", "{1}{W}")
		sa := mage.NewSpellAbility(mage.DestroyTargetPermanent())
		sa.AddTarget(mage.TargetArtifactOrEnchantment())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Healing Salve", func() mage.Card {
		c := mage.NewInstant("Healing Salve", "{W}")
		// Target player gains 3 life (one of two modes, simplified)
		sa := mage.NewSpellAbility(mage.GainLifeTarget(3))
		sa.AddTarget(mage.TargetPlayer())
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
		sa := mage.NewSpellAbility(mage.ReturnFromGraveyardToBattlefield())
		sa.AddTarget(mage.TargetCreatureInYourGraveyard())
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
		sa := mage.NewSpellAbility(mage.GainXLife())
		sa.AddTarget(mage.TargetPlayer())
		c.AddAbility(sa)
		return c
	})

	// ===== BLUE SPELLS =====

	mage.Register("Ancestral Recall", func() mage.Card {
		c := mage.NewInstant("Ancestral Recall", "{U}")
		sa := mage.NewSpellAbility(mage.DrawCardsTarget(3))
		sa.AddTarget(mage.TargetPlayer())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Braingeyser", func() mage.Card {
		c := mage.NewSorcery("Braingeyser", "{X}{U}{U}")
		sa := mage.NewSpellAbility(mage.DrawXCards())
		sa.AddTarget(mage.TargetPlayer())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Counterspell", func() mage.Card {
		c := mage.NewInstant("Counterspell", "{U}{U}")
		sa := mage.NewSpellAbility(mage.CounterSpell())
		sa.AddTarget(mage.TargetSpellOnStack())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Unsummon", func() mage.Card {
		c := mage.NewInstant("Unsummon", "{U}")
		sa := mage.NewSpellAbility(mage.ReturnToHandTarget())
		sa.AddTarget(mage.TargetCreature())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Spell Blast", func() mage.Card {
		c := mage.NewInstant("Spell Blast", "{X}{U}")
		sa := mage.NewSpellAbility(mage.CounterSpell())
		sa.AddTarget(mage.TargetSpellOnStack())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Power Sink", func() mage.Card {
		c := mage.NewInstant("Power Sink", "{X}{U}")
		sa := mage.NewSpellAbility(mage.CounterSpell())
		sa.AddTarget(mage.TargetSpellOnStack())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Blue Elemental Blast", func() mage.Card {
		c := mage.NewInstant("Blue Elemental Blast", "{U}")
		// Counter target red spell
		sa := mage.NewSpellAbility(mage.CounterSpell())
		sa.AddTarget(mage.TargetSpellOnStack())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Twiddle", func() mage.Card {
		c := mage.NewInstant("Twiddle", "{U}")
		// Tap or untap target permanent (simplified: tap)
		sa := mage.NewSpellAbility(mage.TapTarget())
		sa.AddTarget(mage.TargetPermanent())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Timetwister", func() mage.Card {
		// Each player shuffles their hand and graveyard into library, then draws 7
		c := mage.NewSorcery("Timetwister", "{2}{U}")
		sa := mage.NewSpellAbility(mage.DrawCards(7)) // stub
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
		sa := mage.NewSpellAbility(mage.DestroyTarget())
		sa.AddTarget(mage.TargetCreature(
			mage.Not(mage.HasColorFilter(mage.Black)),
			mage.Not(mage.IsArtifact),
		))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Raise Dead", func() mage.Card {
		c := mage.NewSorcery("Raise Dead", "{B}")
		sa := mage.NewSpellAbility(mage.ReturnFromGraveyardToHandTarget())
		sa.AddTarget(mage.TargetCreatureInYourGraveyard())
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
		sa := mage.NewSpellAbility(mage.DrainXLife())
		sa.AddTarget(mage.TargetAnyTarget())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Mind Twist", func() mage.Card {
		c := mage.NewSorcery("Mind Twist", "{X}{B}")
		sa := mage.NewSpellAbility(mage.DiscardXCards())
		sa.AddTarget(mage.TargetPlayer())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Sinkhole", func() mage.Card {
		c := mage.NewSorcery("Sinkhole", "{B}{B}")
		sa := mage.NewSpellAbility(mage.DestroyTargetLand())
		sa.AddTarget(mage.TargetLand())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Animate Dead", func() mage.Card {
		c := mage.NewAura("Animate Dead", "{1}{B}")
		// When Animate Dead enters the battlefield, return target creature card from a graveyard
		// Simplified: put on battlefield
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
		sa := mage.NewSpellAbility(mage.CounterSpell())
		sa.AddTarget(mage.TargetSpellOnStack())
		c.AddAbility(sa)
		return c
	})

	// ===== RED SPELLS =====

	// Lightning Bolt already registered in spells.go

	mage.Register("Fireball", func() mage.Card {
		c := mage.NewSorcery("Fireball", "{X}{R}")
		sa := mage.NewSpellAbility(mage.DealXDamage())
		sa.AddTarget(mage.TargetAnyTarget())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Disintegrate", func() mage.Card {
		c := mage.NewSorcery("Disintegrate", "{X}{R}")
		sa := mage.NewSpellAbility(mage.DealXDamage())
		sa.AddTarget(mage.TargetAnyTarget())
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
		sa := mage.NewSpellAbility(mage.DestroyTargetArtifact())
		sa.AddTarget(mage.TargetArtifact())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Stone Rain", func() mage.Card {
		c := mage.NewSorcery("Stone Rain", "{2}{R}")
		sa := mage.NewSpellAbility(mage.DestroyTargetLand())
		sa.AddTarget(mage.TargetLand())
		c.AddAbility(sa)
		return c
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
		// Each player discards their hand and draws 7
		// Simplified: draw 7 cards
		sa := mage.NewSpellAbility(mage.DrawCards(7))
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
		// Double target creature's power, destroy it at end of turn
		// Simplified: boost +3/+0
		sa := mage.NewSpellAbility(mage.BoostTargetUntilEndOfTurn(3, 0))
		sa.AddTarget(mage.TargetCreature())
		c.AddAbility(sa)
		return c
	})

	// ===== GREEN SPELLS =====

	// Giant Growth already registered in spells.go

	mage.Register("Regrowth", func() mage.Card {
		c := mage.NewSorcery("Regrowth", "{1}{G}")
		sa := mage.NewSpellAbility(mage.ReturnFromGraveyardToHandTarget())
		sa.AddTarget(mage.TargetCardInYourGraveyard())
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
		c := mage.NewSorcery("Ice Storm", "{2}{G}")
		sa := mage.NewSpellAbility(mage.DestroyTargetLand())
		sa.AddTarget(mage.TargetLand())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Stream of Life", func() mage.Card {
		c := mage.NewSorcery("Stream of Life", "{X}{G}")
		sa := mage.NewSpellAbility(mage.GainXLife())
		sa.AddTarget(mage.TargetPlayer())
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
		sa := mage.NewSpellAbility(mage.BoostTargetUntilEndOfTurn(1, 0)) // simplified
		sa.AddTarget(mage.TargetCreature())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Righteousness", func() mage.Card {
		c := mage.NewInstant("Righteousness", "{W}")
		sa := mage.NewSpellAbility(mage.BoostTargetUntilEndOfTurn(7, 7))
		sa.AddTarget(mage.TargetCreature())
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
