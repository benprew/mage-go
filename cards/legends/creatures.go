package legends

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

// Akron Legionnaire {6}{W}{W}
// Creature — Giant Soldier
// 8/4
// Except for creatures named Akron Legionnaire and artifact creatures, creatures you control can't attack.
	Register("Akron Legionnaire", withExpansion(func() Card {
		return NewCreature("Akron Legionnaire", "{6}{W}{W}", 8, 4,
			WithSubTypes("Giant", "Soldier"),
			WithStaticAbility(
				FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					for _, p := range g.Battlefield {
						if p.Controller != src.Controller {
							continue
						}
						if !p.HasType(TypeCreature) {
							continue
						}
						if p.Card.Name() == "Akron Legionnaire" {
							continue
						}
						if p.HasType(TypeArtifact) {
							continue
						}
						g.Effects.RevokeAttr(p.ID(), AttrCanAttack)
					}
					return nil
				}),
			),
		)
	}))

// Amrou Kithkin {W}{W}
// Creature — Kithkin
// 1/1
// This creature can't be blocked by creatures with power 3 or greater.
// TODO: implement — needs engine support for "can't be blocked by creatures with power >= 3" restriction
	Register("Amrou Kithkin", withExpansion(func() Card {
		return NewCreature("Amrou Kithkin", "{W}{W}", 1, 1,
			WithSubTypes("Kithkin"),
		)
	}))

// Clergy of the Holy Nimbus {W}
// Creature — Human Cleric
// 1/1
// If this creature would be destroyed, regenerate it.
// {1}: This creature can't be regenerated this turn. Only your opponents may activate this ability.
// TODO: implement — needs engine support for auto-regeneration replacement effect and opponent-only activated ability
	Register("Clergy of the Holy Nimbus", withExpansion(func() Card {
		return NewCreature("Clergy of the Holy Nimbus", "{W}", 1, 1,
			WithSubTypes("Human", "Cleric"),
		)
	}))

// D'Avenant Archer {2}{W}
// Creature — Human Soldier Archer
// 1/2
// {T}: This creature deals 1 damage to target attacking or blocking creature.
	Register("D'Avenant Archer", withExpansion(func() Card {
		return NewCreature("D'Avenant Archer", "{2}{W}", 1, 2,
			WithSubTypes("Human", "Soldier", "Archer"),
			WithActivatedAbility(
				DealDamage(Fixed(1)),
				TapSourceCost(),
				WithTarget(TargetCreature(Or(IsAttacking, IsBlocking))),
			),
		)
	}))

// Elder Land Wurm {4}{W}{W}{W}
// Creature — Dragon Wurm
// 5/5
// Defender, trample
// When this creature blocks, it loses defender.
	Register("Elder Land Wurm", withExpansion(func() Card {
		return NewCreature("Elder Land Wurm", "{4}{W}{W}{W}", 5, 5,
			WithSubTypes("Dragon", "Wurm"),
			WithKeyword(Defender),
			WithKeyword(Trample),
			WithAbility(BlocksTrigger(FuncEffect("lose defender", EffectProperties{Outcome: OutcomeBenefit}, func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
				perm := g.FindPermanent(sourceID)
				if perm != nil {
					perm.RevokeBaseAttr(Defender)
				}
				return nil
			}), false)),
		)
	}))

// Enchanted Being {1}{W}{W}
// Creature — Human
// 2/2
// Prevent all combat damage that would be dealt to this creature by enchanted creatures.
// TODO: implement — needs engine support for preventing combat damage from enchanted creatures specifically
	Register("Enchanted Being", withExpansion(func() Card {
		return NewCreature("Enchanted Being", "{1}{W}{W}", 2, 2,
			WithSubTypes("Human"),
		)
	}))

// Ivory Guardians {4}{W}{W}
// Creature — Giant Cleric
// 3/3
// Protection from red
// Creatures named Ivory Guardians get +1/+1 as long as an opponent controls a nontoken red permanent.
	Register("Ivory Guardians", withExpansion(func() Card {
		return NewCreature("Ivory Guardians", "{4}{W}{W}", 3, 3,
			WithSubTypes("Giant", "Cleric"),
			WithAbility(ProtectionFromColor(Red)),
			WithStaticAbility(
				FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					// Check if an opponent controls a nontoken red permanent
					opponentHasRed := false
					for _, p := range g.Battlefield {
						if p.Controller == src.Controller {
							continue
						}
						if p.Card.(*BaseCard).IsToken() {
							continue
						}
						if HasColorFilter(Red).Match(p, g) {
							opponentHasRed = true
							break
						}
					}
					if !opponentHasRed {
						return nil
					}
					// Boost all creatures named Ivory Guardians (including self)
					for _, p := range g.Battlefield {
						if p.HasType(TypeCreature) && p.Card.Name() == "Ivory Guardians" {
							p.BoostPT(1, 1)
						}
					}
					return nil
				}),
			),
		)
	}))

// Keepers of the Faith {1}{W}{W}
// Creature — Human Cleric
// 2/3
	Register("Keepers of the Faith", withExpansion(func() Card {
		return NewCreature("Keepers of the Faith", "{1}{W}{W}", 2, 3,
			WithSubTypes("Human", "Cleric"),
		)
	}))

// Osai Vultures {1}{W}
// Creature — Bird
// 1/1
// Flying
// At the beginning of each end step, if a creature died this turn, put a carrion counter on this creature.
// Remove two carrion counters from this creature: This creature gets +1/+1 until end of turn.
// TODO: implement — needs engine support for end-step trigger checking "creature died this turn" condition
	Register("Osai Vultures", withExpansion(func() Card {
		return NewCreature("Osai Vultures", "{1}{W}", 1, 1,
			WithSubTypes("Bird"),
			WithKeyword(Flying),
			WithActivatedAbility(
				BoostUntilEndOfTurn(Fixed(1), Fixed(1), SelectSource),
				RemoveCountersCost(Carrion, 2),
			),
		)
	}))

// Petra Sphinx {2}{W}{W}{W}
// Creature — Sphinx
// 3/4
// {T}: Target player chooses a card name, then reveals the top card of their library. If that card has the chosen name, that player puts it into their hand. If it doesn't, the player puts it into their graveyard.
// TODO: implement — needs engine support for card naming and reveal
	Register("Petra Sphinx", withExpansion(func() Card {
		return NewCreature("Petra Sphinx", "{2}{W}{W}{W}", 3, 4,
			WithSubTypes("Sphinx"),
		)
	}))

// Righteous Avengers {4}{W}
// Creature — Human Soldier
// 3/1
// Plainswalk (This creature can't be blocked as long as defending player controls a Plains.)
	Register("Righteous Avengers", withExpansion(func() Card {
		return NewCreature("Righteous Avengers", "{4}{W}", 3, 1,
			WithSubTypes("Human", "Soldier"),
			WithKeyword(Plainswalk),
		)
	}))

// Thunder Spirit {1}{W}{W}
// Creature — Elemental Spirit
// 2/2
// Flying, first strike
	Register("Thunder Spirit", withExpansion(func() Card {
		return NewCreature("Thunder Spirit", "{1}{W}{W}", 2, 2,
			WithSubTypes("Elemental", "Spirit"),
			WithKeyword(Flying),
			WithKeyword(FirstStrike),
		)
	}))

// Tundra Wolves {W}
// Creature — Wolf
// 1/1
// First strike (This creature deals combat damage before creatures without first strike.)
	Register("Tundra Wolves", withExpansion(func() Card {
		return NewCreature("Tundra Wolves", "{W}", 1, 1,
			WithSubTypes("Wolf"),
			WithKeyword(FirstStrike),
		)
	}))

// Wall of Caltrops {1}{W}
// Creature — Wall
// 2/1
// Defender (This creature can't attack.)
// Whenever this creature blocks a creature, if at least one other Wall creature is blocking that creature and no non-Wall creatures are blocking that creature, this creature gains banding until end of turn. (If any creatures with banding you control are blocking a creature, you divide that creature's combat damage, not its controller, among any of the creatures it's being blocked by.)
// TODO: implement — needs engine support for conditional banding on block with Wall-only check
	Register("Wall of Caltrops", withExpansion(func() Card {
		return NewCreature("Wall of Caltrops", "{1}{W}", 2, 1,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
		)
	}))

// Wall of Light {2}{W}
// Creature — Wall
// 1/5
// Defender (This creature can't attack.)
// Protection from black
	Register("Wall of Light", withExpansion(func() Card {
		return NewCreature("Wall of Light", "{2}{W}", 1, 5,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
			WithAbility(ProtectionFromColor(Black)),
		)
	}))


	// ===== BLUE CREATURES =====

// Azure Drake {3}{U}
// Creature — Drake
// 2/4
// Flying
	Register("Azure Drake", withExpansion(func() Card {
		return NewCreature("Azure Drake", "{3}{U}", 2, 4,
			WithSubTypes("Drake"),
			WithKeyword(Flying),
		)
	}))

// Brine Hag {2}{U}{U}
// Creature — Hag
// 2/2
// When this creature dies, change the base power and toughness of all creatures that dealt damage to it this turn to 0/2. (This effect lasts indefinitely.)
// TODO: implement — needs engine support for tracking "creatures that dealt damage to this" this turn
	Register("Brine Hag", withExpansion(func() Card {
		return NewCreature("Brine Hag", "{2}{U}{U}", 2, 2,
			WithSubTypes("Hag"),
		)
	}))

// Devouring Deep {2}{U}
// Creature — Fish
// 1/2
// Islandwalk (This creature can't be blocked as long as defending player controls an Island.)
	Register("Devouring Deep", withExpansion(func() Card {
		return NewCreature("Devouring Deep", "{2}{U}", 1, 2,
			WithSubTypes("Fish"),
			WithKeyword(Islandwalk),
		)
	}))

