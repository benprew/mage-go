package ron

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronEntryExilePlayer struct {
	*mage.BasePlayer
	modes   []int
	choices int
}

func (p *ronEntryExilePlayer) ChooseMode(_ []string, _ string) int {
	p.choices++
	if len(p.modes) == 0 {
		return 0
	}
	n := p.modes[0]
	p.modes = p.modes[1:]
	return n
}
func TestRONEntryGraveyardPaymentMonster(t *testing.T) {
	for _, scenario := range []string{"split", "zero", "short", "lost", "countered"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronEntryExilePlayer{BasePlayer: mage.NewBasePlayer("A"), modes: []int{0, 1, 2}}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetActivePlayerIndex(0)
			g.SetTurn(1)
			g.SetStep(core.PrecombatMain)
			card := repositoryCard(t, "DRK", "Frankenstein's Monster")
			addRONHandCard(a.BasePlayer, card)
			x := 3
			if scenario == "zero" {
				x = 0
			}
			offerings := make([]mage.Card, 0, 3)
			for i := range 3 {
				c := mage.NewCreature(fmt.Sprint(i), "", 1, 1)
				c.SetOwner(a.PlayerID())
				offerings = append(offerings, c)
				if scenario != "short" || i < 2 {
					a.AddToGraveyard(c)
				}
			}
			a.AddToGraveyard(mage.NewArtifact("Wrong type", ""))
			b.AddToGraveyard(mage.NewCreature("Wrong player", "", 1, 1))
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, x); err == nil {
				t.Fatal("cast without mana")
			}
			a.ManaPool().Add(core.Black, 2)
			a.ManaPool().Add(core.Colorless, x)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, x); err != nil {
				t.Fatal(err)
			}
			if len(g.GetExile()) != 0 || a.choices != 0 {
				t.Fatal("entry payment at announcement")
			}
			if scenario == "countered" {
				g.CounterStackObject(g.StackPeek().ID)
				if len(g.GetExile()) != 0 {
					t.Fatal("countered spell paid")
				}
				return
			}
			if scenario == "lost" {
				a.RemoveFromGraveyard(offerings[0].ID())
			}
			g.SetXValue(99)
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if scenario == "short" || scenario == "lost" {
				if p != nil || len(g.GetExile()) != 0 || a.choices != 0 || !slices.ContainsFunc(a.Graveyard(), func(c mage.Card) bool { return c.ID() == card.ID() }) {
					t.Fatal("partial or failed payment entered")
				}
				return
			}
			if p == nil {
				t.Fatal("missing entrant")
			}
			if scenario == "zero" {
				if p.CurrentPower(g) != 0 || p.CurrentToughness(g) != 1 || a.choices != 0 || len(g.GetExile()) != 0 || len(a.Graveyard()) != 4 {
					t.Fatal("zero payment")
				}
				return
			}
			if len(g.GetExile()) != 3 || p.Counters[core.P2P0] != 1 || p.Counters[core.P1P1] != 1 || p.Counters[core.P0P2] != 1 || p.CurrentPower(g) != 3 || p.CurrentToughness(g) != 4 {
				t.Fatal("wrong allocation or characteristics")
			}
			g.PhaseIn(g.PhaseOut(p.ID()))
			if p.Counters[core.P2P0] != 1 {
				t.Fatal("phasing lost counters")
			}
			if err := mage.ApplyEffect(g, mage.ReturnToHand(mage.This), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if _, ok := a.RemoveFromHand(card.ID()); !ok {
				t.Fatal("return did not reach owner's hand")
			}
			if direct := g.PutOnBattlefield(card, a.PlayerID()); direct == nil || direct.CurrentPower(g) != 0 || direct.CurrentToughness(g) != 1 {
				t.Fatal("return retained entry result")
			}
		})
	}
}
func TestRONEntryGraveyardPaymentDeclarations(t *testing.T) {
	original := repositoryRecord(t, "DRK", "Frankenstein's Monster", "entry-payment.ron")
	for _, pair := range [][2]string{{"MustCost", "MayCost"}, {"CostWasPaid", "CostWasntPaid"}, {"ValueX", "ValueX()"}, {"SinglePlayer(You)", "SinglePlayer(Opponent)"}, {"PutIntoGraveyardInstead", "PutIntoHandInstead"}, {"NumberOfCardsOfTypeExiledThisWay(IsCardtype(Creature))", "Integer(3)"}, {"PTCounter(+2, +0)", "PTCounter(+9, +0)"}, {"IfElse(CostWasPaid", "If(CostWasPaid"}} {
		t.Run(pair[1], func(t *testing.T) {
			line := readMonsterSource(t)
			line = strings.Replace(line, pair[0], pair[1], 1)
			nodes, err := Parse(strings.NewReader(line), "bad.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Compile(nodes[0]); err == nil {
				t.Fatal("invalid entry accepted")
			}
		})
	}
	if _, err := Compile(original); err != nil {
		t.Fatal(err)
	}
}

func readMonsterSource(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../../../cards/mtgish/DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Frankenstein's Monster"`) {
			return line
		}
	}
	t.Fatal("missing record")
	return ""
}

func TestRONEntryGraveyardPaymentCopiesAndClone(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "DRK", "Frankenstein's Monster")
	addRONHandCard(a, card)
	offering := mage.NewCreature("Offering", "", 1, 1)
	offering.SetOwner(a.PlayerID())
	a.AddToGraveyard(offering)
	a.ManaPool().Add(core.Black, 2)
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 1); err != nil {
		t.Fatal(err)
	}
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy missing")
	}
	clone := g.Clone()
	g.ResolveStack()
	if g.FindPermanent(card.ID()) == nil || g.FindPermanent(copied.Card.ID()) != nil || len(g.GetExile()) != 1 {
		t.Fatal("copy borrowed another controller's payment")
	}
	if len(clone.GetExile()) != 0 || len(clone.GetPlayer(a.PlayerID()).Graveyard()) != 1 {
		t.Fatal("pending clone changed")
	}
	clone.ResolveStack()
	if clone.FindPermanent(card.ID()) == nil || len(clone.GetExile()) != 1 {
		t.Fatal("pending clone lost payment")
	}
	first, second := repositoryCard(t, "DRK", "Frankenstein's Monster"), repositoryCard(t, "DRK", "Frankenstein's Monster")
	if first.ID() == second.ID() || first.Abilities()[0].AbilityID() == second.Abilities()[0].AbilityID() {
		t.Fatal("factories share identity")
	}
}

func TestRONEntryGraveyardPaymentLoadAndInventory(t *testing.T) {
	source := readMonsterSource(t)
	files := fstest.MapFS{"DRK.ron": {Data: []byte(source)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 || len(report.Issues) != 0 {
		t.Fatal("inventory", err)
	}
	want := map[string]string{"MustCost": "entry_operation", "Exile": "entry_cost", "NumberCardsInGraveyards": "entry_exile_selection", "NumberOfCardsOfTypeExiledThisWay": "entry_exile_value", "PTCounter": "permanent_counter_type"}
	for _, signature := range report.Signatures {
		if role, ok := want[signature.Name]; ok && signature.Role == role {
			delete(want, signature.Name)
		}
	}
	if len(want) != 0 {
		t.Fatal("missing roles", want)
	}
	files["Bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(source, "CostWasPaid", "CostWasntPaid", 1))}
	if err := LoadInto(registry, files, "*.ron"); err == nil {
		t.Fatal("invalid collection accepted")
	}
	if _, err := registry.CreateCard("Frankenstein's Monster"); err != nil {
		t.Fatal("invalid load changed registry")
	}
}
