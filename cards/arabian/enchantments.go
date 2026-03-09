package arabian

import (
	"github.com/google/uuid"
	. "github.com/mage/mage/pkg/mage"
	. "github.com/mage/mage/pkg/mage/core"
)

func init() {
	registerEnchantments()
}

func registerEnchantments() {
	// ===== GLOBAL ENCHANTMENTS =====

	// Oracle: "At the beginning of your upkeep, put a wind counter on Cyclone, then
	// sacrifice Cyclone unless you pay {G} for each wind counter on it. If you pay,
	// Cyclone deals damage equal to the number of wind counters on it to each creature
	// and each player."
	Register("Cyclone", func() Card {
		return NewEnchantment("Cyclone", "{2}{G}{G}",
			WithAbility(
				BeginningOfUpkeepTrigger(
					FuncEffect("add wind counter, pay or sacrifice, deal damage",
						EffectProperties{Outcome: OutcomeDetriment},
						func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							perm := g.FindPermanent(sourceID)
							if perm == nil {
								return nil
							}
							perm.AddCounter(Wind, 1)
							count := perm.Counters[Wind]
							// Build the mana cost: {G} per wind counter
							cost := ""
							for range count {
								cost += "{G}"
							}
							if g.TryPayCostFromLands(controller, cost) {
								// Deal damage equal to wind counters to each creature and player
								for _, c := range g.FilterBattlefield(IsCreature) {
									g.DealDamageToPermanent(c, count, sourceID)
								}
								for _, p := range g.AllPlayers() {
									g.DealDamageToPlayer(p, count, sourceID)
								}
							} else {
								g.Sacrifice(perm)
							}
							return nil
						}), false,
				),
			),
		)
	})

	// Oracle: "At the beginning of your upkeep, destroy the creature with the least power.
	// It can't be regenerated. If two or more creatures are tied for least power, you choose
	// one of them. When there are no creatures on the battlefield, sacrifice Drop of Honey."
	Register("Drop of Honey", func() Card {
		return NewEnchantment("Drop of Honey", "{G}",
			WithAbility(
				BeginningOfUpkeepTrigger(
					FuncEffect("destroy least power creature or sacrifice self",
						EffectProperties{Outcome: OutcomeDetriment},
						func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							creatures := g.FilterBattlefield(IsCreature)
							if len(creatures) == 0 {
								src := g.FindPermanent(sourceID)
								if src != nil {
									g.Sacrifice(src)
								}
								return nil
							}
							// Find minimum power
							minPower := creatures[0].CurrentPower(g)
							for _, c := range creatures[1:] {
								pw := c.CurrentPower(g)
								if pw < minPower {
									minPower = pw
								}
							}
							// Collect tied creatures
							var tied []*Permanent
							for _, c := range creatures {
								if c.CurrentPower(g) == minPower {
									tied = append(tied, c)
								}
							}
							var target *Permanent
							if len(tied) == 1 {
								target = tied[0]
							} else {
								p := g.GetPlayer(controller)
								if p != nil {
									target = p.ChoosePermanent(tied, "destroy", g)
								}
							}
							if target != nil {
								// Can't be regenerated
								target.GrantBaseAttr(CantRegenerate)
								g.DestroyPermanent(target)
							}
							return nil
						}), false,
				),
			),
			// "When there are no creatures on the battlefield, sacrifice Drop of Honey."
			// State trigger: fires when a creature leaves and no creatures remain
			WithAbility(NewTriggered(EvtLeavesBattlefield, false, SacrificeSource()).
				SetCondition(func(evt *GameEvent, g *Game, sourceID, controllerID uuid.UUID) bool {
					for _, p := range g.FilterBattlefield(AnyPermanent) {
						if p.HasType(TypeCreature) {
							return false
						}
					}
					return true
				})),
		)
	})

	// Oracle: "As Jihad enters, choose a color and an opponent. White creatures get +2/+1
	// as long as the chosen player controls a nontoken permanent of the chosen color.
	// When the chosen player controls no nontoken permanents of the chosen color,
	// sacrifice Jihad."
	Register("Jihad", func() Card {
		return NewEnchantment("Jihad", "{W}{W}{W}",
			WithAbility(
				EntersBattlefieldTrigger(
					FuncEffect("choose color and opponent",
						EffectProperties{},
						func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							perm := g.FindPermanent(sourceID)
							if perm == nil {
								return nil
							}
							p := g.GetPlayer(controller)
							if p == nil {
								return nil
							}
							perm.ChosenColor = p.ChooseManaColor("Jihad: choose a color")
							opp := g.GetOpponent(controller)
							if opp != nil {
								perm.ChosenPlayer = opp.PlayerID()
							}
							return nil
						}), false,
				),
			),
			// Boost white creatures while chosen player controls a nontoken permanent of chosen color
			WithStaticAbility(
				FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					perm := g.FindPermanent(sourceID)
					if perm == nil {
						return nil
					}
					chosenColor := perm.ChosenColor
					chosenPlayer := perm.ChosenPlayer
					if chosenColor == 0 || chosenPlayer == uuid.Nil {
						return nil
					}
					// Check if chosen player controls a nontoken permanent of chosen color
					hasNontoken := false
					for _, p := range g.Battlefield {
						if p.Controller == chosenPlayer &&
							HasColorFilter(chosenColor).Match(p, g) &&
							!p.Card.(*BaseCard).IsToken() {
							hasNontoken = true
							break
						}
					}
					if !hasNontoken {
						return nil // condition not met, no boost (sacrifice handled by trigger below)
					}
					// Boost all white creatures +2/+1
					for _, p := range g.Battlefield {
						if p.HasType(TypeCreature) && HasColorFilter(White).Match(p, g) {
							p.BoostPT(2, 1)
						}
					}
					return nil
				}),
			),
			// When the chosen player controls no nontoken permanents of the chosen color, sacrifice Jihad
			WithAbility(
				NewTriggered(EvtLeavesBattlefield, false,
					SacrificeSource(),
				).SetCondition(func(evt *GameEvent, g *Game, sourceID, controllerID uuid.UUID) bool {
					perm := g.FindPermanent(sourceID)
					if perm == nil {
						return false
					}
					chosenColor := perm.ChosenColor
					chosenPlayer := perm.ChosenPlayer
					if chosenColor == 0 || chosenPlayer == uuid.Nil {
						return false
					}
					for _, p := range g.Battlefield {
						if p.Controller == chosenPlayer &&
							HasColorFilter(chosenColor).Match(p, g) &&
							!p.Card.(*BaseCard).IsToken() {
							return false
						}
					}
					return true
				}),
			),
		)
	})

	// Oracle: "When Oubliette enters, target creature phases out until Oubliette leaves
	// the battlefield. Tap that creature as it phases in this way."
	Register("Oubliette", func() Card {
		return NewEnchantment("Oubliette", "{1}{B}{B}",
			WithETBEffect(FuncEffect("phase out target creature until Oubliette leaves",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					src := g.FindPermanent(sourceID)
					target := g.FindPermanent(targets[0])
					if src == nil || target == nil {
						return nil
					}
					targetID := target.ID()
					// Phase out the target creature (retains counters, auras, etc.)
					target.PhasedOut = true
					src.ControlledPermanent = targetID
					// Register delayed trigger: when Oubliette leaves, phase creature back in
					g.RegisterDelayedTrigger(&DelayedTrigger{
						EventType:    EvtLeavesBattlefield,
						MatchEventID: sourceID,
						SourceID:     sourceID,
						Controller:   controller,
						Effects: []Effect{FuncEffect("phase in creature",
							EffectProperties{Outcome: OutcomeBenefit},
							func(g GameMutator, _, _ uuid.UUID, _ []uuid.UUID) error {
								perm := g.(*Game).FindPermanentIncludingPhased(targetID)
								if perm != nil && perm.PhasedOut {
									perm.PhasedOut = false
									perm.Tapped = true
								}
								return nil
							})},
					})
					return nil
				})),
		)
	})

	// ===== AURAS =====

	// Oracle: "Enchant creature. Enchanted creature has islandwalk."
	Register("Fishliver Oil", func() Card {
		return NewAura("Fishliver Oil", "{1}{U}",
			WithAbility(StaticAbility(GrantAbilityToAttached(Islandwalk, AttachAura))),
		)
	})

	// Oracle: "Enchant creature. Enchanted creature gets +3/+3. At the beginning of the
	// upkeep of enchanted creature's controller, put a -1/-1 counter on that creature."
	Register("Unstable Mutation", func() Card {
		return NewAura("Unstable Mutation", "{U}",
			WithAbility(StaticAbility(BoostAttached(3, 3, AttachAura))),
			WithAbility(BeginningOfAttachedControllerUpkeepTrigger(
				FuncEffect("put a -1/-1 counter on enchanted creature",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						src := g.FindPermanent(sourceID)
						if src == nil || src.AttachedTo == uuid.Nil {
							return nil
						}
						target := g.FindPermanent(src.AttachedTo)
						if target != nil {
							target.AddCounter(M1M1, 1)
						}
						return nil
					}), false,
			)),
		)
	})
}
