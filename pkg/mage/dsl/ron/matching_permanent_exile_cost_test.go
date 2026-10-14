package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const matchingExileRON = `Exile([APermanent(And([IsCardtype(Creature), ControlledByAPlayer(SinglePlayer(You))]))], [])`

func TestRONMatchingPermanentExileCostCity(t *testing.T) {
	for _, scenario := range []string{"success", "copy", "counter", "depart", "phase", "return"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			record := repositoryRecord(t, "DRK", "City of Shadows", "DRK.ron")
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			victimCard := mage.NewCreature("Victim", "", 2, 2)
			victimCard.SetOwner(b.PlayerID())
			victim := g.PutOnBattlefield(victimCard, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(victim.ID()) != nil || g.FindExiledCard(victimCard.ID()) == nil || p.Counters[core.Storage] != 0 {
				t.Fatal("payment did not precede resolution")
			}
			switch scenario {
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "counter":
				g.CounterStackObject(g.StackPeek().ID)
			case "depart", "return":
				g.RemoveFromBattlefield(p)
				if scenario == "return" {
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "phase":
				g.PhaseOut(p.ID())
			}
			clone := g.Clone()
			g.ResolveStack()
			expected := 1
			if scenario == "copy" {
				expected = 2
			}
			if scenario == "counter" || scenario == "depart" || scenario == "return" || scenario == "phase" {
				expected = 0
			}
			if current := g.FindPermanentIncludingPhased(p.ID()); current != nil && current.Counters[core.Storage] != expected {
				t.Fatal("wrong source counter result")
			}
			clone.ResolveStack()
			if len(g.GetExile()) != 1 || len(clone.GetExile()) != 1 {
				t.Fatal("copy paid again or clone lost payment")
			}
		})
	}
}

func TestRONMatchingPermanentExileCostAdditional(t *testing.T) {
	g, a, b := ronGame()
	record := repositoryRecord(t, "FEM", "Soul Exchange", "FEM.ron")
	costRule := record.Fields["Rules"].Args[0]
	nodes, err := Parse(strings.NewReader(`Card(Name:"Paid spell",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([GainLife(Integer(1))]))])`), "paid.ron")
	if err != nil {
		t.Fatal(err)
	}
	synthetic := nodes[0]
	synthetic.Fields["Rules"] = Node{Kind: List, Args: append([]Node{costRule}, synthetic.Fields["Rules"].Args...)}
	factory, err := Compile(synthetic)
	if err != nil {
		t.Fatal(err)
	}
	c := factory()
	victim := mage.NewCreature("Victim", "", 2, 2)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(victim, a.PlayerID())
	castRONSpell(t, g, a, c)
	if g.FindExiledCard(victim.ID()) == nil || a.Life() != 20 {
		t.Fatal("additional payment timing")
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("paid spell did not resolve")
	}
}

