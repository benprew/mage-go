package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func clockworkTriggerCard(t *testing.T, set, name string) mage.Card {
	t.Helper()
	data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), set+".ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range nodes {
		if node.Fields["Name"].String != name {
			continue
		}
		rules := node.Fields["Rules"]
		var selected []Node
		for _, rule := range rules.Args {
			if rule.Name != "ActivatedWithModifiers" {
				selected = append(selected, rule)
			}
		}
		// The restoration rule has a separate compiler gap. This fixture tests the other rules.
		rules.Args = selected
		node.Fields["Rules"] = rules
		factory, err := Compile(node)
		if err != nil {
			t.Fatal(err)
		}
		card := factory()
		if err := mage.ValidateCardDefinition(name, card); err != nil {
			t.Fatal(err)
		}
		return card
	}
	t.Fatal("missing record", name)
	return nil
}

func TestRONEndCombatClockwork(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		counters  int
	}{{"ATQ", "Clockwork Avian", 4}, {"LEA", "Clockwork Beast", 7}} {
		for _, role := range []string{"attack", "block", "idle", "other", "enters-attacking", "removed-from-combat", "both"} {
			t.Run(tc.name+"/"+role, func(t *testing.T) {
				g, a, b := ronGame()
				card := clockworkTriggerCard(t, tc.set, tc.name)
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				if int(p.Counters[core.P1P0]) != tc.counters || p.CurrentPower(g) != tc.counters || p.HasAttr(core.Flying) != (tc.name == "Clockwork Avian") {
					t.Fatal("entry or characteristics")
				}
				g.RunStepWithPriority(core.BeginCombat)
				p = g.FindPermanent(p.ID())
				p.RevokeBaseAttr(core.AttrSummonSick)
				switch role {
				case "attack", "removed-from-combat", "both":
					g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
					if role == "removed-from-combat" {
						g.RemoveFromCombat(p.ID())
					}
					if role == "both" {
						g.FireEvent(core.GameEvent{Type: core.EvtDeclaredBlocker, SourceID: p.ID(), PlayerID: a.PlayerID()})
					}
				case "block":
					q := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 1, 8), b.PlayerID())
					q.RevokeBaseAttr(core.AttrSummonSick)
					g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{q.ID()})
					g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: p.ID(), AttackerID: q.ID()}})
				case "other":
					q := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 8), a.PlayerID())
					q.RevokeBaseAttr(core.AttrSummonSick)
					g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{q.ID()})
				case "enters-attacking":
					g.FireEvent(core.GameEvent{Type: core.EvtEntersAttacking, SourceID: p.ID(), PlayerID: a.PlayerID()})
				}
				clone := g.Clone()
				g.RunStepWithPriority(core.EndCombat)
				want := tc.counters
				if role == "attack" || role == "block" || role == "removed-from-combat" || role == "both" {
					want--
				}
				if int(g.FindPermanent(p.ID()).Counters[core.P1P0]) != want || g.StackSize() != 0 {
					t.Fatalf("counter count = %d, want %d", g.FindPermanent(p.ID()).Counters[core.P1P0], want)
				}
				if int(clone.FindPermanent(p.ID()).Counters[core.P1P0]) != tc.counters {
					t.Fatal("clone shared counters")
				}
				clone.RunStepWithPriority(core.EndCombat)
				if int(clone.FindPermanent(p.ID()).Counters[core.P1P0]) != want {
					t.Fatal("clone lost history")
				}
				g.RunStepWithPriority(core.BeginCombat)
				g.RunStepWithPriority(core.EndCombat)
				if int(g.FindPermanent(p.ID()).Counters[core.P1P0]) != want {
					t.Fatal("prior combat history triggered again")
				}
			})
		}
	}
}

func TestRONEndCombatInterveningIdentity(t *testing.T) {
	for _, change := range []string{"none", "before-phase", "before-removed", "phase", "phase-cycle", "return", "control", "remove-abilities", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := clockworkTriggerCard(t, "ATQ", "Clockwork Avian")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			other := g.PutOnBattlefield(clockworkTriggerCard(t, "ATQ", "Clockwork Avian"), a.PlayerID())
			g.FireEvent(core.GameEvent{Type: core.EvtBeginCombat, PlayerID: a.PlayerID()})
			g.FireEvent(core.GameEvent{Type: core.EvtDeclaredAttacker, SourceID: p.ID(), PlayerID: a.PlayerID()})
			if change == "before-phase" {
				g.PhaseOut(p.ID())
			}
			if change == "before-removed" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndOfCombat, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if change == "before-phase" || change == "before-removed" {
				if g.StackSize() != 0 {
					t.Fatal("inactive source triggered")
				}
				return
			}
			if g.StackSize() != 1 || g.FindPermanent(p.ID()).Counters[core.P1P0] != 4 {
				t.Fatal("event check or early removal")
			}
			switch change {
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(card.ID())
				p = g.PutOnBattlefield(card, a.PlayerID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "remove-abilities":
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			clone.ResolveStack()
			g.ResolveStack()
			want := 3
			if change == "phase" || change == "return" {
				want = 4
			}
			if change == "copy" {
				want = 2
			}
			if int(g.FindPermanentIncludingPhased(p.ID()).Counters[core.P1P0]) != want || int(clone.FindPermanentIncludingPhased(p.ID()).Counters[core.P1P0]) != want || g.FindPermanent(other.ID()).Counters[core.P1P0] != 4 {
				t.Fatal("condition recheck, identity or instance isolation")
			}
		})
	}
}

