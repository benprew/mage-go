package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextLandAuraSpellDeclaration(t *testing.T) {
	valid := func(extra ...CardOption) Card {
		options := make([]CardOption, 0, 3+len(extra))
		options = append(options, WithCastTarget(TargetPermanent(IsLand)), WithAbility(StaticAbility(BecomesBasicLandAttachedEffect("Swamp"))), WithLandTypeAuraSpellTextContinuation())
		return NewAura("Land Aura", "", append(options, extra...)...)
	}
	chosen := NewAura("Chosen Aura", "", WithCastTarget(TargetPermanent(IsLand)), WithAbility(ChooseBasicLandTypeAsEnters()), WithAbility(StaticAbility(BecomesChosenBasicLandTypeAttachedEffect())), WithLandTypeAuraSpellTextContinuation())
	for _, card := range []Card{valid(), chosen} {
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	for _, card := range []Card{
		NewArtifact("Wrong type", "", WithLandTypeAuraSpellTextContinuation()),
		NewAura("No target", "", WithAbility(StaticAbility(BecomesBasicLandAttachedEffect("Swamp"))), WithLandTypeAuraSpellTextContinuation()),
		valid(WithLandTypeAuraSpellTextContinuation()),
		valid(WithAbility(ChooseBasicLandTypeAsEnters())),
		valid(WithAbility((*StaticAbilityHolder)(nil))),
		NewAura("Nil target", "", WithCastTarget((*PermanentTarget)(nil)), WithAbility(StaticAbility(BecomesBasicLandAttachedEffect("Swamp"))), WithLandTypeAuraSpellTextContinuation()),
		valid(WithAdditionalCost(LifePayCost(1))),
		valid(WithAbility(NewActivated(TapCost(This), GainLifeFor(You, Fixed(1))))),
		NewAura("No choice", "", WithCastTarget(TargetPermanent(IsLand)), WithAbility(StaticAbility(BecomesChosenBasicLandTypeAttachedEffect())), WithLandTypeAuraSpellTextContinuation()),
		NewAura("Extra types", "", WithCardType(core.TypeArtifact), WithCastTarget(TargetPermanent(IsLand)), WithAbility(StaticAbility(BecomesBasicLandAttachedEffect("Swamp"))), WithLandTypeAuraSpellTextContinuation()),
	} {
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatalf("accepted invalid marker on %s", card.Name())
		}
	}
}

func TestTextLandAuraSpellRejectsAtomic(t *testing.T) {
	for _, scenario := range []string{"empty", "invalid", "changed-context", "changed-target", "saved-result"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(changed bool) CardFactory {
				return func() Card {
					effect := BecomesBasicLandAttachedEffect("Swamp")
					options := []CardOption{WithLandTypeAuraSpellTextContinuation()}
					if changed && scenario == "changed-context" {
						effect = BecomesChosenBasicLandTypeAttachedEffect()
						options = append(options, WithDefinitionAbilityGroup(1, WithAbility(ChooseBasicLandTypeAsEnters())))
					}
					target := TargetPermanent(IsLand)
					if changed && scenario == "changed-target" {
						target = TargetPermanentYouControl(IsLand)
					}
					options = append(options, WithCastTarget(target), WithDefinitionAbilityGroup(0, WithAbility(StaticAbility(effect))))
					return NewAura("Aura", "", options...)
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
			g, a, b := newSpellCopyGame()
			g.SetStep(core.PrecombatMain)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			host := g.PutOnBattlefield(NewLand("Host"), b.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			before := g.StackPeek()
			if scenario == "saved-result" {
				before.capturedNumbers = map[*NumberBinding]int{BindNumber("foreign"): 1}
			}
			change, _ := ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
			changes := []TextChange{change}
			if scenario == "invalid" {
				changes = []TextChange{{}}
			}
			if scenario == "empty" {
				changes = nil
			}
			if err := g.RevisePrintedSpellText(before.ID, changes...); err == nil || g.StackPeek() != before || before.TargetSpecs[0] != card.CastTargets()[0] {
				t.Fatal("invalid continuation changed the live stack object")
			}
			g.ResolveStack()
			if g.FindPermanent(card.ID()) == nil || !g.FindPermanent(host.ID()).HasSubType("Swamp") {
				t.Fatal("rejection damaged the original rules")
			}
		})
	}
}