func TestRONMatchingPermanentExileCostInvalid(t *testing.T) {
	for _, cost := range []string{`Exile`, `Exile([],[])`, `Exile([APermanent()],[])`, `Exile([APermanent(IsCardtype(Creature))],[])`, `Exile([APermanent(And([]))],[])`, `Exile([APermanent(ControlledByAPlayer(Opponent))],[])`, `Exile([APermanent(Or([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You))]))],[])`, strings.ReplaceAll(matchingExileRON, "You", "You()"), strings.ReplaceAll(matchingExileRON, "You", "Trigger_ThatPlayer"), strings.Replace(matchingExileRON, ", [])", ", [ExilesFaceDown])", 1)} {
		t.Run(cost, func(t *testing.T) {
			nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(%s,ActionList([GainLife(Integer(1))]))])`, cost)), "bad.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Compile(nodes[0]); err == nil {
				t.Fatal("bad exile payment accepted")
			}
		})
	}
}

func TestRONMatchingPermanentExileCostRollback(t *testing.T) {
	for _, scenario := range []string{"success", "phased", "opponent", "wrong-type", "invalid", "double", "mana"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			cost := `And([PayMana([ManaCostGeneric(1)]),TapPermanent(ThisPermanent),` + matchingExileRON + `])`
			if scenario == "double" {
				cost = `And([PayMana([ManaCostGeneric(1)]),TapPermanent(ThisPermanent),` + matchingExileRON + `,` + matchingExileRON + `])`
			}
			c := sacrificeDeclaration(t, cost, `GainLife(Integer(1))`)
			source := g.PutOnBattlefield(c, a.PlayerID())
			victimCard := mage.NewCreature("Victim", "", 2, 2)
			victimCard.SetOwner(b.PlayerID())
			victimController := a.PlayerID()
			if scenario == "opponent" {
				victimController = b.PlayerID()
			}
			victim := g.PutOnBattlefield(victimCard, victimController)
			if scenario == "wrong-type" {
				g.RemoveFromBattlefield(victim)
				victim = g.PutOnBattlefield(mage.NewLand("Land"), a.PlayerID())
			}
			if scenario == "phased" {
				g.PhaseOut(victim.ID())
			}
			a.choice = victim.ID()
			if scenario == "invalid" {
				a.invalid = source
			}
			if scenario != "mana" {
				a.ManaPool().Add(core.Colorless, 1)
			}
			before := a.ManaPool().TotalMana()
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil)
			if scenario == "success" {
				if err != nil {
					t.Fatal(err)
				}
				if a.choices != 1 {
					t.Fatal("payment repeated choice")
				}
				g.ResolveStack()
				if a.Life() != 21 || len(g.GetExile()) != 1 {
					t.Fatal("paid body failed")
				}
			} else if err == nil || g.FindPermanentIncludingPhased(victim.ID()) == nil || g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != before || !g.GetStack().IsEmpty() || len(g.GetExile()) != 0 {
				t.Fatal("failed activation spent resources")
			}
		})
	}
}

func TestRONMatchingPermanentExileCostRecords(t *testing.T) {
	for _, card := range []struct{ set, name string }{{"FEM", "Soul Exchange"}} {
		record := repositoryRecord(t, card.set, card.name, card.set+".ron")
		if _, err := Compile(record); err != nil {
			t.Fatalf("complete %s must compile: %v", card.name, err)
		}
	}
	node, err := Parse(strings.NewReader(matchingExileRON), "payment.ron")
	if err != nil {
		t.Fatal(err)
	}
	subject := node[0].Args[0].Args[0]
	if inventoryArgumentRole(subject, "exile_payment_object", 0) != "permanent_filter" {
		t.Fatal("missing payment filter role")
	}
	for _, rule := range []string{`Activated(` + matchingExileRON + `,ActionList([GainLife(ManaValueOfExiled(TheCardExiledThisWay))]))`, `SpellActions(ActionList([` + matchingExileRON + `]))`, `AsPermanentEnters(ThisPermanent,[MustCost(` + matchingExileRON + `)])`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad context",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("unsupported context/result accepted")
		}
	}
}

func TestRONMatchingPermanentExileCostStrictLoad(t *testing.T) {
	source := `Card(Name:"Payment fixture",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(` + matchingExileRON + `,ActionList([GainLife(Integer(1))]))])`
	files := fstest.MapFS{"fixture.ron": {Data: []byte(source)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Payment fixture")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Payment fixture")
	if err != nil || first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("factory shared ability state", err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory rejected supported declaration", err)
	}
	for _, entry := range []struct{ set, name string }{{"LEG", "Wall of Caltrops"}} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + entry.set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		record := ""
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "`+entry.name+`"`) {
				record = line
			}
		}
		if record == "" {
			t.Fatal("missing complete record")
		}
		files["unsupported.ron"] = &fstest.MapFile{Data: []byte(record)}
		if err = LoadInto(registry, files, "*.ron"); err == nil || !registry.CardRegistered("Payment fixture") || registry.CardRegistered(entry.name) {
			t.Fatal("partial collection published")
		}
	}
}
