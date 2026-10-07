package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCounterCostRealCards(t *testing.T) {
	for _, name := range []string{"Feral Thallid", "Thorn Thallid"} {
		for _, change := range []string{"none", "return", "phase", "phase-cycle", "control"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				c := repositoryCard(t, "FEM", name)
				if err := mage.ValidateCardDefinition(name, c); err != nil {
					t.Fatal(err)
				}
				c.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(c, a.PlayerID())
				g.SetStep(core.Upkeep)
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
				g.PutTriggersOnStack()
				if g.StackSize() != 0 {
					t.Fatal("opponent upkeep triggered placement")
				}
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
				g.PutTriggersOnStack()
				if g.StackSize() != 1 || p.Counters[core.Spore] != 0 {
					t.Fatal("upkeep placement did not wait for resolution")
				}
				g.ResolveStack()
				if g.FindPermanent(p.ID()).Counters[core.Spore] != 1 {
					t.Fatal("upkeep did not place one counter")
				}
				var targets []uuid.UUID
				if name == "Thorn Thallid" {
					targets = []uuid.UUID{b.PlayerID()}
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, targets); err == nil {
					t.Fatal("insufficient counters accepted")
				}
				if g.FindPermanent(p.ID()).Counters[core.Spore] != 1 || g.StackSize() != 0 {
					t.Fatal("failed payment changed state")
				}
				g.MutablePermanent(p.ID()).AddCounter(core.Spore, 5)
				g.SetStep(core.PrecombatMain)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, targets); err != nil {
					t.Fatal(err)
				}
				if g.FindPermanent(p.ID()).Counters[core.Spore] != 3 || b.Life() != 20 || g.HasRegenerationShield(p.ID()) {
					t.Fatal("payment and resolution were not separate")
				}
				switch change {
				case "return":
					g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
					g.PutOnBattlefield(c, a.PlayerID())
				case "phase":
					g.PhaseOut(p.ID())
				case "phase-cycle":
					g.PhaseOut(p.ID())
					g.PhaseIn([]uuid.UUID{p.ID()})
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				clone := g.Clone()
				g.ResolveStack()
				if clone.StackSize() != 1 || clone.HasRegenerationShield(p.ID()) || clone.GetPlayer(b.PlayerID()).Life() != 20 {
					t.Fatal("resolution changed pending clone")
				}
				clone.ResolveStack()
				for _, game := range []*mage.Game{g, clone} {
					if name == "Thorn Thallid" {
						if game.GetPlayer(b.PlayerID()).Life() != 19 {
							t.Fatal("damage lost departed source or controller")
						}
					} else {
						want := change != "return" && change != "phase"
						if game.HasRegenerationShield(p.ID()) != want {
							t.Fatal("regeneration followed wrong incarnation")
						}
						if want {
							game.DestroyPermanent(game.FindPermanent(p.ID()))
							if game.FindPermanent(p.ID()) == nil || !game.FindPermanent(p.ID()).Tapped {
								t.Fatal("regeneration failed to replace destruction")
							}
						}
					}
					wantCounters := uint8(3)
					if change == "return" {
						wantCounters = 0
					}
					if game.FindPermanentIncludingPhased(p.ID()).Counters[core.Spore] != wantCounters {
						t.Fatal("resolution paid counters again")
					}
				}
			})
		}
	}
}

func TestRONCounterCostRollbackAndComposition(t *testing.T) {
	remove := `RemoveCounters([NumberCountersOfTypeFromPermanent(Integer(3), ChargeCounter, ThisPermanent)])`
	for _, cost := range []string{
		`And([PayMana([ManaCostG]), TapPermanent(ThisPermanent), ` + remove + `])`,
		`And([` + remove + `, TapPermanent(ThisPermanent), PayMana([ManaCostG])])`,
		`And([` + remove + `, ` + remove + `])`,
	} {
		t.Run(cost, func(t *testing.T) {
			g, a, _ := ronGame()
			c := permanentScopeFactory(t, "Artifact", `Activated(`+cost+`, ActionList([GainLife(Integer(1))]))`)()
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.MutablePermanent(p.ID()).AddCounter(core.Charge, 5)
			a.ManaPool().Add(core.Green, 1)
			if strings.Count(cost, "RemoveCounters") == 1 {
				g.MutablePermanent(p.ID()).RemoveCounter(core.Charge, 3)
			}
			before := g.FindPermanent(p.ID()).Counters[core.Charge]
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
				t.Fatal("unpayable total cost accepted")
			}
			if g.FindPermanent(p.ID()).Counters[core.Charge] != before || g.FindPermanent(p.ID()).Tapped || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
				t.Fatal("failed payment did not roll back")
			}
			g.MutablePermanent(p.ID()).AddCounter(core.Charge, 4)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			want := before + 4 - 3
			if strings.Count(cost, "RemoveCounters") == 2 {
				want -= 3
			}
			if g.FindPermanent(p.ID()).Counters[core.Charge] != want || a.Life() != 20 {
				t.Fatal("wrong counter payment")
			}
			g.ResolveStack()
			if a.Life() != 21 {
				t.Fatal("effect did not resolve")
			}
		})
	}
}

