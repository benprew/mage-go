package mage

import (
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func copyTextPlan(t *testing.T) *CardDefinitionPlan {
	t.Helper()
	var build func([]TextChange) CardFactory
	build = func(changes []TextChange) CardFactory {
		return func() Card {
			color := core.Red
			for _, change := range changes {
				color = change.ApplyColor(color)
			}
			target := DeclarePermanentTarget("colored permanent", HasColorFilter(color))
			return NewCreature("Red donor", "{R}", 2, 3,
				WithDefinitionAbilityGroup(0, WithAbility(StaticAbility(&testCopyColorDefinition{}))),
				WithDefinitionAbilityGroup(1, WithAbility(NewAction(ActionActivated,
					WithTarget(target.targetSpec()), WithEffect(TapSubject(target)), WithLayeredTextRevisionActivationState()))),
				WithTextDefinitionBuilder(func(changes []TextChange) (CardFactory, error) { return build(changes), nil }))
		}
	}
	plan, err := NewCardDefinitionPlan(build(nil))
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestCopyTextRebindingSnapshot(t *testing.T) {
	plan := copyTextPlan(t)
	values, err := plan.CopiableValues().WithModifications(CopyAddCardTypes(core.TypeEnchantment), CopyRetainColors(core.Blue))
	if err != nil {
		t.Fatal(err)
	}
	values, err = values.WithModifications(CopyAddCardTypes(core.TypeArtifact), CopyRetainColors())
	if err != nil {
		t.Fatal(err)
	}
	card, err := values.Build()
	if err != nil {
		t.Fatal(err)
	}
	first, _ := ReplaceColorWord(core.Red, core.Green)
	second, _ := ReplaceColorWord(core.Green, core.White)
	revision, err := BuildTextDefinitionRevision(card, first)
	if err != nil {
		t.Fatal(err)
	}
	rebuilt, err := BuildTextChangedDefinition(revision.Definition(), second)
	if err != nil {
		t.Fatal(err)
	}
	base := rebuilt.(*BaseCard)
	if !slices.Equal(base.Types(), []core.CardType{core.TypeCreature, core.TypeEnchantment, core.TypeArtifact}) || len(cardColors(base)) != 0 || base.Name() != "Red donor" || base.ManaCost().Red != 1 || len(base.abilities) != 1 || base.copiableValues != values || base.definitionPlan != values.definition {
		t.Fatal("ordered copy exceptions or original copy provenance was lost")
	}
	if _, ok := revision.ReplacementAbility(card.Abilities()[0].AbilityID()); !ok {
		t.Fatal("omitted color declaration shifted the remaining group")
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(rebuilt, a.PlayerID())
	white := g.PutOnBattlefield(NewCreature("White", "{W}", 1, 1), b.PlayerID())
	green := g.PutOnBattlefield(NewCreature("Green", "{G}", 1, 1), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{green.ID()}); err == nil {
		t.Fatal("ordered changes retained an earlier target filter")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{white.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if !g.FindPermanent(white.ID()).Tapped {
		t.Fatal("fresh target binding did not receive the announcement")
	}
	captured, err := g.CaptureCopiableValues(g.FindPermanent(p.ID()))
	if err != nil {
		t.Fatal(err)
	}
	printed, err := captured.Build()
	if err != nil || len(printed.Abilities()) != 1 || !printed.HasType(core.TypeEnchantment) || !printed.HasType(core.TypeArtifact) || len(cardColors(printed)) != 0 {
		t.Fatal("copy capture lost exceptions", err)
	}
	if err := independentCardDefinitions(card, printed); err != nil {
		t.Fatal(err)
	}
	later := g.PutOnBattlefield(printed, a.PlayerID())
	red := g.PutOnBattlefield(NewCreature("Red", "{R}", 1, 1), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), later.ID(), 0, []uuid.UUID{white.ID()}); err == nil {
		t.Fatal("later copy inherited ordinary text changes")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), later.ID(), 0, []uuid.UUID{red.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if !g.FindPermanent(red.ID()).Tapped {
		t.Fatal("later copy lost its independent original target binding")
	}
	original, err := plan.Build()
	if err != nil || original.HasType(core.TypeEnchantment) || len(original.Abilities()) != 2 {
		t.Fatal("modified snapshot changed its source", err)
	}
}

func TestCopyTextRebindingRejectsMaskedDefects(t *testing.T) {
	sentinel := errors.New("text builder failed")
	for _, scenario := range []string{"name", "mana", "type", "color", "nil", "shared", "error", "invalid-word"} {
		t.Run(scenario, func(t *testing.T) {
			plan, err := NewCardDefinitionPlan(func() Card {
				return NewArtifact("Donor", "", WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) {
					if scenario == "error" {
						return nil, sentinel
					}
					if scenario == "nil" {
						return nil, nil
					}
					build := func() Card {
						card := NewArtifact("Donor", "")
						switch scenario {
						case "name":
							card.name = "Changed"
						case "mana":
							card.manaCost = core.ParseManaCost("{R}")
						case "type":
							WithCardType(core.TypeEnchantment)(card)
						case "color":
							card.SetColorOverride([]core.Color{core.Red})
						}
						return card
					}
					if scenario == "shared" {
						card := build()
						return func() Card { return card }, nil
					}
					return build, nil
				}))
			})
			if err != nil {
				t.Fatal(err)
			}
			values, err := plan.CopiableValues().WithModifications(CopyAddCardTypes(core.TypeEnchantment), CopyRetainColors(core.Blue))
			if err != nil {
				t.Fatal(err)
			}
			card, err := values.Build()
			if err != nil {
				t.Fatal(err)
			}
			change, _ := ReplaceColorWord(core.Red, core.Green)
			if scenario == "invalid-word" {
				change = TextChange{}
			}
			if _, err := BuildTextChangedDefinition(card, change); err == nil {
				t.Fatal("copy exception masked an invalid text builder")
			} else if scenario == "error" && !errors.Is(err, sentinel) {
				t.Fatal("builder error was lost", err)
			}
			if len(cardColors(card)) != 1 || cardColors(card)[0] != core.Blue || !card.HasType(core.TypeEnchantment) {
				t.Fatal("rejected builder changed the snapshot")
			}
		})
	}
}
