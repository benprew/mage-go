package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

const counterStateRule = `TriggerA(WhenAPermanentHasNumberCountersOfType(SinglePermanent(ThisPermanent),GreaterThanOrEqualTo(Integer(4)),TideCounter),ActionList([RemoveCounters([AllCountersOfTypeFromPermanent(TideCounter,Trigger_ThatPermanent)])]))`

func TestRONCounterStateHomarid(t *testing.T) {
	a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	c := repositoryCard(t, "FEM", "Homarid")
	if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast accepted")
	}
	if len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("unpaid cast changed state")
	}
	a.ManaPool().Add(core.Blue, 3)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(c.ID()) != nil {
		t.Fatal("entry before resolution")
	}
	g.ResolveStack()
	check := func(wantCounters int, wantPT int) {
		t.Helper()
		g.ApplyContinuousEffects()
		p := g.FindPermanent(c.ID())
		if p.Counters[core.Tide] != wantCounters || p.CurrentPower(g) != wantPT || p.CurrentToughness(g) != wantPT {
			t.Fatalf("counters=%d PT=%d/%d, want %d and %d/%d", p.Counters[core.Tide], p.CurrentPower(g), p.CurrentToughness(g), wantCounters, wantPT, wantPT)
		}
	}
	check(1, 1)
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(core.Upkeep)
	check(1, 1)
	g.SetActivePlayerIndex(0)
	for _, want := range []struct {
		n  int
		pt int
	}{{2, 2}, {3, 3}, {0, 2}, {1, 1}, {2, 2}, {3, 3}, {0, 2}} {
		g.RunStepWithPriority(core.Upkeep)
		check(want.n, want.pt)
	}
}

func TestRONCounterStateInstances(t *testing.T) {
	for _, scenario := range []string{"resolve", "counter", "lower-count", "more-counters", "control", "returned", "absent", "phase", "phase-cycle", "copy", "clone", "pending-clone", "two"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "FEM", "Homarid")
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.MutablePermanent(p.ID()).Counters[core.Tide] = 5
			g.MutablePermanent(p.ID()).AddCounter(core.Charge, 2)
			g.CheckStateTriggers()
			g.CheckStateTriggers()
			if scenario == "pending-clone" {
				clone := g.Clone()
				clone.PutTriggersOnStack()
				clone.ResolveStack()
				if clone.FindPermanent(p.ID()).Counters[core.Tide] != 0 || g.FindPermanent(p.ID()).Counters[core.Tide] != 5 {
					t.Fatal("pending clone changed original")
				}
			}
			g.PutTriggersOnStack()
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || g.FindPermanent(p.ID()).Counters[core.Tide] != 5 {
				t.Fatal("arming or timing")
			}
			switch scenario {
			case "counter":
				g.CounterStackObject(g.StackPeek().ID)
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
				if g.StackSize() != 1 {
					t.Fatal("countered state trigger did not rearm")
				}
			case "lower-count":
				g.MutablePermanent(p.ID()).Counters[core.Tide] = 2
			case "more-counters":
				g.MutablePermanent(p.ID()).Counters[core.Tide] = 9
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				g.ApplyContinuousEffects()
			case "returned", "absent":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				if scenario == "returned" {
					a.RemoveFromHand(c.ID())
					g.PutOnBattlefield(c, a.PlayerID())
					g.MutablePermanent(p.ID()).Counters[core.Tide] = 3
				}
				g.ResolveStack()
				if scenario == "returned" && g.FindPermanent(p.ID()).Counters[core.Tide] != 3 {
					t.Fatal("old trigger reset returned source")
				}
				return
			case "phase":
				g.PhaseOut(p.ID())
				g.ResolveStack()
				if g.FindPermanentIncludingPhased(p.ID()).Counters[core.Tide] != 5 {
					t.Fatal("reset phased source")
				}
				return
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				g.ResolveTopOfStack()
				if g.FindPermanent(p.ID()).Counters[core.Tide] != 0 {
					t.Fatal("copy lost original source")
				}
				g.MutablePermanent(p.ID()).Counters[core.Tide] = 3
			case "clone":
				clone := g.Clone()
				clone.ResolveStack()
				if clone.FindPermanent(p.ID()).Counters[core.Tide] != 0 || g.FindPermanent(p.ID()).Counters[core.Tide] != 5 {
					t.Fatal("stack clone changed original")
				}
			case "two":
				other := g.PutOnBattlefield(repositoryCard(t, "FEM", "Homarid"), b.PlayerID())
				g.MutablePermanent(other.ID()).Counters[core.Tide] = 7
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
				if g.StackSize() != 2 {
					t.Fatal("instances share arming")
				}
				g.ResolveTopOfStack()
				if g.FindPermanent(other.ID()).Counters[core.Tide] != 0 || g.FindPermanent(p.ID()).Counters[core.Tide] != 5 {
					t.Fatal("instances share source")
				}
			}
			g.ResolveStack()
			if g.FindPermanent(p.ID()).Counters[core.Tide] != 0 || g.FindPermanent(p.ID()).Counters[core.Charge] != 2 {
				t.Fatal("reset wrong counters")
			}
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("reset state retriggered")
			}
			g.MutablePermanent(p.ID()).Counters[core.Tide] = 4
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("resolved state trigger did not rearm")
			}
		})
	}
}

