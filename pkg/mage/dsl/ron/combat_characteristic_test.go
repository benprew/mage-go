package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCombatCharacteristicShapes(t *testing.T) {
	for _, host := range []bool{false, true} {
		for _, direction := range []string{"blocking", "blocked", "union"} {
			for _, filter := range []string{
				`IsCreatureType(Wall)`, `And([IsCreatureType(Wall),IsCardtype(Creature)])`,
				`And([IsCardtype(Creature),ToughnessIs(LessThanOrEqualTo(Integer(3)))])`,
				`And([IsCreatureType(Wall),IsNonCreatureType(Goblin),ToughnessIs(EqualTo(Integer(3)))])`,
				`And([IsCreatureType(Wall),ToughnessIs(GreaterThanOrEqualTo(Integer(3)))])`,
			} {
				t.Run(boolName(host)+"/"+direction+"/"+filter, func(t *testing.T) {
					for _, matching := range []bool{false, true} {
						g, a, b := ronGame()
						subject := "ThisPermanent"
						if host {
							subject = "HostPermanent"
						}
						blocking := `WhenACreatureBlocksACreature(SinglePermanent(` + subject + `),` + filter + `)`
						blocked := strings.Replace(blocking, "WhenACreatureBlocksACreature", "WhenACreatureBecomesBlockedByACreature", 1)
						event, ref, kind := blocking, "Trigger_ThatOtherCreature", core.EvtDeclaredBlocker
						switch direction {
						case "blocked":
							event, ref, kind = blocked, "Trigger_TheBlockingCreature", core.EvtCreatureBlockedBy
						case "union":
							event, kind = `Or([`+blocked+`,`+blocking+`])`, core.EvtCombatPair
						}
						rule := `TriggerA(` + event + `,ActionList([CreateFutureTrigger(AtTheNextEndOfCombat,ActionList([DestroyPermanent(` + ref + `)]))]))`
						factory := characteristicFactory(t, host, rule)
						c := factory()
						if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
							t.Fatal(err)
						}
						p := g.PutOnBattlefield(c, a.PlayerID())
						if host {
							p = g.PutOnBattlefield(mage.NewCreature("Host", "", 0, 8), b.PlayerID())
							g.Attach(c.ID(), p.ID())
						}
						toughness, subtype := 3, "Wall"
						if !matching {
							if strings.Contains(filter, "IsCreatureType") {
								subtype = "Goblin"
							} else {
								toughness = 4
							}
						}
						q := g.PutOnBattlefield(mage.NewCreature("Other", "", 0, toughness, mage.WithSubTypes(subtype), mage.WithKeyword(core.Shroud)), b.PlayerID())
						g.FireEvent(core.GameEvent{Type: kind, SourceID: p.ID(), TargetID: q.ID(), PlayerID: b.PlayerID()})
						g.PutTriggersOnStack()
						want := 0
						if matching {
							want = 1
						}
						if g.StackSize() != want {
							t.Fatalf("stack=%d want=%d", g.StackSize(), want)
						}
						g.ResolveStack()
						if g.FindPermanent(q.ID()) == nil {
							t.Fatal("early destruction")
						}
						g.RunStepWithPriority(core.EndCombat)
						if (g.FindPermanent(q.ID()) == nil) != matching {
							t.Fatal("wrong filter or capture")
						}
					}
				})
			}
		}
	}
}

