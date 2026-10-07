package ron

import (
	"fmt"
	"maps"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func hostUntapRecord(t *testing.T, set, name string) Node {
	t.Helper()
	data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), set+".lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range nodes {
		if n.Fields["Name"].String == name {
			return n
		}
	}
	t.Fatal("missing record", name)
	return Node{}
}

func hostUntapClause(t *testing.T, set, name string) mage.Card {
	t.Helper()
	n := hostUntapRecord(t, set, name)
	var selected []Node
	for _, rule := range n.Fields["Rules"].Args {
		if rule.Name == "EnchantPermanent" || rule.Name == "If" || rule.Name == "PermanentRuleEffect" {
			selected = append(selected, rule)
		}
	}
	fields := make(map[string]Node, len(n.Fields))
	maps.Copy(fields, n.Fields)
	rules := fields["Rules"]
	rules.Args = selected
	fields["Rules"] = rules
	n.Fields = fields
	factory, err := Compile(n)
	if err != nil {
		t.Fatal(err)
	}
	c := factory()
	if err := mage.ValidateCardDefinition(name, c); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRONHostStaticUntapClauses(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		counter   core.CounterType
		onHost    bool
	}{
		{"FEM", "Merseine", core.Net, false},
		{"LEG", "Venarian Gold", core.Sleep, true},
		{"LEA", "Paralyze", core.Charge, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			card := hostUntapClause(t, tc.set, tc.name)
			card.SetOwner(a.PlayerID())
			host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 8), b.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
				t.Fatal("unpaid cast accepted")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed cast changed state")
			}
			a.ManaPool().Add(core.Blue, 4)
			a.ManaPool().Add(core.Black, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			if source == nil || source.AttachedTo != host.ID() {
				t.Fatal("Aura did not attach")
			}
			subject := source.ID()
			if tc.onHost {
				subject = host.ID()
			}
			for _, count := range []uint8{0, 1, 3, 0, 2} {
				g.MutablePermanent(subject).Counters[tc.counter] = count
				g.TapPermanent(g.FindPermanent(host.ID()))
				g.SetActivePlayerIndex(1)
				g.RunStepWithPriority(core.Untap)
				want := count > 0 || tc.name == "Paralyze"
				if g.FindPermanent(host.ID()).Tapped != want {
					t.Fatalf("count=%d tapped=%v want %v", count, g.FindPermanent(host.ID()).Tapped, want)
				}
				if !g.UntapPermanent(g.FindPermanent(host.ID())) && want {
					t.Fatal("other untap prohibited")
				}
			}
		})
	}
}

func TestRONHostStaticUntapLifecycle(t *testing.T) {
	for _, scenario := range []string{"move", "source-leaves", "source-return", "host-return", "source-phase", "phase-cycle", "host-phase", "source-remove", "host-remove", "control", "clone", "two"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			card := hostUntapClause(t, "FEM", "Merseine")
			card.SetOwner(a.PlayerID())
			hostCard := mage.NewCreature("Host", "", 2, 8)
			hostCard.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 8), b.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(source.ID(), host.ID())
			g.MutablePermanent(source.ID()).Counters[core.Net] = 1
			clone := g.Clone()
			wantHost, wantOther := true, false
			switch scenario {
			case "move":
				g.Attach(source.ID(), other.ID())
				wantHost, wantOther = false, true
			case "source-leaves", "source-return":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				wantHost = false
				if scenario == "source-return" {
					a.RemoveFromHand(card.ID())
					source = g.PutOnBattlefield(card, a.PlayerID())
					g.Attach(source.ID(), other.ID())
					g.MutablePermanent(source.ID()).Counters[core.Net] = 1
					wantOther = true
				}
			case "host-return":
				g.BouncePermanentToHand(g.FindPermanent(host.ID()))
				b.RemoveFromHand(hostCard.ID())
				host = g.PutOnBattlefield(hostCard, b.PlayerID())
				wantHost = false
			case "source-phase":
				g.PhaseOut(source.ID())
				wantHost = false
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "host-phase":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "source-remove":
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
				wantHost = false
			case "host-remove":
				g.AddContinuousEffect(mage.RemoveAllAbilities(host.ID(), core.Indefinite))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "clone":
				clone.MutablePermanent(source.ID()).Counters[core.Net] = 0
			case "two":
				second := g.PutOnBattlefield(hostUntapClause(t, "FEM", "Merseine"), a.PlayerID())
				g.Attach(second.ID(), other.ID())
				g.MutablePermanent(second.ID()).Counters[core.Net] = 1
				wantOther = true
			}

			g.TapPermanent(g.FindPermanent(host.ID()))
			g.TapPermanent(g.FindPermanent(other.ID()))
			g.SetActivePlayerIndex(1)
			g.RunStepWithPriority(core.Untap)
			if scenario != "control" && g.FindPermanent(host.ID()).Tapped != wantHost {
				t.Fatal("native host untap")
			}
			if g.FindPermanent(other.ID()).Tapped != wantOther {
				t.Fatal("native other untap")
			}
			clone.TapPermanent(clone.FindPermanent(host.ID()))
			clone.SetActivePlayerIndex(1)
			clone.RunStepWithPriority(core.Untap)
			if clone.FindPermanent(host.ID()).Tapped != (scenario != "clone") {
				t.Fatal("clone lost independent condition")
			}
			if scenario == "control" {
				g.SetActivePlayerIndex(0)
				g.RunStepWithPriority(core.Untap)
				if !g.FindPermanent(host.ID()).Tapped {
					t.Fatal("new controller lost restriction")
				}
			}
		})
	}
}

