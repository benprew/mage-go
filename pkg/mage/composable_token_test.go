package mage

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestComposableTokenCreationAndIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	types := []CardType{TypeCreature}
	subtypes := []string{"Spirit"}
	colors := []Color{White}
	keywords := []Keyword{Flying}
	ability := Activate(Costs(), GainLifeFor(You, Fixed(1)))
	def := DefineToken("Spirit", TokenTypes(types...), TokenSubtypes(subtypes...), TokenColors(colors...), TokenPT(2, 3), TokenKeywords(keywords...), TokenAbilities(ability))
	types[0], subtypes[0], colors[0], keywords[0] = TypeArtifact, "Goblin", Red, Trample
	effect := CreateTokenCopies(ActivePlayer, Fixed(2), def)
	if err := ValidateCardDefinition("Maker", NewInstant("Maker", "", WithAbility(Spell(effect)))); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, effect, a.PlayerID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	tokens := g.FilterBattlefield(IsToken)
	if len(tokens) != 2 {
		t.Fatalf("got %d tokens", len(tokens))
	}
	for _, token := range tokens {
		if token.ControllerID() != a.PlayerID() || token.Card.Owner() != a.PlayerID() || !token.HasType(TypeCreature) || !token.HasSubType("Spirit") || !slices.Contains(token.Colors(), White) || token.CurrentPower(g) != 2 || token.CurrentToughness(g) != 3 || !token.HasKeyword(Flying) {
			t.Fatalf("wrong token characteristics: %+v", token)
		}
	}
	if tokens[0].RuntimeAbilities[0] == tokens[1].RuntimeAbilities[0] || tokens[0].RuntimeAbilities[0].AbilityID() == tokens[1].RuntimeAbilities[0].AbilityID() {
		t.Fatal("token abilities share identity")
	}
	clone := g.Clone()
	if len(clone.FilterBattlefield(IsToken)) != 2 || len(g.FilterBattlefield(IsToken)) != 2 {
		t.Fatal("clone changed tokens")
	}
	if err := ApplyEffect(g, CreateTokenCopies(You, Fixed(0), def), a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, CreateTokenCopies(You, Fixed(-1), def), a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.FilterBattlefield(IsToken)) != 2 {
		t.Fatal("nonpositive count created tokens")
	}
}

func TestComposableTokenDefinitionValidation(t *testing.T) {
	outer := DeclarePermanentTarget("outer", IsCreature)
	cases := map[string]TokenDefinition{
		"name":          DefineToken("", TokenTypes(TypeCreature), TokenPT(1, 1)),
		"types":         DefineToken("Bad"),
		"pt":            DefineToken("Bad", TokenTypes(TypeCreature)),
		"color":         DefineToken("Bad", TokenTypes(TypeArtifact), TokenColors(AnyColor)),
		"option":        DefineToken("Bad", TokenTypes(TypeArtifact), nil),
		"duplicate":     DefineToken("Bad", TokenTypes(TypeArtifact), TokenTypes(TypeCreature)),
		"nil ability":   DefineToken("Bad", TokenTypes(TypeArtifact), TokenAbilities(nil)),
		"outer binding": DefineToken("Bad", TokenTypes(TypeCreature), TokenPT(1, 1), TokenAbilities(Activate(Costs(), Destroy(outer)))),
	}
	for name, def := range cases {
		t.Run(name, func(t *testing.T) {
			card := NewInstant(name, "", WithAbility(Spell(CreateTokenCopies(You, Fixed(1), def), WithTargets(outer))))
			if err := ValidateCardDefinition(name, card); err == nil {
				t.Fatal("accepted invalid token definition")
			}
		})
	}
	for name, effect := range map[string]ResolvingEffect{
		"player": CreateTokenCopies(nil, Fixed(1), DefineToken("Stone", TokenTypes(TypeArtifact))),
		"count":  CreateTokenCopies(You, nil, DefineToken("Stone", TokenTypes(TypeArtifact))),
		"token":  CreateTokenCopies(You, Fixed(1), nil),
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateCardDefinition(name, NewInstant(name, "", WithAbility(Spell(effect)))); err == nil || !strings.Contains(err.Error(), "non-nil") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestComposableTokenDynamicCountAndEntryObservation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 3, 1)
	g.PutOnBattlefield(source, a.PlayerID())
	seen := 0
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(NewTriggered(EvtZoneChange, false).SetCondition(func(evt *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		if evt.ToZone == ZoneBattlefield {
			if p := reader.FindPermanent(evt.SourceID); p != nil && p.IsToken && len(reader.FilterBattlefield(IsToken)) == 3 {
				seen++
			}
		}
		return false
	})))
	g.PutOnBattlefield(watcher, a.PlayerID())
	def := DefineToken("Bird", TokenTypes(TypeCreature), TokenSubtypes("Bird"), TokenColors(Blue), TokenPT(1, 1), TokenKeywords(Flying))
	if err := ApplyEffect(g, CreateTokenCopies(You, PowerOf(This), def), source.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.FilterBattlefield(IsToken)) != 3 || seen == 0 {
		t.Fatalf("tokens = %d, observations = %d", len(g.FilterBattlefield(IsToken)), seen)
	}
}
