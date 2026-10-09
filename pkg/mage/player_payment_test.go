package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPlayerPaymentLifetime(t *testing.T) {
	for _, change := range []string{"live", "leave", "return", "phase", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			effect := PlayerManaPaymentThisTurn(You, core.ParseManaCost("{1}"), GainLifeFor(You, Fixed(2)))
			if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "leave", "return":
				g.RemoveFromBattlefield(source)
				if change == "return" {
					g.PutOnBattlefield(source.Card, b.PlayerID())
				}
			case "phase":
				g.PhaseOut(source.ID())
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			if len(g.GetPlayerPayments(a.PlayerID())) != 0 {
				t.Fatal("unaffordable payment offered")
			}
			a.ManaPool().Add(core.Colorless, 3)
			offers := g.GetPlayerPayments(a.PlayerID())
			if len(offers) != 1 || len(g.GetPlayerPayments(b.PlayerID())) != 0 {
				t.Fatal("wrong payer")
			}
			id := offers[0].ID
			clone := g.Clone()
			for range 2 {
				if err := g.TakePlayerPayment(a.PlayerID(), id); err != nil {
					t.Fatal(err)
				}
			}
			if a.Life() != 24 || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
				t.Fatal("payment did not resolve immediately")
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 3 {
				t.Fatal("clone shared payment state")
			}
			if err := g.TakePlayerPayment(b.PlayerID(), id); err == nil {
				t.Fatal("opponent paid")
			}
			g.doCleanupActions()
			if err := g.TakePlayerPayment(a.PlayerID(), id); err == nil {
				t.Fatal("expired payment executed")
			}
			if len(clone.GetPlayerPayments(a.PlayerID())) != 1 {
				t.Fatal("cleanup expired clone")
			}
			if err := clone.TakePlayerPayment(a.PlayerID(), id); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPlayerPaymentDeclarations(t *testing.T) {
	for _, effect := range []ResolvingEffect{
		PlayerManaPaymentThisTurn(nil, core.ParseManaCost("{1}"), GainLifeFor(You, Fixed(1))),
		PlayerManaPaymentThisTurn(You, core.ParseManaCost("{X}"), GainLifeFor(You, Fixed(1))),
		PlayerManaPaymentThisTurn(You, core.ManaCost{Generic: -1}, GainLifeFor(You, Fixed(1))),
		PlayerManaPaymentThisTurn(You, core.ParseManaCost("{1}"), nil),
		PlayerManaPaymentThisTurn(FromPlayer(BindPlayer("missing")), core.ParseManaCost("{1}"), GainLifeFor(You, Fixed(1))),
	} {
		c := NewInstant("Invalid", "", WithAbility(Spell(effect)))
		if err := ValidateCardDefinition(c.Name(), c); err == nil {
			t.Fatal("invalid payment accepted")
		}
	}
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	if err := g.TakePlayerPayment(a.PlayerID(), uuid.New()); err == nil {
		t.Fatal("unknown payment executed")
	}
}

func TestPlayerPaymentFramesAndRollback(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	n := BindNumber("amount")
	body := Sequence(GainLifeFor(You, FromNumber(n)), SnapshotNumber(Fixed(99), n))
	effect := Sequence(SnapshotNumber(Fixed(3), n), PlayerManaPaymentThisTurn(You, core.ParseManaCost("{1}"), body), SnapshotNumber(Fixed(7), n))
	c := NewInstant("Frame", "", WithAbility(Spell(effect)))
	if err := ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	id := g.playerPayments[0].id
	g.resolution.numberBindings = map[*NumberBinding]int{n: 7}
	g.resolution.SetX(42)
	if err := g.TakePlayerPayment(a.PlayerID(), id); err == nil {
		t.Fatal("unaffordable payment executed")
	}
	if a.Life() != 20 || g.resolution.numberBindings[n] != 7 || g.resolution.X() != 42 {
		t.Fatal("failed payment changed frame")
	}
	a.ManaPool().Add(core.Colorless, 2)
	clone := g.Clone()
	for range 2 {
		if err := g.TakePlayerPayment(a.PlayerID(), id); err != nil {
			t.Fatal(err)
		}
	}
	if a.Life() != 26 || g.resolution.numberBindings[n] != 7 || g.resolution.X() != 42 {
		t.Fatal("stored or ambient frame changed")
	}
	if err := clone.TakePlayerPayment(a.PlayerID(), id); err != nil {
		t.Fatal(err)
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 23 {
		t.Fatal("clone lost saved number")
	}
	for _, invalid := range []ResolvingEffect{
		Sequence(PlayerManaPaymentThisTurn(You, core.ManaCost{}, SnapshotNumber(Fixed(1), n)), GainLifeFor(You, FromNumber(n))),
		PlayerManaPaymentThisTurn(You, core.ManaCost{}, GainLifeFor(You, FromNumber(n))),
	} {
		if err := ValidateCardDefinition("Invalid", NewInstant("Invalid", "", WithAbility(Spell(invalid)))); err == nil {
			t.Fatal("binding escaped its producer scope")
		}
	}
}

func TestPlayerPaymentPriorityAndTiming(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	if err := ApplyEffect(g, PlayerManaPaymentThisTurn(You, core.ParseManaCost("{1}"), GainLifeFor(You, Fixed(1))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	a.ManaPool().Add(core.Colorless, 1)
	id := g.GetPlayerPayments(a.PlayerID())[0].ID
	g.resolution.resolvingController = b.PlayerID()
	if len(g.GetPlayerPayments(a.PlayerID())) != 0 {
		t.Fatal("payment offered during resolution")
	}
	if err := g.TakePlayerPayment(a.PlayerID(), id); err == nil {
		t.Fatal("payment taken during resolution")
	}
	g.resolution.resolvingController = uuid.Nil
	actions := 0
	g.SetOnPriority(func(game *Game, index int, _ bool) PriorityAction {
		if index == 0 && actions == 0 {
			actions++
			return PriorityAction{Type: PriorityTakePlayerPayment, PlayerPaymentID: id}
		}
		return PriorityAction{Type: PriorityPass}
	})
	g.RunStepWithPriority(core.PrecombatMain)
	if actions != 1 || a.Life() != 21 || g.StackSize() != 0 {
		t.Fatal("priority payment failed")
	}
}

func TestPlayerPaymentManaSources(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := g.PutOnBattlefield(NewLand("Land", WithAbility(NewManaAbility(core.Green))), a.PlayerID())
	effect := PlayerManaPaymentThisTurn(You, core.ParseManaCost("{G}"), GainLifeFor(You, Fixed(1)))
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if len(g.GetPlayerPayments(a.PlayerID())) != 1 || land.Tapped || a.ManaPool().TotalMana() != 0 {
			t.Fatal("query made payment")
		}
	}
	id := g.GetPlayerPayments(a.PlayerID())[0].ID
	if err := g.TakePlayerPayment(a.PlayerID(), id); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(land.ID()).Tapped || a.Life() != 21 {
		t.Fatal("automatic mana payment failed")
	}
	if err := g.TakePlayerPayment(a.PlayerID(), id); err == nil {
		t.Fatal("used a tapped source twice")
	}
	if a.Life() != 21 || g.StackSize() != 0 {
		t.Fatal("failed payment applied body")
	}
}

func TestPlayerPaymentSourceLKIAndIndependentInstallations(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Source", "", 3, 3)
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	effect := PlayerManaPaymentThisTurn(You, core.ManaCost{}, GainLifeFor(You, PowerOfWithLKI(This)))
	if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, effect, p.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	first := g.GetPlayerPayments(a.PlayerID())[0].ID
	second := g.GetPlayerPayments(b.PlayerID())[0].ID
	if first == second {
		t.Fatal("installations share identity")
	}
	g.BouncePermanentToHand(p)
	a.RemoveFromHand(c.ID())
	q := g.PutOnBattlefield(c, b.PlayerID())
	g.AddContinuousEffect(TemporaryBoost(q.ID(), 6, 6))
	g.ApplyContinuousEffects()
	clone := g.Clone()
	if err := g.TakePlayerPayment(a.PlayerID(), first); err != nil {
		t.Fatal(err)
	}
	if err := g.TakePlayerPayment(b.PlayerID(), second); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 23 || b.Life() != 23 {
		t.Fatal("payment read returned source instead of LKI")
	}
	if err := clone.TakePlayerPayment(b.PlayerID(), second); err != nil {
		t.Fatal(err)
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 23 {
		t.Fatal("independent installation changed")
	}
}

func TestPlayerPaymentRetainsPriority(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	if err := ApplyEffect(g, PlayerManaPaymentThisTurn(You, core.ManaCost{}, GainLifeFor(You, Fixed(1))), uuid.Nil, b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	id := g.GetPlayerPayments(b.PlayerID())[0].ID
	var order []int
	paid := false
	g.SetOnPriority(func(_ *Game, index int, _ bool) PriorityAction {
		order = append(order, index)
		if index == 1 && !paid {
			paid = true
			return PriorityAction{Type: PriorityTakePlayerPayment, PlayerPaymentID: id}
		}
		return PriorityAction{Type: PriorityPass}
	})
	g.RunStepWithPriority(core.PrecombatMain)
	if len(order) != 4 || order[0] != 0 || order[1] != 1 || order[2] != 1 || order[3] != 0 {
		t.Fatalf("priority order=%v", order)
	}
	if b.Life() != 21 {
		t.Fatal("opponent payment failed")
	}
}
