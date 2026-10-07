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

const singleCounterAction = `RemoveCounters([ACounterOfTypeFromPermanent(ChargeCounter,ThisPermanent)])`

func TestRONSingleCounterRemovalVenarianGold(t *testing.T) {
	a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 3), b.PlayerID())
	c := repositoryCard(t, "LEG", "Venarian Gold")
	if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{host.ID()}, 3); err == nil {
		t.Fatal("unpaid cast accepted")
	}
	if len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("failed cast changed state")
	}
	a.ManaPool().Add(core.Blue, 5)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{host.ID()}, 3); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if !g.FindPermanent(host.ID()).Tapped || g.FindPermanent(host.ID()).Counters[core.Sleep] != 3 {
		t.Fatal("entry did not tap and save announced X")
	}
	g.SetActivePlayerIndex(0)
	g.RunStepWithPriority(core.Upkeep)
	if g.FindPermanent(host.ID()).Counters[core.Sleep] != 3 {
		t.Fatal("wrong player's upkeep removed counter")
	}
	g.SetActivePlayerIndex(1)
	for _, want := range []uint8{2, 1, 0, 0} {
		g.RunStepWithPriority(core.Untap)
		if want > 0 && !g.FindPermanent(host.ID()).Tapped {
			t.Fatal("host untapped with sleep counters")
		}
		g.RunStepWithPriority(core.Upkeep)
		if g.FindPermanent(host.ID()).Counters[core.Sleep] != want {
			t.Fatalf("sleep count, want %d", want)
		}
	}
	if g.FindPermanent(host.ID()).Tapped {
		t.Fatal("host did not untap after last sleep counter")
	}
}

