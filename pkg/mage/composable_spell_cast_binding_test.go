package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSpellCastBindingUsesCastStackIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	spell := BindSpell("cast spell")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().WithSpell(spell),
		Sequence(GainLifeFor(You, ManaValueOfSpell(FromSpell(spell))), Counter(FromSpell(spell))))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	card := NewInstant("Cast card", "{X}{U}")
	card.SetOwner(b.PlayerID())
	old := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: b.PlayerID(), XValue: 1}
	cast := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: b.PlayerID(), XValue: 3}
	g.pushStack(old)
	g.pushStack(cast)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: cast.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatalf("stack=%d, want 3", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveTopOfStack()
	if a.Life() != 20 || g.StackSize() != 3 {
		t.Fatal("clone changed original game")
	}
	g.ResolveTopOfStack()
	if a.Life() != 24 || g.stack.FindByID(cast.ID) != nil || g.stack.FindByID(old.ID) == nil {
		t.Fatalf("life=%d, cast=%v, old=%v", a.Life(), g.stack.FindByID(cast.ID), g.stack.FindByID(old.ID))
	}
}

func TestSpellCastBindingAbsentStackObject(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	spell := BindSpell("cast spell")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().WithSpell(spell), GainLifeFor(You, ManaValueOfSpell(FromSpell(spell))))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	card := NewInstant("Cast card", "{3}{U}")
	card.SetOwner(b.PlayerID())
	cast := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: b.PlayerID()}
	g.pushStack(cast)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: cast.ID, PlayerID: b.PlayerID()})
	g.stack.RemoveByID(cast.ID)
	g.pushStack(&StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: b.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveTopOfStack()
	if a.Life() != 20 {
		t.Fatalf("returned stack card supplied old binding: life=%d", a.Life())
	}
}

func TestSpellCastBindingCopyAndSourcePhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	spell := BindSpell("cast spell")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().WithSpell(spell), GainLifeFor(You, ManaValueOfSpell(FromSpell(spell))))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	card := NewInstant("Cast card", "{2}{U}")
	card.SetOwner(b.PlayerID())
	cast := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: b.PlayerID()}
	g.pushStack(cast)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: cast.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	g.PhaseOut(watcher.ID())
	g.ResolveTopOfStack()
	g.ResolveTopOfStack()
	if a.Life() != 23 || b.Life() != 23 {
		t.Fatalf("copied captures: life A=%d B=%d", a.Life(), b.Life())
	}
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: cast.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("phased watcher created trigger: stack=%d", g.StackSize())
	}
}

func TestSpellCastBindingDeclaration(t *testing.T) {
	spell := BindSpell("spell")
	if got := InspectDefinition(SpellCast().WithSpell(spell)); got.Role != "event-pattern" || len(got.Children) != 1 {
		t.Fatalf("inspection=%+v", got)
	}
	for _, pattern := range []EventPattern{
		SpellCast().WithSpell(nil),
		SpellCast().WithSpell(BindSpell("")),
		SpellCast().WithSpell(spell).WithSpell(spell),
	} {
		bad := NewArtifact("Bad", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("invalid spell binding passed registration")
		}
	}
	bad := NewArtifact("Unbound", "", WithAbility(WhenEvent(SpellCast(), Counter(FromSpell(spell)))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("scope validation=%v", err)
	}
}