func TestRONEndCombatPlain(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(AtTheEndOfCombat,ActionList([GainLife(Integer(1))]))`)
	g, a, b := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("end step matched end combat")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndOfCombat, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("opponent combat did not trigger")
	}
	g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatal("plain trigger lifetime or copy controller")
	}
}

func TestRONEndCombatInvalid(t *testing.T) {
	condition := `ACombatPermanentPassesFilter(ThisPermanent,Or([AttackedThisCombat,BlockedThisCombat]))`
	for _, rule := range []string{
		`TriggerA(AtTheEndOfCombat(),ActionList([GainLife(Integer(1))]))`,
		`TriggerI(AtTheEndOfCombat(),` + condition + `,ActionList([GainLife(Integer(1))]))`,
		`TriggerI(AtTheEndOfCombat,ACombatPermanentPassesFilter(HostPermanent,Or([AttackedThisCombat,BlockedThisCombat])),ActionList([GainLife(Integer(1))]))`,
		`TriggerI(AtTheEndOfCombat,ACombatPermanentPassesFilter(ThisPermanent(),Or([AttackedThisCombat,BlockedThisCombat])),ActionList([GainLife(Integer(1))]))`,
		`TriggerI(AtTheEndOfCombat,ACombatPermanentPassesFilter(ThisPermanent,Or([AttackedThisCombat,AttackedThisCombat])),ActionList([GainLife(Integer(1))]))`,
		`TriggerI(AtTheEndOfCombat,ACombatPermanentPassesFilter(ThisPermanent,And([AttackedThisCombat,BlockedThisCombat])),ActionList([GainLife(Integer(1))]))`,
		`TriggerI(AtTheEndOfCombat,ACombatPermanentPassesFilter(ThisPermanent,Or([AttackedThisTurn,BlockedThisCombat])),ActionList([GainLife(Integer(1))]))`,
		`TriggerI(AtTheEndOfCombat,` + condition + `,ActionList([GainLife(Trigger_ThatPlayer)]))`,
		`TriggerI(AtTheEndOfCombat,` + condition + `,ActionList([TapPermanent(Trigger_ThatPermanent)]))`,
		`TriggerI(AtTheEndOfCombat,` + condition + `,ActionList([GainLife(Integer(1))]),Extra)`,
	} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[%s])`, rule)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", rule)
		}
	}
}

func TestRONEndCombatFullRecordsRemainRejected(t *testing.T) {
	for _, tc := range []struct{ set, name, blocker string }{{"LEA", "Clockwork Beast", "PutCountersWithRestriction"}, {"LEB", "Clockwork Beast", "PutCountersWithRestriction"}, {"2ED", "Clockwork Beast", "PutCountersWithRestriction"}, {"3ED", "Clockwork Beast", "PutCountersWithRestriction"}, {"CED", "Clockwork Beast", "PutCountersWithRestriction"}, {"CEI", "Clockwork Beast", "PutCountersWithRestriction"}, {"ATQ", "Clockwork Avian", "PutCountersWithRestriction"}, {"LEG", "The Wretched", "GainControlOfEachPermanentUntil"}} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + tc.set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		var record string
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "`+tc.name+`"`) {
				record = line
			}
		}
		_, err = LoadFS(fstest.MapFS{"full.ron": {Data: []byte(record)}}, "*.ron")
		if err == nil || !strings.Contains(err.Error(), tc.blocker) {
			t.Fatalf("%s: %v", tc.name, err)
		}
	}
}

func TestRONEndCombatInventory(t *testing.T) {
	src := `Card(Name:"History",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:2),Rules:[TriggerI(AtTheEndOfCombat,ACombatPermanentPassesFilter(ThisPermanent,Or([AttackedThisCombat,BlockedThisCombat])),ActionList([GainLife(Integer(1))]))])`
	report, err := InspectFS(fstest.MapFS{"history.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatalf("inventory: %v", err)
	}
	roles := map[string]string{"AtTheEndOfCombat": "event", "ACombatPermanentPassesFilter": "trigger_condition", "ThisPermanent": "permanent_reference", "Or": "combat_history_filter", "AttackedThisCombat": "combat_history_filter", "BlockedThisCombat": "combat_history_filter"}
	for _, signature := range report.Signatures {
		if want, ok := roles[signature.Name]; ok && signature.Role != want {
			t.Fatalf("%s role %s, want %s", signature.Name, signature.Role, want)
		}
	}
}

func TestRONEndCombatStrictLoading(t *testing.T) {
	src := `Card(Name:"History",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:2),Rules:[TriggerI(AtTheEndOfCombat,ACombatPermanentPassesFilter(ThisPermanent,Or([BlockedThisCombat,AttackedThisCombat])),ActionList([GainLife(Integer(1))]))])`
	files := fstest.MapFS{"history.ron": {Data: []byte(src)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory := func() mage.Card {
		card, err := registry.CreateCard("History")
		if err != nil {
			t.Fatal(err)
		}
		return card
	}
	g, a, b := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	q := g.PutOnBattlefield(factory(), b.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtDeclaredAttacker, SourceID: p.ID(), PlayerID: a.PlayerID()})
	g.FireEvent(core.GameEvent{Type: core.EvtEndOfCombat, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("factory instances shared history")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 || g.FindPermanent(q.ID()) == nil {
		t.Fatal("reverse filter order or controller")
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(strings.ReplaceAll(src, `Name:"History"`, `Name:"Invalid"`), "AtTheEndOfCombat,", "AtTheEndOfCombat(),"))}
	if err = LoadInto(registry, files, "*.ron"); err == nil {
		t.Fatal("invalid collection accepted")
	}
	if len(registry.RegisteredCardNames()) != 1 || !registry.CardRegistered("History") {
		t.Fatal("rejection changed registry")
	}
}
