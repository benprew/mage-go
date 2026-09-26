package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDeathEventCardVisit(t *testing.T) {
	for _, leave := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		dead := BindCard("dead")
		watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(AnotherCreatureDies().WithCard(dead), ReturnCardToHand(FromCard(dead)))))
		watcher.SetOwner(a.PlayerID())
		victim := NewCreature("Victim", "", 2, 2)
		victim.SetOwner(b.PlayerID())
		if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
			t.Fatal(err)
		}
		g.PutOnBattlefield(watcher, a.PlayerID())
		g.PutOnBattlefield(victim, b.PlayerID())
		g.DestroyPermanent(g.FindPermanent(victim.ID()))
		g.PutTriggersOnStack()
		if g.StackSize() != 1 {
			t.Fatalf("stack=%d", g.StackSize())
		}
		if leave {
			card, _, ok := g.MoveFromAnyGraveyard(victim.ID(), core.ZoneExile)
			if !ok {
				t.Fatal("first move failed")
			}
			b.AddToGraveyard(card)
		}
		clone := g.Clone()
		clone.ResolveStack()
		if len(b.Hand()) != 0 {
			t.Fatal("clone changed original hand")
		}
		g.ResolveStack()
		if got := len(b.Hand()); got != map[bool]int{false: 1, true: 0}[leave] {
			t.Fatalf("leave=%v, hand=%d", leave, got)
		}
	}
}

func TestDeathEventCardDeclaration(t *testing.T) {
	b := BindCard("dead")
	for _, tc := range []struct {
		name    string
		pattern EventPattern
		body    ResolvingEffect
		want    string
	}{
		{"nil", CreatureDies().WithCard(nil), GainLifeFor(You, Fixed(1)), "binding"},
		{"empty", CreatureDies().WithCard(BindCard("")), GainLifeFor(You, Fixed(1)), "binding name"},
		{"missing", CreatureDies(), ReturnCardToHand(FromCard(b)), "producer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(tc.pattern, tc.body)))
			if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("validation=%v, want %q", err, tc.want)
			}
		})
	}
	wrongScope := NewCreature("Wrong scope", "", 1, 1,
		WithAbility(WhenEvent(CreatureDies().WithCard(b), GainLifeFor(You, Fixed(1)))),
		WithAbility(Activate(Costs(), ReturnCardToHand(FromCard(b)))))
	if err := ValidateCardDefinition(wrongScope.Name(), wrongScope); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("scope validation=%v", err)
	}
	duplicate := NewCreature("Duplicate", "", 1, 1, WithAbility(WhenEvent(
		CreatureDies().WithCard(BindCard("same")),
		Sequence(ReturnCardToHand(FromCard(BindCard("same"))), GainLifeFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil {
		t.Fatal("separate declarations with the same label were accepted")
	}
}

func TestDeathEventCardSimultaneousAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	dead := BindCard("dead")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(AnotherCreatureDies().WithCard(dead), ReturnCardToHand(FromCard(dead)))))
	watcher.SetOwner(a.PlayerID())
	first, second := NewCreature("First", "", 1, 1), NewCreature("Second", "", 1, 1)
	first.SetOwner(b.PlayerID())
	second.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(first, b.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.DestroyPermanents([]*Permanent{g.FindPermanent(first.ID()), g.FindPermanent(second.ID())})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack=%d, want 2", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if len(b.Hand()) != 2 || len(b.Graveyard()) != 0 {
		t.Fatalf("hand=%d, graveyard=%d", len(b.Hand()), len(b.Graveyard()))
	}
}

func TestDeathEventCardPhasedSourceAndToken(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	dead := BindCard("dead")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(AnotherCreatureDies().WithCard(dead), ReturnCardToHand(FromCard(dead)))))
	watcher.SetOwner(a.PlayerID())
	token := NewToken("Token", 1, 1, []core.CardType{core.TypeCreature}, nil)
	token.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(token, b.PlayerID())
	g.DestroyPermanent(g.FindPermanent(token.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("token trigger stack=%d", g.StackSize())
	}
	g.ResolveStack()
	if len(b.Hand()) != 0 {
		t.Fatal("token became a card in hand")
	}
	g.PhaseOut(watcher.ID())
	victim := NewCreature("Victim", "", 1, 1)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.DestroyPermanent(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatalf("phased source trigger stack=%d", g.StackSize())
	}
}

func TestDeathEventCardSourceDiesSimultaneously(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	dead := BindCard("other dead creature")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureDies().WithCard(dead), ReturnCardToHand(FromCard(dead)))))
	watcher.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 1, 1)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.DestroyPermanents([]*Permanent{g.FindPermanent(watcher.ID()), g.FindPermanent(victim.ID())})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack=%d, want 1", g.StackSize())
	}
	g.ResolveStack()
	if len(b.Hand()) != 1 || len(a.Graveyard()) != 1 {
		t.Fatalf("victim hand=%d, source graveyard=%d", len(b.Hand()), len(a.Graveyard()))
	}
}
