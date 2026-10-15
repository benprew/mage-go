package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func installLifeMana(t *testing.T, g *Game, player uuid.UUID, cost int, color core.Color) uuid.UUID {
	t.Helper()
	if err := ApplyEffect(g, PlayerLifeManaThisTurn(You, cost, color), uuid.New(), player, nil); err != nil {
		t.Fatal(err)
	}
	offers := g.GetPlayerManaAbilities(player)
	if len(offers) == 0 {
		t.Fatal("no mana permission")
	}
	return offers[len(offers)-1].ID
}

func TestPlayerLifeManaActivation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.SetLife(2)
	id := installLifeMana(t, g, a.PlayerID(), 1, core.Colorless)
	clone := g.Clone()
	if err := g.ActivatePlayerManaAbility(b.PlayerID(), id); err == nil {
		t.Fatal("wrong player activated")
	}
	for range 2 {
		if err := g.ActivatePlayerManaAbility(a.PlayerID(), id); err != nil {
			t.Fatal(err)
		}
	}
	if a.Life() != 0 || a.ManaPool().Count(core.Colorless) != 2 || g.StackSize() != 0 {
		t.Fatal("activation did not pay life and add real mana immediately")
	}
	if err := g.ActivatePlayerManaAbility(a.PlayerID(), id); err == nil {
		t.Fatal("paid positive life at zero")
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 2 || len(clone.GetPlayerManaAbilities(a.PlayerID())) != 1 {
		t.Fatal("clone changed")
	}
	g.doCleanupActions()
	a.SetLife(5)
	if err := g.ActivatePlayerManaAbility(a.PlayerID(), id); err == nil {
		t.Fatal("expired permission activated")
	}
	if err := clone.ActivatePlayerManaAbility(a.PlayerID(), id); err != nil {
		t.Fatal(err)
	}
}

func TestPlayerLifeManaPayment(t *testing.T) {
	for _, tc := range []struct {
		name      string
		mana      core.ManaCost
		extraLife int
		want      bool
	}{
		{"exact", core.ManaCost{Generic: 3}, 0, true},
		{"too much", core.ManaCost{Generic: 4}, 0, false},
		{"colored", core.ManaCost{Green: 1}, 0, false},
		{"aggregate life", core.ManaCost{Generic: 3}, 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			a.SetLife(3)
			installLifeMana(t, g, a.PlayerID(), 1, core.Colorless)
			installLifeMana(t, g, a.PlayerID(), 1, core.Colorless)
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{&ManaCostPayment{MC: tc.mana}, LifePayCost(tc.extraLife)}})
			if a.Life() != 3 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("query changed resources")
			}
			if !tc.want {
				if err == nil {
					t.Fatal("unaffordable total accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if err = tx.Commit(); err != nil {
				t.Fatal(err)
			}
			if a.Life() != 0 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("wrong committed resources")
			}
		})
	}
}

func TestPlayerLifeManaDeclarations(t *testing.T) {
	for _, effect := range []ResolvingEffect{
		PlayerLifeManaThisTurn(nil, 1, core.Colorless),
		PlayerLifeManaThisTurn(You, 0, core.Colorless),
		PlayerLifeManaThisTurn(You, -1, core.Colorless),
		PlayerLifeManaThisTurn(You, 1, core.AnyColor),
		PlayerLifeManaThisTurn(FromPlayer(BindPlayer("missing")), 1, core.Colorless),
	} {
		c := NewInstant("Invalid", "", WithAbility(Spell(effect)))
		if err := ValidateCardDefinition(c.Name(), c); err == nil {
			t.Fatal("invalid permission accepted")
		}
	}
}

func TestPlayerLifeManaStalePayment(t *testing.T) {
	for _, extra := range []int{0, 1} {
		t.Run(fmt.Sprint(extra), func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			a.SetLife(3 + extra)
			installLifeMana(t, g, a.PlayerID(), 1, core.Colorless)
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{&ManaCostPayment{MC: core.ManaCost{Generic: 3}}, LifePayCost(extra)}})
			if err != nil {
				t.Fatal(err)
			}
			a.SetLife(2 + extra)
			if err := tx.Commit(); err == nil {
				t.Fatal("stale total life budget accepted")
			}
			if a.Life() != 2+extra || a.ManaPool().TotalMana() != 0 {
				t.Fatal("failed commit spent part of life budget")
			}
		})
	}
}

