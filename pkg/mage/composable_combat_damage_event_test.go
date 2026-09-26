package mage

import (
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSourceCombatDamageToPlayerEvent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player, amount := BindPlayer("damaged"), BindNumber("amount")
	source := NewCreature("Source", "", 2, 3, WithAbility(WhenEvent(
		DealsCombatDamageToPlayer(This, player).WithAmount(amount),
		GainLifeFor(FromPlayer(player), FromNumber(amount)))))
	g.PutOnBattlefield(source, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtDamageDealt, SourceID: source.ID(), TargetID: b.PlayerID(), Amount: 3})
	phased := g.PhaseOut(source.ID())
	g.FireEvent(GameEvent{Type: EvtDamageDealt, SourceID: source.ID(), TargetID: b.PlayerID(), Amount: 2, Flag: true})
	g.PhaseIn(phased)
	g.FireEvent(GameEvent{Type: EvtDamageDealt, SourceID: source.ID(), TargetID: b.PlayerID(), Amount: 2, Flag: true})
	g.FireEvent(GameEvent{Type: EvtDamageDealt, SourceID: source.ID(), TargetID: source.ID(), Amount: 1, Flag: true})
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers = %d", g.StackSize())
	}
	for _, obj := range g.StackObjects() {
		if g.CopyStackObjectDirect(obj, b.PlayerID(), false) == nil {
			t.Fatal("trigger copy failed")
		}
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if got := game.GetPlayer(b.PlayerID()).Life(); got != 24 {
			t.Fatalf("damaged player life = %d", got)
		}
	}
}

func TestSourceCombatDamageToPlayerRealCombatAndPrevention(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p, n := BindPlayer("damaged"), BindNumber("amount")
	source := g.PutOnBattlefield(NewCreature("Source", "", 3, 3, WithAbility(WhenEvent(
		DealsCombatDamageToPlayer(This, p).WithAmount(n),
		GainLifeFor(FromPlayer(p), FromNumber(n))))), a.PlayerID())
	g.AddPreventionShield(b.PlayerID(), 1)
	g.combat.AddAttacker(source.ID(), b.PlayerID())
	g.ExecuteCombatDamage()
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers after partial prevention = %d", g.StackSize())
	}
	g.ResolveStack()
	if got := b.Life(); got != 20 {
		t.Fatalf("damage and trigger result = %d", got)
	}
	g.AddPreventionShield(b.PlayerID(), 3)
	g.ExecuteCombatDamage()
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatalf("triggers after full prevention = %d", g.StackSize())
	}
}

func TestSourceCombatDamageToPlayerDeclaration(t *testing.T) {
	p, n := BindPlayer("player"), BindNumber("amount")
	for _, tc := range []struct {
		name    string
		pattern EventPattern
		want    string
	}{
		{"valid", DealsCombatDamageToPlayer(This, p).WithAmount(n), ""},
		{"nil player", DealsCombatDamageToPlayer(This, nil), "non-nil"},
		{"nil amount", DealsCombatDamageToPlayer(This, p).WithAmount(nil), "non-nil"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewCreature("Test", "", 1, 1, WithAbility(WhenEvent(tc.pattern, GainLifeFor(You, Fixed(1)))))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
	if got := InspectDefinition(DealsCombatDamageToPlayer(This, p)); got.Role != "event-pattern" {
		t.Fatalf("inspection = %#v", got)
	}
}
