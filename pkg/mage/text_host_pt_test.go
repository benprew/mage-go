package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextHostPTDeclaration(t *testing.T) {
	for _, land := range []string{"Plains", "Island", "Swamp", "Mountain", "Forest", "", "Desert"} {
		card := NewAura("Rule", "", WithAbility(StaticAbility(CreaturesPTWhileAttachedToBasicLand(land, "Goblin", 1, 2)).WithPTTextRevision()))
		_, valid := basicLandTypeColor(land)
		if err := ValidateCardDefinition(card.Name(), card); (err == nil) != valid {
			t.Fatalf("land %q: %v", land, err)
		}
	}
	for _, effects := range [][]ContinuousEffect{
		{CreaturesPTWhileAttachedToBasicLand("Mountain", "", 1, 2)},
		{CreaturesPTWhileAttachedToBasicLand("Mountain", "Aura", 1, 2)},
		{CreaturesPTWhileAttachedToBasicLand("Mountain", "Goblin", 1, 2), BoostAllCreaturesIncludingSelf(1, 2, AnyPermanent)},
	} {
		card := NewAura("Rule", "", WithAbility(StaticAbility(effects...).WithPTTextRevision()))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestTextHostPTContext(t *testing.T) {
	builds := []func() ContinuousEffect{
		func() ContinuousEffect { return CreaturesPTWhileAttachedToBasicLand("Mountain", "Goblin", 1, 2) },
		func() ContinuousEffect { return CreaturesPTWhileAttachedToBasicLand("Mountain", "Elf", 1, 2) },
		func() ContinuousEffect { return BoostAllCreaturesIncludingSelf(1, 2, AnyPermanent) },
		func() ContinuousEffect { return SourcePTWhileControllerControlsBasicLand("Mountain", 1, 2) },
		func() ContinuousEffect { return SourcePTWhileOpponentControlsBasicLand("Mountain", 1, 2) },
		func() ContinuousEffect { return SourcePTWhileOpponentControlsNontokenColor(core.Red, 1, 2) },
		func() ContinuousEffect {
			return NamedCreaturesPTWhileOpponentControlsNontokenColor("Rule", core.Red, 1, 2)
		},
	}
	for i, before := range builds {
		for j, after := range builds {
			if i == j || (i > 1 && j > 1) {
				continue
			}
			factory := func(build func() ContinuousEffect) CardFactory {
				return func() Card {
					return NewAura("Rule", "", WithCastTarget(TargetPermanent(IsLand)), WithDefinitionAbilityGroup(0, WithAbility(StaticAbility(build()).WithPTTextRevision())))
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
			change, _ := ReplaceBasicLandWord(core.MountainLandType, core.ForestLandType)
			if err := g.RevisePrintedStaticPTAbility(p, id, change); err == nil {
				t.Fatalf("changed context %d -> %d", i, j)
			}
			if g.effects.effects[0] != registration || p.RuntimeAbilities[0].AbilityID() != id {
				t.Fatal("rejection changed state")
			}
		}
	}
}

func TestTextHostPTRegistrationOrder(t *testing.T) {
	factory := func(land string) CardFactory {
		return func() Card {
			return NewAura("Rule", "", WithCastTarget(TargetPermanent(IsLand)), WithDefinitionAbilityGroup(0, WithAbility(StaticAbility(CreaturesPTWhileAttachedToBasicLand(land, "Goblin", 1, 2)).WithPTTextRevision())))
		}
	}
	plan, err := NewCardDefinitionPlan(factory("Mountain"))
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return factory("Forest"), nil })(card.(*BaseCard))
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	before := FuncContinuousEffect(core.LayerPT, core.Indefinite, func(*Game, uuid.UUID) error { return nil })
	g.AddContinuousEffect(before)
	p := g.PutOnBattlefield(card, a.PlayerID())
	after := FuncContinuousEffect(core.LayerPT, core.Indefinite, func(*Game, uuid.UUID) error { return nil })
	host := g.PutOnBattlefield(NewLand("Forest", WithSuperTypes(core.SuperBasic), WithSubTypes("Forest")), b.PlayerID())
	g.Attach(p.ID(), host.ID())
	g.AddContinuousEffect(after)
	member := g.PutOnBattlefield(NewCreature("Goblin", "", 2, 3, WithSubTypes("Goblin")), b.PlayerID())
	id := p.RuntimeAbilities[0].AbilityID()
	change, _ := ReplaceBasicLandWord(core.MountainLandType, core.ForestLandType)
	if err := g.RevisePrintedStaticPTAbility(p, id, change); err != nil {
		t.Fatal(err)
	}
	if len(g.effects.effects) != 3 || g.effects.effects[0] != before || g.effects.effects[2] != after || g.effects.effects[1].SourceID() != p.ID() {
		t.Fatal("registration order changed")
	}
	member = g.FindPermanent(member.ID())
	p = g.FindPermanent(p.ID())
	if member.CurrentPower(g) != 3 || member.CurrentToughness(g) != 5 || p.RuntimeAbilities[0].AbilityID() != id {
		t.Fatalf("revision: P/T %d/%d, ID retained %v, host %v", member.CurrentPower(g), member.CurrentToughness(g), p.RuntimeAbilities[0].AbilityID() == id, p.AttachedTo)
	}
}
