package ron

import "github.com/benprew/mage-go/pkg/mage"

type textWordChoicePlan struct {
	name  string
	color bool
}
type textWordBindings struct {
	colors *mage.ColorWordPairBinding
	lands  *mage.BasicLandWordPairBinding
}

func (p *textWordChoicePlan) binding(bindings instanceBindings) textWordBindings {
	if result, ok := bindings.textWords[p]; ok {
		return result
	}
	var result textWordBindings
	if p.color {
		result.colors = mage.BindColorWordPair(p.name)
	} else {
		result.lands = mage.BindBasicLandWordPair(p.name)
	}
	bindings.textWords[p] = result
	return result
}
func compileTextWordChoice(n Node, ctx scope) (effectFactory, *textWordChoicePlan, error) {
	if _, err := symbol(n); err != nil {
		return nil, nil, err
	}
	if n.Name != "ChooseTwoColorWords" && n.Name != "ChooseTwoBasicLandTypes" {
		return nil, nil, unsupported(n, "text word choice")
	}
	plan := &textWordChoicePlan{name: "text words " + n.Position.String(), color: n.Name == "ChooseTwoColorWords"}
	return func(bindings instanceBindings) mage.Effect {
		if bindings.textWords == nil {
			bindings.textWords = make(map[*textWordChoicePlan]textWordBindings)
		}
		result := plan.binding(bindings)
		if plan.color {
			return mage.ChooseColorWordPair(ctx.player.build(bindings), result.colors)
		}
		return mage.ChooseBasicLandWordPair(ctx.player.build(bindings), result.lands)
	}, plan, nil
}

type colorWordRefFactory func(instanceBindings) mage.ColorWordRef
type landWordRefFactory func(instanceBindings) mage.BasicLandWordRef

func compileColorWordRef(n Node, ctx scope) (colorWordRefFactory, error) {
	if n.Kind != Symbol || (ctx.colorWords == nil || !ctx.colorWords.color) || (n.Name != "TheFirstChosenColorWord" && n.Name != "TheSecondChosenColorWord") {
		return nil, problem(n, "unbound or incompatible color word reference")
	}
	return func(bindings instanceBindings) mage.ColorWordRef {
		pair := ctx.colorWords.binding(bindings).colors
		if n.Name == "TheFirstChosenColorWord" {
			return mage.FirstColorWord(pair)
		}
		return mage.SecondColorWord(pair)
	}, nil
}
func compileLandWordRef(n Node, ctx scope) (landWordRefFactory, error) {
	if n.Kind != Symbol || (ctx.landWords == nil || ctx.landWords.color) || (n.Name != "TheFirstChosenLandType" && n.Name != "TheSecondChosenLandType") {
		return nil, problem(n, "unbound or incompatible land word reference")
	}
	return func(bindings instanceBindings) mage.BasicLandWordRef {
		pair := ctx.landWords.binding(bindings).lands
		if n.Name == "TheFirstChosenLandType" {
			return mage.FirstBasicLandWord(pair)
		}
		return mage.SecondBasicLandWord(pair)
	}, nil
}
