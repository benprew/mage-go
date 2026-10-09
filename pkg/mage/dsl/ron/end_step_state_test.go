package ron

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func endStepStateProbe(t *testing.T, condition string) mage.Card {
	t.Helper()
	nodes, err := Parse(strings.NewReader(`Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:4),Rules:[TriggerI(AtTheBeginningOfAPlayersEndStep(SinglePlayer(You)),`+condition+`,ActionList([GainLife(Integer(2))]))])`), "probe.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	c := factory()
	if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRONEndStepSourceState(t *testing.T) {
	for _, state := range []string{"IsUntapped", "DidntAttackThisTurn"} {
		for _, change := range []string{"none", "false-before", "false-after", "leave", "return", "phase-before", "phase-after", "phase-cycle", "control", "copy", "cleanup-clone"} {
			t.Run(state+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				c := endStepStateProbe(t, "PermanentPassesFilter(ThisPermanent,"+state+")")
				c.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(c, a.PlayerID())
				p.RevokeBaseAttr(core.AttrSummonSick)
				setFalse := func() {
					if state == "IsUntapped" {
						g.TapPermanent(g.FindPermanent(p.ID()))
					} else {
						g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
						g.GetCombat().RemoveFromCombat(p.ID())
					}
				}
				if change == "false-before" {
					setFalse()
				}
				if change == "phase-before" {
					g.PhaseOut(p.ID())
				}
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
				g.PutTriggersOnStack()
				if g.StackSize() != 0 {
					t.Fatal("opponent end step matched")
				}
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				g.PutTriggersOnStack()
				triggered := change != "false-before" && change != "phase-before"
				if (g.StackSize() == 1) != triggered {
					t.Fatal("detection boundary")
				}
				switch change {
				case "false-after":
					setFalse()
				case "leave", "return":
					g.BouncePermanentToHand(g.FindPermanent(p.ID()))
					if change == "return" {
						a.RemoveFromHand(c.ID())
						p = g.PutOnBattlefield(c, a.PlayerID())
						p.RevokeBaseAttr(core.AttrSummonSick)
						setFalse()
					}
				case "phase-after":
					g.PhaseOut(p.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				clone := g.Clone()
				if change == "cleanup-clone" && state == "DidntAttackThisTurn" {
					clone.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
				}
				g.ResolveStack()
				want := 20
				if triggered && change != "false-after" {
					want = 22
				}
				if a.Life() != want {
					t.Fatalf("life=%d want %d", a.Life(), want)
				}
				if change == "copy" && b.Life() != 22 {
					t.Fatal("copied controller")
				}
				clone.ResolveStack()
				cloneWant := want
				if change == "cleanup-clone" && state == "DidntAttackThisTurn" {
					cloneWant = 20
				}
				if clone.GetPlayer(a.PlayerID()).Life() != cloneWant {
					t.Fatal("clone recheck")
				}
			})
		}
	}
}

func TestRONOsaiVulturesEndStep(t *testing.T) {
	for _, scenario := range []string{"none", "late-death", "one", "many", "token", "noncreature", "exile", "bounce", "regenerate", "indestructible", "phase-before", "phase-after", "return", "control", "copy", "cleanup", "cleanup-after"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "LEG", "Osai Vultures")
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			if g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0) == nil || g.StackSize() != 0 || len(a.Hand()) != 1 {
				t.Fatal("unpaid cast rollback")
			}
			a.ManaPool().Add(core.White, 2)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			p := g.FindPermanent(c.ID())
			if !p.HasKeyword(core.Flying) {
				t.Fatal("flying absent")
			}
			victimCard := mage.NewCreature("Victim", "", 1, 1)
			victimCard.SetOwner(b.PlayerID())
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			switch scenario {
			case "none", "late-death":
			case "noncreature":
				land := mage.NewLand("Land")
				land.SetOwner(b.PlayerID())
				g.DestroyPermanent(g.PutOnBattlefield(land, b.PlayerID()))
			case "exile":
				g.ExilePermanent(victim)
			case "bounce":
				g.BouncePermanentToHand(victim)
			case "regenerate":
				g.AddRegenerationShield(victim.ID())
				g.DestroyPermanent(victim)
			case "indestructible":
				victim.GrantBaseAttr(core.Indestructible)
				g.DestroyPermanent(victim)
			default:
				if scenario == "token" {
					victim.IsToken = true
				}
				g.DestroyPermanent(victim)
				if scenario == "many" {
					second := mage.NewCreature("Second", "", 1, 1)
					second.SetOwner(a.PlayerID())
					g.DestroyPermanent(g.PutOnBattlefield(second, a.PlayerID()))
				}
			}
			if scenario == "phase-before" {
				g.PhaseOut(p.ID())
			}
			if scenario == "cleanup" {
				g.RunStepWithPriority(core.Cleanup)
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			triggered := scenario != "none" && scenario != "late-death" && scenario != "noncreature" && scenario != "exile" && scenario != "bounce" && scenario != "regenerate" && scenario != "indestructible" && scenario != "phase-before" && scenario != "cleanup"
			if (g.StackSize() == 1) != triggered || p.Counters[core.Carrion] != 0 {
				t.Fatal("detection or early counter")
			}
			switch scenario {
			case "late-death":
				g.DestroyPermanent(victim)
			case "cleanup-after":
				g.RunStepWithPriority(core.Cleanup)
				if g.StackSize() != 0 {
					t.Fatal("cleanup left pending trigger")
				}
			case "phase-after":
				g.PhaseOut(p.ID())
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(c.ID())
				p = g.PutOnBattlefield(c, a.PlayerID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			want := 0
			if triggered && scenario != "phase-after" && scenario != "return" {
				want = 1
			}
			if scenario == "copy" {
				want = 2
			}
			if g.FindPermanentIncludingPhased(p.ID()).Counters[core.Carrion] != uint8(want) || clone.FindPermanentIncludingPhased(p.ID()).Counters[core.Carrion] != uint8(want) {
				t.Fatal("counter outcome")
			}
		})
	}
}

func TestRONOsaiVulturesActivation(t *testing.T) {
	for _, change := range []string{"none", "return-before", "phase-before", "phase-cycle", "control", "source-left-after", "cleanup-phased"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "LEG", "Osai Vultures")
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			index := 1
			if g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), index, nil) == nil || g.StackSize() != 0 {
				t.Fatal("unpaid counter cost")
			}
			g.MutablePermanent(p.ID()).AddCounter(core.Carrion, 4)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), index, nil); err != nil {
				t.Fatal(err)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), index, nil); err != nil {
				t.Fatal(err)
			}
			p = g.FindPermanent(p.ID())
			if p.Counters[core.Carrion] != 0 || p.CurrentPower(g) != 1 {
				t.Fatal("counter payment or early bonus")
			}
			if g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), index, nil) == nil || g.StackSize() != 2 {
				t.Fatal("third payment accepted")
			}
			switch change {
			case "return-before":
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(c.ID())
				p = g.PutOnBattlefield(c, a.PlayerID())
			case "phase-before":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			if change == "phase-before" {
				g.PhaseIn([]uuid.UUID{p.ID()})
				clone.PhaseIn([]uuid.UUID{p.ID()})
			}
			want := 3
			if change == "return-before" || change == "phase-before" {
				want = 1
			}
			if g.FindPermanent(p.ID()).CurrentPower(g) != want || g.FindPermanent(p.ID()).CurrentToughness(g) != want || clone.FindPermanent(p.ID()).CurrentPower(clone) != want {
				t.Fatal("bonus identity or stacking")
			}
			switch change {
			case "source-left-after":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(c.ID())
				p = g.PutOnBattlefield(c, a.PlayerID())
			case "cleanup-phased":
				g.PhaseOut(p.ID())
			}
			g.RunStepWithPriority(core.Cleanup)
			if change == "cleanup-phased" {
				g.PhaseIn([]uuid.UUID{p.ID()})
				g.ApplyContinuousEffects()
			}
			if g.FindPermanent(p.ID()).CurrentPower(g) != 1 || clone.FindPermanent(p.ID()).CurrentPower(clone) != want {
				t.Fatal("cleanup or clone isolation")
			}
		})
	}
}

