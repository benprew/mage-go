package arabian

import (
	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

func init() {
	registerCreatures()
}

func registerCreatures() {
	// ===== WHITE CREATURES =====

	mage.Register("Abu Ja'far", func() mage.Card {
		return mage.NewCreature("Abu Ja'far", "{W}", 0, 1,
			mage.WithSubTypes("Human"),
			mage.WithAbility(
				mage.NewTriggered(core.EvtCreatureDied, false,
					mage.FuncEffect("destroy all creatures blocking or blocked by Abu Ja'far",
						func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							if g.Combat == nil {
								return nil
							}
							var toDestroy []uuid.UUID
							for _, group := range g.Combat.Groups {
								if group.AttackerID == sourceID {
									toDestroy = append(toDestroy, group.BlockerIDs...)
								}
								for _, bid := range group.BlockerIDs {
									if bid == sourceID {
										toDestroy = append(toDestroy, group.AttackerID)
									}
								}
							}
							for _, id := range toDestroy {
								p := g.FindPermanent(id)
								if p != nil {
									g.DestroyPermanent(p)
								}
							}
							return nil
						}),
				).SetCondition(mage.IsThisSource),
			),
		)
	})

	mage.Register("Camel", func() mage.Card {
		return mage.NewCreature("Camel", "{W}", 0, 1,
			mage.WithSubTypes("Camel"),
			mage.WithKeyword(core.Banding),
			mage.WithAbility(mage.StaticAbility(mage.PreventDamageFromTo(
				mage.HasSubType("Desert"),
				func(sourceID uuid.UUID) mage.PermanentFilter {
					return mage.Or(mage.IsID(sourceID), mage.IsBandedWith(sourceID))
				},
				mage.WhileSourceAttacking,
			))),
		)
	})

	mage.Register("King Suleiman", func() mage.Card {
		return mage.NewCreature("King Suleiman", "{1}{W}", 1, 1,
			mage.WithSubTypes("Human", "Noble"),
			mage.WithAbility(mage.NewActivatedAbility(
				mage.DestroyTarget(),
				mage.TapSourceCost(),
				mage.WithTarget(mage.TargetCreature(
					mage.Or(mage.HasSubType("Djinn"), mage.HasSubType("Efreet")),
				)),
			)),
		)
	})

	mage.Register("Moorish Cavalry", func() mage.Card {
		return mage.NewCreature("Moorish Cavalry", "{2}{W}{W}", 3, 3,
			mage.WithSubTypes("Human", "Knight"),
			mage.WithKeyword(core.Trample),
		)
	})

	mage.Register("Repentant Blacksmith", func() mage.Card {
		return mage.NewCreature("Repentant Blacksmith", "{1}{W}", 1, 2,
			mage.WithSubTypes("Human"),
			mage.WithAbility(mage.ProtectionFromColor(core.Red)),
		)
	})

	mage.Register("War Elephant", func() mage.Card {
		return mage.NewCreature("War Elephant", "{3}{W}", 2, 2,
			mage.WithSubTypes("Elephant"),
			mage.WithKeyword(core.Trample),
			mage.WithKeyword(core.Banding),
		)
	})

	// ===== BLUE CREATURES =====

	mage.Register("Dandân", func() mage.Card {
		return mage.NewCreature("Dandân", "{U}{U}", 4, 1,
			mage.WithSubTypes("Fish"),
			mage.WithAbility(mage.StaticAbility(mage.PreventFromAttackingIfDefendingPlayerControls(mage.HasSubType("Island")))),
			mage.WithAbility(mage.NewTriggered(core.EvtLeavesBattlefield, false, mage.SacrificeSource()).
				SetCondition(func(evt *core.GameEvent, g *mage.Game, sourceID, controllerID uuid.UUID) bool {
					for _, p := range g.Battlefield {
						if p.Controller == controllerID && p.HasSubType("Island") {
							return false
						}
					}
					return true
				})),
		)
	})

	mage.Register("Flying Men", func() mage.Card {
		return mage.NewCreature("Flying Men", "{U}", 1, 1,
			mage.WithSubTypes("Human"),
			mage.WithKeyword(core.Flying),
		)
	})

	mage.Register("Giant Tortoise", func() mage.Card {
		return mage.NewCreature("Giant Tortoise", "{1}{U}", 1, 1,
			mage.WithSubTypes("Turtle"),
			mage.WithAbility(mage.StaticAbility(mage.BoostSelf(0, 3, mage.WhileSourceUntapped))),
		)
	})

	mage.Register("Island Fish Jasconius", func() mage.Card {
		return mage.NewCreature("Island Fish Jasconius", "{4}{U}{U}{U}", 6, 8, mage.WithSubTypes("Fish"))
	})

	mage.Register("Merchant Ship", func() mage.Card {
		return mage.NewCreature("Merchant Ship", "{U}", 0, 2, mage.WithSubTypes("Human"))
	})

	mage.Register("Old Man of the Sea", func() mage.Card {
		return mage.NewCreature("Old Man of the Sea", "{1}{U}{U}", 2, 3, mage.WithSubTypes("Djinn"))
	})

	mage.Register("Serendib Djinn", func() mage.Card {
		return mage.NewCreature("Serendib Djinn", "{2}{U}{U}", 5, 6, mage.WithSubTypes("Djinn"))
	})

	mage.Register("Serendib Efreet", func() mage.Card {
		return mage.NewCreature("Serendib Efreet", "{2}{U}", 3, 4, mage.WithSubTypes("Efreet"))
	})

	mage.Register("Sindbad", func() mage.Card {
		return mage.NewCreature("Sindbad", "{1}{U}", 1, 1, mage.WithSubTypes("Human"))
	})

	// ===== BLACK CREATURES =====

	mage.Register("Cuombajj Witches", func() mage.Card {
		return mage.NewCreature("Cuombajj Witches", "{B}{B}", 1, 3, mage.WithSubTypes("Human", "Wizard"))
	})

	mage.Register("El-Hajjâj", func() mage.Card {
		return mage.NewCreature("El-Hajjâj", "{1}{B}{B}", 1, 1, mage.WithSubTypes("Human", "Wizard"))
	})

	mage.Register("Erg Raiders", func() mage.Card {
		return mage.NewCreature("Erg Raiders", "{1}{B}", 2, 3, mage.WithSubTypes("Human", "Warrior"))
	})

	mage.Register("Guardian Beast", func() mage.Card {
		return mage.NewCreature("Guardian Beast", "{3}{B}", 2, 4, mage.WithSubTypes("Beast"))
	})

	mage.Register("Hasran Ogress", func() mage.Card {
		return mage.NewCreature("Hasran Ogress", "{B}{B}", 3, 2, mage.WithSubTypes("Ogre"))
	})

	mage.Register("Junún Efreet", func() mage.Card {
		return mage.NewCreature("Junún Efreet", "{1}{B}{B}", 3, 3, mage.WithSubTypes("Efreet"))
	})

	mage.Register("Juzám Djinn", func() mage.Card {
		return mage.NewCreature("Juzám Djinn", "{2}{B}{B}", 5, 5, mage.WithSubTypes("Djinn"))
	})

	mage.Register("Khabál Ghoul", func() mage.Card {
		return mage.NewCreature("Khabál Ghoul", "{2}{B}", 1, 1, mage.WithSubTypes("Zombie"))
	})

	mage.Register("Sorceress Queen", func() mage.Card {
		return mage.NewCreature("Sorceress Queen", "{1}{B}{B}", 1, 1, mage.WithSubTypes("Human", "Wizard"))
	})

	mage.Register("Stone-Throwing Devils", func() mage.Card {
		return mage.NewCreature("Stone-Throwing Devils", "{B}", 1, 1, mage.WithSubTypes("Devil"))
	})

	// ===== RED CREATURES =====

	mage.Register("Aladdin", func() mage.Card {
		return mage.NewCreature("Aladdin", "{2}{R}{R}", 1, 1, mage.WithSubTypes("Human", "Rogue"))
	})

	mage.Register("Ali Baba", func() mage.Card {
		return mage.NewCreature("Ali Baba", "{R}", 1, 1, mage.WithSubTypes("Human", "Rogue"))
	})

	mage.Register("Ali from Cairo", func() mage.Card {
		return mage.NewCreature("Ali from Cairo", "{2}{R}{R}", 0, 1, mage.WithSubTypes("Human"))
	})

	mage.Register("Bird Maiden", func() mage.Card {
		return mage.NewCreature("Bird Maiden", "{2}{R}", 1, 2, mage.WithSubTypes("Human", "Bird"))
	})

	mage.Register("Desert Nomads", func() mage.Card {
		return mage.NewCreature("Desert Nomads", "{2}{R}", 2, 2, mage.WithSubTypes("Human", "Nomad"))
	})

	mage.Register("Hurr Jackal", func() mage.Card {
		return mage.NewCreature("Hurr Jackal", "{R}", 1, 1, mage.WithSubTypes("Jackal"))
	})

	mage.Register("Kird Ape", func() mage.Card {
		return mage.NewCreature("Kird Ape", "{R}", 1, 1, mage.WithSubTypes("Ape"))
	})

	mage.Register("Mijae Djinn", func() mage.Card {
		return mage.NewCreature("Mijae Djinn", "{R}{R}{R}", 6, 3, mage.WithSubTypes("Djinn"))
	})

	mage.Register("Rukh Egg", func() mage.Card {
		return mage.NewCreature("Rukh Egg", "{3}{R}", 0, 3, mage.WithSubTypes("Bird", "Egg"))
	})

	mage.Register("Ydwen Efreet", func() mage.Card {
		return mage.NewCreature("Ydwen Efreet", "{R}{R}{R}", 3, 6, mage.WithSubTypes("Efreet"))
	})

	// ===== GREEN CREATURES =====

	mage.Register("Erhnam Djinn", func() mage.Card {
		return mage.NewCreature("Erhnam Djinn", "{3}{G}", 4, 5, mage.WithSubTypes("Djinn"))
	})

	mage.Register("Ghazbán Ogre", func() mage.Card {
		return mage.NewCreature("Ghazbán Ogre", "{G}", 2, 2, mage.WithSubTypes("Ogre"))
	})

	mage.Register("Ifh-Bíff Efreet", func() mage.Card {
		return mage.NewCreature("Ifh-Bíff Efreet", "{2}{G}{G}", 3, 3, mage.WithSubTypes("Efreet"))
	})

	mage.Register("Nafs Asp", func() mage.Card {
		return mage.NewCreature("Nafs Asp", "{G}", 1, 1, mage.WithSubTypes("Snake"))
	})

	mage.Register("Singing Tree", func() mage.Card {
		return mage.NewCreature("Singing Tree", "{3}{G}", 0, 3, mage.WithSubTypes("Plant"))
	})

	mage.Register("Wyluli Wolf", func() mage.Card {
		return mage.NewCreature("Wyluli Wolf", "{1}{G}", 1, 1, mage.WithSubTypes("Wolf"))
	})

	// ===== ARTIFACT CREATURES =====

	mage.Register("Brass Man", func() mage.Card {
		return mage.NewCreature("Brass Man", "{1}", 1, 3,
			mage.WithSubTypes("Construct"),
			mage.WithCardType(core.TypeArtifact),
		)
	})

	mage.Register("Dancing Scimitar", func() mage.Card {
		return mage.NewCreature("Dancing Scimitar", "{4}", 1, 5,
			mage.WithSubTypes("Spirit"),
			mage.WithCardType(core.TypeArtifact),
		)
	})
}
