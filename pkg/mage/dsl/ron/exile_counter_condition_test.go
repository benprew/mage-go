package ron

import (
	"maps"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONExileCounterConditionGameplay(t *testing.T) {
	for _, predicate := range []string{"HasACounterOfType", "HasNoCountersOfType"} {
		for _, scenario := range []string{"zero", "present", "changed", "depart", "return", "hidden", "clone", "copy", "two-instances"} {
			t.Run(predicate+"/"+scenario, func(t *testing.T) {
				condition := `ExiledCardPassesFilter(ThisExiledCard,` + predicate + `(ScreamCounter))`
				registry, err := LoadFS(fstest.MapFS{"source.ron": {Data: []byte(zoneConditionFixture("FromExile", condition))}}, "*.ron")
				if err != nil {
					t.Fatal(err)
				}
				g, a, b := ronGame()
				card, err := registry.CreateCard("Condition watcher")
				if err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				if scenario == "hidden" {
					g.ExileCardFaceDown(card, card.ID(), a.PlayerID())
				} else {
					g.ExileCard(card, card.ID())
				}
				count := 300
				if scenario == "zero" || predicate == "HasNoCountersOfType" && scenario != "present" {
					count = 0
				}
				g.FindExiledCard(card.ID()).Counters[core.Scream] = count
				g.FindExiledCard(card.ID()).Counters[core.Charge] = 2
				if scenario == "two-instances" {
					second, err := registry.CreateCard("Condition watcher")
					if err != nil {
						t.Fatal(err)
					}
					second.SetOwner(a.PlayerID())
					g.ExileCard(second, second.ID())
					g.FindExiledCard(second.ID()).Counters[core.Scream] = 1
					if count > 0 {
						g.FindExiledCard(second.ID()).Counters[core.Scream] = 0
					}
				}
				queueRONUpkeep(g, b)
				if g.StackSize() != 0 {
					t.Fatal("wrong upkeep player")
				}
				queueRONUpkeep(g, a)
				active := predicate == "HasACounterOfType" && count > 0 || predicate == "HasNoCountersOfType" && count == 0
				active = active && scenario != "hidden"
				wantStack := 0
				if active {
					wantStack = 1
				}
				if g.StackSize() != wantStack {
					t.Fatalf("stack = %d, want %d", g.StackSize(), wantStack)
				}
				clone := g.Clone()
				if scenario == "changed" {
					g.FindExiledCard(card.ID()).Counters[core.Scream] = 0
					if predicate == "HasNoCountersOfType" {
						g.FindExiledCard(card.ID()).Counters[core.Scream] = 1
					}
				}
				if scenario == "depart" || scenario == "return" || scenario == "clone" {
					g.RemoveFromExile(card.ID())
					if scenario == "return" {
						g.ExileCard(card, card.ID())
						g.FindExiledCard(card.ID()).Counters[core.Scream] = count
					}
				}
				if scenario == "copy" && active && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy missing")
				}
				g.ResolveStack()
				want := 20
				if active && (scenario == "zero" || scenario == "present" || scenario == "copy" || scenario == "two-instances") {
					want++
				}
				if active && scenario == "copy" {
					want++
				}
				if a.Life() != want || b.Life() != 20 {
					t.Fatalf("life = %d/%d, want %d/20", a.Life(), b.Life(), want)
				}
				if scenario == "clone" {
					clone.ResolveStack()
					if clone.GetPlayer(a.PlayerID()).Life() != 21 || clone.FindExiledCard(card.ID()).Counters[core.Scream] != count {
						t.Fatal("clone lost its counters or pending body")
					}
				}
			})
		}
	}
}

