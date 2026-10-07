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

func TestRONBoundedCountersFullCards(t *testing.T) {
	for _, tc := range []struct {
		set, name   string
		limit, cost int
	}{
		{"ATQ", "Clockwork Avian", 4, 5}, {"LEA", "Clockwork Beast", 7, 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, _ := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast")
			}
			a.ManaPool().Add(core.Colorless, tc.cost)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()) != nil {
				t.Fatal("early entry")
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if int(p.Counters[core.P1P0]) != tc.limit || p.CurrentPower(g) != tc.limit || p.CurrentToughness(g) != 4 || p.HasAttr(core.Flying) != (tc.name == "Clockwork Avian") {
				t.Fatal("entry metadata")
			}
			g.SetStep(core.Upkeep)
			g.SetXValue(3)
			a.ManaPool().Add(core.Colorless, 3)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), boundedActivationIndex(t, p), nil); err == nil {
				t.Fatal("sick tap activation")
			}
			p.RevokeBaseAttr(core.AttrSummonSick)
			g.MutablePermanent(p.ID()).RemoveCounter(core.P1P0, 3)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), boundedActivationIndex(t, p), nil); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 0 || !g.FindPermanent(p.ID()).Tapped || int(g.FindPermanent(p.ID()).Counters[core.P1P0]) != tc.limit-3 {
				t.Fatal("cost or early restoration")
			}
			g.SetXValue(99)
			clone := g.Clone()
			g.ResolveStack()
			if int(g.FindPermanent(p.ID()).Counters[core.P1P0]) != tc.limit || int(clone.FindPermanent(p.ID()).Counters[core.P1P0]) != tc.limit-3 {
				t.Fatal("restoration or clone")
			}
			clone.ResolveStack()
			if int(clone.FindPermanent(p.ID()).Counters[core.P1P0]) != tc.limit {
				t.Fatal("clone restoration")
			}
			if err := mage.ValidateCardDefinition(tc.name, card); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func boundedActivationIndex(t *testing.T, p *mage.Permanent) int {
	t.Helper()
	for i, ability := range p.RuntimeAbilities {
		if action, ok := mage.UnwrapAbility(ability).(*mage.ActionDefinition); ok && action.Kind() == mage.ActionActivated {
			return i
		}
	}
	t.Fatal("missing activated ability")
	return -1
}

type ronBoundedPlayer struct {
	*mage.BasePlayer
	choice    int
	bounds    []int
	modes     int
	modeCalls int
}

func (p *ronBoundedPlayer) ChooseNumber(_, upper int, _ string) int {
	p.bounds = append(p.bounds, upper)
	if p.choice == -1 {
		return upper
	}
	return p.choice
}
func (p *ronBoundedPlayer) ChooseMode(_ []string, _ string) int { p.modeCalls++; return p.modes }

