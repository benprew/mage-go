package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronSearchPlayer struct {
	*mage.BasePlayer
	decline    bool
	choices    int
	shuffles   int
	candidates []mage.Card
	onShuffle  func()
}

func (p *ronSearchPlayer) ChooseCardFromLibrary(cards []mage.Card, _ string, _ mage.GameReader) mage.Card {
	p.choices++
	p.candidates = append([]mage.Card(nil), cards...)
	if p.decline || len(cards) == 0 {
		return nil
	}
	return cards[len(cards)-1]
}

func (p *ronSearchPlayer) ShuffleLibrary() {
	p.shuffles++
	if p.onShuffle != nil {
		p.onShuffle()
	}
	p.BasePlayer.ShuffleLibrary()
}

func TestRONLibrarySearchCards(t *testing.T) {
	for _, name := range []string{"Demonic Tutor", "Untamed Wilds"} {
		sets := []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"}
		if name == "Untamed Wilds" {
			sets = []string{"LEG"}
		}
		for _, set := range sets {
			for _, result := range []string{"find", "decline", "no-match", "empty"} {
				t.Run(name+"/"+set+"/"+result, func(t *testing.T) {
					a := &ronSearchPlayer{BasePlayer: mage.NewBasePlayer("A"), decline: result == "decline"}
					b := mage.NewBasePlayer("B")
					g := mage.NewGame(a, b)
					g.SetActivePlayerIndex(0)
					g.SetTurn(1)
					g.SetStep(core.PrecombatMain)
					card := repositoryCard(t, set, name)
					if err := mage.ValidateCardDefinition(name, card); err != nil {
						t.Fatal(err)
					}
					addRONHandCard(a.BasePlayer, card)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || g.StackSize() != 0 || a.choices != 0 || a.shuffles != 0 {
						t.Fatal("unpaid search did not roll back")
					}
					a.ManaPool().Add(core.Black, 1)
					a.ManaPool().Add(core.Green, 1)
					a.ManaPool().Add(core.Colorless, 2)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
						t.Fatal(err)
					}
					remainingMana := 2
					if name == "Untamed Wilds" {
						remainingMana = 1
					}
					if a.ManaPool().TotalMana() != remainingMana {
						t.Fatal("wrong casting cost")
					}
					if a.choices != 0 || a.shuffles != 0 {
						t.Fatal("search ran during announcement")
					}
					found := repositoryCard(t, "LEB", "Forest")
					found.SetOwner(a.PlayerID())
					other := mage.NewLand("Nonbasic", mage.WithSubTypes("Forest"))
					other.SetOwner(a.PlayerID())
					if result != "empty" {
						a.AddToLibrary(other)
					}
					if result == "find" || result == "decline" {
						a.AddToLibrary(found)
					}
					b.AddToLibrary(mage.NewLand("Opponent library"))
					before := len(a.Library())
					wantFound := result != "empty" && (name == "Demonic Tutor" || result == "find")
					a.onShuffle = func() {
						if name == "Demonic Tutor" && len(a.Hand()) != boolInt(wantFound) {
							t.Error("hand movement must precede shuffle")
						}
						if name == "Untamed Wilds" && (g.FindPermanent(found.ID()) != nil) != wantFound {
							t.Error("entry must precede shuffle")
						}
					}
					pending := g.Clone()
					pending.ResolveStack()
					if g.StackSize() != 1 || len(a.Library()) != before {
						t.Fatal("clone changed pending search")
					}
					g.ResolveStack()
					if a.shuffles != 1 || len(a.Library()) != before-boolInt(wantFound) || len(b.Library()) != 1 {
						t.Fatal("search count or controller")
					}
					if len(g.PublicReveals()) != 0 {
						t.Fatal("search revealed hidden cards")
					}
					if name == "Untamed Wilds" {
						if g.FindPermanent(other.ID()) != nil {
							t.Fatal("found a nonbasic land")
						}
						if wantFound && (g.FindPermanent(found.ID()).Tapped || g.FindPermanent(found.ID()).ControllerID() != a.PlayerID()) {
							t.Fatal("normal entry or controller")
						}
						for _, candidate := range a.candidates {
							if candidate.ID() != found.ID() {
								t.Fatal("nonbasic candidate")
							}
						}
					} else if wantFound && len(a.Hand()) != 1 {
						t.Fatal("required search found no card")
					}
					if len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != card.ID() {
						t.Fatal("spell disposal")
					}
				})
			}
		}
	}
}