func TestRONCounterCostFactoryIsolationAndManaRollback(t *testing.T) {
	g, a, _ := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `Activated(And([RemoveCounters([ACounterOfTypeFromPermanent(ChargeCounter, ThisPermanent)]), TapPermanent(ThisPermanent), PayMana([ManaCostG])]), ActionList([GainLife(Integer(1))]))`)
	first, second := factory(), factory()
	first.SetOwner(a.PlayerID())
	second.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(first, a.PlayerID())
	q := g.PutOnBattlefield(second, a.PlayerID())
	g.MutablePermanent(p.ID()).AddCounter(core.Charge, 1)
	g.MutablePermanent(q.ID()).AddCounter(core.Charge, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
		t.Fatal("unpaid mana accepted")
	}
	if g.FindPermanent(p.ID()).Counters[core.Charge] != 1 || g.FindPermanent(p.ID()).Tapped || g.StackSize() != 0 {
		t.Fatal("mana failure consumed earlier costs")
	}
	a.ManaPool().Add(core.Green, 1)
	clone := g.Clone()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if clone.FindPermanent(p.ID()).Counters[core.Charge] != 1 || clone.FindPermanent(p.ID()).Tapped || clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 1 {
		t.Fatal("payment changed clone")
	}
	if err := clone.ActivateAbilityByIndex(a.PlayerID(), q.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(q.ID()).Counters[core.Charge] != 1 || clone.FindPermanent(p.ID()).Counters[core.Charge] != 1 {
		t.Fatal("factory instances share payment state")
	}
	g.ResolveStack()
	clone.ResolveStack()
	if a.Life() != 21 || clone.GetPlayer(a.PlayerID()).Life() != 21 {
		t.Fatal("independent ability did not resolve")
	}
}

func TestRONCounterCostTargetsCopiesAndInstances(t *testing.T) {
	for _, change := range []string{"none", "return", "phase", "noncreature", "prevention"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			first := repositoryCard(t, "FEM", "Thorn Thallid")
			first.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(first, a.PlayerID())
			second := repositoryCard(t, "FEM", "Thorn Thallid")
			second.SetOwner(a.PlayerID())
			q := g.PutOnBattlefield(second, a.PlayerID())
			g.MutablePermanent(p.ID()).AddCounter(core.Spore, 3)
			g.MutablePermanent(q.ID()).AddCounter(core.Spore, 3)
			victim := repositoryCard(t, "LEB", "Air Elemental")
			victim.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(victim, b.PlayerID())
			if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 1, []uuid.UUID{target.ID()}); err == nil {
				t.Fatal("opponent used source")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{uuid.New()}); err == nil {
				t.Fatal("invalid target accepted")
			}
			g.PhaseOut(p.ID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{target.ID()}); err == nil {
				t.Fatal("phased source paid")
			}
			g.PhaseIn([]uuid.UUID{p.ID()})
			if g.FindPermanent(p.ID()).Counters[core.Spore] != 3 {
				t.Fatal("rejection consumed counters")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			if g.FindPermanent(p.ID()).Counters[core.Spore] != 0 || g.FindPermanent(q.ID()).Counters[core.Spore] != 3 {
				t.Fatal("copy or independent instance paid wrong source")
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(target)
				g.PutOnBattlefield(victim, b.PlayerID())
			case "phase":
				g.PhaseOut(target.ID())
			case "noncreature":
				g.AddContinuousEffect(mage.FuncContinuousEffect(core.LayerType, core.EndOfTurn, func(game *mage.Game, _ uuid.UUID) error {
					game.GetEffects().RevokeAttr(target.ID(), core.AttrIsCreature)
					return nil
				}))
				g.ApplyContinuousEffects()
			case "prevention":
				g.AddPreventionShield(target.ID(), 1)
			}
			g.DestroyPermanent(g.FindPermanent(p.ID()))
			clone := g.Clone()
			g.ResolveStack()
			if clone.FindPermanentIncludingPhased(target.ID()).Damage != 0 {
				t.Fatal("damage crossed clone boundary")
			}
			clone.ResolveStack()
			want := 2
			if change == "prevention" {
				want = 1
			}
			if change == "return" || change == "phase" || change == "noncreature" {
				want = 0
			}
			for _, game := range []*mage.Game{g, clone} {
				if game.FindPermanentIncludingPhased(target.ID()).Damage != want {
					t.Fatal("wrong damage after legality/prevention")
				}
			}
		})
	}
}