func TestRONEndStepStateDeclarations(t *testing.T) {
	conditions := []string{
		`ACreatureOrPlaneswalkerDiedThisTurn`, `ACreatureOrPlaneswalkerDiedThisTurn(IsCardtype(Creature),IsCardtype(Creature))`,
		`ACreatureOrPlaneswalkerDiedThisTurn(IsCardtype(Planeswalker))`, `ACreatureOrPlaneswalkerDiedThisTurn(IsCardtype(Creature()))`,
		`ACreatureOrPlaneswalkerDiedThisTurn(And([IsCardtype(Creature)]))`,
		`PermanentPassesFilter(ThisPermanent,IsUntapped())`, `PermanentPassesFilter(ThisPermanent,DidntAttackThisTurn())`,
		`PermanentPassesFilter(ThisPermanent(),IsUntapped)`, `PermanentPassesFilter(Trigger_ThatPermanent,DidntAttackThisTurn)`,
		`PermanentPassesFilter(Ref_TargetPermanent,IsUntapped)`, `PermanentPassesFilter(HostPermanent,IsUntapped)`,
	}
	for _, condition := range conditions {
		t.Run(condition, func(t *testing.T) {
			src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerI(AtTheBeginningOfAPlayersEndStep(AnyPlayer),` + condition + `,ActionList([GainLife(Integer(1))]))])`
			nodes, err := Parse(strings.NewReader(src), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatal("invalid declaration accepted")
			}
		})
	}
}

func TestRONEndStepStateRecords(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	selected := fstest.MapFS{}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), file)
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range nodes {
			name := n.Fields["Name"].String
			if name != "Erg Raiders" && name != "Voodoo Doll" && name != "Osai Vultures" {
				continue
			}
			counts[name]++
			factory, err := Compile(n)
			if name == "Osai Vultures" || name == "Erg Raiders" || name == "Voodoo Doll" {
				if err != nil {
					t.Fatal(err)
				}
				first, second := factory(), factory()
				if first.ID() == second.ID() {
					t.Fatal("shared factory")
				}
				if err := mage.ValidateCardDefinition(name, first); err != nil {
					t.Fatal(err)
				}
			}
			if name == "Voodoo Doll" {
				rules := n.Fields["Rules"].Args
				for _, rule := range rules {
					if rule.Name != "TriggerI" {
						continue
					}
					condition, err := compileEndStepCondition(rule.Args[1])
					if err != nil || condition == nil {
						t.Fatal("complete-record condition", err)
					}
					rule.Args = append([]Node(nil), rule.Args...)
					probe, _ := Parse(strings.NewReader(`ActionList([GainLife(Integer(2))])`), "body.ron")
					rule.Args[2] = probe[0]
					n.Fields["Rules"] = Node{Kind: List, Args: []Node{rule}}
					probeFactory, err := Compile(n)
					if err != nil {
						t.Fatal(err)
					}
					g, a, _ := ronGame()
					card := probeFactory()
					card.SetOwner(a.PlayerID())
					g.PutOnBattlefield(card, a.PlayerID())
					g.RunStepWithPriority(core.EndStep)
					if a.Life() != 22 {
						t.Fatal("complete-record condition probe")
					}
				}
			}
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "Osai Vultures"`) {
				selected[filepath.Base(file)] = &fstest.MapFile{Data: []byte(line)}
			}
		}
	}
	if counts["Erg Raiders"] != 2 || counts["Voodoo Doll"] != 1 || counts["Osai Vultures"] != 1 {
		t.Fatal("reprint counts", counts)
	}
	registry, err := LoadFS(selected, "*.ron")
	if err != nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("complete selected loading", err)
	}
	selected["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerI(AtTheBeginningOfAPlayersEndStep(AnyPlayer),ACreatureOrPlaneswalkerDiedThisTurn(IsCardtype(Planeswalker)),ActionList([]))])`)}
	if registry, err := LoadFS(selected, "*.ron"); err == nil || registry != nil {
		t.Fatal("partial registry published")
	}
	report, err := InspectFS(selected, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	nested := false
	for _, signature := range report.Signatures {
		if signature.Name == "Creature" && signature.Role == "card_type" {
			nested = true
		}
	}
	if !nested {
		t.Fatal("death filter semantic role")
	}
}

func TestRONOsaiVulturesNativeSteps(t *testing.T) {
	g, a, b := ronGame()
	cards := make([]*mage.Permanent, 0, 2)
	for _, player := range []*mage.BasePlayer{a, b} {
		c := repositoryCard(t, "LEG", "Osai Vultures")
		c.SetOwner(player.PlayerID())
		cards = append(cards, g.PutOnBattlefield(c, player.PlayerID()))
	}
	victim := mage.NewCreature("Victim", "", 1, 1)
	victim.SetOwner(b.PlayerID())
	g.DestroyPermanent(g.PutOnBattlefield(victim, b.PlayerID()))
	g.RunStepWithPriority(core.EndStep)
	for _, p := range cards {
		if g.FindPermanent(p.ID()).Counters[core.Carrion] != 1 {
			t.Fatal("each end step native trigger")
		}
	}
	g.RunStepWithPriority(core.Cleanup)
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(core.EndStep)
	for _, p := range cards {
		if g.FindPermanent(p.ID()).Counters[core.Carrion] != 1 {
			t.Fatal("old death history reused")
		}
	}
	another := mage.NewCreature("Second", "", 1, 1)
	another.SetOwner(a.PlayerID())
	g.DestroyPermanent(g.PutOnBattlefield(another, a.PlayerID()))
	g.RunStepWithPriority(core.EndStep)
	for _, p := range cards {
		if g.FindPermanent(p.ID()).Counters[core.Carrion] != 2 {
			t.Fatal("second controller end step")
		}
	}
	for _, p := range cards {
		if err := g.ActivateAbilityByIndex(p.ControllerID(), p.ID(), 1, nil); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	for _, p := range cards {
		live := g.FindPermanent(p.ID())
		if live.Counters[core.Carrion] != 0 || live.CurrentPower(g) != 2 || live.CurrentToughness(g) != 2 {
			t.Fatal("native earned counter activation")
		}
	}
}
