package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const ronLifeGainDrawRule = `ReplaceWouldGainLife(PlayerWouldGainLife(You),[DrawNumberCards(WouldGainLife_LifeAmount)])`

func TestRONLifeGainDrawLich(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			record := staticBaseRecord(t, set, "Lich")
			rules := record.Fields["Rules"].Args
			if _, err := Compile(record); err == nil {
				t.Fatal("incomplete Lich accepted")
			}
			factory := staticBaseSubrule(t, record, rules[0], rules[1], rules[2])
			g, a, b := ronGame()
			for range 10 {
				a.AddToLibrary(mage.NewLand("Draw"))
				b.AddToLibrary(mage.NewLand("Draw"))
			}
			card := factory()
			card.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.PlayerGainLife(a, 2)
			g.CheckStateBasedActions()
			if a.Life() != 0 || !a.IsAlive() || len(a.Hand()) != 2 || g.PlayerLifeGainedThisTurn(a.PlayerID()) != 0 {
				t.Fatal("Lich clauses failed together")
			}
			clone := g.Clone()
			clone.PhaseOut(p.ID())
			clone.PlayerGainLife(clone.GetPlayer(a.PlayerID()), 1)
			if clone.GetPlayer(a.PlayerID()).Life() != 1 || a.Life() != 0 {
				t.Fatal("phasing or clone isolation")
			}
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			g.PlayerGainLife(b, 2)
			if b.Life() != 20 || len(b.Hand()) != 2 {
				t.Fatal("current controller")
			}
		})
	}
}

func TestRONLifeGainDrawCastingAndInstances(t *testing.T) {
	record := staticBaseRecord(t, "LEA", "Lich")
	rules := record.Fields["Rules"].Args
	factory := staticBaseSubrule(t, record, rules[0], rules[1], rules[2])
	g, a, _ := ronGame()
	for range 10 {
		a.AddToLibrary(mage.NewLand("Draw"))
	}
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid casting accepted")
	}
	if a.Life() != 20 || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("failed casting changed state")
	}
	a.RemoveFromHand(card.ID())
	castRONSpell(t, g, a, card)
	pending := g.Clone()
	g.ResolveStack()
	pending.ResolveStack()
	g.PlayerGainLife(a, 2)
	pending.PlayerGainLife(pending.GetPlayer(a.PlayerID()), 1)
	if a.Life() != 0 || len(a.Hand()) != 2 || len(pending.GetPlayer(a.PlayerID()).Hand()) != 1 {
		t.Fatal("resolved casting or pending clone")
	}
	other := g.PutOnBattlefield(factory(), a.PlayerID())
	g.PhaseOut(card.ID())
	g.PlayerGainLife(a, 1)
	if len(a.Hand()) != 3 {
		t.Fatal("independent instance lost")
	}
	g.BouncePermanentToHand(other)
	g.PlayerGainLife(a, 1)
	if a.Life() != 1 {
		t.Fatal("departed source kept rule")
	}
}

func TestRONLifeGainDrawInvalid(t *testing.T) {
	for _, rule := range []string{
		`ReplaceWouldGainLife`,
		`ReplaceWouldGainLife()`,
		`ReplaceWouldGainLife(PlayerWouldGainLife(You))`,
		`ReplaceWouldGainLife(PlayerWouldGainLife(You()),[DrawNumberCards(WouldGainLife_LifeAmount)])`,
		`ReplaceWouldGainLife(PlayerWouldGainLife(Opponent),[DrawNumberCards(WouldGainLife_LifeAmount)])`,
		`ReplaceWouldGainLife(PlayerWouldGainLife(ActionPlayer),[DrawNumberCards(WouldGainLife_LifeAmount)])`,
		`ReplaceWouldGainLife(PlayerWouldGainLife(You,You),[DrawNumberCards(WouldGainLife_LifeAmount)])`,
		`ReplaceWouldGainLife(PlayerWouldLoseLife(You),[DrawNumberCards(WouldGainLife_LifeAmount)])`,
		`ReplaceWouldGainLife(PlayerWouldGainLife(You),[])`,
		`ReplaceWouldGainLife(PlayerWouldGainLife(You),[DrawNumberCards(WouldGainLife_LifeAmount),DrawNumberCards(WouldGainLife_LifeAmount)])`,
		`ReplaceWouldGainLife(PlayerWouldGainLife(You),[DrawNumberCards(Integer(1))])`,
		`ReplaceWouldGainLife(PlayerWouldGainLife(You),[DrawNumberCards(WouldGainLife_LifeAmount())])`,
		`ReplaceWouldGainLife(PlayerWouldGainLife(You),[GainLife(WouldGainLife_LifeAmount)])`,
		`SpellActions(ActionList([DrawNumberCards(WouldGainLife_LifeAmount)]))`,
		`Activated(PayMana([]),ActionList([` + ronLifeGainDrawRule + `]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[`+rule+`])`), "invalid-life-draw.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
	for _, kind := range []string{"Instant", "Sorcery"} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[`+kind+`],Subtypes:[]),Rules:[`+ronLifeGainDrawRule+`])`), "invalid-life-draw.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("nonpermanent accepted")
		}
	}
}

func TestRONLifeGainDrawInventoryAndResolution(t *testing.T) {
	source := `Card(Name:"Rule",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[` + ronLifeGainDrawRule + `])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 {
		t.Fatalf("rule rejected: %+v", report.Issues)
	}
	want := map[string]string{"PlayerWouldGainLife": "life_gain_replacement_event", "You": "player_reference", "DrawNumberCards": "life_gain_replacement_operation", "WouldGainLife_LifeAmount": "life_gain_replacement_value"}
	for _, signature := range report.Signatures {
		if role, ok := want[signature.Name]; ok {
			if signature.Role != role {
				t.Errorf("%s role=%s, want %s", signature.Name, signature.Role, role)
			}
			delete(want, signature.Name)
		}
	}
	if len(want) != 0 {
		t.Fatal("missing signatures", want)
	}
	g, a, _ := ronGame()
	for range 5 {
		a.AddToLibrary(mage.NewLand("Draw"))
	}
	rule := permanentScopeFactory(t, "Enchantment", ronLifeGainDrawRule)
	g.PutOnBattlefield(rule(), a.PlayerID())
	spell := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([GainLife(Integer(2)),GainLife(Integer(1))]))`)
	castRONSpell(t, g, a, spell())
	if len(a.Hand()) != 0 {
		t.Fatal("announcement drew cards")
	}
	g.ResolveStack()
	if a.Life() != 20 || len(a.Hand()) != 3 || g.StackSize() != 0 || g.PlayerLifeGainedThisTurn(a.PlayerID()) != 0 {
		t.Fatal("RON gain instructions did not use replacement")
	}
}