func TestRONCounterCostFormsAndTiming(t *testing.T) {
	for _, instruction := range []string{
		`ACounterOfTypeFromPermanent(ChargeCounter, ThisPermanent)`,
		`NumberCountersOfTypeFromPermanent(Integer(0), ChargeCounter, ThisPermanent)`,
		`NumberCountersOfTypeFromPermanent(Integer(2), PTCounter(1, 1), ThisPermanent)`,
	} {
		t.Run(instruction, func(t *testing.T) {
			g, a, _ := ronGame()
			factory := permanentScopeFactory(t, "Artifact", `ActivatedWithModifiers(RemoveCounters([`+instruction+`]), ActionList([GainLife(Integer(1))]), And([ActivateOnlyIf(PlayerPassesFilter(You, IsTheirUpkeep)), ActivateOnlyOnceEachTurn]))`)
			c := factory()
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.MutablePermanent(p.ID()).AddCounter(core.Charge, 3)
			g.MutablePermanent(p.ID()).AddCounter(core.P1P1, 3)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
				t.Fatal("wrong timing accepted")
			}
			g.SetStep(core.Upkeep)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
				t.Fatal("second activation accepted")
			}
			wantCharge, wantPT := uint8(3), uint8(3)
			if strings.HasPrefix(instruction, "ACounter") {
				wantCharge--
			}
			if strings.Contains(instruction, "PTCounter") {
				wantPT -= 2
			}
			if g.FindPermanent(p.ID()).Counters[core.Charge] != wantCharge || g.FindPermanent(p.ID()).Counters[core.P1P1] != wantPT {
				t.Fatal("wrong fixed amount or counter kind")
			}
			g.SetStep(core.Draw)
			g.ResolveStack()
			if a.Life() != 21 {
				t.Fatal("timing was checked again at resolution")
			}
		})
	}
}

func TestRONCounterCostInvalid(t *testing.T) {
	for _, cost := range []string{
		`RemoveCounters`, `RemoveCounters()`, `RemoveCounters(ChargeCounter)`, `RemoveCounters([])`,
		`RemoveCounters([ACounterOfTypeFromPermanent(ChargeCounter, ThisPermanent), ACounterOfTypeFromPermanent(ChargeCounter, ThisPermanent)])`,
		`RemoveCounters([ACounterOfTypeFromPermanent])`, `RemoveCounters([ACounterOfTypeFromPermanent(ChargeCounter)])`,
		`RemoveCounters([ACounterOfTypeFromPermanent(UnknownCounter, ThisPermanent)])`, `RemoveCounters([ACounterOfTypeFromPermanent(ChargeCounter(), ThisPermanent)])`,
		`RemoveCounters([ACounterOfTypeFromPermanent(PTCounter(1), ThisPermanent)])`,
		`RemoveCounters([ACounterOfTypeFromPermanent(ChargeCounter, ThisPermanent())])`,
		`RemoveCounters([ACounterOfTypeFromPermanent(ChargeCounter, HostPermanent)])`,
		`RemoveCounters([ACounterOfTypeFromPermanent(ChargeCounter, Ref_TargetPermanent)])`,
		`RemoveCounters([NumberCountersOfTypeFromPermanent(Integer(-1), ChargeCounter, ThisPermanent)])`,
		`RemoveCounters([NumberCountersOfTypeFromPermanent(Integer("3"), ChargeCounter, ThisPermanent)])`,
		`RemoveCounters([NumberCountersOfTypeFromPermanent(ValueX, ChargeCounter, ThisPermanent)])`,
		`RemoveCounters([AnyNumberOfCountersOfTypeFromPermanent(ChargeCounter, HostPermanent)])`,
	} {
		t.Run(cost, func(t *testing.T) {
			nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Activated(%s, ActionList([GainLife(Integer(1))]))])`, cost)), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatal("invalid counter cost accepted")
			}
		})
	}
	remove := `RemoveCounters([ACounterOfTypeFromPermanent(ChargeCounter, HostPermanent)])`
	for _, rule := range []string{
		`Activated(PayMana([]), ActionList([RemoveCounters([NumberCountersOfTypeFromPermanent(Integer(1), ChargeCounter, ThisPermanent)])]))`,
		`TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)), ActionList([MayCost(` + remove + `), If(CostWasPaid, [GainLife(Integer(1))])]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name: "Invalid context", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [`+rule+`])`), "context.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("unsupported removal context accepted")
		}
	}
}

func TestRONCounterCostInventory(t *testing.T) {
	src := `Card(Name: "Inventory", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Activated(RemoveCounters([NumberCountersOfTypeFromPermanent(Integer(3), ChargeCounter, ThisPermanent)]), ActionList([GainLife(Integer(1))]))])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]string{"NumberCountersOfTypeFromPermanent": "counter_removal_instruction", "ChargeCounter": "permanent_counter_type", "ThisPermanent": "permanent_reference", "Integer": "value"}
	for name, role := range roles {
		found := false
		for _, s := range report.Signatures {
			if s.Name == name && s.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s role for %s", role, name)
		}
	}
}
