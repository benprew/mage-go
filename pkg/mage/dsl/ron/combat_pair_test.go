package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCombatPairNative(t *testing.T) {
	for _, name := range []string{"Cockatrice", "Thicket Basilisk"} {
		for _, attacking := range []bool{false, true} {
			for _, wall := range []bool{false, true} {
				t.Run(name+"/"+boolName(attacking)+"/"+boolName(wall), func(t *testing.T) {
					g, a, b := ronGame()
					card := repositoryCard(t, "LEB", name)
					if err := mage.ValidateCardDefinition(name, card); err != nil {
						t.Fatal(err)
					}
					p := g.PutOnBattlefield(card, a.PlayerID())
					otherCard := mage.NewCreature("Other", "", 0, 8)
					if wall {
						otherCard = mage.NewCreature("Other", "", 0, 8, mage.WithSubTypes("Wall"))
					}
					q := g.PutOnBattlefield(otherCard, b.PlayerID())
					p.RevokeBaseAttr(core.AttrSummonSick)
					q.RevokeBaseAttr(core.AttrSummonSick)
					if name == "Cockatrice" {
						q.GrantBaseAttr(core.Flying)
					}
					if attacking {
						g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
						g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: q.ID(), AttackerID: p.ID()}})
					} else {
						g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{q.ID()})
						g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: p.ID(), AttackerID: q.ID()}})
					}
					g.PutTriggersOnStack()
					if g.StackSize() != 0 {
						t.Fatal("native blocker helper did not resolve triggers")
					}
					g.ResolveStack()
					if g.FindPermanent(q.ID()) == nil {
						t.Fatal("destroyed before end of combat")
					}
					g.ExecuteCombatDamage()
					if g.FindPermanent(q.ID()) == nil {
						t.Fatal("destroyed during damage")
					}
					g.RunStepWithPriority(core.EndCombat)
					if (g.FindPermanent(q.ID()) != nil) != wall {
						t.Fatal("wrong delayed destruction")
					}
					if g.FindPermanent(p.ID()) == nil {
						t.Fatal("destroyed source")
					}
				})
			}
		}
	}
}

func boolName(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func TestRONCombatPairIdentity(t *testing.T) {
	for _, scenario := range []string{"normal", "return", "phase", "phase-cycle", "control", "source-leaves", "regenerate", "copy", "removed", "becomes-wall"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEB", "Thicket Basilisk")
			p := g.PutOnBattlefield(card, a.PlayerID())
			target := mage.NewCreature("Other", "", 0, 8)
			q := g.PutOnBattlefield(target, b.PlayerID())
			p.RevokeBaseAttr(core.AttrSummonSick)
			q.RevokeBaseAttr(core.AttrSummonSick)
			if scenario == "removed" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
			}
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
			g.FireEvent(core.GameEvent{Type: core.EvtCreatureBlockedBy, SourceID: p.ID(), TargetID: q.ID(), PlayerID: a.PlayerID()})
			if scenario == "source-leaves" {
				g.RemoveFromBattlefield(p)
			}
			g.PutTriggersOnStack()
			want := 1
			if scenario == "removed" {
				want = 0
			}
			if g.StackSize() != want {
				t.Fatalf("stack=%d want=%d", g.StackSize(), want)
			}
			if scenario == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			switch scenario {
			case "return":
				g.RemoveFromBattlefield(q)
				g.PutOnBattlefield(target, b.PlayerID())
			case "phase":
				g.PhaseOut(q.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(q.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: q.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "becomes-wall":
				g.AddContinuousEffect(mage.AddSubtypes(q.ID(), mage.SubtypeCreature, core.Indefinite, "Wall"))
				g.ApplyContinuousEffects()
			}
			g.ResolveStack()
			if scenario == "regenerate" || scenario == "copy" {
				g.AddRegenerationShield(q.ID())
			}
			clone := g.Clone()
			clone.RunStepWithPriority(core.EndCombat)
			if scenario != "phase" && g.FindPermanent(q.ID()) == nil {
				t.Fatal("clone changed original")
			}
			g.RunStepWithPriority(core.EndCombat)
			for _, game := range []*mage.Game{g, clone} {
				survives := scenario == "return" || scenario == "regenerate" || scenario == "removed"
				if (game.FindPermanent(q.ID()) != nil) != survives {
					t.Fatalf("wrong presence in %s", scenario)
				}
				if scenario == "phase" {
					game.PhaseIn([]uuid.UUID{q.ID()})
					if game.FindPermanent(q.ID()) == nil {
						t.Fatal("destroyed phased target")
					}
				}
			}
		})
	}
}

