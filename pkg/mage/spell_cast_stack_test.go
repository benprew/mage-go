package mage

import (
	"testing"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSpellCastFiltersUseRecastSpell(t *testing.T) {
	g := newPriorityTestGame()
	g.SetStep(PrecombatMain)
	a := g.players[0]
	filter := ManaValueIs(EqualTo(Fixed(3)))
	g.PutOnBattlefield(NewEnchantment("Cast Observer", "{0}",
		WithAbility(WheneverYouCastSpellTrigger(GainLife(1), false, filter)),
		WithAbility(NewTriggeredAbility(WhenPlayerCastsSpell(SelectController(), filter), GainLife(1))),
	), a.PlayerID())

	card := NewCreature("X Creature", "{X}", 2, 2, WithKeyword(Flash),
		WithActivatedAbility(GainLife(2), ManaCostOf("{0}")))
	permanent := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), permanent.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(permanent)
	a.ManaPool().Add(Colorless, 3)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 3); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if got := g.GetStack().Size(); got != 4 {
		t.Fatalf("stack has %d objects, want the old ability, the spell, and both cast triggers", got)
	}
}

func TestSpellCastFiltersUseStackReader(t *testing.T) {
	g := newPriorityTestGame()
	a := g.players[0]
	card := NewInstant("X Spell", "{X}")
	g.pushStack(&StackObject{
		ID:         uuid.New(),
		Card:       card,
		SourceID:   card.ID(),
		Controller: a.PlayerID(),
		XValue:     3,
	})
	event := GameEvent{Type: EvtSpellCast, SourceID: card.ID(), PlayerID: a.PlayerID()}
	filter := SpellCastMatchesObjectFilters{Filters: []ObjectFilter{ManaValueIs(EqualTo(Fixed(3)))}}
	reader := struct{ GameReader }{g}
	if !filter.CheckTriggerCond(&event, reader, uuid.Nil(), a.PlayerID()) {
		t.Fatal("cast filter did not read the spell's announced mana value through GameReader")
	}
}
