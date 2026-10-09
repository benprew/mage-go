package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextAttachedLandTypeDeclaration(t *testing.T) {
	for _, tc := range []struct {
		name   string
		effect ContinuousEffect
		valid  bool
	}{
		{"fixed", BecomesBasicLandAttachedEffect("Swamp"), true},
		{"chosen", BecomesChosenBasicLandTypeAttachedEffect(), true},
		{"multiple", BecomesBasicLandAttachedEffect("Swamp", "Island"), false},
		{"typed-nil", (*basicLandAttachedEffect)(nil), false},
		{"no-type", newBasicLandAttachedEffect(nil, false), false},
		{"nonbasic", newBasicLandAttachedEffect([]string{"Desert"}, false), false},
		{"chosen-with-type", newBasicLandAttachedEffect([]string{"Swamp"}, true), false},
		{"callback", AttachedEffect(core.LayerType, func(*Game, *Permanent, *Permanent) error { return nil }), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewAura("Declaration", "", WithAbility(StaticAbility(tc.effect).WithLandTypeTextRevision()))
			if err := ValidateCardDefinition(card.Name(), card); (err == nil) != tc.valid {
				t.Fatalf("valid=%v: %v", tc.valid, err)
			}
		})
	}
}

func TestTextAttachedLandTypeContext(t *testing.T) {
	effects := []func() ContinuousEffect{
		func() ContinuousEffect { return BecomesBasicLandsEffect(IsLand, "Swamp") },
		func() ContinuousEffect { return BecomesBasicLandAttachedEffect("Swamp") },
		BecomesChosenBasicLandTypeAttachedEffect,
	}
	for i := range effects {
		for j := range effects {
			if i == j {
				continue
			}
			t.Run(string(rune('A'+i))+string(rune('A'+j)), func(t *testing.T) {
				build := func(index int) CardFactory {
					return func() Card {
						return NewAura("Declaration", "", WithCastTarget(TargetPermanent(IsLand)), WithDefinitionAbilityGroup(0, WithAbility(StaticAbility(effects[index]()).WithLandTypeTextRevision())))
					}
				}
				plan, err := NewCardDefinitionPlan(build(i))
				if err != nil {
					t.Fatal(err)
				}
				card, err := plan.Build()
				if err != nil {
					t.Fatal(err)
				}
				WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build(j), nil })(card.(*BaseCard))
				a := NewBasePlayer("A")
				g := NewGame(a, NewBasePlayer("B"))
				host := g.PutOnBattlefield(NewLand("Host", WithSubTypes("Forest")), a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				g.Attach(p.ID(), host.ID())
				id := p.RuntimeAbilities[0].AbilityID()
				before := slices.Clone(g.effects.effects)
				change, _ := ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
				if err := g.RevisePrintedStaticLandType(p, id, change); err == nil {
					t.Fatal("changed land context accepted")
				}
				if !slices.Equal(before, g.effects.effects) {
					t.Fatal("rejection changed registrations")
				}
			})
		}
	}
}

func TestTextAttachedLandTypeRegistrationOrder(t *testing.T) {
	build := func(subtype string) CardFactory {
		return func() Card {
			return NewAura("Source", "", WithCastTarget(TargetPermanent(IsLand)), WithDefinitionAbilityGroup(0, WithAbility(StaticAbility(BecomesBasicLandAttachedEffect(subtype)).WithLandTypeTextRevision())))
		}
	}
	plan, err := NewCardDefinitionPlan(build("Swamp"))
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
	g.effects.Add(before)
	host := g.PutOnBattlefield(NewLand("Host", WithSubTypes("Mountain")), a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	g.Attach(source.ID(), host.ID())
	after := FuncContinuousEffect(core.LayerPT, core.Indefinite, func(*Game, uuid.UUID) error { return nil })
	g.effects.Add(after)
	registrations := slices.Clone(g.effects.effects)
	id := source.RuntimeAbilities[0].AbilityID()
	change, _ := ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	if err := g.RevisePrintedStaticLandType(source, id, change); err != nil {
		t.Fatal(err)
	}
	if len(g.effects.effects) != 3 || g.effects.effects[0] != registrations[0] || g.effects.effects[2] != registrations[2] {
		t.Fatal("revision moved another registration")
	}
	if g.effects.effects[1].SourceID() != source.ID() || source.RuntimeAbilities[0].AbilityID() != id {
		t.Fatal("revision lost source or declaration")
	}
	g.ApplyContinuousEffects()
	if !g.FindPermanent(host.ID()).HasSubType("Island") {
		t.Fatal("revision did not apply the attached rule")
	}
}
