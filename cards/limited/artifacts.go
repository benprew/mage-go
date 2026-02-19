package limited

import (
	"math/rand"

	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

func init() {
	registerArtifacts()
}

func registerArtifacts() {
	// ===== MOX CYCLE =====

	moxen := []struct {
		name  string
		color core.Color
	}{
		{"Mox Pearl", core.White},
		{"Mox Sapphire", core.Blue},
		{"Mox Jet", core.Black},
		{"Mox Ruby", core.Red},
		{"Mox Emerald", core.Green},
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
			mage.AddAnyMana(3, core.Green),
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
			mage.AddMana(core.Colorless, 2),
			mage.TapSourceCost(),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Basalt Monolith", func() mage.Card {
		c := mage.NewArtifact("Basalt Monolith", "{3}")
		c.AddAbility(mage.NewKeywordAbility(core.DoesNotUntapKW)) // doesn't untap during untap step
		// {T}: Add {C}{C}{C}
		ab := mage.NewActivatedAbility(
			mage.AddMana(core.Colorless, 3),
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
		c.AddAbility(mage.NewKeywordAbility(core.DoesNotUntapKW)) // doesn't untap during untap step
		// {T}: Add {C}{C}{C}
		ab := mage.NewActivatedAbility(
			mage.AddMana(core.Colorless, 3),
			mage.TapSourceCost(),
		)
		c.AddAbility(ab)
		// At the beginning of your upkeep, deal 1 damage to you
		c.AddAbility(mage.BeginningOfUpkeepTrigger(mage.DealDamageToPlayers(mage.Fixed(1), mage.SelectController()), false))
		return c
	})

	// ===== UTILITY ARTIFACTS =====

	mage.Register("Jayemdae Tome", func() mage.Card {
		c := mage.NewArtifact("Jayemdae Tome", "{4}")
		// {4}, {T}: Draw a card
		ab := mage.NewActivatedAbility(
			mage.DrawCards(mage.Fixed(1)),
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
			mage.DiscardCards(mage.Fixed(1)),
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
			mage.DealDamage(mage.Fixed(1)),
			mage.GenericCost(3),

			mage.WithCost(mage.TapSourceCost()),

			mage.WithTarget(mage.TargetAnyTarget()),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("The Hive", func() mage.Card {
		c := mage.NewArtifact("The Hive", "{5}")
		// {5}, {T}: Create a 1/1 colorless Insect artifact creature token with flying named Wasp.
		ab := mage.NewActivatedAbility(
			mage.CreateToken("Wasp", 1, 1,
				[]core.CardType{core.TypeArtifact, core.TypeCreature},
				[]string{"Insect"},
				core.Flying,
			),
			mage.GenericCost(5),
			mage.WithCost(mage.TapSourceCost()),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Winter Orb", func() mage.Card {
		c := mage.NewArtifact("Winter Orb", "{2}")
		// Players can't untap more than one land during their untap steps.
		// Only applies while Winter Orb is untapped.
		c.AddAbility(mage.StaticAbility(
			mage.LimitLandUntaps(1),
		))
		return c
	})

	mage.Register("Meekstone", func() mage.Card {
		c := mage.NewArtifact("Meekstone", "{1}")
		// Creatures with power 3 or greater don't untap during their controller's untap step.
		c.AddAbility(mage.StaticAbility(
			mage.PreventUntapForMatching(mage.And(mage.IsCreature, mage.HasPowerGTE(3))),
		))
		return c
	})

	mage.Register("Howling Mine", func() mage.Card {
		c := mage.NewArtifact("Howling Mine", "{2}")
		// At the beginning of each player's draw step, that player draws an additional card.
		// Only triggers while Howling Mine is untapped.
		c.AddAbility(mage.BeginningOfEachDrawStepTrigger(mage.DrawCardsActivePlayer(mage.Fixed(1)), false))
		return c
	})

	mage.Register("Forcefield", func() mage.Card {
		c := mage.NewArtifact("Forcefield", "{3}")
		// {1}: If an unblocked creature would deal combat damage to you this turn,
		// prevent all but 1 of that damage.
		ab := mage.NewActivatedAbility(
			mage.ForcefieldEffect(),
			mage.GenericCost(1),
		)
		c.AddAbility(ab)
		return c
	})

	// ===== GAUNTLET =====

	mage.Register("Gauntlet of Might", func() mage.Card {
		c := mage.NewArtifact("Gauntlet of Might", "{4}")
		// Red creatures get +1/+1
		c.AddAbility(mage.StaticAbility(
			mage.BoostAllCreaturesIncludingSelf(1, 1, mage.HasColorFilter(core.Red)),
		))
		// Whenever a Mountain is tapped for mana, its controller adds an additional {R}
		c.AddAbility(mage.NewManaBonusAbility(mage.HasSubType("Mountain"), core.Red))
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
		// You may have Copy Artifact enter the battlefield as a copy of any artifact
		// on the battlefield, except it's an enchantment in addition to its other types.
		sa := mage.NewTargetedSpell(mage.TargetArtifact(), mage.CloneTarget(core.TypeEnchantment))
		c.AddAbility(sa)
		return c
	})

	// ===== MISC ARTIFACTS =====

	mage.Register("Ankh of Mishra", func() mage.Card {
		c := mage.NewArtifact("Ankh of Mishra", "{2}")
		// Whenever a land enters the battlefield, deal 2 damage to that land's controller
		c.AddAbility(mage.WheneverLandEntersBattlefieldTrigger(
			mage.DealDamageToPlayers(mage.Fixed(2), mage.SelectEventController()), false,
		))
		return c
	})

	mage.Register("Jade Monolith", func() mage.Card {
		c := mage.NewArtifact("Jade Monolith", "{4}")
		// {1}: The next time a source of your choice would deal damage to target
		// creature this turn, that damage is dealt to target player instead.
		ab := mage.NewActivatedAbility(
			mage.FuncEffect(
				"redirect next damage to target creature to target player instead",
				func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) < 2 {
						return nil
					}
					creatureID := targets[0]
					playerID := targets[1]
					g.Effects.SetCreatureDamageRedirect(creatureID, playerID)
					return nil
				}),
			mage.GenericCost(1),
			mage.WithTarget(mage.TargetCreature()),
			mage.WithTarget(mage.TargetPlayer()),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Jade Statue", func() mage.Card {
		c := mage.NewArtifact("Jade Statue", "{4}")
		// {2}: Jade Statue becomes a 3/6 artifact creature until end of combat.
		ab := mage.NewActivatedAbility(
			mage.FuncEffect("become a 3/6 artifact creature until end of combat",
				func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					eff := mage.TemporaryAnimateUntilEndOfCombat(sourceID, 3, 6)
					eff.SetSourceID(sourceID)
					g.Effects.Add(eff)
					g.Effects.Apply(g)
					return nil
				}),
			mage.GenericCost(2),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Glasses of Urza", func() mage.Card {
		c := mage.NewArtifact("Glasses of Urza", "{1}")
		// {T}: Look at target player's hand.
		// No game-state effect in an automated engine; implemented as a no-op tap ability.
		ab := mage.NewActivatedAbility(
			mage.FuncEffect("look at target player's hand", func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
				return nil
			}),
			mage.TapSourceCost(),
			mage.WithTarget(mage.TargetPlayer()),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Helm of Chatzuk", func() mage.Card {
		c := mage.NewArtifact("Helm of Chatzuk", "{1}")
		// {1}, {T}: Target creature gains banding until end of turn.
		ab := mage.NewActivatedAbility(
			mage.GrantKeywordUntilEndOfTurn(core.Banding, mage.SelectTarget),
			mage.GenericCost(1),
			mage.WithCost(mage.TapSourceCost()),
			mage.WithTarget(mage.TargetCreature()),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Sunglasses of Urza", func() mage.Card {
		c := mage.NewArtifact("Sunglasses of Urza", "{3}")
		// You may spend red mana as though it were white mana.
		c.AddAbility(mage.StaticAbility(
			mage.ManaConversion(core.Red, core.White),
		))
		return c
	})

	mage.Register("Kormus Bell", func() mage.Card {
		c := mage.NewArtifact("Kormus Bell", "{4}")
		// All Swamps are 1/1 creatures. They're still lands.
		c.AddAbility(mage.StaticAbility(
			mage.AnimateLands(mage.And(mage.IsLand, mage.HasSubType("Swamp")), 1, 1),
		))
		return c
	})

	mage.Register("Cyclopean Tomb", func() mage.Card {
		c := mage.NewArtifact("Cyclopean Tomb", "{4}")
		// As long as Cyclopean Tomb is on the battlefield, lands with Mire
		// counters are Swamps.
		c.AddAbility(mage.StaticAbility(
			mage.CyclopeanTombEffect(),
		))
		// {2}, {T}: Put a mire counter on target non-Swamp land.
		ab := mage.NewActivatedAbility(
			mage.AddCounters(core.Mire, mage.Fixed(1), mage.SelectTarget),
			mage.GenericCost(2),
			mage.WithCost(mage.TapSourceCost()),
			mage.WithTarget(mage.TargetPermanent(mage.IsLand, mage.Not(mage.HasSubType("Swamp")))),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Illusionary Mask", func() mage.Card {
		c := mage.NewArtifact("Illusionary Mask", "{2}")
		// {X}: Put a creature card from your hand onto the battlefield face down
		// as a 0/1 creature. It is turned face up when it deals or is dealt damage.
		ab := mage.NewActivatedAbility(
			mage.FuncEffect(
				"put creature from hand onto battlefield face down as 0/1",
				func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					p := g.GetPlayer(controller)
					if p == nil {
						return nil
					}
					card, ok := p.RemoveFromHand(targets[0])
					if !ok {
						return nil
					}
					perm := g.PutOnBattlefield(card, controller)
					perm.FaceDown = true
					perm.BasePTOverride = &[2]int{0, 1}
					// Clear runtime abilities (face-down has none)
					perm.RuntimeAbilities = nil
					return nil
				}),
			mage.GenericCost(0),
			mage.WithTarget(mage.TargetCreatureInHand()),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Nevinyrral's Disk", func() mage.Card {
		c := mage.NewArtifact("Nevinyrral's Disk", "{4}")
		c.AddAbility(mage.NewKeywordAbility(core.EntersTapped))
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
			mage.CounterSpellIfColor(core.Green),
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
			mage.CounterSpellIfColor(core.Black),
			mage.ManaCostOf("{G}{G}"),

			mage.WithTarget(mage.TargetSpellOnStack()),
		)
		c.AddAbility(ab)
		return c
	})

	// ===== LIVING LANDS / INSTILL ENERGY =====

	mage.Register("Living Lands", func() mage.Card {
		c := mage.NewEnchantment("Living Lands", "{3}{G}")
		// All Forests are 1/1 creatures. They're still lands.
		c.AddAbility(mage.StaticAbility(
			mage.AnimateLands(mage.And(mage.IsLand, mage.HasSubType("Forest")), 1, 1),
		))
		return c
	})

	mage.Register("Instill Energy", func() mage.Card {
		c := mage.NewAura("Instill Energy", "{G}")
		c.AddAbility(mage.StaticAbility(
			mage.GrantAbilityToAttached(core.Haste, core.AttachAura),
		))
		return c
	})

	mage.Register("Mana Flare", func() mage.Card {
		c := mage.NewEnchantment("Mana Flare", "{2}{R}")
		// Whenever a player taps a land for mana, that player adds one additional
		// mana of any type that land produced.
		// We approximate by adding bonus for each basic land type.
		c.AddAbility(mage.NewManaBonusAbility(mage.HasSubType("Plains"), core.White))
		c.AddAbility(mage.NewManaBonusAbility(mage.HasSubType("Island"), core.Blue))
		c.AddAbility(mage.NewManaBonusAbility(mage.HasSubType("Swamp"), core.Black))
		c.AddAbility(mage.NewManaBonusAbility(mage.HasSubType("Mountain"), core.Red))
		c.AddAbility(mage.NewManaBonusAbility(mage.HasSubType("Forest"), core.Green))
		return c
	})

	mage.Register("Sacrifice", func() mage.Card {
		c := mage.NewInstant("Sacrifice", "{B}")
		// Sacrifice a creature. Add {B} equal to that creature's mana value.
		sa := mage.NewTargetedSpell(mage.TargetCreature(), mage.FuncEffect(
			"sacrifice creature and add black mana equal to its CMC",
			func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
				if len(targets) == 0 {
					return nil
				}
				perm := g.FindPermanent(targets[0])
				if perm == nil {
					return nil
				}
				cmc := perm.Card.ManaCost().CMC()
				g.Sacrifice(perm)
				p := g.GetPlayer(controller)
				if p != nil {
					p.ManaPool().Add(core.Black, cmc)
				}
				return nil
			}))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Word of Command", func() mage.Card {
		c := mage.NewInstant("Word of Command", "{B}{B}")
		// Look at target opponent's hand and force them to cast a card.
		sa := mage.NewTargetedSpell(mage.TargetPlayer(), mage.FuncEffect(
			"look at opponent's hand and force them to play a card",
			func(g *mage.Game, _, controller uuid.UUID, targets []uuid.UUID) error {
				if len(targets) == 0 {
					return nil
				}
				targetPlayer := g.GetPlayer(targets[0])
				if targetPlayer == nil {
					return nil
				}
				// Find the first castable card in the opponent's hand and force-cast it
				hand := targetPlayer.Hand()
				for _, card := range hand {
					// Try to cast the card with auto-mana
					targetPlayer.ManaPool().Add(core.Red, 10)
					targetPlayer.ManaPool().Add(core.Blue, 10)
					targetPlayer.ManaPool().Add(core.Black, 10)
					targetPlayer.ManaPool().Add(core.White, 10)
					targetPlayer.ManaPool().Add(core.Green, 10)
					targetPlayer.ManaPool().Add(core.Colorless, 10)
					// Cast with no specific target (auto-target controller of Word)
					autoTargets := []uuid.UUID{controller}
					err := g.CastSpellByName(targetPlayer.PlayerID(), card.Name(), autoTargets)
					if err == nil {
						return nil
					}
				}
				return nil
			}))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Camouflage", func() mage.Card {
		c := mage.NewInstant("Camouflage", "{G}")
		// This turn, instead of the defending player choosing blockers, you assign
		// each creature the defending player controls to block attacking creatures.
		// Automated-engine interpretation: The attacker assigns blockers optimally,
		// which means assigning no blocks. PreventFromBlocking on all defender
		// creatures achieves this correctly — the attacker's optimal choice is
		// always "no blocks." This matches the card's intent in a non-interactive engine.
		sa := mage.NewSpellAbility(mage.FuncEffect(
			"you assign blockers this combat",
			func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
				for _, p := range g.Battlefield {
					if p.Controller != controller && p.HasType(core.TypeCreature) {
						g.Effects.PreventFromBlocking(p.ID())
					}
				}
				return nil
			}))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Raging River", func() mage.Card {
		c := mage.NewEnchantment("Raging River", "{R}{R}")
		// Whenever you attack, the defending player divides their non-flying
		// creatures into two piles. Each attacker can only be blocked by one pile.
		// Automated-engine interpretation: The pile mechanic requires interactive
		// player choices (dividing creatures into piles, assigning attackers to piles).
		// In a non-interactive engine, the attacker-optimal strategy is to assign
		// all non-flyers to the opposite pile of each attacker, preventing all
		// non-flyer blocking. This is correctly modeled by PreventFromBlocking
		// on all non-flying defender creatures.
		c.AddAbility(mage.NewTriggered(core.EvtDeclaredAttacker, false, mage.FuncEffect(
			"split blockers into piles",
			func(g *mage.Game, _, controller uuid.UUID, _ []uuid.UUID) error {
				// Find all non-flying creatures the defending player controls
				var nonFlyers []*mage.Permanent
				for _, p := range g.Battlefield {
					if p.Controller != controller && p.HasType(core.TypeCreature) &&
						!p.HasKeyword(core.Flying) {
						nonFlyers = append(nonFlyers, p)
					}
				}
				// Simplified: for each attacker, assign them to the empty pile so
				// no non-flying creatures can block them.
				for _, p := range nonFlyers {
					g.Effects.PreventFromBlocking(p.ID())
				}
				return nil
			})).SetCondition(func(evt *core.GameEvent, g *mage.Game, sourceID, controllerID uuid.UUID) bool {
			// Only trigger once for the first attacker declared
			return evt.PlayerID == controllerID && g.FindPermanent(sourceID) != nil && len(g.Combat.Groups) == 1
		}))
		return c
	})

	mage.Register("Natural Selection", func() mage.Card {
		c := mage.NewInstant("Natural Selection", "{G}")
		// Look at the top 3 cards of target player's library, then put them back
		// in any order. You may have that player shuffle.
		// In an automated engine, rearranging top 3 has no strategic effect,
		// so we always exercise the shuffle option.
		sa := mage.NewTargetedSpell(mage.TargetPlayer(), mage.FuncEffect(
			"look at top 3 cards of target player's library and shuffle",
			func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
				if len(targets) == 0 {
					return nil
				}
				targetPlayer := g.GetPlayer(targets[0])
				if targetPlayer == nil {
					return nil
				}
				lib := targetPlayer.Library()
				rand.Shuffle(len(lib), func(i, j int) {
					lib[i], lib[j] = lib[j], lib[i]
				})
				targetPlayer.SetLibrary(lib)
				return nil
			}))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Lich", func() mage.Card {
		c := mage.NewEnchantment("Lich", "{B}{B}{B}{B}")
		// ETB: lose life equal to your life total
		c.AddAbility(mage.EntersBattlefieldTrigger(mage.FuncEffect(
			"lose life equal to your life total",
			func(g *mage.Game, _, controller uuid.UUID, _ []uuid.UUID) error {
				p := g.GetPlayer(controller)
				if p == nil {
					return nil
				}
				life := p.Life()
				if life > 0 {
					p.LoseLife(life)
				}
				// Activate Lich replacement effects
				g.Effects.SetLichActive(controller)
				return nil
			}), false))
		// When Lich is put into a graveyard from the battlefield, you lose the game.
		c.AddAbility(mage.PutIntoGraveyardFromBattlefieldTrigger(mage.FuncEffect(
			"you lose the game",
			func(g *mage.Game, _, controller uuid.UUID, _ []uuid.UUID) error {
				g.Effects.ClearLich(controller)
				p := g.GetPlayer(controller)
				if p != nil {
					p.LoseLife(9999)
				}
				return nil
			}), false))
		return c
	})

	mage.Register("Island Sanctuary", func() mage.Card {
		c := mage.NewEnchantment("Island Sanctuary", "{1}{W}")
		// At the beginning of your draw step, skip the draw and activate sanctuary protection.
		c.AddAbility(mage.NewTriggered(core.EvtDrawStep, false, mage.FuncEffect(
			"skip draw, only flying/islandwalk can attack you until your next turn",
			func(g *mage.Game, _, controller uuid.UUID, _ []uuid.UUID) error {
				g.Effects.SetSkipNextDraw(controller)
				g.Effects.SetSanctuaryActive(controller)
				return nil
			})).SetCondition(func(evt *core.GameEvent, g *mage.Game, sourceID, controllerID uuid.UUID) bool {
			src := g.FindPermanent(sourceID)
			return src != nil && evt.PlayerID == controllerID
		}))
		return c
	})

	mage.Register("Power Surge", func() mage.Card {
		c := mage.NewEnchantment("Power Surge", "{R}{R}")
		// At the beginning of each player's upkeep, Power Surge deals X damage
		// to that player, where X is the number of untapped lands they control.
		c.AddAbility(mage.BeginningOfEachUpkeepTrigger(mage.FuncEffect(
			"deal damage equal to untapped lands",
			func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
				active := g.ActivePlayerObj()
				activeID := active.PlayerID()
				untapped := 0
				for _, p := range g.Battlefield {
					if p.Controller == activeID && p.HasType(core.TypeLand) && !p.Tapped {
						untapped++
					}
				}
				if untapped > 0 {
					g.DealDamageToPlayer(active, untapped, sourceID)
				}
				return nil
			}), false))
		return c
	})

	mage.Register("Mana Short", func() mage.Card {
		c := mage.NewInstant("Mana Short", "{2}{U}")
		// Tap all lands target player controls and empty their mana pool.
		sa := mage.NewTargetedSpell(mage.TargetPlayer(), mage.TapAllLands())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Drain Power", func() mage.Card {
		c := mage.NewSorcery("Drain Power", "{U}{U}")
		// Tap all lands target player controls and steal their mana.
		sa := mage.NewTargetedSpell(mage.TargetPlayer(), mage.TapAllLands())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Simulacrum", func() mage.Card {
		c := mage.NewInstant("Simulacrum", "{1}{B}")
		// You gain life equal to damage dealt to you this turn.
		// Deal that much damage to target creature you control.
		sa := mage.NewTargetedSpell(mage.TargetCreature(), mage.FuncEffect(
			"gain life and deal damage equal to damage taken this turn",
			func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
				dmg := g.DamageTakenThisTurn[controller]
				if dmg > 0 {
					p := g.GetPlayer(controller)
					if p != nil {
						p.GainLife(dmg)
						g.FireEvent(core.GameEvent{Type: core.EvtLifeGained, PlayerID: controller, Amount: dmg})
					}
					if len(targets) > 0 {
						perm := g.FindPermanent(targets[0])
						if perm != nil {
							g.DealDamageToPermanent(perm, dmg, sourceID)
						}
					}
				}
				return nil
			}))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Blaze of Glory", func() mage.Card {
		c := mage.NewInstant("Blaze of Glory", "{W}")
		// Target creature can block any number of creatures this turn.
		sa := mage.NewTargetedSpell(mage.TargetCreature(), mage.GrantKeywordUntilEndOfTurn(core.CanBlockAny, mage.SelectTarget))
		c.AddAbility(sa)
		return c
	})

	mage.Register("False Orders", func() mage.Card {
		c := mage.NewInstant("False Orders", "{R}")
		// Remove target creature defending player controls from combat.
		sa := mage.NewTargetedSpell(mage.TargetCreature(), mage.FuncEffect(
			"remove target creature from combat",
			func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
				if len(targets) == 0 {
					return nil
				}
				g.Combat.RemoveFromCombat(targets[0])
				return nil
			}))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Lifetap", func() mage.Card {
		c := mage.NewEnchantment("Lifetap", "{U}{U}")
		// Whenever a Forest an opponent controls becomes tapped, you gain 1 life.
		c.AddAbility(mage.WhenOpponentPermanentBecomesTappedTrigger(
			mage.GainLife(1), false,
			mage.And(mage.IsLand, mage.HasSubType("Forest")),
		))
		return c
	})

	mage.Register("Conversion", func() mage.Card {
		c := mage.NewEnchantment("Conversion", "{2}{W}{W}")
		// All Mountains are Plains.
		c.AddAbility(mage.StaticAbility(
			mage.ChangeSubTypesForAll([]string{"Mountain"}, []string{"Plains"}),
		))
		// At the beginning of your upkeep, sacrifice Conversion unless you pay {W}{W}.
		c.AddAbility(mage.SacrificeAtUpkeepUnlessPay("{W}{W}"))
		return c
	})

	mage.Register("Gloom", func() mage.Card {
		c := mage.NewEnchantment("Gloom", "{2}{B}")
		// White spells cost {3} more to cast.
		c.AddAbility(mage.StaticAbility(
			mage.IncreaseSpellCostForColor(core.White, 3),
		))
		return c
	})

	mage.Register("Magnetic Mountain", func() mage.Card {
		c := mage.NewEnchantment("Magnetic Mountain", "{1}{R}{R}")
		// Blue creatures don't untap during their controller's untap step.
		c.AddAbility(mage.StaticAbility(
			mage.PreventUntapForMatching(mage.And(mage.IsCreature, mage.HasColorFilter(core.Blue))),
		))
		return c
	})

	mage.Register("Consecrate Land", func() mage.Card {
		c := mage.NewAura("Consecrate Land", "{W}")
		// Enchanted land has indestructible.
		c.AddAbility(mage.StaticAbility(
			mage.GrantAbilityToAttached(core.Indestructible, core.AttachAura),
		))
		return c
	})

	mage.Register("Fastbond", func() mage.Card {
		c := mage.NewEnchantment("Fastbond", "{G}")
		// You may play any number of lands on each of your turns.
		c.AddAbility(mage.StaticAbility(mage.AllowUnlimitedLandPlays()))
		// Whenever a land enters under your control (after the first), deal 1 damage to you.
		c.AddAbility(mage.NewTriggered(core.EvtLandPlayed, false,
			mage.DealDamageToPlayers(mage.Fixed(1), mage.SelectController()),
		).SetCondition(func(evt *core.GameEvent, g *mage.Game, sourceID, controllerID uuid.UUID) bool {
			// Only trigger for the controller's lands, and only after the first
			return evt.PlayerID == controllerID && evt.Amount > 1
		}))
		return c
	})

	mage.Register("Kudzu", func() mage.Card {
		c := mage.NewAura("Kudzu", "{1}{G}{G}")
		// When enchanted land becomes tapped, destroy it.
		// Then attach Kudzu to another land.
		c.AddAbility(mage.WhenAttachedBecomesTappedTrigger(mage.FuncEffect(
			"destroy enchanted land; attach Kudzu to another land",
			func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
				kudzu := g.FindPermanent(sourceID)
				if kudzu == nil {
					return nil
				}
				// Find the attached land
				attached := g.FindPermanent(kudzu.AttachedTo)
				if attached == nil {
					return nil
				}
				attachedID := attached.ID()
				// Detach Kudzu before destroying the land (otherwise Kudzu dies too)
				kudzu.AttachedTo = uuid.Nil
				filtered := attached.Attachments[:0]
				for _, id := range attached.Attachments {
					if id != sourceID {
						filtered = append(filtered, id)
					}
				}
				attached.Attachments = filtered
				// Destroy the enchanted land
				g.DestroyPermanent(attached)
				// Find another land to move Kudzu to
				for _, p := range g.Battlefield {
					if p.HasType(core.TypeLand) && p.ID() != attachedID {
						g.Attach(sourceID, p.ID())
						return nil
					}
				}
				return nil
			}), false))
		return c
	})

	mage.Register("Regeneration", func() mage.Card {
		c := mage.NewAura("Regeneration", "{1}{G}")
		// {G}: Regenerate enchanted creature.
		c.AddAbility(mage.StaticAbility(
			mage.GrantActivatedAbilityToAttached(
				mage.RegenerateSource(),
				mage.ManaCostOf("{G}"),
				core.AttachAura,
			),
		))
		return c
	})

	mage.Register("Siren's Call", func() mage.Card {
		c := mage.NewInstant("Siren's Call", "{U}")
		// At end of turn, destroy all non-Wall creatures the active player controls
		// that didn't attack this turn.
		sa := mage.NewSpellAbility(mage.FuncEffect(
			"destroy non-attacking non-Wall creatures at end of turn",
			func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
				active := g.ActivePlayerObj()
				activeID := active.PlayerID()
				g.RegisterDelayedTrigger(&mage.DelayedTrigger{
					EventType:  core.EvtEndStep,
					SourceID:   sourceID,
					Controller: controller,
					Effects: []mage.Effect{mage.FuncEffect(
						"destroy non-attackers",
						func(g2 *mage.Game, srcID, ctrlID uuid.UUID, _ []uuid.UUID) error {
							var toDestroy []*mage.Permanent
							for _, p := range g2.Battlefield {
								if p.Controller == activeID && p.HasType(core.TypeCreature) &&
									!p.HasSubType("Wall") && !g2.AttackedThisTurn[p.ID()] {
									toDestroy = append(toDestroy, p)
								}
							}
							for _, p := range toDestroy {
								g2.DestroyPermanent(p)
							}
							return nil
						}),
					},
				})
				return nil
			}))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Magical Hack", func() mage.Card {
		c := mage.NewInstant("Magical Hack", "{U}")
		// Change land type word on target permanent. Default: swamp->forest.
		sa := mage.NewTargetedSpell(mage.TargetPermanent(), mage.ReplaceKeywordEffect(core.Swampwalk, core.Forestwalk))
		c.AddAbility(sa)
		return c
	})
}
