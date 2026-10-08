package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCopyTypeModificationSnapshots(t *testing.T) {
	plan := entryCopyPlan(t, "Donor", func() []CardOption { return []CardOption{WithAbility(EntersWithNCounters(core.P1P1, 2))} })
	kinds := []core.CardType{core.TypeEnchantment, core.TypeArtifact, core.TypeEnchantment}
	modification := CopyAddCardTypes(kinds...)
	kinds[0] = core.TypeLand
	values, err := plan.CopiableValues().WithModifications(modification)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		card, err := values.Build()
		if err != nil {
			t.Fatal(err)
		}
		if !card.HasType(core.TypeCreature) || !card.HasType(core.TypeEnchantment) || !card.HasType(core.TypeArtifact) || card.HasType(core.TypeLand) || len(card.Types()) != 3 {
			t.Fatal("copy types or declaration isolation")
		}
		if card.Abilities()[0].Source() != uuid.Nil {
			t.Fatal("saved source in copy")
		}
	}
	original, err := plan.CopiableValues().Build()
	if err != nil || original.HasType(core.TypeEnchantment) {
		t.Fatal("modified source snapshot", err)
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card, err := values.Build()
	if err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(card, a.PlayerID())
	if p.Counters[core.P1P1] != 2 {
		t.Fatal("lost entry rules")
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, AddCardType(This, core.TypeLand)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	captured, err := g.CaptureCopiableValues(p)
	if err != nil {
		t.Fatal(err)
	}
	rebuilt, err := captured.Build()
	if err != nil || rebuilt.HasType(core.TypeLand) || !rebuilt.HasType(core.TypeEnchantment) {
		t.Fatal("ordinary types entered copiable values", err)
	}
}

func TestCopyTypeModificationInvalid(t *testing.T) {
	for _, modification := range []CopyModification{nil, CopyAddCardTypes(), CopyAddCardTypes(core.CardType(-1)), CopyAddCardTypes(core.TypeInstant)} {
		if _, err := entryCopyPlan(t, "Donor", func() []CardOption { return nil }).CopiableValues().WithModifications(modification); err == nil {
			t.Fatal("invalid snapshot modification accepted")
		}
		card := NewArtifact("Invalid", "", WithAbility(MayCopyAsEntersWithModifications(IsCreature, modification)))
		if err := ValidateCardDefinition("Invalid", card); err == nil {
			t.Fatal("invalid entry modification accepted")
		}
	}
	var absent *CopiableValues
	if _, err := absent.WithModifications(CopyAddCardTypes(core.TypeEnchantment)); err == nil {
		t.Fatal("missing snapshot accepted")
	}
	if err := ValidateCardDefinition("Invalid", NewInstant("Invalid", "", WithAbility(MayCopyAsEntersWithModifications(IsCreature, CopyAddCardTypes(core.TypeArtifact))))); err == nil {
		t.Fatal("instant entry accepted")
	}
}

func TestCopyTypeModificationRepetition(t *testing.T) {
	a := &entryCopyPlayer{BasePlayer: NewBasePlayer("A")}
	b := &optionalEntryPlayer{BasePlayer: NewBasePlayer("B"), accept: false}
	g := NewGame(a, b)
	plan := entryCopyPlan(t, "Cycle", func() []CardOption {
		return []CardOption{WithAbility(MayCopyAsEntersWithModifications(IsCreature, CopyAddCardTypes(core.TypeEnchantment)))}
	})
	donor, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	source := g.PutOnBattlefield(donor, b.PlayerID())
	a.pick = func([]*Permanent, *Game) *Permanent { return source }
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(card, a.PlayerID())
	if p == nil || !p.HasType(core.TypeEnchantment) || a.offers != 1 {
		t.Fatal("type-only reconstruction repeated copy rule")
	}
	g.BouncePermanentToHand(p)
	returned := a.Hand()[0]
	a.RemoveFromHand(returned.ID())
	p = g.PutOnBattlefield(returned, a.PlayerID())
	if p == nil || a.offers != 2 {
		t.Fatal("later entry retained replacement state")
	}
}

func TestCopyTypeModificationBatchAndPriority(t *testing.T) {
	a := &optionalEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	donorPlan := entryCopyPlan(t, "Donor", func() []CardOption { return []CardOption{WithAbility(EntersWithNCounters(core.P1P1, 2))} })
	donor, err := donorPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	source := g.PutOnBattlefield(donor, b.PlayerID())
	plan := entryCopyPlan(t, "Physical", func() []CardOption {
		return []CardOption{WithAbility(MayCopyAsEntersWithModifications(IsCreature, CopyAddCardTypes(core.TypeEnchantment)))}
	})
	seen := 0
	g.effects.AddReplacement(&testPrioritizedReplacement{replacementBase: replacementBase{priority: ReplacementPriorityDefault}, matchFunc: func(action Action) bool { _, ok := action.(*battlefieldEntryAction); return ok }, replaceFunc: func(action Action, _ *Game) Action {
		p := action.(*battlefieldEntryAction).permanent
		if p.Name() != "Donor" || !p.HasType(core.TypeEnchantment) || p.Counters[core.P1P1] != 0 {
			t.Fatal("copy exception did not precede default replacement and entry counters")
		}
		seen++
		return action
	}})
	cards := make([]Card, 0, 2)
	for range 2 {
		card, err := plan.Build()
		if err != nil {
			t.Fatal(err)
		}
		cards = append(cards, card)
	}
	entries := g.putTokensOnBattlefieldTogether(cards, a.PlayerID())
	if len(entries) != 2 || seen != 2 || a.offers != 2 {
		t.Fatal("batch did not keep independent entries")
	}
	for _, p := range entries {
		if !p.HasType(core.TypeEnchantment) || p.Counters[core.P1P1] != 2 {
			t.Fatal("batch exception or entry rules")
		}
	}
	entries[0].RemoveCounter(core.P1P1, 1)
	if entries[1].Counters[core.P1P1] != 2 || source.Counters[core.P1P1] != 2 {
		t.Fatal("batch copied mutable state")
	}
}