func characteristicFactory(t *testing.T, host bool, rule string) mage.CardFactory {
	t.Helper()
	typeline := `(Supertypes:[],Cardtypes:[Creature],Subtypes:[])`
	pt := `,CardPT:(Power:0,Toughness:8)`
	if host {
		typeline, pt = `(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura])`, ""
		rule = `EnchantPermanent(IsCardtype(Creature)),` + rule
	}
	nodes, err := Parse(strings.NewReader(`Card(Name:"Characteristic",Typeline:`+typeline+pt+`,Rules:[`+rule+`])`), "characteristic.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	return factory
}

func TestRONCombatCharacteristicInvalid(t *testing.T) {
	for _, filter := range []string{
		`IsCreatureType(Island)`, `IsCreatureType(Wall())`, `IsCreatureType`,
		`IsCreatureType(Wall,Orc)`, `And([IsCreatureType(Wall),IsCreatureType(Orc)])`,
		`And([IsCardtype(Creature),IsCreatureType(Wall),IsCreatureType(Wall)])`,
		`And([IsCardtype(Creature),ToughnessIs(EqualTo(Integer(3))),ToughnessIs(EqualTo(Integer(4)))])`,
		`ToughnessIs(LessThanOrEqualTo(Integer(3)))`,
		`And([IsCardtype(Creature),ToughnessIs(LessThan(Integer(3)))])`,
		`And([IsCardtype(Creature),ToughnessIs(EqualTo(ValueX))])`,
		`And([IsCardtype(Creature),ToughnessIs(EqualTo(Integer("3")))])`,
		`And([IsCardtype(Creature),ToughnessIs(EqualTo(Integer(3),Integer(4)))])`,
		`And([IsCardtype(Creature),ToughnessIs()])`, `And([])`,
	} {
		rule := `TriggerA(WhenACreatureBlocksACreature(SinglePermanent(ThisPermanent),` + filter + `),ActionList([GainLife(Integer(1))]))`
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", filter)
		}
	}
}

func TestRONCombatCharacteristicConsumers(t *testing.T) {
	for _, tc := range []struct {
		set, name, blocker string
		rule               int
	}{
		{"ATQ", "Battering Ram", "Trigger_TheAttackingCreature", 1},
		{"FEM", "Dwarven Soldier", "combat pair union", 0},
		{"LEG", "Infinite Authority", "Trigger_ThatCreature", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := hostUntapRecord(t, tc.set, tc.name)
			if _, err := Compile(n); err == nil || !strings.Contains(err.Error(), tc.blocker) {
				t.Fatalf("complete record boundary: %v", err)
			}
			event := n.Fields["Rules"].Args[tc.rule].Args[0]
			if event.Name == "Or" {
				event = event.Args[0].Args[0]
			}
			filter := event.Args[1]
			compiled, err := compileCombatPairFilter(filter)
			if err != nil {
				t.Fatal(err)
			}
			g, a, _ := ronGame()
			subtype := "Wall"
			if tc.name == "Dwarven Soldier" {
				subtype = "Orc"
			}
			p := g.PutOnBattlefield(mage.NewCreature("Matching", "", 0, 3, mage.WithSubTypes(subtype)), a.PlayerID())
			q := g.PutOnBattlefield(mage.NewCreature("Other", "", 0, 4, mage.WithSubTypes("Human")), a.PlayerID())
			if !compiled.Match(p, g) || compiled.Match(q, g) {
				t.Fatal("real record filter")
			}
			if compiled.Match(g.PutOnBattlefield(mage.NewArtifact("Noncreature", "", mage.WithSubTypes(subtype)), a.PlayerID()), g) {
				t.Fatal("noncreature matched")
			}
		})
	}
}

