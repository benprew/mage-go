package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCombatPairReferenceCards(t *testing.T) {
	for _, name := range []string{"Battering Ram", "Infernal Medusa"} {
		for _, attacking := range []bool{false, true} {
			for _, wall := range []bool{false, true} {
				t.Run(name+"/"+boolName(attacking)+"/"+boolName(wall), func(t *testing.T) {
					g, a, b := ronGame()
					set := "LEG"
					if name == "Battering Ram" {
						set = "ATQ"
					}
					card := repositoryCard(t, set, name)
					if err := mage.ValidateCardDefinition(name, card); err != nil {
						t.Fatal(err)
					}
					p := g.PutOnBattlefield(card, a.PlayerID())
					var opts []mage.CardOption
					if wall {
						opts = append(opts, mage.WithSubTypes("Wall"))
					}
					q := g.PutOnBattlefield(mage.NewCreature("Opponent", "", 0, 8, opts...), b.PlayerID())
					p.RevokeBaseAttr(core.AttrSummonSick)
					q.RevokeBaseAttr(core.AttrSummonSick)
					if attacking {
						g.RunStepWithPriority(core.BeginCombat)
						if name == "Battering Ram" && !g.FindPermanent(p.ID()).HasKeyword(core.Banding) {
							t.Fatal("missing combat banding")
						}
						g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
						g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: q.ID(), AttackerID: p.ID()}})
					} else {
						g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{q.ID()})
						g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: p.ID(), AttackerID: q.ID()}})
					}
					g.ResolveStack()
					g.ExecuteCombatDamage()
					if g.FindPermanent(q.ID()) == nil {
						t.Fatal("destroyed before end of combat")
					}
					g.RunStepWithPriority(core.EndCombat)
					want := attacking && wall
					if name == "Infernal Medusa" {
						want = !attacking || !wall
					}
					if (g.FindPermanent(q.ID()) == nil) != want || g.FindPermanent(p.ID()) == nil {
						t.Fatal("wrong combat participant destroyed")
					}
					g.RunStepWithPriority(core.PostcombatMain)
					if name == "Battering Ram" && g.FindPermanent(p.ID()).HasKeyword(core.Banding) {
						t.Fatal("banding did not expire")
					}
				})
			}
		}
	}
}

func TestRONCombatPairReferenceIdentity(t *testing.T) {
	for _, name := range []string{"Battering Ram", "Infernal Medusa"} {
		for _, change := range []string{"normal", "return", "phase", "phase-cycle", "source-leaves", "subtype", "control", "regenerate", "indestructible", "copy"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				set := "LEG"
				if name == "Battering Ram" {
					set = "ATQ"
				}
				p := g.PutOnBattlefield(repositoryCard(t, set, name), a.PlayerID())
				var opts []mage.CardOption
				if name == "Battering Ram" {
					opts = append(opts, mage.WithSubTypes("Wall"))
				}
				card := mage.NewCreature("Opponent", "", 0, 8, opts...)
				q := g.PutOnBattlefield(card, b.PlayerID())
				g.FireEvent(core.GameEvent{Type: core.EvtCreatureBlockedBy, SourceID: p.ID(), TargetID: q.ID(), PlayerID: b.PlayerID()})
				if change == "source-leaves" {
					g.RemoveFromBattlefield(p)
				}
				g.PutTriggersOnStack()
				if g.StackSize() != 1 {
					t.Fatalf("stack=%d", g.StackSize())
				}
				if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				switch change {
				case "return":
					g.RemoveFromBattlefield(q)
					g.PutOnBattlefield(card, b.PlayerID())
				case "phase":
					g.PhaseOut(q.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(q.ID()))
				case "subtype":
					g.AddContinuousEffect(mage.SetSubtypes(q.ID(), mage.SubtypeCreature, core.Indefinite, "Wall"))
					g.ApplyContinuousEffects()
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: q.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "indestructible":
					g.MutablePermanent(q.ID()).GrantBaseAttr(core.Indestructible)
				}
				clone := g.Clone()
				for _, game := range []*mage.Game{clone, g} {
					game.ResolveStack()
					if change == "regenerate" || change == "copy" {
						game.AddRegenerationShield(q.ID())
					}
					game.RunStepWithPriority(core.EndCombat)
					survives := change == "return" || change == "regenerate" || change == "indestructible"
					if (game.FindPermanent(q.ID()) != nil) != survives {
						t.Fatal("wrong captured incarnation")
					}
					if change == "phase" {
						game.PhaseIn([]uuid.UUID{q.ID()})
						if game.FindPermanent(q.ID()) == nil {
							t.Fatal("destroyed phased opponent")
						}
					}
				}
			})
		}
	}
}