// Elder Spawn {4}{U}{U}{U}
// Creature — Spawn
// 6/6
// At the beginning of your upkeep, unless you sacrifice an Island, sacrifice this creature and it deals 6 damage to you.
// This creature can't be blocked by red creatures.
// TODO: implement — needs engine support for "sacrifice a land" upkeep cost and "can't be blocked by red creatures"
	Register("Elder Spawn", withExpansion(func() Card {
		return NewCreature("Elder Spawn", "{4}{U}{U}{U}", 6, 6,
			WithSubTypes("Spawn"),
		)
	}))

// Psionic Entity {4}{U}
// Creature — Illusion
// 2/2
// {T}: This creature deals 2 damage to any target and 3 damage to itself.
	Register("Psionic Entity", withExpansion(func() Card {
		return NewCreature("Psionic Entity", "{4}{U}", 2, 2,
			WithSubTypes("Illusion"),
			WithActivatedAbility(
				FuncEffect("deal 2 damage to any target and 3 damage to self",
					EffectProperties{Outcome: OutcomeDetriment, DamageValue: Fixed(2)},
					func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						// Deal 2 damage to target
						if len(targets) > 0 {
							if perm := g.FindPermanent(targets[0]); perm != nil {
								g.DealDamageToPermanent(perm, 2, sourceID)
							} else if p := g.GetPlayer(targets[0]); p != nil {
								g.DealDamageToPlayer(p, 2, sourceID)
							}
						}
						// Deal 3 damage to self
						if self := g.FindPermanent(sourceID); self != nil {
							g.DealDamageToPermanent(self, 3, sourceID)
						}
						return nil
					},
				),
				TapSourceCost(),
				WithTarget(TargetAnyTarget()),
			),
		)
	}))

// Segovian Leviathan {4}{U}
// Creature — Leviathan
// 3/3
// Islandwalk (This creature can't be blocked as long as defending player controls an Island.)
	Register("Segovian Leviathan", withExpansion(func() Card {
		return NewCreature("Segovian Leviathan", "{4}{U}", 3, 3,
			WithSubTypes("Leviathan"),
			WithKeyword(Islandwalk),
		)
	}))

// Time Elemental {2}{U}
// Creature — Elemental
// 0/2
// When this creature attacks or blocks, at end of combat, sacrifice it and it deals 5 damage to you.
// {2}{U}{U}, {T}: Return target permanent that isn't enchanted to its owner's hand.
// TODO: implement — needs engine support for end-of-combat delayed sacrifice+damage trigger and "not enchanted" filter
	Register("Time Elemental", withExpansion(func() Card {
		return NewCreature("Time Elemental", "{2}{U}", 0, 2,
			WithSubTypes("Elemental"),
		)
	}))

// Wall of Vapor {3}{U}
// Creature — Wall
// 0/1
// Defender (This creature can't attack.)
// Prevent all damage that would be dealt to this creature by creatures it's blocking.
// TODO: implement — needs engine support for preventing damage from creatures this is blocking
	Register("Wall of Vapor", withExpansion(func() Card {
		return NewCreature("Wall of Vapor", "{3}{U}", 0, 1,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
		)
	}))

// Wall of Wonder {2}{U}{U}
// Creature — Wall
// 1/5
// Defender (This creature can't attack.)
// {2}{U}{U}: This creature gets +4/-4 until end of turn and can attack this turn as though it didn't have defender.
// TODO: implement — needs engine support for "can attack as though it didn't have defender" override
	Register("Wall of Wonder", withExpansion(func() Card {
		return NewCreature("Wall of Wonder", "{2}{U}{U}", 1, 5,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
		)
	}))

// Zephyr Falcon {1}{U}
// Creature — Bird
// 1/1
// Flying, vigilance
	Register("Zephyr Falcon", withExpansion(func() Card {
		return NewCreature("Zephyr Falcon", "{1}{U}", 1, 1,
			WithSubTypes("Bird"),
			WithKeyword(Flying),
			WithKeyword(Vigilance),
		)
	}))


	// ===== BLACK CREATURES =====

// Abomination {3}{B}{B}
// Creature — Horror
// 2/6
// Whenever this creature blocks or becomes blocked by a green or white creature, destroy that creature at end of combat.
// TODO: implement — needs engine support for "blocks or becomes blocked by" trigger with delayed destroy at end of combat
	Register("Abomination", withExpansion(func() Card {
		return NewCreature("Abomination", "{3}{B}{B}", 2, 6,
			WithSubTypes("Horror"),
		)
	}))

// Carrion Ants {2}{B}{B}
// Creature — Insect
// 0/1
// {1}: This creature gets +1/+1 until end of turn.
	Register("Carrion Ants", withExpansion(func() Card {
		return NewCreature("Carrion Ants", "{2}{B}{B}", 0, 1,
			WithSubTypes("Insect"),
			WithActivatedAbility(
				BoostUntilEndOfTurn(Fixed(1), Fixed(1), SelectSource),
				GenericCost(1),
			),
		)
	}))

// Cosmic Horror {3}{B}{B}{B}
// Creature — Horror
// 7/7
// First strike
// At the beginning of your upkeep, destroy this creature unless you pay {3}{B}{B}{B}. If this creature is destroyed this way, it deals 7 damage to you.
// Note: The "deals 7 damage" if destroyed portion is not implemented (SacrificeAtUpkeepUnlessPay just sacrifices)
	Register("Cosmic Horror", withExpansion(func() Card {
		return NewCreature("Cosmic Horror", "{3}{B}{B}{B}", 7, 7,
			WithSubTypes("Horror"),
			WithKeyword(FirstStrike),
			WithAbility(SacrificeAtUpkeepUnlessPay("{3}{B}{B}{B}")),
		)
	}))

// Cyclopean Mummy {1}{B}
// Creature — Zombie
// 2/1
// When this creature dies, exile it.
	Register("Cyclopean Mummy", withExpansion(func() Card {
		return NewCreature("Cyclopean Mummy", "{1}{B}", 2, 1,
			WithSubTypes("Zombie"),
			WithAbility(NewTriggered(EvtCreatureDied, false, FuncEffect("exile this", EffectProperties{}, func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
				p := g.GetPlayer(controller)
				if p == nil {
					return nil
				}
				card, ok := p.RemoveFromGraveyard(sourceID)
				if card != nil && ok {
					g.ExileCard(card, sourceID)
				}
				return nil
			})).SetCondition(IsThisSource)),
		)
	}))

// Evil Eye of Orms-by-Gore {4}{B}
// Creature — Eye
// 3/6
// Non-Eye creatures you control can't attack.
// This creature can't be blocked except by Walls.
	Register("Evil Eye of Orms-by-Gore", withExpansion(func() Card {
		return NewCreature("Evil Eye of Orms-by-Gore", "{4}{B}", 3, 6,
			WithSubTypes("Eye"),
			WithKeyword(CantBeBlockedExceptByWalls),
			WithStaticAbility(
				FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					for _, p := range g.Battlefield {
						if p.Controller != src.Controller {
							continue
						}
						if !p.HasType(TypeCreature) {
							continue
						}
						if p.HasSubType("Eye") {
							continue
						}
						g.Effects.RevokeAttr(p.ID(), AttrCanAttack)
					}
					return nil
				}),
			),
		)
	}))

// Fallen Angel {3}{B}{B}
// Creature — Angel
// 3/3
// Flying
// Sacrifice a creature: This creature gets +2/+1 until end of turn.
	Register("Fallen Angel", withExpansion(func() Card {
		return NewCreature("Fallen Angel", "{3}{B}{B}", 3, 3,
			WithSubTypes("Angel"),
			WithKeyword(Flying),
			WithActivatedAbility(
				BoostUntilEndOfTurn(Fixed(2), Fixed(1), SelectSource),
				SacrificeCreatureCost(),
			),
		)
	}))

// Ghosts of the Damned {1}{B}{B}
// Creature — Spirit
// 0/2
// {T}: Target creature gets -1/-0 until end of turn.
	Register("Ghosts of the Damned", withExpansion(func() Card {
		return NewCreature("Ghosts of the Damned", "{1}{B}{B}", 0, 2,
			WithSubTypes("Spirit"),
			WithActivatedAbility(
				BoostUntilEndOfTurn(Fixed(-1), Fixed(0), SelectTarget),
				TapSourceCost(),
				WithTarget(TargetCreature()),
			),
		)
	}))

// Giant Slug {1}{B}
// Creature — Slug
// 1/1
// {5}: At the beginning of your next upkeep, choose a basic land type. This creature gains landwalk of the chosen type until the end of that turn. (It can't be blocked as long as defending player controls a land of that type.)
// TODO: implement — needs engine support for delayed upkeep trigger with land type choice and landwalk grant
	Register("Giant Slug", withExpansion(func() Card {
		return NewCreature("Giant Slug", "{1}{B}", 1, 1,
			WithSubTypes("Slug"),
		)
	}))

// Headless Horseman {2}{B}
// Creature — Zombie Knight
// 2/2
	Register("Headless Horseman", withExpansion(func() Card {
		return NewCreature("Headless Horseman", "{2}{B}", 2, 2,
			WithSubTypes("Zombie", "Knight"),
		)
	}))

// Hell's Caretaker {3}{B}
// Creature — Horror
// 1/1
// {T}, Sacrifice a creature: Return target creature card from your graveyard to the battlefield. Activate only during your upkeep.
	Register("Hell's Caretaker", withExpansion(func() Card {
		return NewCreature("Hell's Caretaker", "{3}{B}", 1, 1,
			WithSubTypes("Horror"),
			WithActivatedAbility(
				ReturnFromGraveyardToBattlefield(),
				TapSourceCost(),
				WithCost(SacrificeCreatureCost()),
				WithTarget(TargetCreatureInYourGraveyard()),
				WithUpkeepOnly(),
			),
		)
	}))

