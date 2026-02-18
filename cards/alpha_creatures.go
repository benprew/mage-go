package cards

import "github.com/mage/mage"

func init() {
	registerAlphaCreatures()
}

func registerAlphaCreatures() {
	// ===== WHITE CREATURES =====

	mage.Register("Benalish Hero", func() mage.Card {
		c := mage.NewCreature("Benalish Hero", "{W}", 1, 1, "Human", "Soldier")
		c.AddAbility(mage.HasKeyword(mage.Banding))
		return c
	})

	mage.Register("Mesa Pegasus", func() mage.Card {
		c := mage.NewCreature("Mesa Pegasus", "{1}{W}", 1, 1, "Pegasus")
		c.AddAbility(mage.HasKeyword(mage.Flying))
		c.AddAbility(mage.HasKeyword(mage.Banding))
		return c
	})

	mage.Register("Pearled Unicorn", func() mage.Card {
		c := mage.NewCreature("Pearled Unicorn", "{2}{W}", 2, 2, "Unicorn")
		return c
	})

	mage.Register("Savannah Lions", func() mage.Card {
		c := mage.NewCreature("Savannah Lions", "{W}", 2, 1, "Cat")
		return c
	})

	// Serra Angel already registered in creatures.go

	// White Knight already registered in creatures.go

	mage.Register("Northern Paladin", func() mage.Card {
		c := mage.NewCreature("Northern Paladin", "{2}{W}{W}", 3, 3, "Human", "Knight")
		// {W}{W}, {T}: Destroy target black permanent
		ab := mage.NewActivatedAbility(
			mage.DestroyTargetPermanent(),
			mage.ManaCostOf("{W}{W}"),
		
			mage.WithCost(mage.TapSourceCost()),
		
			mage.WithTarget(
			mage.TargetPermanent(mage.HasColorFilter(mage.Black)),
		),
		)
		c.AddAbility(ab)
		return c
	})

	// ===== BLUE CREATURES =====

	mage.Register("Air Elemental", func() mage.Card {
		c := mage.NewCreature("Air Elemental", "{3}{U}{U}", 4, 4, "Elemental")
		c.AddAbility(mage.HasKeyword(mage.Flying))
		return c
	})

	mage.Register("Mahamoti Djinn", func() mage.Card {
		c := mage.NewCreature("Mahamoti Djinn", "{4}{U}{U}", 5, 6, "Djinn")
		c.AddAbility(mage.HasKeyword(mage.Flying))
		return c
	})

	mage.Register("Phantom Monster", func() mage.Card {
		c := mage.NewCreature("Phantom Monster", "{3}{U}", 3, 3, "Illusion")
		c.AddAbility(mage.HasKeyword(mage.Flying))
		return c
	})

	mage.Register("Water Elemental", func() mage.Card {
		c := mage.NewCreature("Water Elemental", "{3}{U}{U}", 5, 4, "Elemental")
		return c
	})

	mage.Register("Merfolk of the Pearl Trident", func() mage.Card {
		c := mage.NewCreature("Merfolk of the Pearl Trident", "{U}", 1, 1, "Merfolk")
		return c
	})

	mage.Register("Prodigal Sorcerer", func() mage.Card {
		c := mage.NewCreature("Prodigal Sorcerer", "{2}{U}", 1, 1, "Human", "Wizard")
		ab := mage.NewActivatedAbility(
			mage.DealDamage(mage.Fixed(1)),
			mage.TapSourceCost(),
		
			mage.WithTarget(mage.TargetAnyTarget()),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Pirate Ship", func() mage.Card {
		c := mage.NewCreature("Pirate Ship", "{4}{U}", 4, 3, "Human", "Pirate")
		ab := mage.NewActivatedAbility(
			mage.DealDamage(mage.Fixed(1)),
			mage.TapSourceCost(),
		
			mage.WithTarget(mage.TargetAnyTarget()),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Phantasmal Forces", func() mage.Card {
		c := mage.NewCreature("Phantasmal Forces", "{3}{U}", 4, 1, "Illusion")
		c.AddAbility(mage.HasKeyword(mage.Flying))
		// At the beginning of your upkeep, sacrifice Phantasmal Forces unless you pay {U}.
		c.AddAbility(mage.BeginningOfUpkeepTrigger(mage.SacrificeSource(), false))
		return c
	})

	mage.Register("Clone", func() mage.Card {
		c := mage.NewCreature("Clone", "{3}{U}", 0, 0, "Shapeshifter")
		// As Clone enters, choose a creature on the battlefield; Clone becomes a copy of that creature
		sa := mage.NewTargetedSpell(mage.TargetCreature(), mage.CloneTarget())
		c.AddAbility(sa)
		return c
	})

	// ===== BLACK CREATURES =====

	mage.Register("Black Knight", func() mage.Card {
		c := mage.NewCreature("Black Knight", "{B}{B}", 2, 2, "Human", "Knight")
		c.AddAbility(mage.HasKeyword(mage.FirstStrike))
		c.AddAbility(mage.ProtectionFromColor(mage.White))
		return c
	})

	mage.Register("Bog Wraith", func() mage.Card {
		c := mage.NewCreature("Bog Wraith", "{3}{B}", 3, 3, "Wraith")
		c.AddAbility(mage.HasKeyword(mage.Swampwalk))
		return c
	})

	mage.Register("Drudge Skeletons", func() mage.Card {
		c := mage.NewCreature("Drudge Skeletons", "{1}{B}", 1, 1, "Skeleton")
		// {B}: Regenerate Drudge Skeletons.
		ab := mage.NewActivatedAbility(
			mage.RegenerateSource(),
			mage.ManaCostOf("{B}"),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Frozen Shade", func() mage.Card {
		c := mage.NewCreature("Frozen Shade", "{2}{B}", 0, 1, "Shade")
		// {B}: +1/+1 until end of turn
		ab := mage.NewActivatedAbility(
			mage.BoostUntilEndOfTurn(mage.Fixed(1), mage.Fixed(1), mage.SelectSource),
			mage.ManaCostOf("{B}"),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Hypnotic Specter", func() mage.Card {
		c := mage.NewCreature("Hypnotic Specter", "{1}{B}{B}", 2, 2, "Specter")
		c.AddAbility(mage.HasKeyword(mage.Flying))
		c.AddAbility(mage.DealsDamageToOpponentTrigger(mage.DiscardRandom(1), false))
		return c
	})

	mage.Register("Nether Shadow", func() mage.Card {
		c := mage.NewCreature("Nether Shadow", "{B}{B}", 1, 1, "Spirit")
		c.AddAbility(mage.HasKeyword(mage.Haste))
		// At beginning of your upkeep, if Nether Shadow is in your graveyard
		// with three or more creature cards above it, put it onto the battlefield.
		c.AddAbility(mage.GraveyardReturnIfCreaturesAbove(3))
		return c
	})

	mage.Register("Nightmare", func() mage.Card {
		c := mage.NewCreature("Nightmare", "{5}{B}", 0, 0, "Nightmare", "Horse")
		c.AddAbility(mage.HasKeyword(mage.Flying))
		// P/T equal to number of Swamps you control
		c.AddAbility(mage.StaticAbility(
			mage.PTEqualsControlledCount(mage.HasSubType("Swamp")),
		))
		return c
	})

	mage.Register("Plague Rats", func() mage.Card {
		c := mage.NewCreature("Plague Rats", "{2}{B}", 0, 0, "Rat")
		// P/T equal to number of Plague Rats on the battlefield
		c.AddAbility(mage.StaticAbility(
			mage.PTEqualsCount(mage.Named("Plague Rats")),
		))
		return c
	})

	mage.Register("Royal Assassin", func() mage.Card {
		c := mage.NewCreature("Royal Assassin", "{1}{B}{B}", 1, 1, "Human", "Assassin")
		// {T}: Destroy target tapped creature
		ab := mage.NewActivatedAbility(
			mage.DestroyTarget(),
			mage.TapSourceCost(),
		
			mage.WithTarget(mage.TargetCreature(mage.IsTapped)),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Scathe Zombies", func() mage.Card {
		c := mage.NewCreature("Scathe Zombies", "{2}{B}", 2, 2, "Zombie")
		return c
	})

	mage.Register("Sengir Vampire", func() mage.Card {
		c := mage.NewCreature("Sengir Vampire", "{3}{B}{B}", 4, 4, "Vampire")
		c.AddAbility(mage.HasKeyword(mage.Flying))
		// Whenever a creature dealt damage by Sengir Vampire this turn dies, put a +1/+1 counter on Sengir Vampire
		c.AddAbility(mage.CreatureDealtDamageBySourceDiesTrigger(mage.AddCounters(mage.P1P1, mage.Fixed(1), mage.SelectSource), false))
		return c
	})

	mage.Register("Will-o'-the-Wisp", func() mage.Card {
		c := mage.NewCreature("Will-o'-the-Wisp", "{B}", 0, 1, "Spirit")
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
		c := mage.NewCreature("Lord of the Pit", "{4}{B}{B}{B}", 7, 7, "Demon")
		c.AddAbility(mage.HasKeyword(mage.Flying))
		c.AddAbility(mage.HasKeyword(mage.Trample))
		// At the beginning of your upkeep, sacrifice a creature other than Lord of the Pit.
		// If you can't, Lord of the Pit deals 7 damage to you.
		c.AddAbility(mage.BeginningOfUpkeepTrigger(mage.SacrificeCreatureOrDamage(7), false))
		return c
	})

	// ===== RED CREATURES =====

	mage.Register("Dragon Whelp", func() mage.Card {
		c := mage.NewCreature("Dragon Whelp", "{2}{R}{R}", 2, 3, "Dragon")
		c.AddAbility(mage.HasKeyword(mage.Flying))
		// {R}: +1/+0 until end of turn. If activated 4+ times, destroy at EOT.
		ab := mage.NewActivatedAbility(
			mage.BoostUntilEndOfTurn(mage.Fixed(1), mage.Fixed(0), mage.SelectSource),
			mage.ManaCostOf("{R}"),
		
			mage.WithEffect(mage.MarkDestroyAtEOTAfterNActivations(4)),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Dwarven Warriors", func() mage.Card {
		c := mage.NewCreature("Dwarven Warriors", "{2}{R}", 1, 1, "Dwarf", "Warrior")
		// {T}: Target creature with power 2 or less can't be blocked this turn.
		ab := mage.NewActivatedAbility(
			mage.MakeUnblockableUntilEndOfTurn(),
			mage.TapSourceCost(),
		
			mage.WithTarget(mage.TargetCreature()),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Earth Elemental", func() mage.Card {
		c := mage.NewCreature("Earth Elemental", "{3}{R}{R}", 4, 5, "Elemental")
		return c
	})

	mage.Register("Fire Elemental", func() mage.Card {
		c := mage.NewCreature("Fire Elemental", "{3}{R}{R}", 5, 4, "Elemental")
		return c
	})

	mage.Register("Goblin Balloon Brigade", func() mage.Card {
		c := mage.NewCreature("Goblin Balloon Brigade", "{R}", 1, 1, "Goblin", "Warrior")
		// {R}: Goblin Balloon Brigade gains flying until end of turn.
		ab := mage.NewActivatedAbility(
			mage.GrantKeywordUntilEndOfTurn(mage.Flying, mage.SelectSource),
			mage.ManaCostOf("{R}"),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Granite Gargoyle", func() mage.Card {
		c := mage.NewCreature("Granite Gargoyle", "{2}{R}", 2, 2, "Gargoyle")
		c.AddAbility(mage.HasKeyword(mage.Flying))
		// {R}: +0/+1 until end of turn
		ab := mage.NewActivatedAbility(
			mage.BoostUntilEndOfTurn(mage.Fixed(0), mage.Fixed(1), mage.SelectSource),
			mage.ManaCostOf("{R}"),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Gray Ogre", func() mage.Card {
		c := mage.NewCreature("Gray Ogre", "{2}{R}", 2, 2, "Ogre")
		return c
	})

	mage.Register("Hill Giant", func() mage.Card {
		c := mage.NewCreature("Hill Giant", "{3}{R}", 3, 3, "Giant")
		return c
	})

	mage.Register("Hurloon Minotaur", func() mage.Card {
		c := mage.NewCreature("Hurloon Minotaur", "{1}{R}{R}", 2, 3, "Minotaur")
		return c
	})

	mage.Register("Ironclaw Orcs", func() mage.Card {
		c := mage.NewCreature("Ironclaw Orcs", "{1}{R}", 2, 2, "Orc")
		return c
	})

	mage.Register("Mons's Goblin Raiders", func() mage.Card {
		c := mage.NewCreature("Mons's Goblin Raiders", "{R}", 1, 1, "Goblin")
		return c
	})

	mage.Register("Roc of Kher Ridges", func() mage.Card {
		c := mage.NewCreature("Roc of Kher Ridges", "{3}{R}", 3, 3, "Bird")
		c.AddAbility(mage.HasKeyword(mage.Flying))
		return c
	})

	mage.Register("Shivan Dragon", func() mage.Card {
		c := mage.NewCreature("Shivan Dragon", "{4}{R}{R}", 5, 5, "Dragon")
		c.AddAbility(mage.HasKeyword(mage.Flying))
		// {R}: +1/+0 until end of turn (firebreathing)
		ab := mage.NewActivatedAbility(
			mage.BoostUntilEndOfTurn(mage.Fixed(1), mage.Fixed(0), mage.SelectSource),
			mage.ManaCostOf("{R}"),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Uthden Troll", func() mage.Card {
		c := mage.NewCreature("Uthden Troll", "{2}{R}", 2, 2, "Troll")
		// {R}: Regenerate Uthden Troll.
		ab := mage.NewActivatedAbility(
			mage.RegenerateSource(),
			mage.ManaCostOf("{R}"),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Sedge Troll", func() mage.Card {
		c := mage.NewCreature("Sedge Troll", "{2}{R}", 2, 2, "Troll")
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
		c := mage.NewCreature("Two-Headed Giant of Foriys", "{4}{R}", 4, 4, "Giant")
		c.AddAbility(mage.HasKeyword(mage.Trample))
		// Two-Headed Giant of Foriys can block an additional creature each combat.
		c.AddAbility(mage.HasKeyword(mage.CanBlockAdditional))
		return c
	})

	mage.Register("Goblin King", func() mage.Card {
		c := mage.NewCreature("Goblin King", "{1}{R}{R}", 2, 2, "Goblin")
		// Other Goblin creatures get +1/+1 and mountainwalk
		c.AddAbility(mage.StaticAbility(
			mage.BoostAllCreatures(1, 1, mage.HasSubType("Goblin")),
			mage.GrantKeywordToAll(mage.Mountainwalk, mage.HasSubType("Goblin")),
		))
		return c
	})

	// ===== GREEN CREATURES =====

	mage.Register("Craw Wurm", func() mage.Card {
		c := mage.NewCreature("Craw Wurm", "{4}{G}{G}", 6, 4, "Wurm")
		return c
	})

	mage.Register("Elvish Archers", func() mage.Card {
		c := mage.NewCreature("Elvish Archers", "{1}{G}", 2, 1, "Elf", "Archer")
		c.AddAbility(mage.HasKeyword(mage.FirstStrike))
		return c
	})

	mage.Register("Force of Nature", func() mage.Card {
		c := mage.NewCreature("Force of Nature", "{2}{G}{G}{G}{G}", 8, 8, "Elemental")
		c.AddAbility(mage.HasKeyword(mage.Trample))
		// At the beginning of your upkeep, Force of Nature deals 8 damage to you
		// unless you pay {G}{G}{G}{G}.
		c.AddAbility(mage.BeginningOfUpkeepTrigger(mage.DealDamageToPlayers(mage.Fixed(8), mage.SelectController()), false))
		return c
	})

	mage.Register("Fungusaur", func() mage.Card {
		c := mage.NewCreature("Fungusaur", "{3}{G}", 2, 2, "Fungus", "Dinosaur")
		// Whenever Fungusaur is dealt damage, put a +1/+1 counter on it.
		c.AddAbility(mage.WhenDamageDealtToThisTrigger(mage.AddCounters(mage.P1P1, mage.Fixed(1), mage.SelectSource), false))
		return c
	})

	// Grizzly Bears already registered in creatures.go

	mage.Register("Ironroot Treefolk", func() mage.Card {
		c := mage.NewCreature("Ironroot Treefolk", "{4}{G}", 3, 5, "Treefolk")
		return c
	})

	mage.Register("Llanowar Elves", func() mage.Card {
		c := mage.NewCreature("Llanowar Elves", "{G}", 1, 1, "Elf", "Druid")
		c.AddAbility(mage.NewManaAbility(mage.Green))
		return c
	})

	mage.Register("Scryb Sprites", func() mage.Card {
		c := mage.NewCreature("Scryb Sprites", "{G}", 1, 1, "Faerie")
		c.AddAbility(mage.HasKeyword(mage.Flying))
		return c
	})

	mage.Register("Shanodin Dryads", func() mage.Card {
		c := mage.NewCreature("Shanodin Dryads", "{G}", 1, 1, "Nymph", "Dryad")
		c.AddAbility(mage.HasKeyword(mage.Forestwalk))
		return c
	})

	mage.Register("Timber Wolves", func() mage.Card {
		c := mage.NewCreature("Timber Wolves", "{G}", 1, 1, "Wolf")
		c.AddAbility(mage.HasKeyword(mage.Banding))
		return c
	})

	mage.Register("Thicket Basilisk", func() mage.Card {
		c := mage.NewCreature("Thicket Basilisk", "{3}{G}{G}", 2, 4, "Basilisk")
		// Whenever Thicket Basilisk deals damage to a creature, destroy that creature at end of combat
		// Simplified: has deathtouch-like behavior
		c.AddAbility(mage.HasKeyword(mage.Deathtouch))
		return c
	})

	mage.Register("War Mammoth", func() mage.Card {
		c := mage.NewCreature("War Mammoth", "{3}{G}", 3, 3, "Elephant")
		c.AddAbility(mage.HasKeyword(mage.Trample))
		return c
	})

	mage.Register("Giant Spider", func() mage.Card {
		c := mage.NewCreature("Giant Spider", "{3}{G}", 2, 4, "Spider")
		c.AddAbility(mage.HasKeyword(mage.Reach))
		return c
	})

	mage.Register("Birds of Paradise", func() mage.Card {
		c := mage.NewCreature("Birds of Paradise", "{G}", 0, 1, "Bird")
		c.AddAbility(mage.HasKeyword(mage.Flying))
		// {T}: Add one mana of any color.
		c.AddAbility(mage.NewAnyColorManaAbility())
		return c
	})

	mage.Register("Cockatrice", func() mage.Card {
		c := mage.NewCreature("Cockatrice", "{3}{G}{G}", 2, 4, "Cockatrice")
		c.AddAbility(mage.HasKeyword(mage.Flying))
		c.AddAbility(mage.HasKeyword(mage.Deathtouch))
		return c
	})

	mage.Register("Verduran Enchantress", func() mage.Card {
		c := mage.NewCreature("Verduran Enchantress", "{1}{G}{G}", 0, 2, "Human", "Druid")
		// Whenever you cast an enchantment spell, draw a card (any color enchantment)
		c.AddAbility(mage.WheneverEnchantmentCastTrigger(mage.DrawCards(mage.Fixed(1)), true))
		return c
	})

	mage.Register("Keldon Warlord", func() mage.Card {
		c := mage.NewCreature("Keldon Warlord", "{2}{R}{R}", 0, 0, "Human", "Barbarian")
		// P/T equal to number of non-Wall creatures you control
		c.AddAbility(mage.StaticAbility(
			mage.PTEqualsControlledCount(mage.And(mage.IsCreature, mage.Not(mage.HasSubType("Wall")))),
		))
		return c
	})

	// ===== WALLS =====

	mage.Register("Wall of Air", func() mage.Card {
		c := mage.NewCreature("Wall of Air", "{1}{U}{U}", 1, 5, "Wall")
		c.AddAbility(mage.HasKeyword(mage.Defender))
		c.AddAbility(mage.HasKeyword(mage.Flying))
		return c
	})

	mage.Register("Wall of Bone", func() mage.Card {
		c := mage.NewCreature("Wall of Bone", "{2}{B}", 1, 4, "Wall", "Skeleton")
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
		c := mage.NewCreature("Wall of Brambles", "{2}{G}", 2, 3, "Wall", "Plant")
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
		c := mage.NewCreature("Wall of Fire", "{1}{R}{R}", 0, 5, "Wall")
		c.AddAbility(mage.HasKeyword(mage.Defender))
		// {R}: +1/+0 until end of turn
		ab := mage.NewActivatedAbility(
			mage.BoostUntilEndOfTurn(mage.Fixed(1), mage.Fixed(0), mage.SelectSource),
			mage.ManaCostOf("{R}"),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Wall of Ice", func() mage.Card {
		c := mage.NewCreature("Wall of Ice", "{2}{G}", 0, 7, "Wall")
		c.AddAbility(mage.HasKeyword(mage.Defender))
		return c
	})

	mage.Register("Wall of Stone", func() mage.Card {
		c := mage.NewCreature("Wall of Stone", "{1}{R}{R}", 0, 8, "Wall")
		c.AddAbility(mage.HasKeyword(mage.Defender))
		return c
	})

	mage.Register("Wall of Swords", func() mage.Card {
		c := mage.NewCreature("Wall of Swords", "{3}{W}", 3, 5, "Wall")
		c.AddAbility(mage.HasKeyword(mage.Defender))
		c.AddAbility(mage.HasKeyword(mage.Flying))
		return c
	})

	mage.Register("Wall of Water", func() mage.Card {
		c := mage.NewCreature("Wall of Water", "{1}{U}{U}", 0, 5, "Wall")
		c.AddAbility(mage.HasKeyword(mage.Defender))
		// {U}: +1/+0 until end of turn
		ab := mage.NewActivatedAbility(
			mage.BoostUntilEndOfTurn(mage.Fixed(1), mage.Fixed(0), mage.SelectSource),
			mage.ManaCostOf("{U}"),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Wall of Wood", func() mage.Card {
		c := mage.NewCreature("Wall of Wood", "{G}", 0, 3, "Wall")
		c.AddAbility(mage.HasKeyword(mage.Defender))
		return c
	})

	// ===== ARTIFACT CREATURES =====

	mage.Register("Obsianus Golem", func() mage.Card {
		c := mage.NewCreature("Obsianus Golem", "{6}", 4, 6, "Golem")
		c.AddType(mage.TypeArtifact)
		return c
	})

	mage.Register("Clockwork Beast", func() mage.Card {
		c := mage.NewCreature("Clockwork Beast", "{6}", 0, 4, "Beast")
		c.AddType(mage.TypeArtifact)
		// Enters with 7 +1/+0 counters
		c.AddAbility(mage.EntersBattlefieldTrigger(
			mage.AddCounters(mage.P1P0, mage.Fixed(7), mage.SelectSource), false,
		))
		// Loses a +1/+0 counter whenever it attacks
		c.AddAbility(mage.AttacksTrigger(
			mage.RemoveCountersFromSource(mage.P1P0, 1), false,
		))
		return c
	})

	mage.Register("Juggernaut", func() mage.Card {
		c := mage.NewCreature("Juggernaut", "{4}", 5, 3, "Juggernaut")
		c.AddType(mage.TypeArtifact)
		// Juggernaut attacks each combat if able. Can't be blocked by Walls.
		c.AddAbility(mage.HasKeyword(mage.CantBeBlockedByWalls))
		return c
	})

	mage.Register("Living Wall", func() mage.Card {
		c := mage.NewCreature("Living Wall", "{4}", 0, 6, "Wall")
		c.AddType(mage.TypeArtifact)
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
		c := mage.NewCreature("Lord of Atlantis", "{U}{U}", 2, 2, "Merfolk")
		// Other Merfolk creatures get +1/+1 and islandwalk
		c.AddAbility(mage.StaticAbility(
			mage.BoostAllCreatures(1, 1, mage.HasSubType("Merfolk")),
			mage.GrantKeywordToAll(mage.Islandwalk, mage.HasSubType("Merfolk")),
		))
		return c
	})

	mage.Register("Zombie Master", func() mage.Card {
		c := mage.NewCreature("Zombie Master", "{1}{B}{B}", 2, 3, "Zombie")
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
		c := mage.NewCreature("Samite Healer", "{1}{W}", 1, 1, "Human", "Cleric")
		// {T}: Prevent the next 1 damage that would be dealt to any target this turn.
		ab := mage.NewActivatedAbility(
			mage.PreventDamageToTarget(mage.Fixed(1)),
			mage.TapSourceCost(),
		
			mage.WithTarget(mage.TargetAnyTarget()),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Ley Druid", func() mage.Card {
		c := mage.NewCreature("Ley Druid", "{2}{G}", 1, 1, "Human", "Druid")
		// {T}: Untap target land
		ab := mage.NewActivatedAbility(
			mage.UntapTarget(),
			mage.TapSourceCost(),
		
			mage.WithTarget(mage.TargetLand()),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Sea Serpent", func() mage.Card {
		c := mage.NewCreature("Sea Serpent", "{5}{U}", 5, 5, "Serpent")
		c.AddAbility(mage.HasKeyword(mage.Islandwalk))
		c.AddAbility(mage.SacrificeUnlessLand("Island"))
		return c
	})

	mage.Register("Nettling Imp", func() mage.Card {
		c := mage.NewCreature("Nettling Imp", "{2}{B}", 1, 1, "Imp")
		// {T}: Target non-Wall creature the active player controls attacks this
		// turn if able. (Simplified: just tap Nettling Imp targeting a creature)
		ab := mage.NewActivatedAbility(
			mage.TapTarget(),
			mage.TapSourceCost(),
		
			mage.WithTarget(mage.TargetCreature()),
		)
		c.AddAbility(ab)
		return c
	})

	mage.Register("Scavenging Ghoul", func() mage.Card {
		c := mage.NewCreature("Scavenging Ghoul", "{3}{B}", 2, 2, "Zombie")
		// Whenever another creature dies, put a +1/+1 counter on Scavenging Ghoul
		c.AddAbility(mage.AnyCreatureDiesTrigger(mage.AddCounters(mage.P1P1, mage.Fixed(1), mage.SelectSource), true))
		return c
	})

	mage.Register("Rock Hydra", func() mage.Card {
		c := mage.NewCreature("Rock Hydra", "{X}{R}{R}", 0, 0, "Hydra")
		// Enters with X +1/+1 counters (replacement effect)
		c.AddAbility(mage.EntersWithXCounters(mage.P1P1))
		return c
	})

	mage.Register("Vesuvan Doppelganger", func() mage.Card {
		c := mage.NewCreature("Vesuvan Doppelganger", "{3}{U}{U}", 0, 0, "Shapeshifter")
		return c
	})

	mage.Register("Personal Incarnation", func() mage.Card {
		c := mage.NewCreature("Personal Incarnation", "{3}{W}{W}{W}", 6, 6, "Avatar", "Incarnation")
		return c
	})

	mage.Register("Veteran Bodyguard", func() mage.Card {
		c := mage.NewCreature("Veteran Bodyguard", "{3}{W}{W}", 2, 5, "Human")
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
