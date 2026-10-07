package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func conversionSource(g *Game, a Player, costs string, output Color, amount int) *Permanent {
	c := NewCreature("Conversion", "", 1, 3, WithAbility(Activate(Costs(ManaPayment(costs)), AddMana(output, amount))))
	c.SetOwner(a.PlayerID())
	return g.PutOnBattlefield(c, a.PlayerID())
}

func TestRepeatableManaPlanning(t *testing.T) {
	for _, state := range []string{"fresh", "tapped", "chain"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := conversionSource(g, a, "{1}", Black, 1)
			if state == "tapped" {
				g.TapPermanent(p)
			}
			if state == "chain" {
				conversionSource(g, a, "{B}", Blue, 1)
			}
			a.ManaPool().Add(Red, 3)
			want := Black
			if state == "chain" {
				want = Blue
			}
			cost := ManaCost{Black: 3}
			if state == "chain" {
				cost = ManaCost{Blue: 3}
			}
			if !g.CanAfford(a.PlayerID(), cost, nil) {
				t.Fatal("repeatable conversion was excluded")
			}
			plan, err := g.mana.PlanManaForCost(g, a.PlayerID(), cost, AutoTapHint{}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(plan.SourcesToTap) < 3 || a.ManaPool().Count(Red) != 3 || !p.HasAttr(AttrSummonSick) {
				t.Fatal("query changed state or omitted conversions")
			}
			clone := g.Clone()
			if err := clone.mana.ApplyManaSolution(clone, a.PlayerID(), plan); err != nil {
				t.Fatal(err)
			}
			cp := clone.GetPlayer(a.PlayerID())
			if cp.ManaPool().Count(want) != 3 || clone.StackSize() != 0 || clone.FindPermanent(p.ID()).Tapped != (state == "tapped") {
				t.Fatal("conversion tapped its source or lost output")
			}
			if a.ManaPool().Count(Red) != 3 {
				t.Fatal("clone changed live payment")
			}
		})
	}
}

func TestRepeatableManaExactPayment(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	conversionSource(g, a, "{1}", Blue, 1)
	a.ManaPool().Add(White, 1)
	a.ManaPool().Add(Red, 2)
	cost := ManaCost{White: 1, Blue: 2}
	if err := g.AutoTapForCost(a.PlayerID(), cost); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(White) != 1 || a.ManaPool().Count(Blue) != 2 {
		t.Fatal("conversion spent the final white requirement")
	}
}

func TestRepeatableManaCycles(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	conversionSource(g, a, "{U}", Black, 1)
	conversionSource(g, a, "{B}", Blue, 1)
	conversionSource(g, a, "{G}", Red, 1)
	a.ManaPool().Add(Blue, 2)
	sources := g.mana.GetUntappedManaSources(g, a.PlayerID())
	if len(sources) != 3 {
		t.Fatal("missing conversion sources")
	}
	stats := &manaSearchStats{}
	if _, ok := searchManaWithStats(ManaSolverInputs{Pool: a.ManaPool(), Cost: ManaCost{Red: 1}, Sources: sources}, stats); ok || stats.ExpandedNodes <= 1 || stats.ExpandedNodes > 20 {
		t.Fatal("cycle did not terminate")
	}
	if g.CanAfford(a.PlayerID(), ManaCost{Black: 3}, nil) {
		t.Fatal("conversions manufactured mana")
	}
}

func TestRepeatableManaExcludedShapes(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePlayerTarget("target")
	for _, ability := range []*ActionDefinition{
		Activate(Costs(ManaPayment("{1}")), AddMana(Blue, 2)),
		Activate(Costs(ManaPayment("{X}")), AddMana(Blue, 1)),
		Activate(Costs(ManaPayment("{1}")), Sequence(AddMana(Blue, 1), GainLifeFor(You, Fixed(1)))),
		NewActivated(ManaPayment("{1}"), AddMana(Blue, 1), WithOncePerTurn()),
		Activate(Costs(ManaPayment("{1}")), AddMana(Blue, 1), WithTargets(target)),
		Activate(Costs(ManaPayment("{1}"), LifePayCost(1)), AddMana(Blue, 1)),
	} {
		c := NewArtifact("Excluded", "", WithAbility(ability))
		p := g.PutOnBattlefield(c, a.PlayerID())
		if _, ok := manaSourceAbilityForPlanning(p.RuntimeAbilities[0], p.ID(), 0, g); ok {
			t.Fatal("unsupported conversion entered planner")
		}
	}
	bad := NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(ManaPayment("{1}")), AddManaAmount(Blue, nil))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
		t.Fatal("invalid amount accepted")
	}
}

