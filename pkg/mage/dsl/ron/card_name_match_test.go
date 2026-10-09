package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCardNameMatchDeclarations(t *testing.T) {
	plan := &cardNameChoicePlan{name: "chosen"}
	for _, text := range []string{`IsNamed`, `IsNamed()`, `IsNamed(TheChosenName,TheChosenName)`, `IsNamed(TheChosenName())`, `IsNamed(TheChosenPlayer)`, `IsNamed("Ice")`, `IsNamed(NamedCard("Ice"))`, `IsColor(TheChosenName)`} {
		if _, err := compileResolvingCardNameFilter(nameScopeNode(t, text), scope{chosenName: plan}); err == nil {
			t.Fatalf("accepted %s", text)
		}
	}
	if _, err := compileResolvingCardNameFilter(nameScopeNode(t, `IsNamed(TheChosenName)`), scope{}); err == nil {
		t.Fatal("unbound result accepted")
	}
}
func TestRONCardNameMatchRealOperands(t *testing.T) {
	for _, name := range []string{"Nebuchadnezzar", "Petra Sphinx"} {
		record := staticBaseRecord(t, "LEG", name)
		var choiceNode, filterNode Node
		var visit func(Node)
		visit = func(n Node) {
			if n.Name == "ChooseACardName" {
				choiceNode = n
			}
			if n.Name == "IsNamed" {
				filterNode = n
			}
			for _, a := range n.Args {
				visit(a)
			}
			for _, f := range n.Fields {
				visit(f)
			}
		}
		visit(record)
		choice, plan, err := compileCardNameChoiceWithResult(choiceNode, scope{})
		if err != nil {
			t.Fatal(err)
		}
		build, err := compileResolvingCardNameFilter(filterNode, scope{chosenName: plan})
		if err != nil {
			t.Fatal(err)
		}
		bindings := instanceBindings{chosenNames: make(map[*cardNameChoicePlan]*mage.CardNameBinding)}
		matcher := build(bindings)
		g, base, _ := ronGame()
		a := &ronCardNameTester{Player: base, name: "Eleven, the Mage"}
		g.SetPlayerAt(0, a)
		ctx := &mage.EffectContext{Game: g, Controller: a.PlayerID()}
		if matcher.Match([]string{"Cecily, Haunted Mage"}, ctx) || a.calls != 0 {
			t.Fatal("declaration chose or matched missing result")
		}
		if err := choice(bindings).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if !matcher.Match([]string{"Cecily, Haunted Mage"}, ctx) || matcher.Match([]string{"Other"}, ctx) || a.calls != 1 {
			t.Fatal("actual operand did not match chosen equivalent")
		}
		other := instanceBindings{chosenNames: make(map[*cardNameChoicePlan]*mage.CardNameBinding)}
		if build(other).Match([]string{"Cecily, Haunted Mage"}, ctx) {
			t.Fatal("instances share results")
		}
		if _, err := Compile(record); err != nil {
			t.Fatalf("unexpected complete record support for %s: %v", name, err)
		}
	}
}

func TestRONCardNameMatchGameplay(t *testing.T) {
	for _, state := range []string{"present", "departed", "returned", "phased"} {
		t.Run(state, func(t *testing.T) {
			g, base, b := ronGame()
			a := &ronCardNameTester{Player: base, name: "Eleven, the Mage"}
			g.SetPlayerAt(0, a)
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(1)]),Targeted([TargetPlayer(AnyPlayer)],ActionList([ChooseACardName(AnyOracleCard),PlayerAction(Ref_TargetPlayer,DiscardCards(IsNamed(TheChosenName)))])))`)
			p := g.PutOnBattlefield(factory(), a.PlayerID())
			for _, name := range []string{"Cecily, Haunted Mage", "Eleven, the Mage", "Other"} {
				c := mage.NewArtifact(name, "")
				c.SetOwner(b.PlayerID())
				b.AddToHand(c)
			}
			a.AddToHand(mage.NewArtifact("Cecily, Haunted Mage", ""))
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err == nil || a.calls != 0 || len(b.Hand()) != 3 {
				t.Fatal("failed payment changed choices or hand")
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			if a.calls != 0 || len(b.Hand()) != 3 {
				t.Fatal("choice or discard before resolution")
			}
			if state == "phased" {
				g.PhaseOut(p.ID())
			} else if state != "present" {
				g.BouncePermanentToHand(p)
				if state == "returned" {
					a.RemoveFromHand(p.ID())
					g.PutOnBattlefield(p.Card, a.PlayerID())
				}
			}
			clone := g.Clone()
			ca := &ronCardNameTester{Player: clone.GetPlayer(a.PlayerID()), name: "Island"}
			clone.SetPlayerAt(0, ca)
			g.ResolveStack()
			clone.ResolveStack()
			if len(b.Hand()) != 1 || b.Hand()[0].Name() != "Other" || len(b.Graveyard()) != 2 || a.calls != 1 {
				t.Fatal("named hand batch failed")
			}
			if len(clone.GetPlayer(b.PlayerID()).Hand()) != 3 || ca.calls != 1 {
				t.Fatal("clone shared result or discarded nonmatching cards")
			}
			want := 1
			if state == "departed" {
				want = 2
			}
			if len(a.Hand()) != want {
				t.Fatal("discard used wrong player")
			}
		})
	}
}

func TestRONCardNameMatchInvalidConsumers(t *testing.T) {
	for _, actions := range []string{
		`PlayerAction(You,DiscardCards(IsNamed(TheChosenName)))`,
		`PlayerActions(You,[DiscardCards(IsNamed(TheChosenName))]),ChooseACardName(AnyOracleCard)`,
		`ChooseACardName(AnyOracleCard),PlayerAction(You,DiscardCards(IsNamed(TheChosenName())))`,
		`ChooseACardName(AnyOracleCard),PlayerAction(You,DiscardCards(IsNamed(TheChosenPlayer)))`,
		`ChooseACardName(AnyOracleCard),PlayerAction(You,DiscardCards(IsNamed(TheChosenName,TheChosenName)))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+actions+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", actions)
		}
	}
}

func TestRONCardNameMatchScopedGameplay(t *testing.T) {
	g, base, b := ronGame()
	a := &ronCardNameTester{Player: base, name: "Ice"}
	other := &ronCardNameTester{Player: b, name: "Eleven, the Mage"}
	g.SetPlayerAt(0, a)
	g.SetPlayerAt(1, other)
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([ChooseACardName(AnyOracleCard),PlayerActions(Ref_TargetPlayer,[PlayerActions(ActionPlayer,[ChooseACardName(AnyOracleCard)]),DiscardCards(IsNamed(TheChosenName))]),PlayerAction(You,DiscardCards(IsNamed(TheChosenName)))])))`)
	for _, player := range []mage.Player{a, other} {
		card := mage.NewArtifact("Cecily, Haunted Mage", "")
		card.SetOwner(player.PlayerID())
		player.AddToHand(card)
	}
	spell := factory()
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	if a.calls != 0 || other.calls != 0 {
		t.Fatal("choice during announcement")
	}
	g.ResolveStack()
	if a.calls != 1 || other.calls != 1 || len(a.Hand()) != 0 || len(other.Hand()) != 0 || len(a.Graveyard()) != 2 || len(b.Graveyard()) != 1 {
		t.Fatal("nested name export or explicit player failed")
	}
}
