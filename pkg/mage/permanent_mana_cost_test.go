package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentManaCostPayment(t *testing.T) {
	for _, tc := range []struct {
		cost string
		mana int
		want bool
	}{{"{2}{G}", 3, true}, {"{2}{G}", 2, false}, {"{X}{X}{G}", 1, true}, {"{0}", 0, true}, {"", 10, false}} {
		t.Run(tc.cost, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			host := g.PutOnBattlefield(NewCreature("Host", tc.cost, 2, 3), b.PlayerID())
			source := g.PutOnBattlefield(NewAura("Aura", "{U}", WithActivatedAbility(GainLifeFor(You, Fixed(1)), PermanentManaCostPayment(AttachedPermanent), WithAnyPlayerMay())), a.PlayerID())
			g.Attach(source.ID(), host.ID())
			b.ManaPool().Add(core.Green, tc.mana)
			ability := source.Card.Abilities()[0].(ActivatedAbility)
			for range 2 {
				if ability.CanActivate(b.PlayerID(), g) != tc.want {
					t.Fatal("query ignored permanent cost")
				}
			}
			if b.ManaPool().TotalMana() != tc.mana {
				t.Fatal("query spent mana")
			}
			err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, nil)
			if (err == nil) != tc.want {
				t.Fatalf("activation: %v", err)
			}
			if !tc.want {
				if b.ManaPool().TotalMana() != tc.mana || g.StackSize() != 0 {
					t.Fatal("failed payment changed state")
				}
				return
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			if b.Life() != 21 || clone.GetPlayer(b.PlayerID()).Life() != 21 || a.Life() != 20 || b.ManaPool().TotalMana() != 0 {
				t.Fatal("payer or clone result")
			}
		})
	}
}

func TestPermanentManaCostPaymentAggregateAndLock(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host := g.PutOnBattlefield(NewCreature("Host", "{G}", 2, 3), a.PlayerID())
	source := g.PutOnBattlefield(NewAura("Aura", "{U}", WithActivatedAbility(GainLifeFor(You, Fixed(1)), PermanentManaCostPayment(AttachedPermanent), WithCost(ManaCostOf("{G}")))), a.PlayerID())
	g.Attach(source.ID(), host.ID())
	a.ManaPool().Add(core.Green, 1)
	if source.Card.Abilities()[0].(ActivatedAbility).CanActivate(a.PlayerID(), g) {
		t.Fatal("separate mana checks reused one mana")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil || a.ManaPool().TotalMana() != 1 {
		t.Fatal("aggregate failure spent mana")
	}
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{PermanentManaCostPayment(AttachedPermanent)}, ApplyActivationReduction: true})
	if err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(host)
	if err := tx.Commit(); err != nil {
		t.Fatal("locked mana cost changed after host departure", err)
	}
	if a.ManaPool().TotalMana() != 0 {
		t.Fatal("locked mana not spent")
	}
}

func TestPermanentManaCostPaymentInvalid(t *testing.T) {
	card := NewArtifact("Invalid", "{1}", WithActivatedAbility(GainLifeFor(You, Fixed(1)), PermanentManaCostPayment(nil)))
	if err := ValidateCardDefinition("invalid", card); err == nil {
		t.Fatal("nil subject accepted")
	}
}

func TestPermanentManaCostPaymentModifiersAndRollback(t *testing.T) {
	for _, mode := range []string{"reduction", "tax", "no-cost-reduction", "mana", "rollback"} {
		t.Run(mode, func(t *testing.T) {
			g := newPriorityTestGame()
			a := g.players[0]
			id := a.PlayerID()
			cost := "{2}{G}"
			if mode == "no-cost-reduction" {
				cost = ""
			}
			host := g.PutOnBattlefield(NewCreature("Host", cost, 2, 3), id)
			parts := []ActionOption{}
			if mode == "rollback" {
				parts = append(parts, WithCost(LifePayCost(21)))
			}
			source := g.PutOnBattlefield(NewAura("Aura", "{U}", WithActivatedAbility(GainLifeFor(You, Fixed(1)), PermanentManaCostPayment(AttachedPermanent), parts...)), id)
			g.Attach(source.ID(), host.ID())
			if mode == "reduction" || mode == "no-cost-reduction" {
				reduction := g.PutOnBattlefield(NewAura("Reduction", "{U}", WithCastTarget(TargetPermanent(IsEnchantment)), WithStaticAbility(ReduceActivationCostStatic(AttachedPermanent, 2, 0))), id)
				g.Attach(reduction.ID(), source.ID())
			}
			if mode == "tax" {
				g.PutOnBattlefield(NewEnchantment("Tax", "{U}", WithStaticAbility(IncreaseActivationCostStatic(IsEnchantment, core.ManaCost{Blue: 1}))), id)
				a.ManaPool().Add(core.Blue, 1)
			}
			lands := []*Permanent{}
			if mode == "mana" || mode == "rollback" {
				for range 3 {
					lands = append(lands, addLand(t, g, id, "Forest", core.Green))
				}
			} else {
				amount := 3
				if mode == "reduction" {
					amount = 1
				}
				a.ManaPool().Add(core.Green, amount)
			}
			before := a.ManaPool().TotalMana()
			err := g.ActivateAbilityByIndex(id, source.ID(), 0, nil)
			fail := mode == "rollback" || mode == "no-cost-reduction"
			if (err != nil) != fail {
				t.Fatalf("activation %v", err)
			}
			if fail {
				if a.Life() != 20 || a.ManaPool().TotalMana() != before || g.StackSize() != 0 {
					t.Fatal("failure spent resources")
				}
				for _, land := range lands {
					if land.Tapped {
						t.Fatal("failure tapped mana source")
					}
				}
				return
			}
			if a.ManaPool().TotalMana() != 0 {
				t.Fatal("incorrect aggregate payment")
			}
			for _, land := range lands {
				if !g.FindPermanent(land.ID()).Tapped {
					t.Fatal("mana ability omitted")
				}
			}
			g.ResolveStack()
			if a.Life() != 21 {
				t.Fatal("effect missing")
			}
		})
	}
}

