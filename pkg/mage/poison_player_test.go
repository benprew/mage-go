package mage

import "testing"

func TestAddPoisonFor(t *testing.T) {
	for _, amount := range []int{-2, 0, 1, 2, 10} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		player := BindPlayer("damaged")
		card := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, player), AddPoisonFor(FromPlayer(player), Fixed(amount)))))
		if err := ValidateCardDefinition("Source", card); err != nil {
			t.Fatal(err)
		}
		p := g.PutOnBattlefield(card, a.PlayerID())
		g.DealDamageToPlayer(b, 1, p.ID())
		clone := g.Clone()
		g.PutTriggersOnStack()
		g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
		g.ResolveStack()
		want := max(0, amount)
		if a.PoisonCounters() != 0 || b.PoisonCounters() != 2*want || b.Life() != 19 {
			t.Fatal("wrong poison recipient, amount, or life")
		}
		if clone.GetPlayer(b.PlayerID()).PoisonCounters() != 0 {
			t.Fatal("live poison changed clone")
		}
		clone.PutTriggersOnStack()
		clone.ResolveStack()
		if clone.GetPlayer(b.PlayerID()).PoisonCounters() != want {
			t.Fatal("clone lost captured player")
		}
	}
}

func TestAddPoisonForValidation(t *testing.T) {
	for _, effect := range []ResolvingEffect{AddPoisonFor(nil, Fixed(1)), AddPoisonFor(You, nil), AddPoisonFor(FromPlayer(BindPlayer("unbound")), Fixed(1))} {
		if err := ValidateCardDefinition("Bad", NewInstant("Bad", "", Spell(effect))); err == nil {
			t.Fatal("invalid poison declaration accepted")
		}
	}
	g := NewGame(NewBasePlayer("A"), NewBasePlayer("B"))
	if err := AddPoisonFor(FromPlayer(BindPlayer("absent")), Fixed(2)).Apply(&EffectContext{Game: g}); err != nil {
		t.Fatal(err)
	}
	for _, p := range g.players {
		if p.PoisonCounters() != 0 {
			t.Fatal("missing reference changed poison")
		}
	}
}

func TestAddPoisonForEventValues(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player, amount := BindPlayer("damaged"), BindNumber("damage")
	card := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, player).WithAmount(amount), AddPoisonFor(FromPlayer(player), FromNumber(amount)))))
	if err := ValidateCardDefinition("Source", card); err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(card, a.PlayerID())
	g.DealDamageToPlayer(a, 2, p.ID())
	g.DealDamageToPlayer(b, 3, p.ID())
	g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.PoisonCounters() != 2 || b.PoisonCounters() != 3 {
		t.Fatal("event player or amount was overwritten")
	}
}
