package arabian

import (
	"github.com/google/uuid"
	. "github.com/mage/mage/pkg/mage"
	. "github.com/mage/mage/pkg/mage/core"
)

func init() {
	registerCreatures()
}

func registerCreatures() {
	// ===== WHITE CREATURES =====

	Register("Abu Ja'far", func() Card {
		return NewCreature("Abu Ja'far", "{W}", 0, 1,
			WithSubTypes("Human"),
			WithAbility(
				NewTriggered(EvtCreatureDied, false,
					FuncEffect("destroy all creatures blocking or blocked by Abu Ja'far",
						func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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
				).SetCondition(IsThisSource),
			),
		)
	})

	Register("Camel", func() Card {
		return NewCreature("Camel", "{W}", 0, 1,
			WithSubTypes("Camel"),
			WithKeyword(Banding),
			WithStaticAbility(PreventDamageFromTo(
				HasSubType("Desert"),
				func(sourceID uuid.UUID) PermanentFilter {
					return Or(IsID(sourceID), IsBandedWith(sourceID))
				},
				WhileSourceAttacking,
			)),
		)
	})

	Register("King Suleiman", func() Card {
		return NewCreature("King Suleiman", "{1}{W}", 1, 1,
			WithSubTypes("Human", "Noble"),
			WithActivatedAbility(
				DestroyTarget(),
				TapSourceCost(),
				WithTarget(TargetCreature(
					Or(HasSubType("Djinn"), HasSubType("Efreet")),
				)),
			),
		)
	})

	Register("Moorish Cavalry", func() Card {
		return NewCreature("Moorish Cavalry", "{2}{W}{W}", 3, 3,
			WithSubTypes("Human", "Knight"),
			WithKeyword(Trample),
		)
	})

	Register("Repentant Blacksmith", func() Card {
		return NewCreature("Repentant Blacksmith", "{1}{W}", 1, 2,
			WithSubTypes("Human"),
			WithAbility(ProtectionFromColor(Red)),
		)
	})

	Register("War Elephant", func() Card {
		return NewCreature("War Elephant", "{3}{W}", 2, 2,
			WithSubTypes("Elephant"),
			WithKeyword(Trample),
			WithKeyword(Banding),
		)
	})

	// ===== BLUE CREATURES =====

	Register("Dandân", func() Card {
		return NewCreature("Dandân", "{U}{U}", 4, 1,
			WithSubTypes("Fish"),
			WithStaticAbility(PreventFromAttackingIfDefendingPlayerControls(HasSubType("Island"))),
			WithAbility(NewTriggered(EvtLeavesBattlefield, false, SacrificeSource()).
				SetCondition(func(evt *GameEvent, g *Game, sourceID, controllerID uuid.UUID) bool {
					for _, p := range g.Battlefield {
						if p.Controller == controllerID && p.HasSubType("Island") {
							return false
						}
					}
					return true
				})),
		)
	})

	Register("Flying Men", func() Card {
		return NewCreature("Flying Men", "{U}", 1, 1,
			WithSubTypes("Human"),
			WithKeyword(Flying),
		)
	})

	Register("Giant Tortoise", func() Card {
		return NewCreature("Giant Tortoise", "{1}{U}", 1, 1,
			WithSubTypes("Turtle"),
			WithStaticAbility(BoostSelf(0, 3, WhileSourceUntapped)),
		)
	})

	Register("Island Fish Jasconius", func() Card {
		return NewCreature("Island Fish Jasconius", "{4}{U}{U}{U}", 6, 8, WithSubTypes("Fish"))
	})

	Register("Merchant Ship", func() Card {
		return NewCreature("Merchant Ship", "{U}", 0, 2, WithSubTypes("Human"))
	})

	Register("Old Man of the Sea", func() Card {
		return NewCreature("Old Man of the Sea", "{1}{U}{U}", 2, 3, WithSubTypes("Djinn"))
	})

	Register("Serendib Djinn", func() Card {
		return NewCreature("Serendib Djinn", "{2}{U}{U}", 5, 6, WithSubTypes("Djinn"))
	})

	Register("Serendib Efreet", func() Card {
		return NewCreature("Serendib Efreet", "{2}{U}", 3, 4, WithSubTypes("Efreet"))
	})

	Register("Sindbad", func() Card {
		return NewCreature("Sindbad", "{1}{U}", 1, 1, WithSubTypes("Human"))
	})

	// ===== BLACK CREATURES =====

	Register("Cuombajj Witches", func() Card {
		return NewCreature("Cuombajj Witches", "{B}{B}", 1, 3, WithSubTypes("Human", "Wizard"))
	})

	Register("El-Hajjâj", func() Card {
		return NewCreature("El-Hajjâj", "{1}{B}{B}", 1, 1, WithSubTypes("Human", "Wizard"))
	})

	Register("Erg Raiders", func() Card {
		return NewCreature("Erg Raiders", "{1}{B}", 2, 3, WithSubTypes("Human", "Warrior"))
	})

	Register("Guardian Beast", func() Card {
		return NewCreature("Guardian Beast", "{3}{B}", 2, 4, WithSubTypes("Beast"))
	})

	Register("Hasran Ogress", func() Card {
		return NewCreature("Hasran Ogress", "{B}{B}", 3, 2, WithSubTypes("Ogre"))
	})

	Register("Junún Efreet", func() Card {
		return NewCreature("Junún Efreet", "{1}{B}{B}", 3, 3, WithSubTypes("Efreet"))
	})

	Register("Juzám Djinn", func() Card {
		return NewCreature("Juzám Djinn", "{2}{B}{B}", 5, 5, WithSubTypes("Djinn"))
	})

	Register("Khabál Ghoul", func() Card {
		return NewCreature("Khabál Ghoul", "{2}{B}", 1, 1, WithSubTypes("Zombie"))
	})

	Register("Sorceress Queen", func() Card {
		return NewCreature("Sorceress Queen", "{1}{B}{B}", 1, 1, WithSubTypes("Human", "Wizard"))
	})

	Register("Stone-Throwing Devils", func() Card {
		return NewCreature("Stone-Throwing Devils", "{B}", 1, 1, WithSubTypes("Devil"))
	})

	// ===== RED CREATURES =====

	Register("Aladdin", func() Card {
		return NewCreature("Aladdin", "{2}{R}{R}", 1, 1, WithSubTypes("Human", "Rogue"))
	})

	Register("Ali Baba", func() Card {
		return NewCreature("Ali Baba", "{R}", 1, 1, WithSubTypes("Human", "Rogue"))
	})

	Register("Ali from Cairo", func() Card {
		return NewCreature("Ali from Cairo", "{2}{R}{R}", 0, 1, WithSubTypes("Human"))
	})

	Register("Bird Maiden", func() Card {
		return NewCreature("Bird Maiden", "{2}{R}", 1, 2, WithSubTypes("Human", "Bird"))
	})

	Register("Desert Nomads", func() Card {
		return NewCreature("Desert Nomads", "{2}{R}", 2, 2, WithSubTypes("Human", "Nomad"))
	})

	Register("Hurr Jackal", func() Card {
		return NewCreature("Hurr Jackal", "{R}", 1, 1, WithSubTypes("Jackal"))
	})

	Register("Kird Ape", func() Card {
		return NewCreature("Kird Ape", "{R}", 1, 1, WithSubTypes("Ape"))
	})

	Register("Mijae Djinn", func() Card {
		return NewCreature("Mijae Djinn", "{R}{R}{R}", 6, 3, WithSubTypes("Djinn"))
	})

	Register("Rukh Egg", func() Card {
		return NewCreature("Rukh Egg", "{3}{R}", 0, 3, WithSubTypes("Bird", "Egg"))
	})

	Register("Ydwen Efreet", func() Card {
		return NewCreature("Ydwen Efreet", "{R}{R}{R}", 3, 6, WithSubTypes("Efreet"))
	})

	// ===== GREEN CREATURES =====

	Register("Erhnam Djinn", func() Card {
		return NewCreature("Erhnam Djinn", "{3}{G}", 4, 5, WithSubTypes("Djinn"))
	})

	Register("Ghazbán Ogre", func() Card {
		return NewCreature("Ghazbán Ogre", "{G}", 2, 2, WithSubTypes("Ogre"))
	})

	Register("Ifh-Bíff Efreet", func() Card {
		return NewCreature("Ifh-Bíff Efreet", "{2}{G}{G}", 3, 3, WithSubTypes("Efreet"))
	})

	Register("Nafs Asp", func() Card {
		return NewCreature("Nafs Asp", "{G}", 1, 1, WithSubTypes("Snake"))
	})

	Register("Singing Tree", func() Card {
		return NewCreature("Singing Tree", "{3}{G}", 0, 3, WithSubTypes("Plant"))
	})

	Register("Wyluli Wolf", func() Card {
		return NewCreature("Wyluli Wolf", "{1}{G}", 1, 1, WithSubTypes("Wolf"))
	})

	// ===== ARTIFACT CREATURES =====

	Register("Brass Man", func() Card {
		return NewCreature("Brass Man", "{1}", 1, 3,
			WithSubTypes("Construct"),
			WithCardType(TypeArtifact),
		)
	})

	Register("Dancing Scimitar", func() Card {
		return NewCreature("Dancing Scimitar", "{4}", 1, 5,
			WithSubTypes("Spirit"),
			WithCardType(TypeArtifact),
		)
	})
}
