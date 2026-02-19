package arabian

import "github.com/mage/mage/pkg/mage"

func init() {
	registerArtifacts()
}

func registerArtifacts() {
	mage.Register("Aladdin's Lamp", func() mage.Card {
		return mage.NewArtifact("Aladdin's Lamp", "{10}")
	})

	mage.Register("Aladdin's Ring", func() mage.Card {
		return mage.NewArtifact("Aladdin's Ring", "{8}")
	})

	mage.Register("Bottle of Suleiman", func() mage.Card {
		return mage.NewArtifact("Bottle of Suleiman", "{4}")
	})

	mage.Register("City in a Bottle", func() mage.Card {
		return mage.NewArtifact("City in a Bottle", "{2}")
	})

	mage.Register("Ebony Horse", func() mage.Card {
		return mage.NewArtifact("Ebony Horse", "{3}")
	})

	mage.Register("Flying Carpet", func() mage.Card {
		return mage.NewArtifact("Flying Carpet", "{4}")
	})

	mage.Register("Jandor's Ring", func() mage.Card {
		return mage.NewArtifact("Jandor's Ring", "{6}")
	})

	mage.Register("Jandor's Saddlebags", func() mage.Card {
		return mage.NewArtifact("Jandor's Saddlebags", "{2}")
	})

	mage.Register("Jeweled Bird", func() mage.Card {
		return mage.NewArtifact("Jeweled Bird", "{1}")
	})

	mage.Register("Pyramids", func() mage.Card {
		return mage.NewArtifact("Pyramids", "{6}")
	})

	mage.Register("Ring of Ma'rûf", func() mage.Card {
		return mage.NewArtifact("Ring of Ma'rûf", "{5}")
	})

	mage.Register("Sandals of Abdallah", func() mage.Card {
		return mage.NewArtifact("Sandals of Abdallah", "{4}")
	})
}