func TestRONBoundedCountersResolution(t *testing.T) {
	for _, name := range []string{"Clockwork Avian", "Clockwork Beast"} {
		for _, scenario := range []string{"partial", "decline", "zero", "full", "overfull", "current-room", "current-X", "left", "returned", "phased", "cycle", "control", "copy", "remove-ability", "unpaid", "wrong-step", "opponent-upkeep", "sick", "wrong-controller"} {
			t.Run(name+"/"+scenario, func(t *testing.T) {
				a := &ronBoundedPlayer{BasePlayer: mage.NewBasePlayer("A"), choice: -1}
				b := &ronBoundedPlayer{BasePlayer: mage.NewBasePlayer("B"), choice: 0}
				g := mage.NewGame(a, b)
				g.SetTurn(1)
				g.SetStep(core.Upkeep)
				set, limit := "ATQ", 4
				if name == "Clockwork Beast" {
					set, limit = "LEA", 7
				}
				card := repositoryCard(t, set, name)
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				index := boundedActivationIndex(t, p)
				p.RevokeBaseAttr(core.AttrSummonSick)
				p.RemoveCounter(core.P1P0, limit-1)
				initial, x, mana := 1, 3, 3
				payer := a
				switch scenario {
				case "partial":
					a.choice = 1
				case "decline":
					a.choice = 0
				case "zero":
					x, mana = 0, 0
				case "full":
					p.AddCounter(core.P1P0, limit-1)
					initial = limit
				case "overfull":
					p.AddCounter(core.P1P0, limit+1)
					initial = limit + 2
				case "unpaid":
					mana = 2
				case "wrong-step":
					g.SetStep(core.Draw)
				case "opponent-upkeep":
					g.SetActivePlayerIndex(1)
				case "sick":
					p.GrantBaseAttr(core.AttrSummonSick)
				case "wrong-controller":
					payer = b
				}
				payer.ManaPool().Add(core.Colorless, mana)
				g.SetXValue(x)
				err := g.ActivateAbilityByIndex(payer.PlayerID(), p.ID(), index, nil)
				illegal := scenario == "unpaid" || scenario == "wrong-step" || scenario == "opponent-upkeep" || scenario == "sick" || scenario == "wrong-controller"
				if illegal {
					if err == nil || payer.ManaPool().TotalMana() != mana || g.FindPermanent(p.ID()).Tapped || g.StackSize() != 0 || len(a.bounds) != 0 {
						t.Fatal("illegal activation or rollback")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if len(a.bounds) != 0 || !g.FindPermanent(p.ID()).Tapped || payer.ManaPool().TotalMana() != 0 || int(g.FindPermanent(p.ID()).Counters[core.P1P0]) != initial {
					t.Fatal("announcement timing or costs")
				}
				want := min(initial+x, limit)
				switch scenario {
				case "partial":
					want = 2
				case "decline", "zero":
					want = initial
				case "overfull":
					want = initial
				case "current-room":
					g.MutablePermanent(p.ID()).AddCounter(core.P1P0, limit-2)
					want = limit
				case "current-X":
					g.SetXValue(99)
				case "left", "returned":
					g.BouncePermanentToHand(g.FindPermanent(p.ID()))
					if scenario == "returned" {
						a.RemoveFromHand(card.ID())
						g.PutOnBattlefield(card, a.PlayerID())
						want = limit
					}
				case "phased":
					g.PhaseOut(p.ID())
					want = initial
				case "cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "copy":
					if copied := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false); copied == nil || copied.XValue != x {
						t.Fatal("copy X")
					}
				case "remove-ability":
					g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
				}
				clone := g.Clone()
				before := g.FindPermanentIncludingPhased(p.ID())
				g.ResolveStack()
				if live := g.FindPermanentIncludingPhased(p.ID()); live != nil && int(live.Counters[core.P1P0]) != want {
					t.Fatalf("counters %d want %d", live.Counters[core.P1P0], want)
				}
				if before != nil && int(clone.FindPermanentIncludingPhased(p.ID()).Counters[core.P1P0]) != int(before.Counters[core.P1P0]) {
					t.Fatal("clone shared mutation")
				}
				if scenario == "left" || scenario == "returned" || scenario == "phased" || scenario == "zero" || scenario == "full" || scenario == "overfull" {
					if len(a.bounds) != 0 {
						t.Fatal("choice for absent or full source")
					}
				} else if len(a.bounds) != 1 {
					t.Fatal("missing controller choice")
				}
				if scenario == "copy" && len(b.bounds) != 1 {
					t.Fatal("copy choice did not use its controller")
				}
				clone.ResolveStack()
				cloneWant := want
				if scenario == "partial" || scenario == "decline" {
					cloneWant = min(initial+x, limit)
				}
				if scenario == "copy" {
					cloneWant = min(initial+2*x, limit)
				}
				if live := clone.FindPermanentIncludingPhased(p.ID()); live != nil && int(live.Counters[core.P1P0]) != cloneWant {
					t.Fatal("clone pending resolution")
				}
			})
		}
	}
}

const boundedCounterAction = `PutCountersWithRestriction([UptoNumberCountersOfTypeOnPermanent(ValueX,PTCounter(1,0),ThisPermanent)],CantCauseTheNumberOfCountersOfTypeOnPermanentToBeGreaterThan(PTCounter(1,0),ThisPermanent,Integer(7)))`

func TestRONBoundedCountersInvalid(t *testing.T) {
	for _, action := range []string{
		`PutCountersWithRestriction`,
		`PutCountersWithRestriction([])`,
		strings.Replace(boundedCounterAction, `],CantCause`, `,UptoNumberCountersOfTypeOnPermanent(ValueX,PTCounter(1,0),ThisPermanent)],CantCause`, 1),
		strings.Replace(boundedCounterAction, `UptoNumberCountersOfTypeOnPermanent`, `NumberCountersOfTypeOnPermanent`, 1),
		strings.Replace(boundedCounterAction, `ValueX`, `Trigger_ThatPlayer`, 1),
		strings.Replace(boundedCounterAction, `ValueX`, `ValueX()`, 1),
		strings.Replace(boundedCounterAction, `Integer(7)`, `ValueX`, 1),
		strings.Replace(boundedCounterAction, `Integer(7)`, `Integer(-1)`, 1),
		strings.Replace(boundedCounterAction, `Integer(7)`, `Integer(256)`, 1),
		strings.Replace(boundedCounterAction, `Integer(7)`, `Integer()`, 1),
		strings.Replace(boundedCounterAction, `Integer(7)`, `Integer(7,8)`, 1),
		strings.Replace(boundedCounterAction, `Integer(7)`, `Integer("7")`, 1),
		strings.Replace(boundedCounterAction, `CantCauseTheNumberOfCountersOfTypeOnPermanentToBeGreaterThan`, `OtherRestriction`, 1),
		strings.Replace(boundedCounterAction, `Than(PTCounter(1,0)`, `Than(PTCounter(1,1)`, 1),
		strings.Replace(boundedCounterAction, `Than(PTCounter(1,0),ThisPermanent`, `Than(PTCounter(1,0),Ref_TargetPermanent`, 1),
		strings.ReplaceAll(boundedCounterAction, `ThisPermanent`, `Ref_TargetPermanent`),
		strings.ReplaceAll(boundedCounterAction, `ThisPermanent`, `HostPermanent`),
		strings.ReplaceAll(boundedCounterAction, `ThisPermanent`, `ThisPermanent()`),
		strings.ReplaceAll(boundedCounterAction, `PTCounter(1,0)`, `PTCounter(9,9)`),
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayManaX([ManaCostX],ValueX),ActionList([` + action + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", action)
		}
	}
	for _, rule := range []string{
		`Activated(PayMana([]),ActionList([` + boundedCounterAction + `]))`,
		`Activated(` + boundedCounterAction + `,ActionList([GainLife(Integer(1))]))`,
		`PermanentLayerEffect(ThisPermanent,[` + boundedCounterAction + `])`,
		`AsPermanentEnters(ThisPermanent,[` + boundedCounterAction + `])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted context", rule)
		}
	}
}

func TestRONBoundedCountersInventory(t *testing.T) {
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayManaX([ManaCostX],ValueX),ActionList([` + boundedCounterAction + `]))])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatalf("inventory %v", err)
	}
	roles := map[string]string{"UptoNumberCountersOfTypeOnPermanent": "counter_instruction", "CantCauseTheNumberOfCountersOfTypeOnPermanentToBeGreaterThan": "counter_placement_restriction", "PTCounter": "permanent_counter_type", "ThisPermanent": "permanent_reference", "Integer": "value"}
	for _, s := range report.Signatures {
		if want, ok := roles[s.Name]; ok && s.Role != want {
			t.Fatalf("%s role %s want %s", s.Name, s.Role, want)
		}
	}
}