func TestPlayerLifeManaWindowsAndQueries(t *testing.T) {
	a, b := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: true}, NewBasePlayer("B")
	g := NewGame(a, b)
	a.SetLife(4)
	id := installLifeMana(t, g, a.PlayerID(), 1, core.Colorless)
	if !g.CanAfford(a.PlayerID(), core.ManaCost{Generic: 4}, nil) || g.CanAfford(a.PlayerID(), core.ManaCost{Generic: 5}, nil) {
		t.Fatal("wrong life affordability")
	}
	if g.MaxXValue(a.PlayerID(), core.ParseManaCost("{X}"), nil) != 4 || g.HypotheticalMana(a.PlayerID()) != 4 {
		t.Fatal("wrong available mana")
	}
	if a.Life() != 4 || a.ManaPool().TotalProducedThisTurn() != 0 {
		t.Fatal("query produced mana")
	}
	g.resolution.resolvingController = a.PlayerID()
	if err := g.ActivatePlayerManaAbility(a.PlayerID(), id); err == nil {
		t.Fatal("activation outside payment during resolution")
	}
	effect := IfManaPaid(You, core.ParseManaCost("{X}"), Fixed(3), GainLifeFor(You, Fixed(2)))
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 3 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("resolution payment did not use mana window")
	}
	g.resolution.resolvingController = uuid.Nil
	p := g.PutOnBattlefield(NewArtifact("Ability", "", WithAbility(Activate(Costs(&ManaCostPayment{MC: core.ManaCost{Generic: 2}}), GainLifeFor(You, Fixed(1))))), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 1 {
		t.Fatal("ability payment did not use mana permission")
	}
	g.ResolveStack()
	if a.Life() != 2 {
		t.Fatal("ability body failed")
	}
	if !g.executePriorityAction(0, PriorityAction{Type: PriorityActivatePlayerManaAbility, PlayerManaAbilityID: id}) {
		t.Fatal("priority action failed")
	}
	if a.Life() != 1 || a.ManaPool().Count(core.Colorless) != 1 {
		t.Fatal("priority action did not produce")
	}
}

func TestPlayerLifeManaSourceAndPlayer(t *testing.T) {
	for _, change := range []string{"leave", "return", "phase", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			effect := PlayerLifeManaThisTurn(ControllerOf(This), 2, core.Green)
			c := NewArtifact("Valid", "", WithAbility(Activate(Costs(), effect)))
			if err := ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			id := g.GetPlayerManaAbilities(a.PlayerID())[0].ID
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
			if err := g.ActivatePlayerManaAbility(a.PlayerID(), id); err != nil {
				t.Fatal(err)
			}
			if a.Life() != 18 || a.ManaPool().Count(core.Green) != 1 || b.Life() != 20 {
				t.Fatal("saved payer changed")
			}
		})
	}
}

func TestPlayerLifeManaPlannerResources(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	a.SetLife(4)
	installLifeMana(t, g, a.PlayerID(), 2, core.Green)
	installLifeMana(t, g, a.PlayerID(), 1, core.Colorless)
	if !g.CanAfford(a.PlayerID(), core.ManaCost{Green: 1, Generic: 2}, nil) || g.CanAfford(a.PlayerID(), core.ManaCost{Green: 1, Generic: 3}, nil) {
		t.Fatal("independent grants did not share life budget")
	}
	a.ManaPool().Add(core.Colorless, 1)
	land := g.PutOnBattlefield(NewLand("Forest", WithManaAbility(core.Green)), a.PlayerID())
	if err := g.AutoTapForCost(a.PlayerID(), core.ManaCost{Green: 1, Generic: 1}); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 4 || !land.Tapped {
		t.Fatal("planner spent life instead of available mana")
	}
	a.ManaPool().Clear()
	ctx := &core.SpellPaymentContext{XSpendingRestricted: true, RestrictedGeneric: 1, XSpendingColor: core.Black}
	if g.CanAfford(a.PlayerID(), core.ManaCost{Generic: 1}, ctx) {
		t.Fatal("colorless production paid restricted black X")
	}
}

func TestPlayerLifeManaWaitingAndReplacements(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	body := PlayerLifeManaThisTurn(ControllerOfWithLKI(This), 1, core.Colorless)
	if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(p.ID())
	for range 2 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || len(g.playerManaAbilities) != 0 || a.Life() != 20 {
			t.Fatal("waiting query executed installation")
		}
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
	g.RunStepWithPriority(core.EndStep)
	g.ResolveStack()
	offers := g.GetPlayerManaAbilities(a.PlayerID())
	if len(offers) != 1 {
		t.Fatal("delayed source LKI lost payer")
	}
	if err := ApplyEffect(g, ReplaceTappedManaTypeForPlayerThisTurn(You, AnyPermanent, core.Blue), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivatePlayerManaAbility(a.PlayerID(), offers[0].ID); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(core.Colorless) != 1 || a.ManaPool().Count(core.Blue) != 0 {
		t.Fatal("untapped player permission used tapped-permanent replacement")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("player permission produced tapped-for-mana trigger")
	}
}