func TestRONExileCounterConditionAllHallowsEve(t *testing.T) {
	node := repositoryRecord(t, "LEG", "All Hallow's Eve", "LEG.ron")
	if _, err := Compile(node); err == nil || !strings.Contains(err.Error(), "PutEachCardFromEachPlayersGraveyardOntoTheBattlefield") {
		t.Fatalf("full record must retain its independent mass-entry gap: %v", err)
	}
	wrapper := node.Fields["Rules"].Args[1]
	trigger := wrapper.Args[0]
	condition := trigger.Args[1]
	branch := trigger.Args[2].Args[0].Args[1]
	probe, err := Parse(strings.NewReader(`ActionList([If(ExiledCardPassesFilter(ThisExiledCard,HasNoCountersOfType(ScreamCounter)),[GainLife(Integer(2))]),GainLife(Integer(1))])`), "probe.ron")
	if err != nil {
		t.Fatal(err)
	}
	probe[0].Args[0].Args[0].Args[0] = branch.Args[0]
	spell, err := Parse(strings.NewReader(`SpellActions(ActionList([GainLife(Integer(0))]))`), "probe.ron")
	if err != nil {
		t.Fatal(err)
	}
	trigger.Args = []Node{trigger.Args[0], condition, probe[0]}
	wrapper.Args = []Node{trigger}
	node.Fields = maps.Clone(node.Fields)
	node.Fields["Rules"] = Node{Kind: List, Args: []Node{spell[0], wrapper}}
	factory, err := Compile(node)
	if err != nil {
		t.Fatal(err)
	}
	for _, count := range []int{0, 1, 2, 300} {
		g, a, _ := ronGame()
		card := factory()
		card.SetOwner(a.PlayerID())
		g.ExileCard(card, card.ID())
		g.FindExiledCard(card.ID()).Counters[core.Scream] = count
		queueRONUpkeep(g, a)
		g.ResolveStack()
		want := 20
		if count > 0 {
			want++
		}
		if a.Life() != want {
			t.Fatalf("record predicate with %d counters: life = %d", count, a.Life())
		}
	}
	trigger.Name = "TriggerA"
	trigger.Args = []Node{trigger.Args[0], probe[0]}
	wrapper.Args = []Node{trigger}
	node.Fields["Rules"] = Node{Kind: List, Args: []Node{spell[0], wrapper}}
	factory, err = Compile(node)
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"zero", "one", "depart", "return", "clone"} {
		g, a, _ := ronGame()
		card := factory()
		card.SetOwner(a.PlayerID())
		g.ExileCard(card, card.ID())
		g.FindExiledCard(card.ID()).Counters[core.Scream] = 1
		queueRONUpkeep(g, a)
		clone := g.Clone()
		if scenario == "zero" || scenario == "clone" {
			g.FindExiledCard(card.ID()).Counters[core.Scream] = 0
		}
		if scenario == "depart" || scenario == "return" {
			g.RemoveFromExile(card.ID())
			if scenario == "return" {
				g.ExileCard(card, card.ID())
			}
		}
		g.ResolveStack()
		want := 21
		if scenario == "zero" || scenario == "clone" {
			want = 23
		}
		if a.Life() != want {
			t.Fatalf("resolving record predicate in %s: life = %d, want %d", scenario, a.Life(), want)
		}
		if scenario == "clone" {
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 21 {
				t.Fatal("resolving predicate shared clone counters")
			}
		}
	}
}

func TestRONExileCounterConditionInvalid(t *testing.T) {
	for _, condition := range []string{
		`ExiledCardPassesFilter(ThisExiledCard,HasACounterOfType())`,
		`ExiledCardPassesFilter(ThisExiledCard,HasACounterOfType(ScreamCounter,ChargeCounter))`,
		`ExiledCardPassesFilter(ThisExiledCard,HasNoCountersOfType(UnknownCounter))`,
		`ExiledCardPassesFilter(ThisExiledCard,HasNoCountersOfType(ScreamCounter()))`,
		`ExiledCardPassesFilter(ThisGraveyardCard,HasACounterOfType(ScreamCounter))`,
		`ExiledCardPassesFilter(ThisExiledCard(),HasACounterOfType(ScreamCounter))`,
		`ExiledCardPassesFilter(Ref_TargetExiledCard,HasACounterOfType(ScreamCounter))`,
		`ExiledCardPassesFilter(ThisExiledCard,And([HasACounterOfType(ScreamCounter)]))`,
		`ExiledCardPassesFilter(ThisExiledCard,HasNumberCountersOfType(GreaterThan(Integer(1)),ScreamCounter))`,
	} {
		if _, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(zoneConditionFixture("FromExile", condition))}}, "*.ron"); err == nil {
			t.Fatalf("invalid declaration accepted: %s", condition)
		}
	}
	condition := `ExiledCardPassesFilter(ThisExiledCard,HasNoCountersOfType(ScreamCounter))`
	for _, source := range []string{
		zoneConditionFixture("FromGraveyard", condition),
		`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([If(` + condition + `,[GainLife(Integer(1))])]))])`,
		strings.Replace(zoneConditionFixture("FromExile", condition), `PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))`, `If(`+condition+`,[])`, 1),
	} {
		if _, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(source)}}, "*.ron"); err == nil {
			t.Fatalf("invalid context accepted: %s", source)
		}
	}
	registry := mage.NewRegistry()
	registry.Register("Existing", func() mage.Card { return mage.NewArtifact("Existing", "") })
	if err := LoadInto(registry, fstest.MapFS{
		"good.ron": {Data: []byte(zoneConditionFixture("FromExile", condition))},
		"bad.ron":  {Data: []byte(zoneConditionFixture("FromGraveyard", condition))},
	}, "*.ron"); err == nil || !registry.CardRegistered("Existing") || registry.CardRegistered("Condition watcher") {
		t.Fatal("invalid collection changed the registry")
	}
}

func TestRONExileCounterConditionInventory(t *testing.T) {
	source := zoneConditionFixture("FromExile", `ExiledCardPassesFilter(ThisExiledCard,HasACounterOfType(ScreamCounter))`)
	source = strings.Replace(source, `PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))`, `If(ExiledCardPassesFilter(ThisExiledCard,HasNoCountersOfType(ScreamCounter)),[GainLife(Integer(1))])`, 1)
	report, err := InspectFS(fstest.MapFS{"source.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"ThisExiledCard": "card_reference", "HasACounterOfType": "card_filter", "HasNoCountersOfType": "card_filter", "ScreamCounter": "card_counter_type"} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == role {
				found = true
				if signature.Support != "unverified" {
					t.Fatal("inventory claimed gameplay coverage")
				}
			}
		}
		if !found {
			t.Fatalf("missing %s in %s", name, role)
		}
	}
}
