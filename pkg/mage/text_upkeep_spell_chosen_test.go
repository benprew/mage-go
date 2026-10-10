package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func chosenUpkeepSpellFactory(scenario string, changed bool) CardFactory {
	return func() Card {
		player := BindPlayer("event player")
		event := ChosenPlayerUpkeep(player)
		if changed && scenario == "changed-context" {
			event = TurnStepBegins(YourUpkeep, player)
		}
		trigger := WhenEvent(event, GainLifeFor(FromPlayer(player), Fixed(1))).(*GenericTriggered).WithLayeredUpkeepTextRevision()
		choiceGroup := 0
		if changed && scenario == "changed-choice-group" {
			choiceGroup = 2
		}
		opts := []CardOption{WithUpkeepSpellTextContinuation(), WithUpkeepObjectTextRevision()}
		if !changed || scenario != "lost-choice" && scenario != "changed-context" {
			choice := WithAbility(ChooseOpponentAsEnters())
			if scenario != "ungrouped-choice" {
				choice = WithDefinitionAbilityGroup(choiceGroup, choice)
			}
			opts = append(opts, choice)
		}
		opts = append(opts, WithDefinitionAbilityGroup(1, WithAbility(trigger)))
		if changed && scenario == "added-choice" {
			opts = append(opts, WithDefinitionAbilityGroup(2, WithAbility(ChooseOpponentAsEnters())))
		}
		if changed && scenario == "foreign-entry" {
			opts = append(opts, WithDefinitionAbilityGroup(2, WithAbility(EnterTappedAsEnters())))
		}
		return NewArtifact("Chosen upkeep", "", opts...)
	}
}

func TestTextUpkeepSpellChosenDeclarations(t *testing.T) {
	for _, scenario := range []string{"valid", "lost-choice", "changed-context", "added-choice", "foreign-entry"} {
		t.Run(scenario, func(t *testing.T) {
			card := chosenUpkeepSpellFactory(scenario, scenario != "valid")()
			valid := scenario == "valid" || scenario == "changed-context"
			if err := ValidateCardDefinition(card.Name(), card); (err == nil) != valid {
				t.Fatalf("validation: %v", err)
			}
		})
	}
}

func TestTextUpkeepSpellChosenAtomic(t *testing.T) {
	for _, scenario := range []string{"lost-choice", "added-choice", "foreign-entry", "changed-context", "changed-choice-group", "ungrouped-choice"} {
		t.Run(scenario, func(t *testing.T) {
			plan, err := NewCardDefinitionPlan(chosenUpkeepSpellFactory(scenario, false))
			if err != nil {
				t.Fatal(err)
			}
			card, err := plan.Build()
			if err != nil {
				t.Fatal(err)
			}
			WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return chosenUpkeepSpellFactory(scenario, true), nil })(card.(*BaseCard))
			g, a, _ := newSpellCopyGame()
			g.SetStep(core.PrecombatMain)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			before := g.StackPeek()
			word, _ := ReplaceColorWord(core.Red, core.Blue)
			if err := g.RevisePrintedSpellText(before.ID, word); err == nil {
				t.Fatal("invalid choice revision accepted")
			}
			if g.StackPeek() != before {
				t.Fatal("rejection changed the stack visit")
			}
		})
	}
}

func TestTextUpkeepSpellChosenEntry(t *testing.T) {
	factory := chosenUpkeepSpellFactory("valid", false)
	plan, err := NewCardDefinitionPlan(factory)
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return factory, nil })(card.(*BaseCard))
	g, a, b := newSpellCopyGame()
	g.SetStep(core.PrecombatMain)
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	word, _ := ReplaceColorWord(core.Red, core.Blue)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, word); err != nil {
		t.Fatal(err)
	}
	spellRule := g.StackPeek().spellTextDefinition.(*BaseCard).abilities[1].(*GenericTriggered)
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	g.ResolveStack()
	clone.ResolveStack()
	for _, game := range []*Game{g, clone} {
		p, token := game.FindPermanent(card.ID()), game.FindPermanent(copied.SourceID)
		if p.ChosenPlayer != b.PlayerID() || token.ChosenPlayer != a.PlayerID() || !token.Card.IsToken() {
			t.Fatal("entry choice or copy identity lost")
		}
		rule := p.RuntimeAbilities[1].(*GenericTriggered)
		copyRule := token.RuntimeAbilities[1].(*GenericTriggered)
		if upkeepObjectPlayer(rule) == upkeepObjectPlayer(spellRule) || upkeepObjectPlayer(rule) == upkeepObjectPlayer(copyRule) {
			t.Fatal("entry shared event bindings")
		}
		game.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
		game.PutTriggersOnStack()
		if game.StackSize() != 1 || game.GetPlayer(b.PlayerID()).Life() != 20 {
			t.Fatal("choice detection or timing lost")
		}
		game.ResolveStack()
		if game.GetPlayer(b.PlayerID()).Life() != 21 || game.GetPlayer(a.PlayerID()).Life() != 20 {
			t.Fatal("choice capture failed")
		}
	}
}
