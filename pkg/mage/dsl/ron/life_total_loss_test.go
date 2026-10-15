package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const ronLifeTotalLossRule = `PlayerEffect(You,[DoesntLoseTheGameForHaving0OrLessLife])`

func TestRONLifeTotalLossLich(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			record := staticBaseRecord(t, set, "Lich")
			rules := record.Fields["Rules"].Args
			if _, err := Compile(record); err != nil {
				t.Fatalf("complete Lich rejected: %v", err)
			}
			factory := staticBaseSubrule(t, record, rules[0], rules[1])
			g, a, b := ronGame()
			c := factory()
			c.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.CheckStateBasedActions()
			if a.Life() != 0 || !a.IsAlive() || g.IsGameOver() || b.Life() != 20 {
				t.Fatal("entry and loss-exemption clauses failed")
			}
			clone := g.Clone()
			clone.PhaseOut(p.ID())
			clone.CheckStateBasedActions()
			if clone.GetPlayer(a.PlayerID()).IsAlive() || !a.IsAlive() {
				t.Fatal("clone or phasing")
			}
			g.PlayerLoseLife(a, 3)
			g.CheckStateBasedActions()
			if a.Life() != -3 || !a.IsAlive() {
				t.Fatal("negative life exemption failed")
			}
			g.BouncePermanentToHand(p)
			g.CheckStateBasedActions()
			if a.IsAlive() || g.Winner() != b.Name() {
				t.Fatal("source departure left exemption")
			}
		})
	}
}

func TestRONLifeTotalLossResolution(t *testing.T) {
	factory := permanentScopeFactory(t, "Enchantment", `AsPermanentEnters(ThisPermanent,[LoseLife(LifeTotalOfPlayer(You))]),`+ronLifeTotalLossRule)
	g, a, _ := ronGame()
	c := factory()
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 {
		t.Fatal("announcement changed life")
	}
	clone := g.Clone()
	g.ResolveStack()
	clone.ResolveStack()
	if a.Life() != 0 || !a.IsAlive() || clone.IsGameOver() {
		t.Fatal("resolved entry lost before static rule")
	}
	g.AddContinuousEffect(mage.RemoveAllAbilities(c.ID(), core.EndOfTurn))
	g.CheckStateBasedActions()
	if a.IsAlive() || !clone.GetPlayer(a.PlayerID()).IsAlive() {
		t.Fatal("ability removal or clone isolation")
	}
}

func TestRONLifeTotalLossInvalid(t *testing.T) {
	for _, rule := range []string{
		`PlayerEffect(You(),[DoesntLoseTheGameForHaving0OrLessLife])`,
		`PlayerEffect(Opponent,[DoesntLoseTheGameForHaving0OrLessLife])`,
		`PlayerEffect(ActionPlayer,[DoesntLoseTheGameForHaving0OrLessLife])`,
		`PlayerEffect(You,[DoesntLoseTheGameForHaving0OrLessLife()])`,
		`PlayerEffect(You,[DoesntLoseTheGameForHaving0OrLessLife(Integer(1))])`,
		`PlayerEffect(You,[DoesntLoseTheGameForHaving0OrLessLife,HasNoMaximumHandSize])`,
		`EachPlayerEffect(AnyPlayer,[DoesntLoseTheGameForHaving0OrLessLife])`,
		`SpellActions(ActionList([DoesntLoseTheGameForHaving0OrLessLife]))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[` + rule + `])`
		nodes, err := Parse(strings.NewReader(src), "invalid-loss.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
	for _, kind := range []string{"Instant", "Sorcery"} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[` + kind + `],Subtypes:[]),Rules:[` + ronLifeTotalLossRule + `])`
		nodes, err := Parse(strings.NewReader(src), "invalid-loss.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("nonpermanent accepted")
		}
	}
}

func TestRONLifeTotalLossInventory(t *testing.T) {
	src := `Card(Name:"Exemption",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + ronLifeTotalLossRule + `])`
	r, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if r.Compiled != 1 {
		t.Fatalf("rule rejected: %+v", r.Issues)
	}
	found := false
	for _, s := range r.Signatures {
		if s.Name == "DoesntLoseTheGameForHaving0OrLessLife" {
			found = true
			if s.Role != "player_rule_operation" {
				t.Fatal("wrong inventory role", s.Role)
			}
		}
	}
	if !found {
		t.Fatal("missing inventory operation")
	}
}

func TestRONLifeTotalLossCastingPaymentAndRestoration(t *testing.T) {
	record := staticBaseRecord(t, "LEA", "Lich")
	rules := record.Fields["Rules"].Args
	factory := staticBaseSubrule(t, record, rules[0], rules[1])
	g, a, _ := ronGame()
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
	if a.Life() != 0 || !a.IsAlive() || pending.IsGameOver() {
		t.Fatal("paid casting did not install exemption")
	}
	other := g.PutOnBattlefield(factory(), a.PlayerID())
	g.PhaseOut(card.ID())
	g.CheckStateBasedActions()
	if !a.IsAlive() {
		t.Fatal("independent factory rule lost")
	}
	g.PhaseIn([]uuid.UUID{card.ID()})
	g.BouncePermanentToHand(other)
	g.CheckStateBasedActions()
	if !a.IsAlive() {
		t.Fatal("phase cycle failed to restore rule")
	}

	restore := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([LoseLife(Integer(21)),GainLife(Integer(22))]))`)
	h, x, _ := ronGame()
	castRONSpell(t, h, x, restore())
	h.ResolveStack()
	if x.Life() != 21 || !x.IsAlive() || h.IsGameOver() {
		t.Fatal("state-based loss interrupted resolution")
	}
}
