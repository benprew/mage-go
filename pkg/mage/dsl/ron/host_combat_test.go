package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func hostCombatFactory(t *testing.T, event, body string) mage.CardFactory {
	t.Helper()
	src := `Card(Name:"Watcher",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),TriggerA(` + event + `,ActionList([` + body + `]))])`
	nodes, err := Parse(strings.NewReader(src), "host-combat.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Watcher", factory()); err != nil {
		t.Fatal(err)
	}
	return factory
}

func TestRONHostCombatIdentity(t *testing.T) {
	for _, event := range []string{"WhenACreatureAttacks", "WhenACreatureBlocks", "WhenACreatureAttacksAndIsntBlocked"} {
		for _, scenario := range []string{"normal", "other", "reattach", "source-left", "source-control", "host-return", "host-phase", "phase-cycle", "source-phase", "removed", "copy", "two"} {
			t.Run(event+"/"+scenario, func(t *testing.T) {
				factory := hostCombatFactory(t, event+`(SinglePermanent(HostPermanent))`, `TapPermanent(Trigger_ThatCreature),CreatePermanentLayerEffectUntil(Trigger_ThatPermanent,[AdjustPT(1,0)],UntilEndOfTurn),GainLife(Integer(1))`)
				g, a, b := ronGame()
				card := mage.NewCreature("Host", "", 2, 8)
				host := g.PutOnBattlefield(card, b.PlayerID())
				other := g.PutOnBattlefield(mage.NewCreature("Other", "", 3, 8), b.PlayerID())
				source := g.PutOnBattlefield(factory(), a.PlayerID())
				g.Attach(source.ID(), host.ID())
				if scenario == "source-phase" {
					g.PhaseOut(source.ID())
				}
				if scenario == "removed" {
					g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
					g.ApplyContinuousEffects()
				}
				kind := core.EvtDeclaredAttacker
				if event == "WhenACreatureBlocks" {
					kind = core.EvtDeclaredBlocker
				}
				if event == "WhenACreatureAttacksAndIsntBlocked" {
					kind = core.EvtAttacksUnblocked
				}
				target := host
				if scenario == "other" {
					target = other
				}
				g.FireEvent(core.GameEvent{Type: kind, SourceID: target.ID(), PlayerID: b.PlayerID(), TargetID: a.PlayerID(), Flag: true})
				want := 1
				if scenario == "other" || scenario == "source-phase" || scenario == "removed" {
					want = 0
				}
				switch scenario {
				case "reattach":
					g.Attach(source.ID(), other.ID())
				case "source-left":
					g.RemoveFromBattlefield(source)
				case "source-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "host-return":
					g.RemoveFromBattlefield(host)
					host = g.PutOnBattlefield(card, b.PlayerID())
				case "host-phase":
					g.PhaseOut(host.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(host.ID()))
				case "two":
					g.Attach(source.ID(), other.ID())
					g.FireEvent(core.GameEvent{Type: kind, SourceID: other.ID(), PlayerID: b.PlayerID(), TargetID: a.PlayerID(), Flag: true})
					want++
				}
				g.PutTriggersOnStack()
				if g.StackSize() != want {
					t.Fatalf("stack=%d want=%d", g.StackSize(), want)
				}
				if scenario == "copy" {
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
					want++
				}
				clone := g.Clone()
				clone.ResolveStack()
				if p := g.FindPermanentIncludingPhased(host.ID()); p.Tapped {
					t.Fatal("clone changed original")
				}
				g.ResolveStack()
				for _, game := range []*mage.Game{g, clone} {
					wantA, wantB := 20+want, 20
					if scenario == "copy" {
						wantA, wantB = 21, 21
					}
					if game.GetPlayer(a.PlayerID()).Life() != wantA || game.GetPlayer(b.PlayerID()).Life() != wantB {
						t.Fatal("event controller or independent later action changed")
					}
					p := game.FindPermanentIncludingPhased(host.ID())
					affected := want > 0 && scenario != "host-return" && scenario != "host-phase"
					bonus := 0
					if affected {
						bonus = 1
						if scenario == "copy" {
							bonus = 2
						}
					}
					if p.Tapped != affected || p.CurrentPower(game) != 2+bonus {
						t.Fatalf("host tapped=%t power=%d", p.Tapped, p.CurrentPower(game))
					}
					q := game.FindPermanent(other.ID())
					if scenario == "two" {
						if !q.Tapped || q.CurrentPower(game) != 4 {
							t.Fatal("second event lost its visit")
						}
					} else if q.Tapped || q.CurrentPower(game) != 3 {
						t.Fatal("event followed new attachment")
					}
				}
			})
		}
	}
}