func TestRepeatableManaPlanLifecycle(t *testing.T) {
	for _, change := range []string{"returned", "phase", "phase-cycle", "control", "ability-removed", "payment-missing", "forbidden", "cost-reduced"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			cost := "{1}"
			if change == "cost-reduced" {
				cost = "{2}"
			}
			p := conversionSource(g, a, cost, Black, 1)
			a.ManaPool().Add(Red, 4)
			plan, err := g.mana.PlanManaForCost(g, a.PlayerID(), ManaCost{Black: 2}, AutoTapHint{}, nil)
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "returned":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, a.PlayerID())
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
				g.ApplyContinuousEffects()
			case "ability-removed":
				g.AddContinuousEffect(RemoveAllAbilities(p.ID(), Indefinite))
				g.ApplyContinuousEffects()
			case "payment-missing":
				a.ManaPool().Clear()
			case "forbidden":
				p.GrantBaseAttr(AttrCantActivate)
			case "cost-reduced":
				g.effects.Rules.ActivationCostReductions[p.ID()] = 1
			}
			before := a.ManaPool().TotalMana()
			err = g.mana.ActivatePlannedManaSource(g, a.PlayerID(), plan.SourcesToTap[0])
			if change == "phase-cycle" {
				if err != nil || a.ManaPool().Count(Black) != 1 {
					t.Fatal("phase cycle lost conversion")
				}
			} else if err == nil || a.ManaPool().TotalMana() != before || a.ManaPool().Count(Black) != 0 {
				t.Fatal("stale plan changed state")
			}
		})
	}
}

func TestRepeatableManaTapEventsAndBonuses(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	conversion := Activate(Costs(ManaPayment("{1}")), AddMana(Black, 1))
	p := g.PutOnBattlefield(NewCreature("Mixed source", "", 1, 3, WithKeyword(Haste), WithAbility(NewManaAbility(Red)), WithAbility(conversion)), a.PlayerID())
	g.PutOnBattlefield(NewEnchantment("Bonus", "", WithAbility(NewManaFlareAbility(AnyPermanent))), a.PlayerID())
	activated, tapped := 0, 0
	watcher := NewTriggered(EvtAbilityActivated, false, GainLifeFor(You, Fixed(1))).SetCondition(func(evt *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if evt.SourceID == p.ID() {
			activated++
		}
		return false
	})
	tapWatcher := NewTriggered(EvtTappedForMana, false, GainLifeFor(You, Fixed(1))).SetCondition(func(evt *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if evt.SourceID == p.ID() {
			tapped++
		}
		return false
	})
	g.PutOnBattlefield(NewEnchantment("Watcher", "", WithAbility(watcher), WithAbility(tapWatcher)), a.PlayerID())
	if err := g.AutoTapForCost(a.PlayerID(), ManaCost{Black: 2}); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(p.ID()).Tapped || a.ManaPool().Count(Black) != 2 || a.ManaPool().TotalMana() != 2 || activated != 3 || tapped != 1 {
		t.Fatalf("pool=%d activated=%d tapped=%d", a.ManaPool().TotalMana(), activated, tapped)
	}
	if g.HypotheticalMana(a.PlayerID()) != 2 {
		t.Fatal("conversion counted as extra mana")
	}
	if g.MaxXValue(a.PlayerID(), ParseManaCost("{X}"), nil) != 2 {
		t.Fatal("conversion inflated payable X")
	}
}

func TestRepeatableManaCostRollback(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := conversionSource(g, a, "{1}", Black, 1)
	land := g.PutOnBattlefield(NewLand("Mana", WithAbility(NewMultiManaAbility(ManaProduction{Color: Red, Amount: 2}))), a.PlayerID())
	spell := NewSorcery("Unpaid", "{B}{B}", WithAdditionalCost(LifePayCost(21)), WithAbility(Spell(GainLifeFor(You, Fixed(1)))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err == nil {
		t.Fatal("unpaid total cost accepted")
	}
	if g.FindPermanent(land.ID()).Tapped || g.FindPermanent(p.ID()).Tapped || a.ManaPool().TotalMana() != 0 || len(a.Hand()) != 1 || a.Life() != 20 || g.StackSize() != 0 {
		t.Fatal("failed transaction committed conversions")
	}
}

func TestRepeatableManaBoundAndPreference(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := conversionSource(g, a, "{1}", Black, 1)
	a.ManaPool().Add(Red, 5)
	tapID := uuid.New()
	sources := g.mana.GetUntappedManaSources(g, a.PlayerID())
	sources = append(sources, manaSourceInfo{PermanentID: tapID, Abilities: []manaSourceAbility{{Productions: []ManaProduction{{Color: Black, Amount: 5}}}}})
	plan, err := SolveMana(ManaSolverInputs{Pool: a.ManaPool(), Cost: ManaCost{Black: 5}, Sources: sources, Scores: []int{0, 1}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.SourcesToTap) != 5 {
		t.Fatal("path length overrode the lower preservation score")
	}
	for _, action := range plan.SourcesToTap {
		if action.PermanentID != p.ID() {
			t.Fatal("used costly tap source")
		}
	}
	unsafe := ManaSolverInputs{Sources: []manaSourceInfo{{Abilities: []manaSourceAbility{{Repeatable: true, Productions: []ManaProduction{{Color: Black, Amount: 1}}}}}}, Cost: ManaCost{Black: 2}}
	if CanSolveMana(unsafe) {
		t.Fatal("unbounded production entered repeated search")
	}
}
