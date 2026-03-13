package limited

import (
	"github.com/google/uuid"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

func init() {
	registerEnchantments()
}

func registerEnchantments() {
	// ===== GLOBAL ENCHANTMENTS =====

	Register("Crusade", func() Card {
		return NewEnchantment("Crusade", "{W}{W}",
			WithStaticAbility(
				BoostAllCreaturesIncludingSelf(1, 1, HasColorFilter(White)),
			),
		)
	})

	Register("Bad Moon", func() Card {
		return NewEnchantment("Bad Moon", "{1}{B}",
			WithStaticAbility(
				BoostAllCreaturesIncludingSelf(1, 1, HasColorFilter(Black)),
			),
		)
	})

	Register("Orcish Oriflamme", func() Card {
		return NewEnchantment("Orcish Oriflamme", "{3}{R}",
			WithStaticAbility(
				BoostControlledCreatures(1, 0, IsAttacking),
			),
		)
	})

	Register("Castle", func() Card {
		return NewEnchantment("Castle", "{3}{W}",
			WithStaticAbility(
				BoostControlledCreatures(0, 2, IsUntapped),
			),
		)
	})

	// ===== AURAS (CREATURE ENCHANTMENTS) =====

	Register("Holy Strength", func() Card {
		return NewBoostAura("Holy Strength", "{W}", 1, 2)
	})

	Register("Unholy Strength", func() Card {
		return NewBoostAura("Unholy Strength", "{B}", 2, 1)
	})

	Register("Weakness", func() Card {
		return NewBoostAura("Weakness", "{B}", -2, -1)
	})

	Register("Holy Armor", func() Card {
		return NewBoostAura("Holy Armor", "{W}", 0, 2)
	})

	Register("Blessing", func() Card {
		return NewAura("Blessing", "{W}{W}",
			WithStaticAbility(
				GrantActivatedAbilityToAttached(
					BoostUntilEndOfTurn(Fixed(1), Fixed(1), SelectSource),
					ManaCostOf("{W}"),
					AttachAura,
				),
			),
		)
	})

	Register("Lance", func() Card {
		return NewAura("Lance", "{W}",
			WithStaticAbility(
				GrantAbilityToAttached(FirstStrike, AttachAura),
			),
		)
	})

	Register("Web", func() Card {
		return NewAura("Web", "{G}",
			WithStaticAbility(
				BoostAttached(0, 2, AttachAura),
				GrantAbilityToAttached(Reach, AttachAura),
			),
		)
	})

	Register("Firebreathing", func() Card {
		return NewAura("Firebreathing", "{R}",
			WithStaticAbility(
				GrantActivatedAbilityToAttached(
					BoostUntilEndOfTurn(Fixed(1), Fixed(0), SelectSource),
					ManaCostOf("{R}"),
					AttachAura,
				),
			),
		)
	})

	Register("Flight", func() Card {
		return NewAura("Flight", "{U}",
			WithStaticAbility(
				GrantAbilityToAttached(Flying, AttachAura),
			),
		)
	})

	Register("Jump", func() Card {
		return NewInstant("Jump", "{U}",
			NewTargetedSpell(TargetCreature(), GrantKeywordUntilEndOfTurn(Flying, SelectTarget)),
		)
	})

	Register("Fear", func() Card {
		return NewAura("Fear", "{B}{B}",
			WithStaticAbility(
				GrantAbilityToAttached(Fear, AttachAura),
			),
		)
	})

	Register("Burrowing", func() Card {
		return NewAura("Burrowing", "{R}",
			WithStaticAbility(
				GrantAbilityToAttached(Mountainwalk, AttachAura),
			),
		)
	})

	Register("Invisibility", func() Card {
		return NewAura("Invisibility", "{U}{U}",
			WithStaticAbility(
				GrantAbilityToAttached(CantBeBlockedExceptByWalls, AttachAura),
			),
		)
	})

	Register("Lure", func() Card {
		return NewAura("Lure", "{1}{G}{G}",
			WithStaticAbility(
				GrantAbilityToAttached(MustBeBlocked, AttachAura),
			),
		)
	})

	Register("Paralyze", func() Card {
		return NewAura("Paralyze", "{B}",
			WithAbility(EntersBattlefieldTrigger(TapAttachedCreature(), false)),
			WithStaticAbility(
				PreventAttachedFromUntapping(AttachAura),
			),
		)
	})

	Register("Earthbind", func() Card {
		return NewAura("Earthbind", "{R}",
			WithStaticAbility(
				RemoveKeywordFromAttached(Flying, AttachAura),
			),
		)
	})

	Register("Control Magic", func() Card {
		return NewAura("Control Magic", "{2}{U}{U}",
			WithStaticAbility(ControlChangeContinuous()),
		)
	})

	// Animate Dead is registered in spells.go

	// ===== ENCHANT LAND =====

	Register("Wild Growth", func() Card {
		return NewAura("Wild Growth", "{G}",
			WithCastTarget(TargetLand()),
			WithAbility(NewAttachedManaBonusAbility(Green)),
		)
	})

	Register("Evil Presence", func() Card {
		return NewAura("Evil Presence", "{B}",
			WithCastTarget(TargetLand()),
			WithStaticAbility(
				ChangeAttachedSubTypes([]string{"Swamp"}),
			),
		)
	})

	Register("Phantasmal Terrain", func() Card {
		return NewAura("Phantasmal Terrain", "{U}{U}",
			WithCastTarget(TargetLand()),
			WithStaticAbility(
				ChangeAttachedSubTypes([]string{"Island"}),
			),
		)
	})

	Register("Psychic Venom", func() Card {
		return NewAura("Psychic Venom", "{1}{U}",
			WithCastTarget(TargetLand()),
			WithAbility(WhenAttachedBecomesTappedTrigger(
				DealDamageToPlayers(Fixed(2), SelectAttachedController()), false,
			)),
		)
	})

	// ===== WARD/PROTECTION ENCHANTMENTS =====

	copColors := []struct {
		name  string
		color Color
	}{
		{"Circle of Protection: Blue", Blue},
		{"Circle of Protection: Green", Green},
		{"Circle of Protection: Red", Red},
		{"Circle of Protection: White", White},
	}
	for _, cop := range copColors {
		name := cop.name
		color := cop.color
		Register(name, func() Card {
			return NewEnchantment(name, "{1}{W}",
				// {1}: Prevent all damage from one source of this color this turn.
				WithActivatedAbility(
					FuncEffect(
						"prevent all damage from one source of the chosen color",
						EffectProperties{},
						func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							g.AddColorPrevention(controller, color)
							return nil
						}),
					GenericCost(1),
				),
			)
		})
	}

	Register("Black Ward", func() Card {
		return NewAura("Black Ward", "{W}",
			WithStaticAbility(GrantProtectionToAttached(Black, AttachAura)),
		)
	})

	Register("Blue Ward", func() Card {
		return NewAura("Blue Ward", "{W}",
			WithStaticAbility(GrantProtectionToAttached(Blue, AttachAura)),
		)
	})

	Register("Green Ward", func() Card {
		return NewAura("Green Ward", "{W}",
			WithStaticAbility(GrantProtectionToAttached(Green, AttachAura)),
		)
	})

	Register("Red Ward", func() Card {
		return NewAura("Red Ward", "{W}",
			WithStaticAbility(GrantProtectionToAttached(Red, AttachAura)),
		)
	})

	Register("White Ward", func() Card {
		return NewAura("White Ward", "{W}",
			WithStaticAbility(GrantProtectionToAttached(White, AttachAura)),
		)
	})

	// ===== TRIGGERED ENCHANTMENTS =====

	Register("Copper Tablet", func() Card {
		return NewArtifact("Copper Tablet", "{2}",
			WithAbility(BeginningOfEachUpkeepTrigger(DealDamageToPlayers(Fixed(1), SelectActivePlayer()), false)),
		)
	})

	Register("Black Vise", func() Card {
		return NewArtifact("Black Vise", "{1}",
			WithAbility(BeginningOfEachUpkeepTrigger(BlackViseEffect(), false)),
		)
	})

	Register("Wanderlust", func() Card {
		return NewAura("Wanderlust", "{2}{G}",
			WithAbility(BeginningOfAttachedControllerUpkeepTrigger(
				DealDamageToPlayers(Fixed(1), SelectAttachedController()), false,
			)),
		)
	})

	Register("Cursed Land", func() Card {
		return NewAura("Cursed Land", "{2}{B}{B}",
			WithCastTarget(TargetLand()),
			WithAbility(BeginningOfAttachedControllerUpkeepTrigger(
				DealDamageToPlayers(Fixed(1), SelectAttachedController()), false,
			)),
		)
	})

	Register("Feedback", func() Card {
		return NewAura("Feedback", "{2}{U}",
			WithCastTarget(TargetPermanent(IsEnchantment)),
			WithAbility(BeginningOfAttachedControllerUpkeepTrigger(
				DealDamageToPlayers(Fixed(1), SelectAttachedController()), false,
			)),
		)
	})

	Register("Warp Artifact", func() Card {
		return NewAura("Warp Artifact", "{B}{B}",
			WithCastTarget(TargetArtifact()),
			WithAbility(BeginningOfAttachedControllerUpkeepTrigger(
				DealDamageToPlayers(Fixed(1), SelectAttachedController()), false,
			)),
		)
	})

	Register("Karma", func() Card {
		return NewEnchantment("Karma", "{2}{W}{W}",
			WithAbility(BeginningOfEachUpkeepTrigger(
				DealDamageToPlayers(
					CountBattlefield(SelectActivePlayer(), HasSubType("Swamp")),
					SelectActivePlayer(),
				), false)),
		)
	})

	Register("Farmstead", func() Card {
		return NewAura("Farmstead", "{1}{W}{W}",
			WithAbility(BeginningOfUpkeepTrigger(GainLife(1), false)),
		)
	})

	// ===== LUCKY CHARMS =====

	Register("Crystal Rod", func() Card {
		return NewLuckyCharm("Crystal Rod", "{1}", Blue)
	})

	Register("Iron Star", func() Card {
		return NewLuckyCharm("Iron Star", "{1}", Red)
	})

	Register("Ivory Cup", func() Card {
		return NewLuckyCharm("Ivory Cup", "{1}", White)
	})

	Register("Throne of Bone", func() Card {
		return NewLuckyCharm("Throne of Bone", "{1}", Black)
	})

	Register("Wooden Sphere", func() Card {
		return NewLuckyCharm("Wooden Sphere", "{1}", Green)
	})

	Register("Soul Net", func() Card {
		return NewArtifact("Soul Net", "{1}",
			WithAbility(AnyCreatureDiesTrigger(GainLife(1), true)),
		)
	})

	// ===== LACE CYCLE =====

	laces := []struct {
		name  string
		cost  string
		color Color
	}{
		{"Chaoslace", "{R}", Red},
		{"Deathlace", "{B}", Black},
		{"Lifelace", "{G}", Green},
		{"Purelace", "{W}", White},
		{"Thoughtlace", "{U}", Blue},
	}
	for _, lace := range laces {
		name := lace.name
		color := lace.color
		cost := lace.cost
		Register(name, func() Card {
			return NewInstant(name, cost,
				NewTargetedSpell(TargetPermanent(), ChangeColorEffect(color)),
			)
		})
	}
}