func TestPermanentManaCostPaymentSubjectsAndCostFreeze(t *testing.T) {
	g := newPriorityTestGame()
	a := g.players[0]
	id := a.PlayerID()
	host := g.PutOnBattlefield(NewCreature("Host", "{G}", 2, 3), id)
	source := g.PutOnBattlefield(NewAura("Aura", "{U}"), id)
	g.Attach(source.ID(), host.ID())
	payment := PermanentManaCostPayment(AttachedPermanent)
	a.ManaPool().Add(core.Green, 3)
	phase := g.PhaseOut(host.ID())
	if payment.CanPay(source.ID(), id, g) {
		t.Fatal("phased host supplied mana cost")
	}
	g.PhaseIn(phase)
	if !payment.CanPay(source.ID(), id, g) {
		t.Fatal("phase cycle lost host")
	}
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: source.ID(), Controller: id, Costs: []Cost{payment}, ApplyActivationReduction: true})
	if err != nil {
		t.Fatal(err)
	}
	g.MutablePermanent(host.ID()).Card.(*BaseCard).manaCost = core.ParseManaCost("{99}{U}")
	if err := tx.Commit(); err != nil || a.ManaPool().TotalMana() != 2 {
		t.Fatal("changed characteristics replaced locked cost", err)
	}
	if payment.CanPay(source.ID(), id, g) {
		t.Fatal("next payment reused old cost")
	}
	other := g.PutOnBattlefield(NewCreature("Other", "{0}", 2, 3), id)
	g.Attach(source.ID(), other.ID())
	if !payment.CanPay(source.ID(), id, g) {
		t.Fatal("new host ignored")
	}
	phase = g.PhaseOut(source.ID())
	if payment.CanPay(source.ID(), id, g) {
		t.Fatal("phased source used LKI")
	}
	g.PhaseIn(phase)
	g.RemoveFromBattlefield(other)
	g.PutOnBattlefield(other.Card, id)
	if payment.CanPay(source.ID(), id, g) {
		t.Fatal("returned host replaced attachment visit")
	}
	if PermanentManaCostPayment(This).CanPay(source.ID(), id, g) {
		t.Fatal("source colored cost paid with green")
	}
	a.ManaPool().Add(core.Blue, 1)
	if err := PermanentManaCostPayment(This).Pay(source.ID(), id, g); err != nil {
		t.Fatal(err)
	}
}

func TestPermanentManaCostPaymentCopiedHost(t *testing.T) {
	g := newPriorityTestGame()
	a := g.players[0]
	id := a.PlayerID()
	plan, err := NewCardDefinitionPlan(func() Card { return NewCreature("Model", "{X}{G}", 2, 3) })
	if err != nil {
		t.Fatal(err)
	}
	modelCard, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	model := g.PutOnBattlefield(modelCard, id)
	values, err := g.CaptureCopiableValues(model)
	if err != nil {
		t.Fatal(err)
	}
	copied, err := BuildEntryCopy(entryCopyPhysical(t), values)
	if err != nil {
		t.Fatal(err)
	}
	host := g.PutOnBattlefield(copied, id)
	source := g.PutOnBattlefield(NewAura("Aura", "{U}"), id)
	g.Attach(source.ID(), host.ID())
	a.ManaPool().Add(core.Green, 1)
	if err := PermanentManaCostPayment(AttachedPermanent).Pay(source.ID(), id, g); err != nil || a.ManaPool().TotalMana() != 0 {
		t.Fatal("copy used physical mana cost", err)
	}
	host = g.PutOnBattlefield(newPermanentSpellCopyCard(modelCard.Copy()), id)
	g.Attach(source.ID(), host.ID())
	a.ManaPool().Add(core.Green, 1)
	if err := PermanentManaCostPayment(AttachedPermanent).Pay(source.ID(), id, g); err != nil {
		t.Fatal("spell copy lost cost presence", err)
	}
}

func TestPermanentManaCostPaymentHybridFreeze(t *testing.T) {
	g := newPriorityTestGame()
	a := g.players[0]
	id := a.PlayerID()
	host := g.PutOnBattlefield(NewCreature("Host", "{G/U}", 2, 3), id)
	source := g.PutOnBattlefield(NewAura("Aura", "{U}"), id)
	g.Attach(source.ID(), host.ID())
	a.ManaPool().Add(core.Green, 1)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: source.ID(), Controller: id, Costs: []Cost{PermanentManaCostPayment(AttachedPermanent)}, ApplyActivationReduction: true})
	if err != nil {
		t.Fatal(err)
	}
	g.MutablePermanent(host.ID()).Card.(*BaseCard).manaCost.Hybrid[0].A = core.Red
	if err := tx.Commit(); err != nil {
		t.Fatal("hybrid cost was not frozen", err)
	}
}
