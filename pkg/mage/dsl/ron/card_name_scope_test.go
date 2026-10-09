package ron

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
)

func nameScopeNode(t *testing.T, text string) Node {
	t.Helper()
	nodes, err := Parse(strings.NewReader(`Card(Name:"Scope",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+text+`]))])`), "scope.ron")
	if err != nil {
		t.Fatal(err)
	}
	return nodes[0].Fields["Rules"].Args[0].Args[0].Args[0].Args[0]
}

func TestRONCardNameScopeOperand(t *testing.T) {
	n := Node{Kind: Symbol, Name: "TheChosenName"}
	if _, err := compileCardNameRef(n, scope{}); err == nil {
		t.Fatal("unbound name accepted")
	}
	plan := &cardNameChoicePlan{name: "chosen"}
	for _, bad := range []Node{{Kind: Tuple, Name: "TheChosenName"}, {Kind: Symbol, Name: "TheChosenPlayer"}, {Kind: String, String: "Ice"}} {
		if _, err := compileCardNameRef(bad, scope{chosenName: plan}); err == nil {
			t.Fatal("invalid name operand accepted")
		}
	}
	build, err := compileCardNameRef(n, scope{chosenName: plan})
	if err != nil {
		t.Fatal(err)
	}
	first, second := instanceBindings{chosenNames: make(map[*cardNameChoicePlan]*mage.CardNameBinding)}, instanceBindings{chosenNames: make(map[*cardNameChoicePlan]*mage.CardNameBinding)}
	ref := build(first)
	if ref != build(first) || ref == build(second) {
		t.Fatal("name declarations not local to instance")
	}
}

func TestRONCardNameScopePlayerActions(t *testing.T) {
	for _, body := range []string{
		`PlayerActions(You,[ChooseACardName(AnyOracleCard)])`,
		`PlayerActions(You,[PlayerActions(You,[ChooseACardName(AnyOracleCard)])])`,
	} {
		n := nameScopeNode(t, body)
		effect, plan, err := compilePlayerActionsWithName(n, scope{})
		if err != nil {
			t.Fatal(err)
		}
		if plan == nil {
			t.Fatal("unconditional scoped name did not escape")
		}
		build, err := compileCardNameRef(Node{Kind: Symbol, Name: "TheChosenName"}, scope{chosenName: plan})
		if err != nil {
			t.Fatal(err)
		}
		g, base, _ := ronGame()
		a := &ronCardNameTester{Player: base, name: "Ice"}
		g.SetPlayerAt(0, a)
		bindings := instanceBindings{chosenNames: make(map[*cardNameChoicePlan]*mage.CardNameBinding)}
		ref := build(bindings)
		action := effect(bindings)
		ctx := &mage.EffectContext{Game: g, Controller: a.PlayerID()}
		if _, ok := ref.Resolve(ctx); ok || a.calls != 0 {
			t.Fatal("declaration executed choice")
		}
		if err := action.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if name, ok := ref.Resolve(ctx); !ok || name != "Ice" || a.calls != 1 {
			t.Fatalf("name=%q present=%v calls=%d", name, ok, a.calls)
		}
		clone := g.Clone()
		a.name = "Fire"
		if err := action.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if name, _ := ref.Resolve(&mage.EffectContext{Game: clone}); name != "Ice" {
			t.Fatal("clone shared chosen name")
		}
	}
}

func TestRONCardNameScopeRealOperands(t *testing.T) {
	for _, name := range []string{"Nebuchadnezzar", "Petra Sphinx"} {
		record := staticBaseRecord(t, "LEG", name)
		var chosen, choiceNode Node
		var visit func(Node)
		visit = func(n Node) {
			if n.Name == "ChooseACardName" {
				choiceNode = n
			}
			if n.Name == "TheChosenName" {
				chosen = n
			}
			for _, arg := range n.Args {
				visit(arg)
			}
			for _, field := range n.Fields {
				visit(field)
			}
		}
		visit(record)
		if chosen.Name == "" {
			t.Fatal("missing real operand")
		}
		if _, err := compileCardNameRef(chosen, scope{}); err == nil {
			t.Fatal("real operand accepted without producer")
		}
		choice, plan, err := compileCardNameChoiceWithResult(choiceNode, scope{})
		if err != nil {
			t.Fatal(err)
		}
		read, err := compileCardNameRef(chosen, scope{chosenName: plan})
		if err != nil {
			t.Fatal(err)
		}
		bindings := instanceBindings{chosenNames: make(map[*cardNameChoicePlan]*mage.CardNameBinding)}
		action, ref := choice(bindings), read(bindings)
		g, base, _ := ronGame()
		chooser := &ronCardNameTester{Player: base, name: "Ice"}
		g.SetPlayerAt(0, chooser)
		ctx := &mage.EffectContext{Game: g, Controller: chooser.PlayerID()}
		if err := action.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if got, ok := ref.Resolve(ctx); !ok || got != "Ice" || chooser.calls != 1 {
			t.Fatal("real choice/read operands did not execute")
		}
		if _, err := Compile(record); (err == nil) != (name == "Petra Sphinx") {
			t.Fatalf("unexpected complete record support for %s: %v", name, err)
		}
	}
}

func TestRONCardNameScopeSequence(t *testing.T) {
	first := nameScopeNode(t, `ChooseACardName(AnyOracleCard)`)
	scoped := nameScopeNode(t, `PlayerActions(TheChosenPlayer,[ChooseACardName(AnyOracleCard)])`)
	for _, parts := range [][]Node{{first}, {first, scoped}, {first, nameScopeNode(t, `PlayerActions(TheChosenPlayer,[GainLife(Integer(1))])`)}} {
		playerPlan := &playerChoicePlan{name: "chosen player"}
		builds, last, err := compileActionSequenceWithName(parts, scope{chosenPlayer: playerPlan})
		if err != nil {
			t.Fatal(err)
		}
		read, err := compileCardNameRef(Node{Kind: Symbol, Name: "TheChosenName"}, scope{chosenName: last})
		if err != nil {
			t.Fatal(err)
		}
		g, base, b := ronGame()
		a := &ronCardNameTester{Player: base, name: "Ice"}
		other := &ronCardNameTester{Player: b, name: "Fire"}
		g.SetPlayerAt(0, a)
		g.SetPlayerAt(1, other)
		bindings := instanceBindings{chosenNames: make(map[*cardNameChoicePlan]*mage.CardNameBinding)}
		player := mage.BindPlayer("chosen player")
		if err := mage.SnapshotPlayer(mage.You, player).Apply(&mage.EffectContext{Game: g, Controller: b.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		bindings.chosenPlayers = map[*playerChoicePlan]*mage.PlayerBinding{playerPlan: player}
		var effects []mage.Effect
		for _, build := range builds {
			effects = append(effects, build(bindings))
		}
		ref := read(bindings)
		ctx := &mage.EffectContext{Game: g, Controller: a.PlayerID()}
		if err := mage.Sequence(effects...).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		want := "Ice"
		if len(parts) > 1 && parts[1].Args[1].Args[0].Name == "ChooseACardName" {
			want = "Fire"
		}
		if name, ok := ref.Resolve(ctx); !ok || name != want {
			t.Fatalf("name=%q present=%v want=%q", name, ok, want)
		}
		for plan, binding := range bindings.chosenNames {
			if name, _ := binding.Resolve(ctx); plan != last && name != "Ice" {
				t.Fatal("nested choice overwrote outer binding")
			}
		}
		otherBindings := instanceBindings{chosenNames: make(map[*cardNameChoicePlan]*mage.CardNameBinding)}
		otherRef := read(otherBindings)
		if _, ok := otherRef.Resolve(ctx); ok {
			t.Fatal("factory instances shared result")
		}
	}
}

func TestRONCardNameScopeBranchBoundary(t *testing.T) {
	branch := nameScopeNode(t, `If(PermanentPassesFilter(ThisPermanent,IsTapped),[ChooseACardName(AnyOracleCard)])`)
	_, last, err := compileActionSequenceWithName([]Node{branch}, scope{permanent: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := compileCardNameRef(Node{Kind: Symbol, Name: "TheChosenName"}, scope{chosenName: last}); err == nil {
		t.Fatal("conditional choice escaped branch")
	}
	outer := &cardNameChoicePlan{name: "outer"}
	_, last, err = compileActionSequenceWithName([]Node{branch}, scope{permanent: true, chosenName: outer})
	if err != nil {
		t.Fatal(err)
	}
	if last != outer {
		t.Fatal("branch replaced outer lexical binding")
	}
}