func TestRONHostStaticUntapCocoonClause(t *testing.T) {
	n := hostUntapRecord(t, "LEG", "Cocoon")
	for _, rule := range n.Fields["Rules"].Args {
		if rule.Name != "If" {
			continue
		}
		option, err := compileRule(rule, scope{permanent: true, aura: true})
		if err != nil {
			t.Fatal(err)
		}
		g, a, b := ronGame()
		host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 8), b.PlayerID())
		card := mage.NewEnchantment("Cocoon clause", "", option())
		card.AddSubType("Aura")
		source := g.PutOnBattlefield(card, a.PlayerID())
		g.Attach(source.ID(), host.ID())
		g.MutablePermanent(source.ID()).Counters[core.Pupa] = 1
		g.TapPermanent(g.FindPermanent(host.ID()))
		g.SetActivePlayerIndex(1)
		g.RunStepWithPriority(core.Untap)
		if !g.FindPermanent(host.ID()).Tapped {
			t.Fatal("pupa counter did not stop untap")
		}
		g.MutablePermanent(source.ID()).Counters[core.Pupa] = 0
		g.RunStepWithPriority(core.Untap)
		if g.FindPermanent(host.ID()).Tapped {
			t.Fatal("removed counter still stops untap")
		}
		return
	}
	t.Fatal("missing conditional rule")
}

func TestRONHostStaticUntapInvalid(t *testing.T) {
	for _, rule := range []string{
		`PermanentRuleEffect(HostPermanent(),[DoesntUntapDuringControllersUntap])`,
		`PermanentRuleEffect(Ref_TargetPermanent,[DoesntUntapDuringControllersUntap])`,
		`PermanentRuleEffect(HostPermanent,[DoesntUntapDuringControllersUntap()])`,
		`PermanentRuleEffect(HostPermanent,[])`,
		`PermanentRuleEffect(HostPermanent,[DoesntUntapDuringControllersUntap,DoesntUntapDuringControllersUntap])`,
		`PermanentRuleEffect(HostPermanent,[MayChooseNotToUntapDuringUntap])`,
		`If(PermanentPassesFilter(ThisPermanent,HasACounterOfType(NetCounter)),[EachPermanentRuleEffect(AnyPermanent,[DoesntUntapDuringControllersUntap])])`,
		`If(PermanentPassesFilter(ThisPermanent,HasACounterOfType(NetCounter)),[PermanentRuleEffect(HostPermanent,[DoesntUntapDuringControllersUntap]),PermanentLayerEffect(HostPermanent,[AdjustPT(1,1)])])`,
	} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),%s])`, rule)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", rule)
		}
	}
	nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[PermanentRuleEffect(HostPermanent,[DoesntUntapDuringControllersUntap])])`), "invalid.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(nodes[0]); err == nil {
		t.Fatal("host outside Aura scope")
	}
}

func TestRONHostStaticUntapInventory(t *testing.T) {
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),If(PermanentPassesFilter(ThisPermanent,HasACounterOfType(NetCounter)),[PermanentRuleEffect(HostPermanent,[DoesntUntapDuringControllersUntap])])])`
	r, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if r.Compiled != 1 {
		t.Fatal("supported shape rejected", r.Issues)
	}
	for _, s := range r.Signatures {
		if s.Name == "DoesntUntapDuringControllersUntap" && s.Role == "permanent_rule_operation" {
			return
		}
	}
	t.Fatal("missing operation role")
}

func TestRONHostStaticUntapSource(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `If(PermanentPassesFilter(ThisPermanent, IsTapped), [PermanentRuleEffect(ThisPermanent, [DoesntUntapDuringControllersUntap])])`)
	g, a, _ := ronGame()
	c := factory()
	p := g.PutOnBattlefield(c, a.PlayerID())
	g.TapPermanent(p)
	g.RunStepWithPriority(core.Untap)
	if !g.FindPermanent(p.ID()).Tapped {
		t.Fatal("conditional source untapped")
	}
	if !g.UntapPermanent(g.FindPermanent(p.ID())) {
		t.Fatal("other untap blocked")
	}
	g.RunStepWithPriority(core.Untap)
	if g.FindPermanent(p.ID()).Tapped {
		t.Fatal("false source condition changed tap state")
	}
}
