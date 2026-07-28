package fourthedition

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/dsl"
)

func init() {
	registerArtifacts()
}

func registerArtifacts() {

	// Dingus Egg {4}
	// Artifact
	// Whenever a land is put into a graveyard from the battlefield, Dingus Egg deals 2 damage to that land's controller.
	Register("Dingus Egg", func() Card {
		return NewArtifact("Dingus Egg", "{4}",
			WithAbility(
				NewTriggered(EvtZoneChange, false,
					DealDamageToPlayers(Fixed(2), SelectTargetPermanentController()),
				).SetCondition(func(evt *GameEvent, g GameReader, _, _ uuid.UUID) bool {
					card := g.FindCardAnywhere(evt.SourceID)
					return card != nil && card.HasType(TypeLand)
				}).AndConditionData(EventZoneChangeMatches{From: ZoneBattlefield, To: ZoneGraveyard}),
			),
		)
	})

	// Fellwar Stone {2}
	// Artifact
	// {T}: Add one mana of any color that a land an opponent controls could produce.
	Register("Fellwar Stone", func() Card {
		return NewArtifact("Fellwar Stone", "{2}",
			WithStaticAbility(SourceHasManaAbilitiesOpponentLandsCouldProduce()),
		)
	})

	// Library of Leng {1}
	// Artifact
	// You have no maximum hand size.
	// If an effect causes you to discard a card, discard it, but you may put it on top of your library instead of into your graveyard.
	Register("Library of Leng", func() Card {
		return NewArtifact("Library of Leng", "{1}",
			WithStaticAbility(
				FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
					source := g.FindPermanent(sourceID)
					if source != nil {
						g.SetNoMaximumHandSize(source.ControllerID())
					}
					return nil
				}),
				DiscardToLibraryReplacement(),
			),
		)
	})
}
