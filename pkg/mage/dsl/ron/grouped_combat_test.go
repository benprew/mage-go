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

const groupedCombatEvent = `Or([WhenACreatureBlocksAnyNumberOfCreatures(SinglePermanent(ThisPermanent),IsCreatureType(Orc)),WhenACreatureBecomesBlockedByAnyNumberOfCreatures(SinglePermanent(ThisPermanent),IsCreatureType(Orc))])`

func TestRONGroupedCombatSoldier(t *testing.T) {
	for _, attacking := range []bool{false, true} {
		for _, matches := range []int{0, 1, 2} {
			t.Run(fmt.Sprintf("attack=%t/matches=%d", attacking, matches), func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "FEM", "Dwarven Soldier")
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				p := g.PutOnBattlefield(card, a.PlayerID())
				p.RevokeBaseAttr(core.AttrSummonSick)
				if !attacking {
					p.GrantBaseAttr(core.CanBlockAny)
				}
				others := make([]uuid.UUID, 3)
				for i := range others {
					subtype := "Human"
					if i < matches {
						subtype = "Orc"
					}
					q := g.PutOnBattlefield(mage.NewCreature("Opponent", "", 0, 8, mage.WithSubTypes(subtype)), b.PlayerID())
					q.RevokeBaseAttr(core.AttrSummonSick)
					others[i] = q.ID()
				}
				assignments := make([]mage.BlockAssignment, 3)
				if attacking {
					g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
					for i, id := range others {
						assignments[i] = mage.BlockAssignment{BlockerID: id, AttackerID: p.ID()}
					}
				} else {
					g.ExecuteAttackers(b.PlayerID(), others)
					for i, id := range others {
						assignments[i] = mage.BlockAssignment{BlockerID: p.ID(), AttackerID: id}
					}
				}
				g.ExecuteBlockers(assignments)
				want := 1
				if matches > 0 {
					want = 3
				}
				checkConditionalPT(t, g, p.ID(), 2, want)
				g.ExecuteBlockers(assignments)
				checkConditionalPT(t, g, p.ID(), 2, want)
				g.RunStepWithPriority(core.Cleanup)
				checkConditionalPT(t, g, p.ID(), 2, 1)
			})
		}
	}
}

func TestRONGroupedCombatIdentity(t *testing.T) {
	for _, change := range []string{"normal", "subtype", "opponent-phase", "source-phase", "phase-cycle", "return", "control", "remove-before", "copy", "clone"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "FEM", "Dwarven Soldier")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Orc", "", 0, 8, mage.WithSubTypes("Orc")), b.PlayerID())
			second := g.PutOnBattlefield(repositoryCard(t, "FEM", "Dwarven Soldier"), a.PlayerID())
			if change == "remove-before" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
			}
			g.FireEvent(core.GameEvent{Type: core.EvtBlockerGroup, SourceID: p.ID(), PlayerID: a.PlayerID(), RelatedIDs: []uuid.UUID{other.ID()}})
			switch change {
			case "subtype":
				g.AddContinuousEffect(mage.SetSubtypes(other.ID(), mage.SubtypeCreature, core.EndOfTurn, "Human"))
				g.ApplyContinuousEffects()
			case "opponent-phase":
				g.PhaseOut(other.ID())
			case "source-phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.PutTriggersOnStack()
			wantStack := 1
			if change == "remove-before" {
				wantStack = 0
			}
			if g.StackSize() != wantStack {
				t.Fatalf("stack=%d want=%d", g.StackSize(), wantStack)
			}
			if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			clone.ResolveStack()
			if g.FindPermanentIncludingPhased(p.ID()).CurrentToughness(g) != 1 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			want := 3
			if change == "source-phase" || change == "return" || change == "remove-before" {
				want = 1
			}
			if change == "copy" {
				want = 5
			}
			for _, game := range []*mage.Game{g, clone} {
				if game.FindPermanentIncludingPhased(p.ID()).CurrentToughness(game) != want {
					t.Fatalf("toughness=%d want=%d", game.FindPermanentIncludingPhased(p.ID()).CurrentToughness(game), want)
				}
				checkConditionalPT(t, game, second.ID(), 2, 1)
			}
			if change == "source-phase" {
				g.PhaseIn([]uuid.UUID{p.ID()})
				checkConditionalPT(t, g, p.ID(), 2, 1)
			}
		})
	}
}

