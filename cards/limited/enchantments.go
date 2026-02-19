package limited

import (
	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

func init() {
	registerEnchantments()
}

func registerEnchantments() {
	// ===== GLOBAL ENCHANTMENTS =====

	mage.Register("Crusade", func() mage.Card {
		return mage.NewEnchantment("Crusade", "{W}{W}",
			mage.WithAbility(mage.StaticAbility(
				mage.BoostAllCreaturesIncludingSelf(1, 1, mage.HasColorFilter(core.White)),
			)),
		)
	})

	mage.Register("Bad Moon", func() mage.Card {
		return mage.NewEnchantment("Bad Moon", "{1}{B}",
			mage.WithAbility(mage.StaticAbility(
				mage.BoostAllCreaturesIncludingSelf(1, 1, mage.HasColorFilter(core.Black)),
			)),
		)
	})

	mage.Register("Orcish Oriflamme", func() mage.Card {
		return mage.NewEnchantment("Orcish Oriflamme", "{3}{R}",
			mage.WithAbility(mage.StaticAbility(
				mage.BoostControlledCreatures(1, 0, mage.IsAttacking),
			)),
		)
	})

	mage.Register("Castle", func() mage.Card {
		return mage.NewEnchantment("Castle", "{3}{W}",
			mage.WithAbility(mage.StaticAbility(
				mage.BoostControlledCreatures(0, 2, mage.IsUntapped),
			)),
		)
	})

	// ===== AURAS (CREATURE ENCHANTMENTS) =====

	mage.Register("Holy Strength", func() mage.Card {
		return mage.NewBoostAura("Holy Strength", "{W}", 1, 2)
	})

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
		return mage.NewAura("Blessing", "{W}{W}",
			mage.WithAbility(mage.StaticAbility(
				mage.GrantActivatedAbilityToAttached(
					mage.BoostUntilEndOfTurn(mage.Fixed(1), mage.Fixed(1), mage.SelectSource),
					mage.ManaCostOf("{W}"),
					core.AttachAura,
				),
			)),
		)
	})

	mage.Register("Lance", func() mage.Card {
		return mage.NewAura("Lance", "{W}",
			mage.WithAbility(mage.StaticAbility(
				mage.GrantAbilityToAttached(core.FirstStrike, core.AttachAura),
			)),
		)
	})

	mage.Register("Web", func() mage.Card {
		return mage.NewAura("Web", "{G}",
			mage.WithAbility(mage.StaticAbility(
				mage.BoostAttached(0, 2, core.AttachAura),
				mage.GrantAbilityToAttached(core.Reach, core.AttachAura),
			)),
		)
	})

	mage.Register("Firebreathing", func() mage.Card {
		return mage.NewAura("Firebreathing", "{R}",
			mage.WithAbility(mage.StaticAbility(
				mage.GrantActivatedAbilityToAttached(
					mage.BoostUntilEndOfTurn(mage.Fixed(1), mage.Fixed(0), mage.SelectSource),
					mage.ManaCostOf("{R}"),
					core.AttachAura,
				),
			)),
		)
	})

	mage.Register("Flight", func() mage.Card {
		return mage.NewAura("Flight", "{U}",
			mage.WithAbility(mage.StaticAbility(
				mage.GrantAbilityToAttached(core.Flying, core.AttachAura),
			)),
		)
	})

	mage.Register("Jump", func() mage.Card {
		return mage.NewInstant("Jump", "{U}",
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetCreature(), mage.GrantKeywordUntilEndOfTurn(core.Flying, mage.SelectTarget))),
		)
	})

	mage.Register("Fear", func() mage.Card {
		return mage.NewAura("Fear", "{B}{B}",
			mage.WithAbility(mage.StaticAbility(
				mage.GrantAbilityToAttached(core.Fear, core.AttachAura),
			)),
		)
	})

	mage.Register("Burrowing", func() mage.Card {
		return mage.NewAura("Burrowing", "{R}",
			mage.WithAbility(mage.StaticAbility(
				mage.GrantAbilityToAttached(core.Mountainwalk, core.AttachAura),
			)),
		)
	})

	mage.Register("Invisibility", func() mage.Card {
		return mage.NewAura("Invisibility", "{U}{U}",
			mage.WithAbility(mage.StaticAbility(
				mage.GrantAbilityToAttached(core.CantBeBlockedExceptByWalls, core.AttachAura),
			)),
		)
	})

	mage.Register("Lure", func() mage.Card {
		return mage.NewAura("Lure", "{1}{G}{G}",
			mage.WithAbility(mage.StaticAbility(
				mage.GrantAbilityToAttached(core.MustBeBlocked, core.AttachAura),
			)),
		)
	})

	mage.Register("Paralyze", func() mage.Card {
		return mage.NewAura("Paralyze", "{B}",
			mage.WithAbility(mage.EntersBattlefieldTrigger(mage.TapAttachedCreature(), false)),
			mage.WithAbility(mage.StaticAbility(
				mage.PreventAttachedFromUntapping(core.AttachAura),
			)),
		)
	})

	mage.Register("Earthbind", func() mage.Card {
		return mage.NewAura("Earthbind", "{R}",
			mage.WithAbility(mage.StaticAbility(
				mage.RemoveKeywordFromAttached(core.Flying, core.AttachAura),
			)),
		)
	})

	mage.Register("Control Magic", func() mage.Card {
		return mage.NewAura("Control Magic", "{2}{U}{U}",
			mage.WithAbility(mage.StaticAbility(mage.ControlChangeContinuous())),
		)
	})

	// Animate Dead is registered in spells.go

	// ===== ENCHANT LAND =====

	mage.Register("Wild Growth", func() mage.Card {
		return mage.NewAura("Wild Growth", "{G}",
			mage.WithAbility(mage.NewAttachedManaBonusAbility(core.Green)),
		)
	})

	mage.Register("Evil Presence", func() mage.Card {
		return mage.NewAura("Evil Presence", "{B}",
			mage.WithAbility(mage.StaticAbility(
				mage.ChangeAttachedSubTypes([]string{"Swamp"}),
			)),
		)
	})

	mage.Register("Phantasmal Terrain", func() mage.Card {
		return mage.NewAura("Phantasmal Terrain", "{U}{U}",
			mage.WithAbility(mage.StaticAbility(
				mage.ChangeAttachedSubTypes([]string{"Island"}),
			)),
		)
	})

	mage.Register("Psychic Venom", func() mage.Card {
		return mage.NewAura("Psychic Venom", "{1}{U}",
			mage.WithAbility(mage.WhenAttachedBecomesTappedTrigger(
				mage.DealDamageToPlayers(mage.Fixed(2), mage.SelectAttachedController()), false,
			)),
		)
	})

	// ===== WARD/PROTECTION ENCHANTMENTS =====

	copColors := []struct {
		name  string
		color core.Color
	}{
		{"Circle of Protection: Blue", core.Blue},
		{"Circle of Protection: Green", core.Green},
		{"Circle of Protection: Red", core.Red},
		{"Circle of Protection: White", core.White},
	}
	for _, cop := range copColors {
		name := cop.name
		color := cop.color
		mage.Register(name, func() mage.Card {
			return mage.NewEnchantment(name, "{1}{W}",
				// {1}: Prevent all damage from one source of this color this turn.
				mage.WithAbility(mage.NewActivatedAbility(
					mage.FuncEffect(
						"prevent all damage from one source of the chosen color",
						func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							g.Effects.AddColorPrevention(controller, color)
							return nil
						}),
					mage.GenericCost(1),
				)),
			)
		})
	}

	mage.Register("Black Ward", func() mage.Card {
		return mage.NewAura("Black Ward", "{W}",
			mage.WithAbility(mage.StaticAbility(mage.GrantProtectionToAttached(core.Black, core.AttachAura))),
		)
	})

	mage.Register("Blue Ward", func() mage.Card {
		return mage.NewAura("Blue Ward", "{W}",
			mage.WithAbility(mage.StaticAbility(mage.GrantProtectionToAttached(core.Blue, core.AttachAura))),
		)
	})

	mage.Register("Green Ward", func() mage.Card {
		return mage.NewAura("Green Ward", "{W}",
			mage.WithAbility(mage.StaticAbility(mage.GrantProtectionToAttached(core.Green, core.AttachAura))),
		)
	})

	mage.Register("Red Ward", func() mage.Card {
		return mage.NewAura("Red Ward", "{W}",
			mage.WithAbility(mage.StaticAbility(mage.GrantProtectionToAttached(core.Red, core.AttachAura))),
		)
	})

	mage.Register("White Ward", func() mage.Card {
		return mage.NewAura("White Ward", "{W}",
			mage.WithAbility(mage.StaticAbility(mage.GrantProtectionToAttached(core.White, core.AttachAura))),
		)
	})

	// ===== TRIGGERED ENCHANTMENTS =====

	mage.Register("Copper Tablet", func() mage.Card {
		return mage.NewArtifact("Copper Tablet", "{2}",
			mage.WithAbility(mage.BeginningOfEachUpkeepTrigger(mage.DealDamageToPlayers(mage.Fixed(1), mage.SelectActivePlayer()), false)),
		)
	})

	mage.Register("Black Vise", func() mage.Card {
		return mage.NewArtifact("Black Vise", "{1}",
			mage.WithAbility(mage.BeginningOfEachUpkeepTrigger(mage.BlackViseEffect(), false)),
		)
	})

	mage.Register("Wanderlust", func() mage.Card {
		return mage.NewAura("Wanderlust", "{2}{G}",
			mage.WithAbility(mage.BeginningOfAttachedControllerUpkeepTrigger(
				mage.DealDamageToPlayers(mage.Fixed(1), mage.SelectAttachedController()), false,
			)),
		)
	})

	mage.Register("Cursed Land", func() mage.Card {
		return mage.NewAura("Cursed Land", "{2}{B}{B}",
			mage.WithAbility(mage.BeginningOfAttachedControllerUpkeepTrigger(
				mage.DealDamageToPlayers(mage.Fixed(1), mage.SelectAttachedController()), false,
			)),
		)
	})

	mage.Register("Feedback", func() mage.Card {
		return mage.NewAura("Feedback", "{2}{U}",
			mage.WithAbility(mage.BeginningOfAttachedControllerUpkeepTrigger(
				mage.DealDamageToPlayers(mage.Fixed(1), mage.SelectAttachedController()), false,
			)),
		)
	})

	mage.Register("Warp Artifact", func() mage.Card {
		return mage.NewAura("Warp Artifact", "{B}{B}",
			mage.WithAbility(mage.BeginningOfAttachedControllerUpkeepTrigger(
				mage.DealDamageToPlayers(mage.Fixed(1), mage.SelectAttachedController()), false,
			)),
		)
	})

	mage.Register("Karma", func() mage.Card {
		return mage.NewEnchantment("Karma", "{2}{W}{W}",
			mage.WithAbility(mage.BeginningOfEachUpkeepTrigger(mage.DealDamagePerSwamp(), false)),
		)
	})

	mage.Register("Farmstead", func() mage.Card {
		return mage.NewAura("Farmstead", "{1}{W}{W}",
			mage.WithAbility(mage.BeginningOfUpkeepTrigger(mage.GainLife(1), false)),
		)
	})

	// ===== LUCKY CHARMS =====

	mage.Register("Crystal Rod", func() mage.Card {
		return mage.NewLuckyCharm("Crystal Rod", "{1}", core.Blue)
	})

	mage.Register("Iron Star", func() mage.Card {
		return mage.NewLuckyCharm("Iron Star", "{1}", core.Red)
	})

	mage.Register("Ivory Cup", func() mage.Card {
		return mage.NewLuckyCharm("Ivory Cup", "{1}", core.White)
	})

	mage.Register("Throne of Bone", func() mage.Card {
		return mage.NewLuckyCharm("Throne of Bone", "{1}", core.Black)
	})

	mage.Register("Wooden Sphere", func() mage.Card {
		return mage.NewLuckyCharm("Wooden Sphere", "{1}", core.Green)
	})

	mage.Register("Soul Net", func() mage.Card {
		return mage.NewArtifact("Soul Net", "{1}",
			mage.WithAbility(mage.AnyCreatureDiesTrigger(mage.GainLife(1), true)),
		)
	})

	// ===== LACE CYCLE =====

	laces := []struct {
		name  string
		cost  string
		color core.Color
	}{
		{"Chaoslace", "{R}", core.Red},
		{"Deathlace", "{B}", core.Black},
		{"Lifelace", "{G}", core.Green},
		{"Purelace", "{W}", core.White},
		{"Thoughtlace", "{U}", core.Blue},
	}
	for _, lace := range laces {
		name := lace.name
		color := lace.color
		cost := lace.cost
		mage.Register(name, func() mage.Card {
			return mage.NewInstant(name, cost,
				mage.WithAbility(mage.NewTargetedSpell(mage.TargetPermanent(), mage.ChangeColorEffect(color))),
			)
		})
	}
}
