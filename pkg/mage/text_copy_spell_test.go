package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextCopySpellDeclaration(t *testing.T) {
	valid := func(extra ...CardOption) Card {
		opts := make([]CardOption, 0, 2+len(extra))
		opts = append(opts, WithCopyEntrySpellTextContinuation(), WithAbility(MayCopyAsEnters(IsCreature)))
		return NewCreature("Copier", "", 0, 1, append(opts, extra...)...)
	}
	if err := ValidateCardDefinition("Copier", valid()); err != nil {
		t.Fatal(err)
	}
	for _, card := range []Card{
		valid(WithCopyEntrySpellTextContinuation()), valid(WithAdditionalCost(LifePayCost(1))),
		valid(WithCastTarget(TargetPermanent(IsCreature))), valid(WithKeyword(core.Flying)),
		valid(WithAbility(NewActivated(TapCost(This), GainLifeFor(You, Fixed(1))))),
		valid(WithAbility(MayCopyAsEnters(IsCreature))), valid(WithAbility((*entryCopyAbility)(nil))),
		NewCreature("Empty", "", 1, 1, WithCopyEntrySpellTextContinuation()),
		NewInstant("Wrong type", "", WithCopyEntrySpellTextContinuation()),
	} {
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatalf("accepted invalid continuation on %s", card.Name())
		}
	}
}

func TestTextCopySpellDonorContract(t *testing.T) {
	build := func(word core.Attr) CardFactory {
		return func() Card {
			return NewCreature("Donor", "", 2, 2, WithDefinitionAbilityGroup(0, WithLayeredPrintedBasicLandwalk(word)))
		}
	}
	plan, err := NewCardDefinitionPlan(build(core.Swampwalk))
	if err != nil {
		t.Fatal(err)
	}
	physical := entryCopyPhysical(t)
	original := physical.(*BaseCard)
	land, _ := ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	if _, err := BuildEntryCopyWithText(physical, plan.CopiableValues(), land); err == nil {
		t.Fatal("donor without checked text builder accepted")
	}
	donorPlan, err := NewCardDefinitionPlan(func() Card {
		card := build(core.Swampwalk)().(*BaseCard)
		WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build(core.Islandwalk), nil })(card)
		return card
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := BuildEntryCopyWithText(physical, donorPlan.CopiableValues(), land)
	if err != nil {
		t.Fatal(err)
	}
	if result.ID() != original.ID() || result.Owner() != original.Owner() || result.AttrSeeds()[core.Islandwalk] != 1 || cardForBattlefieldDeparture(result).Name() != original.Name() {
		t.Fatal("continued donor lost physical identity or rules")
	}
	g := NewGame(NewBasePlayer("A"), NewBasePlayer("B"))
	p := g.PutOnBattlefield(result, g.players[0].PlayerID())
	values, err := g.CaptureCopiableValues(p)
	if err != nil {
		t.Fatal(err)
	}
	printed, err := values.Build()
	if err != nil || printed.AttrSeeds()[core.Swampwalk] != 1 || printed.AttrSeeds()[core.Islandwalk] != 0 {
		t.Fatal("text changes entered copy values", err)
	}
	for _, changes := range [][]TextChange{{{}}, {land, {}}} {
		if _, err := BuildEntryCopyWithText(physical, donorPlan.CopiableValues(), changes...); err == nil {
			t.Fatal("invalid words accepted")
		}
	}
	if original.Name() != "Physical" || original.AttrSeeds()[core.Islandwalk] != 0 {
		t.Fatal("rejected donor changed physical card")
	}
	p.Card.(*BaseCard).attrSeeds[core.Flying] = 1
	if _, err := g.CaptureCopiableValues(p); err == nil {
		t.Fatal("untracked entry change supplied copy values")
	}
}

func TestTextCopySpellRejectsAtomic(t *testing.T) {
	for _, scenario := range []string{"empty", "invalid", "changed-modification", "changed-keep-color", "changed-subtype", "saved-result", "no-builder", "no-group", "countered"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(changed bool) CardFactory {
				return func() Card {
					rule := MayCopyAsEnters(IsCreature)
					if changed && scenario == "changed-modification" {
						rule = MayCopyAsEntersWithModifications(IsCreature, CopyAddCardTypes(core.TypeEnchantment))
					}
					if changed && scenario == "changed-keep-color" {
						rule = MayCopyAsEntersKeepingColors(IsCreature)
					}
					option := WithAbility(rule)
					if scenario != "no-group" {
						option = WithDefinitionAbilityGroup(0, option)
					}
					opts := []CardOption{WithCopyEntrySpellTextContinuation(), option}
					if changed && scenario == "changed-subtype" {
						opts = append(opts, WithSubTypes("Goblin"))
					}
					return NewCreature("Copier", "", 0, 1, opts...)
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
			land, _ := ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
			changes := []TextChange{land}
			if scenario == "empty" {
				changes = nil
			}
			if scenario == "invalid" {
				changes = []TextChange{{}}
			}
			if scenario == "countered" {
				g.CounterSpellOnStack(card.ID())
			}
			if err := g.RevisePrintedSpellText(before.ID, changes...); err == nil {
				t.Fatal("invalid continuation accepted")
			}
			if scenario == "countered" {
				if g.StackSize() != 0 || len(a.Graveyard()) != 1 || a.Graveyard()[0].Name() != "Copier" {
					t.Fatal("revision changed countered departure")
				}
			} else if g.StackPeek() != before {
				t.Fatal("rejected continuation changed stack state")
			}
		})
	}
}