func TestRONCombatCharacteristicIdentity(t *testing.T) {
	for _, filter := range []string{`IsCreatureType(Wall)`, `And([IsCardtype(Creature),ToughnessIs(LessThanOrEqualTo(Integer(3)))])`} {
		for _, change := range []string{"characteristics", "return", "phase", "phase-cycle", "source-return", "host-move", "copy", "regenerate", "indestructible", "control"} {
			t.Run(filter+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				factory := characteristicFactory(t, true, `TriggerA(WhenACreatureBlocksACreature(SinglePermanent(HostPermanent),`+filter+`),ActionList([CreateFutureTrigger(AtTheNextEndOfCombat,ActionList([DestroyPermanent(Trigger_ThatOtherCreature)]))]))`)
				c := factory()
				source := g.PutOnBattlefield(c, a.PlayerID())
				host := g.PutOnBattlefield(mage.NewCreature("Host", "", 0, 8), a.PlayerID())
				g.Attach(source.ID(), host.ID())
				victim := mage.NewCreature("Victim", "", 0, 3, mage.WithSubTypes("Wall"))
				q := g.PutOnBattlefield(victim, b.PlayerID())
				g.FireEvent(core.GameEvent{Type: core.EvtDeclaredBlocker, SourceID: host.ID(), TargetID: q.ID()})
				switch change {
				case "characteristics":
					g.AddContinuousEffect(mage.SetSubtypes(q.ID(), mage.SubtypeCreature, core.Indefinite, "Goblin"))
					g.AddContinuousEffect(mage.TemporaryBoost(q.ID(), 0, 5))
					g.ApplyContinuousEffects()
				case "return":
					g.RemoveFromBattlefield(q)
					g.PutOnBattlefield(victim, b.PlayerID())
				case "phase":
					g.PhaseOut(q.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(q.ID()))
				case "source-return":
					g.RemoveFromBattlefield(source)
					g.PutOnBattlefield(c, b.PlayerID())
				case "host-move":
					other := g.PutOnBattlefield(mage.NewCreature("Other host", "", 0, 8), a.PlayerID())
					g.Attach(source.ID(), other.ID())
				case "indestructible":
					g.MutablePermanent(q.ID()).GrantBaseAttr(core.Indestructible)
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: q.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				}
				g.PutTriggersOnStack()
				if g.StackSize() != 1 {
					t.Fatal("pending event was lost")
				}
				if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				clone := g.Clone()
				for _, game := range []*mage.Game{clone, g} {
					game.ResolveStack()
					if change == "regenerate" || change == "copy" {
						game.AddRegenerationShield(q.ID())
					}
					game.RunStepWithPriority(core.EndCombat)
					wantLive := change == "return" || change == "regenerate" || change == "indestructible"
					if (game.FindPermanent(q.ID()) != nil) != wantLive {
						t.Fatal("wrong identity or destruction result")
					}
					if change == "phase" {
						game.PhaseIn([]uuid.UUID{q.ID()})
						if game.FindPermanent(q.ID()) == nil {
							t.Fatal("phased victim destroyed")
						}
					}
					if game == clone && g.StackSize() == 0 {
						t.Fatal("clone changed live stack")
					}
				}
			})
		}
	}
}

func TestRONCombatCharacteristicNativeClauses(t *testing.T) {
	for _, name := range []string{"Battering Ram", "Infinite Authority"} {
		for _, attacking := range []bool{false, true} {
			if name == "Battering Ram" && !attacking {
				continue
			}
			for _, matching := range []bool{false, true} {
				t.Run(name+"/"+boolName(attacking)+"/"+boolName(matching), func(t *testing.T) {
					set := "LEG"
					if name == "Battering Ram" {
						set = "ATQ"
					}
					record := hostUntapRecord(t, set, name)
					event := record.Fields["Rules"].Args[1].Args[0]
					body, err := Parse(strings.NewReader(`ActionList([GainLife(Integer(1))])`), "probe.ron")
					if err != nil {
						t.Fatal(err)
					}
					build, err := compileCombatPairTrigger(event, body[0], scope{permanent: true, aura: name == "Infinite Authority"})
					if err != nil {
						t.Fatal(err)
					}
					g, a, b := ronGame()
					c := mage.NewCreature("Event clause", "", 0, 8, build())
					if name == "Infinite Authority" {
						c = mage.NewEnchantment("Event clause", "", mage.WithSubTypes("Aura"), mage.WithCastTarget(mage.TargetPermanent(mage.IsCreature)), build())
					}
					if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
						t.Fatal(err)
					}
					p := g.PutOnBattlefield(c, a.PlayerID())
					if name == "Infinite Authority" {
						p = g.PutOnBattlefield(mage.NewCreature("Host", "", 0, 8), a.PlayerID())
						g.Attach(c.ID(), p.ID())
					}
					opponents := make([]*mage.Permanent, 0, 2)
					for range 2 {
						subtype, toughness := "Wall", 3
						if !matching {
							subtype, toughness = "Goblin", 4
						}
						q := g.PutOnBattlefield(mage.NewCreature("Opponent", "", 0, toughness, mage.WithSubTypes(subtype)), b.PlayerID())
						q.RevokeBaseAttr(core.AttrSummonSick)
						opponents = append(opponents, q)
					}
					p.RevokeBaseAttr(core.AttrSummonSick)
					if attacking {
						g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
						g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: opponents[0].ID(), AttackerID: p.ID()}, {BlockerID: opponents[1].ID(), AttackerID: p.ID()}})
					} else {
						g.MutablePermanent(p.ID()).GrantBaseAttr(core.CanBlockAdditional)
						g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{opponents[0].ID(), opponents[1].ID()})
						g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: p.ID(), AttackerID: opponents[0].ID()}, {BlockerID: p.ID(), AttackerID: opponents[1].ID()}})
					}
					want := 20
					if matching {
						want = 22
					}
					if a.Life() != want || b.Life() != 20 {
						t.Fatalf("native clause life=%d want=%d", a.Life(), want)
					}
				})
			}
		}
	}
}