func TestRONCounterStateInactive(t *testing.T) {
	for _, scenario := range []string{"phase", "abilities"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, _ := ronGame()
			p := g.PutOnBattlefield(repositoryCard(t, "FEM", "Homarid"), a.PlayerID())
			if scenario == "phase" {
				g.MutablePermanent(p.ID()).Counters[core.Tide] = 5
				g.PhaseOut(p.ID())
			} else {
				donor := g.PutOnBattlefield(mage.NewArtifact("Remover", ""), a.PlayerID())
				e := mage.RemoveAllAbilities(p.ID(), core.WhileOnBattlefield)
				e.SetSourceID(donor.ID())
				g.AddContinuousEffect(e)
				g.ApplyContinuousEffects()
				g.MutablePermanent(p.ID()).Counters[core.Tide] = 5
			}
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("inactive source triggered")
			}
		})
	}
}

func TestRONCounterStateTidalClause(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/FEM.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "FEM.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range nodes {
		if n.Fields["Name"].String != "Tidal Influence" {
			continue
		}
		if _, err := Compile(n); err == nil || !strings.Contains(err.Error(), "casting window") {
			t.Fatalf("full Tidal Influence must reject casting restriction: %v", err)
		}
		for _, rule := range n.Fields["Rules"].Args {
			if rule.Name != "TriggerA" || rule.Args[0].Name != "WhenAPermanentHasNumberCountersOfType" {
				continue
			}
			option, err := compileTriggerRule(rule, scope{permanent: true})
			if err != nil {
				t.Fatal(err)
			}
			g, a, _ := ronGame()
			p := g.PutOnBattlefield(mage.NewEnchantment("Clause", "", option()), a.PlayerID())
			g.MutablePermanent(p.ID()).Counters[core.Tide] = 8
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("missing reset")
			}
			g.ResolveStack()
			if g.FindPermanent(p.ID()).Counters[core.Tide] != 0 {
				t.Fatal("clause reset")
			}
			return
		}
	}
	t.Fatal("Tidal Influence clause missing")
}

func TestRONCounterStateDeclarations(t *testing.T) {
	for _, rule := range []string{
		strings.Replace(counterStateRule, "SinglePermanent(ThisPermanent)", "SinglePermanent(HostPermanent)", 1),
		strings.Replace(counterStateRule, "SinglePermanent(ThisPermanent)", "AnyPermanent", 1),
		strings.Replace(counterStateRule, "Integer(4)", "Integer(-1)", 1),
		strings.Replace(counterStateRule, "Integer(4)", "ValueX", 1),
		strings.Replace(counterStateRule, "GreaterThanOrEqualTo", "EqualTo", 1),
		strings.Replace(counterStateRule, "TideCounter)", "PoisonCounter)", 1),
		strings.Replace(counterStateRule, "TideCounter)", "TideCounter())", 1),
		strings.Replace(counterStateRule, "SinglePermanent(ThisPermanent)", "SinglePermanent(ThisPermanent,ThisPermanent)", 1),
		strings.Replace(counterStateRule, "Integer(4)", "Integer(4,5)", 1),
		`Activated(PayMana([]),ActionList([RemoveCounters([AllCountersOfTypeFromPermanent(TideCounter,Trigger_ThatPermanent)])]))`,
		`TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([RemoveCounters([AllCountersOfTypeFromPermanent(TideCounter,Trigger_ThatPermanent)])]))`,
		`Activated(PayMana([]),ActionList([RemoveCounters([AllCountersOfTypeFromPermanent(TideCounter)])]))`,
		`Activated(PayMana([]),ActionList([RemoveCounters([AllCountersOfTypeFromPermanent(PoisonCounter,ThisPermanent)])]))`,
		`Activated(PayMana([]),ActionList([RemoveCounters([AllCountersOfTypeFromPermanent(TideCounter,ThisPermanent),AllCountersOfTypeFromPermanent(ChargeCounter,ThisPermanent)])]))`,
		counterStateRule + `,Activated(PayMana([]),ActionList([TapPermanent(Trigger_ThatPermanent)]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Errorf("accepted %s", rule)
		}
	}
	for _, threshold := range []string{"0", "256"} {
		factory := permanentScopeFactory(t, "Artifact", strings.Replace(counterStateRule, "Integer(4)", "Integer("+threshold+")", 1))
		if err := mage.ValidateCardDefinition("valid", factory()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRONCounterStateResetTarget(t *testing.T) {
	for _, scenario := range []string{"present", "returned", "phase", "shroud"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			actor := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([RemoveCounters([AllCountersOfTypeFromPermanent(TideCounter,Ref_TargetPermanent)])])))`)(), a.PlayerID())
			c := mage.NewCreature("Target", "", 2, 3)
			c.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(c, b.PlayerID())
			g.MutablePermanent(p.ID()).Counters[core.Tide] = 7
			if err := g.ActivateAbilityByIndex(a.PlayerID(), actor.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "returned":
				g.BouncePermanentToHand(p)
				b.RemoveFromHand(c.ID())
				g.PutOnBattlefield(c, b.PlayerID())
				g.MutablePermanent(p.ID()).Counters[core.Tide] = 3
			case "phase":
				g.PhaseOut(p.ID())
			case "shroud":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.GrantKeywordModification(mage.This, core.Shroud)), p.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			g.ResolveStack()
			want := int(7)
			if scenario == "present" {
				want = 0
			}
			if scenario == "returned" {
				want = 3
			}
			if g.FindPermanentIncludingPhased(p.ID()).Counters[core.Tide] != want {
				t.Fatal("reset target identity or legality")
			}
		})
	}
}

