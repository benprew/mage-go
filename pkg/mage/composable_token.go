package mage

import (
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// TokenDefinition is an immutable token declaration.
type TokenDefinition interface {
	tokenDefinition()
	definitionInspector
}

// TokenOption sets one token characteristic.
type TokenOption interface {
	tokenOption()
	applyToken(*tokenDefinition)
}

type tokenDefinition struct {
	name             string
	types            []core.CardType
	subtypes         []string
	colors           []core.Color
	keywords         []core.Attr
	abilities        []Ability
	power, toughness int
	hasPT            bool
	problem          string
}

func (*tokenDefinition) tokenDefinition() {}

type tokenTypes []core.CardType

func (tokenTypes) tokenOption()                    {}
func (o tokenTypes) applyToken(d *tokenDefinition) { d.types = slices.Clone(o) }

// TokenTypes sets the token's card types.
func TokenTypes(types ...core.CardType) TokenOption { return tokenTypes(slices.Clone(types)) }

type tokenSubtypes []string

func (tokenSubtypes) tokenOption()                    {}
func (o tokenSubtypes) applyToken(d *tokenDefinition) { d.subtypes = slices.Clone(o) }

// TokenSubtypes sets the token's subtypes.
func TokenSubtypes(subtypes ...string) TokenOption { return tokenSubtypes(slices.Clone(subtypes)) }

type tokenColors []core.Color

func (tokenColors) tokenOption()                    {}
func (o tokenColors) applyToken(d *tokenDefinition) { d.colors = slices.Clone(o) }

// TokenColors sets the token's colors. An empty list makes it colorless.
func TokenColors(colors ...core.Color) TokenOption { return tokenColors(slices.Clone(colors)) }

type tokenKeywords []core.Attr

func (tokenKeywords) tokenOption()                    {}
func (o tokenKeywords) applyToken(d *tokenDefinition) { d.keywords = slices.Clone(o) }

// TokenKeywords sets the token's keywords.
func TokenKeywords(keywords ...core.Attr) TokenOption {
	return tokenKeywords(slices.Clone(keywords))
}

type tokenPT struct{ power, toughness int }

func (tokenPT) tokenOption() {}
func (o tokenPT) applyToken(d *tokenDefinition) {
	d.power, d.toughness, d.hasPT = o.power, o.toughness, true
}

// TokenPT sets the token's printed power and toughness.
func TokenPT(power, toughness int) TokenOption { return tokenPT{power, toughness} }

type tokenAbilities []Ability

func (tokenAbilities) tokenOption() {}
func (o tokenAbilities) applyToken(d *tokenDefinition) {
	for _, ability := range o {
		if nilDefinition(ability) {
			d.abilities = append(d.abilities, nil)
		} else {
			d.abilities = append(d.abilities, cloneTokenAbility(ability))
		}
	}
}

// TokenAbilities adds abilities to each created token.
func TokenAbilities(abilities ...Ability) TokenOption { return tokenAbilities(slices.Clone(abilities)) }

// DefineToken captures the token's characteristics and abilities.
func DefineToken(name string, options ...TokenOption) TokenDefinition {
	d := &tokenDefinition{name: name}
	seen := make(map[string]bool)
	for _, option := range options {
		if nilDefinition(option) {
			d.problem = "supply a non-nil token option"
			continue
		}
		key := fmt.Sprintf("%T", option)
		if seen[key] && key != "mage.tokenAbilities" {
			d.problem = "set each token characteristic once"
		}
		seen[key] = true
		option.applyToken(d)
	}
	return d
}

func (d *tokenDefinition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "token-definition", Text: d.name, problem: d.problem}
	if d.name == "" {
		n.problem = "supply a token name"
	}
	if len(d.types) == 0 {
		n.problem = "supply token types"
	}
	if slices.Contains(d.types, core.TypeCreature) != d.hasPT {
		n.problem = "set power and toughness exactly for creature tokens"
	}
	for _, color := range d.colors {
		if color <= core.Colorless || color >= core.AnyColor {
			n.problem = "supply valid token colors"
		}
	}
	for _, keyword := range d.keywords {
		if !core.IsKeywordAttr(keyword) {
			n.problem = "supply valid token keywords"
		}
	}
	for i, ability := range d.abilities {
		n.children = append(n.children, definitionEdge{fmt.Sprintf("abilities[%d]", i), tokenAbilityDefinition{ability}})
	}
	return n
}

type tokenAbilityDefinition struct{ ability Ability }

func (d tokenAbilityDefinition) inspectDefinition() definitionNode {
	return definitionNode{Role: "token-ability", bindingScope: true, children: []definitionEdge{{"definition", d.ability}}}
}

type createTokenCopiesEffect struct {
	player PlayerRef
	count  ResolutionValue
	token  TokenDefinition
}

// CreateTokenCopies creates the evaluated number of tokens for one player.
func CreateTokenCopies(player PlayerRef, count ResolutionValue, token TokenDefinition) ResolvingEffect {
	return &createTokenCopiesEffect{player, count, token}
}
func (e *createTokenCopiesEffect) Text() string { return "create token copies" }
func (e *createTokenCopiesEffect) Properties() EffectProperties {
	p := EffectProperties{Outcome: OutcomeBenefit}
	if d, ok := e.token.(*tokenDefinition); ok {
		p.TokenPower, p.TokenToughness = d.power, d.toughness
	}
	return p
}
func (e *createTokenCopiesEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"player", e.player}, {"count", e.count}, {"token", e.token}}}
}
func (e *createTokenCopiesEffect) Apply(ctx *EffectContext) error {
	player, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	count := e.count.resolveValue(ctx)
	if count <= 0 {
		return nil
	}
	d := e.token.(*tokenDefinition)
	return createTokenBatch(ctx.Game, player, count, d.name, d.power, d.toughness, d.types, d.subtypes, d.colors, d.keywords, d.abilities)
}