func TestRONBoundedCountersReplacements(t *testing.T) {
	for _, tc := range []struct {
		name         string
		choice, want int
		prevent      bool
	}{
		{"add-first", 0, 4, false}, {"double-first", 1, 3, false}, {"limit", 0, 7, false}, {"prevent", 0, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &counterReplacementChooser{BasePlayer: mage.NewBasePlayer("A"), index: 1 - tc.choice}
			b := &counterReplacementChooser{BasePlayer: mage.NewBasePlayer("B"), index: tc.choice}
			g := mage.NewGame(a, b)
			action := strings.ReplaceAll(strings.ReplaceAll(boundedCounterAction, "ValueX", "Integer(1)"), "ThisPermanent", "Ref_TargetPermanent")
			if tc.name == "limit" {
				action = strings.ReplaceAll(action, "Integer(1)", "Integer(6)")
			}
			action = strings.ReplaceAll(action, "PTCounter(1,0)", "PTCounter(1,1)")
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([`+action+`,GainLife(Integer(1))])))`)
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			p := g.PutOnBattlefield(mage.NewCreature("Target", "", 1, 8), b.PlayerID())
			g.AddReplacementEffect(&counterPlacementReplacement{source: source.ID(), prevent: tc.prevent})
			g.AddCounterDoubler(source.ID(), core.P1P1, mage.AnyPermanent)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if int(g.FindPermanent(p.ID()).Counters[core.P1P1]) != tc.want || a.Life() != 21 || b.choices != 1 || a.choices != 0 {
				t.Fatal("replacement order, restriction or chooser")
			}
		})
	}
}

func TestRONBoundedCountersInstances(t *testing.T) {
	g, a, b := ronGame()
	first := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Clockwork Avian"), a.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Clockwork Avian"), b.PlayerID())
	for _, p := range []*mage.Permanent{first, second} {
		p.RevokeBaseAttr(core.AttrSummonSick)
		p.RemoveCounter(core.P1P0, 4)
	}
	g.SetStep(core.Upkeep)
	g.SetXValue(1)
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), boundedActivationIndex(t, first), nil); err != nil {
		t.Fatal(err)
	}
	g.SetActivePlayerIndex(1)
	g.SetXValue(3)
	b.ManaPool().Add(core.Colorless, 3)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), second.ID(), boundedActivationIndex(t, second), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(first.ID()).Counters[core.P1P0] != 0 || g.FindPermanent(second.ID()).Counters[core.P1P0] != 3 {
		t.Fatal("pending instances overlap")
	}
	g.ResolveStack()
	if g.FindPermanent(first.ID()).Counters[core.P1P0] != 1 {
		t.Fatal("original X lost")
	}
}

func TestRONBoundedCountersTypedTarget(t *testing.T) {
	for _, scenario := range []string{"normal", "return", "phase", "shroud", "control", "copy"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			action := strings.ReplaceAll(strings.ReplaceAll(boundedCounterAction, "ValueX", "Integer(2)"), "ThisPermanent", "Ref_TargetPermanent")
			nodes, err := Parse(strings.NewReader(`Card(Name:"Counter spell",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([`+action+`])))])`), "target.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			creature := mage.NewCreature("Subject", "", 1, 8)
			creature.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(creature, b.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{a.PlayerID()}, 0); err == nil {
				t.Fatal("player accepted as permanent")
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "return":
				g.BouncePermanentToHand(p)
				b.RemoveFromHand(creature.ID())
				g.PutOnBattlefield(creature, b.PlayerID())
			case "phase":
				g.PhaseOut(p.ID())
			case "shroud":
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
			}
			g.ResolveStack()
			want := 2
			if scenario == "return" || scenario == "phase" || scenario == "shroud" {
				want = 0
			}
			if scenario == "copy" {
				want = 4
			}
			if int(g.FindPermanentIncludingPhased(p.ID()).Counters[core.P1P0]) != want {
				t.Fatal("target legality or identity")
			}
		})
	}
}

func TestRONBoundedCountersStrictRegistry(t *testing.T) {
	files := fstest.MapFS{}
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI", "ATQ"} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "Clockwork Beast"`) || strings.Contains(line, `Name: "Clockwork Avian"`) {
				files[set+".ron"] = &fstest.MapFile{Data: []byte(line)}
			}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("reprints not canonical")
	}
	for _, name := range registry.RegisteredCardNames() {
		card, err := registry.CreateCard(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := mage.ValidateCardDefinition(name, card); err != nil {
			t.Fatal(err)
		}
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayManaX([ManaCostX],ValueX),ActionList([` + strings.ReplaceAll(boundedCounterAction, "Integer(7)", "Integer(-1)") + `]))])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("invalid collection modified registry")
	}
}
