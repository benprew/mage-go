package ron

import (
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronBoundedSearchPlayer struct {
	*ronSearchPlayer
	accept        bool
	limit, offers int
}

func (p *ronBoundedSearchPlayer) ChooseMayAbility(string) bool { p.offers++; return p.accept }
func (p *ronBoundedSearchPlayer) ChooseCardFromLibrary(cards []mage.Card, reason string, g mage.GameReader) mage.Card {
	if p.choices >= p.limit {
		p.choices++
		return nil
	}
	return p.ronSearchPlayer.ChooseCardFromLibrary(cards, reason, g)
}
func TestRONBoundedLibrarySearchLandTax(t *testing.T) {
	for _, accept := range []bool{false, true} {
		for _, count := range []int{0, 1, 2, 3, 4} {
			t.Run(fmt.Sprintf("%t/%d", accept, count), func(t *testing.T) {
				a := &ronBoundedSearchPlayer{ronSearchPlayer: &ronSearchPlayer{BasePlayer: mage.NewBasePlayer("A")}, accept: accept, limit: count}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				g.SetTurn(2)
				g.SetActivePlayerIndex(0)
				g.SetStep(core.PrecombatMain)
				card := repositoryCard(t, "LEG", "Land Tax")
				addRONHandCard(a.BasePlayer, card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 {
					t.Fatal("unpaid casting must roll back")
				}
				a.ManaPool().Add(core.White, 1)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				if a.ManaPool().TotalMana() != 0 || g.FindPermanent(card.ID()) == nil {
					t.Fatal("printed cost or entry")
				}
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
				for _, name := range []string{"Plains", "Island", "Swamp", "Mountain"} {
					c := repositoryCard(t, "LEB", name)
					c.SetOwner(a.PlayerID())
					a.AddToLibrary(c)
				}
				a.AddToLibrary(repositoryCard(t, "LEB", "Tropical Island"))
				queueRONUpkeep(g, b)
				if g.StackSize() != 0 {
					t.Fatal("opponent upkeep triggered")
				}
				queueRONUpkeep(g, a)
				if g.StackSize() != 1 || a.offers != 0 || a.choices != 0 {
					t.Fatal("upkeep detection or early choice")
				}
				want := 0
				if accept {
					want = min(count, 3)
				}
				a.onShuffle = func() {
					if len(a.Hand()) != want || len(g.PublicReveals()) != want {
						t.Error("search instruction order")
					}
				}
				g.ResolveStack()
				if len(a.Hand()) != want || len(a.Library()) != 5-want || a.shuffles != boolInt(accept) || a.offers != 1 {
					t.Fatal("Land Tax search count or decline")
				}
				if len(g.PublicReveals()) != want {
					t.Fatal("found cards must be revealed")
				}
				for _, c := range a.Hand() {
					if !c.HasSuperType(core.SuperBasic) || !c.HasType(core.TypeLand) {
						t.Fatal("found a nonbasic card")
					}
				}
			})
		}
	}
}

func TestRONBoundedLibrarySearchLifecycle(t *testing.T) {
	for _, change := range []string{"condition-false", "recheck", "phase-land", "phase-cycle", "source-left", "source-return", "source-phase", "source-control", "abilities", "copy", "copy-same", "two"} {
		t.Run(change, func(t *testing.T) {
			a := &ronBoundedSearchPlayer{ronSearchPlayer: &ronSearchPlayer{BasePlayer: mage.NewBasePlayer("A")}, accept: true, limit: 6}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetTurn(2)
			g.SetActivePlayerIndex(0)
			card := repositoryCard(t, "LEG", "Land Tax")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
			for range 6 {
				c := repositoryCard(t, "LEB", "Plains")
				c.SetOwner(a.PlayerID())
				a.AddToLibrary(c)
			}
			if change == "condition-false" {
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
			}
			if change == "two" {
				g.PutOnBattlefield(repositoryCard(t, "LEG", "Land Tax"), a.PlayerID())
			}
			queueRONUpkeep(g, a)
			if change == "condition-false" {
				if g.StackSize() != 0 {
					t.Fatal("false detection condition triggered")
				}
				return
			}
			switch change {
			case "recheck":
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
			case "phase-land":
				g.PhaseOut(land.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(land.ID()))
			case "source-left", "source-return":
				g.BouncePermanentToHand(source)
				if change == "source-return" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "source-phase":
				g.PhaseOut(source.ID())
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "abilities":
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
				g.ApplyContinuousEffects()
			case "copy-same":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "copy":
				b.AddToLibrary(repositoryCard(t, "LEB", "Island"))
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			before := len(a.Hand())
			clone := g.Clone()
			g.SetActivePlayerIndex(1)
			g.ResolveStack()
			clone.ResolveStack()
			want := 3
			if change == "recheck" || change == "phase-land" {
				want = 0
			}
			if change == "two" || change == "copy-same" {
				want = 6
			}
			for _, game := range []*mage.Game{g, clone} {
				if len(game.GetPlayer(a.PlayerID()).Hand()) != before+want || len(game.PublicReveals()) != want {
					t.Fatalf("condition recheck, saved controller or independent instance: hand=%d reveals=%d wantHand=%d wantReveals=%d", len(game.GetPlayer(a.PlayerID()).Hand()), len(game.PublicReveals()), before+want, want)
				}
				if change == "copy" && len(game.GetPlayer(b.PlayerID()).Library()) != 1 {
					t.Fatal("copied condition used the original controller")
				}
			}
		})
	}
}

func TestRONBoundedLibrarySearchInvalid(t *testing.T) {
	good := `SearchLibrary([FindNumberCardsOfType(Integer(3),And([IsSupertype(Basic),IsCardtype(Land)])),RevealFoundCards,PutFoundCardsIntoHand,Shuffle])`
	for _, action := range []string{
		strings.Replace(good, "Integer(3)", "Integer(-1)", 1), strings.Replace(good, "Integer(3)", "ValueX", 1),
		strings.Replace(good, "RevealFoundCards,", "", 1), strings.Replace(good, "RevealFoundCards", "RevealFoundCards()", 1),
		strings.Replace(good, "Shuffle]", "Shuffle()]", 1), strings.Replace(good, "IsCardtype(Land)", "IsCardtype(Creature)", 1),
		strings.Replace(good, "PutFoundCardsIntoHand", "PutFoundCardsOntoBattlefield([EntersNormally])", 1),
		strings.Replace(good, "FindNumberCardsOfType(Integer(3),", "FindNumberCardsOfType(", 1),
	} {
		n, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([MayAction(`+action+`)]))])`), "invalid-bounded-search.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(n[0]); err == nil {
			t.Fatal("accepted invalid search", action)
		}
	}
	for _, rule := range []string{`MayAction(` + good + `)`, `AsPermanentEnters(ThisPermanent,[MayAction(` + good + `)])`, `Activated(MayAction(` + good + `),ActionList([]))`} {
		n, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid-context.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(n[0]); err == nil {
			t.Fatal("accepted invalid context", rule)
		}
	}
}

func TestRONBoundedLibrarySearchInventory(t *testing.T) {
	record := repositoryRecord(t, "LEG", "Land Tax", "LEG.ron")
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		c := factory()
		if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
			t.Fatal(err)
		}
	}
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([UnknownAction,MayAction(SearchLibrary([FindNumberCardsOfType(Integer(3),And([IsSupertype(Basic),IsCardtype(Land)])),RevealFoundCards,PutFoundCardsIntoHand,Shuffle]))]))])`
	report, err := InspectFS(fstest.MapFS{"search.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"FindNumberCardsOfType": "search_instruction", "Integer": "value", "And": "library_card_filter", "Basic": "supertype", "RevealFoundCards": "search_instruction"}
	for _, sig := range report.Signatures {
		if role, ok := want[sig.Name]; ok {
			if role != sig.Role {
				t.Errorf("%s role=%s, want %s", sig.Name, sig.Role, role)
			}
			delete(want, sig.Name)
		}
	}
	if len(want) != 0 || report.Compiled != 0 {
		t.Fatal("missing nested roles or weakened load")
	}
}

func TestRONBoundedLibrarySearchLoad(t *testing.T) {
	data, err := fs.ReadFile(repositoryFixtures.files, "LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Land Tax"`) {
			record = line
			break
		}
	}
	if record == "" {
		t.Fatal("missing Land Tax record")
	}
	files := fstest.MapFS{"LEG.ron": {Data: []byte(record)}, "reprint.ron": {Data: []byte(record)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || registry == nil {
		t.Fatal("valid search records did not load", err)
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(strings.Replace(record, `"Land Tax"`, `"Invalid"`, 1), "Integer(3)", "ValueX", 1))}
	registry, err = LoadFS(files, "*.ron")
	if err == nil || registry != nil {
		t.Fatal("published a partial collection")
	}
}

func TestRONBoundedLibrarySearchResolvingContexts(t *testing.T) {
	search := `MayAction(SearchLibrary([FindNumberCardsOfType(Integer(0),And([IsCardtype(Land),IsSupertype(Basic)])),RevealFoundCards,PutFoundCardsIntoHand,Shuffle]))`
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerActions(Ref_TargetPlayer,[`+search+`])])))`)
	a := mage.NewBasePlayer("A")
	b := &ronBoundedSearchPlayer{ronSearchPlayer: &ronSearchPlayer{BasePlayer: mage.NewBasePlayer("B")}, accept: true, limit: 3}
	g := mage.NewGame(a, b)
	g.SetTurn(2)
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	b.AddToLibrary(repositoryCard(t, "LEB", "Forest"))
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if b.offers != 1 || b.choices != 0 || b.shuffles != 1 || len(b.Library()) != 1 || len(g.PublicReveals()) != 0 {
		t.Fatal("zero bound or player action scope")
	}
}