func TestRONHostCombatNative(t *testing.T) {
	for _, name := range []string{"Farrel's Mantle", "Imprison"} {
		for _, blocked := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", name, blocked), func(t *testing.T) {
				set := "LEG"
				if name == "Farrel's Mantle" {
					set = "FEM"
				}
				record := repositoryRecord(t, set, name, set+".lines.ron")
				var event Node
				for _, rule := range record.Fields["Rules"].Args {
					if rule.Name == "TriggerA" && (rule.Args[0].Name == "Or" || rule.Args[0].Name == "WhenACreatureAttacksAndIsntBlocked") {
						event = rule.Args[0]
					}
				}
				if event.Name == "" {
					t.Fatal("missing real event")
				}
				body, err := Parse(strings.NewReader(`ActionList([CreatePermanentLayerEffectUntil(Trigger_ThatCreature,[AdjustPT(1,0)],UntilEndOfTurn)])`), "body.ron")
				if err != nil {
					t.Fatal(err)
				}
				rule := Node{Kind: Tuple, Name: "TriggerA", Args: []Node{event, body[0]}}
				enchant := record.Fields["Rules"].Args[0]
				rules := record.Fields["Rules"]
				rules.Args = []Node{enchant, rule}
				record.Fields["Rules"] = rules
				factory, err := Compile(record)
				if err != nil {
					t.Fatal(err)
				}
				a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
				g := mage.NewGame(a, b)
				g.SetTurn(1)
				host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 8), a.PlayerID())
				host.RevokeBaseAttr(core.AttrSummonSick)
				blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 0, 8), b.PlayerID())
				aura := g.PutOnBattlefield(factory(), a.PlayerID())
				g.Attach(aura.ID(), host.ID())
				a.SetAttackers(1, []string{"Host"})
				if blocked {
					b.SetBlockers(1, map[string]string{"Blocker": "Host"})
				}
				g.RunStepWithPriority(core.DeclareAttackers)
				early := 2
				if name == "Imprison" {
					early = 3
				}
				if host.CurrentPower(g) != early {
					t.Fatal("wrong attacker timing")
				}
				g.RunStepWithPriority(core.DeclareBlockers)
				final := early
				if name == "Farrel's Mantle" && !blocked {
					final++
				}
				if host.CurrentPower(g) != final {
					t.Fatal("wrong unblocked timing")
				}
				if blocked && !g.GetCombat().IsBlocking(blocker.ID()) {
					t.Fatal("block rejected")
				}
			})
		}
	}
}

func TestRONHostCombatUnionBlocksOnce(t *testing.T) {
	event := `Or([WhenACreatureAttacks(SinglePermanent(HostPermanent)),WhenACreatureBlocks(SinglePermanent(HostPermanent))])`
	g, a, b := ronGame()
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 8, mage.WithKeyword(core.CanBlockAny)), b.PlayerID())
	aura := g.PutOnBattlefield(hostCombatFactory(t, event, `GainLife(Integer(1))`)(), b.PlayerID())
	g.Attach(aura.ID(), host.ID())
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 8), a.PlayerID())
	first.RevokeBaseAttr(core.AttrSummonSick)
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 8), a.PlayerID())
	second.RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{first.ID(), second.ID()})
	g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: host.ID(), AttackerID: first.ID()}, {BlockerID: host.ID(), AttackerID: second.ID()}})
	if !g.GetCombat().IsBlocking(host.ID()) {
		t.Fatal("block declaration rejected")
	}
	if b.Life() != 21 {
		t.Fatal("block event repeated per attacker")
	}
}