func TestRONSingleCounterRemovalSource(t *testing.T) {
	for _, scenario := range []string{"normal", "zero", "one", "control", "returned", "absent", "phase", "phase-cycle", "clone", "copy", "two"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([`+singleCounterAction+`,GainLife(Integer(1))]))`)
			c := factory()
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			count := uint8(3)
			if scenario == "zero" {
				count = 0
			}
			if scenario == "one" {
				count = 1
			}
			g.MutablePermanent(p.ID()).Counters[core.Charge] = count
			g.MutablePermanent(p.ID()).Counters[core.Tide] = 2
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(p.ID()).Counters[core.Charge] != count {
				t.Fatal("removed during announcement")
			}
			want := count
			if want > 0 {
				want--
			}
			switch scenario {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				g.ApplyContinuousEffects()
			case "returned", "absent":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				if scenario == "returned" {
					a.RemoveFromHand(c.ID())
					g.PutOnBattlefield(c, a.PlayerID())
					g.MutablePermanent(p.ID()).Counters[core.Charge] = 4
					want = 4
				}
			case "phase":
				g.PhaseOut(p.ID())
				want = count
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "clone":
				clone := g.Clone()
				clone.ResolveStack()
				if clone.FindPermanent(p.ID()).Counters[core.Charge] != want || g.FindPermanent(p.ID()).Counters[core.Charge] != count {
					t.Fatal("clone isolation")
				}
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				g.ResolveTopOfStack()
				want--
				if b.Life() != 21 || a.Life() != 20 {
					t.Fatal("copy controller binding")
				}
			case "two":
				other := g.PutOnBattlefield(factory(), b.PlayerID())
				g.MutablePermanent(other.ID()).Counters[core.Charge] = 7
				if err := g.ActivateAbilityByIndex(b.PlayerID(), other.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveTopOfStack()
				if g.FindPermanent(other.ID()).Counters[core.Charge] != 6 || g.FindPermanent(p.ID()).Counters[core.Charge] != count {
					t.Fatal("factory bindings shared")
				}
			}
			g.ResolveStack()
			if scenario != "absent" && g.FindPermanentIncludingPhased(p.ID()).Counters[core.Charge] != want {
				t.Fatalf("counter identity, want %d", want)
			}
			if scenario != "returned" && scenario != "absent" && g.FindPermanentIncludingPhased(p.ID()).Counters[core.Tide] != 2 {
				t.Fatal("wrong counter kind removed")
			}
			if a.Life() != 21 {
				t.Fatal("later instruction did not execute")
			}
		})
	}
}

func TestRONSingleCounterRemovalTarget(t *testing.T) {
	for _, scenario := range []string{"present", "returned", "phase", "shroud"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			rule := `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([RemoveCounters([ACounterOfTypeFromPermanent(ChargeCounter,Ref_TargetPermanent)]),GainLife(Integer(1))])))`
			actor := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", rule)(), a.PlayerID())
			c := mage.NewCreature("Target", "", 2, 3)
			c.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(c, b.PlayerID())
			g.MutablePermanent(p.ID()).Counters[core.Charge] = 3
			if err := g.ActivateAbilityByIndex(a.PlayerID(), actor.ID(), 0, []uuid.UUID{actor.ID()}); err == nil {
				t.Fatal("illegal target accepted")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), actor.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "returned":
				g.BouncePermanentToHand(p)
				b.RemoveFromHand(c.ID())
				g.PutOnBattlefield(c, b.PlayerID())
				g.MutablePermanent(p.ID()).Counters[core.Charge] = 3
			case "phase":
				g.PhaseOut(p.ID())
			case "shroud":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.GrantKeywordModification(mage.This, core.Shroud)), p.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			g.ResolveStack()
			want := uint8(3)
			life := 20
			if scenario == "present" {
				want = 2
				life = 21
			}
			if g.FindPermanentIncludingPhased(p.ID()).Counters[core.Charge] != want || a.Life() != life {
				t.Fatal("target legality or incarnation")
			}
		})
	}
}

func TestRONSingleCounterRemovalClauses(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		complete  bool
	}{{"LEG", "Divine Intervention", false}, {"FEM", "Ebon Praetor", false}, {"FEM", "Merseine", false}, {"FEM", "Tourach's Gate", false}, {"ATQ", "Armageddon Clock", true}, {"ATQ", "Clockwork Avian", true}, {"2ED", "Clockwork Beast", true}, {"LEG", "Glyph of Delusion", false}} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile("../../../../cards/mtgish/" + tc.set + ".lines.ron")
			if err != nil {
				t.Fatal(err)
			}
			nodes, err := Parse(strings.NewReader(string(data)), tc.set+".ron")
			if err != nil {
				t.Fatal(err)
			}
			var action *Node
			var walk func(Node)
			walk = func(n Node) {
				if n.Name == "ActionList" {
					for _, a := range n.Args[0].Args {
						if a.Name == "RemoveCounters" {
							v := a
							action = &v
						}
					}
				}
				for _, a := range n.Args {
					walk(a)
				}
			}
			for _, n := range nodes {
				if n.Fields["Name"].String == tc.name {
					if _, err := Compile(n); (err == nil) != tc.complete {
						t.Fatalf("full-card support: %v, expected complete=%v", err, tc.complete)
					}
					for _, r := range n.Fields["Rules"].Args {
						walk(r)
					}
				}
			}
			if action == nil {
				t.Fatal("missing real resolving clause")
			}
			effect, err := compileResolvingCounterRemoval(*action, scope{permanent: true})
			if err != nil {
				t.Fatal(err)
			}
			counter, err := compilePermanentCounterType(action.Args[0].Args[0].Args[0])
			if err != nil {
				t.Fatal(err)
			}
			g, a, _ := ronGame()
			p := g.PutOnBattlefield(mage.NewCreature("Clause", "", 5, 5), a.PlayerID())
			g.MutablePermanent(p.ID()).Counters[counter] = 3
			if err := mage.ApplyEffect(g, effect(instanceBindings{}), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(p.ID()).Counters[counter] != 2 {
				t.Fatal("real clause did not remove exactly one")
			}
		})
	}
}

func TestRONSingleCounterRemovalDeclarations(t *testing.T) {
	for _, action := range []string{
		`RemoveCounters([])`, `RemoveCounters([ACounterOfTypeFromPermanent(ChargeCounter,ThisPermanent),ACounterOfTypeFromPermanent(TideCounter,ThisPermanent)])`,
		`RemoveCounters([ACounterOfTypeFromPermanent(ChargeCounter)])`, `RemoveCounters([ACounterOfTypeFromPermanent(ChargeCounter,ThisPermanent,ThisPermanent)])`,
		`RemoveCounters([ACounterOfTypeFromPermanent(PoisonCounter,ThisPermanent)])`, `RemoveCounters([ACounterOfTypeFromPermanent(ChargeCounter(),ThisPermanent)])`,
		`RemoveCounters([ACounterOfTypeFromPermanent(ChargeCounter,You)])`, `RemoveCounters([ACounterOfTypeFromPermanent(ChargeCounter,HostPermanent)])`,
		`RemoveCounters([ACounterOfTypeFromPermanent(ChargeCounter,Ref_TargetPermanent)])`, `RemoveCounters([ACounterOfTypeFromPermanent(ChargeCounter,Trigger_ThatPermanent)])`,
		`RemoveCounters([AnyNumberOfCountersOfTypeFromPermanent(ChargeCounter,ThisPermanent)])`, `RemoveCounters([NumberCountersOfTypeFromPermanent(Integer(1),ChargeCounter,ThisPermanent)])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+action+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Errorf("accepted %s", action)
		}
	}
	src := `Card(Name:"Single",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([` + singleCounterAction + `]))])`
	report, err := InspectFS(fstest.MapFS{"single.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 {
		t.Fatal(report.Issues)
	}
	for name, role := range map[string]string{"ACounterOfTypeFromPermanent": "counter_removal_instruction", "ChargeCounter": "permanent_counter_type", "ThisPermanent": "permanent_reference"} {
		found := false
		for _, s := range report.Signatures {
			if s.Name == name && s.Role == role {
				found = true
			}
		}
		if !found {
			t.Errorf("missing role %s for %s", role, name)
		}
	}
}

