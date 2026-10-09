package ron

import (
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage/core"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
)

type ronCardNameTester struct {
	mage.Player
	calls int
	name  string
}

func (p *ronCardNameTester) ChooseTargets(ids []uuid.UUID, _, _ int, _ *mage.Game) []uuid.UUID {
	return ids[len(ids)-1:]
}

func (p *ronCardNameTester) ChooseMode(names []string, _ string) int {
	p.calls++
	return slices.Index(names, p.name)
}

func TestRONCardNameChoiceGameplay(t *testing.T) {
	for _, kind := range []string{"spell", "activation", "trigger"} {
		t.Run(kind, func(t *testing.T) {
			g, base, b := ronGame()
			a := &ronCardNameTester{Player: base, name: "Pick Your Poison"}
			other := &ronCardNameTester{Player: b, name: "Fire"}
			g.SetPlayerAt(0, a)
			g.SetPlayerAt(1, other)
			action := `Targeted([TargetPlayer(AnyPlayer)],ActionList([ChooseACardName(AnyOracleCard),PlayerActions(Ref_TargetPlayer,[ChooseACardName(AnyOracleCard),GainLife(Integer(1))])]))`
			rule, typ := `SpellActions(`+action+`)`, "Instant"
			if kind == "activation" {
				rule, typ = `Activated(PayMana([]),`+action+`)`, "Artifact"
			}
			if kind == "trigger" {
				rule, typ = `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),`+action+`)`, "Artifact"
			}
			factory := permanentScopeFactory(t, typ, rule)
			card := factory()
			card.SetOwner(a.PlayerID())
			if kind == "spell" {
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
					t.Fatal(err)
				}
			} else {
				p := g.PutOnBattlefield(card, a.PlayerID())
				if kind == "activation" {
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
						t.Fatal(err)
					}
				} else {
					g.PutTriggersOnStack()
				}
			}
			if a.calls != 0 || other.calls != 0 {
				t.Fatal("choice ran before resolution")
			}
			clone := g.Clone()
			ca := &ronCardNameTester{Player: clone.GetPlayer(a.PlayerID()), name: "Ice"}
			cb := &ronCardNameTester{Player: clone.GetPlayer(b.PlayerID()), name: "Insectile Aberration"}
			clone.SetPlayerAt(0, ca)
			clone.SetPlayerAt(1, cb)
			g.ResolveStack()
			clone.ResolveStack()
			if a.calls != 1 || ca.calls != 1 {
				t.Fatal("missing or shared controller choice")
			}
			if other.calls != 1 || cb.calls != 1 || b.Life() != 21 {
				t.Fatal("wrong explicit chooser")
			}
			if factory().Abilities()[0] == card.Abilities()[0] {
				t.Fatal("shared factory declarations")
			}
		})
	}
}

func TestRONCardNameChoiceDeclarations(t *testing.T) {
	for _, body := range []string{`ChooseACardName`, `ChooseACardName()`, `ChooseACardName(AnyOracleCard())`, `ChooseACardName(AnyOracleCard,AnyOracleCard)`, `ChooseACardName(AnyCard)`, `ChooseACardName(IsCardtype(Creature))`, `GainLife(TheChosenName)`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+body+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	for _, rule := range []string{`Activated(ChooseACardName(AnyOracleCard),ActionList([GainLife(Integer(1))]))`, `AsPermanentEnters(ThisPermanent,[ChooseACardName(AnyOracleCard)])`, `PermanentLayerEffect(ThisPermanent,[ChooseACardName(AnyOracleCard)])`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid role")
		}
	}
	for _, name := range []string{"Nebuchadnezzar", "Petra Sphinx"} {
		if _, err := Compile(staticBaseRecord(t, "LEG", name)); err != nil {
			t.Fatalf("unexpected complete record support for %s: %v", name, err)
		}
	}
}

func TestRONCardNameChoicePendingSource(t *testing.T) {
	for _, state := range []string{"departed", "phased"} {
		g, base, b := ronGame()
		a := &ronCardNameTester{Player: base, name: "Ice"}
		g.SetPlayerAt(0, a)
		card := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([ChooseACardName(AnyOracleCard),GainLife(Integer(1))]))`)()
		p := g.PutOnBattlefield(card, a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		if state == "departed" {
			g.BouncePermanentToHand(p)
		} else {
			g.PhaseOut(p.ID())
		}
		g.ResolveStack()
		if a.calls != 1 || a.Life() != 21 || b.Life() != 20 {
			t.Fatal("pending choice lost source")
		}
	}
}

func TestRONCardNameChoicePaymentAndCopy(t *testing.T) {
	g, base, b := ronGame()
	a := &ronCardNameTester{Player: base, name: "Fire"}
	other := &ronCardNameTester{Player: b, name: "Ice"}
	g.SetPlayerAt(0, a)
	g.SetPlayerAt(1, other)
	p := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(1)]),ActionList([ChooseACardName(AnyOracleCard)]))`)(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil || a.calls != 0 || g.StackSize() != 0 {
		t.Fatal("failed payment made a choice")
	}
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if a.calls != 1 || other.calls != 1 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("copies shared choice or payment failed")
	}
}

func TestRONCardNameChoiceInventory(t *testing.T) {
	source := `Card(Name:"Choice",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([ChooseACardName(AnyOracleCard)]))])`
	inventory, err := InspectFS(fstest.MapFS{"choice.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, sig := range inventory.Signatures {
		if sig.Name == "AnyOracleCard" && sig.Role == "card_name_catalog" {
			found = true
		}
	}
	if !found {
		t.Fatal("name catalog role missing")
	}
}

func TestRONCardNameChoiceRealDeclarations(t *testing.T) {
	for _, name := range []string{"Nebuchadnezzar", "Petra Sphinx"} {
		record := staticBaseRecord(t, "LEG", name)
		var choices []Node
		var visit func(Node)
		visit = func(n Node) {
			if n.Name == "ChooseACardName" {
				choices = append(choices, n)
			}
			for _, arg := range n.Args {
				visit(arg)
			}
			for _, field := range n.Fields {
				visit(field)
			}
		}
		visit(record)
		if len(choices) != 1 {
			t.Fatal("real record lost name choice")
		}
		build, err := compileCardNameChoice(choices[0], scope{})
		if err != nil {
			t.Fatal(err)
		}
		g, base, _ := ronGame()
		chooser := &ronCardNameTester{Player: base, name: "Pick Your Poison"}
		g.SetPlayerAt(0, chooser)
		if err := mage.ApplyEffect(g, build(instanceBindings{}), uuid.Nil, chooser.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if chooser.calls != 1 {
			t.Fatal("real name declaration did not execute")
		}
	}
}
