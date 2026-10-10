package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func sacrificeTextTestFactory(_ []TextChange, alter string) CardFactory {
	return func() Card {
		paid := BindNumber("paid mana value")
		filter := IsCreature
		options := make([]ActionOption, 0, 3)
		options = append(options, WithSpellSacrificeManaValueContinuation(paid))
		cost := SacrificeMatchingIncludingSourceCostWithManaValue(filter, "Sacrifice payment", paid)
		extra := []any{}
		if alter != "" {
			switch alter {
			case "nil":
				options[0] = WithSpellSacrificeManaValueContinuation(nil)
			case "duplicate":
				options = append(options, WithSpellSacrificeManaValueContinuation(paid))
			case "unmarked":
				options = options[:0]
			case "ordinary":
				options[0] = WithSpellTextContinuation()
			case "absent":
				cost = nil
			case "stats":
				cost = SacrificeCreatureIncludingSourceCostWithStats(BindNumber("power"), paid)
			case "count":
				cost.(*sacrificeMatchingCost).count = 2
			case "other-binding":
				cost = SacrificeMatchingIncludingSourceCostWithManaValue(filter, "Sacrifice payment", BindNumber("other"))
			case "source-policy":
				cost = SacrificeMatchingCostWithManaValue(filter, "Sacrifice payment", paid)
			case "two-costs":
				extra = append(extra, WithAdditionalCost(LifePayCost(1)))
			case "alternate":
				extra = append(extra, WithAlternateCost(core.ZoneHand, core.ManaCost{Generic: 1}))
			}
		}
		if cost != nil {
			extra = append(extra, WithAdditionalCost(cost))
		}
		extra = append(extra, WithDefinitionAbilityGroup(0, WithAbility(NewAction(ActionSpell, append(options, WithEffects(GainLifeFor(You, FromNumber(paid))))...))))
		return NewInstant("Paid declaration", "", extra...)
	}
}

func TestTextSacrificeSpellContinuationDeclaration(t *testing.T) {
	for _, scenario := range []string{"nil", "duplicate", "absent", "stats", "count", "other-binding", "two-costs", "alternate"} {
		t.Run(scenario, func(t *testing.T) {
			card := sacrificeTextTestFactory(nil, scenario)()
			if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "spell sacrifice text continuation") {
				t.Fatalf("invalid marker: %v", err)
			}
		})
	}
	card := sacrificeTextTestFactory(nil, "")()
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"activated", "modal", "permanent"} {
		card := sacrificeTextTestFactory(nil, "")().(*BaseCard)
		action := card.abilities[0].(*ActionDefinition)
		switch kind {
		case "activated":
			action.kind = ActionActivated
		case "modal":
			action.modes = []Mode{{Effects: action.effects}, {Effects: action.effects}}
		case "permanent":
			card.types = []core.CardType{core.TypeEnchantment}
		}
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatalf("%s marker accepted", kind)
		}
	}
}

