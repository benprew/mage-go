package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestControllerCastDetectionAndBindings(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	caster := BindPlayer("caster")
	spell := BindSpell("spell")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().ByController().Matching(SpellCardMatches(IsEnchantmentCard)).WithCaster(caster).WithSpell(spell),
		Sequence(GainLifeFor(FromPlayer(caster), Fixed(1)), Counter(FromSpell(spell))))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	source := g.PutOnBattlefield(watcher, a.PlayerID())
	for _, tc := range []struct {
		player *BasePlayer
		card   Card
	}{
		{a, NewArtifact("Artifact", "")},
		{b, NewEnchantment("Opponent enchantment", "")},
		{a, NewEnchantment("Own enchantment", "")},
	} {
		tc.card.SetOwner(tc.player.PlayerID())
		obj := &StackObject{ID: uuid.New(), Card: tc.card, SourceID: tc.card.ID(), Controller: tc.player.PlayerID()}
		g.pushStack(obj)
		g.FireEvent(core.GameEvent{Type: core.EvtSpellCast, SourceID: tc.card.ID(), TargetID: obj.ID, PlayerID: tc.player.PlayerID()})
	}
	victim := g.StackPeek()
	victim.Controller = b.PlayerID()
	g.BouncePermanentToHand(source)
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 4 {
		t.Fatalf("stack=%d, want 4", g.StackSize())
	}
	clone := g.Clone()
	g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	g.ResolveTopOfStack()
	g.ResolveTopOfStack()
	if a.Life() != 22 || b.Life() != 20 || g.GetStack().FindByID(victim.ID) != nil {
		t.Fatal("pending event lost its saved caster or spell")
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetStack().FindByID(victim.ID) == nil {
		t.Fatal("live resolution changed clone")
	}
	clone.ResolveTopOfStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || clone.GetStack().FindByID(victim.ID) != nil {
		t.Fatal("clone lost event bindings")
	}
}

func TestControllerCastDeclarations(t *testing.T) {
	for _, pattern := range []spellCastPattern{
		SpellCast().ByController().ByController(),
		SpellCast().ByController().ByOpponent(),
		SpellCast().ByOpponent().ByController(),
		SpellCast().ByController().WithCaster(nil),
		SpellCast().ByController().Matching(SpellFilter{}),
	} {
		card := NewArtifact("Invalid", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid controller cast declaration accepted")
		}
	}
}