func TestRONSingleCounterRemovalHost(t *testing.T) {
	for _, scenario := range []string{"present", "reattach", "source-departed", "source-returned", "host-returned", "phase", "phase-cycle", "control", "clone"} {
		t.Run(scenario, func(t *testing.T) {
			n := hostUntapRecord(t, "LEG", "Venarian Gold")
			factory, err := Compile(n)
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			hostCard := mage.NewCreature("Host", "", 2, 3)
			hostCard.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 3), b.PlayerID())
			c := factory()
			c.SetOwner(a.PlayerID())
			aura := g.PutOnBattlefield(c, a.PlayerID())
			g.Attach(aura.ID(), host.ID())
			g.PutTriggersOnStack()
			g.ResolveStack()
			g.MutablePermanent(host.ID()).Counters[core.Sleep] = 3
			g.MutablePermanent(other.ID()).Counters[core.Sleep] = 7
			g.SetActivePlayerIndex(1)
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("host upkeep trigger missing")
			}
			wantHost, wantOther := uint8(2), uint8(7)
			switch scenario {
			case "reattach":
				g.Attach(aura.ID(), other.ID())
				wantHost, wantOther = 3, 6
			case "source-departed", "source-returned":
				g.BouncePermanentToHand(g.FindPermanent(aura.ID()))
				if scenario == "source-returned" {
					a.RemoveFromHand(c.ID())
					g.PutOnBattlefield(c, a.PlayerID())
					g.Attach(c.ID(), other.ID())
				}
			case "host-returned":
				g.BouncePermanentToHand(g.FindPermanent(host.ID()))
				b.RemoveFromHand(hostCard.ID())
				g.PutOnBattlefield(hostCard, b.PlayerID())
				g.MutablePermanent(host.ID()).Counters[core.Sleep] = 3
				wantHost = 3
			case "phase":
				g.PhaseOut(host.ID())
				wantHost = 3
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: host.ID(), TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				g.ApplyContinuousEffects()
			case "clone":
				clone := g.Clone()
				clone.ResolveStack()
				if clone.FindPermanent(host.ID()).Counters[core.Sleep] != 2 || g.FindPermanent(host.ID()).Counters[core.Sleep] != 3 {
					t.Fatal("host clone isolation")
				}
			}
			g.ResolveStack()
			if g.FindPermanentIncludingPhased(host.ID()).Counters[core.Sleep] != wantHost || g.FindPermanent(other.ID()).Counters[core.Sleep] != wantOther {
				t.Fatal("host attachment or incarnation")
			}
		})
	}
}

func TestRONSingleCounterRemovalEvent(t *testing.T) {
	g, a, b := ronGame()
	rule := `TriggerA(WhenAPermanentBecomesTapped(SinglePermanent(HostPermanent)),ActionList([RemoveCounters([ACounterOfTypeFromPermanent(ChargeCounter,Trigger_ThatPermanent)])]))`
	nodes, err := Parse(strings.NewReader(`Card(Name:"Event",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),`+rule+`])`), "event.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 3), a.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 3), b.PlayerID())
	aura := g.PutOnBattlefield(factory(), a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	g.MutablePermanent(host.ID()).Counters[core.Charge] = 3
	g.MutablePermanent(other.ID()).Counters[core.Charge] = 7
	g.TapPermanent(g.FindPermanent(host.ID()))
	g.PutTriggersOnStack()
	g.Attach(aura.ID(), other.ID())
	g.TapPermanent(g.FindPermanent(other.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("independent events not queued")
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(host.ID()).Counters[core.Charge] != 3 || g.FindPermanent(other.ID()).Counters[core.Charge] != 6 {
		t.Fatal("event binding overwritten")
	}
	g.ResolveStack()
	if g.FindPermanent(host.ID()).Counters[core.Charge] != 2 {
		t.Fatal("first event lost identity")
	}
}

func TestRONSingleCounterRemovalVenarianCopies(t *testing.T) {
	a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 3), b.PlayerID())
	c := repositoryCard(t, "LEG", "Venarian Gold")
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	a.ManaPool().Add(core.Blue, 4)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{host.ID()}, 2); err != nil {
		t.Fatal(err)
	}
	spellCopy := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
	if spellCopy == nil {
		t.Fatal("Aura copy failed")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if clone.FindPermanent(host.ID()).Counters[core.Sleep] != 4 || g.FindPermanent(host.ID()).Counters[core.Sleep] != 0 {
		t.Fatal("copied spell X or clone isolation")
	}
	g.ResolveStack()
	if g.FindPermanent(host.ID()).Counters[core.Sleep] != 4 || !g.FindPermanent(spellCopy.Card.ID()).IsToken {
		t.Fatal("copy entry counters or token")
	}
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(core.Upkeep)
	if g.FindPermanent(host.ID()).Counters[core.Sleep] != 2 {
		t.Fatal("Aura copies did not remove independent counters")
	}
	g.RunStepWithPriority(core.Upkeep)
	if g.FindPermanent(host.ID()).Counters[core.Sleep] != 0 {
		t.Fatal("Aura copies did not reach zero")
	}
	g.RunStepWithPriority(core.Upkeep)
	if g.FindPermanent(host.ID()).Counters[core.Sleep] != 0 {
		t.Fatal("removal at zero changed count")
	}
}
