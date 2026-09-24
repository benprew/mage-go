package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTypedCounterAmountContext(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Recipient", "{1}", 1, 4)
	amount := NewCreature("Amount", "{3}", 3, 4)
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(amount, b.PlayerID())
	saved := BindPermanent("recipient")
	body := Sequence(Snapshot(This, saved), AddCountersTo(From(saved), core.P1P1, ValueAtResolution(TargetPermanentManaValue())))
	if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), []uuid.UUID{amount.ID()}); err != nil {
		t.Fatal(err)
	}
	if got := g.FindPermanent(source.ID()).Counters[core.P1P1]; got != 3 {
		t.Fatalf("counters = %d, want 3 from outer target", got)
	}
	clone := g.Clone()
	body = Sequence(Snapshot(This, saved), AddCountersTo(From(saved), core.P1P1, PowerOf(From(saved))))
	if err := ApplyEffect(clone, body, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if clone.FindPermanent(source.ID()).Counters[core.P1P1] != 7 || g.FindPermanent(source.ID()).Counters[core.P1P1] != 3 {
		t.Fatal("saved amount or clone isolation changed")
	}
}

func TestTypedCounterAmountReplacementAndAbsence(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Recipient", "", 1, 4)
	g.PutOnBattlefield(source, a.PlayerID())
	g.AddCounterDoubler(source.ID(), core.P1P1, IsCreature)
	calls := 0
	effect := AddCountersTo(This, core.P1P1, MinValue(Fixed(2), countedLifeAmount{&calls}))
	if err := ApplyEffect(g, effect, uuid.New(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("absent subject evaluated amount")
	}
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || g.FindPermanent(source.ID()).Counters[core.P1P1] != 4 {
		t.Fatal("amount capture or replacement failed")
	}
	for _, n := range []int{0, -2} {
		if err := ApplyEffect(g, AddCountersTo(This, core.P1P1, Fixed(n)), source.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if g.FindPermanent(source.ID()).Counters[core.P1P1] != 4 {
		t.Fatal("nonpositive amount changed counters")
	}
}

func TestTypedCounterAmountValidation(t *testing.T) {
	for _, amount := range []ResolutionValue{nil, PowerOf(nil), PowerOf(From(BindPermanent("missing"))), PowerOf(DeclarePermanentTarget("undeclared", IsCreature)), ValueAtResolution(nil)} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), AddCountersTo(This, core.P1P1, amount))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid amount accepted")
		}
	}
	calls := 0
	info := InspectDefinition(AddCountersTo(This, core.P1P1, countedLifeAmount{&calls}))
	if calls != 0 || info.Properties.Outcome != OutcomeUnknown {
		t.Fatal("inspection evaluated dynamic amount")
	}
}

func TestTypedCounterAmountCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	recipient := NewCreature("Recipient", "", 2, 8)
	g.PutOnBattlefield(recipient, a.PlayerID())
	target := DeclarePermanentTarget("recipient", IsCreature)
	ability := Spell(AddCountersTo(target, core.P1P1, PowerOf(target)), WithTargets(target))
	card := NewSorcery("Counters", "", WithAbility(ability))
	original := &StackObject{ID: card.ID(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{recipient.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(original)
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy missing")
	}
	g.ResolveStackObject(copied)
	if got := g.FindPermanent(recipient.ID()).Counters[core.P1P1]; got != 2 {
		t.Fatalf("copy counters = %d, want 2", got)
	}
	g.ResolveStackObject(original)
	if got := g.FindPermanent(recipient.ID()).Counters[core.P1P1]; got != 6 {
		t.Fatalf("original counters = %d, want 6 from current power", got)
	}
}
