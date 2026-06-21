package limited

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	. "github.com/benprew/mage-go/pkg/mage/dsl"
)

func init() {
	registerCreatures()
}

func registerCreatures() {
	// ===== WHITE CREATURES =====

	Register("Benalish Hero", func() Card {
		return NewCreature("Benalish Hero", "{W}", 1, 1,
			WithSubTypes("Human", "Soldier"),
			WithKeyword(Banding),
		)
	})

	Register("Mesa Pegasus", func() Card {
		return NewCreature("Mesa Pegasus", "{1}{W}", 1, 1,
			WithSubTypes("Pegasus"),
			WithKeyword(Flying),
			WithKeyword(Banding),
		)
	})

	Register("Pearled Unicorn", func() Card {
		return NewCreature("Pearled Unicorn", "{2}{W}", 2, 2, WithSubTypes("Unicorn"))
	})

	Register("Savannah Lions", func() Card {
		return NewCreature("Savannah Lions", "{W}", 2, 1, WithSubTypes("Cat"))
	})

	Register("Serra Angel", func() Card {
		return NewCreature("Serra Angel", "{3}{W}{W}", 4, 4,
			WithSubTypes("Angel"),
			WithKeyword(Flying),
			WithKeyword(Vigilance),
		)
	})

	Register("White Knight", func() Card {
		return NewCreature("White Knight", "{W}{W}", 2, 2,
			WithSubTypes("Human", "Knight"),
			WithKeyword(FirstStrike),
			WithAbility(ProtectionFromColor(Black)),
		)
	})

	Register("Northern Paladin", func() Card {
		return NewCreature("Northern Paladin", "{2}{W}{W}", 3, 3,
			WithSubTypes("Human", "Knight"),
			WithActivatedAbility(
				DestroyTargetPermanent(),
				ManaCostOf("{W}{W}"),
				WithCost(Tap()),
				WithTarget(TargetPermanent(HasColorFilter(Black))),
			),
		)
	})

	// ===== BLUE CREATURES =====

	Register("Air Elemental", func() Card {
		return NewCreature("Air Elemental", "{3}{U}{U}", 4, 4,
			WithSubTypes("Elemental"),
			WithKeyword(Flying),
		)
	})

	Register("Mahamoti Djinn", func() Card {
		return NewCreature("Mahamoti Djinn", "{4}{U}{U}", 5, 6,
			WithSubTypes("Djinn"),
			WithKeyword(Flying),
		)
	})

	Register("Phantom Monster", func() Card {
		return NewCreature("Phantom Monster", "{3}{U}", 3, 3,
			WithSubTypes("Illusion"),
			WithKeyword(Flying),
		)
	})

	Register("Water Elemental", func() Card {
		return NewCreature("Water Elemental", "{3}{U}{U}", 5, 4, WithSubTypes("Elemental"))
	})

	Register("Merfolk of the Pearl Trident", func() Card {
		return NewCreature("Merfolk of the Pearl Trident", "{U}", 1, 1, WithSubTypes("Merfolk"))
	})

	Register("Prodigal Sorcerer", func() Card {
		return NewCreature("Prodigal Sorcerer", "{2}{U}", 1, 1,
			WithSubTypes("Human", "Wizard"),
			WithActivatedAbility(
				DealDamage(Fixed(1)),
				Tap(),
				WithTarget(TargetDamageAnyTarget()),
			),
		)
	})

	Register("Pirate Ship", func() Card {
		return NewCreature("Pirate Ship", "{4}{U}", 4, 3,
			WithSubTypes("Human", "Pirate"),
			// Pirate Ship can't attack unless defending player controls an Island.
			WithStaticAbility(
				PreventFromAttackingIfDefendingPlayerControls(HasSubType("Island")),
			),
			WithActivatedAbility(
				DealDamage(Fixed(1)),
				Tap(),
				WithTarget(TargetDamageAnyTarget()),
			),
			// When you control no Islands, sacrifice Pirate Ship.
			WithAbility(SacrificeUnlessLand("Island")),
		)
	})

	Register("Phantasmal Forces", func() Card {
		return NewCreature("Phantasmal Forces", "{3}{U}", 4, 1,
			WithSubTypes("Illusion"),
			WithKeyword(Flying),
			// At the beginning of your upkeep, sacrifice Phantasmal Forces unless you pay {U}.
			WithAbility(SacrificeAtUpkeepUnlessPay("{U}")),
		)
	})

	Register("Clone", func() Card {
		return NewCreature("Clone", "{3}{U}", 0, 0,
			WithSubTypes("Shapeshifter"),
			WithAbility(NewTargetedSpell(TargetCreature(), CloneTarget())),
		)
	})

	// ===== BLACK CREATURES =====

	Register("Black Knight", func() Card {
		return NewCreature("Black Knight", "{B}{B}", 2, 2,
			WithSubTypes("Human", "Knight"),
			WithKeyword(FirstStrike),
			WithAbility(ProtectionFromColor(White)),
		)
	})

	Register("Bog Wraith", func() Card {
		return NewCreature("Bog Wraith", "{3}{B}", 3, 3,
			WithSubTypes("Wraith"),
			WithKeyword(Swampwalk),
		)
	})

	Register("Drudge Skeletons", func() Card {
		return NewCreature("Drudge Skeletons", "{1}{B}", 1, 1,
			WithSubTypes("Skeleton"),
			// {B}: Regenerate Drudge Skeletons.
			WithActivatedAbility(
				RegenerateSource(),
				ManaCostOf("{B}"),
			),
		)
	})

	Register("Frozen Shade", func() Card {
		return NewCreature("Frozen Shade", "{2}{B}", 0, 1,
			WithSubTypes("Shade"),
			// {B}: +1/+1 until end of turn
			WithActivatedAbility(
				Boost(Fixed(1), Fixed(1)).Targeting(ToSource()),
				ManaCostOf("{B}"),
			),
		)
	})

	Register("Hypnotic Specter", func() Card {
		return NewCreature("Hypnotic Specter", "{1}{B}{B}", 2, 2,
			WithSubTypes("Specter"),
			WithKeyword(Flying),
			WithAbility(DealsDamageToOpponentTrigger(DiscardRandom(1), false)),
		)
	})

	Register("Nether Shadow", func() Card {
		return NewCreature("Nether Shadow", "{B}{B}", 1, 1,
			WithSubTypes("Spirit"),
			WithKeyword(Haste),
			// At beginning of your upkeep, if Nether Shadow is in your graveyard
			// with three or more creature cards above it, put it onto the battlefield.
			WithAbility(NewTriggered(EvtUpkeep, false,
				ReturnSourceFromGraveyardToBattlefield(),
			).FromGraveyard().SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
				EventPlayerIsController{},
				SourceInOwnGraveyardWithCreaturesAbove{N: 3},
			}})),
		)
	})

	Register("Nightmare", func() Card {
		return NewCreature("Nightmare", "{5}{B}", 0, 0,
			WithSubTypes("Nightmare", "Horse"),
			WithKeyword(Flying),
			// P/T equal to number of Swamps you control
			WithStaticAbility(
				PTEqualsControlledCount(HasSubType("Swamp")),
			),
		)
	})

	Register("Plague Rats", func() Card {
		return NewCreature("Plague Rats", "{2}{B}", 0, 0,
			WithSubTypes("Rat"),
			// P/T equal to number of Plague Rats on the battlefield
			WithStaticAbility(
				PTEqualsCount(Named("Plague Rats")),
			),
		)
	})

	Register("Royal Assassin", func() Card {
		return NewCreature("Royal Assassin", "{1}{B}{B}", 1, 1,
			WithSubTypes("Human", "Assassin"),
			// {T}: Destroy target tapped creature
			WithActivatedAbility(
				DestroyTarget(),
				Tap(),
				WithTarget(TargetCreature(IsTapped)),
			),
		)
	})

	Register("Scathe Zombies", func() Card {
		return NewCreature("Scathe Zombies", "{2}{B}", 2, 2, WithSubTypes("Zombie"))
	})

	Register("Sengir Vampire", func() Card {
		return NewCreature("Sengir Vampire", "{3}{B}{B}", 4, 4,
			WithSubTypes("Vampire"),
			WithKeyword(Flying),
			// Whenever a creature dealt damage by Sengir Vampire this turn dies, put a +1/+1 counter on Sengir Vampire
			WithAbility(CreatureDealtDamageBySourceDiesTrigger(AddCounters(P1P1, Fixed(1)).Targeting(ToSource()), false)),
		)
	})

	Register("Will-o'-the-Wisp", func() Card {
		return NewCreature("Will-o'-the-Wisp", "{B}", 0, 1,
			WithSubTypes("Spirit"),
			WithKeyword(Flying),
			// {B}: Regenerate Will-o'-the-Wisp.
			WithActivatedAbility(
				RegenerateSource(),
				ManaCostOf("{B}"),
			),
		)
	})

	Register("Lord of the Pit", func() Card {
		return NewCreature("Lord of the Pit", "{4}{B}{B}{B}", 7, 7,
			WithSubTypes("Demon"),
			WithKeyword(Flying),
			WithKeyword(Trample),
			// At the beginning of your upkeep, sacrifice a creature other than Lord of the Pit.
			// If you can't, Lord of the Pit deals 7 damage to you.
			WithAbility(BeginningOfUpkeepTrigger(SacrificeCreatureOrDamage(7), false)),
		)
	})

	// ===== RED CREATURES =====

	Register("Dragon Whelp", func() Card {
		return NewCreature("Dragon Whelp", "{2}{R}{R}", 2, 3,
			WithSubTypes("Dragon"),
			WithKeyword(Flying),
			WithActivatedAbility(
				Boost(Fixed(1), Fixed(0)).Targeting(ToSource()),
				ManaCostOf("{R}"),
				// TODO: convert to pipeline — needs delayed trigger registration + counter tracking primitives
				WithEffect(FuncEffect(
					"if activated 4+ times, sacrifice at end of turn",
					EffectProperties{},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						perm := g.MutablePermanent(sourceID)
						if perm == nil {
							return nil
						}
						perm.AddCounter(Charge, 1)
						if perm.Counters[Charge] >= 4 {
							g.RegisterDelayedTrigger(&DelayedTrigger{
								EventType:  EvtEndStep,
								TargetID:   perm.ID(),
								SourceID:   sourceID,
								Controller: controller,
								Effects: []Effect{FuncEffect(
									"sacrifice this creature",
									EffectProperties{},
									func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
										if len(targets) == 0 {
											return nil
										}
										perm := g.FindPermanent(targets[0])
										if perm != nil {
											g.Sacrifice(perm)
										}
										return nil
									})},
							})
						}
						return nil
					})),
			),
		)
	})

	Register("Dwarven Warriors", func() Card {
		return NewCreature("Dwarven Warriors", "{2}{R}", 1, 1,
			WithSubTypes("Dwarf", "Warrior"),
			// {T}: Target creature with power 2 or less can't be blocked this turn.
			WithActivatedAbility(
				MakeUnblockableUntilEndOfTurn(),
				Tap(),
				WithTarget(TargetCreature(HasPowerLTE(2))),
			),
		)
	})

	Register("Earth Elemental", func() Card {
		return NewCreature("Earth Elemental", "{3}{R}{R}", 4, 5, WithSubTypes("Elemental"))
	})

	Register("Fire Elemental", func() Card {
		return NewCreature("Fire Elemental", "{3}{R}{R}", 5, 4, WithSubTypes("Elemental"))
	})

	Register("Goblin Balloon Brigade", func() Card {
		return NewCreature("Goblin Balloon Brigade", "{R}", 1, 1,
			WithSubTypes("Goblin", "Warrior"),
			// {R}: Goblin Balloon Brigade gains flying until end of turn.
			WithActivatedAbility(
				GrantKeyword(Flying).Targeting(ToSource()),
				ManaCostOf("{R}"),
			),
		)
	})

	Register("Granite Gargoyle", func() Card {
		return NewCreature("Granite Gargoyle", "{2}{R}", 2, 2,
			WithSubTypes("Gargoyle"),
			WithKeyword(Flying),
			// {R}: +0/+1 until end of turn
			WithActivatedAbility(
				Boost(Fixed(0), Fixed(1)).Targeting(ToSource()),
				ManaCostOf("{R}"),
			),
		)
	})

	Register("Gray Ogre", func() Card {
		return NewCreature("Gray Ogre", "{2}{R}", 2, 2, WithSubTypes("Ogre"))
	})

	Register("Hill Giant", func() Card {
		return NewCreature("Hill Giant", "{3}{R}", 3, 3, WithSubTypes("Giant"))
	})

	Register("Hurloon Minotaur", func() Card {
		return NewCreature("Hurloon Minotaur", "{1}{R}{R}", 2, 3, WithSubTypes("Minotaur"))
	})

	// Ironclaw Orcs {1}{R}
	// Creature — Orc
	// 2/2
	// This creature can't block creatures with power 2 or greater.
	Register("Ironclaw Orcs", func() Card {
		return NewCreature("Ironclaw Orcs", "{1}{R}", 2, 2,
			WithSubTypes("Orc"),
			WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
				for _, p := range g.AllBattlefield() {
					if p.HasType(TypeCreature) && p.CurrentPower(g) >= 2 {
						g.PreventBlockPair(sourceID, p.ID())
					}
				}
				return nil
			})),
		)
	})

	Register("Mons's Goblin Raiders", func() Card {
		return NewCreature("Mons's Goblin Raiders", "{R}", 1, 1, WithSubTypes("Goblin"))
	})

	Register("Roc of Kher Ridges", func() Card {
		return NewCreature("Roc of Kher Ridges", "{3}{R}", 3, 3,
			WithSubTypes("Bird"),
			WithKeyword(Flying),
		)
	})

	Register("Shivan Dragon", func() Card {
		return NewCreature("Shivan Dragon", "{4}{R}{R}", 5, 5,
			WithSubTypes("Dragon"),
			WithKeyword(Flying),
			// {R}: +1/+0 until end of turn (firebreathing)
			WithActivatedAbility(
				Boost(Fixed(1), Fixed(0)).Targeting(ToSource()),
				ManaCostOf("{R}"),
			),
		)
	})

	Register("Uthden Troll", func() Card {
		return NewCreature("Uthden Troll", "{2}{R}", 2, 2,
			WithSubTypes("Troll"),
			// {R}: Regenerate Uthden Troll.
			WithActivatedAbility(
				RegenerateSource(),
				ManaCostOf("{R}"),
			),
		)
	})

	Register("Sedge Troll", func() Card {
		return NewCreature("Sedge Troll", "{2}{R}", 2, 2,
			WithSubTypes("Troll"),
			// Sedge Troll gets +1/+1 as long as you control a Swamp.
			WithStaticAbility(
				BoostSelf(1, 1, WhileControlling(And(IsLand, HasSubType("Swamp")))),
			),
			// {B}: Regenerate Sedge Troll.
			WithActivatedAbility(
				RegenerateSource(),
				ManaCostOf("{B}"),
			),
		)
	})

	Register("Two-Headed Giant of Foriys", func() Card {
		return NewCreature("Two-Headed Giant of Foriys", "{4}{R}", 4, 4,
			WithSubTypes("Giant"),
			WithKeyword(Trample),
			// Two-Headed Giant of Foriys can block an additional creature each combat.
			WithKeyword(CanBlockAdditional),
		)
	})

	Register("Goblin King", func() Card {
		return NewCreature("Goblin King", "{1}{R}{R}", 2, 2,
			WithSubTypes("Goblin"),
			// Other Goblin creatures get +1/+1 and mountainwalk
			WithStaticAbility(
				BoostAllCreatures(1, 1, HasSubType("Goblin")),
				GrantKeywordToAll(Mountainwalk, HasSubType("Goblin")),
			),
		)
	})

	// ===== GREEN CREATURES =====

	Register("Craw Wurm", func() Card {
		return NewCreature("Craw Wurm", "{4}{G}{G}", 6, 4, WithSubTypes("Wurm"))
	})

	Register("Elvish Archers", func() Card {
		return NewCreature("Elvish Archers", "{1}{G}", 2, 1,
			WithSubTypes("Elf", "Archer"),
			WithKeyword(FirstStrike),
		)
	})

	Register("Force of Nature", func() Card {
		return NewCreature("Force of Nature", "{2}{G}{G}{G}{G}", 8, 8,
			WithSubTypes("Elemental"),
			WithKeyword(Trample),
			// At the beginning of your upkeep, Force of Nature deals 8 damage to you
			// unless you pay {G}{G}{G}{G}.
			WithAbility(NewTriggered(EvtUpkeep, false, UnlessTargetPays(
				SelectController(), ManaCostOf("{G}{G}{G}{G}"),
				"pay {G}{G}{G}{G} or Force of Nature deals 8 damage to you",
				DealDamageToPlayers(Fixed(8), SelectController()),
			)).SetConditionData(EventPlayerIsController{})),
		)
	})

	Register("Fungusaur", func() Card {
		return NewCreature("Fungusaur", "{3}{G}", 2, 2,
			WithSubTypes("Fungus", "Dinosaur"),
			// Whenever Fungusaur is dealt damage, put a +1/+1 counter on it.
			WithAbility(WhenDamageDealtToThisTrigger(AddCounters(P1P1, Fixed(1)).Targeting(ToSource()), false)),
		)
	})

	Register("Grizzly Bears", func() Card {
		return NewCreature("Grizzly Bears", "{1}{G}", 2, 2, WithSubTypes("Bear"))
	})

	Register("Ironroot Treefolk", func() Card {
		return NewCreature("Ironroot Treefolk", "{4}{G}", 3, 5, WithSubTypes("Treefolk"))
	})

	Register("Llanowar Elves", func() Card {
		return NewCreature("Llanowar Elves", "{G}", 1, 1,
			WithSubTypes("Elf", "Druid"),
			WithManaAbility(Green),
		)
	})

	Register("Scryb Sprites", func() Card {
		return NewCreature("Scryb Sprites", "{G}", 1, 1,
			WithSubTypes("Faerie"),
			WithKeyword(Flying),
		)
	})

	Register("Shanodin Dryads", func() Card {
		return NewCreature("Shanodin Dryads", "{G}", 1, 1,
			WithSubTypes("Nymph", "Dryad"),
			WithKeyword(Forestwalk),
		)
	})

	Register("Timber Wolves", func() Card {
		return NewCreature("Timber Wolves", "{G}", 1, 1,
			WithSubTypes("Wolf"),
			WithKeyword(Banding),
		)
	})

	Register("Thicket Basilisk", func() Card {
		return NewCreature("Thicket Basilisk", "{3}{G}{G}", 2, 4,
			WithSubTypes("Basilisk"),
			WithKeyword(BasiliskTouch),
		)
	})

	Register("War Mammoth", func() Card {
		return NewCreature("War Mammoth", "{3}{G}", 3, 3,
			WithSubTypes("Elephant"),
			WithKeyword(Trample),
		)
	})

	Register("Giant Spider", func() Card {
		return NewCreature("Giant Spider", "{3}{G}", 2, 4,
			WithSubTypes("Spider"),
			WithKeyword(Reach),
		)
	})

	Register("Birds of Paradise", func() Card {
		return NewCreature("Birds of Paradise", "{G}", 0, 1,
			WithSubTypes("Bird"),
			WithKeyword(Flying),
			// {T}: Add one mana of any color.
			WithAnyColorMana(),
		)
	})

	Register("Cockatrice", func() Card {
		return NewCreature("Cockatrice", "{3}{G}{G}", 2, 4,
			WithSubTypes("Cockatrice"),
			WithKeyword(Flying),
			WithKeyword(BasiliskTouch),
		)
	})

	Register("Verduran Enchantress", func() Card {
		return NewCreature("Verduran Enchantress", "{1}{G}{G}", 0, 2,
			WithSubTypes("Human", "Druid"),
			// Whenever you cast an enchantment spell, draw a card (any color enchantment)
			WithAbility(WheneverEnchantmentCastTrigger(DrawCards(Fixed(1)), true)),
		)
	})

	Register("Keldon Warlord", func() Card {
		return NewCreature("Keldon Warlord", "{2}{R}{R}", 0, 0,
			WithSubTypes("Human", "Barbarian"),
			// P/T equal to number of non-Wall creatures you control
			WithStaticAbility(
				PTEqualsControlledCount(And(IsCreature, Not(HasSubType("Wall")))),
			),
		)
	})

	// ===== WALLS =====

	Register("Wall of Air", func() Card {
		return NewCreature("Wall of Air", "{1}{U}{U}", 1, 5,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
			WithKeyword(Flying),
		)
	})

	Register("Wall of Bone", func() Card {
		return NewCreature("Wall of Bone", "{2}{B}", 1, 4,
			WithSubTypes("Wall", "Skeleton"),
			WithKeyword(Defender),
			// {B}: Regenerate Wall of Bone.
			WithActivatedAbility(
				RegenerateSource(),
				ManaCostOf("{B}"),
			),
		)
	})

	Register("Wall of Brambles", func() Card {
		return NewCreature("Wall of Brambles", "{2}{G}", 2, 3,
			WithSubTypes("Wall", "Plant"),
			WithKeyword(Defender),
			// {G}: Regenerate Wall of Brambles.
			WithActivatedAbility(
				RegenerateSource(),
				ManaCostOf("{G}"),
			),
		)
	})

	Register("Wall of Fire", func() Card {
		return NewCreature("Wall of Fire", "{1}{R}{R}", 0, 5,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
			// {R}: +1/+0 until end of turn
			WithActivatedAbility(
				Boost(Fixed(1), Fixed(0)).Targeting(ToSource()),
				ManaCostOf("{R}"),
			),
		)
	})

	Register("Wall of Ice", func() Card {
		return NewCreature("Wall of Ice", "{2}{G}", 0, 7,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
		)
	})

	Register("Wall of Stone", func() Card {
		return NewCreature("Wall of Stone", "{1}{R}{R}", 0, 8,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
		)
	})

	Register("Wall of Swords", func() Card {
		return NewCreature("Wall of Swords", "{3}{W}", 3, 5,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
			WithKeyword(Flying),
		)
	})

	Register("Wall of Water", func() Card {
		return NewCreature("Wall of Water", "{1}{U}{U}", 0, 5,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
			// {U}: +1/+0 until end of turn
			WithActivatedAbility(
				Boost(Fixed(1), Fixed(0)).Targeting(ToSource()),
				ManaCostOf("{U}"),
			),
		)
	})

	Register("Wall of Wood", func() Card {
		return NewCreature("Wall of Wood", "{G}", 0, 3,
			WithSubTypes("Wall"),
			WithKeyword(Defender),
		)
	})

	// ===== ARTIFACT CREATURES =====

	Register("Obsianus Golem", func() Card {
		return NewCreature("Obsianus Golem", "{6}", 4, 6,
			WithSubTypes("Golem"),
			WithCardType(TypeArtifact),
		)
	})

	Register("Clockwork Beast", func() Card {
		return NewCreature("Clockwork Beast", "{6}", 0, 4,
			WithSubTypes("Beast"),
			WithCardType(TypeArtifact),
			// Enters with 7 +1/+0 counters
			WithAbility(EntersWithNCounters(P1P0, 7)),
			// At end of combat, if Clockwork Beast attacked or blocked, remove a +1/+0 counter
			WithAbility(NewTriggered(EvtEndOfCombat, false,
				RemoveCounters(P1P0, 1),
			).
				SetConditionData(SourceAttackedOrBlockedThisTurn{})),
			// {X}, {T}: Put up to X +1/+0 counters on Clockwork Beast (max 7 total). Upkeep only.
			WithActivatedAbility(
				AddCounters(P1P0, XValue()).Targeting(ToSource()).Max(7),
				XManaCost(),
				WithCost(Tap()),
				WithUpkeepOnly(),
			),
		)
	})

	Register("Juggernaut", func() Card {
		return NewCreature("Juggernaut", "{4}", 5, 3,
			WithSubTypes("Juggernaut"),
			WithCardType(TypeArtifact),
			// Juggernaut attacks each combat if able. Can't be blocked by Walls.
			WithKeyword(MustAttack),
			WithKeyword(CantBeBlockedByWalls),
		)
	})

	Register("Living Wall", func() Card {
		return NewCreature("Living Wall", "{4}", 0, 6,
			WithSubTypes("Wall"),
			WithCardType(TypeArtifact),
			WithKeyword(Defender),
			// {1}: Regenerate Living Wall.
			WithActivatedAbility(
				RegenerateSource(),
				GenericCost(1),
			),
		)
	})

	// ===== LORD/ANTHEM CREATURES =====

	Register("Lord of Atlantis", func() Card {
		return NewCreature("Lord of Atlantis", "{U}{U}", 2, 2,
			WithSubTypes("Merfolk"),
			// Other Merfolk creatures get +1/+1 and islandwalk
			WithStaticAbility(
				BoostAllCreatures(1, 1, HasSubType("Merfolk")),
				GrantKeywordToAll(Islandwalk, HasSubType("Merfolk")),
			),
		)
	})

	Register("Zombie Master", func() Card {
		return NewCreature("Zombie Master", "{1}{B}{B}", 2, 3,
			WithSubTypes("Zombie"),
			// Other Zombie creatures have swampwalk and "{B}: Regenerate"
			WithStaticAbility(
				GrantKeywordToAll(Swampwalk, HasSubType("Zombie")),
				GrantActivatedAbilityToAll(
					RegenerateSource(),
					ManaCostOf("{B}"),
					HasSubType("Zombie"),
				),
			),
		)
	})

	// ===== MISC CREATURES =====

	Register("Samite Healer", func() Card {
		return NewCreature("Samite Healer", "{1}{W}", 1, 1,
			WithSubTypes("Human", "Cleric"),
			// {T}: Prevent the next 1 damage that would be dealt to any target this turn.
			WithActivatedAbility(
				PreventDamageToTarget(Fixed(1)),
				Tap(),
				WithTarget(TargetDamageAnyTarget()),
			),
		)
	})

	Register("Ley Druid", func() Card {
		return NewCreature("Ley Druid", "{2}{G}", 1, 1,
			WithSubTypes("Human", "Druid"),
			// {T}: Untap target land
			WithActivatedAbility(
				UntapTarget(),
				Tap(),
				WithTarget(TargetLand()),
			),
		)
	})

	Register("Sea Serpent", func() Card {
		return NewCreature("Sea Serpent", "{5}{U}", 5, 5,
			WithSubTypes("Serpent"),
			WithStaticAbility(
				PreventFromAttackingIfDefendingPlayerControls(HasSubType("Island")),
			),
			WithAbility(SacrificeUnlessLand("Island")),
		)
	})

	// {T}: Choose target non-Wall creature the active player has controlled continuously since the beginning of the turn. That creature attacks this turn if able. Destroy it at the beginning of the next end step if it didn't attack this turn. Activate only during an opponent's turn, before attackers are declared.
	Register("Nettling Imp", func() Card {
		validTarget := And(
			Not(HasSubType("Wall")),
			NewPermanentFilter("active player has controlled continuously since the beginning of the turn", func(p *Permanent, g *Game) bool {
				active := g.ActivePlayerObj()
				return active != nil && p.Controller == active.PlayerID() && p.TurnControlGained < g.CurrentTurn()
			}),
			NewPermanentFilter("creature without summoning sickness", func(p *Permanent, _ *Game) bool {
				return !p.HasAttr(core.AttrSummonSick) || p.HasAttr(core.Haste)
			}),
		)

		return NewCreature("Nettling Imp", "{2}{B}", 1, 1,
			WithSubTypes("Imp"),
			WithActivatedAbility(
				// TODO: convert to pipeline — needs delayed trigger registration + closure over targetID
				FuncEffect("force creature to attack or destroy at EOT",
					EffectProperties{},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						targetID := targets[0]
						if err := ApplyEffect(g, GrantKeyword(MustAttack), sourceID, controller, targets); err != nil {
							return err
						}
						g.RegisterDelayedTrigger(&DelayedTrigger{
							EventType:  EvtEndStep,
							SourceID:   sourceID,
							Controller: controller,
							Effects: []Effect{FuncEffect(
								"destroy creature that didn't attack",
								EffectProperties{},
								func(g2 *Game, srcID, ctrlID uuid.UUID, _ []uuid.UUID) error {
									if !g2.HasAttackedThisTurn(targetID) {
										perm := g2.FindPermanent(targetID)
										if perm != nil {
											g2.DestroyPermanent(perm)
										}
									}
									return nil
								}),
							},
						})
						return nil
					}),
				Tap(),
				WithTarget(TargetCreature(validTarget)),
				mage.WithActivationCondition(func(g *Game, _ *Permanent, controller uuid.UUID) bool {
					step := g.GetStep()
					return g.ActivePlayerObj().PlayerID() != controller &&
						step != core.Untap && step < core.DeclareAttackers
				}),
			),
		)
	})

	// At the beginning of each end step, put a corpse counter on this creature for each creature that died this turn.
	// Remove a corpse counter from this creature: Regenerate this creature.
	Register("Scavenging Ghoul", func() Card {
		return NewCreature("Scavenging Ghoul", "{3}{B}", 2, 2,
			WithSubTypes("Zombie"),
			// End step: put corpse counters equal to creatures that died this turn
			// TODO: convert to pipeline — needs CreatureDeaths() access as pipeline primitive
			WithAbility(BeginningOfEachEndStepTrigger(
				FuncEffect(
					"put corpse counters on Scavenging Ghoul for each creature that died this turn",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						deaths := g.CreatureDeaths()
						if deaths > 0 {
							perm := g.MutablePermanent(sourceID)
							if perm != nil {
								perm.AddCounter(Corpse, deaths)
							}
						}
						return nil
					},
				),
				false,
			)),
			// Remove a corpse counter: Regenerate
			WithActivatedAbility(RegenerateSource(), RemoveCountersCost(Corpse, 1)),
		)
	})

	Register("Rock Hydra", func() Card {
		return NewCreature("Rock Hydra", "{X}{R}{R}", 0, 0,
			WithSubTypes("Hydra"),
			// Enters with X +1/+1 counters (replacement effect)
			WithAbility(EntersWithXCounters(P1P1)),
		)
	})

	// XXX: should preserve original color when copying (Oracle: "except it doesn't copy that creature's color")
	Register("Vesuvan Doppelganger", func() Card {
		return NewCreature("Vesuvan Doppelganger", "{3}{U}{U}", 0, 0,
			WithSubTypes("Shapeshifter"),
			// As Vesuvan Doppelganger enters, copy target creature's P/T and keywords
			WithAbility(CopyCreatureOnETB()),
			// At the beginning of your upkeep, you may have this become a copy of another creature
			// TODO: convert to pipeline — needs CopyEffectCurrentName/UpdateCopyEffect primitives
			WithAbility(BeginningOfUpkeepTrigger(FuncEffect(
				"become a copy of target creature",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					perm := g.FindPermanent(sourceID)
					if perm == nil {
						return nil
					}
					currentName := g.CopyEffectCurrentName(perm.ID())
					var best *Permanent
					for _, p := range g.FilterBattlefield(AnyPermanent) {
						if p.ID() == perm.ID() {
							continue
						}
						if !p.HasType(TypeCreature) {
							continue
						}
						if p.Name() == currentName {
							continue
						}
						best = p
						break
					}
					if best != nil {
						g.UpdateCopyEffect(perm.ID(), best)
					}
					return nil
				},
			), true)),
		)
	})

	Register("Personal Incarnation", func() Card {
		return NewCreature("Personal Incarnation", "{3}{W}{W}{W}", 6, 6,
			WithSubTypes("Avatar", "Incarnation"),
			// {0}: The next 1 damage that would be dealt to this creature this turn
			// is dealt to its owner instead. Only this creature's owner may activate.
			WithActivatedAbility(
				// TODO: convert to pipeline — needs SetCreatureDamageRedirect primitive
				FuncEffect("redirect next 1 damage to owner",
					EffectProperties{},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						g.SetCreatureDamageRedirect(sourceID, controller)
						return nil
					}),
				GenericCost(0),
			),
			WithAbility(PutIntoGraveyardFromBattlefieldTrigger(
				LoseLifeAmount(HalfRoundUp(PlayerLifeValue())), false)),
		)
	})

	Register("Veteran Bodyguard", func() Card {
		return NewCreature("Veteran Bodyguard", "{3}{W}{W}", 2, 5,
			WithSubTypes("Human"),
			// As long as Veteran Bodyguard is untapped, all combat damage that would
			// be dealt to you is dealt to Veteran Bodyguard instead.
			WithStaticAbility(
				BodyguardContinuous(),
			),
		)
	})

	// Dwarven Demolition Team {2}{R}
	// Creature — Dwarf 1/1
	// {T}: Destroy target Wall.
	Register("Dwarven Demolition Team", func() Card {
		return NewCreature("Dwarven Demolition Team", "{2}{R}", 1, 1,
			WithSubTypes("Dwarf"),
			WithActivatedAbility(
				DestroyTarget(),
				Tap(),
				WithTarget(TargetCreature(HasSubType("Wall"))),
			),
		)
	})

	// Orcish Artillery {1}{R}{R}
	// Creature — Orc Warrior 1/3
	// {T}: Orcish Artillery deals 2 damage to any target and 3 damage to you.
	Register("Orcish Artillery", func() Card {
		return NewCreature("Orcish Artillery", "{1}{R}{R}", 1, 3,
			WithSubTypes("Orc", "Warrior"),
			WithActivatedAbility(
				CompositeEffects("deal 2 damage to any target and 3 damage to you",
					DealDamage(Fixed(2)),
					DealDamageToPlayers(Fixed(3), SelectController()),
				),
				Tap(),
				WithTarget(TargetDamageAnyTarget()),
			),
		)
	})
}
