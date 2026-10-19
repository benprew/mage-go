package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONOrdinaryDrawStepBindings(t *testing.T) {
	for _, players := range []string{"AnyPlayer", "SinglePlayer(You)"} {
		t.Run(players, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Enchantment", `TriggerA(AtTheBeginningOfAPlayersDrawStep(`+players+`),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(2))),GainLife(Integer(1))]))`)
			g, a, b := ronGame()
			c := factory()
			c.SetOwner(a.PlayerID())
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("wrong step triggered")
			}
			for _, player := range []mage.Player{a, b, a} {
				g.FireEvent(core.GameEvent{Type: core.EvtDrawStep, PlayerID: player.PlayerID()})
			}
			g.BouncePermanentToHand(g.FindPermanent(p.ID()))
			a.RemoveFromHand(c.ID())
			g.PutOnBattlefield(c, b.PlayerID())
			g.PutTriggersOnStack()
			wantCount, wantA, wantB := 3, 27, 22
			if players == "SinglePlayer(You)" {
				wantCount, wantA, wantB = 2, 26, 20
			}
			if g.StackSize() != wantCount || a.Life() != 20 || b.Life() != 20 {
				t.Fatal("event filtering or detection timing")
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				if game.GetPlayer(a.PlayerID()).Life() != wantA || game.GetPlayer(b.PlayerID()).Life() != wantB {
					t.Fatal("pending events lost their player or controller")
				}
			}
		})
	}
}

func ordinaryDrawClockEventProbe(t *testing.T) mage.Card {
	t.Helper()
	n := hostUntapRecord(t, "ATQ", "Armageddon Clock")
	event := n.Fields["Rules"].Args[1].Args[0]
	// A test body keeps the missing counter value outside this event test.
	body, err := Parse(strings.NewReader(`ActionList([PermanentDealsDamage(ThisPermanent,Integer(3),EachPlayer(AnyPlayer))])`), "draw-probe.ron")
	if err != nil {
		t.Fatal(err)
	}
	build, err := compileDrawStepTrigger(event, body[0], scope{permanent: true})
	if err != nil {
		t.Fatal(err)
	}
	c := mage.NewArtifact("Clock event probe", "", build())
	if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRONOrdinaryDrawStepEventLifecycle(t *testing.T) {
	for _, change := range []string{"live", "control-before", "control-after", "leave", "return", "phase-before", "phase-after", "phase-cycle", "remove-before", "remove-after", "two", "counter"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			c := ordinaryDrawClockEventProbe(t)
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			if change == "two" {
				g.PutOnBattlefield(ordinaryDrawClockEventProbe(t), a.PlayerID())
			}
			if change == "control-before" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			if change == "phase-before" {
				g.PhaseOut(p.ID())
			}
			if change == "remove-before" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
			}
			g.FireEvent(core.GameEvent{Type: core.EvtDrawStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if change == "control-before" || change == "phase-before" || change == "remove-before" {
				if g.StackSize() != 0 {
					t.Fatal("ineligible source triggered")
				}
				if change == "control-before" {
					g.FireEvent(core.GameEvent{Type: core.EvtDrawStep, PlayerID: b.PlayerID()})
					g.PutTriggersOnStack()
					if g.StackSize() != 1 {
						t.Fatal("new controller did not trigger")
					}
				}
				return
			}
			wantCount := 1
			if change == "two" {
				wantCount = 2
			}
			if g.StackSize() != wantCount || a.Life() != 20 || b.Life() != 20 {
				t.Fatal("damage occurred before resolution or instance lost")
			}
			switch change {
			case "control-after":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "leave", "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				if change == "return" {
					a.RemoveFromHand(c.ID())
					g.PutOnBattlefield(c, b.PlayerID())
				}
			case "phase-after":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "remove-after":
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
			case "counter":
				g.CounterStackObject(g.StackPeek().ID)
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone dealt damage in original")
			}
			g.ResolveStack()
			want := 17
			if change == "two" {
				want = 14
			}
			if change == "counter" {
				want = 20
			}
			for _, game := range []*mage.Game{g, clone} {
				if game.GetPlayer(a.PlayerID()).Life() != want || game.GetPlayer(b.PlayerID()).Life() != want {
					t.Fatalf("damage used wrong source information: %d/%d, want %d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life(), want)
				}
			}
		})
	}
}