func TestRONLibrarySearchCopiesAndEntry(t *testing.T) {
	for _, name := range []string{"Demonic Tutor", "Untamed Wilds"} {
		g, a, b := ronGame()
		g.SetActivePlayerIndex(0)
		set := "LEB"
		if name == "Untamed Wilds" {
			set = "LEG"
		}
		card := repositoryCard(t, set, name)
		addRONHandCard(a, card)
		a.ManaPool().Add(core.Black, 1)
		a.ManaPool().Add(core.Green, 1)
		a.ManaPool().Add(core.Colorless, 2)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		first := repositoryCard(t, "LEB", "Forest")
		second := repositoryCard(t, "LEB", "Island")
		first.SetOwner(a.PlayerID())
		second.SetOwner(b.PlayerID())
		a.AddToLibrary(first)
		b.AddToLibrary(second)
		if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
			t.Fatal("copy failed")
		}
		pending := g.Clone()
		g.ResolveTopOfStack()
		if len(a.Library()) != 1 || len(b.Library()) != 0 || len(pending.GetPlayer(b.PlayerID()).Library()) != 1 {
			t.Fatal("copy controller or clone isolation")
		}
		g.ResolveStack()
		pending.ResolveStack()
		for _, game := range []*mage.Game{g, pending} {
			for _, c := range []mage.Card{first, second} {
				if name == "Demonic Tutor" {
					if len(game.GetPlayer(c.Owner()).Hand()) != 1 || game.GetPlayer(c.Owner()).Hand()[0].ID() != c.ID() {
						t.Fatal("copy hand destination")
					}
				} else if p := game.FindPermanent(c.ID()); p == nil || p.ControllerID() != c.Owner() || p.Tapped {
					t.Fatal("copy entry destination")
				}
			}
		}
	}
}

