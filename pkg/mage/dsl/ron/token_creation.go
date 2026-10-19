package ron

import (
	"slices"
	"strings"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileTokenCreation(n Node, ctx scope) (effectFactory, error) {
	return compileTokenCreationWithResult(n, ctx, nil)
}

func compileTokenCreationWithResult(n Node, ctx scope, result *resolutionTokensPlan) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	instructions, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(instructions) != 1 {
		return nil, problem(n, "token creation requires one instruction")
	}
	modifiers, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(modifiers) != 0 {
		return nil, unsupported(n.Args[1], "token creation modifiers")
	}
	tokenNode := instructions[0]
	var count valueFactory = func(instanceBindings) mage.ResolutionValue { return mage.ValueAtResolution(mage.Fixed(1)) }
	if tokenNode.Name == "NumberTokens" {
		if err := arity(tokenNode, 2); err != nil {
			return nil, err
		}
		count, err = compileInstanceValue(tokenNode.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		tokenNode = tokenNode.Args[1]
	}
	if tokenNode.Name == "TokenCopyOfPermanent" {
		if result != nil {
			return nil, unsupported(tokenNode, "saved token copy result")
		}
		if err := arity(tokenNode, 2); err != nil {
			return nil, err
		}
		modifier := tokenNode.Args[1]
		if modifier.Kind != Symbol || modifier.Name != "NoTokenCopyEffects" {
			return nil, unsupported(modifier, "token copy modifications")
		}
		subject, err := compilePermanentRef(tokenNode.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.Effect {
			if ctx.createdTokenLink != nil {
				return mage.CreateTokenCopiesOfPermanentLinked(ctx.player.build(bindings), count(bindings), subject.build(bindings), ctx.createdTokenLink)
			}
			return mage.CreateTokenCopiesOfPermanent(ctx.player.build(bindings), count(bindings), subject.build(bindings))
		}, nil
	}
	token, err := compileCreatureToken(tokenNode)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		if result != nil {
			return mage.CreateTokenCopiesInto(ctx.player.build(bindings), count(bindings), token(), bindings.resolutionTokens[result])
		}
		if ctx.createdTokenLink != nil {
			return mage.CreateTokenCopiesLinked(ctx.player.build(bindings), count(bindings), token(), ctx.createdTokenLink)
		}
		return mage.CreateTokenCopies(ctx.player.build(bindings), count(bindings), token())
	}, nil
}

func compileCreatureToken(n Node) (func() mage.TokenDefinition, error) {
	name := ""
	switch n.Name {
	case "NamedTokenWithPT":
		if err := arity(n, 7); err != nil {
			return nil, err
		}
		if n.Args[0].Kind != String || strings.TrimSpace(n.Args[0].String) == "" {
			return nil, problem(n.Args[0], "token name must be nonempty")
		}
		name = n.Args[0].String
		n.Args = n.Args[1:]
	case "TokenWithPT":
		if err := arity(n, 6); err != nil {
			return nil, err
		}
	default:
		return nil, unsupported(n, "creature token definition")
	}
	pt := n.Args[0]
	if pt.Name != "PT" {
		return nil, unsupported(pt, "token power and toughness")
	}
	if err := arity(pt, 2); err != nil {
		return nil, err
	}
	if pt.Args[0].Kind != IntegerNode || pt.Args[1].Kind != IntegerNode {
		return nil, problem(pt, "token power and toughness require integer literals")
	}
	var colors []core.Color
	colorNode := n.Args[1]
	if colorNode.Kind != Symbol || colorNode.Name != "Colorless" {
		if colorNode.Name != "Colors" {
			return nil, unsupported(colorNode, "token colors")
		}
		if err := arity(colorNode, 1); err != nil {
			return nil, err
		}
		entries, err := list(colorNode.Args[0])
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			c, err := color(entry)
			if err != nil {
				return nil, err
			}
			if c == core.Colorless || slices.Contains(colors, c) {
				return nil, problem(entry, "token colors require distinct colored values")
			}
			colors = append(colors, c)
		}
	}
	supers, err := list(n.Args[2])
	if err != nil {
		return nil, err
	}
	legendary := false
	for _, super := range supers {
		if super.Kind != Symbol || super.Name != "Legendary" {
			return nil, unsupported(super, "token supertype")
		}
		if legendary {
			return nil, problem(super, "token supertypes must be distinct")
		}
		legendary = true
	}
	entries, err := list(n.Args[3])
	if err != nil {
		return nil, err
	}
	var types []core.CardType
	for _, entry := range entries {
		kind, err := cardType(entry)
		if err != nil {
			return nil, err
		}
		if kind == core.TypeInstant || kind == core.TypeSorcery || slices.Contains(types, kind) {
			return nil, problem(entry, "token types require distinct permanent types")
		}
		types = append(types, kind)
	}
	if !slices.Contains(types, core.TypeCreature) {
		return nil, problem(n.Args[3], "power and toughness require a creature token")
	}
	subNode := n.Args[4]
	if subNode.Name != "SubtypesList" {
		return nil, unsupported(subNode, "token subtypes")
	}
	if err := arity(subNode, 1); err != nil {
		return nil, err
	}
	entries, err = list(subNode.Args[0])
	if err != nil {
		return nil, err
	}
	var subtypes []string
	for _, entry := range entries {
		sub, err := symbol(entry)
		if err != nil {
			return nil, err
		}
		sub = subtypeName(sub)
		if err := mage.ValidateSubtypeFamily(mage.SubtypeCreature, sub); err != nil {
			return nil, problem(entry, err.Error())
		}
		if slices.Contains(subtypes, sub) {
			return nil, problem(entry, "token subtypes must be distinct")
		}
		subtypes = append(subtypes, sub)
	}
	if name == "" {
		if len(subtypes) == 0 {
			return nil, problem(subNode, "unnamed token requires subtypes")
		}
		name = strings.Join(subtypes, " ") + " Token"
	}
	entries, err = list(n.Args[5])
	if err != nil {
		return nil, err
	}
	var attrs []core.Attr
	var abilities []func() mage.Ability
	for _, entry := range entries {
		if entry.Name == "PermanentRuleEffect" {
			ability, err := compileTokenEnchantmentRule(entry)
			if err != nil {
				return nil, err
			}
			abilities = append(abilities, ability)
			continue
		}
		if entry.Name == "BandsWithOthers" {
			ability, err := compileTokenBandsAbility(entry)
			if err != nil {
				return nil, err
			}
			abilities = append(abilities, ability)
			continue
		}
		if entry.Name == "TriggerA" {
			if err := arity(entry, 2); err != nil {
				return nil, err
			}
			if entry.Args[0].Name != "WhenAPermanentDealsDamageToAPlayer" {
				return nil, unsupported(entry.Args[0], "token trigger")
			}
			ability, err := compileDamagePlayerAbility(entry.Args[0], entry.Args[1], scope{permanent: true})
			if err != nil {
				return nil, err
			}
			abilities = append(abilities, ability)
			continue
		}
		attr, ok := keywords[entry.Name]
		if entry.Kind != Symbol || !ok {
			return nil, unsupported(entry, "token keyword")
		}
		if slices.Contains(attrs, attr) {
			return nil, problem(entry, "token keywords must be distinct")
		}
		attrs = append(attrs, attr)
	}
	options := []mage.TokenOption{mage.TokenPT(pt.Args[0].Integer, pt.Args[1].Integer), mage.TokenTypes(types...)}
	if legendary {
		options = append(options, mage.TokenLegendary())
	}
	if len(colors) > 0 {
		options = append(options, mage.TokenColors(colors...))
	}
	if len(subtypes) > 0 {
		options = append(options, mage.TokenSubtypes(subtypes...))
	}
	if len(attrs) > 0 {
		options = append(options, mage.TokenKeywords(attrs...))
	}
	factory, err := mage.NewTokenDefinitionFactory(func() mage.TokenDefinition {
		instanceOptions := slices.Clone(options)
		for _, ability := range abilities {
			instanceOptions = append(instanceOptions, mage.TokenAbilities(ability()))
		}
		return mage.DefineToken(name, instanceOptions...)
	})
	if err != nil {
		return nil, problem(n, err.Error())
	}
	return factory, nil
}