func TestRONCounterStateInventory(t *testing.T) {
	src := `Card(Name:"Counter",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + counterStateRule + `])`
	report, err := InspectFS(fstest.MapFS{"counter.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 {
		t.Fatal(report.Issues)
	}
	for name, role := range map[string]string{"SinglePermanent": "permanent_filter", "GreaterThanOrEqualTo": "counter_state_comparison", "Integer": "counter_state_value", "TideCounter": "permanent_counter_type", "AllCountersOfTypeFromPermanent": "counter_removal_instruction", "Trigger_ThatPermanent": "permanent_reference"} {
		found := false
		for _, s := range report.Signatures {
			if s.Name == name && s.Role == role {
				found = true
			}
		}
		if !found {
			t.Errorf("%s missing role %s", name, role)
		}
	}
}

func TestRONCounterStatePermanentSpellCopy(t *testing.T) {
	g, a, b := ronGame()
	c := repositoryCard(t, "FEM", "Homarid")
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	a.ManaPool().Add(core.Blue, 3)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	spellCopy := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	if spellCopy == nil {
		t.Fatal("permanent spell copy failed")
	}
	id := spellCopy.Card.ID()
	g.ResolveStack()
	g.ApplyContinuousEffects()
	for _, source := range []uuid.UUID{c.ID(), id} {
		p := g.FindPermanent(source)
		if p == nil || p.Counters[core.Tide] != 1 || p.CurrentPower(g) != 1 || p.CurrentToughness(g) != 1 {
			t.Fatal("copied spell entry or static identity")
		}
	}
	g.MutablePermanent(id).Counters[core.Tide] = 4
	g.CheckStateTriggers()
	g.PutTriggersOnStack()
	g.ResolveStack()
	g.ApplyContinuousEffects()
	if g.FindPermanent(id).Counters[core.Tide] != 0 || g.FindPermanent(id).CurrentPower(g) != 2 || g.FindPermanent(c.ID()).Counters[core.Tide] != 1 || g.FindPermanent(c.ID()).CurrentPower(g) != 1 {
		t.Fatal("copied permanent reset changed original")
	}
}

func TestRONCounterStateEntryReplacements(t *testing.T) {
	g, a, _ := ronGame()
	donor := g.PutOnBattlefield(mage.NewArtifact("Donor", ""), a.PlayerID())
	g.AddETBAdditionalCounters(donor.ID(), core.Tide, 3, mage.IsCreature, false)
	p := g.PutOnBattlefield(repositoryCard(t, "FEM", "Homarid"), a.PlayerID())
	g.PutTriggersOnStack()
	if p.Counters[core.Tide] != 4 || g.StackSize() != 1 {
		t.Fatal("state trigger missed replaced entry")
	}
	g.ResolveStack()
	if g.FindPermanent(p.ID()).Counters[core.Tide] != 0 {
		t.Fatal("entry reset did not resolve")
	}
}

func TestRONCounterStateResetHost(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name:"Host reset",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),Activated(PayMana([]),ActionList([RemoveCounters([AllCountersOfTypeFromPermanent(TideCounter,HostPermanent)])]))])`), "host-reset.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"present", "reattach", "departed", "returned"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, _ := ronGame()
			host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 3), a.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 3), a.PlayerID())
			g.MutablePermanent(host.ID()).Counters[core.Tide] = 5
			g.MutablePermanent(other.ID()).Counters[core.Tide] = 7
			c := factory()
			c.SetOwner(a.PlayerID())
			aura := g.PutOnBattlefield(c, a.PlayerID())
			g.Attach(aura.ID(), host.ID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), aura.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "reattach":
				g.Attach(aura.ID(), other.ID())
			case "departed", "returned":
				g.BouncePermanentToHand(g.FindPermanent(aura.ID()))
				if scenario == "returned" {
					a.RemoveFromHand(c.ID())
					g.PutOnBattlefield(c, a.PlayerID())
					g.Attach(c.ID(), other.ID())
				}
			}
			g.ResolveStack()
			wantHost, wantOther := int(0), int(7)
			if scenario == "reattach" {
				wantHost, wantOther = 5, 0
			}
			if g.FindPermanent(host.ID()).Counters[core.Tide] != wantHost || g.FindPermanent(other.ID()).Counters[core.Tide] != wantOther {
				t.Fatal("host reset lost attachment identity")
			}
		})
	}
}
