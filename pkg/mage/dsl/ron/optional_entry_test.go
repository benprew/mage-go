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

type ronOptionalEntryPlayer struct {
	mage.Player
	answers   []bool
	offers    int
	landCalls int
}

func (p *ronOptionalEntryPlayer) ChooseMayAbility(string) bool {
	answer := p.answers[p.offers%len(p.answers)]
	p.offers++
	return answer
}

func (p *ronOptionalEntryPlayer) ChooseString([]string, string) string {
	p.landCalls++
	return "Forest"
}

func TestRONOptionalEntryGameplay(t *testing.T) {
	for _, accept := range []bool{false, true} {
		t.Run(fmt.Sprint(accept), func(t *testing.T) {
			g, base, owner := ronGame()
			a := &ronOptionalEntryPlayer{Player: base, answers: []bool{accept}}
			b := &ronOptionalEntryPlayer{Player: owner, answers: []bool{!accept}}
			g.SetPlayerAt(0, a)
			g.SetPlayerAt(1, b)
			factory := permanentScopeFactory(t, "Artifact", `AsPermanentEnters(ThisPermanent,[MayActions([EntersTapped,ChooseAPlayer(Opponent),ChooseABasicLandType])]),TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([GainLife(Integer(2))]))`)
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if a.offers != 0 || a.landCalls != 0 || g.FindPermanent(card.ID()) != nil {
				t.Fatal("entry choice at announcement or payment")
			}
			pending := g.Clone()
			cloneA := &ronOptionalEntryPlayer{Player: pending.GetPlayer(a.PlayerID()), answers: []bool{!accept}}
			pending.SetPlayerAt(0, cloneA)
			for _, game := range []*mage.Game{g, pending} {
				game.ResolveStack()
				p := game.FindPermanent(card.ID())
				want := accept
				if game == pending {
					want = !want
				}
				if p == nil || p.Tapped != want || (p.ChosenPlayer == b.PlayerID()) != want || (p.ChosenBasicLandType == core.ForestLandType) != want {
					t.Fatal("entry body, decline or clone isolation")
				}
				game.PutTriggersOnStack()
				game.ResolveStack()
				if game.GetPlayer(a.PlayerID()).Life() != 22 {
					t.Fatal("entry trigger did not use the entered permanent")
				}
			}
			if a.offers != 1 || a.landCalls != boolInt(accept) || b.offers != 0 || cloneA.offers != 1 || cloneA.landCalls != boolInt(!accept) {
				t.Fatal("wrong chooser or repeated optional body")
			}
			p := g.FindPermanent(card.ID())
			g.PhaseIn(g.PhaseOut(p.ID()))
			if a.offers != 1 {
				t.Fatal("phase cycle repeated choice")
			}
			g.BouncePermanentToHand(g.FindPermanent(p.ID()))
			a.RemoveFromHand(card.ID())
			returned := g.PutOnBattlefield(card, b.PlayerID())
			if returned.Tapped != !accept || b.offers != 1 || returned == p {
				t.Fatal("new entry reused old decision")
			}
			second := factory()
			if second.ID() == card.ID() {
				t.Fatal("factory reused identity")
			}
			g.PutOnBattlefield(second, a.PlayerID())
			if a.offers != 2 {
				t.Fatal("factory instances shared optional state")
			}
		})
	}
}

func TestRONOptionalEntryNested(t *testing.T) {
	for _, answers := range [][]bool{{false}, {true, false}, {true, true}} {
		g, base, _ := ronGame()
		a := &ronOptionalEntryPlayer{Player: base, answers: answers}
		g.SetPlayerAt(0, a)
		card := permanentScopeFactory(t, "Artifact", `AsPermanentEnters(ThisPermanent,[MayActions([MayActions([EntersTapped]),ChooseABasicLandType])])`)()
		p := g.PutOnBattlefield(card, a.PlayerID())
		if p.Tapped != (len(answers) == 2 && answers[1]) || a.offers != len(answers) || a.landCalls != boolInt(answers[0]) {
			t.Fatal("nested choice did not keep its local decision")
		}
	}
}

func TestRONOptionalEntryInvalid(t *testing.T) {
	for _, body := range []string{`MayActions`, `MayActions()`, `MayActions(EntersTapped)`, `MayActions([])`, `MayActions([EntersTapped()])`, `MayActions([ChooseAPlayer(You)])`, `MayActions([DrawNumberCards(Integer(1))])`, `MayActions([MayActions([])])`, `MayActions([EntersWithNumberCounters(Integer(2),ChargeCounter)])`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[AsPermanentEnters(ThisPermanent,[` + body + `])])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONOptionalEntryScopeAndInventory(t *testing.T) {
	for _, rules := range []string{
		`AsPermanentEnters(ThisPermanent,[MayActions([ChooseAPlayer(Opponent)])]),TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(TheChosenPlayer)),ActionList([GainLife(Integer(1))]))`,
		`AsPermanentEnters(ThisPermanent,[MayActions([ChooseABasicLandType])]),PermanentLayerEffect(HostPermanent,[SetLandTypeVariable(TheChosenLandType)])`,
		`AsPermanentEnters(HostPermanent,[MayActions([EntersTapped])])`,
		`SpellActions(ActionList([MayActions([EntersTapped])]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[`+rules+`])`), "scope.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted invalid scope %s", rules)
		}
	}
	nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[AsPermanentEnters(ThisPermanent,[MayActions([EntersTapped])])])`), "scope.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(nodes[0]); err == nil {
		t.Fatal("entry on an instant accepted")
	}
	src := `Card(Name:"Entry",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[AsPermanentEnters(ThisPermanent,[MayActions([ChooseAPlayer(Opponent),MayActions([EntersTapped])])])])`
	report, err := InspectFS(fstest.MapFS{"entry.ron": &fstest.MapFile{Data: []byte(src)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("accepted optional entry missing from inventory", err)
	}
	for name, role := range map[string]string{"MayActions": "entry_operation", "ChooseAPlayer": "entry_operation", "EntersTapped": "entry_operation", "Opponent": "player_filter"} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s role %s", name, role)
		}
	}
}

func TestRONOptionalEntryFailedCast(t *testing.T) {
	g, base, _ := ronGame()
	a := &ronOptionalEntryPlayer{Player: base, answers: []bool{true}}
	g.SetPlayerAt(0, a)
	nodes, err := Parse(strings.NewReader(`Card(Name:"Entry",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),ManaCost:[ManaCostGeneric(1)],Rules:[AsPermanentEnters(ThisPermanent,[MayActions([EntersTapped])])])`), "cost.ron")
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
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || a.offers != 0 || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("failed payment offered entry or changed state")
	}
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if a.offers != 0 {
		t.Fatal("successful payment offered entry")
	}
	g.CounterSpellOnStack(card.ID())
	g.ResolveStack()
	if a.offers != 0 || g.FindPermanent(card.ID()) != nil {
		t.Fatal("countered spell offered entry")
	}
}

func TestRONOptionalEntryCopyDependencies(t *testing.T) {
	count := 0
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), set)
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range nodes {
			name := n.Fields["Name"].String
			if name != "Clone" && name != "Copy Artifact" && name != "Vesuvan Doppelganger" {
				continue
			}
			count++
			_, err := Compile(n)
			if err != nil {
				t.Fatalf("%s/%s: %v", set, name, err)
			}
		}
	}
	if count != 18 {
		t.Fatalf("copy records=%d", count)
	}
}
