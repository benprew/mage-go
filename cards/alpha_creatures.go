package cards

import "github.com/mage/mage"

func init() {
	registerAlphaCreatures()
}

func registerAlphaCreatures() {
	// ===== WHITE CREATURES =====

	mage.Register("Benalish Hero", func() mage.Card {
		c := mage.NewCreature("Benalish Hero", "{W}", "Human", "Soldier")
		c.Power_ = 1
		c.Toughness_ = 1
		c.AddAbility(mage.HasKeyword(mage.Banding))
		return c
	})

	mage.Register("Mesa Pegasus", func() mage.Card {
		c := mage.NewCreature("Mesa Pegasus", "{1}{W}", "Pegasus")
		c.Power_ = 1
		c.Toughness_ = 1
		c.AddAbility(mage.HasKeyword(mage.Flying))
		c.AddAbility(mage.HasKeyword(mage.Banding))
		return c
	})

	mage.Register("Pearled Unicorn", func() mage.Card {
		c := mage.NewCreature("Pearled Unicorn", "{2}{W}", "Unicorn")
		c.Power_ = 2
		c.Toughness_ = 2
		return c
	})

	mage.Register("Savannah Lions", func() mage.Card {
		c := mage.NewCreature("Savannah Lions", "{W}", "Cat")
		c.Power_ = 2
		c.Toughness_ = 1
		return c
	})

	// Serra Angel already registered in creatures.go

	// White Knight already registered in creatures.go

	mage.Register("Northern Paladin", func() mage.Card {
		c := mage.NewCreature("Northern Paladin", "{2}{W}{W}", "Human", "Knight")
		c.Power_ = 3
		c.Toughness_ = 3
		// {W}{W}, {T}: Destroy target black permanent
		ab := mage.NewActivatedAbility(
			mage.DestroyTargetPermanent(),
			mage.ManaCostOf("{W}{W}"),
		).AddCost(mage.TapSourceCost()).AddTarget(
			mage.TargetPermanent(mage.HasColorFilter(mage.Black)),
		)
		c.AddAbility(ab)
		return c
	})

	// ===== BLUE CREATURES =====

	mage.Register("Air Elemental", func() mage.Card {
		c := mage.NewCreature("Air Elemental", "{3}{U}{U}", "Elemental")
		c.Power_ = 4
		c.Toughness_ = 4
		c.AddAbility(mage.HasKeyword(mage.Flying))
		return c
	})

	mage.Register("Mahamoti Djinn", func() mage.Card {
		c := mage.NewCreature("Mahamoti Djinn", "{4}{U}{U}", "Djinn")
		c.Power_ = 5
		c.Toughness_ = 6
		c.AddAbility(mage.HasKeyword(mage.Flying))
		return c
	})

	mage.Register("Phantom Monster", func() mage.Card {
		c := mage.NewCreature("Phantom Monster", "{3}{U}", "Illusion")
		c.Power_ = 3
		c.Toughness_ = 3
		c.AddAbility(mage.HasKeyword(mage.Flying))
		return c
	})

	mage.Register("Water Elemental", func() mage.Card {
		c := mage.NewCreature("Water Elemental", "{3}{U}{U}", "Elemental")
		c.Power_ = 5
		c.Toughness_ = 4
		return c
	})

	mage.Register("Merfolk of the Pearl Trident", func() mage.Card {
		c := mage.NewCreature("Merfolk of the Pearl Trident", "{U}", "Merfolk")
		c.Power_ = 1
		c.Toughness_ = 1
		return c
	})

	mage.Register("Prodigal Sorcerer", func() mage.Card {
		c := mage.NewCreature("Prodigal Sorcerer", "{2}{U}", "Human", "Wizard")
		c.Power_ = 1
		c.Toughness_ = 1
		ab := mage.NewActivatedAbility(
			mage.DealDamage(1),
			mage.TapSourceCost(),
		).AddTarget(mage.TargetAnyTarget())
		c.AddAbility(ab)
		return c
	})

	mage.Register("Pirate Ship", func() mage.Card {
		c := mage.NewCreature("Pirate Ship", "{4}{U}", "Human", "Pirate")
		c.Power_ = 4
		c.Toughness_ = 3
		ab := mage.NewActivatedAbility(
			mage.DealDamage(1),
			mage.TapSourceCost(),
		).AddTarget(mage.TargetAnyTarget())
		c.AddAbility(ab)
		return c
	})

	mage.Register("Phantasmal Forces", func() mage.Card {
		c := mage.NewCreature("Phantasmal Forces", "{3}{U}", "Illusion")
		c.Power_ = 4
		c.Toughness_ = 1
		c.AddAbility(mage.HasKeyword(mage.Flying))
		// At the beginning of your upkeep, sacrifice Phantasmal Forces unless you pay {U}.
		c.AddAbility(mage.BeginningOfUpkeepTrigger(mage.SacrificeSource(), false))
		return c
	})

	mage.Register("Clone", func() mage.Card {
		// Simplified: Clone enters as a 0/0 - real copy is too complex
		c := mage.NewCreature("Clone", "{3}{U}", "Shapeshifter")
		c.Power_ = 0
		c.Toughness_ = 0
		return c
	})

	// ===== BLACK CREATURES =====

	mage.Register("Black Knight", func() mage.Card {
		c := mage.NewCreature("Black Knight", "{B}{B}", "Human", "Knight")
		c.Power_ = 2
		c.Toughness_ = 2
		c.AddAbility(mage.HasKeyword(mage.FirstStrike))
		c.AddAbility(mage.ProtectionFromColor(mage.White))
		return c
	})

	mage.Register("Bog Wraith", func() mage.Card {
		c := mage.NewCreature("Bog Wraith", "{3}{B}", "Wraith")
		c.Power_ = 3
		c.Toughness_ = 3
		c.AddAbility(mage.HasKeyword(mage.Swampwalk))
		return c
	})

	mage.Register("Drudge Skeletons", func() mage.Card {
		c := mage.NewCreature("Drudge Skeletons", "{1}{B}", "Skeleton")
		c.Power_ = 1
		c.Toughness_ = 1
		// {B}: Regenerate Drudge Skeletons.
		ab := mage.NewActivatedAbility(
			mage.RegenerateSource(),
			mage.ManaCostOf("{B}"),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Frozen Shade", func() mage.Card {
		c := mage.NewCreature("Frozen Shade", "{2}{B}", "Shade")
		c.Power_ = 0
		c.Toughness_ = 1
		// {B}: +1/+1 until end of turn
		ab := mage.NewActivatedAbility(
			mage.BoostSourceUntilEndOfTurn(1, 1),
			mage.ManaCostOf("{B}"),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Hypnotic Specter", func() mage.Card {
		c := mage.NewCreature("Hypnotic Specter", "{1}{B}{B}", "Specter")
		c.Power_ = 2
		c.Toughness_ = 2
		c.AddAbility(mage.HasKeyword(mage.Flying))
		c.AddAbility(mage.DealsDamageToOpponentTrigger(mage.DiscardRandom(1), false))
		return c
	})

	mage.Register("Nether Shadow", func() mage.Card {
		c := mage.NewCreature("Nether Shadow", "{B}{B}", "Spirit")
		c.Power_ = 1
		c.Toughness_ = 1
		c.AddAbility(mage.HasKeyword(mage.Haste))
		return c
	})

	mage.Register("Nightmare", func() mage.Card {
		c := mage.NewCreature("Nightmare", "{5}{B}", "Nightmare", "Horse")
		c.Power_ = 0
		c.Toughness_ = 0
		c.AddAbility(mage.HasKeyword(mage.Flying))
		// P/T equal to number of Swamps you control
		c.AddAbility(mage.StaticAbility(
			mage.PTEqualsControlledCount(mage.HasSubType("Swamp")),
		))
		return c
	})

	mage.Register("Plague Rats", func() mage.Card {
		c := mage.NewCreature("Plague Rats", "{2}{B}", "Rat")
		c.Power_ = 0
		c.Toughness_ = 0
		// P/T equal to number of Plague Rats on the battlefield
		c.AddAbility(mage.StaticAbility(
			mage.PTEqualsCount(mage.Named("Plague Rats")),
		))
		return c
	})

	mage.Register("Royal Assassin", func() mage.Card {
		c := mage.NewCreature("Royal Assassin", "{1}{B}{B}", "Human", "Assassin")
		c.Power_ = 1
		c.Toughness_ = 1
		// {T}: Destroy target tapped creature
		ab := mage.NewActivatedAbility(
			mage.DestroyTarget(),
			mage.TapSourceCost(),
		).AddTarget(mage.TargetCreature(mage.IsTapped))
		c.AddAbility(ab)
		return c
	})

	mage.Register("Scathe Zombies", func() mage.Card {
		c := mage.NewCreature("Scathe Zombies", "{2}{B}", "Zombie")
		c.Power_ = 2
		c.Toughness_ = 2
		return c
	})

	mage.Register("Sengir Vampire", func() mage.Card {
		c := mage.NewCreature("Sengir Vampire", "{3}{B}{B}", "Vampire")
		c.Power_ = 4
		c.Toughness_ = 4
		c.AddAbility(mage.HasKeyword(mage.Flying))
		// Whenever a creature dealt damage by Sengir Vampire this turn dies, put a +1/+1 counter on Sengir Vampire
		c.AddAbility(mage.CreatureDealtDamageBySourceDiesTrigger(mage.AddCountersToSource(mage.P1P1, 1), false))
		return c
	})

	mage.Register("Will-o'-the-Wisp", func() mage.Card {
		c := mage.NewCreature("Will-o'-the-Wisp", "{B}", "Spirit")
		c.Power_ = 0
		c.Toughness_ = 1
		c.AddAbility(mage.HasKeyword(mage.Flying))
		// {B}: Regenerate Will-o'-the-Wisp.
		ab := mage.NewActivatedAbility(
			mage.RegenerateSource(),
			mage.ManaCostOf("{B}"),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Lord of the Pit", func() mage.Card {
		c := mage.NewCreature("Lord of the Pit", "{4}{B}{B}{B}", "Demon")
		c.Power_ = 7
		c.Toughness_ = 7
		c.AddAbility(mage.HasKeyword(mage.Flying))
		c.AddAbility(mage.HasKeyword(mage.Trample))
		// At the beginning of your upkeep, sacrifice a creature other than Lord of the Pit.
		// If you can't, Lord of the Pit deals 7 damage to you.
		c.AddAbility(mage.BeginningOfUpkeepTrigger(mage.SacrificeCreatureOrDamage(7), false))
		return c
	})

	// ===== RED CREATURES =====

	mage.Register("Dragon Whelp", func() mage.Card {
		c := mage.NewCreature("Dragon Whelp", "{2}{R}{R}", "Dragon")
		c.Power_ = 2
		c.Toughness_ = 3
		c.AddAbility(mage.HasKeyword(mage.Flying))
		// {R}: +1/+0 until end of turn. If activated 4+ times, destroy at EOT.
		ab := mage.NewActivatedAbility(
			mage.BoostSourceUntilEndOfTurn(1, 0),
			mage.ManaCostOf("{R}"),
		).AddEffect(mage.MarkDestroyAtEOTAfterNActivations(4))
		c.AddAbility(ab)
		return c
	})

	mage.Register("Dwarven Warriors", func() mage.Card {
		c := mage.NewCreature("Dwarven Warriors", "{2}{R}", "Dwarf", "Warrior")
		c.Power_ = 1
		c.Toughness_ = 1
		// {T}: Target creature with power 2 or less can't be blocked this turn.
		ab := mage.NewActivatedAbility(
			mage.MakeUnblockableUntilEndOfTurn(),
			mage.TapSourceCost(),
		).AddTarget(mage.TargetCreature())
		c.AddAbility(ab)
		return c
	})

	mage.Register("Earth Elemental", func() mage.Card {
		c := mage.NewCreature("Earth Elemental", "{3}{R}{R}", "Elemental")
		c.Power_ = 4
		c.Toughness_ = 5
		return c
	})

	mage.Register("Fire Elemental", func() mage.Card {
		c := mage.NewCreature("Fire Elemental", "{3}{R}{R}", "Elemental")
		c.Power_ = 5
		c.Toughness_ = 4
		return c
	})

	mage.Register("Goblin Balloon Brigade", func() mage.Card {
		c := mage.NewCreature("Goblin Balloon Brigade", "{R}", "Goblin", "Warrior")
		c.Power_ = 1
		c.Toughness_ = 1
		// {R}: Goblin Balloon Brigade gains flying until end of turn.
		ab := mage.NewActivatedAbility(
			mage.GrantKeywordSourceUntilEndOfTurn(mage.Flying),
			mage.ManaCostOf("{R}"),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Granite Gargoyle", func() mage.Card {
		c := mage.NewCreature("Granite Gargoyle", "{2}{R}", "Gargoyle")
		c.Power_ = 2
		c.Toughness_ = 2
		c.AddAbility(mage.HasKeyword(mage.Flying))
		// {R}: +0/+1 until end of turn
		ab := mage.NewActivatedAbility(
			mage.BoostSourceUntilEndOfTurn(0, 1),
			mage.ManaCostOf("{R}"),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Gray Ogre", func() mage.Card {
		c := mage.NewCreature("Gray Ogre", "{2}{R}", "Ogre")
		c.Power_ = 2
		c.Toughness_ = 2
		return c
	})

	mage.Register("Hill Giant", func() mage.Card {
		c := mage.NewCreature("Hill Giant", "{3}{R}", "Giant")
		c.Power_ = 3
		c.Toughness_ = 3
		return c
	})

	mage.Register("Hurloon Minotaur", func() mage.Card {
		c := mage.NewCreature("Hurloon Minotaur", "{1}{R}{R}", "Minotaur")
		c.Power_ = 2
		c.Toughness_ = 3
		return c
	})

	mage.Register("Ironclaw Orcs", func() mage.Card {
		c := mage.NewCreature("Ironclaw Orcs", "{1}{R}", "Orc")
		c.Power_ = 2
		c.Toughness_ = 2
		return c
	})

	mage.Register("Mons's Goblin Raiders", func() mage.Card {
		c := mage.NewCreature("Mons's Goblin Raiders", "{R}", "Goblin")
		c.Power_ = 1
		c.Toughness_ = 1
		return c
	})

	mage.Register("Roc of Kher Ridges", func() mage.Card {
		c := mage.NewCreature("Roc of Kher Ridges", "{3}{R}", "Bird")
		c.Power_ = 3
		c.Toughness_ = 3
		c.AddAbility(mage.HasKeyword(mage.Flying))
		return c
	})

	mage.Register("Shivan Dragon", func() mage.Card {
		c := mage.NewCreature("Shivan Dragon", "{4}{R}{R}", "Dragon")
		c.Power_ = 5
		c.Toughness_ = 5
		c.AddAbility(mage.HasKeyword(mage.Flying))
		// {R}: +1/+0 until end of turn (firebreathing)
		ab := mage.NewActivatedAbility(
			mage.BoostSourceUntilEndOfTurn(1, 0),
			mage.ManaCostOf("{R}"),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Uthden Troll", func() mage.Card {
		c := mage.NewCreature("Uthden Troll", "{2}{R}", "Troll")
		c.Power_ = 2
		c.Toughness_ = 2
		// {R}: Regenerate Uthden Troll.
		ab := mage.NewActivatedAbility(
			mage.RegenerateSource(),
			mage.ManaCostOf("{R}"),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Sedge Troll", func() mage.Card {
		c := mage.NewCreature("Sedge Troll", "{2}{R}", "Troll")
		c.Power_ = 2
		c.Toughness_ = 2
		// Sedge Troll gets +1/+1 as long as you control a Swamp.
		c.AddAbility(mage.StaticAbility(
			mage.BoostSelfWhileControlling(1, 1, mage.And(mage.IsLand, mage.HasSubType("Swamp"))),
		))
		// {B}: Regenerate Sedge Troll.
		ab := mage.NewActivatedAbility(
			mage.RegenerateSource(),
			mage.ManaCostOf("{B}"),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Two-Headed Giant of Foriys", func() mage.Card {
		c := mage.NewCreature("Two-Headed Giant of Foriys", "{4}{R}", "Giant")
		c.Power_ = 4
		c.Toughness_ = 4
		c.AddAbility(mage.HasKeyword(mage.Trample))
		return c
	})

	mage.Register("Goblin King", func() mage.Card {
		c := mage.NewCreature("Goblin King", "{1}{R}{R}", "Goblin")
		c.Power_ = 2
		c.Toughness_ = 2
		// Other Goblin creatures get +1/+1 and mountainwalk
		c.AddAbility(mage.StaticAbility(
			mage.BoostAllCreatures(1, 1, mage.HasSubType("Goblin")),
			mage.GrantKeywordToAll(mage.Mountainwalk, mage.HasSubType("Goblin")),
		))
		return c
	})

	// ===== GREEN CREATURES =====

	mage.Register("Craw Wurm", func() mage.Card {
		c := mage.NewCreature("Craw Wurm", "{4}{G}{G}", "Wurm")
		c.Power_ = 6
		c.Toughness_ = 4
		return c
	})

	mage.Register("Elvish Archers", func() mage.Card {
		c := mage.NewCreature("Elvish Archers", "{1}{G}", "Elf", "Archer")
		c.Power_ = 2
		c.Toughness_ = 1
		c.AddAbility(mage.HasKeyword(mage.FirstStrike))
		return c
	})

	mage.Register("Force of Nature", func() mage.Card {
		c := mage.NewCreature("Force of Nature", "{2}{G}{G}{G}{G}", "Elemental")
		c.Power_ = 8
		c.Toughness_ = 8
		c.AddAbility(mage.HasKeyword(mage.Trample))
		// At the beginning of your upkeep, Force of Nature deals 8 damage to you
		// unless you pay {G}{G}{G}{G}.
		c.AddAbility(mage.BeginningOfUpkeepTrigger(mage.DealDamageToSourceController(8), false))
		return c
	})

	mage.Register("Fungusaur", func() mage.Card {
		c := mage.NewCreature("Fungusaur", "{3}{G}", "Fungus", "Dinosaur")
		c.Power_ = 2
		c.Toughness_ = 2
		// Whenever Fungusaur is dealt damage, put a +1/+1 counter on it.
		c.AddAbility(mage.WhenDamageDealtToThisTrigger(mage.AddCountersToSource(mage.P1P1, 1), false))
		return c
	})

	// Grizzly Bears already registered in creatures.go

	mage.Register("Ironroot Treefolk", func() mage.Card {
		c := mage.NewCreature("Ironroot Treefolk", "{4}{G}", "Treefolk")
		c.Power_ = 3
		c.Toughness_ = 5
		return c
	})

	mage.Register("Llanowar Elves", func() mage.Card {
		c := mage.NewCreature("Llanowar Elves", "{G}", "Elf", "Druid")
		c.Power_ = 1
		c.Toughness_ = 1
		c.AddAbility(mage.NewManaAbility(mage.Green))
		return c
	})

	mage.Register("Scryb Sprites", func() mage.Card {
		c := mage.NewCreature("Scryb Sprites", "{G}", "Faerie")
		c.Power_ = 1
		c.Toughness_ = 1
		c.AddAbility(mage.HasKeyword(mage.Flying))
		return c
	})

	mage.Register("Shanodin Dryads", func() mage.Card {
		c := mage.NewCreature("Shanodin Dryads", "{G}", "Nymph", "Dryad")
		c.Power_ = 1
		c.Toughness_ = 1
		c.AddAbility(mage.HasKeyword(mage.Forestwalk))
		return c
	})

	mage.Register("Timber Wolves", func() mage.Card {
		c := mage.NewCreature("Timber Wolves", "{G}", "Wolf")
		c.Power_ = 1
		c.Toughness_ = 1
		c.AddAbility(mage.HasKeyword(mage.Banding))
		return c
	})

	mage.Register("Thicket Basilisk", func() mage.Card {
		c := mage.NewCreature("Thicket Basilisk", "{3}{G}{G}", "Basilisk")
		c.Power_ = 2
		c.Toughness_ = 4
		// Whenever Thicket Basilisk deals damage to a creature, destroy that creature at end of combat
		// Simplified: has deathtouch-like behavior
		c.AddAbility(mage.HasKeyword(mage.Deathtouch))
		return c
	})

	mage.Register("War Mammoth", func() mage.Card {
		c := mage.NewCreature("War Mammoth", "{3}{G}", "Elephant")
		c.Power_ = 3
		c.Toughness_ = 3
		c.AddAbility(mage.HasKeyword(mage.Trample))
		return c
	})

	mage.Register("Giant Spider", func() mage.Card {
		c := mage.NewCreature("Giant Spider", "{3}{G}", "Spider")
		c.Power_ = 2
		c.Toughness_ = 4
		c.AddAbility(mage.HasKeyword(mage.Reach))
		return c
	})

	mage.Register("Birds of Paradise", func() mage.Card {
		c := mage.NewCreature("Birds of Paradise", "{G}", "Bird")
		c.Power_ = 0
		c.Toughness_ = 1
		c.AddAbility(mage.HasKeyword(mage.Flying))
		// Tap: Add one mana of any color (simplified: add Green for testing)
		c.AddAbility(mage.NewManaAbility(mage.Green))
		return c
	})

	mage.Register("Cockatrice", func() mage.Card {
		c := mage.NewCreature("Cockatrice", "{3}{G}{G}", "Cockatrice")
		c.Power_ = 2
		c.Toughness_ = 4
		c.AddAbility(mage.HasKeyword(mage.Flying))
		c.AddAbility(mage.HasKeyword(mage.Deathtouch))
		return c
	})

	mage.Register("Verduran Enchantress", func() mage.Card {
		c := mage.NewCreature("Verduran Enchantress", "{1}{G}{G}", "Human", "Druid")
		c.Power_ = 0
		c.Toughness_ = 2
		// Whenever you cast an enchantment spell, draw a card (any color enchantment)
		c.AddAbility(mage.WheneverEnchantmentCastTrigger(mage.DrawCards(1), true))
		return c
	})

	mage.Register("Keldon Warlord", func() mage.Card {
		c := mage.NewCreature("Keldon Warlord", "{2}{R}{R}", "Human", "Barbarian")
		c.Power_ = 0
		c.Toughness_ = 0
		// P/T equal to number of non-Wall creatures you control
		c.AddAbility(mage.StaticAbility(
			mage.PTEqualsControlledCount(mage.And(mage.IsCreature, mage.Not(mage.HasSubType("Wall")))),
		))
		return c
	})

	// ===== WALLS =====

	mage.Register("Wall of Air", func() mage.Card {
		c := mage.NewCreature("Wall of Air", "{1}{U}{U}", "Wall")
		c.Power_ = 1
		c.Toughness_ = 5
		c.AddAbility(mage.HasKeyword(mage.Defender))
		c.AddAbility(mage.HasKeyword(mage.Flying))
		return c
	})

	mage.Register("Wall of Bone", func() mage.Card {
		c := mage.NewCreature("Wall of Bone", "{2}{B}", "Wall", "Skeleton")
		c.Power_ = 1
		c.Toughness_ = 4
		c.AddAbility(mage.HasKeyword(mage.Defender))
		// {B}: Regenerate Wall of Bone.
		ab := mage.NewActivatedAbility(
			mage.RegenerateSource(),
			mage.ManaCostOf("{B}"),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Wall of Brambles", func() mage.Card {
		c := mage.NewCreature("Wall of Brambles", "{2}{G}", "Wall", "Plant")
		c.Power_ = 2
		c.Toughness_ = 3
		c.AddAbility(mage.HasKeyword(mage.Defender))
		// {G}: Regenerate Wall of Brambles.
		ab := mage.NewActivatedAbility(
			mage.RegenerateSource(),
			mage.ManaCostOf("{G}"),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Wall of Fire", func() mage.Card {
		c := mage.NewCreature("Wall of Fire", "{1}{R}{R}", "Wall")
		c.Power_ = 0
		c.Toughness_ = 5
		c.AddAbility(mage.HasKeyword(mage.Defender))
		// {R}: +1/+0 until end of turn
		ab := mage.NewActivatedAbility(
			mage.BoostSourceUntilEndOfTurn(1, 0),
			mage.ManaCostOf("{R}"),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Wall of Ice", func() mage.Card {
		c := mage.NewCreature("Wall of Ice", "{2}{G}", "Wall")
		c.Power_ = 0
		c.Toughness_ = 7
		c.AddAbility(mage.HasKeyword(mage.Defender))
		return c
	})

	mage.Register("Wall of Stone", func() mage.Card {
		c := mage.NewCreature("Wall of Stone", "{1}{R}{R}", "Wall")
		c.Power_ = 0
		c.Toughness_ = 8
		c.AddAbility(mage.HasKeyword(mage.Defender))
		return c
	})

	mage.Register("Wall of Swords", func() mage.Card {
		c := mage.NewCreature("Wall of Swords", "{3}{W}", "Wall")
		c.Power_ = 3
		c.Toughness_ = 5
		c.AddAbility(mage.HasKeyword(mage.Defender))
		c.AddAbility(mage.HasKeyword(mage.Flying))
		return c
	})

	mage.Register("Wall of Water", func() mage.Card {
		c := mage.NewCreature("Wall of Water", "{1}{U}{U}", "Wall")
		c.Power_ = 0
		c.Toughness_ = 5
		c.AddAbility(mage.HasKeyword(mage.Defender))
		// {U}: +1/+0 until end of turn
		ab := mage.NewActivatedAbility(
			mage.BoostSourceUntilEndOfTurn(1, 0),
			mage.ManaCostOf("{U}"),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Wall of Wood", func() mage.Card {
		c := mage.NewCreature("Wall of Wood", "{G}", "Wall")
		c.Power_ = 0
		c.Toughness_ = 3
		c.AddAbility(mage.HasKeyword(mage.Defender))
		return c
	})

	// ===== ARTIFACT CREATURES =====

	mage.Register("Obsianus Golem", func() mage.Card {
		c := mage.NewCreature("Obsianus Golem", "{6}", "Golem")
		c.Types_ = []mage.CardType{mage.TypeArtifact, mage.TypeCreature}
		c.Power_ = 4
		c.Toughness_ = 6
		return c
	})

	mage.Register("Clockwork Beast", func() mage.Card {
		c := mage.NewCreature("Clockwork Beast", "{6}", "Beast")
		c.Types_ = []mage.CardType{mage.TypeArtifact, mage.TypeCreature}
		c.Power_ = 0
		c.Toughness_ = 4
		// Enters with 7 +1/+0 counters
		c.AddAbility(mage.EntersBattlefieldTrigger(
			mage.AddCountersToSource(mage.P1P0, 7), false,
		))
		// Loses a +1/+0 counter whenever it attacks
		c.AddAbility(mage.AttacksTrigger(
			mage.RemoveCountersFromSource(mage.P1P0, 1), false,
		))
		return c
	})

	mage.Register("Juggernaut", func() mage.Card {
		c := mage.NewCreature("Juggernaut", "{4}", "Juggernaut")
		c.Types_ = []mage.CardType{mage.TypeArtifact, mage.TypeCreature}
		c.Power_ = 5
		c.Toughness_ = 3
		// Juggernaut attacks each combat if able. Can't be blocked by Walls.
		c.CantBeBlockedByWalls_ = true
		return c
	})

	mage.Register("Living Wall", func() mage.Card {
		c := mage.NewCreature("Living Wall", "{4}", "Wall")
		c.Types_ = []mage.CardType{mage.TypeArtifact, mage.TypeCreature}
		c.Power_ = 0
		c.Toughness_ = 6
		c.AddAbility(mage.HasKeyword(mage.Defender))
		// {1}: Regenerate Living Wall.
		ab := mage.NewActivatedAbility(
			mage.RegenerateSource(),
			mage.GenericCost(1),
		)
		c.AddAbility(ab)
		return c
	})

	// ===== LORD/ANTHEM CREATURES =====

	mage.Register("Lord of Atlantis", func() mage.Card {
		c := mage.NewCreature("Lord of Atlantis", "{U}{U}", "Merfolk")
		c.Power_ = 2
		c.Toughness_ = 2
		// Other Merfolk creatures get +1/+1 and islandwalk
		c.AddAbility(mage.StaticAbility(
			mage.BoostAllCreatures(1, 1, mage.HasSubType("Merfolk")),
			mage.GrantKeywordToAll(mage.Islandwalk, mage.HasSubType("Merfolk")),
		))
		return c
	})

	mage.Register("Zombie Master", func() mage.Card {
		c := mage.NewCreature("Zombie Master", "{1}{B}{B}", "Zombie")
		c.Power_ = 2
		c.Toughness_ = 3
		// Other Zombie creatures have swampwalk and "{B}: Regenerate"
		c.AddAbility(mage.StaticAbility(
			mage.GrantKeywordToAll(mage.Swampwalk, mage.HasSubType("Zombie")),
			mage.GrantActivatedAbilityToAll(
				mage.RegenerateSource(),
				mage.ManaCostOf("{B}"),
				mage.HasSubType("Zombie"),
			),
		))
		return c
	})

	// ===== MISC CREATURES =====

	mage.Register("Samite Healer", func() mage.Card {
		c := mage.NewCreature("Samite Healer", "{1}{W}", "Human", "Cleric")
		c.Power_ = 1
		c.Toughness_ = 1
		// {T}: Prevent the next 1 damage that would be dealt to any target this turn.
		ab := mage.NewActivatedAbility(
			mage.PreventDamageToTarget(1),
			mage.TapSourceCost(),
		).AddTarget(mage.TargetAnyTarget())
		c.AddAbility(ab)
		return c
	})

	mage.Register("Ley Druid", func() mage.Card {
		c := mage.NewCreature("Ley Druid", "{2}{G}", "Human", "Druid")
		c.Power_ = 1
		c.Toughness_ = 1
		// {T}: Untap target land
		ab := mage.NewActivatedAbility(
			mage.UntapTarget(),
			mage.TapSourceCost(),
		).AddTarget(mage.TargetLand())
		c.AddAbility(ab)
		return c
	})

	mage.Register("Sea Serpent", func() mage.Card {
		c := mage.NewCreature("Sea Serpent", "{5}{U}", "Serpent")
		c.Power_ = 5
		c.Toughness_ = 5
		c.AddAbility(mage.HasKeyword(mage.Islandwalk))
		c.SacrificeUnlessLand_ = "Island"
		return c
	})

	mage.Register("Nettling Imp", func() mage.Card {
		c := mage.NewCreature("Nettling Imp", "{2}{B}", "Imp")
		c.Power_ = 1
		c.Toughness_ = 1
		// {T}: Target non-Wall creature the active player controls attacks this
		// turn if able. (Simplified: just tap Nettling Imp targeting a creature)
		ab := mage.NewActivatedAbility(
			mage.TapTarget(),
			mage.TapSourceCost(),
		).AddTarget(mage.TargetCreature())
		c.AddAbility(ab)
		return c
	})

	mage.Register("Scavenging Ghoul", func() mage.Card {
		c := mage.NewCreature("Scavenging Ghoul", "{3}{B}", "Zombie")
		c.Power_ = 2
		c.Toughness_ = 2
		// Whenever another creature dies, put a +1/+1 counter on Scavenging Ghoul
		c.AddAbility(mage.AnyCreatureDiesTrigger(mage.AddCountersToSource(mage.P1P1, 1), true))
		return c
	})

	mage.Register("Rock Hydra", func() mage.Card {
		c := mage.NewCreature("Rock Hydra", "{X}{R}{R}", "Hydra")
		c.Power_ = 0
		c.Toughness_ = 0
		// Enters with X +1/+1 counters (replacement effect)
		c.EntersWithXCounters_ = mage.P1P1
		c.EntersWithXCountersSet = true
		return c
	})

	mage.Register("Vesuvan Doppelganger", func() mage.Card {
		c := mage.NewCreature("Vesuvan Doppelganger", "{3}{U}{U}", "Shapeshifter")
		c.Power_ = 0
		c.Toughness_ = 0
		return c
	})

	mage.Register("Personal Incarnation", func() mage.Card {
		c := mage.NewCreature("Personal Incarnation", "{3}{W}{W}{W}", "Avatar", "Incarnation")
		c.Power_ = 6
		c.Toughness_ = 6
		return c
	})

	mage.Register("Veteran Bodyguard", func() mage.Card {
		c := mage.NewCreature("Veteran Bodyguard", "{3}{W}{W}", "Human")
		c.Power_ = 2
		c.Toughness_ = 5
		return c
	})

	mage.Register("Aspect of Wolf", func() mage.Card {
		c := mage.NewAura("Aspect of Wolf", "{1}{G}")
		// Enchanted creature gets +X/+Y where X is half Forests you control
		// (rounded down) and Y is half (rounded up).
		c.AddAbility(mage.StaticAbility(
			mage.BoostAttachedByForestCount(),
		))
		return c
	})
}
