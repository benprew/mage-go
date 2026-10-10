package ron

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDrawReplacementScopeEvents(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"LEA", "Island Sanctuary"}, {"LEG", "Chains of Mephistopheles"}} {
		t.Run(tc.name, func(t *testing.T) {
			record := staticBaseRecord(t, tc.set, tc.name)
			if _, err := Compile(record); tc.name == "Island Sanctuary" && err == nil {
				t.Fatal("complete Sanctuary still has unsupported operations")
			} else if tc.name == "Chains of Mephistopheles" && err != nil {
				t.Fatal(err)
			}
			rule := record.Fields["Rules"].Args[0]
			body, err := Parse(strings.NewReader(`PlayerAction(WouldDrawACard_ThatPlayer,GainLife(Integer(2)))`), "body.ron")
			if err != nil {
				t.Fatal(err)
			}
			rule.Args[1] = Node{Kind: List, Args: body}
			record.Fields["Rules"] = Node{Kind: List, Args: []Node{rule}}
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			g.SetTurn(3)
			g.SetStep(core.Draw)
			g.PutOnBattlefield(factory(), a.PlayerID())
			a.AddToLibrary(mage.NewLand("One"))
			a.AddToLibrary(mage.NewLand("Two"))
			b.AddToLibrary(mage.NewLand("Other"))
			g.PlayerDrawCard(a)
			g.PlayerDrawCard(a)
			g.PlayerDrawCard(b)
			if tc.name == "Island Sanctuary" {
				if a.Life() != 24 || len(a.Hand()) != 0 || b.Life() != 20 || len(b.Hand()) != 1 {
					t.Fatal("controller draw-step scope")
				}
			} else if a.Life() != 22 || len(a.Hand()) != 1 || b.Life() != 22 || len(b.Hand()) != 0 {
				t.Fatal("except-first-draw scope")
			}
		})
	}
}

func TestRONDrawReplacementScopeInvalid(t *testing.T) {
	valid := `ReplaceWouldDraw(PlayerWouldDrawDuringTheirDrawStep(You),[PlayerAction(WouldDrawACard_ThatPlayer,GainLife(Integer(2)))])`
	for _, rule := range []string{
		`ReplaceWouldDraw`, `ReplaceWouldDraw()`,
		strings.Replace(valid, "You)", "You())", 1),
		strings.Replace(valid, "You)", "Opponent)", 1),
		strings.Replace(valid, "PlayerWouldDrawDuringTheirDrawStep(You)", "APlayerWouldDrawExceptFirstDrawStepDraw(You)", 1),
		strings.Replace(valid, "WouldDrawACard_ThatPlayer", "Trigger_ThatPlayer", 1),
		strings.Replace(valid, "WouldDrawACard_ThatPlayer", "WouldDrawACard_ThatPlayer()", 1),
		strings.Replace(valid, "GainLife(Integer(2))", "PlayerMustCost(WouldDrawACard_ThatPlayer,DiscardACard)", 1),
		strings.Replace(valid, "[PlayerAction(WouldDrawACard_ThatPlayer,GainLife(Integer(2)))]", "[]", 1),
		`SpellActions(ActionList([PlayerAction(WouldDrawACard_ThatPlayer,DrawACard)]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration", rule)
		}
	}
}

func TestRONDrawReplacementScopeGeneratedDraws(t *testing.T) {
	rule := `ReplaceWouldDraw(PlayerWouldDrawDuringTheirDrawStep(You),[PlayerAction(WouldDrawACard_ThatPlayer,GainLife(Integer(1))),PlayerAction(WouldDrawACard_ThatPlayer,DrawACard)])`
	factory := permanentScopeFactory(t, "Enchantment", rule)
	g, a, b := ronGame()
	g.SetTurn(3)
	g.SetStep(core.PrecombatMain)
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 {
		t.Fatal("replacement ran during casting")
	}
	g.ResolveStack()
	seedRONDrawLibrary(a)
	seedRONDrawLibrary(b)
	clone := g.Clone()
	g.RunStepWithPriority(core.Draw)
	if len(a.Hand()) != 1 || a.Life() != 21 || b.Life() != 20 || g.StackSize() != 0 {
		t.Fatal("normal draw or immediate timing")
	}
	g.PlayerDrawCard(a)
	if len(a.Hand()) != 2 || a.Life() != 22 {
		t.Fatal("effect draw recursed or shared exclusions")
	}
	clone.RunStepWithPriority(core.Draw)
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 || a.Life() != 22 {
		t.Fatal("clone isolation")
	}
}

func TestRONDrawReplacementScopeInventory(t *testing.T) {
	source := `Card(Name:"Rule",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[ReplaceWouldDraw(APlayerWouldDrawExceptFirstDrawStepDraw(AnyPlayer),[PlayerAction(WouldDrawACard_ThatPlayer,DrawACard)])])`
	nodes, err := Parse(strings.NewReader(source), "inventory.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries := make(map[string]*InventorySignature)
	walkInventory(nodes[0], "Rule", "card", "card", "card", true, entries)
	for name, role := range map[string]string{"APlayerWouldDrawExceptFirstDrawStepDraw": "draw_replacement_event", "AnyPlayer": "player_filter", "PlayerAction": "action", "WouldDrawACard_ThatPlayer": "player_reference", "DrawACard": "player_action"} {
		found := false
		for _, entry := range entries {
			if entry.Name == name {
				found = true
				if entry.Role != role {
					t.Fatalf("%s role=%s", name, entry.Role)
				}
			}
		}
		if !found {
			t.Fatal("missing signature", name)
		}
	}
}

func TestRONDrawReplacementScopeDepartureLKI(t *testing.T) {
	source := `Card(Name:"Rule",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:3,Toughness:2),Rules:[ReplaceWouldDraw(APlayerWouldDrawExceptFirstDrawStepDraw(AnyPlayer),[SacrificePermanent(ThisPermanent),PlayerAction(WouldDrawACard_ThatPlayer,GainLife(PowerOfPermanent(ThisPermanent)))])])`
	nodes, err := Parse(strings.NewReader(source), "departure.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	g.SetTurn(3)
	g.SetStep(core.Upkeep)
	card := factory()
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	seedRONDrawLibrary(b)
	g.PlayerDrawCard(b)
	if b.Life() != 23 || a.Life() != 20 || g.FindPermanent(card.ID()) != nil || len(a.Graveyard()) != 1 || len(b.Hand()) != 0 {
		t.Fatal("departure lost drawing player or source LKI")
	}
	g.PutOnBattlefield(card, b.PlayerID())
	g.PlayerDrawCard(b)
	if b.Life() != 26 || g.FindPermanent(card.ID()) != nil || len(b.Hand()) != 0 {
		t.Fatal("returned rule used stale exclusion")
	}
}