func TestRONOrdinaryDrawStepNativeTiming(t *testing.T) {
	g, a, b := ronGame()
	seedRONDrawLibrary(a)
	seedRONDrawLibrary(b)
	g.SetTurn(2)
	c := ordinaryDrawClockEventProbe(t)
	g.PutOnBattlefield(c, a.PlayerID())
	first := true
	g.SetOnPriority(func(game *mage.Game, _ int, _ bool) mage.PriorityAction {
		if first {
			first = false
			if len(a.Hand()) != 1 || game.StackSize() != 1 || a.Life() != 20 || b.Life() != 20 {
				t.Fatal("draw or damage timing before priority")
			}
		}
		return mage.PriorityAction{Type: mage.PriorityPass}
	})
	g.RunStepWithPriority(core.Draw)
	if first || a.Life() != 17 || b.Life() != 17 {
		t.Fatal("native draw did not resolve event probe")
	}
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(core.Draw)
	if len(b.Hand()) != 1 || a.Life() != 17 || b.Life() != 17 {
		t.Fatal("opponent step triggered controller-only rule")
	}
}

func TestRONOrdinaryDrawStepCopy(t *testing.T) {
	factory := permanentScopeFactory(t, "Enchantment", `TriggerA(AtTheBeginningOfAPlayersDrawStep(AnyPlayer),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(2))),GainLife(Integer(1))]))`)
	g, a, b := ronGame()
	g.PutOnBattlefield(factory(), a.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtDrawStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 25 {
		t.Fatal("copy changed event player or retained original controller")
	}
}

func TestRONOrdinaryDrawStepDeclarations(t *testing.T) {
	for _, event := range []string{
		`AtTheBeginningOfAPlayersDrawStep`, `AtTheBeginningOfAPlayersDrawStep()`,
		`AtTheBeginningOfAPlayersDrawStep(AnyPlayer,AnyPlayer)`, `AtTheBeginningOfAPlayersDrawStep(AnyPlayer())`,
		`AtTheBeginningOfAPlayersDrawStep(Opponent)`, `AtTheBeginningOfAPlayersDrawStep(SinglePlayer(You()))`,
		`AtTheBeginningOfAPlayersDrawStep(SinglePlayer(HostController))`, `AtTheBeginningOfAPlayersDrawStep(SinglePlayer(Trigger_ThatPlayer))`,
	} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[TriggerA(%s,ActionList([GainLife(Integer(1))]))])`, event)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid event", event)
		}
	}
	for _, src := range []string{
		`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[TriggerA(AtTheBeginningOfAPlayersDrawStep(AnyPlayer),ActionList([GainLife(Integer(1))]))])`,
		`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerA(AtTheBeginningOfAPlayersDrawStep(AnyPlayer),ActionList([])),Activated(PayMana([]),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))]))])`,
	} {
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid context")
		}
	}
}

func TestRONOrdinaryDrawStepInventoryAndBoundaries(t *testing.T) {
	src := `Card(Name:"Draw",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[TriggerA(AtTheBeginningOfAPlayersDrawStep(SinglePlayer(You)),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))]))])`
	r, err := InspectFS(fstest.MapFS{"draw.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil || r.Compiled != 1 {
		t.Fatalf("inspection: %v, %v", r, err)
	}
	if _, err := LoadFS(fstest.MapFS{"draw.ron": {Data: []byte(src)}}, "*.ron"); err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"AtTheBeginningOfAPlayersDrawStep": "event", "SinglePlayer": "player_filter", "You": "player_reference"} {
		found := false
		for _, s := range r.Signatures {
			found = found || (s.Name == name && s.Role == role)
		}
		if !found {
			t.Fatal("missing inventory role", name)
		}
	}
	clock, err := Compile(hostUntapRecord(t, "ATQ", "Armageddon Clock"))
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Armageddon Clock", clock()); err != nil {
		t.Fatal(err)
	}
	library, err := Compile(hostUntapRecord(t, "LEG", "Sylvan Library"))
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Sylvan Library", library()); err != nil {
		t.Fatal(err)
	}
}