// Infernal Medusa {3}{B}{B}
// Creature — Gorgon
// 2/4
// Whenever this creature blocks a creature, destroy that creature at end of combat.
// Whenever this creature becomes blocked by a non-Wall creature, destroy that creature at end of combat.
// TODO: implement — needs engine support for "becomes blocked by" trigger and delayed end-of-combat destroy
	Register("Infernal Medusa", withExpansion(func() Card {
		return NewCreature("Infernal Medusa", "{3}{B}{B}", 2, 4,
			WithSubTypes("Gorgon"),
		)
	}))

// Lesser Werewolf {3}{B}
// Creature — Werewolf
// 2/4
// {B}: If this creature's power is 1 or more, it gets -1/-0 until end of turn and put a -0/-1 counter on target creature blocking or blocked by this creature. Activate only during the declare blockers step.
// TODO: implement — needs engine support for "blocking or blocked by this creature" targeting and declare blockers step restriction
	Register("Lesser Werewolf", withExpansion(func() Card {
		return NewCreature("Lesser Werewolf", "{3}{B}", 2, 4,
			WithSubTypes("Werewolf"),
		)
	}))

// Lost Soul {1}{B}{B}
// Creature — Spirit Minion
// 2/1
// Swampwalk (This creature can't be blocked as long as defending player controls a Swamp.)
	Register("Lost Soul", withExpansion(func() Card {
		return NewCreature("Lost Soul", "{1}{B}{B}", 2, 1,
			WithSubTypes("Spirit", "Minion"),
			WithKeyword(Swampwalk),
		)
	}))

// Mold Demon {5}{B}{B}
// Creature — Fungus Demon
// 6/6
// When this creature enters, sacrifice it unless you sacrifice two Swamps.
// TODO: implement — needs engine support for "sacrifice two Swamps" ETB cost
	Register("Mold Demon", withExpansion(func() Card {
		return NewCreature("Mold Demon", "{5}{B}{B}", 6, 6,
			WithSubTypes("Fungus", "Demon"),
		)
	}))

// Pit Scorpion {2}{B}
// Creature — Scorpion
// 1/1
// Whenever this creature deals damage to a player, that player gets a poison counter. (A player with ten or more poison counters loses the game.)
	Register("Pit Scorpion", withExpansion(func() Card {
		return NewCreature("Pit Scorpion", "{2}{B}", 1, 1,
			WithSubTypes("Scorpion"),
			WithAbility(NewTriggered(EvtDamageDealt, false, FuncEffect(
				"poison counter",
				EffectProperties{},
				func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					// Give a poison counter to the player who took damage
					// The event PlayerID holds the damaged player's ID
					for _, p := range g.AllPlayers() {
						if p.PlayerID() != controller {
							p.AddPoisonCounters(1)
							return nil
						}
					}
					return nil
				},
			)).SetCondition(func(evt *GameEvent, g *Game, sourceID, _ uuid.UUID) bool {
				// Only trigger when this creature deals damage to a player
				return evt.SourceID == sourceID && evt.PlayerID != uuid.Nil
			})),
		)
	}))

// Shimian Night Stalker {3}{B}{B}
// Creature — Nightstalker
// 4/4
// {B}, {T}: All damage that would be dealt to you this turn by target attacking creature is dealt to this creature instead.
// TODO: implement — needs engine support for per-creature damage redirection
	Register("Shimian Night Stalker", withExpansion(func() Card {
		return NewCreature("Shimian Night Stalker", "{3}{B}{B}", 4, 4,
			WithSubTypes("Nightstalker"),
		)
	}))

// The Wretched {3}{B}{B}
// Creature — Demon
// 2/5
// At end of combat, gain control of all creatures blocking this creature for as long as you control this creature.
// TODO: implement — needs engine support for end-of-combat control change of blockers
	Register("The Wretched", withExpansion(func() Card {
		return NewCreature("The Wretched", "{3}{B}{B}", 2, 5,
			WithSubTypes("Demon"),
		)
	}))

// Vampire Bats {B}
// Creature — Bat
// 0/1
// Flying (This creature can't be blocked except by creatures with flying or reach.)
// {B}: This creature gets +1/+0 until end of turn. Activate no more than twice each turn.
// TODO: twice-per-turn activation limit not yet enforced
	Register("Vampire Bats", withExpansion(func() Card {
		return NewCreature("Vampire Bats", "{B}", 0, 1,
			WithSubTypes("Bat"),
			WithKeyword(Flying),
			WithActivatedAbility(
				BoostUntilEndOfTurn(Fixed(1), Fixed(0), SelectSource),
				ManaCostOf("{B}"),
			),
		)
	}))

// Walking Dead {1}{B}
// Creature — Zombie
// 1/1
// {B}: Regenerate this creature.
	Register("Walking Dead", withExpansion(func() Card {
		return NewCreature("Walking Dead", "{1}{B}", 1, 1,
			WithSubTypes("Zombie"),
			WithActivatedAbility(
				RegenerateSource(),
				ManaCostOf("{B}"),
			),
		)
	}))

// Wall of Putrid Flesh {2}{B}
// Creature — Wall
// 2/4
// Defender (This creature can't attack.)
// Protection from white
// Prevent all damage that would be dealt to this creature by enchanted creatures.
// Note: "prevent damage from enchanted creatures" is not implemented (needs engine support)
	Register("Wall of Putrid Flesh", withExpansion(func() Card {
		return NewCreature("Wall of Putrid Flesh", "{2}{B}", 2, 4,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
			WithAbility(ProtectionFromColor(White)),
		)
	}))

// Wall of Shadows {1}{B}{B}
// Creature — Wall
// 0/1
// Defender (This creature can't attack.)
// Prevent all damage that would be dealt to this creature by creatures it's blocking.
// This creature can't be the target of spells that can target only Walls or of abilities that can target only Walls.
// TODO: implement — needs engine support for preventing damage from blocked creatures and Wall-only targeting restriction
	Register("Wall of Shadows", withExpansion(func() Card {
		return NewCreature("Wall of Shadows", "{1}{B}{B}", 0, 1,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
		)
	}))

// Wall of Tombstones {1}{B}
// Creature — Wall
// 0/1
// Defender (This creature can't attack.)
// At the beginning of your upkeep, change this creature's base toughness to 1 plus the number of creature cards in your graveyard. (This effect lasts indefinitely.)
// TODO: implement — needs engine support for setting base toughness indefinitely (not just until EOT)
	Register("Wall of Tombstones", withExpansion(func() Card {
		return NewCreature("Wall of Tombstones", "{1}{B}", 0, 1,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
		)
	}))


	// ===== RED CREATURES =====

// Aerathi Berserker {2}{R}{R}{R}
// Creature — Human Berserker
// 2/4
// Rampage 3 (Whenever this creature becomes blocked, it gets +3/+3 until end of turn for each creature blocking it beyond the first.)
	Register("Aerathi Berserker", withExpansion(func() Card {
		return NewCreature("Aerathi Berserker", "{2}{R}{R}{R}", 2, 4,
			WithSubTypes("Human", "Berserker"),
			WithAbility(RampageTrigger(3)),
		)
	}))

// Beasts of Bogardan {4}{R}
// Creature — Beast
// 3/3
// Protection from red
// This creature gets +1/+1 as long as an opponent controls a nontoken white permanent.
	Register("Beasts of Bogardan", withExpansion(func() Card {
		return NewCreature("Beasts of Bogardan", "{4}{R}", 3, 3,
			WithSubTypes("Beast"),
			WithAbility(ProtectionFromColor(Red)),
			WithStaticAbility(
				FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					for _, p := range g.Battlefield {
						if p.Controller == src.Controller {
							continue
						}
						if p.Card.(*BaseCard).IsToken() {
							continue
						}
						if HasColorFilter(White).Match(p, g) {
							src.BoostPT(1, 1)
							return nil
						}
					}
					return nil
				}),
			),
		)
	}))

// Blazing Effigy {1}{R}
// Creature — Elemental
// 0/3
// When this creature dies, it deals X damage to target creature, where X is 3 plus the amount of damage dealt to this creature this turn by other sources named Blazing Effigy.
// TODO: implement — needs engine support for tracking damage dealt to this by named sources
	Register("Blazing Effigy", withExpansion(func() Card {
		return NewCreature("Blazing Effigy", "{1}{R}", 0, 3,
			WithSubTypes("Elemental"),
		)
	}))

// Crimson Kobolds {0}
// Creature — Kobold
// 0/1
	Register("Crimson Kobolds", withExpansion(func() Card {
		return NewCreature("Crimson Kobolds", "{0}", 0, 1,
			WithSubTypes("Kobold"),
		)
	}))

// Crimson Manticore {2}{R}{R}
// Creature — Manticore
// 2/2
// Flying
// {R}, {T}: This creature deals 1 damage to target attacking or blocking creature.
	Register("Crimson Manticore", withExpansion(func() Card {
		return NewCreature("Crimson Manticore", "{2}{R}{R}", 2, 2,
			WithSubTypes("Manticore"),
			WithKeyword(Flying),
			WithActivatedAbility(
				DealDamage(Fixed(1)),
				ManaCostOf("{R}"),
				WithCost(TapSourceCost()),
				WithTarget(TargetCreature(Or(IsAttacking, IsBlocking))),
			),
		)
	}))

// Crookshank Kobolds {0}
// Creature — Kobold
// 0/1
	Register("Crookshank Kobolds", withExpansion(func() Card {
		return NewCreature("Crookshank Kobolds", "{0}", 0, 1,
			WithSubTypes("Kobold"),
		)
	}))

// Firestorm Phoenix {4}{R}{R}
// Creature — Phoenix
// 3/2
// Flying
// If this creature would die, return it to its owner's hand instead. Until that player's next turn, that player plays with that card revealed in their hand and can't play it.
// TODO: implement — needs engine support for die replacement effect (return to hand instead) and cast restriction
	Register("Firestorm Phoenix", withExpansion(func() Card {
		return NewCreature("Firestorm Phoenix", "{4}{R}{R}", 3, 2,
			WithSubTypes("Phoenix"),
			WithKeyword(Flying),
		)
	}))

