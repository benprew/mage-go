package ron

import (
	"fmt"
	"maps"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const spellExileAction = `Exile([Spell(ThisSpell)],[ExilesWithNumberCounters(Integer(2),ScreamCounter)])`

func TestRONSpellExileGameplay(t *testing.T) {
	for _, scenario := range []string{"normal", "zero", "wide", "copy", "clone", "other-controller", "two-instances"} {
		t.Run(scenario, func(t *testing.T) {
			count := 2
			if scenario == "zero" {
				count = 0
			}
			if scenario == "wide" {
				count = 300
			}
			action := fmt.Sprintf(`Exile([Spell(ThisSpell)],[ExilesWithNumberCounters(Integer(%d),ScreamCounter)])`, count)
			src := `Card(Name:"Exile spell",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + action + `,GainLife(Integer(1)),` + spellExileAction + `]))])`
			registry := mage.NewRegistry()
			if err := LoadInto(registry, fstest.MapFS{"probe.ron": {Data: []byte(src)}}, "*.ron"); err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			card, err := registry.CreateCard("Exile spell")
			if err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindExiledCard(card.ID()) != nil {
				t.Fatal("exile ran at announcement")
			}
			if scenario == "copy" {
				cp := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
				g.ResolveTopOfStack()
				if b.Life() != 21 || g.FindExiledCard(cp.SourceID) != nil || g.FindExiledCard(card.ID()) != nil {
					t.Fatal("copy reused original identity or did not cease")
				}
			}
			if scenario == "other-controller" {
				g.StackPeek().Controller = b.PlayerID()
			}
			if scenario == "two-instances" {
				second, err := registry.CreateCard("Exile spell")
				if err != nil {
					t.Fatal(err)
				}
				second.SetOwner(a.PlayerID())
				a.AddToHand(second)
				if err := g.CastSpellByID(a.PlayerID(), second.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveTopOfStack()
				if g.FindExiledCard(second.ID()) == nil || g.FindExiledCard(card.ID()) != nil {
					t.Fatal("factory instances shared source")
				}
			}
			if scenario == "clone" {
				clone := g.Clone()
				clone.ResolveStack()
				if clone.FindExiledCard(card.ID()).Counters[core.Scream] != count || g.FindExiledCard(card.ID()) != nil {
					t.Fatal("clone shared execution")
				}
				clone.FindExiledCard(card.ID()).Counters[core.Scream] = 7
			}
			g.ResolveStack()
			ec := g.FindExiledCard(card.ID())
			if ec == nil || ec.FaceDown || ec.Card.Owner() != a.PlayerID() || ec.Counters[core.Scream] != count || len(a.Graveyard()) != 0 || len(b.Graveyard()) != 0 {
				t.Fatal("spell departure or counters failed")
			}
		})
	}
}

func TestRONSpellExileInvalid(t *testing.T) {
	for _, action := range []string{
		`Exile([Spell(ThisSpell)], [ExilesWithNumberCounters(Integer(-1),ScreamCounter)])`,
		`Exile([Spell(ThisSpell)], [ExilesWithNumberCounters(ValueX,ScreamCounter)])`,
		`Exile([Spell(ThisSpell)], [ExilesWithNumberCounters(Integer(2),UnknownCounter)])`,
		`Exile([Spell(ThisSpell)], [ExilesWithNumberCounters(Integer(2),ScreamCounter())])`,
		`Exile([Spell(ThisSpell)], [ExilesWithNumberCounters(Integer(2))])`,
		`Exile([Spell(ThisSpell)], [ExilesWithNumberCounters(Integer(2),ScreamCounter),ExilesWithNumberCounters(Integer(2),ChargeCounter)])`,
		`Exile([Spell(ThisSpell)], [ExilesFaceDown])`, `Exile([Spell(ThisSpell())],[])`,
		`Exile([Spell(ThisPermanent)],[])`, `Exile([Spell(Trigger_ThatSpell)],[])`,
		`Exile([Spell(Ref_TargetSpell)],[])`, `Exile([Spell(ThisSpell),Spell(ThisSpell)],[])`,
		`Exile([Spell(ThisSpell,ThisSpell)],[])`, `Exile([Spell],[])`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + action + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
	for _, rules := range []string{
		`Activated(PayMana([]),ActionList([` + spellExileAction + `]))`,
		`FromExile(TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([` + spellExileAction + `])))`,
	} {
		types := "Artifact"
		if strings.HasPrefix(rules, "FromExile") {
			types = "Sorcery"
			rules = `SpellActions(ActionList([GainLife(Integer(0))])),` + rules
		}
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[`+types+`],Subtypes:[]),Rules:[`+rules+`])`), "context.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted wrong context %s", rules)
		}
	}
}

