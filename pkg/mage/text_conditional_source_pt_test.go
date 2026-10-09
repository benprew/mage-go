package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextConditionalSourcePTDeclaration(t *testing.T) {
	for _, land := range []string{"Plains", "Island", "Swamp", "Mountain", "Forest", "", "Desert", "Unknown"} {
		t.Run(land, func(t *testing.T) {
			effect := SourcePTWhileControllerControlsBasicLand(land, 1, 2)
			card := NewCreature("Source", "R", 1, 1, WithAbility(StaticAbility(effect).WithPTTextRevision()))
			_, valid := basicLandTypeColor(land)
			if err := ValidateCardDefinition(card.Name(), card); (err == nil) != valid {
				t.Fatalf("land %q: %v", land, err)
			}
		})
	}
}

func TestTextConditionalSourcePTContext(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		build := func(conditional bool) CardFactory {
			return func() Card {
				effect := BoostAllCreaturesIncludingSelf(1, 2, AnyPermanent)
				if conditional {
					effect = SourcePTWhileControllerControlsBasicLand("Forest", 1, 2)
				}
				return NewCreature("Source", "R", 1, 1, WithDefinitionAbilityGroup(0, WithAbility(StaticAbility(effect).WithPTTextRevision())))
			}
		}
		plan, err := NewCardDefinitionPlan(build(reverse))
		if err != nil {
			t.Fatal(err)
		}
		card, err := plan.Build()
		if err != nil {
			t.Fatal(err)
		}
		WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build(!reverse), nil })(card.(*BaseCard))
		a := NewBasePlayer("A")
		g := NewGame(a, NewBasePlayer("B"))
		g.PutOnBattlefield(NewLand("Forest", WithSubTypes("Forest")), a.PlayerID())
		p := g.PutOnBattlefield(card, a.PlayerID())
		id := p.RuntimeAbilities[0].AbilityID()
		before := g.effects.effects[0]
		change, _ := ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
		if err := g.RevisePrintedStaticPTAbility(p, id, change); err == nil {
			t.Fatal("static context changed")
		}
		if g.effects.effects[0] != before || p.RuntimeAbilities[0].AbilityID() != id || p.CurrentPower(g) != 2 || p.CurrentToughness(g) != 3 {
			t.Fatal("rejection changed state")
		}
	}
	for _, effects := range [][]ContinuousEffect{
		{SourcePTWhileControllerControlsBasicLand("Forest", 1, 1), BoostAllCreaturesIncludingSelf(1, 1, AnyPermanent)},
		{SourcePTWhileControllerControlsBasicLand("Forest", 1, 1), SourcePTWhileControllerControlsBasicLand("Swamp", 1, 1)},
	} {
		card := NewCreature("Mixed", "", 1, 1, WithAbility(StaticAbility(effects...).WithPTTextRevision()))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("mixed conditional revision accepted")
		}
	}
}

func TestTextConditionalSourcePTRegistrationOrder(t *testing.T) {
	build := func(land string) CardFactory {
		return func() Card {
			return NewCreature("Source", "R", 1, 1, WithDefinitionAbilityGroup(0, WithAbility(StaticAbility(SourcePTWhileControllerControlsBasicLand(land, 1, 2)).WithPTTextRevision())))
		}
	}
	plan, err := NewCardDefinitionPlan(build("Forest"))
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build("Island"), nil })(card.(*BaseCard))
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	before := FuncContinuousEffect(core.LayerPT, core.Indefinite, func(*Game, uuid.UUID) error { return nil })
	g.AddContinuousEffect(before)
	p := g.PutOnBattlefield(card, a.PlayerID())
	after := FuncContinuousEffect(core.LayerPT, core.Indefinite, func(*Game, uuid.UUID) error { return nil })
	g.AddContinuousEffect(after)
	g.PutOnBattlefield(NewLand("Island", WithSubTypes("Island")), a.PlayerID())
	id := p.RuntimeAbilities[0].AbilityID()
	change, _ := ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
	if err := g.RevisePrintedStaticPTAbility(p, id, change); err != nil {
		t.Fatal(err)
	}
	if len(g.effects.effects) != 3 || g.effects.effects[0] != before || g.effects.effects[2] != after || g.effects.effects[1].SourceID() != p.ID() {
		t.Fatal("registration order changed")
	}
	if p.CurrentPower(g) != 2 || p.CurrentToughness(g) != 3 || p.RuntimeAbilities[0].AbilityID() != id {
		t.Fatal("revision changed execution or identity")
	}
}
