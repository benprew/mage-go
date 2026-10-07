package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestStaticActivationCostTotal(t *testing.T) {
	for _, tc := range []struct {
		base core.ManaCost
		want core.ManaCost
	}{
		{core.ManaCost{}, core.ManaCost{Generic: 1}},
		{core.ManaCost{Generic: 4}, core.ManaCost{Generic: 3}},
		{core.ManaCost{Blue: 1}, core.ManaCost{Generic: 0, Blue: 1}},
		{core.ManaCost{Generic: 1, Blue: 1}, core.ManaCost{Generic: 0, Blue: 1}},
	} {
		g, a, _ := newActivationDivisionGame()
		host := g.PutOnBattlefield(NewArtifact("Host", ""), a.PlayerID())
		g.PutOnBattlefield(NewEnchantment("Tax", "", WithStaticAbility(IncreaseActivationCostStatic(IsArtifact, core.ManaCost{Generic: 3}))), a.PlayerID())
		aura := g.PutOnBattlefield(NewAura("Reduction", "", WithCastTarget(TargetArtifact()), WithStaticAbility(ReduceActivationCostStatic(AttachedPermanent, 2, 1))), a.PlayerID())
		g.Attach(aura.ID(), host.ID())
		second := g.PutOnBattlefield(NewAura("Reduction 2", "", WithCastTarget(TargetArtifact()), WithStaticAbility(ReduceActivationCostStatic(AttachedPermanent, 2, 1))), a.PlayerID())
		g.Attach(second.ID(), host.ID())
		got := g.modifyActivationManaCost(host.ID(), tc.base)
		if got.String() != tc.want.String() {
			t.Fatalf("base %v: got %v want %v", tc.base, got, tc.want)
		}
	}
}

func TestStaticActivationCostLifecycle(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	host := g.PutOnBattlefield(NewArtifact("Host", ""), a.PlayerID())
	card := NewEnchantment("Tax", "", WithStaticAbility(IncreaseActivationCostStatic(IsArtifact, core.ManaCost{Generic: 3})))
	source := g.PutOnBattlefield(card, b.PlayerID())
	check := func(game *Game, want int) {
		t.Helper()
		if got := game.modifyActivationManaCost(host.ID(), core.ManaCost{}).Generic; got != want {
			t.Fatalf("got %d want %d", got, want)
		}
	}
	check(g, 3)
	saved := g.effects.Rules.activationCostModifiers[0]
	clone := g.Clone()
	ids := g.PhaseOut(source.ID())
	check(g, 0)
	check(clone, 3)
	g.PhaseIn(ids)
	check(g, 3)
	remover := g.PutOnBattlefield(NewArtifact("Remover", ""), a.PlayerID())
	removal := RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
	removal.SetSourceID(remover.ID())
	g.AddContinuousEffect(removal)
	check(g, 0)
	g.RemoveFromBattlefield(remover)
	check(g, 3)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	check(g, 0)
	g.PutOnBattlefield(card, a.PlayerID())
	check(g, 3)
	if saved.applies(g, host.ID()) {
		t.Fatal("returned source restored old rule")
	}
	clone.effects.Rules.activationCostModifiers = nil
	check(clone, 0)
	check(g, 3)
}

func TestStaticActivationCostPayment(t *testing.T) {
	for _, mana := range []bool{false, true} {
		g, a, _ := newActivationDivisionGame()
		var options []CardOption
		if mana {
			options = []CardOption{WithManaAbility(core.Blue)}
		} else {
			options = []CardOption{WithActivatedAbility(GainLifeFor(You, Fixed(1)), Tap())}
		}
		host := g.PutOnBattlefield(NewArtifact("Host", "", options...), a.PlayerID())
		g.PutOnBattlefield(NewEnchantment("Tax", "", WithStaticAbility(IncreaseActivationCostStatic(IsArtifact, core.ManaCost{Generic: 3}))), a.PlayerID())
		a.ManaPool().Add(core.Colorless, 2)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), host.ID(), 0, nil); err == nil {
			t.Fatal("accepted underpayment")
		}
		if host.Tapped || a.ManaPool().TotalMana() != 2 || g.StackSize() != 0 {
			t.Fatal("failed cost changed resources")
		}
		a.ManaPool().Add(core.Colorless, 1)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), host.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		if !g.FindPermanent(host.ID()).Tapped {
			t.Fatal("tap not paid")
		}
		g.ResolveStack()
		if mana {
			if a.ManaPool().TotalMana() != 1 {
				t.Fatal("wrong mana payment")
			}
		} else if a.Life() != 21 || a.ManaPool().TotalMana() != 0 {
			t.Fatal("wrong ordinary payment")
		}
	}
}

func TestStaticActivationCostDeclarations(t *testing.T) {
	for _, e := range []ContinuousEffect{
		IncreaseActivationCostStatic(PermanentFilter{}, core.ManaCost{Generic: 1}),
		IncreaseActivationCostStatic(IsArtifact, core.ManaCost{HasX: true}),
		IncreaseActivationCostStatic(IsArtifact, core.ManaCost{Generic: -1}),
		ReduceActivationCostStatic(nil, 2, 1),
		ReduceActivationCostStatic(This, -1, 1),
		ReduceActivationCostStatic(This, 2, -1),
	} {
		if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithStaticAbility(e))); err == nil {
			t.Fatal("accepted invalid modifier")
		}
	}
}