// Frost Giant {3}{R}{R}{R}
// Creature — Giant
// 4/4
// Rampage 2 (Whenever this creature becomes blocked, it gets +2/+2 until end of turn for each creature blocking it beyond the first.)
	Register("Frost Giant", withExpansion(func() Card {
		return NewCreature("Frost Giant", "{3}{R}{R}{R}", 4, 4,
			WithSubTypes("Giant"),
			WithAbility(RampageTrigger(2)),
		)
	}))

// Hyperion Blacksmith {1}{R}{R}
// Creature — Human Artificer
// 2/2
// {T}: You may tap or untap target artifact an opponent controls.
	Register("Hyperion Blacksmith", withExpansion(func() Card {
		return NewCreature("Hyperion Blacksmith", "{1}{R}{R}", 2, 2,
			WithSubTypes("Human", "Artificer"),
			WithActivatedAbility(
				TapOrUntapTarget(),
				TapSourceCost(),
				WithTarget(TargetPermanent(IsArtifact)),
			),
		)
	}))

// Kobold Drill Sergeant {1}{R}
// Creature — Kobold Soldier
// 1/2
// Other Kobold creatures you control get +0/+1 and have trample.
	Register("Kobold Drill Sergeant", withExpansion(func() Card {
		return NewCreature("Kobold Drill Sergeant", "{1}{R}", 1, 2,
			WithSubTypes("Kobold", "Soldier"),
			WithStaticAbility(
				FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					for _, p := range g.Battlefield {
						if p.ID() == sourceID || p.Controller != src.Controller {
							continue
						}
						if !p.HasType(TypeCreature) || !p.HasSubType("Kobold") {
							continue
						}
						p.BoostPT(0, 1)
					}
					return nil
				}),
				FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					for _, p := range g.Battlefield {
						if p.ID() == sourceID || p.Controller != src.Controller {
							continue
						}
						if !p.HasType(TypeCreature) || !p.HasSubType("Kobold") {
							continue
						}
						g.Effects.GrantAttr(p.ID(), Trample)
					}
					return nil
				}),
			),
		)
	}))

// Kobold Overlord {1}{R}
// Creature — Kobold
// 1/2
// First strike
// Other Kobold creatures you control have first strike.
	Register("Kobold Overlord", withExpansion(func() Card {
		return NewCreature("Kobold Overlord", "{1}{R}", 1, 2,
			WithSubTypes("Kobold"),
			WithKeyword(FirstStrike),
			WithStaticAbility(
				FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					for _, p := range g.Battlefield {
						if p.ID() == sourceID || p.Controller != src.Controller {
							continue
						}
						if !p.HasType(TypeCreature) || !p.HasSubType("Kobold") {
							continue
						}
						g.Effects.GrantAttr(p.ID(), FirstStrike)
					}
					return nil
				}),
			),
		)
	}))

// Kobold Taskmaster {1}{R}
// Creature — Kobold
// 1/2
// Other Kobold creatures you control get +1/+0.
	Register("Kobold Taskmaster", withExpansion(func() Card {
		return NewCreature("Kobold Taskmaster", "{1}{R}", 1, 2,
			WithSubTypes("Kobold"),
			WithStaticAbility(
				FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					for _, p := range g.Battlefield {
						if p.ID() == sourceID || p.Controller != src.Controller {
							continue
						}
						if !p.HasType(TypeCreature) || !p.HasSubType("Kobold") {
							continue
						}
						p.BoostPT(1, 0)
					}
					return nil
				}),
			),
		)
	}))

// Kobolds of Kher Keep {0}
// Creature — Kobold
// 0/1
	Register("Kobolds of Kher Keep", withExpansion(func() Card {
		return NewCreature("Kobolds of Kher Keep", "{0}", 0, 1,
			WithSubTypes("Kobold"),
		)
	}))

// Mountain Yeti {2}{R}{R}
// Creature — Yeti
// 3/3
// Mountainwalk (This creature can't be blocked as long as defending player controls a Mountain.)
// Protection from white
	Register("Mountain Yeti", withExpansion(func() Card {
		return NewCreature("Mountain Yeti", "{2}{R}{R}", 3, 3,
			WithSubTypes("Yeti"),
			WithKeyword(Mountainwalk),
			WithAbility(ProtectionFromColor(White)),
		)
	}))

// Primordial Ooze {R}
// Creature — Ooze
// 1/1
// This creature attacks each combat if able.
// At the beginning of your upkeep, put a +1/+1 counter on this creature. Then you may pay {X}, where X is the number of +1/+1 counters on it. If you don't, tap this creature and it deals X damage to you.
// TODO: implement — needs engine support for mandatory attack and complex upkeep pay-or-damage trigger
	Register("Primordial Ooze", withExpansion(func() Card {
		return NewCreature("Primordial Ooze", "{R}", 1, 1,
			WithSubTypes("Ooze"),
		)
	}))

// Quarum Trench Gnomes {3}{R}
// Creature — Gnome
// 1/1
// {T}: If target Plains is tapped for mana, it produces colorless mana instead of white mana. (This effect lasts indefinitely.)
// TODO: implement — needs engine support for mana production replacement on specific lands
	Register("Quarum Trench Gnomes", withExpansion(func() Card {
		return NewCreature("Quarum Trench Gnomes", "{3}{R}", 1, 1,
			WithSubTypes("Gnome"),
		)
	}))

// Raging Bull {2}{R}
// Creature — Ox
// 2/2
	Register("Raging Bull", withExpansion(func() Card {
		return NewCreature("Raging Bull", "{2}{R}", 2, 2,
			WithSubTypes("Ox"),
		)
	}))

// Spinal Villain {2}{R}
// Creature — Beast
// 1/2
// {T}: Destroy target blue creature.
	Register("Spinal Villain", withExpansion(func() Card {
		return NewCreature("Spinal Villain", "{2}{R}", 1, 2,
			WithSubTypes("Beast"),
			WithActivatedAbility(
				DestroyTarget(),
				TapSourceCost(),
				WithTarget(TargetCreature(HasColorFilter(Blue))),
			),
		)
	}))

// Tempest Efreet {1}{R}{R}{R}
// Creature — Efreet
// 3/3
// Remove this card from your deck before playing if you're not playing for ante.
// {T}, Sacrifice this creature: Target opponent may pay 10 life. If that player doesn't, they reveal a card at random from their hand. Exchange ownership of the revealed card and Tempest Efreet. Put the revealed card into your hand and Tempest Efreet from anywhere into that player's graveyard. This change in ownership is permanent.
// XXX: ante card, not implementable
	Register("Tempest Efreet", withExpansion(func() Card {
		return NewCreature("Tempest Efreet", "{1}{R}{R}{R}", 3, 3,
			WithSubTypes("Efreet"),
		)
	}))

// Wall of Dust {2}{R}
// Creature — Wall
// 1/4
// Defender (This creature can't attack.)
// Whenever this creature blocks a creature, that creature can't attack during its controller's next turn.
// TODO: implement — needs engine support for "can't attack next turn" delayed restriction
	Register("Wall of Dust", withExpansion(func() Card {
		return NewCreature("Wall of Dust", "{2}{R}", 1, 4,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
		)
	}))

// Wall of Earth {1}{R}
// Creature — Wall
// 0/6
// Defender (This creature can't attack.)
	Register("Wall of Earth", withExpansion(func() Card {
		return NewCreature("Wall of Earth", "{1}{R}", 0, 6,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
		)
	}))

// Wall of Heat {2}{R}
// Creature — Wall
// 2/6
// Defender (This creature can't attack.)
	Register("Wall of Heat", withExpansion(func() Card {
		return NewCreature("Wall of Heat", "{2}{R}", 2, 6,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
		)
	}))

// Wall of Opposition {3}{R}{R}
// Creature — Wall
// 0/6
// Defender (This creature can't attack.)
// {1}: This creature gets +1/+0 until end of turn.
	Register("Wall of Opposition", withExpansion(func() Card {
		return NewCreature("Wall of Opposition", "{3}{R}{R}", 0, 6,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
			WithActivatedAbility(
				BoostUntilEndOfTurn(Fixed(1), Fixed(0), SelectSource),
				GenericCost(1),
			),
		)
	}))


	// ===== GREEN CREATURES =====

// Aisling Leprechaun {G}
// Creature — Faerie
// 1/1
// Whenever this creature blocks or becomes blocked by a creature, that creature becomes green. (This effect lasts indefinitely.)
// TODO: implement — needs engine support for "blocks or becomes blocked by" trigger and indefinite color change
	Register("Aisling Leprechaun", withExpansion(func() Card {
		return NewCreature("Aisling Leprechaun", "{G}", 1, 1,
			WithSubTypes("Faerie"),
		)
	}))

// Barbary Apes {1}{G}
// Creature — Ape
// 2/2
	Register("Barbary Apes", withExpansion(func() Card {
		return NewCreature("Barbary Apes", "{1}{G}", 2, 2,
			WithSubTypes("Ape"),
		)
	}))

// Cat Warriors {1}{G}{G}
// Creature — Cat Warrior
// 2/2
// Forestwalk (This creature can't be blocked as long as defending player controls a Forest.)
	Register("Cat Warriors", withExpansion(func() Card {
		return NewCreature("Cat Warriors", "{1}{G}{G}", 2, 2,
			WithSubTypes("Cat", "Warrior"),
			WithKeyword(Forestwalk),
		)
	}))

// Craw Giant {3}{G}{G}{G}{G}
// Creature — Giant
// 6/4
// Trample
// Rampage 2 (Whenever this creature becomes blocked, it gets +2/+2 until end of turn for each creature blocking it beyond the first.)
	Register("Craw Giant", withExpansion(func() Card {
		return NewCreature("Craw Giant", "{3}{G}{G}{G}{G}", 6, 4,
			WithSubTypes("Giant"),
			WithKeyword(Trample),
			WithAbility(RampageTrigger(2)),
		)
	}))