func TestRONGroupedCombatCastingAndEntry(t *testing.T) {
	g, a, b := ronGame()
	c := repositoryCard(t, "FEM", "Dwarven Soldier")
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast")
	}
	if a.ManaPool().TotalMana() != 1 || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("cost rollback")
	}
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(c.ID()) != nil {
		t.Fatal("early entry")
	}
	g.ResolveStack()
	p := g.FindPermanent(c.ID())
	if p == nil || !p.HasSubType("Dwarf") || !p.HasSubType("Soldier") || a.ManaPool().TotalMana() != 0 {
		t.Fatal("metadata or cost")
	}
	checkConditionalPT(t, g, p.ID(), 2, 1)
	if c.Abilities()[0] == repositoryCard(t, "FEM", "Dwarven Soldier").Abilities()[0] {
		t.Fatal("shared factory ability")
	}
	attacker := g.PutOnBattlefield(mage.NewCreature("Orc", "", 0, 8, mage.WithSubTypes("Orc")), b.PlayerID())
	attacker.RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{attacker.ID()})
	entering := g.PutOnBattlefieldBlocking(repositoryCard(t, "FEM", "Dwarven Soldier"), a.PlayerID(), attacker.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	checkConditionalPT(t, g, entering.ID(), 2, 1)
}

func TestRONGroupedCombatDeclarations(t *testing.T) {
	body := `ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[AdjustPT(0,2)],UntilEndOfTurn)])`
	for _, event := range []string{
		strings.Replace(groupedCombatEvent, "IsCreatureType(Orc)", "IsCreatureType(Goblin)", 1),
		strings.Replace(groupedCombatEvent, "ThisPermanent", "HostPermanent", 1),
		strings.ReplaceAll(groupedCombatEvent, "ThisPermanent", "ThisPermanent()"),
		strings.ReplaceAll(groupedCombatEvent, "Orc", "Island"),
		strings.ReplaceAll(groupedCombatEvent, "Orc", "\"Orc\""),
		`Or([])`,
		`Or([WhenACreatureBlocksAnyNumberOfCreatures(SinglePermanent(ThisPermanent),IsCreatureType(Orc))])`,
		strings.Replace(groupedCombatEvent, "WhenACreatureBecomesBlockedByAnyNumberOfCreatures", "WhenACreatureBecomesBlockedByACreature", 1),
		strings.ReplaceAll(groupedCombatEvent, "IsCreatureType(Orc)", "IsCreatureType(Orc,Orc)"),
		strings.ReplaceAll(groupedCombatEvent, "IsCreatureType(Orc)", "AnyPermanent"),
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:4),Rules:[TriggerA(` + event + `,` + body + `)])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", event)
		}
	}
	for _, event := range []string{groupedCombatEvent, `WhenACreatureBecomesBlocked(SinglePermanent(ThisPermanent))`} {
		for _, ref := range []string{"Trigger_ThatOtherCreature", "Trigger_TheBlockingCreature", "Trigger_DefendingPlayer"} {
			badBody := `ActionList([DestroyPermanent(` + ref + `)])`
			nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid binding",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:4),Rules:[TriggerA(`+event+`,`+badBody+`)])`), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatal("invented group binding", ref)
			}
		}
	}
	n := hostUntapRecord(t, "FEM", "Dwarven Soldier")
	rule := n.Fields["Rules"].Args[0]
	reversed := rule.Args[0]
	reversed.Args = append([]Node(nil), reversed.Args...)
	elements := reversed.Args[0]
	elements.Args = []Node{elements.Args[1], elements.Args[0]}
	reversed.Args[0] = elements
	if _, err := compileTriggerRule(Node{Kind: Tuple, Name: "TriggerA", Args: []Node{reversed, rule.Args[1]}}, scope{permanent: true}); err != nil {
		t.Fatal("reversed union", err)
	}
}