func TestStaticActivationCostSplitAndX(t *testing.T) {
	for _, x := range []int{0, 2} {
		g, a, _ := newActivationDivisionGame()
		host := g.PutOnBattlefield(NewArtifact("Host", "", WithAbility(Activate(Costs(ManaCostOf("{X}{U}"), GenericCost(2), Tap()), GainLifeFor(You, Fixed(1))))), a.PlayerID())
		aura := g.PutOnBattlefield(NewAura("Reduce", "", WithCastTarget(TargetArtifact()), WithStaticAbility(ReduceActivationCostStatic(AttachedPermanent, 2, 1))), a.PlayerID())
		g.Attach(aura.ID(), host.ID())
		g.resolution.SetX(x)
		a.ManaPool().Add(core.Blue, 1)
		a.ManaPool().Add(core.Colorless, x)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), host.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		if a.ManaPool().TotalMana() != 0 || g.StackPeek().XValue != x {
			t.Fatal("total reduction or X")
		}
		if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
			t.Fatal("copy required payment")
		}
		g.ResolveStack()
		if a.Life() != 22 {
			t.Fatal("copied activation")
		}
	}
}

func TestStaticActivationCostHostIdentity(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	hostCard := NewArtifact("Host", "")
	host := g.PutOnBattlefield(hostCard, b.PlayerID())
	other := g.PutOnBattlefield(NewArtifact("Other", ""), a.PlayerID())
	aura := g.PutOnBattlefield(NewAura("Reduce", "", WithCastTarget(TargetArtifact()), WithStaticAbility(ReduceActivationCostStatic(AttachedPermanent, 2, 1))), a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	check := func(id uuid.UUID, want int) {
		t.Helper()
		if got := g.modifyActivationManaCost(id, core.ManaCost{Generic: 3}).Generic; got != want {
			t.Fatalf("cost %d want %d", got, want)
		}
	}
	check(host.ID(), 1)
	check(other.ID(), 3)
	g.Attach(aura.ID(), other.ID())
	check(host.ID(), 3)
	check(other.ID(), 1)
	ids := g.PhaseOut(other.ID())
	check(other.ID(), 3)
	g.PhaseIn(ids)
	check(other.ID(), 1)
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: other.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	check(other.ID(), 1)
	card := other.Card
	g.RemoveFromBattlefield(g.FindPermanent(other.ID()))
	g.PutOnBattlefield(card, a.PlayerID())
	check(other.ID(), 3)
}

func TestStaticActivationCostManaPlanning(t *testing.T) {
	for _, direct := range []bool{false, true} {
		g, a, _ := newActivationDivisionGame()
		sourceCard := NewArtifact("Mana", "", WithAbility(Activate(Costs(GenericCost(2), GenericCost(2), Tap()), AddMana(core.Blue, 1))))
		if direct {
			sourceCard = NewArtifact("Mana", "", WithManaAbility(core.Blue))
		}
		host := g.PutOnBattlefield(sourceCard, a.PlayerID())
		if direct {
			g.PutOnBattlefield(NewEnchantment("Tax", "", WithStaticAbility(IncreaseActivationCostStatic(IsArtifact, core.ManaCost{Generic: 2}))), a.PlayerID())
		} else {
			aura := g.PutOnBattlefield(NewAura("Reduce", "", WithCastTarget(TargetArtifact()), WithStaticAbility(ReduceActivationCostStatic(AttachedPermanent, 2, 1))), a.PlayerID())
			g.Attach(aura.ID(), host.ID())
		}
		a.ManaPool().Add(core.Colorless, 1)
		if g.CanAfford(a.PlayerID(), core.ManaCost{Blue: 1}, nil) {
			t.Fatal("planned unpaid mana cost")
		}
		if g.FindPermanent(host.ID()).Tapped || a.ManaPool().TotalMana() != 1 {
			t.Fatal("query changed state")
		}
		a.ManaPool().Add(core.Colorless, 1)
		spell := NewInstant("Probe", "{U}", Spell(GainLifeFor(You, Fixed(1))))
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		if !g.FindPermanent(host.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
			t.Fatal("plan and payment differ")
		}
		g.ResolveStack()
		if a.Life() != 21 {
			t.Fatal("spell")
		}
	}
}

func TestStaticActivationCostColoredAndLocked(t *testing.T) {
	g, a, _ := newActivationDivisionGame()
	host := g.PutOnBattlefield(NewArtifact("Host", "", WithAbility(Activate(Costs(GenericCost(2), GenericCost(2), Tap()), GainLifeFor(You, Fixed(1))))), a.PlayerID())
	tax := g.PutOnBattlefield(NewEnchantment("Tax", "", WithStaticAbility(IncreaseActivationCostStatic(IsArtifact, core.ManaCost{Generic: 1, Blue: 1}))), a.PlayerID())
	aura := g.PutOnBattlefield(NewAura("Reduce", "", WithCastTarget(TargetArtifact()), WithStaticAbility(ReduceActivationCostStatic(AttachedPermanent, 2, 1))), a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	a.ManaPool().Add(core.Colorless, 4)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), host.ID(), 0, nil); err == nil {
		t.Fatal("generic paid colored increase")
	}
	if a.ManaPool().TotalMana() != 4 || g.FindPermanent(host.ID()).Tapped {
		t.Fatal("rollback")
	}
	a.ManaPool().Clear()
	a.ManaPool().Add(core.Colorless, 3)
	a.ManaPool().Add(core.Blue, 1)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: host.ID(), Costs: []Cost{GenericCost(2), GenericCost(2), Tap()}, ApplyActivationReduction: true})
	if err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(tax)
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().TotalMana() != 0 || !g.FindPermanent(host.ID()).Tapped {
		t.Fatal("locked total changed")
	}
}
