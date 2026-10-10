package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextIntrinsicSpellDeclaration(t *testing.T) {
	valid := func(extra ...CardOption) Card {
		opts := make([]CardOption, 0, 4+len(extra))
		opts = append(opts, WithIntrinsicCreatureSpellTextContinuation(), WithLayeredPrintedBasicLandwalk(core.Swampwalk), WithAbility(ProtectionFromColors(core.White).WithLayeredColorTextRevision()), WithKeyword(core.FirstStrike))
		return NewCreature("Intrinsic", "", 2, 2, append(opts, extra...)...)
	}
	if err := ValidateCardDefinition("Intrinsic", valid()); err != nil {
		t.Fatal(err)
	}
	for _, card := range []Card{
		valid(WithIntrinsicCreatureSpellTextContinuation()), valid(WithAdditionalCost(LifePayCost(1))),
		valid(WithAbility((*ProtectionAbility)(nil))), valid(WithAbility(NewActivated(TapCost(This), GainLifeFor(You, Fixed(1))))),
		valid(WithCardType(core.TypeArtifact)), valid(WithCastTarget(TargetPermanent(IsCreature))),
		NewArtifact("Wrong type", "", WithIntrinsicCreatureSpellTextContinuation()),
		NewCreature("No declaration", "", 1, 1, WithIntrinsicCreatureSpellTextContinuation()),
		NewCreature("Unmarked protection", "", 1, 1, WithIntrinsicCreatureSpellTextContinuation(), WithAbility(ProtectionFromColors(core.White))),
		NewCreature("Unmarked landwalk", "", 1, 1, WithIntrinsicCreatureSpellTextContinuation(), WithPrintedBasicLandwalk(core.Swampwalk)),
	} {
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatalf("accepted invalid marker on %s", card.Name())
		}
	}
}

func TestTextIntrinsicSpellRejectsAtomic(t *testing.T) {
	for _, scenario := range []string{"empty", "invalid", "changed-kind", "changed-keyword", "changed-subtype", "saved-result", "no-builder", "no-group", "countered"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(changed bool) CardFactory {
				return func() Card {
					option := WithLayeredPrintedBasicLandwalk(core.Swampwalk)
					if changed && scenario == "changed-kind" {
						option = WithAbility(ProtectionFromColors(core.White).WithLayeredColorTextRevision())
					}
					if scenario != "no-group" {
						option = WithDefinitionAbilityGroup(0, option)
					}
					opts := []CardOption{WithIntrinsicCreatureSpellTextContinuation(), option, WithKeyword(core.FirstStrike)}
					if changed && scenario == "changed-keyword" {
						opts = append(opts, WithKeyword(core.Flying))
					}
					if changed && scenario == "changed-subtype" {
						opts = append(opts, WithSubTypes("Goblin"))
					}
					return NewCreature("Intrinsic", "", 2, 2, opts...)
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
			if scenario != "no-builder" {
				WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build(true), nil })(card.(*BaseCard))
			}
			g, a, _ := newSpellCopyGame()
			g.SetStep(core.PrecombatMain)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
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
			if scenario == "countered" {
				g.CounterSpellOnStack(card.ID())
			}
			if err := g.RevisePrintedSpellText(before.ID, changes...); err == nil {
				t.Fatal("invalid continuation accepted")
			}
			if scenario == "countered" {
				if len(a.Graveyard()) != 1 || g.StackSize() != 0 {
					t.Fatal("revision changed departure")
				}
				return
			}
			if g.StackPeek() != before {
				t.Fatal("invalid continuation changed stack state")
			}
			g.ResolveStack()
			if p := g.FindPermanent(card.ID()); p == nil || !p.HasKeyword(core.Swampwalk) || p.HasKeyword(core.Flying) {
				t.Fatal("rejection damaged original rules")
			}
		})
	}
}

func TestTextIntrinsicSpellCopyProvenance(t *testing.T) {
	build := func(word core.Attr) CardFactory {
		return func() Card {
			return NewCreature("Intrinsic", "", 2, 2, WithIntrinsicCreatureSpellTextContinuation(), WithDefinitionAbilityGroup(0, WithLayeredPrintedBasicLandwalk(word)))
		}
	}
	plan, err := NewCardDefinitionPlan(build(core.Swampwalk))
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build(core.Islandwalk), nil })(card.(*BaseCard))
	g, a, _ := newSpellCopyGame()
	g.SetStep(core.PrecombatMain)
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	change, _ := ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	if _, err := g.CaptureCopiableValues(p); err != nil {
		t.Fatal(err)
	}
	p.Card.(*BaseCard).attrSeeds[core.Flying] = 1
	if _, err := g.CaptureCopiableValues(p); err == nil {
		t.Fatal("untracked entry mutation supplied copy values")
	}
}