func TestRONCombatPairReferenceDeclarations(t *testing.T) {
	for _, event := range []string{"WhenACreatureBlocksACreature", "WhenACreatureBecomesBlockedByACreature"} {
		for _, ref := range []string{"Trigger_ThatOtherPermanent", "Trigger_ThatOtherPermanent()", "Trigger_TheAttackingCreature"} {
			source := `Card(Name:"Pair",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:4),Rules:[TriggerA(` + event + `(SinglePermanent(ThisPermanent),IsCardtype(Creature)),ActionList([DestroyPermanent(` + ref + `)]))])`
			nodes, err := Parse(strings.NewReader(source), "pair.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if ref != "Trigger_ThatOtherPermanent" {
				if err == nil {
					t.Fatalf("accepted %s", ref)
				}
				continue
			}
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			p := g.PutOnBattlefield(factory(), a.PlayerID())
			q := g.PutOnBattlefield(mage.NewCreature("Other", "", 0, 8), b.PlayerID())
			kind := core.EvtDeclaredBlocker
			if event == "WhenACreatureBecomesBlockedByACreature" {
				kind = core.EvtCreatureBlockedBy
			}
			g.FireEvent(core.GameEvent{Type: kind, SourceID: p.ID(), TargetID: q.ID(), PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			if g.FindPermanent(q.ID()) != nil || g.FindPermanent(p.ID()) == nil {
				t.Fatal("alias did not read opponent")
			}
		}
	}
	for _, rule := range []string{`Activated(PayMana([]),ActionList([DestroyPermanent(Trigger_ThatOtherPermanent)]))`, `TriggerA(WhenACreatureAttacks(SinglePermanent(ThisPermanent)),ActionList([DestroyPermanent(Trigger_ThatOtherPermanent)]))`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted out-of-scope opponent")
		}
	}
}

func TestRONCombatPairReferenceCasting(t *testing.T) {
	for _, name := range []string{"Battering Ram", "Infernal Medusa"} {
		g, a, _ := ronGame()
		set, cost, black := "LEG", 3, 2
		if name == "Battering Ram" {
			set, cost, black = "ATQ", 2, 0
		}
		card := repositoryCard(t, set, name)
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
			t.Fatal("unpaid spell did not roll back")
		}
		a.ManaPool().Add(core.Colorless, cost)
		a.ManaPool().Add(core.Black, black)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		if g.FindPermanent(card.ID()) != nil {
			t.Fatal("early battlefield entry")
		}
		g.ResolveStack()
		p := g.FindPermanent(card.ID())
		if p == nil || a.ManaPool().TotalMana() != 0 {
			t.Fatal("wrong payment or entry")
		}
		if name == "Battering Ram" {
			if !p.HasType(core.TypeArtifact) || !p.HasSubType("Construct") || p.CurrentPower(g) != 1 || p.CurrentToughness(g) != 1 {
				t.Fatal("wrong Ram characteristics")
			}
		} else if !p.HasSubType("Gorgon") || p.CurrentPower(g) != 2 || p.CurrentToughness(g) != 4 {
			t.Fatal("wrong Medusa characteristics")
		}
	}
}

