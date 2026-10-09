package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextOpponentPTDeclaration(t *testing.T) {
	for _, land := range []string{"Plains", "Island", "Swamp", "Mountain", "Forest", "", "Desert"} {
		card := NewCreature("Source", "", 1, 1, WithAbility(StaticAbility(SourcePTWhileOpponentControlsBasicLand(land, 1, 2)).WithPTTextRevision()))
		_, valid := basicLandTypeColor(land)
		if err := ValidateCardDefinition(card.Name(), card); (err == nil) != valid {
			t.Fatalf("land %q: %v", land, err)
		}
	}
	for _, color := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green, core.Colorless, core.AnyColor, -1} {
		for _, named := range []bool{false, true} {
			effect := SourcePTWhileOpponentControlsNontokenColor(color, 1, 2)
			if named {
				effect = NamedCreaturesPTWhileOpponentControlsNontokenColor("Member", color, 1, 2)
			}
			card := NewCreature("Source", "", 1, 1, WithAbility(StaticAbility(effect).WithPTTextRevision()))
			if err := ValidateCardDefinition(card.Name(), card); (err == nil) != (color >= core.White && color <= core.Green) {
				t.Fatalf("color %v: %v", color, err)
			}
		}
	}
	for _, effects := range [][]ContinuousEffect{
		{NamedCreaturesPTWhileOpponentControlsNontokenColor("", core.Red, 1, 1)},
		{SourcePTWhileOpponentControlsBasicLand("Island", 1, 1), BoostAllCreaturesIncludingSelf(1, 1, AnyPermanent)},
		{SourcePTWhileOpponentControlsNontokenColor(core.Red, 1, 1), SourcePTWhileOpponentControlsNontokenColor(core.Blue, 1, 1)},
	} {
		card := NewCreature("Source", "", 1, 1, WithAbility(StaticAbility(effects...).WithPTTextRevision()))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid conditional declaration accepted")
		}
	}
}

func TestTextOpponentPTRegistrationOrder(t *testing.T) {
	for _, context := range []ptRuleContext{opponentBasicLandSourcePT, opponentNontokenColorSourcePT, opponentNontokenColorNamedPT} {
		build := func(revised bool) CardFactory {
			return func() Card {
				land, color := "Island", core.Red
				if revised {
					land, color = "Mountain", core.Blue
				}
				effect := SourcePTWhileOpponentControlsBasicLand(land, 1, 2)
				switch context {
				case opponentNontokenColorSourcePT:
					effect = SourcePTWhileOpponentControlsNontokenColor(color, 1, 2)
				case opponentNontokenColorNamedPT:
					effect = NamedCreaturesPTWhileOpponentControlsNontokenColor("Source", color, 1, 2)
				}
				return NewCreature("Source", "", 1, 1, WithDefinitionAbilityGroup(0, WithAbility(StaticAbility(effect).WithPTTextRevision())))
			}
		}
		plan, err := NewCardDefinitionPlan(build(false))
		if err != nil {
			t.Fatal(err)
		}
		card, err := plan.Build()
		if err != nil {
			t.Fatal(err)
		}
		WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build(true), nil })(card.(*BaseCard))
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		before := FuncContinuousEffect(core.LayerPT, core.Indefinite, func(*Game, uuid.UUID) error { return nil })
		g.AddContinuousEffect(before)
		p := g.PutOnBattlefield(card, a.PlayerID())
		after := FuncContinuousEffect(core.LayerPT, core.Indefinite, func(*Game, uuid.UUID) error { return nil })
		g.AddContinuousEffect(after)
		g.PutOnBattlefield(NewLand("Mountain", WithSubTypes("Mountain")), b.PlayerID())
		g.PutOnBattlefield(NewArtifact("Blue", "{U}"), b.PlayerID())
		id := p.RuntimeAbilities[0].AbilityID()
		change, _ := ReplaceColorWord(core.Red, core.Blue)
		if err := g.RevisePrintedStaticPTAbility(p, id, change); err != nil {
			t.Fatal(err)
		}
		if len(g.effects.effects) != 3 || g.effects.effects[0] != before || g.effects.effects[2] != after || g.effects.effects[1].SourceID() != p.ID() {
			t.Fatal("registration order changed")
		}
		if p.CurrentPower(g) != 2 || p.CurrentToughness(g) != 3 || p.RuntimeAbilities[0].AbilityID() != id {
			t.Fatalf("context %d: P/T %d/%d, ID retained %v", context, p.CurrentPower(g), p.CurrentToughness(g), p.RuntimeAbilities[0].AbilityID() == id)
		}
	}
}

func TestTextOpponentPTContext(t *testing.T) {
	builds := []func() ContinuousEffect{
		func() ContinuousEffect { return BoostAllCreaturesIncludingSelf(1, 2, AnyPermanent) },
		func() ContinuousEffect { return SourcePTWhileControllerControlsBasicLand("Island", 1, 2) },
		func() ContinuousEffect { return SourcePTWhileOpponentControlsBasicLand("Island", 1, 2) },
		func() ContinuousEffect { return SourcePTWhileOpponentControlsNontokenColor(core.Red, 1, 2) },
		func() ContinuousEffect {
			return NamedCreaturesPTWhileOpponentControlsNontokenColor("Source", core.Red, 1, 2)
		},
		func() ContinuousEffect {
			return NamedCreaturesPTWhileOpponentControlsNontokenColor("Other", core.Red, 1, 2)
		},
	}
	for i, before := range builds {
		for j, after := range builds {
			if i == j {
				continue
			}
			factory := func(build func() ContinuousEffect) CardFactory {
				return func() Card {
					return NewCreature("Source", "", 1, 1, WithDefinitionAbilityGroup(0, WithAbility(StaticAbility(build()).WithPTTextRevision())))
				}
			}
			plan, err := NewCardDefinitionPlan(factory(before))
			if err != nil {
				t.Fatal(err)
			}
			card, err := plan.Build()
			if err != nil {
				t.Fatal(err)
			}
			WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return factory(after), nil })(card.(*BaseCard))
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(card, a.PlayerID())
			id := p.RuntimeAbilities[0].AbilityID()
			registration := g.effects.effects[0]
			change, _ := ReplaceColorWord(core.Red, core.Blue)
			if err := g.RevisePrintedStaticPTAbility(p, id, change); err == nil {
				t.Fatalf("context changed %d -> %d", i, j)
			}
			if g.effects.effects[0] != registration || p.RuntimeAbilities[0].AbilityID() != id {
				t.Fatal("rejection changed state")
			}
		}
	}
}
