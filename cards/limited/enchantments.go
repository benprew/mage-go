package limited

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/dsl"
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
		return NewAura("Holy Armor", "{W}",
			WithStaticAbility(
				BoostAttached(0, 2, AttachAura),
				GrantActivatedAbilityToAttached(
					Boost(Fixed(0), Fixed(1)).Targeting(ToSource()),
					ManaCostOf("{W}"),
					AttachAura,
				),
			),
		)
	})

	Register("Blessing", func() Card {
		return NewAura("Blessing", "{W}{W}",
			WithStaticAbility(
				GrantActivatedAbilityToAttached(
					Boost(Fixed(1), Fixed(1)).Targeting(ToSource()),
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
					Boost(Fixed(1), Fixed(0)).Targeting(ToSource()),
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
			// At the beginning of enchanted creature's controller's upkeep,
			// that player may pay {4}. If the player does, untap the creature.
			// TODO: convert to pipeline — needs UntapAttached + TryPayMana primitives
			WithAbility(BeginningOfAttachedControllerUpkeepTrigger(FuncEffect(
				"pay {4} to untap enchanted creature",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					aura := g.FindPermanent(sourceID)
					if aura == nil {
						return nil
					}
					attached := g.MutablePermanent(aura.AttachedTo)
					if attached == nil {
						return nil
					}
					if g.TryPayCostFromLands(attached.Controller, "{4}") {
						attached.Tapped = false
					}
					return nil
				}), false)),
		)
	})

	Register("Earthbind", func() Card {
		return NewAura("Earthbind", "{R}",
			WithAbility(EntersBattlefieldTrigger(FuncEffect(
				"if enchanted creature has flying, deal 2 damage",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					aura := g.FindPermanent(sourceID)
					if aura == nil {
						return nil
					}
					attached := g.FindPermanent(aura.AttachedTo)
					// Damage is dealt before flying is removed (CR 603.4): check
					// the creature's flying ignoring Earthbind's own loses-flying
					// static ability, which has already stripped it by resolution.
					if attached != nil && g.HasKeywordIgnoringSource(attached.ID(), Flying, sourceID) {
						g.DealDamageToPermanent(attached, 2, sourceID)
					}
					return nil
				}), false)),
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

	Register("Animate Dead", func() Card {
		// XXX: missing leave-battlefield sacrifice trigger — engine needs last-known-information for attachments
		// XXX: Oracle says "creature card in a graveyard" (any graveyard); restricted to controller's graveyard for now.
		return NewAura("Animate Dead", "{1}{B}",
			WithCastTarget(TargetCreatureInYourGraveyard()),
			WithAbility(NewSpellAbility(ReturnFromGraveyardToBattlefield())),
			WithStaticAbility(
				BoostAttached(-1, 0, AttachAura),
			),
		)
	})

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
			WithAbility(EntersBattlefieldTrigger(
				ChooseColorStep("Choose basic land type for Phantasmal Terrain"),
				false)),
			WithStaticAbility(
				ChangeAttachedSubTypesByChosenColor(),
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
		{"Circle of Protection: Black", Black},
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
					AddColorPreventionStep(color),
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
		return NewAura("Farmstead", "{W}{W}{W}",
			WithCastTarget(TargetLand()),
			WithAbility(BeginningOfUpkeepTrigger(
				EffectIfPaid(ManaCostOf("{W}{W}"), GainLife(1)),
				false)),
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

	// ===== MOVED FROM OTHER FILES =====

	Register("Steal Artifact", func() Card {
		return NewAura("Steal Artifact", "{2}{U}{U}",
			WithCastTarget(TargetArtifact()),
			WithStaticAbility(ControlChangeContinuous()),
		)
	})

	Register("Copy Artifact", func() Card {
		return NewArtifact("Copy Artifact", "{1}{U}",
			WithAbility(NewTargetedSpell(TargetArtifact(), CloneTarget(TypeEnchantment))),
		)
	})

	Register("Deathgrip", func() Card {
		return NewEnchantment("Deathgrip", "{B}{B}",
			WithActivatedAbility(
				CounterSpellIfColor(Green),
				ManaCostOf("{B}{B}"),
				WithTarget(TargetSpellOnStack()),
			),
		)
	})

	Register("Lifeforce", func() Card {
		return NewEnchantment("Lifeforce", "{G}{G}",
			WithActivatedAbility(
				CounterSpellIfColor(Black),
				ManaCostOf("{G}{G}"),
				WithTarget(TargetSpellOnStack()),
			),
		)
	})

	Register("Living Lands", func() Card {
		return NewEnchantment("Living Lands", "{3}{G}",
			WithStaticAbility(
				AnimateLands(And(IsLand, HasSubType("Forest")), 1, 1),
			),
		)
	})

	// Instill Energy
	// Enchant creature
	// Enchanted creature can attack as though it had haste.
	// {0}: Untap enchanted creature. Activate only once each turn.
	Register("Instill Energy", func() Card {
		return NewAura("Instill Energy", "{G}",
			WithStaticAbility(
				GrantAbilityToAttached(Haste, AttachAura),
				GrantActivatedAbilityToAttached(
					UntapSource(),
					GenericCost(0),
					AttachAura,
					WithOncePerTurn(),
				),
			),
		)
	})

	Register("Mana Flare", func() Card {
		return NewEnchantment("Mana Flare", "{2}{R}",
			WithAbility(NewManaFlareAbility(IsLand)),
		)
	})

	Register("Raging River", func() Card {
		return NewEnchantment("Raging River", "{R}{R}",
			// TODO: convert to pipeline — needs PreventBlockingUntilEndOfCombat step
			WithAbility(NewTriggered(EvtDeclaredAttacker, false, FuncEffect(
				"split blockers into piles",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					var nonFlyers []*Permanent
					for _, p := range g.FilterBattlefield(AnyPermanent) {
						if p.Controller != controller && p.HasType(TypeCreature) &&
							!p.HasKeyword(Flying) {
							nonFlyers = append(nonFlyers, p)
						}
					}
					for _, p := range nonFlyers {
						eff := PreventBlockingUntilEndOfCombat(p.ID())
						eff.SetSourceID(sourceID)
						g.AddContinuousEffect(eff)
					}
					g.ApplyContinuousEffects()
					return nil
				})).
				SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
					EventPlayerIsController{},
					SourceOnBattlefield{},
					CombatGroupCountEquals{N: 1},
				}})),
		)
	})

	Register("Lich", func() Card {
		return NewEnchantment("Lich", "{B}{B}{B}{B}",
			// ETB: lose life equal to your life total
			// TODO: convert to pipeline — needs Lich-specific game rule steps
			WithAbility(EntersBattlefieldTrigger(FuncEffect(
				"lose life equal to your life total",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					p := g.GetPlayer(controller)
					if p == nil {
						return nil
					}
					life := p.Life()
					if life > 0 {
						p.LoseLife(life)
					}
					g.SetLichActive(controller, sourceID)
					return nil
				}), false)),
			// When Lich is put into a graveyard from the battlefield, you lose the game.
			// TODO: convert to pipeline — needs Lich-specific game rule steps
			WithAbility(PutIntoGraveyardFromBattlefieldTrigger(FuncEffect(
				"you lose the game",
				EffectProperties{},
				func(g *Game, _, controller uuid.UUID, _ []uuid.UUID) error {
					g.ClearLich(controller)
					p := g.GetPlayer(controller)
					if p != nil {
						p.LoseLife(9999)
					}
					return nil
				}), false)),
		)
	})

	Register("Island Sanctuary", func() Card {
		// Oracle: "If you would draw a card during your draw step, instead you
		// may skip that draw. If you do, until your next turn, you can't be
		// attacked except by creatures with flying or islandwalk."
		// Implemented as a draw replacement registered on ETB: when the
		// controller's draw-step draw fires, the replacement asks them whether
		// to skip; accepting both replaces the draw with nothing and arms the
		// sanctuary attack-restriction until their next turn.
		return NewEnchantment("Island Sanctuary", "{1}{W}",
			WithAbility(EntersBattlefieldTrigger(FuncEffect(
				"register Island Sanctuary draw replacement",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					g.AddIslandSanctuaryReplacement(controller, sourceID)
					return nil
				}), false)),
		)
	})

	Register("Power Surge", func() Card {
		return NewEnchantment("Power Surge", "{R}{R}",
			WithAbility(BeginningOfEachUpkeepTrigger(
				DealDamageToPlayers(
					UntappedLandsAtTurnStart(SelectActivePlayer()),
					SelectActivePlayer(),
				), false)),
		)
	})

	Register("Lifetap", func() Card {
		return NewEnchantment("Lifetap", "{U}{U}",
			WithAbility(WhenOpponentPermanentBecomesTappedTrigger(
				GainLife(1), false,
				And(IsLand, HasSubType("Forest")),
			)),
		)
	})

	Register("Conversion", func() Card {
		return NewEnchantment("Conversion", "{2}{W}{W}",
			WithStaticAbility(
				ChangeSubTypesForAll([]string{"Mountain"}, []string{"Plains"}),
			),
			WithAbility(SacrificeAtUpkeepUnlessPay("{W}{W}")),
		)
	})

	Register("Gloom", func() Card {
		return NewEnchantment("Gloom", "{2}{B}",
			WithStaticAbility(
				IncreaseSpellCostForColor(White, 3),
			),
			// XXX: missing "Activated abilities of white enchantments cost {3} more to activate" — no engine support for increasing activated ability costs
		)
	})

	Register("Magnetic Mountain", func() Card {
		return NewEnchantment("Magnetic Mountain", "{1}{R}{R}",
			WithStaticAbility(
				PreventUntapForMatching(And(IsCreature, HasColorFilter(Blue))),
			),
			// TODO: convert to pipeline — needs ForEach + TryPayMana + untap per creature
			WithAbility(BeginningOfEachUpkeepTrigger(
				FuncEffect("pay {4} to untap blue creatures",
					EffectProperties{},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						// Find the active player (whose upkeep it is)
						activePlayer := g.ActivePlayerObj().PlayerID()
						blues := g.FilterBattlefield(And(IsCreature, HasColorFilter(Blue), ControlledBy(activePlayer), IsTapped))
						for _, blue := range blues {
							if g.TryPayCostFromLands(activePlayer, "{4}") {
								blue = g.MutablePermanent(blue.ID())
								if blue == nil {
									continue
								}
								blue.Tapped = false
							}
						}
						return nil
					}), false,
			)),
		)
	})

	// TODO implement
	//   "Text": "Enchant land\nEnchanted land has indestructible and can't be enchanted by other Auras.",
	Register("Consecrate Land", func() Card {
		return NewAura("Consecrate Land", "{W}",
			WithCastTarget(TargetLand()),
			WithStaticAbility(
				GrantAbilityToAttached(Indestructible, AttachAura),
			),
		)
	})

	// TODO implement
	// Fastbond's effects only apply to caster, not all players
	Register("Fastbond", func() Card {
		return NewEnchantment("Fastbond", "{G}",
			WithStaticAbility(AllowUnlimitedLandPlays()),
			WithAbility(NewTriggered(EvtLandPlayed, false,
				DealDamageToPlayers(Fixed(1), SelectController()),
			).
				SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{EventPlayerIsController{}, EventAmountGreaterThan{N: 1}}})),
		)
	})

	Register("Kudzu", func() Card {
		return NewAura("Kudzu", "{1}{G}{G}",
			WithCastTarget(TargetLand()),
			// TODO: convert to pipeline — complex attachment manipulation
			WithAbility(WhenAttachedBecomesTappedTrigger(FuncEffect(
				"destroy enchanted land; its controller picks another land to attach Kudzu to",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					kudzu := g.MutablePermanent(sourceID)
					if kudzu == nil {
						return nil
					}
					attached := g.MutablePermanent(kudzu.AttachedTo)
					if attached == nil {
						return nil
					}
					attachedID := attached.ID()
					attachedController := attached.Controller
					kudzu.AttachedTo = uuid.Nil
					filtered := attached.Attachments[:0]
					for _, id := range attached.Attachments {
						if id != sourceID {
							filtered = append(filtered, id)
						}
					}
					attached.Attachments = filtered
					g.DestroyPermanent(attached)
					var candidates []*Permanent
					for _, p := range g.FilterBattlefield(AnyPermanent) {
						if p.HasType(TypeLand) && p.ID() != attachedID {
							candidates = append(candidates, p)
						}
					}
					if len(candidates) == 0 {
						return nil
					}
					chooser := g.GetPlayer(attachedController)
					if chooser == nil {
						return nil
					}
					chosen := chooser.ChoosePermanent(candidates, "attach Kudzu to a land", g)
					if chosen == nil {
						return nil
					}
					g.Attach(sourceID, chosen.ID())
					return nil
				}), false)),
		)
	})

	Register("Regeneration", func() Card {
		return NewAura("Regeneration", "{1}{G}",
			WithStaticAbility(
				GrantActivatedAbilityToAttached(
					RegenerateSource(),
					ManaCostOf("{G}"),
					AttachAura,
				),
			),
		)
	})

	// Smoke {R}{R}
	// Enchantment
	// Players can't untap more than one creature during their untap steps.
	Register("Smoke", func() Card {
		return NewEnchantment("Smoke", "{R}{R}",
			WithStaticAbility(LimitCreatureUntaps(1)),
		)
	})

	// Manabarbs {3}{R}
	// Enchantment
	// Whenever a player taps a land for mana, Manabarbs deals 1 damage to that player.
	Register("Manabarbs", func() Card {
		return NewEnchantment("Manabarbs", "{3}{R}",
			WithAbility(NewTriggered(EvtTapped, false,
				DealDamageToPlayers(Fixed(1), SelectTargetPermanentController()),
			).SetConditionData(EventSourceHasType{Type: TypeLand})),
		)
	})

	// Animate Wall {W}
	// Enchantment — Aura
	// Enchant Wall
	// Enchanted Wall can attack as though it didn't have defender.
	Register("Animate Wall", func() Card {
		return NewAura("Animate Wall", "{W}",
			WithCastTarget(TargetCreature(HasSubType("Wall"))),
			WithStaticAbility(
				GrantAbilityToAttached(AttrCanAttack, AttachAura),
			),
		)
	})

	Register("Aspect of Wolf", func() Card {
		return NewAura("Aspect of Wolf", "{1}{G}",
			// Enchanted creature gets +X/+Y where X is half Forests you control
			// (rounded down) and Y is half (rounded up).
			WithStaticAbility(
				BoostAttachedByCount(
					And(IsLand, HasSubType("Forest")),
					func(n int) int { return n / 2 },
					func(n int) int { return (n + 1) / 2 },
				),
			),
		)
	})

	Register("Stasis", func() Card {
		return NewEnchantment("Stasis", "{1}{U}",
			// Players skip their untap steps.
			WithStaticAbility(PreventAllUntaps()),
			// At the beginning of your upkeep, sacrifice Stasis unless you pay {U}.
			WithAbility(SacrificeAtUpkeepUnlessPay("{U}")),
		)
	})

	Register("Pestilence", func() Card {
		return NewEnchantment("Pestilence", "{2}{B}{B}",
			// At the beginning of the end step, if no creatures are on the battlefield, sacrifice Pestilence.
			WithAbility(BeginningOfEachEndStepTrigger(
				IfElse("sacrifice if no creatures",
					NoBattlefieldPermanentMatching{Filter: IsCreature},
					SacrificeSourceStep(),
					nil,
				), false,
			)),
			// {B}: Deal 1 damage to each creature and each player
			WithActivatedAbility(
				CompositeEffects("deal 1 damage to each creature and each player",
					DealDamageToAllCreatures(Fixed(1), PermanentFilter{}),
					DealDamageToPlayers(Fixed(1), SelectEachPlayer()),
				),
				ManaCostOf("{B}"),
			),
		)
	})

}