func TestRONCombatPairReferenceIndependentEvents(t *testing.T) {
	for _, name := range []string{"Battering Ram", "Infernal Medusa"} {
		g, a, b := ronGame()
		set := "LEG"
		if name == "Battering Ram" {
			set = "ATQ"
		}
		victims := make([]*mage.Permanent, 0, 4)
		for range 2 {
			p := g.PutOnBattlefield(repositoryCard(t, set, name), a.PlayerID())
			for j := range 2 {
				kind := core.EvtCreatureBlockedBy
				opts := []mage.CardOption{mage.WithKeyword(core.Shroud)}
				if name == "Battering Ram" {
					opts = append(opts, mage.WithSubTypes("Wall"))
				} else if j == 1 {
					kind = core.EvtDeclaredBlocker
				}
				q := g.PutOnBattlefield(mage.NewCreature("Other", "", 0, 8, opts...), b.PlayerID())
				victims = append(victims, q)
				g.FireEvent(core.GameEvent{Type: kind, SourceID: p.ID(), TargetID: q.ID(), PlayerID: a.PlayerID()})
			}
		}
		g.PutTriggersOnStack()
		if g.StackSize() != 4 {
			t.Fatalf("stack=%d", g.StackSize())
		}
		clone := g.Clone()
		clone.ResolveStack()
		clone.RunStepWithPriority(core.EndCombat)
		if g.StackSize() != 4 {
			t.Fatal("clone changed original stack")
		}
		for _, q := range victims {
			if clone.FindPermanent(q.ID()) != nil || g.FindPermanent(q.ID()) == nil {
				t.Fatal("clone or event binding crossed")
			}
		}
		g.ResolveStack()
		g.RunStepWithPriority(core.EndCombat)
		for _, q := range victims {
			if g.FindPermanent(q.ID()) != nil {
				t.Fatal("lost independent capture")
			}
		}
	}
}

func TestRONCombatPairReferenceNextTurnRule(t *testing.T) {
	if _, err := Compile(hostUntapRecord(t, "LEG", "Wall of Dust")); err != nil {
		t.Fatal(err)
	}
}

func TestRONCombatPairReferenceStrictLoad(t *testing.T) {
	records := make([]string, 0, 2)
	for _, tc := range []struct{ set, name string }{{"ATQ", "Battering Ram"}, {"LEG", "Infernal Medusa"}} {
		n := hostUntapRecord(t, tc.set, tc.name)
		data, err := os.ReadFile("../../../../cards/mtgish/" + tc.set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, strings.Split(string(data), "\n")[n.Position.Line-1])
	}
	files := fstest.MapFS{"cards.ron": {Data: []byte(strings.Join(records, "\n"))}, "reprints.ron": {Data: []byte(strings.Join(records, "\n"))}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("reprints changed unique names")
	}
	for _, name := range []string{"Battering Ram", "Infernal Medusa"} {
		first, err := registry.CreateCard(name)
		if err != nil {
			t.Fatal(err)
		}
		second, err := registry.CreateCard(name)
		if err != nil {
			t.Fatal(err)
		}
		if first.ID() == second.ID() || first.Abilities()[0] == second.Abilities()[0] {
			t.Fatal("factory instances share state")
		}
		if err := mage.ValidateCardDefinition(name, first); err != nil {
			t.Fatal(err)
		}
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([DestroyPermanent(Trigger_ThatOtherPermanent)]))])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("invalid collection changed registry")
	}
}

func TestRONCombatPairReferenceUnion(t *testing.T) {
	rule := `TriggerA(Or([WhenACreatureBlocksACreature(SinglePermanent(ThisPermanent),IsCardtype(Creature)),WhenACreatureBecomesBlockedByACreature(SinglePermanent(ThisPermanent),IsCardtype(Creature))]),ActionList([CreateFutureTrigger(AtTheNextEndOfCombat,ActionList([DestroyPermanent(Trigger_ThatOtherPermanent)]))]))`
	nodes, err := Parse(strings.NewReader(`Card(Name:"Union",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:4),Rules:[`+rule+`])`), "union.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("Other", "", 0, 8), b.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtCombatPair, SourceID: p.ID(), TargetID: q.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("union capture missing")
	}
	g.ResolveStack()
	g.RunStepWithPriority(core.EndCombat)
	if g.FindPermanent(q.ID()) != nil || g.FindPermanent(p.ID()) == nil {
		t.Fatal("union alias changed subject")
	}
}
