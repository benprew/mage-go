package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONBeginCombatBinding(t *testing.T) {
	for _, players := range []string{"AnyPlayer", "SinglePlayer(You)"} {
		t.Run(players, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Enchantment", `TriggerA(AtTheBeginningOfCombatDuringAPlayersTurn(`+players+`),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(2))),GainLife(Integer(1))]))`)
			g, a, b := ronGame()
			c := factory()
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.FireEvent(core.GameEvent{Type: core.EvtEndOfCombat, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("wrong step")
			}
			for _, player := range []mage.Player{a, b, a} {
				g.FireEvent(core.GameEvent{Type: core.EvtBeginCombat, PlayerID: player.PlayerID()})
			}
			g.BouncePermanentToHand(g.FindPermanent(p.ID()))
			a.RemoveFromHand(c.ID())
			g.PutOnBattlefield(c, b.PlayerID())
			g.PutTriggersOnStack()
			want := 3
			if players != "AnyPlayer" {
				want = 2
			}
			if g.StackSize() != want || a.Life() != 20 || b.Life() != 20 {
				t.Fatal("capture timing")
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			wantA, wantB := 27, 22
			if players != "AnyPlayer" {
				wantA, wantB = 26, 20
			}
			if a.Life() != wantA || b.Life() != wantB || clone.GetPlayer(a.PlayerID()).Life() != wantA {
				t.Fatal("event player or saved controller")
			}
		})
	}
}

func TestRONBeginCombatBatteringRamClause(t *testing.T) {
	n := hostUntapRecord(t, "ATQ", "Battering Ram")
	rule := n.Fields["Rules"].Args[0]
	build, err := compileRule(rule, scope{permanent: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"none", "control", "return", "phase-before", "phase-after", "phase-cycle", "remove-before", "remove-after", "copy", "two"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			c := mage.NewCreature("Combat clause", "", 1, 1, build())
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 1, build()), a.PlayerID())
			if change != "two" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(other.ID(), core.Indefinite))
			}
			if change == "phase-before" {
				g.PhaseOut(p.ID())
			}
			if change == "remove-before" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
			}
			g.FireEvent(core.GameEvent{Type: core.EvtBeginCombat, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("opponent combat triggered")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtBeginCombat, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if change == "phase-before" || change == "remove-before" {
				if g.StackSize() != 0 {
					t.Fatal("inactive source triggered")
				}
				return
			}
			want := 1
			if change == "two" {
				want = 2
			}
			if g.StackSize() != want || g.FindPermanent(p.ID()).HasKeyword(core.Banding) {
				t.Fatal("detection executed effect")
			}
			switch change {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(c.ID())
				p = g.PutOnBattlefield(c, a.PlayerID())
			case "phase-after":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "remove-after":
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			clone.ResolveStack()
			g.ResolveStack()
			has := change != "return" && change != "phase-after"
			for _, game := range []*mage.Game{g, clone} {
				current := game.FindPermanentIncludingPhased(p.ID())
				if current.HasKeyword(core.Banding) != has {
					t.Fatal("source identity or presence")
				}
				if game.FindPermanent(other.ID()).HasKeyword(core.Banding) != (change == "two") {
					t.Fatal("instance binding")
				}
				game.RunStepWithPriority(core.EndCombat)
				if game.FindPermanentIncludingPhased(p.ID()).HasKeyword(core.Banding) {
					t.Fatal("combat duration did not expire")
				}
			}
		})
	}
}

