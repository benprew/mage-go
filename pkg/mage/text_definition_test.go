package mage

import (
	"errors"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextDefinitionInvalidWords(t *testing.T) {
	for _, pair := range [][2]core.Color{{core.Colorless, core.White}, {core.White, core.AnyColor}, {core.Red, core.Red}, {-1, core.Blue}} {
		if _, err := ReplaceColorWord(pair[0], pair[1]); err == nil {
			t.Fatal("accepted invalid color pair", pair)
		}
	}
	for _, pair := range [][2]core.BasicLandType{{0, core.IslandLandType}, {core.PlainsLandType, 6}, {core.ForestLandType, core.ForestLandType}} {
		if _, err := ReplaceBasicLandWord(pair[0], pair[1]); err == nil {
			t.Fatal("accepted invalid land pair", pair)
		}
	}
	if _, err := BuildTextChangedDefinition(nil); err == nil {
		t.Fatal("accepted nil definition")
	}
	if _, err := BuildTextChangedDefinition(NewArtifact("legacy", "")); err == nil {
		t.Fatal("accepted absent provenance")
	}
}

func TestTextDefinitionBuilderContracts(t *testing.T) {
	sentinel := errors.New("builder failed")
	cases := []struct {
		name    string
		builder func([]TextChange) (CardFactory, error)
	}{
		{"error", func([]TextChange) (CardFactory, error) { return nil, sentinel }},
		{"nil", func([]TextChange) (CardFactory, error) { return nil, nil }},
		{"name", func([]TextChange) (CardFactory, error) { return func() Card { return NewArtifact("renamed", "") }, nil }},
		{"mana", func([]TextChange) (CardFactory, error) {
			return func() Card { return NewArtifact("original", "{W}") }, nil
		}},
		{"indicator", func([]TextChange) (CardFactory, error) {
			return func() Card { c := NewArtifact("original", ""); c.SetColorOverride([]core.Color{core.Red}); return c }, nil
		}},
		{"shared", func([]TextChange) (CardFactory, error) {
			c := NewArtifact("original", "")
			return func() Card { return c }, nil
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := NewCardDefinitionPlan(func() Card { return NewArtifact("original", "", WithTextDefinitionBuilder(tc.builder)) })
			if err != nil {
				t.Fatal(err)
			}
			card, err := plan.Build()
			if err != nil {
				t.Fatal(err)
			}
			if _, err = BuildTextChangedDefinition(card); err == nil {
				t.Fatal("accepted invalid builder")
			}
			if tc.name == "error" && !errors.Is(err, sentinel) {
				t.Fatal("builder error lost", err)
			}
			if _, err := BuildTextChangedDefinition(card, TextChange{}); err == nil {
				t.Fatal("accepted zero substitution")
			}
			card.(*BaseCard).CloneFrom(NewArtifact("new rules", ""))
			if _, err := BuildTextChangedDefinition(card); err == nil {
				t.Fatal("legacy replacement retained text provenance")
			}
		})
	}
}

type textSubstitutionTestEffect struct{ ref TextChangeRef }

func (*textSubstitutionTestEffect) Text() string                     { return "read text substitution" }
func (*textSubstitutionTestEffect) Properties() EffectProperties     { return EffectProperties{} }
func (e *textSubstitutionTestEffect) Apply(ctx *EffectContext) error { e.ref.Resolve(ctx); return nil }
func (e *textSubstitutionTestEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "text-substitution-test", children: []definitionEdge{{"substitution", e.ref}}}
}

func TestTextDefinitionTypedReads(t *testing.T) {
	pair := BindColorWordPair("color pair")
	read := ColorWordSubstitution(FirstColorWord(pair), SecondColorWord(pair))
	use := &textSubstitutionTestEffect{ref: read}
	unbound := NewInstant("unbound", "", WithAbility(Spell(use)))
	if err := ValidateCardDefinition(unbound.Name(), unbound); err == nil {
		t.Fatal("unbound substitution accepted")
	}
	valid := NewInstant("bound", "", WithAbility(Spell(Sequence(ChooseColorWordPair(You, pair), use))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	nilRead := NewInstant("nil", "", WithAbility(Spell(&textSubstitutionTestEffect{ref: ColorWordSubstitution(nil, nil)})))
	if err := ValidateCardDefinition(nilRead.Name(), nilRead); err == nil {
		t.Fatal("nil substitution accepted")
	}
	a := &textWordTester{Player: NewBasePlayer("A"), answers: []string{"Red", "Blue"}}
	g := NewGame(a, NewBasePlayer("B"))
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if _, ok := read.Resolve(ctx); ok {
		t.Fatal("missing result supplied words")
	}
	if err := ChooseColorWordPair(You, pair).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	change, ok := read.Resolve(ctx)
	if !ok || change.ApplyColor(core.Red) != core.Blue {
		t.Fatal("typed result not used")
	}
	clone := g.Clone()
	clone.resolution.Clear()
	if _, ok := read.Resolve(&EffectContext{Game: clone}); ok {
		t.Fatal("clone retained cleared words")
	}
	if _, ok := read.Resolve(ctx); !ok {
		t.Fatal("clone changed original result")
	}
	if _, ok := ColorWordSubstitution(FirstColorWord(pair), FirstColorWord(pair)).Resolve(ctx); ok {
		t.Fatal("same resolved words accepted")
	}
	if _, ok := BasicLandWordSubstitution(nil, nil).Resolve(ctx); ok {
		t.Fatal("nil land operands resolved")
	}
}