func TestRONGroupedCombatStrictLoad(t *testing.T) {
	raw, err := os.ReadFile("../../../../cards/mtgish/FEM.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var data []byte
	for line := range strings.SplitSeq(string(raw), "\n") {
		if strings.Contains(line, `Name: "Dwarven Soldier"`) {
			data = []byte(line)
		}
	}
	if len(data) == 0 {
		t.Fatal("missing complete record")
	}
	registry, err := LoadFS(fstest.MapFS{"a.ron": {Data: data}, "b.ron": {Data: data}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("reprints")
	}
	if partial, err := LoadFS(fstest.MapFS{"a.ron": {Data: data}, "bad.ron": {Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Unknown])`)}}, "*.ron"); err == nil || partial != nil {
		t.Fatal("partial registry")
	}
}

func TestRONGroupedCombatBecomesBlocked(t *testing.T) {
	factory := characteristicFactory(t, false, `TriggerA(WhenACreatureBecomesBlocked(SinglePermanent(ThisPermanent)),ActionList([CreatePermanentLayerEffectUntil(Trigger_ThatCreature,[AdjustPT(0,2)],UntilEndOfTurn)]))`)
	g, a, b := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	p.RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
	for range 2 {
		g.PutOnBattlefieldBlocking(mage.NewCreature("Blocker", "", 0, 8), b.PlayerID(), p.ID())
		g.PutTriggersOnStack()
		g.ResolveStack()
	}
	checkConditionalPT(t, g, p.ID(), 0, 10)
}

func TestRONGroupedCombatLegalityAndMatching(t *testing.T) {
	for _, scenario := range []string{"tapped", "phased", "noncreature", "other-source", "later-subtype", "new-group", "attacker-entry"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			p := g.PutOnBattlefield(repositoryCard(t, "FEM", "Dwarven Soldier"), a.PlayerID())
			p.RevokeBaseAttr(core.AttrSummonSick)
			q := g.PutOnBattlefield(mage.NewCreature("Orc", "", 0, 8, mage.WithSubTypes("Orc")), b.PlayerID())
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
			want := 1
			switch scenario {
			case "tapped":
				g.TapPermanent(q)
			case "phased":
				g.PhaseOut(q.ID())
			case "noncreature":
				q = g.PutOnBattlefield(mage.NewArtifact("Orc artifact", "", mage.WithSubTypes("Orc")), b.PlayerID())
			case "other-source":
				other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 8), a.PlayerID())
				other.RevokeBaseAttr(core.AttrSummonSick)
				g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{other.ID()})
				g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: q.ID(), AttackerID: other.ID()}})
				checkConditionalPT(t, g, p.ID(), 2, 1)
				return
			case "later-subtype":
				g.AddContinuousEffect(mage.SetSubtypes(q.ID(), mage.SubtypeCreature, core.Indefinite, "Human"))
				g.ApplyContinuousEffects()
			case "new-group":
				want = 5
			case "attacker-entry":
				g.PutOnBattlefieldBlocking(mage.NewCreature("Entering Orc", "", 0, 8, mage.WithSubTypes("Orc")), b.PlayerID(), p.ID())
				g.PutTriggersOnStack()
				g.ResolveStack()
				checkConditionalPT(t, g, p.ID(), 2, 3)
				return
			}
			g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: q.ID(), AttackerID: p.ID()}})
			if scenario == "new-group" {
				checkConditionalPT(t, g, p.ID(), 2, 3)
				g.PutOnBattlefieldBlocking(mage.NewCreature("New Orc", "", 0, 8, mage.WithSubTypes("Orc")), b.PlayerID(), p.ID())
				g.PutTriggersOnStack()
				g.ResolveStack()
			}
			if scenario == "later-subtype" {
				g.AddContinuousEffect(mage.SetSubtypes(q.ID(), mage.SubtypeCreature, core.Indefinite, "Orc"))
				g.ApplyContinuousEffects()
				g.PutTriggersOnStack()
				g.ResolveStack()
			}
			checkConditionalPT(t, g, p.ID(), 2, want)
		})
	}
}

func TestRONGroupedCombatBecomesBlockedIdentity(t *testing.T) {
	for _, scenario := range []string{"return", "phase", "phase-cycle", "control", "copy", "other", "removed", "pair-only"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			factory := characteristicFactory(t, false, `TriggerA(WhenACreatureBecomesBlocked(SinglePermanent(ThisPermanent)),ActionList([CreatePermanentLayerEffectUntil(Trigger_ThatCreature,[AdjustPT(0,2)],UntilEndOfTurn)]))`)
			c := factory()
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			other := g.PutOnBattlefield(factory(), b.PlayerID())
			event := core.GameEvent{Type: core.EvtCreatureBecomesBlocked, SourceID: p.ID(), PlayerID: a.PlayerID()}
			if scenario == "other" {
				event.SourceID = other.ID()
				event.PlayerID = b.PlayerID()
			}
			if scenario == "pair-only" {
				event.Type = core.EvtCreatureBlockedBy
				event.TargetID = other.ID()
			}
			if scenario == "removed" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
			}
			g.FireEvent(event)
			switch scenario {
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(c.ID())
				g.PutOnBattlefield(c, a.PlayerID())
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.PutTriggersOnStack()
			if scenario == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			clone.ResolveStack()
			g.ResolveStack()
			want := 10
			if scenario == "return" || scenario == "phase" || scenario == "other" || scenario == "removed" || scenario == "pair-only" {
				want = 8
			}
			if scenario == "copy" {
				want = 12
			}
			for _, game := range []*mage.Game{g, clone} {
				if got := game.FindPermanentIncludingPhased(p.ID()).CurrentToughness(game); got != want {
					t.Fatalf("toughness=%d want=%d", got, want)
				}
			}
		})
	}
}

func TestRONGroupedCombatSpittingSlugBoundary(t *testing.T) {
	n := hostUntapRecord(t, "DRK", "Spitting Slug")
	if _, err := Compile(n); err != nil {
		t.Fatalf("supported payment and collection: %v", err)
	}
}

func TestRONGroupedCombatDamage(t *testing.T) {
	g, a, b := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "FEM", "Dwarven Soldier"), a.PlayerID())
	p.RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
	first := g.PutOnBattlefield(mage.NewCreature("First Orc", "", 1, 3, mage.WithSubTypes("Orc")), b.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second Orc", "", 1, 3, mage.WithSubTypes("Orc")), b.PlayerID())
	g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: first.ID(), AttackerID: p.ID()}, {BlockerID: second.ID(), AttackerID: p.ID()}})
	checkConditionalPT(t, g, p.ID(), 2, 3)
	g.ExecuteCombatDamage()
	if g.FindPermanent(p.ID()) == nil || g.FindPermanent(p.ID()).Damage != 2 {
		t.Fatal("bonus did not keep the soldier alive")
	}
	if g.FindPermanent(first.ID()).Damage+g.FindPermanent(second.ID()).Damage != 2 || b.Life() != 20 {
		t.Fatal("combat damage")
	}
}

func TestRONGroupedCombatInvalidBlockedDeclarations(t *testing.T) {
	for _, event := range []string{
		`WhenACreatureBecomesBlocked()`,
		`WhenACreatureBecomesBlocked(SinglePermanent(ThisPermanent),IsCreatureType(Orc))`,
		`WhenACreatureBecomesBlocked(SinglePermanent(HostPermanent))`,
		`WhenACreatureBecomesBlocked(SinglePermanent(ThisPermanent()))`,
		`WhenACreatureBecomesBlocked(SinglePermanent(Ref_TargetPermanent))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:4),Rules:[TriggerA(`+event+`,ActionList([DrawCards(Integer(1))]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", event)
		}
	}
	for _, event := range []string{groupedCombatEvent, `WhenACreatureBecomesBlocked(SinglePermanent(ThisPermanent))`} {
		for _, types := range []string{"Instant", "Sorcery"} {
			nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[`+types+`],Subtypes:[]),Rules:[TriggerA(`+event+`,ActionList([DrawCards(Integer(1))]))])`), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatal("permanent event on spell", types)
			}
		}
	}
}