func TestRONCombatPairInvalid(t *testing.T) {
	event := `WhenACreatureBlocksACreature(SinglePermanent(ThisPermanent),And([IsCardtype(Creature),IsNonCreatureType(Wall)]))`
	body := `ActionList([CreateFutureTrigger(AtTheNextEndOfCombat,ActionList([DestroyPermanent(Trigger_ThatOtherCreature)]))])`
	for _, rule := range []string{
		`TriggerA(` + strings.Replace(event, "ThisPermanent", "HostPermanent", 1) + `,` + body + `)`,
		`TriggerA(` + strings.Replace(event, "SinglePermanent(ThisPermanent)", "IsCardtype(Creature)", 1) + `,` + body + `)`,
		`TriggerA(` + strings.Replace(event, "ThisPermanent", "ThisPermanent()", 1) + `,` + body + `)`,
		`TriggerA(` + strings.Replace(event, "Wall", "Island", 1) + `,` + body + `)`,
		`TriggerA(` + strings.Replace(event, "IsNonCreatureType(Wall)", "IsNonCreatureType(Wall),IsNonCreatureType(Goblin)", 1) + `,` + body + `)`,
		`TriggerA(` + event + `,` + strings.Replace(body, "Trigger_ThatOtherCreature", "Trigger_TheBlockingCreature", 1) + `)`,
		`TriggerA(` + event + `,` + strings.Replace(body, "Trigger_ThatOtherCreature", "Trigger_ThatOtherCreature()", 1) + `)`,
		`TriggerA(` + event + `,` + body + `),Activated(PayMana([]),ActionList([DestroyPermanent(Trigger_ThatOtherCreature)]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:4),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONCombatPairIndependentOpponents(t *testing.T) {
	g, a, b := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "LEB", "Thicket Basilisk"), a.PlayerID())
	p.RevokeBaseAttr(core.AttrSummonSick)
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 0, 8), b.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 0, 8), b.PlayerID())
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
	g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: first.ID(), AttackerID: p.ID()}, {BlockerID: second.ID(), AttackerID: p.ID()}})
	returned := second.Card
	g.RemoveFromBattlefield(second)
	g.PutOnBattlefield(returned, b.PlayerID())
	g.RunStepWithPriority(core.EndCombat)
	if g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) == nil {
		t.Fatal("opposing event bindings crossed or followed a returned card")
	}
}

func TestRONCombatPairCasting(t *testing.T) {
	for _, name := range []string{"Cockatrice", "Thicket Basilisk"} {
		g, a, _ := ronGame()
		card := repositoryCard(t, "LEB", name)
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
			t.Fatal("unpaid casting did not roll back")
		}
		a.ManaPool().Add(core.Green, 2)
		a.ManaPool().Add(core.Colorless, 3)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		p := g.FindPermanent(card.ID())
		if p == nil || p.CurrentPower(g) != 2 || p.CurrentToughness(g) != 4 || p.HasKeyword(core.Flying) != (name == "Cockatrice") {
			t.Fatal("wrong real RON characteristics")
		}
		if a.ManaPool().TotalMana() != 0 {
			t.Fatal("wrong casting cost")
		}
		other := repositoryCard(t, "LEB", name)
		if card.Abilities()[0] == other.Abilities()[0] {
			t.Fatal("factory shared ability")
		}
	}
}

func TestRONCombatPairFactoryBindings(t *testing.T) {
	g, a, b := ronGame()
	first := g.PutOnBattlefield(repositoryCard(t, "LEB", "Thicket Basilisk"), a.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "LEB", "Thicket Basilisk"), a.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("First target", "", 0, 8), b.PlayerID())
	r := g.PutOnBattlefield(mage.NewCreature("Second target", "", 0, 8), b.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtCreatureBlockedBy, SourceID: first.ID(), TargetID: q.ID(), PlayerID: a.PlayerID()})
	g.FireEvent(core.GameEvent{Type: core.EvtCreatureBlockedBy, SourceID: second.ID(), TargetID: r.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("events did not stay separate")
	}
	g.ResolveStack()
	g.AddRegenerationShield(q.ID())
	g.RunStepWithPriority(core.EndCombat)
	if g.FindPermanent(q.ID()) == nil || g.FindPermanent(r.ID()) != nil {
		t.Fatal("bindings crossed factory instances")
	}
}

func TestRONCombatPairFilterShapes(t *testing.T) {
	for _, event := range []struct {
		name, ref string
		kind      core.EventType
	}{
		{"WhenACreatureBlocksACreature", "Trigger_ThatOtherCreature", core.EvtDeclaredBlocker},
		{"WhenACreatureBecomesBlockedByACreature", "Trigger_TheBlockingCreature", core.EvtCreatureBlockedBy},
	} {
		for _, filter := range []string{`IsCardtype(Creature)`, `And([IsCardtype(Creature)])`, `And([IsCardtype(Creature),IsNonCreatureType(Goblin)])`} {
			for _, goblin := range []bool{false, true} {
				g, a, b := ronGame()
				src := `Card(Name:"Pair",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:4),Rules:[TriggerA(` + event.name + `(SinglePermanent(ThisPermanent),` + filter + `),ActionList([DestroyPermanent(` + event.ref + `)]))])`
				nodes, err := Parse(strings.NewReader(src), "pair.ron")
				if err != nil {
					t.Fatal(err)
				}
				factory, err := Compile(nodes[0])
				if err != nil {
					t.Fatal(err)
				}
				card := factory()
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				p := g.PutOnBattlefield(card, a.PlayerID())
				var options []mage.CardOption
				if goblin {
					options = append(options, mage.WithSubTypes("Goblin"))
				}
				q := g.PutOnBattlefield(mage.NewCreature("Other", "", 0, 8, options...), b.PlayerID())
				g.FireEvent(core.GameEvent{Type: event.kind, SourceID: p.ID(), TargetID: q.ID(), PlayerID: a.PlayerID()})
				g.PutTriggersOnStack()
				g.ResolveStack()
				survives := goblin && strings.Contains(filter, "IsNonCreatureType")
				if (g.FindPermanent(q.ID()) != nil) != survives {
					t.Fatalf("%s %s goblin=%t", event.name, filter, goblin)
				}
			}
		}
	}
}