// Durkwood Boars {4}{G}
// Creature — Boar
// 4/4
	Register("Durkwood Boars", withExpansion(func() Card {
		return NewCreature("Durkwood Boars", "{4}{G}", 4, 4,
			WithSubTypes("Boar"),
		)
	}))

// Elven Riders {3}{G}{G}
// Creature — Elf
// 3/3
// This creature can't be blocked except by Walls and/or creatures with flying.
// TODO: implement — needs engine support for "can't be blocked except by Walls and/or creatures with flying"
	Register("Elven Riders", withExpansion(func() Card {
		return NewCreature("Elven Riders", "{3}{G}{G}", 3, 3,
			WithSubTypes("Elf"),
		)
	}))

// Emerald Dragonfly {1}{G}
// Creature — Insect
// 1/1
// Flying
// {G}{G}: This creature gains first strike until end of turn.
	Register("Emerald Dragonfly", withExpansion(func() Card {
		return NewCreature("Emerald Dragonfly", "{1}{G}", 1, 1,
			WithSubTypes("Insect"),
			WithKeyword(Flying),
			WithActivatedAbility(
				GrantKeywordUntilEndOfTurn(FirstStrike, SelectSource),
				ManaCostOf("{G}{G}"),
			),
		)
	}))

// Fire Sprites {1}{G}
// Creature — Faerie
// 1/1
// Flying
// {G}, {T}: Add {R}.
	Register("Fire Sprites", withExpansion(func() Card {
		return NewCreature("Fire Sprites", "{1}{G}", 1, 1,
			WithSubTypes("Faerie"),
			WithKeyword(Flying),
			WithActivatedAbility(
				AddMana(Red, 1),
				ManaCostOf("{G}"),
				WithCost(TapSourceCost()),
			),
		)
	}))

// Floral Spuzzem {3}{G}
// Creature — Elemental
// 2/2
// Whenever this creature attacks and isn't blocked, you may destroy target artifact defending player controls. If you do, this creature assigns no combat damage this turn.
// TODO: implement — needs engine support for "attacks and isn't blocked" trigger and "assigns no combat damage" replacement
	Register("Floral Spuzzem", withExpansion(func() Card {
		return NewCreature("Floral Spuzzem", "{3}{G}", 2, 2,
			WithSubTypes("Elemental"),
		)
	}))

// Giant Turtle {1}{G}{G}
// Creature — Turtle
// 2/4
// This creature can't attack if it attacked during your last turn.
// TODO: implement — needs engine support for tracking "attacked last turn" state
	Register("Giant Turtle", withExpansion(func() Card {
		return NewCreature("Giant Turtle", "{1}{G}{G}", 2, 4,
			WithSubTypes("Turtle"),
		)
	}))

// Hornet Cobra {1}{G}{G}
// Creature — Snake
// 2/1
// First strike
	Register("Hornet Cobra", withExpansion(func() Card {
		return NewCreature("Hornet Cobra", "{1}{G}{G}", 2, 1,
			WithSubTypes("Snake"),
			WithKeyword(FirstStrike),
		)
	}))

// Ichneumon Druid {1}{G}{G}
// Creature — Human Druid
// 1/1
// Whenever an opponent casts an instant spell other than the first instant spell that player casts each turn, this creature deals 4 damage to that player.
// TODO: implement — needs engine support for tracking "first instant cast this turn" per player
	Register("Ichneumon Druid", withExpansion(func() Card {
		return NewCreature("Ichneumon Druid", "{1}{G}{G}", 1, 1,
			WithSubTypes("Human", "Druid"),
		)
	}))

// Killer Bees {1}{G}{G}
// Creature — Insect
// 0/1
// Flying
// {G}: This creature gets +1/+1 until end of turn.
	Register("Killer Bees", withExpansion(func() Card {
		return NewCreature("Killer Bees", "{1}{G}{G}", 0, 1,
			WithSubTypes("Insect"),
			WithKeyword(Flying),
			WithActivatedAbility(
				BoostUntilEndOfTurn(Fixed(1), Fixed(1), SelectSource),
				ManaCostOf("{G}"),
			),
		)
	}))

// Master of the Hunt {2}{G}{G}
// Creature — Human
// 2/2
// {2}{G}{G}: Create a 1/1 green Wolf creature token named Wolves of the Hunt. It has "bands with other creatures named Wolves of the Hunt."
	Register("Master of the Hunt", withExpansion(func() Card {
		return NewCreature("Master of the Hunt", "{2}{G}{G}", 2, 2,
			WithSubTypes("Human"),
			WithActivatedAbility(
				CreateToken("Wolves of the Hunt", 1, 1, []CardType{TypeCreature}, []string{"Wolf"}),
				ManaCostOf("{2}{G}{G}"),
			),
		)
	}))

// Moss Monster {3}{G}{G}
// Creature — Elemental
// 3/6
	Register("Moss Monster", withExpansion(func() Card {
		return NewCreature("Moss Monster", "{3}{G}{G}", 3, 6,
			WithSubTypes("Elemental"),
		)
	}))

// Pixie Queen {2}{G}{G}
// Creature — Faerie
// 1/1
// Flying
// {G}{G}{G}, {T}: Target creature gains flying until end of turn.
	Register("Pixie Queen", withExpansion(func() Card {
		return NewCreature("Pixie Queen", "{2}{G}{G}", 1, 1,
			WithSubTypes("Faerie"),
			WithKeyword(Flying),
			WithActivatedAbility(
				GrantKeywordUntilEndOfTurn(Flying, SelectTarget),
				ManaCostOf("{G}{G}{G}"),
				WithCost(TapSourceCost()),
				WithTarget(TargetCreature()),
			),
		)
	}))

// Pradesh Gypsies {2}{G}
// Creature — Human Nomad
// 1/1
// {1}{G}, {T}: Target creature gets -2/-0 until end of turn.
	Register("Pradesh Gypsies", withExpansion(func() Card {
		return NewCreature("Pradesh Gypsies", "{2}{G}", 1, 1,
			WithSubTypes("Human", "Nomad"),
			WithActivatedAbility(
				BoostUntilEndOfTurn(Fixed(-2), Fixed(0), SelectTarget),
				ManaCostOf("{1}{G}"),
				WithCost(TapSourceCost()),
				WithTarget(TargetCreature()),
			),
		)
	}))

// Rabid Wombat {2}{G}{G}
// Creature — Wombat
// 0/1
// Vigilance
// This creature gets +2/+2 for each Aura attached to it.
	Register("Rabid Wombat", withExpansion(func() Card {
		return NewCreature("Rabid Wombat", "{2}{G}{G}", 0, 1,
			WithSubTypes("Wombat"),
			WithKeyword(Vigilance),
			WithStaticAbility(
				FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					auraCount := 0
					for _, attID := range src.Attachments {
						att := g.FindPermanent(attID)
						if att != nil && att.HasSubType("Aura") {
							auraCount++
						}
					}
					src.BoostPT(2*auraCount, 2*auraCount)
					return nil
				}),
			),
		)
	}))

// Radjan Spirit {3}{G}
// Creature — Spirit
// 3/2
// {T}: Target creature loses flying until end of turn.
	Register("Radjan Spirit", withExpansion(func() Card {
		return NewCreature("Radjan Spirit", "{3}{G}", 3, 2,
			WithSubTypes("Spirit"),
			WithActivatedAbility(
				FuncEffect("target creature loses flying until end of turn", EffectProperties{Outcome: OutcomeDetriment}, func(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					perm := g.FindPermanent(targets[0])
					if perm == nil {
						return nil
					}
					g.AddContinuousEffect(TargetEffect(LayerAbility, EndOfTurn, perm.ID(), func(g *Game, target *Permanent) error {
						g.Effects.RevokeAttr(target.ID(), Flying)
						return nil
					}))
					return nil
				}),
				TapSourceCost(),
				WithTarget(TargetCreature()),
			),
		)
	}))

// Shelkin Brownie {1}{G}
// Creature — Ouphe
// 1/1
// {T}: Target creature loses all "bands with other" abilities until end of turn.
// TODO: implement — needs engine support for "bands with other" ability removal
	Register("Shelkin Brownie", withExpansion(func() Card {
		return NewCreature("Shelkin Brownie", "{1}{G}", 1, 1,
			WithSubTypes("Ouphe"),
		)
	}))

// Whirling Dervish {G}{G}
// Creature — Human Monk
// 1/1
// Protection from black
// At the beginning of each end step, if this creature dealt damage to an opponent this turn, put a +1/+1 counter on it.
// TODO: implement — needs engine support for end-step trigger checking "dealt damage to opponent this turn"
	Register("Whirling Dervish", withExpansion(func() Card {
		return NewCreature("Whirling Dervish", "{G}{G}", 1, 1,
			WithSubTypes("Human", "Monk"),
			WithAbility(ProtectionFromColor(Black)),
		)
	}))

// Willow Satyr {2}{G}{G}
// Creature — Satyr
// 1/1
// You may choose not to untap this creature during your untap step.
// {T}: Gain control of target legendary creature for as long as you control this creature and this creature remains tapped.
// TODO: implement — needs engine support for conditional control change (while tapped and controlled)
	Register("Willow Satyr", withExpansion(func() Card {
		return NewCreature("Willow Satyr", "{2}{G}{G}", 1, 1,
			WithSubTypes("Satyr"),
			WithKeyword(AttrMayNotUntap),
		)
	}))

// Wolverine Pack {2}{G}{G}
// Creature — Wolverine
// 2/4
// Rampage 2 (Whenever this creature becomes blocked, it gets +2/+2 until end of turn for each creature blocking it beyond the first.)
	Register("Wolverine Pack", withExpansion(func() Card {
		return NewCreature("Wolverine Pack", "{2}{G}{G}", 2, 4,
			WithSubTypes("Wolverine"),
			WithAbility(RampageTrigger(2)),
		)
	}))

