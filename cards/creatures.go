package cards

import "github.com/mage/mage"

func init() {
	registerCreatures()
}

func registerCreatures() {
	mage.Register("Grizzly Bears", func() mage.Card {
		c := mage.NewCreature("Grizzly Bears", "{1}{G}", "Bear")
		c.Power_ = 2
		c.Toughness_ = 2
		return c
	})

	mage.Register("Serra Angel", func() mage.Card {
		c := mage.NewCreature("Serra Angel", "{3}{W}{W}", "Angel")
		c.Power_ = 4
		c.Toughness_ = 4
		c.AddAbility(mage.HasKeyword(mage.Flying))
		c.AddAbility(mage.HasKeyword(mage.Vigilance))
		return c
	})

	mage.Register("Elvish Mystic", func() mage.Card {
		c := mage.NewCreature("Elvish Mystic", "{G}", "Elf", "Druid")
		c.Power_ = 1
		c.Toughness_ = 1
		c.AddAbility(mage.NewManaAbility(mage.Green))
		return c
	})

	// First striker for testing
	mage.Register("White Knight", func() mage.Card {
		c := mage.NewCreature("White Knight", "{W}{W}", "Human", "Knight")
		c.Power_ = 2
		c.Toughness_ = 2
		c.AddAbility(mage.HasKeyword(mage.FirstStrike))
		c.AddAbility(mage.ProtectionFromColor(mage.Black))
		return c
	})

	// Double striker for testing
	mage.Register("Fencing Ace", func() mage.Card {
		c := mage.NewCreature("Fencing Ace", "{1}{W}", "Human", "Soldier")
		c.Power_ = 1
		c.Toughness_ = 1
		c.AddAbility(mage.HasKeyword(mage.DoubleStrike))
		return c
	})

	// A bigger double striker
	mage.Register("Boros Swiftblade", func() mage.Card {
		c := mage.NewCreature("Boros Swiftblade", "{R}{W}", "Human", "Soldier")
		c.Power_ = 1
		c.Toughness_ = 2
		c.AddAbility(mage.HasKeyword(mage.DoubleStrike))
		return c
	})

	// Hexproof creature for testing
	mage.Register("Gladecover Scout", func() mage.Card {
		c := mage.NewCreature("Gladecover Scout", "{G}", "Elf", "Scout")
		c.Power_ = 1
		c.Toughness_ = 1
		c.AddAbility(mage.HasKeyword(mage.Hexproof))
		return c
	})

	// Shroud creature for testing
	mage.Register("Blurred Mongoose", func() mage.Card {
		c := mage.NewCreature("Blurred Mongoose", "{1}{G}", "Mongoose")
		c.Power_ = 2
		c.Toughness_ = 1
		c.AddAbility(mage.HasKeyword(mage.Shroud))
		return c
	})

	// Protection from red for testing
	mage.Register("Kor Firewalker", func() mage.Card {
		c := mage.NewCreature("Kor Firewalker", "{W}{W}", "Kor", "Soldier")
		c.Power_ = 2
		c.Toughness_ = 2
		c.AddAbility(mage.ProtectionFromColor(mage.Red))
		return c
	})

	// Vanilla 3/3 for testing combat
	mage.Register("Centaur Courser", func() mage.Card {
		c := mage.NewCreature("Centaur Courser", "{2}{G}", "Centaur", "Warrior")
		c.Power_ = 3
		c.Toughness_ = 3
		return c
	})

	// 1/1 red creature for testing protection blocking
	mage.Register("Goblin Piker", func() mage.Card {
		c := mage.NewCreature("Goblin Piker", "{1}{R}", "Goblin", "Warrior")
		c.Power_ = 2
		c.Toughness_ = 1
		return c
	})

	// Flying creature for testing
	mage.Register("Wind Drake", func() mage.Card {
		c := mage.NewCreature("Wind Drake", "{2}{U}", "Drake")
		c.Power_ = 2
		c.Toughness_ = 2
		c.AddAbility(mage.HasKeyword(mage.Flying))
		return c
	})
}