func TestTextCopySpellRejectsDonorGroups(t *testing.T) {
	for _, scenario := range []string{"missing-group", "changed-group", "changed-mana", "changed-name"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(changed bool) CardFactory {
				return func() Card {
					option := WithLayeredPrintedBasicLandwalk(core.Swampwalk)
					if !changed || scenario != "missing-group" {
						group := 0
						if changed && scenario == "changed-group" {
							group = 1
						}
						option = WithDefinitionAbilityGroup(group, option)
					}
					name, cost := "Donor", "{R}"
					if changed && scenario == "changed-name" {
						name = "Changed"
					}
					if changed && scenario == "changed-mana" {
						cost = "{G}"
					}
					return NewCreature(name, cost, 2, 2, option)
				}
			}
			plan, err := NewCardDefinitionPlan(func() Card {
				card := build(false)().(*BaseCard)
				WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build(true), nil })(card)
				return card
			})
			if err != nil {
				t.Fatal(err)
			}
			original := entryCopyPhysical(t)
			land, _ := ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
			if _, err := BuildEntryCopyWithText(original, plan.CopiableValues(), land); err == nil {
				t.Fatal("invalid donor builder accepted")
			}
			if original.Name() != "Physical" || original.AttrSeeds()[core.Swampwalk] != 0 {
				t.Fatal("rejected donor changed physical rules")
			}
		})
	}
}

func TestTextCopySpellFaceDownExceptions(t *testing.T) {
	values := &CopiableValues{faceDown: true, printed: copiableCharacteristics{types: []core.CardType{core.TypeCreature}, power: 2, toughness: 2}}
	modified, err := values.WithModifications(CopyAddCardTypes(core.TypeEnchantment), CopyRetainColors(core.Blue))
	if err != nil {
		t.Fatal(err)
	}
	modified, err = modified.WithModifications(CopyAddCardTypes(core.TypeArtifact))
	if err != nil {
		t.Fatal(err)
	}
	change, _ := ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	copied, err := BuildEntryCopyWithText(entryCopyPhysical(t), modified, change)
	if err != nil {
		t.Fatal(err)
	}
	if copied.Name() != "" || copied.Power() != 2 || copied.Toughness() != 2 || !copied.HasType(core.TypeEnchantment) || !copied.HasType(core.TypeArtifact) || len(copied.Abilities()) != 0 {
		t.Fatal("modified face-down values acquired text or lost exceptions")
	}
	if _, err := BuildEntryCopyWithText(entryCopyPhysical(t), modified, TextChange{}); err == nil {
		t.Fatal("textless values bypassed word validation")
	}
}

func TestTextCopySpellAddedTypes(t *testing.T) {
	for _, kind := range []string{"intrinsic", "aura"} {
		t.Run(kind, func(t *testing.T) {
			plan, err := NewCardDefinitionPlan(func() Card {
				if kind == "intrinsic" {
					return NewCreature("Intrinsic", "", 2, 2, WithIntrinsicCreatureSpellTextContinuation(), WithLayeredPrintedBasicLandwalk(core.Swampwalk))
				}
				return NewAura("Aura", "", WithLandTypeAuraSpellTextContinuation(), WithCastTarget(TargetPermanent(IsLand)), WithAbility(StaticAbility(BecomesBasicLandAttachedEffect("Swamp"))))
			})
			if err != nil {
				t.Fatal(err)
			}
			values, err := plan.CopiableValues().WithModifications(CopyAddCardTypes(core.TypeArtifact))
			if err != nil {
				t.Fatal(err)
			}
			copied, err := values.Build()
			if err != nil {
				t.Fatal(err)
			}
			base := copied.(*BaseCard)
			if !base.HasType(core.TypeArtifact) || base.intrinsicSpellText != 0 || base.landAuraSpellText != 0 || len(base.abilities) != 1 {
				t.Fatal("copy type exception lost rules or retained an incompatible spell marker")
			}
			original, err := plan.Build()
			if err != nil {
				t.Fatal(err)
			}
			if original.HasType(core.TypeArtifact) || original.(*BaseCard).intrinsicSpellText+original.(*BaseCard).landAuraSpellText != 1 {
				t.Fatal("copy exception changed original declaration")
			}
		})
	}
}
