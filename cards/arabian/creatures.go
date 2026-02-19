package arabian

import (
	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
)

func init() {
	registerCreatures()
}

func registerCreatures() {
	// ===== WHITE CREATURES =====

	mage.Register("Abu Ja'far", func() mage.Card {
		// When Abu Ja'far dies, destroy all creatures blocking or blocked by it.
		// They can't be regenerated.
		c := mage.NewCreature("Abu Ja'far", "{W}", 0, 1, "Human")
		c.AddAbility(
			mage.NewTriggered(mage.EvtCreatureDied, false,
				mage.FuncEffect("destroy all creatures blocking or blocked by Abu Ja'far",
					func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if g.Combat == nil {
							return nil
						}
						var toDestroy []uuid.UUID
						// simple predicate filter doesn't work since attacker/blocker state isn't on the permanent
						// ZZZ for now....
						for _, group := range g.Combat.Groups {
							// Abu Ja'far was the attacker — destroy its blockers
							if group.AttackerID == sourceID {
								toDestroy = append(toDestroy, group.BlockerIDs...)
							}
							// Abu Ja'far was a blocker — destroy the attacker
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
		)
		return c
	})

	mage.Register("Camel", func() mage.Card {
		// Banding
		// As long as Camel is attacking, prevent all damage Deserts would deal to
		// Camel and to creatures banded with Camel.
		c := mage.NewCreature("Camel", "{W}", 0, 1, "Camel")
		c.AddAbility(mage.NewKeywordAbility(mage.Banding))
		c.AddAbility(mage.StaticAbility(mage.PreventDamageFromTo(
			mage.HasSubType("Desert"),
			func(sourceID uuid.UUID) mage.PermanentFilter {
				return mage.Or(mage.IsID(sourceID), mage.IsBandedWith(sourceID))
			},
			mage.WhileSourceAttacking,
		)))
		return c
	})

	mage.Register("King Suleiman", func() mage.Card {
		// {T}: Destroy target Djinn or Efreet.
		c := mage.NewCreature("King Suleiman", "{1}{W}", 1, 1, "Human", "Noble")
		ab := mage.NewActivatedAbility(
			mage.DestroyTarget(),
			mage.TapSourceCost(),
			mage.WithTarget(mage.TargetCreature(
				mage.Or(mage.HasSubType("Djinn"), mage.HasSubType("Efreet")),
			)),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Moorish Cavalry", func() mage.Card {
		// Trample
		c := mage.NewCreature("Moorish Cavalry", "{2}{W}{W}", 3, 3, "Human", "Knight")
		c.AddAbility(mage.NewKeywordAbility(mage.Trample))
		return c
	})

	mage.Register("Repentant Blacksmith", func() mage.Card {
		// Protection from red
		c := mage.NewCreature("Repentant Blacksmith", "{1}{W}", 1, 2, "Human")
		c.AddAbility(mage.ProtectionFromColor(mage.Red))
		return c
	})

	mage.Register("War Elephant", func() mage.Card {
		// Trample; banding
		c := mage.NewCreature("War Elephant", "{3}{W}", 2, 2, "Elephant")
		c.AddAbility(mage.NewKeywordAbility(mage.Trample))
		c.AddAbility(mage.NewKeywordAbility(mage.Banding))
		return c
	})

	// ===== BLUE CREATURES =====

	mage.Register("Dandân", func() mage.Card {
		// Dandân can't attack unless defending player controls an Island.
		// When you control no Islands, sacrifice Dandân.
		c := mage.NewCreature("Dandân", "{U}{U}", 4, 1, "Fish")
		c.AddAbility(mage.StaticAbility(mage.PreventFromAttackingIfDefendingPlayerControls(mage.HasSubType("Island"))))
		sa := mage.NewTriggered(mage.EvtLeavesBattlefield,
			false,
			mage.SacrificeSource()).
			SetCondition(func(evt *mage.GameEvent, g *mage.Game, sourceID, controllerID uuid.UUID) bool {
				for _, p := range g.Battlefield {
					if p.Controller == controllerID && p.HasSubType("Island") {
						return false
					}
				}
				return true
			})
		c.AddAbility(sa)
		return c
	})

	mage.Register("Flying Men", func() mage.Card {
		// Flying
		c := mage.NewCreature("Flying Men", "{U}", 1, 1, "Human")
		c.AddAbility(mage.NewKeywordAbility(mage.Flying))
		return c
	})

	mage.Register("Giant Tortoise", func() mage.Card {
		// Giant Tortoise gets +0/+3 as long as it's untapped.
		c := mage.NewCreature("Giant Tortoise", "{1}{U}", 1, 1, "Turtle")
		c.AddAbility(mage.StaticAbility(mage.BoostSelf(0, 3, mage.WhileSourceUntapped)))
		return c
	})

	mage.Register("Island Fish Jasconius", func() mage.Card {
		// Island Fish Jasconius doesn't untap during your untap step.
		// At the beginning of your upkeep, you may pay {U}{U}{U}. If you do, untap
		// Island Fish Jasconius.
		// Island Fish Jasconius can't attack unless defending player controls an Island.
		// When you control no Islands, sacrifice Island Fish Jasconius.
		c := mage.NewCreature("Island Fish Jasconius", "{4}{U}{U}{U}", 6, 8, "Fish")
		return c
	})

	mage.Register("Merchant Ship", func() mage.Card {
		// Merchant Ship can't attack unless defending player controls an Island.
		// Whenever Merchant Ship attacks and isn't blocked, you gain 2 life.
		// When you control no Islands, sacrifice Merchant Ship.
		c := mage.NewCreature("Merchant Ship", "{U}", 0, 2, "Human")
		return c
	})

	mage.Register("Old Man of the Sea", func() mage.Card {
		// You may choose not to untap Old Man of the Sea during your untap step.
		// {T}: Gain control of target creature with power less than or equal to
		// Old Man of the Sea's power for as long as Old Man of the Sea remains tapped
		// and that creature's power remains less than or equal to Old Man of the Sea's power.
		c := mage.NewCreature("Old Man of the Sea", "{1}{U}{U}", 2, 3, "Djinn")
		return c
	})

	mage.Register("Serendib Djinn", func() mage.Card {
		// Flying
		// At the beginning of your upkeep, sacrifice a land. If you sacrifice an
		// Island this way, Serendib Djinn deals 3 damage to you.
		// When you control no lands, sacrifice Serendib Djinn.
		c := mage.NewCreature("Serendib Djinn", "{2}{U}{U}", 5, 6, "Djinn")
		return c
	})

	mage.Register("Serendib Efreet", func() mage.Card {
		// Flying
		// At the beginning of your upkeep, Serendib Efreet deals 1 damage to you.
		c := mage.NewCreature("Serendib Efreet", "{2}{U}", 3, 4, "Efreet")
		return c
	})

	mage.Register("Sindbad", func() mage.Card {
		// {T}: Draw a card and reveal it. If it isn't a land card, discard it.
		c := mage.NewCreature("Sindbad", "{1}{U}", 1, 1, "Human")
		return c
	})

	// ===== BLACK CREATURES =====

	mage.Register("Cuombajj Witches", func() mage.Card {
		// {T}: Cuombajj Witches deals 1 damage to any target and 1 damage to any
		// target of an opponent's choice.
		c := mage.NewCreature("Cuombajj Witches", "{B}{B}", 1, 3, "Human", "Wizard")
		return c
	})

	mage.Register("El-Hajjâj", func() mage.Card {
		// Whenever El-Hajjâj deals damage, you gain that much life.
		c := mage.NewCreature("El-Hajjâj", "{1}{B}{B}", 1, 1, "Human", "Wizard")
		return c
	})

	mage.Register("Erg Raiders", func() mage.Card {
		// At the beginning of your end step, if Erg Raiders didn't attack this turn,
		// Erg Raiders deals 2 damage to you unless it came under your control this turn.
		c := mage.NewCreature("Erg Raiders", "{1}{B}", 2, 3, "Human", "Warrior")
		return c
	})

	mage.Register("Guardian Beast", func() mage.Card {
		// As long as Guardian Beast is untapped, noncreature artifacts you control
		// can't be enchanted, they have indestructible, and other players can't gain
		// control of them. This effect doesn't remove Auras already attached to
		// those artifacts.
		c := mage.NewCreature("Guardian Beast", "{3}{B}", 2, 4, "Beast")
		return c
	})

	mage.Register("Hasran Ogress", func() mage.Card {
		// Whenever Hasran Ogress attacks, it deals 3 damage to you unless you pay {2}.
		c := mage.NewCreature("Hasran Ogress", "{B}{B}", 3, 2, "Ogre")
		return c
	})

	mage.Register("Junún Efreet", func() mage.Card {
		// Flying
		// At the beginning of your upkeep, sacrifice Junún Efreet unless you pay {B}{B}.
		c := mage.NewCreature("Junún Efreet", "{1}{B}{B}", 3, 3, "Efreet")
		return c
	})

	mage.Register("Juzám Djinn", func() mage.Card {
		// At the beginning of your upkeep, Juzám Djinn deals 1 damage to you.
		c := mage.NewCreature("Juzám Djinn", "{2}{B}{B}", 5, 5, "Djinn")
		return c
	})

	mage.Register("Khabál Ghoul", func() mage.Card {
		// At the beginning of each end step, put a +1/+1 counter on Khabál Ghoul
		// for each creature that died this turn.
		c := mage.NewCreature("Khabál Ghoul", "{2}{B}", 1, 1, "Zombie")
		return c
	})

	mage.Register("Sorceress Queen", func() mage.Card {
		// {T}: Target creature other than Sorceress Queen has base power and
		// toughness 0/2 until end of turn.
		c := mage.NewCreature("Sorceress Queen", "{1}{B}{B}", 1, 1, "Human", "Wizard")
		return c
	})

	mage.Register("Stone-Throwing Devils", func() mage.Card {
		// First strike
		c := mage.NewCreature("Stone-Throwing Devils", "{B}", 1, 1, "Devil")
		return c
	})

	// ===== RED CREATURES =====

	mage.Register("Aladdin", func() mage.Card {
		// {1}{R}{R}, {T}: Gain control of target artifact for as long as you
		// control Aladdin.
		c := mage.NewCreature("Aladdin", "{2}{R}{R}", 1, 1, "Human", "Rogue")
		return c
	})

	mage.Register("Ali Baba", func() mage.Card {
		// {R}: Tap target Wall.
		c := mage.NewCreature("Ali Baba", "{R}", 1, 1, "Human", "Rogue")
		return c
	})

	mage.Register("Ali from Cairo", func() mage.Card {
		// Damage that would reduce your life total to less than 1 reduces it to 1 instead.
		c := mage.NewCreature("Ali from Cairo", "{2}{R}{R}", 0, 1, "Human")
		return c
	})

	mage.Register("Bird Maiden", func() mage.Card {
		// Flying
		c := mage.NewCreature("Bird Maiden", "{2}{R}", 1, 2, "Human", "Bird")
		return c
	})

	mage.Register("Desert Nomads", func() mage.Card {
		// Desertwalk
		// Prevent all damage that would be dealt to Desert Nomads by Deserts.
		c := mage.NewCreature("Desert Nomads", "{2}{R}", 2, 2, "Human", "Nomad")
		return c
	})

	mage.Register("Hurr Jackal", func() mage.Card {
		// {T}: Target creature can't be regenerated this turn.
		c := mage.NewCreature("Hurr Jackal", "{R}", 1, 1, "Jackal")
		return c
	})

	mage.Register("Kird Ape", func() mage.Card {
		// Kird Ape gets +1/+2 as long as you control a Forest.
		c := mage.NewCreature("Kird Ape", "{R}", 1, 1, "Ape")
		return c
	})

	mage.Register("Mijae Djinn", func() mage.Card {
		// Whenever Mijae Djinn attacks, flip a coin. If you lose the flip, remove
		// Mijae Djinn from combat and tap it.
		c := mage.NewCreature("Mijae Djinn", "{R}{R}{R}", 6, 3, "Djinn")
		return c
	})

	mage.Register("Rukh Egg", func() mage.Card {
		// When Rukh Egg dies, create a 4/4 red Bird creature token with flying at
		// the beginning of the next end step.
		c := mage.NewCreature("Rukh Egg", "{3}{R}", 0, 3, "Bird", "Egg")
		return c
	})

	mage.Register("Ydwen Efreet", func() mage.Card {
		// Whenever Ydwen Efreet blocks, flip a coin. If you lose the flip, remove
		// Ydwen Efreet from combat and it can't block this turn. Creatures it was
		// blocking that had become blocked by only Ydwen Efreet this combat become
		// unblocked.
		c := mage.NewCreature("Ydwen Efreet", "{R}{R}{R}", 3, 6, "Efreet")
		return c
	})

	// ===== GREEN CREATURES =====

	mage.Register("Erhnam Djinn", func() mage.Card {
		// At the beginning of your upkeep, target non-Wall creature an opponent
		// controls gains forestwalk until your next upkeep.
		c := mage.NewCreature("Erhnam Djinn", "{3}{G}", 4, 5, "Djinn")
		return c
	})

	mage.Register("Ghazbán Ogre", func() mage.Card {
		// At the beginning of your upkeep, if a player has more life than each other
		// player, the player with the most life gains control of Ghazbán Ogre.
		c := mage.NewCreature("Ghazbán Ogre", "{G}", 2, 2, "Ogre")
		return c
	})

	mage.Register("Ifh-Bíff Efreet", func() mage.Card {
		// Flying
		// {G}: Ifh-Bíff Efreet deals 1 damage to each creature with flying and each
		// player. Any player may activate this ability.
		c := mage.NewCreature("Ifh-Bíff Efreet", "{2}{G}{G}", 3, 3, "Efreet")
		return c
	})

	mage.Register("Nafs Asp", func() mage.Card {
		// Whenever Nafs Asp deals damage to a player, that player loses 1 life at the
		// beginning of their next draw step unless they pay {1} before that draw step.
		c := mage.NewCreature("Nafs Asp", "{G}", 1, 1, "Snake")
		return c
	})

	mage.Register("Singing Tree", func() mage.Card {
		// {T}: Target attacking creature has base power 0 until end of turn.
		c := mage.NewCreature("Singing Tree", "{3}{G}", 0, 3, "Plant")
		return c
	})

	mage.Register("Wyluli Wolf", func() mage.Card {
		// {T}: Target creature gets +1/+1 until end of turn.
		c := mage.NewCreature("Wyluli Wolf", "{1}{G}", 1, 1, "Wolf")
		return c
	})

	// ===== ARTIFACT CREATURES =====

	mage.Register("Brass Man", func() mage.Card {
		// Brass Man doesn't untap during your untap step.
		// At the beginning of your upkeep, you may pay {1}. If you do, untap Brass Man.
		c := mage.NewCreature("Brass Man", "{1}", 1, 3, "Construct")
		c.AddType(mage.TypeArtifact)
		return c
	})

	mage.Register("Dancing Scimitar", func() mage.Card {
		// Flying
		c := mage.NewCreature("Dancing Scimitar", "{4}", 1, 5, "Spirit")
		c.AddType(mage.TypeArtifact)
		return c
	})
}