func TestRONBeginCombatInvalid(t *testing.T) {
	for _, event := range []string{`AtTheBeginningOfCombatDuringAPlayersTurn`, `AtTheBeginningOfCombatDuringAPlayersTurn()`, `AtTheBeginningOfCombatDuringAPlayersTurn(AnyPlayer,AnyPlayer)`, `AtTheBeginningOfCombatDuringAPlayersTurn(AnyPlayer())`, `AtTheBeginningOfCombatDuringAPlayersTurn(Opponent)`, `AtTheBeginningOfCombatDuringAPlayersTurn(SinglePlayer(You()))`, `AtTheBeginningOfCombatDuringAPlayersTurn(SinglePlayer(HostController))`, `AtTheBeginningOfCombatDuringAPlayersTurn(SinglePlayer(Trigger_ThatPlayer))`} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[TriggerA(%s,ActionList([GainLife(Integer(1))]))])`, event)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", event)
		}
	}
}

func TestRONBeginCombatInventory(t *testing.T) {
	src := `Card(Name:"Combat",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[TriggerA(AtTheBeginningOfCombatDuringAPlayersTurn(SinglePlayer(You)),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))]))])`
	report, err := InspectFS(fstest.MapFS{"combat.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 {
		t.Fatal(report.Issues)
	}
	for name, role := range map[string]string{"SinglePlayer": "player_filter", "You": "player_reference", "AtTheBeginningOfCombatDuringAPlayersTurn": "event"} {
		found := false
		for _, s := range report.Signatures {
			if s.Name == name && s.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatal("missing role", name, role)
		}
	}
}

func TestRONBeginCombatNativeAndControl(t *testing.T) {
	build, err := compileRule(hostUntapRecord(t, "ATQ", "Battering Ram").Fields["Rules"].Args[0], scope{permanent: true})
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	p := g.PutOnBattlefield(mage.NewCreature("Combat clause", "", 1, 1, build()), a.PlayerID())
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(core.BeginCombat)
	if g.FindPermanent(p.ID()).HasKeyword(core.Banding) {
		t.Fatal("opponent native step")
	}
	g.SetActivePlayerIndex(0)
	for range 2 {
		g.RunStepWithPriority(core.BeginCombat)
		if !g.FindPermanent(p.ID()).HasKeyword(core.Banding) {
			t.Fatal("native combat start")
		}
		g.RunStepWithPriority(core.EndCombat)
		if g.FindPermanent(p.ID()).HasKeyword(core.Banding) {
			t.Fatal("native combat end")
		}
	}
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.RunStepWithPriority(core.BeginCombat)
	if g.FindPermanent(p.ID()).HasKeyword(core.Banding) {
		t.Fatal("old controller still triggers")
	}
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(core.BeginCombat)
	if !g.FindPermanent(p.ID()).HasKeyword(core.Banding) {
		t.Fatal("new controller does not trigger")
	}
}

func TestRONBeginCombatConsumerBoundaries(t *testing.T) {
	for _, tc := range []struct {
		set, name, remaining string
		any                  bool
	}{
		{"ATQ", "Battering Ram", "unbound or incompatible permanent reference (Trigger_TheAttackingCreature)", false},
		{"FEM", "Goblin Flotilla", "CreateTriggerUntil", true},
		{"LEG", "Johan", "optional payment cost", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := hostUntapRecord(t, tc.set, tc.name)
			if _, err := Compile(n); err == nil || !strings.Contains(err.Error(), tc.remaining) {
				t.Fatalf("remaining gap: %v", err)
			}
			var event Node
			for _, r := range n.Fields["Rules"].Args {
				if r.Name == "TriggerA" && r.Args[0].Name == "AtTheBeginningOfCombatDuringAPlayersTurn" {
					event = r.Args[0]
				}
			}
			bodyNodes, err := Parse(strings.NewReader(`ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))])`), "body.ron")
			if err != nil {
				t.Fatal(err)
			}
			build, err := compileBoundStepTrigger(event, bodyNodes[0], scope{permanent: true}, mage.EachBeginCombat, mage.YourBeginCombat, "begin-combat")
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			g.PutOnBattlefield(mage.NewEnchantment("Event clause", "", build()), a.PlayerID())
			g.SetActivePlayerIndex(1)
			g.RunStepWithPriority(core.BeginCombat)
			wantB := 20
			if tc.any {
				wantB = 21
			}
			if b.Life() != wantB {
				t.Fatal("real event player filter")
			}
			g.SetActivePlayerIndex(0)
			g.RunStepWithPriority(core.BeginCombat)
			if a.Life() != 21 {
				t.Fatal("real event on controller turn")
			}
		})
	}
}

func TestRONBeginCombatScopeRejection(t *testing.T) {
	for _, rules := range []string{
		`TriggerA(AtTheBeginningOfCombatDuringAPlayersTurn(AnyPlayer),ActionList([])),Activated(PayMana([]),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))]))`,
		`TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbility([Banding])],UntilEndOfCombat)]))`,
		`TriggerA(AtTheBeginningOfCombatDuringAPlayersTurn(AnyPlayer),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbility([Banding])],UntilEndOfCombat)]))]))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + rules + `])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid scope", rules)
		}
	}
}

func TestRONBeginCombatCopiedEvent(t *testing.T) {
	factory := permanentScopeFactory(t, "Enchantment", `TriggerA(AtTheBeginningOfCombatDuringAPlayersTurn(AnyPlayer),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(2))),GainLife(Integer(1))]))`)
	g, a, b := ronGame()
	c := factory()
	if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(c, a.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtBeginCombat, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("missing copy")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	for _, game := range []*mage.Game{g, clone} {
		if game.GetPlayer(a.PlayerID()).Life() != 21 || game.GetPlayer(b.PlayerID()).Life() != 25 {
			t.Fatal("copy changed event player or retained original controller")
		}
	}
}