// Wood Elemental {3}{G}
// Creature — Elemental
// */*
// As this creature enters, sacrifice any number of untapped Forests.
// Wood Elemental's power and toughness are each equal to the number of Forests sacrificed as it entered.
// TODO: implement — needs engine support for "as enters" replacement effect with variable sacrifice count
	Register("Wood Elemental", withExpansion(func() Card {
		return NewCreature("Wood Elemental", "{3}{G}", 0, 0,
			WithSubTypes("Elemental"),
		)
	}))


	// ===== MULTICOLOR CREATURES =====

// Adun Oakenshield {B}{R}{G}
// Legendary Creature — Human Knight
// 1/2
// {B}{R}{G}, {T}: Return target creature card from your graveyard to your hand.
	Register("Adun Oakenshield", withExpansion(func() Card {
		return NewCreature("Adun Oakenshield", "{B}{R}{G}", 1, 2,
			WithSubTypes("Human", "Knight"),
			WithSuperTypes(SuperLegendary),
			WithActivatedAbility(
				ReturnFromGraveyardToHandTarget(),
				ManaCostOf("{B}{R}{G}"),
				WithCost(TapSourceCost()),
				WithTarget(TargetCreatureInYourGraveyard()),
			),
		)
	}))

// Angus Mackenzie {G}{W}{U}
// Legendary Creature — Human Cleric
// 2/2
// {G}{W}{U}, {T}: Prevent all combat damage that would be dealt this turn. Activate only before the combat damage step.
	Register("Angus Mackenzie", withExpansion(func() Card {
		return NewCreature("Angus Mackenzie", "{G}{W}{U}", 2, 2,
			WithSubTypes("Human", "Cleric"),
			WithSuperTypes(SuperLegendary),
			WithActivatedAbility(
				PreventAllCombatDamage(),
				ManaCostOf("{G}{W}{U}"),
				WithCost(TapSourceCost()),
			),
		)
	}))

// Arcades Sabboth {2}{G}{G}{W}{W}{U}{U}
// Legendary Creature — Elder Dragon
// 7/7
// Flying
// At the beginning of your upkeep, sacrifice Arcades Sabboth unless you pay {G}{W}{U}.
// Each untapped creature you control gets +0/+2 as long as it's not attacking.
// {W}: Arcades Sabboth gets +0/+1 until end of turn.
	Register("Arcades Sabboth", withExpansion(func() Card {
		return NewCreature("Arcades Sabboth", "{2}{G}{G}{W}{W}{U}{U}", 7, 7,
			WithSubTypes("Elder", "Dragon"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(Flying),
			WithAbility(SacrificeAtUpkeepUnlessPay("{G}{W}{U}")),
			WithStaticAbility(
				FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					src := g.FindPermanent(sourceID)
					if src == nil {
						return nil
					}
					for _, p := range g.Battlefield {
						if p.Controller != src.Controller || !p.HasType(TypeCreature) {
							continue
						}
						if !p.Tapped && !g.Combat.IsAttacking(p.ID()) {
							p.BoostPT(0, 2)
						}
					}
					return nil
				}),
			),
			WithActivatedAbility(
				BoostUntilEndOfTurn(Fixed(0), Fixed(1), SelectSource),
				ManaCostOf("{W}"),
			),
		)
	}))