func TestRONHostCombatInvalid(t *testing.T) {
	valid := `TriggerA(WhenACreatureAttacks(SinglePermanent(HostPermanent)),ActionList([TapPermanent(Trigger_ThatCreature)]))`
	for _, rule := range []string{
		strings.Replace(valid, "WhenACreatureAttacks", "WhenACreatureBecomesBlocked", 1),
		strings.Replace(valid, "HostPermanent", "HostPermanent()", 1),
		strings.Replace(valid, "HostPermanent", "Ref_TargetPermanent", 1),
		strings.Replace(valid, "Trigger_ThatCreature", "Trigger_ThatCreature()", 1),
		strings.Replace(valid, "Trigger_ThatCreature", "Trigger_DefendingPlayer", 1),
		strings.Replace(valid, "WhenACreatureAttacks(SinglePermanent(HostPermanent))", `Or([WhenACreatureAttacks(SinglePermanent(HostPermanent)),WhenACreatureBlocks(SinglePermanent(ThisPermanent))])`, 1),
		strings.Replace(valid, "WhenACreatureAttacks(SinglePermanent(HostPermanent))", `Or([WhenACreatureAttacks(SinglePermanent(HostPermanent)),WhenACreatureAttacks(SinglePermanent(HostPermanent))])`, 1),
		valid + `,Activated(PayMana([]),ActionList([TapPermanent(Trigger_ThatCreature)]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONHostCombatImprisonClause(t *testing.T) {
	for _, attacking := range []bool{false, true} {
		for _, payment := range []string{"paid", "declined", "short"} {
			t.Run(fmt.Sprintf("%t/%s", attacking, payment), func(t *testing.T) {
				record := repositoryRecord(t, "LEG", "Imprison", "LEG.lines.ron")
				rules := record.Fields["Rules"]
				rules.Args = []Node{rules.Args[0], rules.Args[2]}
				record.Fields["Rules"] = rules
				factory, err := Compile(record)
				if err != nil {
					t.Fatal(err)
				}
				a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: payment != "declined"}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				g.SetTurn(1)
				host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 8), b.PlayerID())
				host.RevokeBaseAttr(core.AttrSummonSick)
				other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 8), a.PlayerID())
				other.RevokeBaseAttr(core.AttrSummonSick)
				aura := g.PutOnBattlefield(factory(), a.PlayerID())
				g.Attach(aura.ID(), host.ID())
				if payment != "short" {
					a.ManaPool().Add(core.Colorless, 1)
				}
				if attacking {
					g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{host.ID()})
				} else {
					g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{other.ID()})
					g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: host.ID(), AttackerID: other.ID()}})
				}
				g.PutTriggersOnStack()
				g.ResolveStack()
				paid := payment == "paid"
				if (g.FindPermanent(aura.ID()) != nil) != paid {
					t.Fatal("wrong payment branch")
				}
				if paid {
					if !host.Tapped || g.GetCombat().IsAttacking(host.ID()) || g.GetCombat().IsBlocking(host.ID()) {
						t.Fatal("paid host stayed in combat")
					}
					if a.ManaPool().TotalMana() != 0 {
						t.Fatal("payment not spent")
					}
				} else if !attacking && !g.GetCombat().IsBlocking(host.ID()) {
					t.Fatal("unpaid branch removed blocker")
				}
				if payment == "declined" && a.ManaPool().TotalMana() != 1 {
					t.Fatal("decline spent mana")
				}
			})
		}
	}
}

func TestRONHostCombatDelayedAndFactories(t *testing.T) {
	factory := hostCombatFactory(t, `WhenACreatureAttacks(SinglePermanent(HostPermanent))`, `CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([TapPermanent(Trigger_ThatCreature)]))`)
	g, a, b := ronGame()
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 8), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 8), b.PlayerID())
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	g.Attach(source.ID(), first.ID())
	sibling := g.PutOnBattlefield(factory(), b.PlayerID())
	g.Attach(sibling.ID(), second.ID())
	for _, p := range []*mage.Permanent{first, second} {
		g.FireEvent(core.GameEvent{Type: core.EvtDeclaredAttacker, SourceID: p.ID()})
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("factory event bindings collided")
	}
	g.ResolveStack()
	g.Attach(source.ID(), second.ID())
	clone := g.Clone()
	g.RemoveFromBattlefield(first)
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if !g.FindPermanent(second.ID()).Tapped || clone.FindPermanent(second.ID()).Tapped {
		t.Fatalf("delayed second tapped=%t clone=%t stack=%d", g.FindPermanent(second.ID()).Tapped, clone.FindPermanent(second.ID()).Tapped, g.StackSize())
	}
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if !clone.FindPermanent(first.ID()).Tapped || !clone.FindPermanent(second.ID()).Tapped {
		t.Fatal("delay lost exact captured host")
	}
}

func TestRONHostCombatCompleteRecordsStayRejected(t *testing.T) {
	for _, tc := range []struct{ set, name, blocker string }{{"LEG", "Imprison", "supply one artifact source"}} {
		_, err := Compile(repositoryRecord(t, tc.set, tc.name, tc.set+".lines.ron"))
		if err == nil || !strings.Contains(err.Error(), tc.blocker) {
			t.Fatalf("%s: %v", tc.name, err)
		}
	}
}