func TestRONSpellExileAllHallowsEve(t *testing.T) {
	node := repositoryRecord(t, "LEG", "All Hallow's Eve", "LEG.ron")
	factory, err := Compile(node)
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Black, 4)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if ec := g.FindExiledCard(card.ID()); ec == nil || ec.Counters[core.Scream] != 2 || len(a.Graveyard()) != 0 {
		t.Fatal("original spell instruction did not initialize exile")
	}
	queueRONUpkeep(g, b)
	g.ResolveStack()
	if g.FindExiledCard(card.ID()).Counters[core.Scream] != 2 {
		t.Fatal("opponent upkeep removed a counter")
	}
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if g.FindExiledCard(card.ID()).Counters[core.Scream] != 1 {
		t.Fatal("first upkeep did not remove one counter")
	}
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if g.FindExiledCard(card.ID()) != nil || len(a.Graveyard()) != 1 || a.Life() != 20 {
		t.Fatal("second upkeep did not resolve zero branch")
	}
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("departed source armed")
	}
}

func TestRONSpellExileRecall(t *testing.T) {
	node := repositoryRecord(t, "LEG", "Recall", "LEG.ron")
	if _, err := Compile(node); err != nil {
		t.Fatalf("complete Recall record rejected: %v", err)
	}
	body := node.Fields["Rules"].Args[0].Args[0]
	body.Args = []Node{{Kind: List, Args: []Node{body.Args[0].Args[2]}}}
	rule := node.Fields["Rules"].Args[0]
	rule.Args = []Node{body}
	node.Fields = maps.Clone(node.Fields)
	node.Fields["Rules"] = Node{Kind: List, Args: []Node{rule}}
	factory, err := Compile(node)
	if err != nil {
		t.Fatal(err)
	}
	g, a, _ := ronGame()
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Blue, 5)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 2); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if ec := g.FindExiledCard(card.ID()); ec == nil || ec.Counters[core.Scream] != 0 || len(a.Graveyard()) != 0 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("Recall's original exile instruction failed")
	}
}

func TestRONSpellExileFizzleAndAtomicRejection(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([`+spellExileAction+`,GainLife(Integer(1))])))`)
	g, a, _ := ronGame()
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	victim := g.PutOnBattlefield(mage.NewArtifact("Victim", ""), a.PlayerID())
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{victim.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ExilePermanent(victim)
	g.ResolveStack()
	if g.FindExiledCard(card.ID()) != nil || len(a.Graveyard()) != 1 || a.Life() != 20 {
		t.Fatal("illegal targets did not stop spell body")
	}
	registry := mage.NewRegistry()
	sentinel := mage.NewArtifact("Sentinel", "")
	registry.Register("Sentinel", func() mage.Card { return sentinel.Copy() })
	src := `Card(Name:"Valid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + spellExileAction + `]))])
 Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + spellExileAction + `]))]))])`
	if err := LoadInto(registry, fstest.MapFS{"mixed.ron": {Data: []byte(src)}}, "*.ron"); err == nil {
		t.Fatal("delayed self-exile accepted")
	}
	if _, err := registry.CreateCard("Sentinel"); err != nil {
		t.Fatal("invalid collection changed registry")
	}
	if _, err := registry.CreateCard("Valid"); err == nil {
		t.Fatal("published partial collection")
	}
}

func TestRONSpellExileInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := InspectFS(fstest.MapFS{"LEG.ron": {Data: data}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"Spell": "exile_subject", "ThisSpell": "spell_reference", "ExilesWithNumberCounters": "exile_modifier", "ScreamCounter": "card_counter_type"} {
		found := false
		for _, use := range inventory.Signatures {
			if use.Name == name && use.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s role %s", name, role)
		}
	}
}
