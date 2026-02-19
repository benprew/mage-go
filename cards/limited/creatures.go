package limited

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

	mage.Register("Benalish Hero", func() mage.Card {
		return mage.NewCreature("Benalish Hero", "{W}", 1, 1,
			mage.WithSubTypes("Human", "Soldier"),
			mage.WithKeyword(core.Banding),
		)
	})

	mage.Register("Mesa Pegasus", func() mage.Card {
		return mage.NewCreature("Mesa Pegasus", "{1}{W}", 1, 1,
			mage.WithSubTypes("Pegasus"),
			mage.WithKeyword(core.Flying),
			mage.WithKeyword(core.Banding),
		)
	})

	mage.Register("Pearled Unicorn", func() mage.Card {
		return mage.NewCreature("Pearled Unicorn", "{2}{W}", 2, 2, mage.WithSubTypes("Unicorn"))
	})

	mage.Register("Savannah Lions", func() mage.Card {
		return mage.NewCreature("Savannah Lions", "{W}", 2, 1, mage.WithSubTypes("Cat"))
	})

	mage.Register("Serra Angel", func() mage.Card {
		return mage.NewCreature("Serra Angel", "{3}{W}{W}", 4, 4,
			mage.WithSubTypes("Angel"),
			mage.WithKeyword(core.Flying),
			mage.WithKeyword(core.Vigilance),
		)
	})

	mage.Register("White Knight", func() mage.Card {
		return mage.NewCreature("White Knight", "{W}{W}", 2, 2,
			mage.WithSubTypes("Human", "Knight"),
			mage.WithKeyword(core.FirstStrike),
			mage.WithAbility(mage.ProtectionFromColor(core.Black)),
		)
	})

	mage.Register("Northern Paladin", func() mage.Card {
		return mage.NewCreature("Northern Paladin", "{2}{W}{W}", 3, 3,
			mage.WithSubTypes("Human", "Knight"),
			mage.WithAbility(mage.NewActivatedAbility(
				mage.DestroyTargetPermanent(),
				mage.ManaCostOf("{W}{W}"),
				mage.WithCost(mage.TapSourceCost()),
				mage.WithTarget(mage.TargetPermanent(mage.HasColorFilter(core.Black))),
			)),
		)
	})

	// ===== BLUE CREATURES =====

	mage.Register("Air Elemental", func() mage.Card {
		return mage.NewCreature("Air Elemental", "{3}{U}{U}", 4, 4,
			mage.WithSubTypes("Elemental"),
			mage.WithKeyword(core.Flying),
		)
	})

	mage.Register("Mahamoti Djinn", func() mage.Card {
		return mage.NewCreature("Mahamoti Djinn", "{4}{U}{U}", 5, 6,
			mage.WithSubTypes("Djinn"),
			mage.WithKeyword(core.Flying),
		)
	})

	mage.Register("Phantom Monster", func() mage.Card {
		return mage.NewCreature("Phantom Monster", "{3}{U}", 3, 3,
			mage.WithSubTypes("Illusion"),
			mage.WithKeyword(core.Flying),
		)
	})

	mage.Register("Water Elemental", func() mage.Card {
		return mage.NewCreature("Water Elemental", "{3}{U}{U}", 5, 4, mage.WithSubTypes("Elemental"))
	})

	mage.Register("Merfolk of the Pearl Trident", func() mage.Card {
		return mage.NewCreature("Merfolk of the Pearl Trident", "{U}", 1, 1, mage.WithSubTypes("Merfolk"))
	})

	mage.Register("Prodigal Sorcerer", func() mage.Card {
		return mage.NewCreature("Prodigal Sorcerer", "{2}{U}", 1, 1,
			mage.WithSubTypes("Human", "Wizard"),
			mage.WithAbility(mage.NewActivatedAbility(
				mage.DealDamage(mage.Fixed(1)),
				mage.TapSourceCost(),
				mage.WithTarget(mage.TargetAnyTarget()),
			)),
		)
	})

	mage.Register("Pirate Ship", func() mage.Card {
		return mage.NewCreature("Pirate Ship", "{4}{U}", 4, 3,
			mage.WithSubTypes("Human", "Pirate"),
			mage.WithAbility(mage.NewActivatedAbility(
				mage.DealDamage(mage.Fixed(1)),
				mage.TapSourceCost(),
				mage.WithTarget(mage.TargetAnyTarget()),
			)),
		)
	})

	mage.Register("Phantasmal Forces", func() mage.Card {
		return mage.NewCreature("Phantasmal Forces", "{3}{U}", 4, 1,
			mage.WithSubTypes("Illusion"),
			mage.WithKeyword(core.Flying),
			// At the beginning of your upkeep, sacrifice Phantasmal Forces unless you pay {U}.
			mage.WithAbility(mage.BeginningOfUpkeepTrigger(mage.SacrificeSource(), false)),
		)
	})

	mage.Register("Clone", func() mage.Card {
		return mage.NewCreature("Clone", "{3}{U}", 0, 0,
			mage.WithSubTypes("Shapeshifter"),
			// As Clone enters, choose a creature on the battlefield; Clone becomes a copy of that creature
			mage.WithAbility(mage.NewTargetedSpell(mage.TargetCreature(), mage.CloneTarget())),
		)
	})

	// ===== BLACK CREATURES =====

	mage.Register("Black Knight", func() mage.Card {
		return mage.NewCreature("Black Knight", "{B}{B}", 2, 2,
			mage.WithSubTypes("Human", "Knight"),
			mage.WithKeyword(core.FirstStrike),
			mage.WithAbility(mage.ProtectionFromColor(core.White)),
		)
	})

	mage.Register("Bog Wraith", func() mage.Card {
		return mage.NewCreature("Bog Wraith", "{3}{B}", 3, 3,
			mage.WithSubTypes("Wraith"),
			mage.WithKeyword(core.Swampwalk),
		)
	})

	mage.Register("Drudge Skeletons", func() mage.Card {
		return mage.NewCreature("Drudge Skeletons", "{1}{B}", 1, 1,
			mage.WithSubTypes("Skeleton"),
			// {B}: Regenerate Drudge Skeletons.
			mage.WithAbility(mage.NewActivatedAbility(
				mage.RegenerateSource(),
				mage.ManaCostOf("{B}"),
			)),
		)
	})

	mage.Register("Frozen Shade", func() mage.Card {
		return mage.NewCreature("Frozen Shade", "{2}{B}", 0, 1,
			mage.WithSubTypes("Shade"),
			// {B}: +1/+1 until end of turn
			mage.WithAbility(mage.NewActivatedAbility(
				mage.BoostUntilEndOfTurn(mage.Fixed(1), mage.Fixed(1), mage.SelectSource),
				mage.ManaCostOf("{B}"),
			)),
		)
	})

	mage.Register("Hypnotic Specter", func() mage.Card {
		return mage.NewCreature("Hypnotic Specter", "{1}{B}{B}", 2, 2,
			mage.WithSubTypes("Specter"),
			mage.WithKeyword(core.Flying),
			mage.WithAbility(mage.DealsDamageToOpponentTrigger(mage.DiscardRandom(1), false)),
		)
	})

	mage.Register("Nether Shadow", func() mage.Card {
		return mage.NewCreature("Nether Shadow", "{B}{B}", 1, 1,
			mage.WithSubTypes("Spirit"),
			mage.WithKeyword(core.Haste),
			// At beginning of your upkeep, if Nether Shadow is in your graveyard
			// with three or more creature cards above it, put it onto the battlefield.
			mage.WithAbility(mage.GraveyardReturnIfCreaturesAbove(3)),
		)
	})

	mage.Register("Nightmare", func() mage.Card {
		return mage.NewCreature("Nightmare", "{5}{B}", 0, 0,
			mage.WithSubTypes("Nightmare", "Horse"),
			mage.WithKeyword(core.Flying),
			// P/T equal to number of Swamps you control
			mage.WithAbility(mage.StaticAbility(
				mage.PTEqualsControlledCount(mage.HasSubType("Swamp")),
			)),
		)
	})

	mage.Register("Plague Rats", func() mage.Card {
		return mage.NewCreature("Plague Rats", "{2}{B}", 0, 0,
			mage.WithSubTypes("Rat"),
			// P/T equal to number of Plague Rats on the battlefield
			mage.WithAbility(mage.StaticAbility(
				mage.PTEqualsCount(mage.Named("Plague Rats")),
			)),
		)
	})

	mage.Register("Royal Assassin", func() mage.Card {
		return mage.NewCreature("Royal Assassin", "{1}{B}{B}", 1, 1,
			mage.WithSubTypes("Human", "Assassin"),
			// {T}: Destroy target tapped creature
			mage.WithAbility(mage.NewActivatedAbility(
				mage.DestroyTarget(),
				mage.TapSourceCost(),
				mage.WithTarget(mage.TargetCreature(mage.IsTapped)),
			)),
		)
	})

	mage.Register("Scathe Zombies", func() mage.Card {
		return mage.NewCreature("Scathe Zombies", "{2}{B}", 2, 2, mage.WithSubTypes("Zombie"))
	})

	mage.Register("Sengir Vampire", func() mage.Card {
		return mage.NewCreature("Sengir Vampire", "{3}{B}{B}", 4, 4,
			mage.WithSubTypes("Vampire"),
			mage.WithKeyword(core.Flying),
			// Whenever a creature dealt damage by Sengir Vampire this turn dies, put a +1/+1 counter on Sengir Vampire
			mage.WithAbility(mage.CreatureDealtDamageBySourceDiesTrigger(mage.AddCounters(core.P1P1, mage.Fixed(1), mage.SelectSource), false)),
		)
	})

	mage.Register("Will-o'-the-Wisp", func() mage.Card {
		return mage.NewCreature("Will-o'-the-Wisp", "{B}", 0, 1,
			mage.WithSubTypes("Spirit"),
			mage.WithKeyword(core.Flying),
			// {B}: Regenerate Will-o'-the-Wisp.
			mage.WithAbility(mage.NewActivatedAbility(
				mage.RegenerateSource(),
				mage.ManaCostOf("{B}"),
			)),
		)
	})

	mage.Register("Lord of the Pit", func() mage.Card {
		return mage.NewCreature("Lord of the Pit", "{4}{B}{B}{B}", 7, 7,
			mage.WithSubTypes("Demon"),
			mage.WithKeyword(core.Flying),
			mage.WithKeyword(core.Trample),
			// At the beginning of your upkeep, sacrifice a creature other than Lord of the Pit.
			// If you can't, Lord of the Pit deals 7 damage to you.
			mage.WithAbility(mage.BeginningOfUpkeepTrigger(mage.SacrificeCreatureOrDamage(7), false)),
		)
	})

	// ===== RED CREATURES =====

	mage.Register("Dragon Whelp", func() mage.Card {
		return mage.NewCreature("Dragon Whelp", "{2}{R}{R}", 2, 3,
			mage.WithSubTypes("Dragon"),
			mage.WithKeyword(core.Flying),
			// {R}: +1/+0 until end of turn. If activated 4+ times, destroy at EOT.
			mage.WithAbility(mage.NewActivatedAbility(
				mage.BoostUntilEndOfTurn(mage.Fixed(1), mage.Fixed(0), mage.SelectSource),
				mage.ManaCostOf("{R}"),
				mage.WithEffect(mage.MarkDestroyAtEOTAfterNActivations(4)),
			)),
		)
	})

	mage.Register("Dwarven Warriors", func() mage.Card {
		return mage.NewCreature("Dwarven Warriors", "{2}{R}", 1, 1,
			mage.WithSubTypes("Dwarf", "Warrior"),
			// {T}: Target creature with power 2 or less can't be blocked this turn.
			mage.WithAbility(mage.NewActivatedAbility(
				mage.MakeUnblockableUntilEndOfTurn(),
				mage.TapSourceCost(),
				mage.WithTarget(mage.TargetCreature()),
			)),
		)
	})

	mage.Register("Earth Elemental", func() mage.Card {
		return mage.NewCreature("Earth Elemental", "{3}{R}{R}", 4, 5, mage.WithSubTypes("Elemental"))
	})

	mage.Register("Fire Elemental", func() mage.Card {
		return mage.NewCreature("Fire Elemental", "{3}{R}{R}", 5, 4, mage.WithSubTypes("Elemental"))
	})

	mage.Register("Goblin Balloon Brigade", func() mage.Card {
		return mage.NewCreature("Goblin Balloon Brigade", "{R}", 1, 1,
			mage.WithSubTypes("Goblin", "Warrior"),
			// {R}: Goblin Balloon Brigade gains flying until end of turn.
			mage.WithAbility(mage.NewActivatedAbility(
				mage.GrantKeywordUntilEndOfTurn(core.Flying, mage.SelectSource),
				mage.ManaCostOf("{R}"),
			)),
		)
	})

	mage.Register("Granite Gargoyle", func() mage.Card {
		return mage.NewCreature("Granite Gargoyle", "{2}{R}", 2, 2,
			mage.WithSubTypes("Gargoyle"),
			mage.WithKeyword(core.Flying),
			// {R}: +0/+1 until end of turn
			mage.WithAbility(mage.NewActivatedAbility(
				mage.BoostUntilEndOfTurn(mage.Fixed(0), mage.Fixed(1), mage.SelectSource),
				mage.ManaCostOf("{R}"),
			)),
		)
	})

	mage.Register("Gray Ogre", func() mage.Card {
		return mage.NewCreature("Gray Ogre", "{2}{R}", 2, 2, mage.WithSubTypes("Ogre"))
	})

	mage.Register("Hill Giant", func() mage.Card {
		return mage.NewCreature("Hill Giant", "{3}{R}", 3, 3, mage.WithSubTypes("Giant"))
	})

	mage.Register("Hurloon Minotaur", func() mage.Card {
		return mage.NewCreature("Hurloon Minotaur", "{1}{R}{R}", 2, 3, mage.WithSubTypes("Minotaur"))
	})

	mage.Register("Ironclaw Orcs", func() mage.Card {
		return mage.NewCreature("Ironclaw Orcs", "{1}{R}", 2, 2, mage.WithSubTypes("Orc"))
	})

	mage.Register("Mons's Goblin Raiders", func() mage.Card {
		return mage.NewCreature("Mons's Goblin Raiders", "{R}", 1, 1, mage.WithSubTypes("Goblin"))
	})

	mage.Register("Roc of Kher Ridges", func() mage.Card {
		return mage.NewCreature("Roc of Kher Ridges", "{3}{R}", 3, 3,
			mage.WithSubTypes("Bird"),
			mage.WithKeyword(core.Flying),
		)
	})

	mage.Register("Shivan Dragon", func() mage.Card {
		return mage.NewCreature("Shivan Dragon", "{4}{R}{R}", 5, 5,
			mage.WithSubTypes("Dragon"),
			mage.WithKeyword(core.Flying),
			// {R}: +1/+0 until end of turn (firebreathing)
			mage.WithAbility(mage.NewActivatedAbility(
				mage.BoostUntilEndOfTurn(mage.Fixed(1), mage.Fixed(0), mage.SelectSource),
				mage.ManaCostOf("{R}"),
			)),
		)
	})

	mage.Register("Uthden Troll", func() mage.Card {
		return mage.NewCreature("Uthden Troll", "{2}{R}", 2, 2,
			mage.WithSubTypes("Troll"),
			// {R}: Regenerate Uthden Troll.
			mage.WithAbility(mage.NewActivatedAbility(
				mage.RegenerateSource(),
				mage.ManaCostOf("{R}"),
			)),
		)
	})

	mage.Register("Sedge Troll", func() mage.Card {
		return mage.NewCreature("Sedge Troll", "{2}{R}", 2, 2,
			mage.WithSubTypes("Troll"),
			// Sedge Troll gets +1/+1 as long as you control a Swamp.
			mage.WithAbility(mage.StaticAbility(
				mage.BoostSelf(1, 1, mage.WhileControlling(mage.And(mage.IsLand, mage.HasSubType("Swamp")))),
			)),
			// {B}: Regenerate Sedge Troll.
			mage.WithAbility(mage.NewActivatedAbility(
				mage.RegenerateSource(),
				mage.ManaCostOf("{B}"),
			)),
		)
	})

	mage.Register("Two-Headed Giant of Foriys", func() mage.Card {
		return mage.NewCreature("Two-Headed Giant of Foriys", "{4}{R}", 4, 4,
			mage.WithSubTypes("Giant"),
			mage.WithKeyword(core.Trample),
			// Two-Headed Giant of Foriys can block an additional creature each combat.
			mage.WithKeyword(core.CanBlockAdditional),
		)
	})

	mage.Register("Goblin King", func() mage.Card {
		return mage.NewCreature("Goblin King", "{1}{R}{R}", 2, 2,
			mage.WithSubTypes("Goblin"),
			// Other Goblin creatures get +1/+1 and mountainwalk
			mage.WithAbility(mage.StaticAbility(
				mage.BoostAllCreatures(1, 1, mage.HasSubType("Goblin")),
				mage.GrantKeywordToAll(core.Mountainwalk, mage.HasSubType("Goblin")),
			)),
		)
	})

	// ===== GREEN CREATURES =====

	mage.Register("Craw Wurm", func() mage.Card {
		return mage.NewCreature("Craw Wurm", "{4}{G}{G}", 6, 4, mage.WithSubTypes("Wurm"))
	})

	mage.Register("Elvish Archers", func() mage.Card {
		return mage.NewCreature("Elvish Archers", "{1}{G}", 2, 1,
			mage.WithSubTypes("Elf", "Archer"),
			mage.WithKeyword(core.FirstStrike),
		)
	})

	mage.Register("Force of Nature", func() mage.Card {
		return mage.NewCreature("Force of Nature", "{2}{G}{G}{G}{G}", 8, 8,
			mage.WithSubTypes("Elemental"),
			mage.WithKeyword(core.Trample),
			// At the beginning of your upkeep, Force of Nature deals 8 damage to you
			// unless you pay {G}{G}{G}{G}.
			mage.WithAbility(mage.BeginningOfUpkeepTrigger(mage.DealDamageToPlayers(mage.Fixed(8), mage.SelectController()), false)),
		)
	})

	mage.Register("Fungusaur", func() mage.Card {
		return mage.NewCreature("Fungusaur", "{3}{G}", 2, 2,
			mage.WithSubTypes("Fungus", "Dinosaur"),
			// Whenever Fungusaur is dealt damage, put a +1/+1 counter on it.
			mage.WithAbility(mage.WhenDamageDealtToThisTrigger(mage.AddCounters(core.P1P1, mage.Fixed(1), mage.SelectSource), false)),
		)
	})

	mage.Register("Grizzly Bears", func() mage.Card {
		return mage.NewCreature("Grizzly Bears", "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
	})

	mage.Register("Ironroot Treefolk", func() mage.Card {
		return mage.NewCreature("Ironroot Treefolk", "{4}{G}", 3, 5, mage.WithSubTypes("Treefolk"))
	})

	mage.Register("Llanowar Elves", func() mage.Card {
		return mage.NewCreature("Llanowar Elves", "{G}", 1, 1,
			mage.WithSubTypes("Elf", "Druid"),
			mage.WithManaAbility(core.Green),
		)
	})

	mage.Register("Scryb Sprites", func() mage.Card {
		return mage.NewCreature("Scryb Sprites", "{G}", 1, 1,
			mage.WithSubTypes("Faerie"),
			mage.WithKeyword(core.Flying),
		)
	})

	mage.Register("Shanodin Dryads", func() mage.Card {
		return mage.NewCreature("Shanodin Dryads", "{G}", 1, 1,
			mage.WithSubTypes("Nymph", "Dryad"),
			mage.WithKeyword(core.Forestwalk),
		)
	})

	mage.Register("Timber Wolves", func() mage.Card {
		return mage.NewCreature("Timber Wolves", "{G}", 1, 1,
			mage.WithSubTypes("Wolf"),
			mage.WithKeyword(core.Banding),
		)
	})

	mage.Register("Thicket Basilisk", func() mage.Card {
		return mage.NewCreature("Thicket Basilisk", "{3}{G}{G}", 2, 4,
			mage.WithSubTypes("Basilisk"),
			mage.WithKeyword(core.BasiliskTouch),
		)
	})

	mage.Register("War Mammoth", func() mage.Card {
		return mage.NewCreature("War Mammoth", "{3}{G}", 3, 3,
			mage.WithSubTypes("Elephant"),
			mage.WithKeyword(core.Trample),
		)
	})

	mage.Register("Giant Spider", func() mage.Card {
		return mage.NewCreature("Giant Spider", "{3}{G}", 2, 4,
			mage.WithSubTypes("Spider"),
			mage.WithKeyword(core.Reach),
		)
	})

	mage.Register("Birds of Paradise", func() mage.Card {
		return mage.NewCreature("Birds of Paradise", "{G}", 0, 1,
			mage.WithSubTypes("Bird"),
			mage.WithKeyword(core.Flying),
			// {T}: Add one mana of any color.
			mage.WithAnyColorMana(),
		)
	})

	mage.Register("Cockatrice", func() mage.Card {
		return mage.NewCreature("Cockatrice", "{3}{G}{G}", 2, 4,
			mage.WithSubTypes("Cockatrice"),
			mage.WithKeyword(core.Flying),
			mage.WithKeyword(core.Deathtouch),
		)
	})

	mage.Register("Verduran Enchantress", func() mage.Card {
		return mage.NewCreature("Verduran Enchantress", "{1}{G}{G}", 0, 2,
			mage.WithSubTypes("Human", "Druid"),
			// Whenever you cast an enchantment spell, draw a card (any color enchantment)
			mage.WithAbility(mage.WheneverEnchantmentCastTrigger(mage.DrawCards(mage.Fixed(1)), true)),
		)
	})

	mage.Register("Keldon Warlord", func() mage.Card {
		return mage.NewCreature("Keldon Warlord", "{2}{R}{R}", 0, 0,
			mage.WithSubTypes("Human", "Barbarian"),
			// P/T equal to number of non-Wall creatures you control
			mage.WithAbility(mage.StaticAbility(
				mage.PTEqualsControlledCount(mage.And(mage.IsCreature, mage.Not(mage.HasSubType("Wall")))),
			)),
		)
	})

	// ===== WALLS =====

	mage.Register("Wall of Air", func() mage.Card {
		return mage.NewCreature("Wall of Air", "{1}{U}{U}", 1, 5,
			mage.WithSubTypes("Wall"),
			mage.WithKeyword(core.Defender),
			mage.WithKeyword(core.Flying),
		)
	})

	mage.Register("Wall of Bone", func() mage.Card {
		return mage.NewCreature("Wall of Bone", "{2}{B}", 1, 4,
			mage.WithSubTypes("Wall", "Skeleton"),
			mage.WithKeyword(core.Defender),
			// {B}: Regenerate Wall of Bone.
			mage.WithAbility(mage.NewActivatedAbility(
				mage.RegenerateSource(),
				mage.ManaCostOf("{B}"),
			)),
		)
	})

	mage.Register("Wall of Brambles", func() mage.Card {
		return mage.NewCreature("Wall of Brambles", "{2}{G}", 2, 3,
			mage.WithSubTypes("Wall", "Plant"),
			mage.WithKeyword(core.Defender),
			// {G}: Regenerate Wall of Brambles.
			mage.WithAbility(mage.NewActivatedAbility(
				mage.RegenerateSource(),
				mage.ManaCostOf("{G}"),
			)),
		)
	})

	mage.Register("Wall of Fire", func() mage.Card {
		return mage.NewCreature("Wall of Fire", "{1}{R}{R}", 0, 5,
			mage.WithSubTypes("Wall"),
			mage.WithKeyword(core.Defender),
			// {R}: +1/+0 until end of turn
			mage.WithAbility(mage.NewActivatedAbility(
				mage.BoostUntilEndOfTurn(mage.Fixed(1), mage.Fixed(0), mage.SelectSource),
				mage.ManaCostOf("{R}"),
			)),
		)
	})

	mage.Register("Wall of Ice", func() mage.Card {
		return mage.NewCreature("Wall of Ice", "{2}{G}", 0, 7,
			mage.WithSubTypes("Wall"),
			mage.WithKeyword(core.Defender),
		)
	})

	mage.Register("Wall of Stone", func() mage.Card {
		return mage.NewCreature("Wall of Stone", "{1}{R}{R}", 0, 8,
			mage.WithSubTypes("Wall"),
			mage.WithKeyword(core.Defender),
		)
	})

	mage.Register("Wall of Swords", func() mage.Card {
		return mage.NewCreature("Wall of Swords", "{3}{W}", 3, 5,
			mage.WithSubTypes("Wall"),
			mage.WithKeyword(core.Defender),
			mage.WithKeyword(core.Flying),
		)
	})

	mage.Register("Wall of Water", func() mage.Card {
		return mage.NewCreature("Wall of Water", "{1}{U}{U}", 0, 5,
			mage.WithSubTypes("Wall"),
			mage.WithKeyword(core.Defender),
			// {U}: +1/+0 until end of turn
			mage.WithAbility(mage.NewActivatedAbility(
				mage.BoostUntilEndOfTurn(mage.Fixed(1), mage.Fixed(0), mage.SelectSource),
				mage.ManaCostOf("{U}"),
			)),
		)
	})

	mage.Register("Wall of Wood", func() mage.Card {
		return mage.NewCreature("Wall of Wood", "{G}", 0, 3,
			mage.WithSubTypes("Wall"),
			mage.WithKeyword(core.Defender),
		)
	})

	// ===== ARTIFACT CREATURES =====

	mage.Register("Obsianus Golem", func() mage.Card {
		return mage.NewCreature("Obsianus Golem", "{6}", 4, 6,
			mage.WithSubTypes("Golem"),
			mage.WithCardType(core.TypeArtifact),
		)
	})

	mage.Register("Clockwork Beast", func() mage.Card {
		return mage.NewCreature("Clockwork Beast", "{6}", 0, 4,
			mage.WithSubTypes("Beast"),
			mage.WithCardType(core.TypeArtifact),
			// Enters with 7 +1/+0 counters
			mage.WithAbility(mage.EntersBattlefieldTrigger(
				mage.AddCounters(core.P1P0, mage.Fixed(7), mage.SelectSource), false,
			)),
			// Loses a +1/+0 counter whenever it attacks
			mage.WithAbility(mage.AttacksTrigger(
				mage.RemoveCountersFromSource(core.P1P0, 1), false,
			)),
		)
	})

	mage.Register("Juggernaut", func() mage.Card {
		return mage.NewCreature("Juggernaut", "{4}", 5, 3,
			mage.WithSubTypes("Juggernaut"),
			mage.WithCardType(core.TypeArtifact),
			// Juggernaut attacks each combat if able. Can't be blocked by Walls.
			mage.WithKeyword(core.CantBeBlockedByWalls),
		)
	})

	mage.Register("Living Wall", func() mage.Card {
		return mage.NewCreature("Living Wall", "{4}", 0, 6,
			mage.WithSubTypes("Wall"),
			mage.WithCardType(core.TypeArtifact),
			mage.WithKeyword(core.Defender),
			// {1}: Regenerate Living Wall.
			mage.WithAbility(mage.NewActivatedAbility(
				mage.RegenerateSource(),
				mage.GenericCost(1),
			)),
		)
	})

	// ===== LORD/ANTHEM CREATURES =====

	mage.Register("Lord of Atlantis", func() mage.Card {
		return mage.NewCreature("Lord of Atlantis", "{U}{U}", 2, 2,
			mage.WithSubTypes("Merfolk"),
			// Other Merfolk creatures get +1/+1 and islandwalk
			mage.WithAbility(mage.StaticAbility(
				mage.BoostAllCreatures(1, 1, mage.HasSubType("Merfolk")),
				mage.GrantKeywordToAll(core.Islandwalk, mage.HasSubType("Merfolk")),
			)),
		)
	})

	mage.Register("Zombie Master", func() mage.Card {
		return mage.NewCreature("Zombie Master", "{1}{B}{B}", 2, 3,
			mage.WithSubTypes("Zombie"),
			// Other Zombie creatures have swampwalk and "{B}: Regenerate"
			mage.WithAbility(mage.StaticAbility(
				mage.GrantKeywordToAll(core.Swampwalk, mage.HasSubType("Zombie")),
				mage.GrantActivatedAbilityToAll(
					mage.RegenerateSource(),
					mage.ManaCostOf("{B}"),
					mage.HasSubType("Zombie"),
				),
			)),
		)
	})

	// ===== MISC CREATURES =====

	mage.Register("Samite Healer", func() mage.Card {
		return mage.NewCreature("Samite Healer", "{1}{W}", 1, 1,
			mage.WithSubTypes("Human", "Cleric"),
			// {T}: Prevent the next 1 damage that would be dealt to any target this turn.
			mage.WithAbility(mage.NewActivatedAbility(
				mage.PreventDamageToTarget(mage.Fixed(1)),
				mage.TapSourceCost(),
				mage.WithTarget(mage.TargetAnyTarget()),
			)),
		)
	})

	mage.Register("Ley Druid", func() mage.Card {
		return mage.NewCreature("Ley Druid", "{2}{G}", 1, 1,
			mage.WithSubTypes("Human", "Druid"),
			// {T}: Untap target land
			mage.WithAbility(mage.NewActivatedAbility(
				mage.UntapTarget(),
				mage.TapSourceCost(),
				mage.WithTarget(mage.TargetLand()),
			)),
		)
	})

	mage.Register("Sea Serpent", func() mage.Card {
		return mage.NewCreature("Sea Serpent", "{5}{U}", 5, 5,
			mage.WithSubTypes("Serpent"),
			mage.WithKeyword(core.Islandwalk),
			mage.WithAbility(mage.SacrificeUnlessLand("Island")),
		)
	})

	mage.Register("Nettling Imp", func() mage.Card {
		return mage.NewCreature("Nettling Imp", "{2}{B}", 1, 1,
			mage.WithSubTypes("Imp"),
			// {T}: Target non-Wall creature the active player controls attacks this
			// turn if able. Destroy it at end of turn if it didn't attack.
			mage.WithAbility(mage.NewActivatedAbility(
				mage.FuncEffect("force creature to attack or destroy at EOT",
					func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						if len(targets) == 0 {
							return nil
						}
						targetID := targets[0]
						mage.GrantKeywordUntilEndOfTurn(core.MustAttack, mage.SelectTarget).Apply(g, sourceID, controller, targets)
						g.RegisterDelayedTrigger(&mage.DelayedTrigger{
							EventType:  core.EvtEndStep,
							SourceID:   sourceID,
							Controller: controller,
							Effects: []mage.Effect{mage.FuncEffect(
								"destroy creature that didn't attack",
								func(g2 *mage.Game, srcID, ctrlID uuid.UUID, _ []uuid.UUID) error {
									if !g2.AttackedThisTurn[targetID] {
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
				mage.TapSourceCost(),
				mage.WithTarget(mage.TargetCreature(mage.Not(mage.HasSubType("Wall")))),
			)),
		)
	})

	mage.Register("Scavenging Ghoul", func() mage.Card {
		return mage.NewCreature("Scavenging Ghoul", "{3}{B}", 2, 2,
			mage.WithSubTypes("Zombie"),
			// Whenever another creature dies, put a +1/+1 counter on Scavenging Ghoul
			mage.WithAbility(mage.AnyCreatureDiesTrigger(mage.AddCounters(core.P1P1, mage.Fixed(1), mage.SelectSource), true)),
		)
	})

	mage.Register("Rock Hydra", func() mage.Card {
		return mage.NewCreature("Rock Hydra", "{X}{R}{R}", 0, 0,
			mage.WithSubTypes("Hydra"),
			// Enters with X +1/+1 counters (replacement effect)
			mage.WithAbility(mage.EntersWithXCounters(core.P1P1)),
		)
	})

	mage.Register("Vesuvan Doppelganger", func() mage.Card {
		return mage.NewCreature("Vesuvan Doppelganger", "{3}{U}{U}", 0, 0,
			mage.WithSubTypes("Shapeshifter"),
			// As Vesuvan Doppelganger enters, copy target creature's P/T and keywords
			mage.WithAbility(mage.CopyCreatureOnETB()),
			// At the beginning of your upkeep, you may have this become a copy of another creature
			mage.WithAbility(mage.BeginningOfUpkeepTrigger(mage.FuncEffect(
				"become a copy of target creature",
				func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					perm := g.FindPermanent(sourceID)
					if perm == nil {
						return nil
					}
					currentName := g.Effects.CopyEffectCurrentName(perm.ID())
					var best *mage.Permanent
					for _, p := range g.Battlefield {
						if p.ID() == perm.ID() {
							continue
						}
						if !p.HasType(core.TypeCreature) {
							continue
						}
						if p.Name() == currentName {
							continue
						}
						best = p
						break
					}
					if best != nil {
						g.Effects.UpdateCopyEffect(perm.ID(), best)
					}
					return nil
				},
			), false)),
		)
	})

	mage.Register("Personal Incarnation", func() mage.Card {
		return mage.NewCreature("Personal Incarnation", "{3}{W}{W}{W}", 6, 6,
			mage.WithSubTypes("Avatar", "Incarnation"),
			mage.WithKeyword(core.Flying),
			// All damage that would be dealt to you is dealt to Personal Incarnation instead.
			mage.WithAbility(mage.StaticAbility(mage.PersonalIncarnationRedirect())),
			// When Personal Incarnation dies, you lose half your life (rounded up).
			mage.WithAbility(mage.PutIntoGraveyardFromBattlefieldTrigger(mage.FuncEffect(
				"lose half your life rounded up",
				func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					p := g.GetPlayer(controller)
					if p == nil {
						return nil
					}
					halfLife := (p.Life() + 1) / 2
					if halfLife > 0 {
						p.LoseLife(halfLife)
					}
					return nil
				}), false)),
		)
	})

	mage.Register("Veteran Bodyguard", func() mage.Card {
		return mage.NewCreature("Veteran Bodyguard", "{3}{W}{W}", 2, 5,
			mage.WithSubTypes("Human"),
			// As long as Veteran Bodyguard is untapped, all combat damage that would
			// be dealt to you is dealt to Veteran Bodyguard instead.
			mage.WithAbility(mage.StaticAbility(
				mage.BodyguardContinuous(),
			)),
		)
	})

	mage.Register("Aspect of Wolf", func() mage.Card {
		return mage.NewAura("Aspect of Wolf", "{1}{G}",
			// Enchanted creature gets +X/+Y where X is half Forests you control
			// (rounded down) and Y is half (rounded up).
			mage.WithAbility(mage.StaticAbility(
				mage.BoostAttachedByForestCount(),
			)),
		)
	})
}
