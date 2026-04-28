package fallen_empires

import (
	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/dsl"
)

func init() {
	registerEnchantments()
}

func registerEnchantments() {

	// Breeding Pit {3}{B}
	// Enchantment
	// At the beginning of your upkeep, sacrifice this enchantment unless you pay {B}{B}.
	// At the beginning of your end step, create a 0/1 black Thrull creature token.
	Register("Breeding Pit", withExpansion(func() Card {
		return NewEnchantment("Breeding Pit", "{3}{B}",
			WithAbility(SacrificeAtUpkeepUnlessPay("{B}{B}")),
			WithAbility(
				NewTriggered(EvtEndStep, false,
					CreateToken("Thrull", 0, 1, []CardType{TypeCreature}, []string{"Thrull"}),
				).SetConditionData(EventPlayerIsController{}),
			),
		)
	}))

	// Elven Fortress {G}
	// Enchantment
	// {1}{G}: Target blocking creature gets +0/+1 until end of turn.
	Register("Elven Fortress", withExpansion(func() Card {
		return NewEnchantment("Elven Fortress", "{G}",
			WithActivatedAbility(
				Boost(Fixed(0), Fixed(1)),
				ManaCostOf("{1}{G}"),
				WithTarget(TargetCreature(IsBlocking)),
			),
		)
	}))

	// Farrel's Mantle {2}{W}
	// Enchantment — Aura
	// Enchant creature
	// Whenever enchanted creature attacks and isn't blocked, its controller may have it deal damage equal to its power plus 2 to another target creature. If that player does, the attacking creature assigns no combat damage this turn.
	// TODO: implement
	Register("Farrel's Mantle", withExpansion(func() Card {
		return NewAura("Farrel's Mantle", "{2}{W}")
	}))

	// Fungal Bloom {G}{G}
	// Enchantment
	// {G}{G}: Put a spore counter on target Fungus.
	Register("Fungal Bloom", withExpansion(func() Card {
		return NewEnchantment("Fungal Bloom", "{G}{G}",
			WithActivatedAbility(
				AddCounters(Spore, Fixed(1)),
				ManaCostOf("{G}{G}"),
				WithTarget(TargetCreature(HasSubType("Fungus"))),
			),
		)
	}))

	// Goblin Kites {1}{R}
	// Enchantment
	// {R}: Target creature you control with toughness 2 or less gains flying until end of turn. Flip a coin at the beginning of the next end step. If you lose the flip, sacrifice that creature.
	// TODO: implement
	Register("Goblin Kites", withExpansion(func() Card {
		return NewEnchantment("Goblin Kites", "{1}{R}")
	}))

	// Goblin War Drums {2}{R}
	// Enchantment
	// Creatures you control have menace. (They can't be blocked except by two or more creatures.)
	Register("Goblin War Drums", withExpansion(func() Card {
		return NewEnchantment("Goblin War Drums", "{2}{R}",
			WithStaticAbility(
				GrantKeywordToControlled(Menace, AnyPermanent),
			),
		)
	}))

	// Goblin Warrens {2}{R}
	// Enchantment
	// {2}{R}, Sacrifice two Goblins: Create three 1/1 red Goblin creature tokens.
	// TODO: implement
	Register("Goblin Warrens", withExpansion(func() Card {
		return NewEnchantment("Goblin Warrens", "{2}{R}")
	}))

	// Heroism {2}{W}
	// Enchantment
	// Sacrifice a white creature: For each attacking red creature, prevent all combat damage that would be dealt by that creature this turn unless its controller pays {2}{R}.
	// TODO: implement
	Register("Heroism", withExpansion(func() Card {
		return NewEnchantment("Heroism", "{2}{W}")
	}))

	// Homarid Spawning Bed {U}{U}
	// Enchantment
	// {1}{U}{U}, Sacrifice a blue creature: Create X 1/1 blue Camarid creature tokens, where X is the sacrificed creature's mana value.
	// TODO: implement
	Register("Homarid Spawning Bed", withExpansion(func() Card {
		return NewEnchantment("Homarid Spawning Bed", "{U}{U}")
	}))

	// Merseine {2}{U}{U}
	// Enchantment — Aura
	// Enchant creature
	// This Aura enters with three net counters on it.
	// Enchanted creature doesn't untap during its controller's untap step if this Aura has a net counter on it.
	// Pay enchanted creature's mana cost: Remove a net counter from this Aura. Only the controller of the enchanted creature may activate this ability.
	// TODO: implement
	Register("Merseine", withExpansion(func() Card {
		return NewAura("Merseine", "{2}{U}{U}")
	}))

	// Night Soil {G}{G}
	// Enchantment
	// {1}, Exile two creature cards from a single graveyard: Create a 1/1 green Saproling creature token.
	// TODO: convert to pipeline — needs ExileFromGraveyard + CreateToken steps (graveyard iteration/exile not available)
	Register("Night Soil", withExpansion(func() Card {
		return NewEnchantment("Night Soil", "{G}{G}",
			WithActivatedAbility(
				FuncEffect("exile two creature cards from a single graveyard, create a 1/1 green Saproling",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						// Find a graveyard with at least 2 creature cards
						for _, player := range g.AllPlayers() {
							var creatures []Card
							for _, c := range player.Graveyard() {
								if c.HasType(TypeCreature) {
									creatures = append(creatures, c)
								}
							}
							if len(creatures) >= 2 {
								// Exile the first two creature cards
								for i := range 2 {
									if _, ok := player.RemoveFromGraveyard(creatures[i].ID()); ok {
										g.ExileCard(creatures[i], sourceID)
									}
								}
								// Create a 1/1 green Saproling creature token
								token := NewToken("Saproling", 1, 1, []CardType{TypeCreature}, []string{"Saproling"})
								token.SetOwner(controller)
								perm := g.PutOnBattlefield(token, controller)
								perm.IsToken = true
								return nil
							}
						}
						return nil
					}),
				GenericCost(1),
			),
		)
	}))

	// Raiding Party {2}{R}
	// Enchantment
	// This enchantment can't be the target of white spells or abilities from white sources.
	// Sacrifice an Orc: Each player may tap any number of untapped white creatures they control. For each creature tapped this way, that player chooses up to two Plains. Then destroy all Plains that weren't chosen this way by any player.
	// TODO: implement
	Register("Raiding Party", withExpansion(func() Card {
		return NewEnchantment("Raiding Party", "{2}{R}")
	}))

	// Thelon's Chant {1}{G}{G}
	// Enchantment
	// At the beginning of your upkeep, sacrifice this enchantment unless you pay {G}.
	// Whenever a player puts a Swamp onto the battlefield, this enchantment deals 3 damage to that player unless the player puts a -1/-1 counter on a creature they control.
	// TODO: implement
	Register("Thelon's Chant", withExpansion(func() Card {
		return NewEnchantment("Thelon's Chant", "{1}{G}{G}")
	}))

	// Thelon's Curse {G}{G}
	// Enchantment
	// Blue creatures don't untap during their controllers' untap steps.
	// At the beginning of each player's upkeep, that player may choose any number of tapped blue creatures they control and pay {U} for each creature chosen this way. If the player does, untap those creatures.
	// TODO: implement
	Register("Thelon's Curse", withExpansion(func() Card {
		return NewEnchantment("Thelon's Curse", "{G}{G}")
	}))

	// Thrull Retainer {B}
	// Enchantment — Aura
	// Enchant creature
	// Enchanted creature gets +1/+1.
	// Sacrifice this Aura: Regenerate enchanted creature.
	Register("Thrull Retainer", withExpansion(func() Card {
		return NewAura("Thrull Retainer", "{B}",
			WithStaticAbility(BoostAttached(1, 1, AttachAura)),
			WithActivatedAbility(
				Pipeline("Regenerate enchanted creature",
					EffectProperties{},
					SnapshotAttached("attached"),
					RegenerateGathered("attached"),
				),
				SacrificeSourceCost(),
			),
		)
	}))

	// Tidal Flats {U}
	// Enchantment
	// {U}{U}: For each attacking creature without flying, its controller may pay {1}. If that player doesn't, creatures you control blocking that creature gain first strike until end of turn.
	// TODO: implement
	Register("Tidal Flats", withExpansion(func() Card {
		return NewEnchantment("Tidal Flats", "{U}")
	}))

	// Tidal Influence {2}{U}
	// Enchantment
	// Cast this spell only if no permanents named Tidal Influence are on the battlefield.
	// This enchantment enters with a tide counter on it.
	// At the beginning of your upkeep, put a tide counter on this enchantment.
	// As long as there is exactly one tide counter on this enchantment, all blue creatures get -2/-0.
	// As long as there are exactly three tide counters on this enchantment, all blue creatures get +2/+0.
	// Whenever there are four or more tide counters on this enchantment, remove all tide counters from it.
	// TODO: implement
	Register("Tidal Influence", withExpansion(func() Card {
		return NewEnchantment("Tidal Influence", "{2}{U}")
	}))

	// Tourach's Chant {1}{B}{B}
	// Enchantment
	// At the beginning of your upkeep, sacrifice this enchantment unless you pay {B}.
	// Whenever a player puts a Forest onto the battlefield, this enchantment deals 3 damage to that player unless they put a -1/-1 counter on a creature they control.
	// TODO: implement
	Register("Tourach's Chant", withExpansion(func() Card {
		return NewEnchantment("Tourach's Chant", "{1}{B}{B}")
	}))

	// Tourach's Gate {1}{B}{B}
	// Enchantment — Aura
	// Enchant land you control
	// Sacrifice a Thrull: Put three time counters on this Aura.
	// At the beginning of your upkeep, remove a time counter from this Aura. If there are no time counters on this Aura, sacrifice it.
	// Tap enchanted land: Attacking creatures you control get +2/-1 until end of turn. Activate only if enchanted land is untapped.
	// TODO: implement
	Register("Tourach's Gate", withExpansion(func() Card {
		return NewAura("Tourach's Gate", "{1}{B}{B}")
	}))

}
