package arabian

import . "github.com/mage/mage/pkg/mage"

func init() {
	registerArtifacts()
}

func registerArtifacts() {
	Register("Aladdin's Lamp", func() Card {
		return NewArtifact("Aladdin's Lamp", "{10}")
	})

	Register("Aladdin's Ring", func() Card {
		return NewArtifact("Aladdin's Ring", "{8}")
	})

	Register("Bottle of Suleiman", func() Card {
		return NewArtifact("Bottle of Suleiman", "{4}")
	})

	Register("City in a Bottle", func() Card {
		return NewArtifact("City in a Bottle", "{2}")
	})

	Register("Ebony Horse", func() Card {
		return NewArtifact("Ebony Horse", "{3}")
	})

	Register("Flying Carpet", func() Card {
		return NewArtifact("Flying Carpet", "{4}")
	})

	Register("Jandor's Ring", func() Card {
		return NewArtifact("Jandor's Ring", "{6}")
	})

	Register("Jandor's Saddlebags", func() Card {
		return NewArtifact("Jandor's Saddlebags", "{2}")
	})

	Register("Jeweled Bird", func() Card {
		return NewArtifact("Jeweled Bird", "{1}")
	})

	Register("Pyramids", func() Card {
		return NewArtifact("Pyramids", "{6}")
	})

	Register("Ring of Ma'rûf", func() Card {
		return NewArtifact("Ring of Ma'rûf", "{5}")
	})

	Register("Sandals of Abdallah", func() Card {
		return NewArtifact("Sandals of Abdallah", "{4}")
	})
}