func TestRONCombatCharacteristicDetectionAndInstances(t *testing.T) {
	for _, filter := range []string{`IsCreatureType(Orc)`, `And([IsCardtype(Creature),ToughnessIs(LessThanOrEqualTo(Integer(3)))])`} {
		g, a, b := ronGame()
		factory := characteristicFactory(t, false, `TriggerA(WhenACreatureBlocksACreature(SinglePermanent(ThisPermanent),`+filter+`),ActionList([DestroyPermanent(Trigger_ThatOtherCreature)]))`)
		p := g.PutOnBattlefield(factory(), a.PlayerID())
		r := g.PutOnBattlefield(factory(), b.PlayerID())
		q := g.PutOnBattlefield(mage.NewCreature("Opponent", "", 0, 4, mage.WithSubTypes("Goblin")), b.PlayerID())
		fire := func(source, target *mage.Permanent) {
			g.FireEvent(core.GameEvent{Type: core.EvtDeclaredBlocker, SourceID: source.ID(), TargetID: target.ID()})
		}
		fire(p, q)
		g.AddContinuousEffect(mage.AddSubtypes(q.ID(), mage.SubtypeCreature, core.Indefinite, "Orc"))
		g.AddContinuousEffect(mage.TemporaryBoost(q.ID(), 0, -1))
		g.ApplyContinuousEffects()
		g.PutTriggersOnStack()
		if g.StackSize() != 0 {
			t.Fatal("later characteristics caused a trigger")
		}
		phase := g.PhaseOut(q.ID())
		fire(p, q)
		g.PhaseIn(phase)
		g.PutTriggersOnStack()
		if g.StackSize() != 0 {
			t.Fatal("phased opponent triggered")
		}
		fire(p, q)
		s := g.PutOnBattlefield(mage.NewCreature("Second opponent", "", 0, 3, mage.WithSubTypes("Orc")), a.PlayerID())
		fire(r, s)
		g.PutTriggersOnStack()
		if g.StackSize() != 2 {
			t.Fatal("independent events missing")
		}
		g.AddRegenerationShield(q.ID())
		clone := g.Clone()
		clone.ResolveStack()
		if g.StackSize() != 2 || g.FindPermanent(s.ID()) == nil {
			t.Fatal("clone changed original")
		}
		g.ResolveStack()
		for _, game := range []*mage.Game{g, clone} {
			if game.FindPermanent(q.ID()) == nil || game.FindPermanent(s.ID()) != nil {
				t.Fatal("factory or event bindings crossed")
			}
		}
	}
}