func TestRONLibrarySearchInvalid(t *testing.T) {
	for _, action := range []string{
		`SearchLibrary`, `SearchLibrary()`, `SearchLibrary([],[])`, `SearchLibrary([])`,
		`SearchLibrary([FindAGenericCard(),PutFoundCardsIntoHand,Shuffle])`,
		`SearchLibrary([FindAGenericCard,PutFoundCardsIntoHand])`,
		`SearchLibrary([FindAGenericCard,Shuffle,PutFoundCardsIntoHand])`,
		`SearchLibrary([FindAGenericCard,PutFoundCardsIntoHand,Shuffle()])`,
		`SearchLibrary([FindAGenericCard,PutFoundCardsIntoHand,Shuffle,RevealFoundCards])`,
		`SearchLibrary([FindAGenericCard,PutFoundCardsOntoBattlefield([EntersNormally]),Shuffle])`,
		`SearchLibrary([FindACardOfType(And([IsSupertype(Basic),IsCardtype(Land)])),PutFoundCardsIntoHand,Shuffle])`,
		`SearchLibrary([FindACardOfType(And([IsSupertype(Basic),IsCardtype(Land)])),PutFoundCardsOntoBattlefield([EntersTapped]),Shuffle])`,
		`SearchLibrary([FindACardOfType(And([IsSupertype(Basic),IsCardtype(Land)])),PutFoundCardsOntoBattlefield([EntersNormally(),EntersNormally]),Shuffle])`,
		`SearchLibrary([FindACardOfType(And([IsSupertype(Basic),IsCardtype(Land),ControlledByAPlayer(You)])),PutFoundCardsOntoBattlefield([EntersNormally]),Shuffle])`,
		`SearchLibrary([FindACardOfType(And([IsSupertype(Basic()),IsCardtype(Land)])),PutFoundCardsOntoBattlefield([EntersNormally]),Shuffle])`,
		`SearchLibrary([FindACardOfType(And([IsSupertype(Basic),IsCardtype(Creature)])),PutFoundCardsOntoBattlefield([EntersNormally]),Shuffle])`,
	} {
		src := fmt.Sprintf(`Card(Name:"Invalid search",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([%s]))])`, action)
		nodes, err := Parse(strings.NewReader(src), "invalid-search.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
	search := `SearchLibrary([FindAGenericCard,PutFoundCardsIntoHand,Shuffle])`
	for _, rule := range []string{search, `Activated(` + search + `,ActionList([]))`, `PlayerAction(You,` + search + `)`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Wrong context",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "search-context.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted context %s", rule)
		}
	}
}

func TestRONLibrarySearchNormalEntryRules(t *testing.T) {
	a := &ronSearchPlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetActivePlayerIndex(0)
	g.SetStep(core.PrecombatMain)
	card := repositoryCard(t, "LEG", "Untamed Wilds")
	addRONHandCard(a.BasePlayer, card)
	land := mage.NewCreature("Entry land", "", 2, 3, mage.WithSuperTypes(core.SuperBasic),
		mage.WithCardType(core.TypeLand), mage.WithKeyword(core.EntersTapped),
		mage.WithAbility(mage.EntersWithNCounters(core.P1P1, 1)))
	land.SetOwner(a.PlayerID())
	a.AddToLibrary(land)
	doubler := g.PutOnBattlefield(mage.NewArtifact("Doubler", ""), a.PlayerID())
	g.AddCounterDoubler(doubler.ID(), core.P1P1, mage.IsLand)
	a.onShuffle = func() {
		p := g.FindPermanent(land.ID())
		if p == nil || !p.Tapped || p.Counters[core.P1P1] != 2 {
			t.Errorf("normal entry bypassed replacement rules: permanent=%+v", p)
		}
	}
	a.ManaPool().Add(core.Green, 1)
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.shuffles != 1 {
		t.Fatal("search did not shuffle after entry")
	}
}

func TestRONLibrarySearchActivationKeepsController(t *testing.T) {
	for _, change := range []string{"leave", "return", "phase", "control"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([SearchLibrary([FindAGenericCard,PutFoundCardsIntoHand,Shuffle])]))`)
			card := factory()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			found := mage.NewArtifact("Found", "")
			found.SetOwner(a.PlayerID())
			a.AddToLibrary(found)
			b.AddToLibrary(mage.NewArtifact("Other", ""))
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "leave", "return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if change == "return" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phase":
				g.PhaseOut(source.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.ResolveStack()
			if len(a.Hand()) != 1 || a.Hand()[0].ID() != found.ID() || len(b.Hand()) != 0 || len(b.Library()) != 1 {
				t.Fatal("search followed source instead of ability controller")
			}
		})
	}
}

func TestRONLibrarySearchInventory(t *testing.T) {
	src := `Card(Name:"Nested search",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([UnknownAction,SearchLibrary([FindACardOfType(And([IsSupertype(Basic),IsCardtype(Land)])),PutFoundCardsOntoBattlefield([EntersNormally]),Shuffle])]))])`
	report, err := InspectFS(fstest.MapFS{"search.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"FindACardOfType": "search_instruction", "And": "library_card_filter", "IsSupertype": "library_card_filter", "Basic": "supertype", "EntersNormally": "search_entry_operation", "Shuffle": "search_instruction"}
	for _, signature := range report.Signatures {
		if role, ok := want[signature.Name]; ok {
			if signature.Role != role {
				t.Errorf("%s role=%s, want %s", signature.Name, signature.Role, role)
			}
			delete(want, signature.Name)
		}
	}
	if len(want) != 0 || report.Compiled != 0 {
		t.Fatal("missing nested roles or weakened loading")
	}
}
