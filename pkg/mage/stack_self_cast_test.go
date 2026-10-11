package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestStackSelfCastExactVisit(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	binding := BindSpell("cast")
	card := NewEnchantment("Self cast", "", WithAbility(WhenEvent(SourceSpellCast().WithSpell(binding), Counter(FromSpell(binding)))))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	g.CopyStackObjectDirect(original, b.PlayerID(), false)
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatal("copy created another cast trigger")
	}
	g.CounterStackObject(g.StackPeek().ID)
	g.FireEvent(core.GameEvent{Type: core.EvtSpellCast, SourceID: card.ID(), TargetID: original.ID, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatal("copy inherited another stack visit's self-cast event")
	}
	g.ResolveTopOfStack()
	if g.StackSize() != 1 || g.StackPeek().ID == original.ID {
		t.Fatal("counter did not use captured exact spell")
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(card.ID()) != nil {
		t.Fatal("copy reused physical source")
	}
	if len(a.Graveyard()) != 1 {
		t.Fatal("counter did not keep card owner")
	}
}

func TestStackSelfCastPendingCaster(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(1)
	g.SetStep(core.PrecombatMain)
	caster := BindPlayer("caster")
	card := NewInstant("Self cast draw", "", WithAbility(WhenEvent(SourceSpellCast().WithCaster(caster), GainLifeFor(FromPlayer(caster), Fixed(2)))))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	spell := g.StackPeek()
	spell.Controller = b.PlayerID()
	g.CounterStackObject(spell.ID)
	g.PutTriggersOnStack()
	clone := g.Clone()
	clone.ResolveTopOfStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 20 || a.Life() != 20 {
		t.Fatal("clone lost pending caster or changed original")
	}
	g.ResolveTopOfStack()
	if a.Life() != 22 || b.Life() != 20 {
		t.Fatal("pending caster changed with departed spell")
	}
}

func TestStackSelfCastDeclaration(t *testing.T) {
	spell := BindSpell("self")
	valid := NewInstant("Valid", "", WithAbility(WhenEvent(SourceSpellCast().WithSpell(spell), Counter(FromSpell(spell)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	invalid := NewInstant("Invalid", "", WithAbility(WhenEvent(SourceSpellCast().WithSpell(spell).WithSpell(spell), Counter(FromSpell(spell)))))
	if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil {
		t.Fatal("duplicate event binding accepted")
	}
	invalid = NewInstant("Unbound", "", WithAbility(WhenEvent(SourceSpellCast(), Counter(FromSpell(spell)))))
	if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil {
		t.Fatal("unbound self spell accepted")
	}
}
