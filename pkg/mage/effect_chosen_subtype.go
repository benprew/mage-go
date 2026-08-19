package mage

import . "github.com/benprew/mage-go/pkg/mage/core"

type chooseRandomCreatureSubtypeFromTargetLibraryEffect struct{}

// ChooseRandomCreatureSubtypeFromTargetLibrary creates an effect that chooses
// uniformly among the distinct creature subtypes in the targeted player's
// library and stores the result on the source permanent.
func ChooseRandomCreatureSubtypeFromTargetLibrary() Effect {
	return &chooseRandomCreatureSubtypeFromTargetLibraryEffect{}
}

func (*chooseRandomCreatureSubtypeFromTargetLibraryEffect) Text() string {
	return "choose a random creature type from target opponent's library"
}
func (*chooseRandomCreatureSubtypeFromTargetLibraryEffect) Properties() EffectProperties {
	return EffectProperties{}
}
func (*chooseRandomCreatureSubtypeFromTargetLibraryEffect) Apply(ctx *EffectContext) error {
	if len(ctx.Targets) == 0 {
		return nil
	}
	player := ctx.Game.GetPlayer(ctx.Targets[0])
	source := ctx.Game.FindPermanent(ctx.SourceID)
	if player == nil || source == nil {
		return nil
	}
	seen := make(map[string]bool)
	var creatureTypes []string
	for _, card := range player.Library() {
		if !card.HasType(TypeCreature) {
			continue
		}
		for _, subtype := range card.SubTypes() {
			if !seen[subtype] {
				seen[subtype] = true
				creatureTypes = append(creatureTypes, subtype)
			}
		}
	}
	if len(creatureTypes) > 0 {
		source.ChosenSubtype = creatureTypes[ctx.Game.RandIntn(len(creatureTypes))]
	}
	return nil
}
