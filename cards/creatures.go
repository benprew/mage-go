package cards

import "github.com/mage/mage"

func init() {
	registerCreatures()
}

func registerCreatures() {
	mage.Register("Grizzly Bears", func() mage.Card {
		c := mage.NewCreature("Grizzly Bears", "{1}{G}", 2, 2, "Bear")
		return c
	})

	mage.Register("Serra Angel", func() mage.Card {
		c := mage.NewCreature("Serra Angel", "{3}{W}{W}", 4, 4, "Angel")
		c.AddAbility(mage.NewKeywordAbility(mage.Flying))
		c.AddAbility(mage.NewKeywordAbility(mage.Vigilance))
		return c
	})

	mage.Register("Elvish Mystic", func() mage.Card {
		c := mage.NewCreature("Elvish Mystic", "{G}", 1, 1, "Elf", "Druid")
		c.AddAbility(mage.NewManaAbility(mage.Green))
		return c
	})

	// First striker for testing
	mage.Register("White Knight", func() mage.Card {
		c := mage.NewCreature("White Knight", "{W}{W}", 2, 2, "Human", "Knight")
		c.AddAbility(mage.NewKeywordAbility(mage.FirstStrike))
		c.AddAbility(mage.ProtectionFromColor(mage.Black))
		return c
	})

	// Double striker for testing
	mage.Register("Fencing Ace", func() mage.Card {
		c := mage.NewCreature("Fencing Ace", "{1}{W}", 1, 1, "Human", "Soldier")
		c.AddAbility(mage.NewKeywordAbility(mage.DoubleStrike))
		return c
	})

	// A bigger double striker
	mage.Register("Boros Swiftblade", func() mage.Card {
		c := mage.NewCreature("Boros Swiftblade", "{R}{W}", 1, 2, "Human", "Soldier")
		c.AddAbility(mage.NewKeywordAbility(mage.DoubleStrike))
		return c
	})

	// Hexproof creature for testing
	mage.Register("Gladecover Scout", func() mage.Card {
		c := mage.NewCreature("Gladecover Scout", "{G}", 1, 1, "Elf", "Scout")
		c.AddAbility(mage.NewKeywordAbility(mage.Hexproof))
		return c
	})

	// Shroud creature for testing
	mage.Register("Blurred Mongoose", func() mage.Card {
		c := mage.NewCreature("Blurred Mongoose", "{1}{G}", 2, 1, "Mongoose")
		c.AddAbility(mage.NewKeywordAbility(mage.Shroud))
		return c
	})

	// Protection from red for testing
	mage.Register("Kor Firewalker", func() mage.Card {
		c := mage.NewCreature("Kor Firewalker", "{W}{W}", 2, 2, "Kor", "Soldier")
		c.AddAbility(mage.ProtectionFromColor(mage.Red))
		return c
	})

	// Vanilla 3/3 for testing combat
	mage.Register("Centaur Courser", func() mage.Card {
		c := mage.NewCreature("Centaur Courser", "{2}{G}", 3, 3, "Centaur", "Warrior")
		return c
	})

	// 1/1 red creature for testing protection blocking
	mage.Register("Goblin Piker", func() mage.Card {
		c := mage.NewCreature("Goblin Piker", "{1}{R}", 2, 1, "Goblin", "Warrior")
		return c
	})

	// Flying creature for testing
	mage.Register("Wind Drake", func() mage.Card {
		c := mage.NewCreature("Wind Drake", "{2}{U}", 2, 2, "Drake")
		c.AddAbility(mage.NewKeywordAbility(mage.Flying))
		return c
	})
}