// Axelrod Gunnarson {4}{B}{B}{R}{R}
// Legendary Creature — Giant
// 5/5
// Trample
// Whenever a creature dealt damage by Axelrod Gunnarson this turn dies, you gain 1 life and Axelrod Gunnarson deals 1 damage to target player or planeswalker.
	Register("Axelrod Gunnarson", withExpansion(func() Card {
		return NewCreature("Axelrod Gunnarson", "{4}{B}{B}{R}{R}", 5, 5,
			WithSubTypes("Giant"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(Trample),
			WithAbility(CreatureDealtDamageBySourceDiesTrigger(
				CompositeEffects("gain 1 life and deal 1 damage to opponent",
					GainLife(1),
					DealDamageToPlayers(Fixed(1), SelectEachOpponent()),
				),
				false,
			)),
		)
	}))

// Ayesha Tanaka {W}{W}{U}{U}
// Legendary Creature — Human Artificer
// 2/2
// Banding (Any creatures with banding, and up to one without, can attack in a band. Bands are blocked as a group. If any creatures with banding you control are blocking or being blocked by a creature, you divide that creature's combat damage, not its controller, among any of the creatures it's being blocked by or is blocking.)
// {T}: Counter target activated ability from an artifact source unless that ability's controller pays {W}. (Mana abilities can't be targeted.)
// TODO: implement — needs engine support for countering activated abilities from artifact sources
	Register("Ayesha Tanaka", withExpansion(func() Card {
		return NewCreature("Ayesha Tanaka", "{W}{W}{U}{U}", 2, 2,
			WithSubTypes("Human", "Artificer"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(Banding),
		)
	}))

// Barktooth Warbeard {4}{B}{R}{R}
// Legendary Creature — Human Warrior
// 6/5
	Register("Barktooth Warbeard", withExpansion(func() Card {
		return NewCreature("Barktooth Warbeard", "{4}{B}{R}{R}", 6, 5,
			WithSubTypes("Human", "Warrior"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Bartel Runeaxe {3}{B}{R}{G}
// Legendary Creature — Giant Warrior
// 6/5
// Vigilance
// Bartel Runeaxe can't be the target of Aura spells.
// Note: "can't be target of Aura spells" is not enforced (needs engine support for aura targeting restriction)
	Register("Bartel Runeaxe", withExpansion(func() Card {
		return NewCreature("Bartel Runeaxe", "{3}{B}{R}{G}", 6, 5,
			WithSubTypes("Giant", "Warrior"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(Vigilance),
		)
	}))

// Boris Devilboon {3}{B}{R}
// Legendary Creature — Zombie Wizard
// 2/2
// {2}{B}{R}, {T}: Create a 1/1 black and red Demon creature token named Minor Demon.
	Register("Boris Devilboon", withExpansion(func() Card {
		return NewCreature("Boris Devilboon", "{3}{B}{R}", 2, 2,
			WithSubTypes("Zombie", "Wizard"),
			WithSuperTypes(SuperLegendary),
			WithActivatedAbility(
				CreateToken("Minor Demon", 1, 1, []CardType{TypeCreature}, []string{"Demon"}),
				ManaCostOf("{2}{B}{R}"),
				WithCost(TapSourceCost()),
			),
		)
	}))

// Chromium {2}{W}{W}{U}{U}{B}{B}
// Legendary Creature — Elder Dragon
// 7/7
// Flying
// Rampage 2 (Whenever this creature becomes blocked, it gets +2/+2 until end of turn for each creature blocking it beyond the first.)
// At the beginning of your upkeep, sacrifice Chromium unless you pay {W}{U}{B}.
	Register("Chromium", withExpansion(func() Card {
		return NewCreature("Chromium", "{2}{W}{W}{U}{U}{B}{B}", 7, 7,
			WithSubTypes("Elder", "Dragon"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(Flying),
			WithAbility(RampageTrigger(2)),
			WithAbility(SacrificeAtUpkeepUnlessPay("{W}{U}{B}")),
		)
	}))

// Dakkon Blackblade {2}{W}{U}{U}{B}
// Legendary Creature — Human Warrior
// */*
// Dakkon Blackblade's power and toughness are each equal to the number of lands you control.
	Register("Dakkon Blackblade", withExpansion(func() Card {
		return NewCreature("Dakkon Blackblade", "{2}{W}{U}{U}{B}", 0, 0,
			WithSubTypes("Human", "Warrior"),
			WithSuperTypes(SuperLegendary),
			WithStaticAbility(
				PTEqualsControlledCount(IsLand),
			),
		)
	}))

// Gabriel Angelfire {3}{G}{G}{W}{W}
// Legendary Creature — Angel
// 4/4
// At the beginning of your upkeep, choose flying, first strike, trample, or rampage 3. Gabriel Angelfire gains that ability until your next upkeep. (Whenever a creature with rampage 3 becomes blocked, it gets +3/+3 until end of turn for each creature blocking it beyond the first.)
// TODO: implement — needs engine support for upkeep mode choice with "until next upkeep" duration and granting rampage
	Register("Gabriel Angelfire", withExpansion(func() Card {
		return NewCreature("Gabriel Angelfire", "{3}{G}{G}{W}{W}", 4, 4,
			WithSubTypes("Angel"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Gosta Dirk {3}{W}{W}{U}{U}
// Legendary Creature — Human Warrior
// 4/4
// First strike
// Creatures with islandwalk can be blocked as though they didn't have islandwalk.
	Register("Gosta Dirk", withExpansion(func() Card {
		return NewCreature("Gosta Dirk", "{3}{W}{W}{U}{U}", 4, 4,
			WithSubTypes("Human", "Warrior"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(FirstStrike),
			WithStaticAbility(
				NullifyLandwalkEffect(Islandwalk),
			),
		)
	}))

// Gwendlyn Di Corci {U}{B}{B}{R}
// Legendary Creature — Human Rogue
// 3/5
// {T}: Target player discards a card at random. Activate only during your turn.
// Note: "only during your turn" restriction is not enforced
	Register("Gwendlyn Di Corci", withExpansion(func() Card {
		return NewCreature("Gwendlyn Di Corci", "{U}{B}{B}{R}", 3, 5,
			WithSubTypes("Human", "Rogue"),
			WithSuperTypes(SuperLegendary),
			WithActivatedAbility(
				DiscardRandom(1),
				TapSourceCost(),
				WithTarget(TargetPlayer()),
			),
		)
	}))

// Halfdane {1}{W}{U}{B}
// Legendary Creature — Shapeshifter
// 3/3
// At the beginning of your upkeep, change Halfdane's base power and toughness to the power and toughness of target creature other than Halfdane until the end of your next upkeep.
// TODO: implement — needs engine support for "until your next upkeep" duration and base P/T copy
	Register("Halfdane", withExpansion(func() Card {
		return NewCreature("Halfdane", "{1}{W}{U}{B}", 3, 3,
			WithSubTypes("Shapeshifter"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Hazezon Tamar {4}{R}{G}{W}
// Legendary Creature — Human Warrior
// 2/4
// When Hazezon enters, create X 1/1 Sand Warrior creature tokens that are red, green, and white at the beginning of your next upkeep, where X is the number of lands you control at that time.
// When Hazezon leaves the battlefield, exile all Sand Warriors.
// TODO: implement
	Register("Hazezon Tamar", withExpansion(func() Card {
		return NewCreature("Hazezon Tamar", "{4}{R}{G}{W}", 2, 4,
			WithSubTypes("Human", "Warrior"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Hunding Gjornersen {3}{W}{U}{U}
// Legendary Creature — Human Warrior
// 5/4
// Rampage 1 (Whenever this creature becomes blocked, it gets +1/+1 until end of turn for each creature blocking it beyond the first.)
	Register("Hunding Gjornersen", withExpansion(func() Card {
		return NewCreature("Hunding Gjornersen", "{3}{W}{U}{U}", 5, 4,
			WithSubTypes("Human", "Warrior"),
			WithSuperTypes(SuperLegendary),
			WithAbility(RampageTrigger(1)),
		)
	}))

// Jacques le Vert {1}{R}{G}{W}
// Legendary Creature — Human Warrior
// 3/2
// Green creatures you control get +0/+2.
// TODO: implement
	Register("Jacques le Vert", withExpansion(func() Card {
		return NewCreature("Jacques le Vert", "{1}{R}{G}{W}", 3, 2,
			WithSubTypes("Human", "Warrior"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Jasmine Boreal {3}{G}{W}
// Legendary Creature — Human
// 4/5
	Register("Jasmine Boreal", withExpansion(func() Card {
		return NewCreature("Jasmine Boreal", "{3}{G}{W}", 4, 5,
			WithSubTypes("Human"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Jedit Ojanen {4}{W}{W}{U}
// Legendary Creature — Cat Warrior
// 5/5
	Register("Jedit Ojanen", withExpansion(func() Card {
		return NewCreature("Jedit Ojanen", "{4}{W}{W}{U}", 5, 5,
			WithSubTypes("Cat", "Warrior"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Jerrard of the Closed Fist {3}{R}{G}{G}
// Legendary Creature — Human Knight
// 6/5
	Register("Jerrard of the Closed Fist", withExpansion(func() Card {
		return NewCreature("Jerrard of the Closed Fist", "{3}{R}{G}{G}", 6, 5,
			WithSubTypes("Human", "Knight"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Johan {3}{R}{G}{W}
// Legendary Creature — Human Wizard
// 5/4
// At the beginning of combat on your turn, you may have Johan gain "Johan can't attack" until end of combat. If you do, attacking doesn't cause creatures you control to tap this combat if Johan is untapped.
// TODO: implement
	Register("Johan", withExpansion(func() Card {
		return NewCreature("Johan", "{3}{R}{G}{W}", 5, 4,
			WithSubTypes("Human", "Wizard"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Kasimir the Lone Wolf {4}{W}{U}
// Legendary Creature — Human Warrior
// 5/3
	Register("Kasimir the Lone Wolf", withExpansion(func() Card {
		return NewCreature("Kasimir the Lone Wolf", "{4}{W}{U}", 5, 3,
			WithSubTypes("Human", "Warrior"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Kei Takahashi {2}{G}{W}
// Legendary Creature — Human Cleric
// 2/2
// {T}: Prevent the next 2 damage that would be dealt to target creature this turn.
// TODO: implement
	Register("Kei Takahashi", withExpansion(func() Card {
		return NewCreature("Kei Takahashi", "{2}{G}{W}", 2, 2,
			WithSubTypes("Human", "Cleric"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Lady Caleria {3}{G}{G}{W}{W}
// Legendary Creature — Elf Archer
// 3/6
// {T}: Lady Caleria deals 3 damage to target attacking or blocking creature.
// TODO: implement
	Register("Lady Caleria", withExpansion(func() Card {
		return NewCreature("Lady Caleria", "{3}{G}{G}{W}{W}", 3, 6,
			WithSubTypes("Elf", "Archer"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Lady Evangela {W}{U}{B}
// Legendary Creature — Human Cleric
// 1/2
// {W}{B}, {T}: Prevent all combat damage that would be dealt by target creature this turn.
// TODO: implement
	Register("Lady Evangela", withExpansion(func() Card {
		return NewCreature("Lady Evangela", "{W}{U}{B}", 1, 2,
			WithSubTypes("Human", "Cleric"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Lady Orca {5}{B}{R}
// Legendary Creature — Demon
// 7/4
	Register("Lady Orca", withExpansion(func() Card {
		return NewCreature("Lady Orca", "{5}{B}{R}", 7, 4,
			WithSubTypes("Demon"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Livonya Silone {2}{R}{R}{G}{G}
// Legendary Creature — Human Warrior
// 4/4
// First strike; legendary landwalk (This creature can't be blocked as long as defending player controls a legendary land.)
// TODO: implement
	Register("Livonya Silone", withExpansion(func() Card {
		return NewCreature("Livonya Silone", "{2}{R}{R}{G}{G}", 4, 4,
			WithSubTypes("Human", "Warrior"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Lord Magnus {3}{G}{W}{W}
// Legendary Creature — Human Druid
// 4/3
// First strike
// Creatures with plainswalk can be blocked as though they didn't have plainswalk.
// Creatures with forestwalk can be blocked as though they didn't have forestwalk.
// TODO: implement
	Register("Lord Magnus", withExpansion(func() Card {
		return NewCreature("Lord Magnus", "{3}{G}{W}{W}", 4, 3,
			WithSubTypes("Human", "Druid"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Marhault Elsdragon {3}{R}{R}{G}
// Legendary Creature — Elf Warrior
// 4/6
// Rampage 1 (Whenever this creature becomes blocked, it gets +1/+1 until end of turn for each creature blocking it beyond the first.)
	Register("Marhault Elsdragon", withExpansion(func() Card {
		return NewCreature("Marhault Elsdragon", "{3}{R}{R}{G}", 4, 6,
			WithSubTypes("Elf", "Warrior"),
			WithSuperTypes(SuperLegendary),
			WithAbility(RampageTrigger(1)),
		)
	}))

// Nebuchadnezzar {3}{U}{B}
// Legendary Creature — Human Wizard
// 3/3
// {X}, {T}: Choose a card name. Target opponent reveals X cards at random from their hand. Then that player discards all cards with that name revealed this way. Activate only during your turn.
// TODO: implement
	Register("Nebuchadnezzar", withExpansion(func() Card {
		return NewCreature("Nebuchadnezzar", "{3}{U}{B}", 3, 3,
			WithSubTypes("Human", "Wizard"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Nicol Bolas {2}{U}{U}{B}{B}{R}{R}
// Legendary Creature — Elder Dragon
// 7/7
// Flying
// At the beginning of your upkeep, sacrifice Nicol Bolas unless you pay {U}{B}{R}.
// Whenever Nicol Bolas deals damage to an opponent, that player discards their hand.
// TODO: implement
	Register("Nicol Bolas", withExpansion(func() Card {
		return NewCreature("Nicol Bolas", "{2}{U}{U}{B}{B}{R}{R}", 7, 7,
			WithSubTypes("Elder", "Dragon"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Palladia-Mors {2}{R}{R}{G}{G}{W}{W}
// Legendary Creature — Elder Dragon
// 7/7
// Flying, trample
// At the beginning of your upkeep, sacrifice Palladia-Mors unless you pay {R}{G}{W}.
	Register("Palladia-Mors", withExpansion(func() Card {
		return NewCreature("Palladia-Mors", "{2}{R}{R}{G}{G}{W}{W}", 7, 7,
			WithSubTypes("Elder", "Dragon"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(Flying),
			WithKeyword(Trample),
			WithAbility(SacrificeAtUpkeepUnlessPay("{R}{G}{W}")),
		)
	}))

// Pavel Maliki {4}{B}{R}
// Legendary Creature — Human
// 5/3
// {B}{R}: Pavel Maliki gets +1/+0 until end of turn.
	Register("Pavel Maliki", withExpansion(func() Card {
		return NewCreature("Pavel Maliki", "{4}{B}{R}", 5, 3,
			WithSubTypes("Human"),
			WithSuperTypes(SuperLegendary),
			WithActivatedAbility(
				BoostUntilEndOfTurn(Fixed(1), Fixed(0), SelectSource),
				ManaCostOf("{B}{R}"),
			),
		)
	}))

// Princess Lucrezia {3}{U}{U}{B}
// Legendary Creature — Human Wizard
// 5/4
// {T}: Add {U}.
	Register("Princess Lucrezia", withExpansion(func() Card {
		return NewCreature("Princess Lucrezia", "{3}{U}{U}{B}", 5, 4,
			WithSubTypes("Human", "Wizard"),
			WithSuperTypes(SuperLegendary),
			WithManaAbility(Blue),
		)
	}))

// Ragnar {G}{W}{U}
// Legendary Creature — Human Cleric
// 2/2
// {G}{W}{U}, {T}: Regenerate target creature.
// TODO: implement
	Register("Ragnar", withExpansion(func() Card {
		return NewCreature("Ragnar", "{G}{W}{U}", 2, 2,
			WithSubTypes("Human", "Cleric"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Ramirez DePietro {3}{U}{B}{B}
// Legendary Creature — Human Pirate
// 4/3
// First strike
	Register("Ramirez DePietro", withExpansion(func() Card {
		return NewCreature("Ramirez DePietro", "{3}{U}{B}{B}", 4, 3,
			WithSubTypes("Human", "Pirate"),
			WithSuperTypes(SuperLegendary),
			WithKeyword(FirstStrike),
		)
	}))

// Ramses Overdark {2}{U}{U}{B}{B}
// Legendary Creature — Human Assassin
// 4/3
// {T}: Destroy target enchanted creature.
// TODO: implement
	Register("Ramses Overdark", withExpansion(func() Card {
		return NewCreature("Ramses Overdark", "{2}{U}{U}{B}{B}", 4, 3,
			WithSubTypes("Human", "Assassin"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Rasputin Dreamweaver {4}{W}{U}
// Legendary Creature — Human Wizard
// 4/1
// Rasputin enters with seven dream counters on it.
// Remove a dream counter from Rasputin: Add {C}.
// Remove a dream counter from Rasputin: Prevent the next 1 damage that would be dealt to Rasputin this turn.
// At the beginning of your upkeep, if Rasputin started the turn untapped, put a dream counter on it.
// Rasputin can't have more than seven dream counters on it.
// TODO: implement
	Register("Rasputin Dreamweaver", withExpansion(func() Card {
		return NewCreature("Rasputin Dreamweaver", "{4}{W}{U}", 4, 1,
			WithSubTypes("Human", "Wizard"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Riven Turnbull {5}{U}{B}
// Legendary Creature — Human Advisor
// 5/7
// {T}: Add {B}.
	Register("Riven Turnbull", withExpansion(func() Card {
		return NewCreature("Riven Turnbull", "{5}{U}{B}", 5, 7,
			WithSubTypes("Human", "Advisor"),
			WithSuperTypes(SuperLegendary),
			WithManaAbility(Black),
		)
	}))

// Rohgahh of Kher Keep {2}{B}{B}{R}{R}
// Legendary Creature — Kobold
// 5/5
// At the beginning of your upkeep, you may pay {R}{R}{R}. If you don't, tap Rohgahh and all creatures named Kobolds of Kher Keep, then an opponent gains control of them.
// Creatures you control named Kobolds of Kher Keep get +2/+2.
// TODO: implement
	Register("Rohgahh of Kher Keep", withExpansion(func() Card {
		return NewCreature("Rohgahh of Kher Keep", "{2}{B}{B}{R}{R}", 5, 5,
			WithSubTypes("Kobold"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Rubinia Soulsinger {2}{G}{W}{U}
// Legendary Creature — Faerie
// 2/3
// You may choose not to untap Rubinia Soulsinger during your untap step.
// {T}: Gain control of target creature for as long as you control Rubinia Soulsinger and Rubinia Soulsinger remains tapped.
// TODO: implement
	Register("Rubinia Soulsinger", withExpansion(func() Card {
		return NewCreature("Rubinia Soulsinger", "{2}{G}{W}{U}", 2, 3,
			WithSubTypes("Faerie"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Sir Shandlar of Eberyn {4}{G}{W}
// Legendary Creature — Human Knight
// 4/7
	Register("Sir Shandlar of Eberyn", withExpansion(func() Card {
		return NewCreature("Sir Shandlar of Eberyn", "{4}{G}{W}", 4, 7,
			WithSubTypes("Human", "Knight"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Sivitri Scarzam {5}{U}{B}
// Legendary Creature — Human
// 6/4
	Register("Sivitri Scarzam", withExpansion(func() Card {
		return NewCreature("Sivitri Scarzam", "{5}{U}{B}", 6, 4,
			WithSubTypes("Human"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Sol'kanar the Swamp King {2}{U}{B}{R}
// Legendary Creature — Demon
// 5/5
// Swampwalk (This creature can't be blocked as long as defending player controls a Swamp.)
// Whenever a player casts a black spell, you gain 1 life.
// TODO: implement
	Register("Sol'kanar the Swamp King", withExpansion(func() Card {
		return NewCreature("Sol'kanar the Swamp King", "{2}{U}{B}{R}", 5, 5,
			WithSubTypes("Demon"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Stangg {4}{R}{G}
// Legendary Creature — Human Warrior
// 3/4
// When Stangg enters, create Stangg Twin, a legendary 3/4 red and green Human Warrior creature token. Exile that token when Stangg leaves the battlefield. Sacrifice Stangg when that token leaves the battlefield.
// TODO: implement
	Register("Stangg", withExpansion(func() Card {
		return NewCreature("Stangg", "{4}{R}{G}", 3, 4,
			WithSubTypes("Human", "Warrior"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Sunastian Falconer {3}{R}{G}
// Legendary Creature — Human Shaman
// 4/4
// {T}: Add {C}{C}.
	Register("Sunastian Falconer", withExpansion(func() Card {
		return NewCreature("Sunastian Falconer", "{3}{R}{G}", 4, 4,
			WithSubTypes("Human", "Shaman"),
			WithSuperTypes(SuperLegendary),
			WithActivatedAbility(
				AddMana(Colorless, 2),
				TapSourceCost(),
			),
		)
	}))

// Tetsuo Umezawa {U}{B}{R}
// Legendary Creature — Human Archer
// 3/3
// Tetsuo Umezawa can't be the target of Aura spells.
// {U}{B}{B}{R}, {T}: Destroy target tapped or blocking creature.
// TODO: implement
	Register("Tetsuo Umezawa", withExpansion(func() Card {
		return NewCreature("Tetsuo Umezawa", "{U}{B}{R}", 3, 3,
			WithSubTypes("Human", "Archer"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// The Lady of the Mountain {4}{R}{G}
// Legendary Creature — Giant
// 5/5
	Register("The Lady of the Mountain", withExpansion(func() Card {
		return NewCreature("The Lady of the Mountain", "{4}{R}{G}", 5, 5,
			WithSubTypes("Giant"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Tobias Andrion {3}{W}{U}
// Legendary Creature — Human Advisor
// 4/4
	Register("Tobias Andrion", withExpansion(func() Card {
		return NewCreature("Tobias Andrion", "{3}{W}{U}", 4, 4,
			WithSubTypes("Human", "Advisor"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Tor Wauki {2}{B}{B}{R}
// Legendary Creature — Human Archer
// 3/3
// {T}: Tor Wauki deals 2 damage to target attacking or blocking creature.
// TODO: implement
	Register("Tor Wauki", withExpansion(func() Card {
		return NewCreature("Tor Wauki", "{2}{B}{B}{R}", 3, 3,
			WithSubTypes("Human", "Archer"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Torsten Von Ursus {3}{G}{G}{W}
// Legendary Creature — Human Soldier
// 5/5
	Register("Torsten Von Ursus", withExpansion(func() Card {
		return NewCreature("Torsten Von Ursus", "{3}{G}{G}{W}", 5, 5,
			WithSubTypes("Human", "Soldier"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Tuknir Deathlock {R}{R}{G}{G}
// Legendary Creature — Human Wizard
// 2/2
// Flying
// {R}{G}, {T}: Target creature gets +2/+2 until end of turn.
// TODO: implement
	Register("Tuknir Deathlock", withExpansion(func() Card {
		return NewCreature("Tuknir Deathlock", "{R}{R}{G}{G}", 2, 2,
			WithSubTypes("Human", "Wizard"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Ur-Drago {3}{U}{U}{B}{B}
// Legendary Creature — Elemental
// 4/4
// First strike
// Creatures with swampwalk can be blocked as though they didn't have swampwalk.
// TODO: implement
	Register("Ur-Drago", withExpansion(func() Card {
		return NewCreature("Ur-Drago", "{3}{U}{U}{B}{B}", 4, 4,
			WithSubTypes("Elemental"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Vaevictis Asmadi {2}{B}{B}{R}{R}{G}{G}
// Legendary Creature — Elder Dragon
// 7/7
// Flying
// At the beginning of your upkeep, sacrifice Vaevictis Asmadi unless you pay {B}{R}{G}.
// {B}: Vaevictis Asmadi gets +1/+0 until end of turn.
// {R}: Vaevictis Asmadi gets +1/+0 until end of turn.
// {G}: Vaevictis Asmadi gets +1/+0 until end of turn.
// TODO: implement
	Register("Vaevictis Asmadi", withExpansion(func() Card {
		return NewCreature("Vaevictis Asmadi", "{2}{B}{B}{R}{R}{G}{G}", 7, 7,
			WithSubTypes("Elder", "Dragon"),
			WithSuperTypes(SuperLegendary),
		)
	}))

// Xira Arien {B}{R}{G}
// Legendary Creature — Insect Wizard
// 1/2
// Flying
// {B}{R}{G}, {T}: Target player draws a card.
// TODO: implement
	Register("Xira Arien", withExpansion(func() Card {
		return NewCreature("Xira Arien", "{B}{R}{G}", 1, 2,
			WithSubTypes("Insect", "Wizard"),
			WithSuperTypes(SuperLegendary),
		)
	}))


	// ===== COLORLESS CREATURES =====

// Bronze Horse {7}
// Artifact Creature — Horse
// 4/4
// Trample
// As long as you control another creature, prevent all damage that would be dealt to this creature by spells that target it.
// TODO: implement
	Register("Bronze Horse", withExpansion(func() Card {
		return NewCreature("Bronze Horse", "{7}", 4, 4,
			WithSubTypes("Horse"),
			WithCardType(TypeArtifact),
		)
	}))

// Marble Priest {5}
// Artifact Creature — Cleric
// 3/3
// All Walls able to block this creature do so.
// Prevent all combat damage that would be dealt to this creature by Walls.
// TODO: implement
	Register("Marble Priest", withExpansion(func() Card {
		return NewCreature("Marble Priest", "{5}", 3, 3,
			WithSubTypes("Cleric"),
			WithCardType(TypeArtifact),
		)
	}))

// Sentinel {4}
// Artifact Creature — Shapeshifter
// 1/1
// {0}: Change this creature's base toughness to 1 plus the power of target creature blocking or blocked by this creature. (This effect lasts indefinitely.)
// TODO: implement
	Register("Sentinel", withExpansion(func() Card {
		return NewCreature("Sentinel", "{4}", 1, 1,
			WithSubTypes("Shapeshifter"),
			WithCardType(TypeArtifact),
		)
	}))

}
