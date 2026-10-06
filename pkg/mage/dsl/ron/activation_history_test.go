package ron

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONActivationHistoryDragons(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		power     int
	}{{"LEA", "Dragon Whelp", 2}, {"PROMOS", "Nalathni Dragon", 1}} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, _ := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			a.ManaPool().Add(core.Red, 4)
			for range 3 {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
			}
			if len(g.DelayedTriggers()) != 0 || g.FindPermanent(p.ID()).CurrentPower(g) != tc.power+3 {
				t.Fatal("early sacrifice or wrong bonus")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if len(g.DelayedTriggers()) != 0 {
				t.Fatal("delay before resolution")
			}
			clone := g.Clone()
			g.ResolveStack()
			if len(g.DelayedTriggers()) != 1 || g.FindPermanent(p.ID()).CurrentPower(g) != tc.power+4 {
				t.Fatal("fourth resolution")
			}
			g.RunStepWithPriority(core.EndStep)
			if g.FindPermanent(p.ID()) != nil || len(a.Graveyard()) != 1 {
				t.Fatal("missing delayed sacrifice")
			}
			if clone.FindPermanent(p.ID()) == nil {
				t.Fatal("clone shared sacrifice")
			}
			clone.ResolveStack()
			clone.RunStepWithPriority(core.EndStep)
			if clone.FindPermanent(p.ID()) != nil {
				t.Fatal("clone lost pending history")
			}
		})
	}
}

func TestRONActivationHistoryInvalid(t *testing.T) {
	valid := `If(TimesThisAbilityHasBeenActivatedThisTurnIs(GreaterThanOrEqualTo(Integer(4))), [GainLife(Integer(1))])`
	for _, bad := range []string{
		`If(TimesThisAbilityHasResolvedThisTurnIs(GreaterThanOrEqualTo(Integer(4))), [GainLife(Integer(1))])`,
		`If(TimesThisAbilityHasBeenActivatedThisTurnIs, [GainLife(Integer(1))])`,
		`If(TimesThisAbilityHasBeenActivatedThisTurnIs(EqualTo(Integer(4))), [GainLife(Integer(1))])`,
		`If(TimesThisAbilityHasBeenActivatedThisTurnIs(GreaterThanOrEqualTo(Integer(-1))), [GainLife(Integer(1))])`,
		`If(TimesThisAbilityHasBeenActivatedThisTurnIs(GreaterThanOrEqualTo(Integer(4))), [])`,
		`CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + valid + `]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+bad+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([`+valid+`]))`)
	first, second := factory(), factory()
	if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
		t.Fatal(err)
	}
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(first, a.PlayerID())
	q := g.PutOnBattlefield(second, a.PlayerID())
	for range 4 {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), q.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("factory instances share history")
	}
}

func TestRONActivationHistoryDragonTimingAndIdentity(t *testing.T) {
	for _, change := range []string{"pending-four", "countered-four", "copy-three", "copy-four", "source-return", "source-phase", "phase-cycle", "control", "after-end-begins"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEA", "Dragon Whelp")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			a.ManaPool().Add(core.Red, 4)
			if change == "after-end-begins" {
				g.RunStepWithPriority(core.EndStep)
				a.ManaPool().Add(core.Red, 4)
			}
			for i := range 3 {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if change != "pending-four" && change != "countered-four" && (change != "copy-three" || i != 2) {
					g.ResolveStack()
				}
			}
			if change != "copy-three" {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			}

			switch change {
			case "countered-four":
				g.GetStack().Pop()
			case "copy-three", "copy-four":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
			case "source-return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			case "source-phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.ResolveStack()
			wantDelays := 1
			if change == "pending-four" {
				wantDelays = 4
			}
			if change == "countered-four" {
				wantDelays = 3
			}
			if change == "copy-three" {
				wantDelays = 0
			}
			if change == "copy-four" {
				wantDelays = 2
			}
			if len(g.DelayedTriggers()) != wantDelays {
				t.Fatalf("delays %d want %d", len(g.DelayedTriggers()), wantDelays)
			}
			if change == "after-end-begins" {
				g.RunStepWithPriority(core.Cleanup)
				if g.FindPermanent(p.ID()) == nil {
					t.Fatal("same end step consumed delay")
				}
			}
			g.RunStepWithPriority(core.EndStep)
			survives := change == "copy-three" || change == "source-return" || change == "source-phase" || change == "control"
			if (g.FindPermanentIncludingPhased(p.ID()) != nil) != survives {
				t.Fatal("wrong sacrifice identity or phasing")
			}
			if change == "source-phase" {
				g.PhaseIn([]uuid.UUID{p.ID()})
				g.RunStepWithPriority(core.EndStep)
				if g.FindPermanent(p.ID()) == nil {
					t.Fatal("expired delay repeated")
				}
			}
		})
	}
}

func TestRONActivationHistoryAllReprints(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	selected := fstest.MapFS{}
	counts := map[string]int{}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(bytes.NewReader(data), filepath.Base(file))
		if err != nil {
			t.Fatal(err)
		}
		var records []string
		lines := strings.Split(string(data), "\n")
		for _, node := range nodes {
			name := node.Fields["Name"].String
			if name != "Dragon Whelp" && name != "Nalathni Dragon" {
				continue
			}
			counts[name]++
			records = append(records, lines[node.Position.Line-1])
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			g, a, _ := ronGame()
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Red, 4)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if p == nil || !p.HasKeyword(core.Flying) || p.HasKeyword(core.Banding) != (name == "Nalathni Dragon") {
				t.Fatal("entry or keywords")
			}
			a.ManaPool().Add(core.Red, 4)
			for range 4 {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
			}
			g.RunStepWithPriority(core.EndStep)
			if g.FindPermanent(p.ID()) != nil {
				t.Fatal("reprint missed sacrifice")
			}
		}
		if len(records) > 0 {
			selected[filepath.Base(file)] = &fstest.MapFile{Data: []byte(strings.Join(records, "\n"))}
		}
	}
	if counts["Dragon Whelp"] != 6 || counts["Nalathni Dragon"] != 1 {
		t.Fatal(counts)
	}
	registry, err := LoadFS(selected, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("reprint canonicalization")
	}
	report, err := InspectFS(selected, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 7 || report.UniqueCards != 2 {
		t.Fatal("inventory counts")
	}
	roles := map[string]bool{}
	for _, sig := range report.Signatures {
		if sig.Name == "TimesThisAbilityHasBeenActivatedThisTurnIs" && sig.Role == "activation_history_condition" {
			roles["condition"] = true
		}
		if sig.Name == "GreaterThanOrEqualTo" && sig.Role == "activation_history_comparison" {
			roles["comparison"] = true
		}
	}
	if len(roles) != 2 {
		t.Fatal("missing history roles")
	}
	selected["unsupported.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Unsupported",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[UnknownRule])`)}
	if partial, err := LoadFS(selected, "*.ron"); err == nil || partial != nil {
		t.Fatal("partial registry published")
	}
}