func TestTextSacrificeSpellContinuationContext(t *testing.T) {
	for _, scenario := range []string{"unmarked", "ordinary", "source-policy", "two-costs", "alternate", "missing-number", "wrong-number", "extra-number", "players", "cards", "sets", "permanents", "empty", "invalid"} {
		t.Run(scenario, func(t *testing.T) {
			plan, err := NewCardDefinitionPlan(sacrificeTextTestFactory(nil, ""))
			if err != nil {
				t.Fatal(err)
			}
			card, err := plan.Build()
			if err != nil {
				t.Fatal(err)
			}
			alter := scenario
			if scenario == "missing-number" || scenario == "wrong-number" || scenario == "extra-number" || scenario == "players" || scenario == "cards" || scenario == "sets" || scenario == "permanents" || scenario == "empty" || scenario == "invalid" {
				alter = ""
			}
			WithTextDefinitionBuilder(func(changes []TextChange) (CardFactory, error) { return sacrificeTextTestFactory(changes, alter), nil })(card.(*BaseCard))
			g, a, _ := newSpellCopyGame()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			payment := g.PutOnBattlefield(NewCreature("Payment", "{4}", 1, 1), a.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			obj := g.StackPeek()
			switch scenario {
			case "missing-number":
				obj.capturedNumbers = nil
			case "wrong-number":
				obj.capturedNumbers = map[*NumberBinding]int{BindNumber("wrong"): 4}
			case "extra-number":
				obj.capturedNumbers[BindNumber("extra")] = 7
			case "players":
				obj.capturedPlayers = map[*PlayerBinding]uuid.UUID{BindPlayer("player"): a.PlayerID()}
			case "cards":
				obj.capturedCards = map[*CardBinding]capturedCard{BindCard("card"): {}}
			case "sets":
				obj.capturedCardSets = map[*CardsBinding][]capturedCard{BindCards("cards"): {}}
			case "permanents":
				obj.capturedPermanents = map[*PermanentBinding]capturedPermanent{BindPermanent("permanent"): {}}
			}
			change, _ := ReplaceColorWord(core.Red, core.Green)
			changes := []TextChange{change}
			if scenario == "empty" {
				changes = nil
			}
			if scenario == "invalid" {
				changes = []TextChange{{}}
			}
			if err := g.RevisePrintedSpellText(obj.ID, changes...); err == nil || g.StackPeek() != obj {
				t.Fatal("invalid transfer changed the live stack visit")
			}
			if g.FindPermanent(payment.ID()) != nil || len(a.Graveyard()) != 1 || a.Life() != 20 {
				t.Fatal("rejection changed payment or resolved the body")
			}
		})
	}
}

func TestTextSacrificeSpellContinuationTransfer(t *testing.T) {
	plan, err := NewCardDefinitionPlan(sacrificeTextTestFactory(nil, ""))
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	WithTextDefinitionBuilder(func(changes []TextChange) (CardFactory, error) { return sacrificeTextTestFactory(changes, ""), nil })(card.(*BaseCard))
	g, a, b := newSpellCopyGame()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	g.PutOnBattlefield(NewCreature("Payment", "{4}", 1, 1), a.PlayerID())
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 9); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	old, _ := spellTextAction(original.spellTextDefinition)
	change, _ := ReplaceColorWord(core.Red, core.Green)
	if err := g.RevisePrintedSpellText(original.ID, change); err != nil {
		t.Fatal(err)
	}
	rebuilt := g.StackPeek()
	fresh, _ := spellTextAction(rebuilt.spellTextDefinition)
	if fresh.spellSacrificeManaValue == old.spellSacrificeManaValue || len(rebuilt.capturedNumbers) != 1 || rebuilt.capturedNumbers[fresh.spellSacrificeManaValue] != 4 || original.capturedNumbers[old.spellSacrificeManaValue] != 4 {
		t.Fatal("saved value was not transferred to a fresh declaration")
	}
	cp := g.CopyStackObjectDirect(rebuilt, b.PlayerID(), false)
	clone := g.Clone()
	change, _ = ReplaceColorWord(core.Green, core.Blue)
	if err := g.RevisePrintedSpellText(cp.ID, change); err != nil {
		t.Fatal(err)
	}
	cp = g.GetStack().FindByID(cp.ID)
	copyAction, _ := spellTextAction(cp.spellTextDefinition)
	if copyAction.spellSacrificeManaValue == fresh.spellSacrificeManaValue || cp.capturedNumbers[copyAction.spellSacrificeManaValue] != 4 || g.GetStack().FindByID(original.ID).capturedNumbers[fresh.spellSacrificeManaValue] != 4 {
		t.Fatal("copy revision changed original bindings")
	}
	if cp.XValue != 9 || rebuilt.XValue != 9 || cp.SourceID == rebuilt.SourceID {
		t.Fatal("revision changed announcement or copy identity")
	}
	g.ResolveStack()
	clone.ResolveStack()
	for _, game := range []*Game{g, clone} {
		if game.GetPlayer(a.PlayerID()).Life() != 24 || game.GetPlayer(b.PlayerID()).Life() != 24 || len(game.GetPlayer(a.PlayerID()).Graveyard()) != 2 {
			t.Fatal("transferred values did not resolve independently")
		}
	}
}

func TestTextSacrificeSpellContinuationEarlierCopy(t *testing.T) {
	plan, err := NewCardDefinitionPlan(sacrificeTextTestFactory(nil, ""))
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	WithTextDefinitionBuilder(func(changes []TextChange) (CardFactory, error) { return sacrificeTextTestFactory(changes, ""), nil })(card.(*BaseCard))
	g, a, b := newSpellCopyGame()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	g.PutOnBattlefield(NewCreature("Payment", "{2}", 1, 1), a.PlayerID())
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	cp := g.CopyStackObjectDirect(original, b.PlayerID(), false)
	change, _ := ReplaceColorWord(core.Red, core.Green)
	if err := g.RevisePrintedSpellText(original.ID, change); err != nil {
		t.Fatal(err)
	}
	old, _ := spellTextAction(cp.spellTextDefinition)
	fresh, _ := spellTextAction(g.GetStack().FindByID(original.ID).spellTextDefinition)
	if old.spellSacrificeManaValue == fresh.spellSacrificeManaValue || cp.capturedNumbers[old.spellSacrificeManaValue] != 2 {
		t.Fatal("revision changed an earlier copy")
	}
	if err := g.RevisePrintedSpellText(cp.ID, change); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 22 {
		t.Fatal("earlier copy lost its paid value")
	}
}
