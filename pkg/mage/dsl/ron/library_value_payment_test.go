package ron

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONLibraryValuePaymentTransmute(t *testing.T) {
	for _, scenario := range []string{"less", "equal", "pay", "decline", "poor", "no-artifact", "no-find", "countered", "lost-offering"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronValueSearchPlayer{ronSearchPlayer: ronSearchPlayer{BasePlayer: mage.NewBasePlayer("A"), decline: scenario == "no-find"}, accept: scenario != "decline"}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetActivePlayerIndex(0)
			g.SetTurn(1)
			g.SetStep(core.PrecombatMain)
			card := repositoryCard(t, "ATQ", "Transmute Artifact")
			addRONHandCard(a.BasePlayer, card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid spell cast")
			}
			old := mage.NewArtifact("Old", "{2}")
			old.SetOwner(a.PlayerID())
			if scenario != "no-artifact" {
				g.PutOnBattlefield(old, a.PlayerID())
			}
			cost := "{5}"
			if scenario == "less" {
				cost = "{1}"
			}
			if scenario == "equal" {
				cost = "{2}"
			}
			found := mage.NewArtifact("Found", cost)
			found.SetOwner(a.PlayerID())
			a.AddToLibrary(found)
			a.AddToLibrary(mage.NewLand("Other"))
			a.ManaPool().Add(core.Blue, 2)
			mana := 3
			if scenario == "poor" {
				mana = 2
			}
			a.ManaPool().Add(core.Colorless, mana)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if a.choices != 0 || a.shuffles != 0 || scenario != "no-artifact" && g.FindPermanent(old.ID()) == nil {
				t.Fatal("resolution action ran at announcement")
			}

			if scenario == "countered" {
				g.CounterStackObject(g.StackPeek().ID)
				if g.FindPermanent(old.ID()) == nil || a.choices != 0 || a.shuffles != 0 {
					t.Fatal("countered spell ran its actions")
				}
				return
			}
			if scenario == "lost-offering" {
				g.ExilePermanent(g.FindPermanent(old.ID()))
			}
			g.ResolveStack()
			entered := scenario == "less" || scenario == "equal" || scenario == "pay"
			if (g.FindPermanent(found.ID()) != nil) != entered {
				t.Fatal("wrong found-card destination")
			}
			if scenario == "decline" || scenario == "poor" {
				if len(a.Graveyard()) != 3 {
					t.Fatal("unpaid find did not go to graveyard")
				}
			}
			if scenario == "no-artifact" || scenario == "lost-offering" {
				if a.shuffles != 0 || a.choices != 0 {
					t.Fatal("searched without sacrifice")
				}
			} else if a.shuffles != 1 {
				t.Fatal("missing shuffle")
			}
			expected := mana
			if scenario == "pay" {
				expected -= 3
			}
			if a.ManaPool().TotalMana() != expected {
				t.Fatal("wrong payment")
			}
		})
	}
}

type ronValueSearchPlayer struct {
	ronSearchPlayer
	accept bool
}

func (p *ronValueSearchPlayer) ChooseMayAbility(string) bool { return p.accept }

func TestRONLibraryValuePaymentDeclarations(t *testing.T) {
	for _, body := range []string{
		`MustCost(SacrificeAPermanent(IsCardtype(Artifact))),If(CostWasPaid,[GainLife(ManaValueOfTheSacrificedPermanent)])`,
	} {
		permanentScopeFactory(t, "Sorcery", `SpellActions(ActionList([`+body+`]))`)
	}
	for _, body := range []string{
		`GainLife(ManaValueOfTheSacrificedPermanent)`,
		`MustCost(SacrificeAPermanent(IsCardtype(Artifact))),If(CostWasntPaid,[GainLife(ManaValueOfTheSacrificedPermanent)])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([`+body+`]))])`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("unbound value accepted")
		}
	}
}

func TestRONLibraryValuePaymentCopiesAndClone(t *testing.T) {
	g, a, b := ronGame()
	g.SetActivePlayerIndex(0)
	card := repositoryCard(t, "ATQ", "Transmute Artifact")
	addRONHandCard(a, card)
	a.ManaPool().Add(core.Blue, 2)
	oldA := mage.NewArtifact("A offering", "{1}")
	oldB := mage.NewArtifact("B offering", "{4}")
	g.PutOnBattlefield(oldA, a.PlayerID())
	g.PutOnBattlefield(oldB, b.PlayerID())
	first := mage.NewArtifact("A find", "{1}")
	second := mage.NewArtifact("B find", "{4}")
	a.AddToLibrary(first)
	b.AddToLibrary(second)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	g.ResolveTopOfStack()
	if g.FindPermanent(second.ID()) == nil || g.FindPermanent(first.ID()) != nil || clone.FindPermanent(oldB.ID()) == nil {
		t.Fatal("copy controller or clone isolation")
	}
	g.ResolveStack()
	clone.ResolveStack()
	for _, game := range []*mage.Game{g, clone} {
		for _, c := range []mage.Card{first, second} {
			if p := game.FindPermanent(c.ID()); p == nil || p.ControllerID() != c.Owner() {
				t.Fatal("independent search result missing")
			}
		}
	}
	factory, err := Compile(repositoryRecord(t, "ATQ", "Transmute Artifact", "factory.ron"))
	if err != nil {
		t.Fatal(err)
	}
	instanceA, instanceB := factory(), factory()
	if instanceA.ID() == instanceB.ID() {
		t.Fatal("factory shares card identity")
	}
}

func TestRONLibraryValuePaymentSearchDeclarations(t *testing.T) {
	base := repositoryRecord(t, "ATQ", "Transmute Artifact", "invalid.ron")
	changes := map[string]string{
		"PutFoundCardsIntoGraveyard": "PutFoundCardsIntoHand",
		"ManaValueOfTheFoundCard":    "ValueX",
		"LessThanOrEqualTo":          "GreaterThanOrEqualTo",
		"ManaCostX":                  "ManaCostU",
		"EntersNormally":             "EntersTapped",
		"FindACardOfType":            "FindAGenericCard",
		"CostWasPaid":                "CostWasntPaid",
		"Shuffle":                    "Unknown",
	}
	for before, after := range changes {
		t.Run(before, func(t *testing.T) {
			node := cloneRepositoryNode(base)
			var change func(*Node)
			change = func(n *Node) {
				if n.Name == before {
					n.Name = after
				}
				for i := range n.Args {
					change(&n.Args[i])
				}
				for key, value := range n.Fields {
					change(&value)
					n.Fields[key] = value
				}
			}
			change(&node)
			if _, err := Compile(node); err == nil {
				t.Fatal("invalid search shape accepted")
			}
		})
	}
}

func TestRONLibraryValuePaymentLoadAndInventory(t *testing.T) {
	node := repositoryRecord(t, "ATQ", "Transmute Artifact", "inventory.ron")
	entries := map[string]*InventorySignature{}
	walkInventory(node, "Transmute Artifact", "Card", "card", "collection", true, entries)
	want := map[string]string{"ACardOfTypeWasFoundThisWay": "search_condition", "ManaValueIs": "library_card_filter", "LessThanOrEqualTo": "mana_value_comparison", "Minus": "value", "ManaValueOfTheFoundCard": "value", "PutFoundCardsIntoGraveyard": "search_instruction"}
	for name, role := range want {
		found := false
		for _, entry := range entries {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Errorf("missing inventory role %s / %s", name, role)
		}
	}
}
