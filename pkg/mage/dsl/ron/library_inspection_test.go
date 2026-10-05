package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronLibraryViewer struct {
	*mage.BasePlayer
	seenOrder      []uuid.UUID
	offers, orders int
	shuffles       int
	retarget       uuid.UUID
	accept         bool
}

func (p *ronLibraryViewer) ChooseScryPlacement(top []mage.Card, _ string, _ mage.GameReader) ([]uuid.UUID, []uuid.UUID) {
	p.orders++
	order := make([]uuid.UUID, len(top))
	for i, c := range top {
		order[len(top)-1-i] = c.ID()
	}
	p.seenOrder = order
	return nil, order
}
func (p *ronLibraryViewer) ShuffleLibrary() { p.shuffles++; p.BasePlayer.ShuffleLibrary() }

func (p *ronLibraryViewer) ChooseTargets(possible []uuid.UUID, minimum, maximum int, g *mage.Game) []uuid.UUID {
	if p.retarget != uuid.Nil {
		return []uuid.UUID{p.retarget}
	}
	return p.BasePlayer.ChooseTargets(possible, minimum, maximum, g)
}

func (p *ronLibraryViewer) ChooseMayAbility(string) bool { p.offers++; return p.accept }

func TestRONLibraryInspectionCards(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		count     int
		spell     bool
		color     core.Color
	}{{"LEA", "Natural Selection", 3, true, core.Green}, {"FEM", "Orcish Spy", 3, false, core.Red}, {"LEG", "Visions", 5, true, core.White}} {
		for _, scenario := range []string{"normal", "self", "short", "empty", "decline", "copy", "source-left", "clone", "unpaid", "illegal", "source-phase"} {
			t.Run(tc.name+"/"+scenario, func(t *testing.T) {
				a := &ronLibraryViewer{BasePlayer: mage.NewBasePlayer("A"), accept: scenario != "decline"}
				b := &ronLibraryViewer{BasePlayer: mage.NewBasePlayer("B")}
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				library := b
				if scenario == "self" {
					library = a
				}
				n := 7
				if scenario == "short" {
					n = 2
				}
				if scenario == "empty" {
					n = 0
				}
				cards := make([]mage.Card, n)
				for i := range cards {
					cards[i] = mage.NewLand(fmt.Sprintf("Card %d", i))
					cards[i].SetOwner(library.PlayerID())
					library.AddToLibrary(cards[i])
				}
				c := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
					t.Fatal(err)
				}
				c.SetOwner(a.PlayerID())
				var source uuid.UUID
				var err error
				targets := []uuid.UUID{library.PlayerID()}
				if scenario == "illegal" {
					targets = []uuid.UUID{uuid.New()}
				}
				if tc.spell {
					a.AddToHand(c)
					if scenario != "unpaid" {
						a.ManaPool().Add(tc.color, 1)
					}
					err = g.CastSpellByID(a.PlayerID(), c.ID(), targets, 0)
				} else {
					p := g.PutOnBattlefield(c, a.PlayerID())
					source = p.ID()
					p.RevokeBaseAttr(core.AttrSummonSick)
					if scenario == "unpaid" {
						g.TapPermanent(p)
					}
					err = g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, targets)
				}
				if scenario == "illegal" || scenario == "unpaid" {
					if err == nil || g.StackSize() != 0 || len(g.LibraryLooks(a.PlayerID())) != 0 || a.offers != 0 {
						t.Fatal("illegal announcement changed state")
					}
					if tc.spell && len(a.Hand()) != 1 {
						t.Fatal("failed cast lost card")
					}
					if scenario == "illegal" && tc.spell && a.ManaPool().Count(tc.color) != 1 {
						t.Fatal("failed target check spent mana")
					}
					if !tc.spell && scenario == "illegal" && g.FindPermanent(source).Tapped {
						t.Fatal("illegal target paid tap cost")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if len(g.LibraryLooks(a.PlayerID())) != 0 || a.offers != 0 || a.orders != 0 || g.StackSize() != 1 {
					t.Fatal("inspection ran at announcement")
				}
				if !tc.spell && !g.FindPermanent(source).Tapped {
					t.Fatal("tap cost missing")
				}
				if scenario == "source-left" && !tc.spell {
					g.BouncePermanentToHand(g.FindPermanent(source))
				}
				if scenario == "source-phase" && !tc.spell {
					g.PhaseOut(source)
				}
				if scenario == "copy" {
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				if scenario == "clone" {
					clone := g.Clone()
					clone.ResolveStack()
					if len(clone.LibraryLooks(a.PlayerID())) != min(tc.count, n) || len(g.LibraryLooks(a.PlayerID())) != 0 || a.offers != 0 {
						t.Fatal("clone changed original inspection")
					}
				}
				g.ResolveStack()
				want := min(tc.count, n)
				if len(g.LibraryLooks(a.PlayerID())) != want || len(g.PublicReveals()) != 0 {
					t.Fatal("wrong private inspection count")
				}
				if len(g.LibraryLooks(b.PlayerID())) != map[bool]int{false: 0, true: want}[scenario == "copy"] {
					t.Fatal("copy viewer identity")
				}
				{
					for i, seen := range g.LibraryLooks(a.PlayerID()) {
						expected := cards[i].ID()
						if scenario == "copy" && tc.name == "Natural Selection" {
							expected = cards[want-1-i].ID()
						}
						if seen.CardID != expected {
							t.Fatal("inspection used wrong cards")
						}
					}
				}
				wantOffers := 0
				if tc.spell {
					wantOffers = 1
				}
				wantShuffles := 0
				if tc.spell && scenario != "decline" {
					wantShuffles = 1
				}
				if library.shuffles != wantShuffles {
					t.Fatal("optional shuffle was not applied or decline was ignored")
				}
				if a.offers != wantOffers || (scenario != "copy" && b.offers != 0) {
					t.Fatal("wrong optional shuffle chooser")
				}
				if tc.name == "Natural Selection" && n > 0 && a.orders != 1 {
					t.Fatal("missing controller order choice")
				}
				if tc.name != "Natural Selection" && a.orders != 0 {
					t.Fatal("unchanged-order action prompted for order")
				}
				if len(library.Library()) != n || len(library.Graveyard()) != map[bool]int{false: 0, true: 1}[tc.spell && library == a] {
					t.Fatal("inspection moved library cards")
				}
			})
		}
	}
}

func TestRONLibraryInspectionReprints(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		c := repositoryCard(t, set, "Natural Selection")
		if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRONLibraryInspectionInvalid(t *testing.T) {
	for _, body := range []string{
		`LookAtTheTopNumberCardsOfPlayersLibrary(You,Integer(3),[])`,
		`LookAtTheTopNumberCardsOfPlayersLibrary(You,Integer(-1),[LeaveRemainingCardsOnTopOfLibraryInSameOrder])`,
		`LookAtTheTopNumberCardsOfPlayersLibrary(You,ValueX,[LeaveRemainingCardsOnTopOfLibraryInSameOrder])`,
		`LookAtTheTopNumberCardsOfPlayersLibrary(Ref_TargetPermanent,Integer(3),[LeaveRemainingCardsOnTopOfLibraryInSameOrder])`,
		`LookAtTheTopNumberCardsOfPlayersLibrary(Ref_TargetPlayer,Integer(3),[LeaveRemainingCardsOnTopOfLibraryInSameOrder])`,
		`LookAtTheTopNumberCardsOfPlayersLibrary(You,Integer(3),[LeaveRemainingCardsOnTopOfLibraryInSameOrder()])`,
		`LookAtTheTopNumberCardsOfPlayersLibrary(You,Integer(3),[LeaveRemainingCardsOnTopOfLibraryInSameOrder,Shuffle])`,
		`LookAtTheTopNumberCardsOfPlayersLibrary(You,Integer(3),[PutTheRemainingCardsOnBottomOfLibraryInAnyOrder])`,
		`MayHavePlayerAction(You,DrawACard)`, `MayHavePlayerAction(Ref_TargetPlayer,Shuffle)`, `MayHavePlayerAction(You,Shuffle())`,
	} {
		source := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([%s]))])`, body)
		nodes, err := Parse(strings.NewReader(source), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONLibraryInspectionInstancesAndInventory(t *testing.T) {
	source := `Card(Name:"Inspection",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([LookAtTheTopNumberCardsOfPlayersLibrary(Ref_TargetPlayer,Integer(3),[LeaveRemainingCardsOnTopOfLibraryInSameOrder]),MayHavePlayerAction(Ref_TargetPlayer,Shuffle)])))])`
	nodes, err := Parse(strings.NewReader(source), "instances.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	a := &ronLibraryViewer{BasePlayer: mage.NewBasePlayer("A")}
	b := &ronLibraryViewer{BasePlayer: mage.NewBasePlayer("B")}
	g := mage.NewGame(a, b)
	first, second := factory(), factory()
	if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("factory shared target declaration")
	}
	for _, p := range []*ronLibraryViewer{a, b} {
		for range 4 {
			p.AddToLibrary(mage.NewLand(p.Name() + " library"))
		}
	}
	first.SetOwner(a.PlayerID())
	second.SetOwner(b.PlayerID())
	a.AddToHand(first)
	b.AddToHand(second)
	if err = g.CastSpellByID(a.PlayerID(), first.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	if err = g.CastSpellByID(b.PlayerID(), second.ID(), []uuid.UUID{a.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	for _, p := range []*ronLibraryViewer{a, b} {
		seen := g.LibraryLooks(p.PlayerID())
		if len(seen) != 3 || seen[0].Name == p.Name()+" library" || p.offers != 1 {
			t.Fatal("pending resolution used another instance's binding")
		}
	}
	report, err := InspectFS(fstest.MapFS{"probe.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 {
		t.Fatal(report.Issues)
	}
	found := map[string]bool{}
	for _, sig := range report.Signatures {
		switch sig.Name {
		case "LeaveRemainingCardsOnTopOfLibraryInSameOrder":
			if sig.Role != "library_ordering_operation" {
				t.Fatal(sig.Role)
			}
			found[sig.Name] = true
		case "Shuffle":
			if sig.Role != "optional_library_action" {
				t.Fatal(sig.Role)
			}
			found[sig.Name] = true
		}
	}
	if len(found) != 2 {
		t.Fatal("inventory lost library operation roles")
	}
	for _, rule := range []string{
		`LookAtTheTopNumberCardsOfPlayersLibrary(You,Integer(3),[LeaveRemainingCardsOnTopOfLibraryInSameOrder])`,
		`Activated(LookAtTheTopNumberCardsOfPlayersLibrary(You,Integer(3),[LeaveRemainingCardsOnTopOfLibraryInSameOrder]),ActionList([DrawACard]))`,
		`AsPermanentEnters(ThisPermanent,[LookAtTheTopNumberCardsOfPlayersLibrary(You,Integer(3),[LeaveRemainingCardsOnTopOfLibraryInSameOrder])])`,
	} {
		nodes, err = Parse(strings.NewReader(fmt.Sprintf(`Card(Name:"Wrong context",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[%s])`, rule)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("wrong context accepted")
		}
	}
}

func TestRONLibraryInspectionSpyCastingAndSickness(t *testing.T) {
	g, a, b := ronGame()
	g.SetStep(core.PrecombatMain)
	c := repositoryCard(t, "FEM", "Orcish Spy")
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil {
		t.Fatal("unpaid creature cast")
	}
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(c.ID()) != nil || len(g.LibraryLooks(a.PlayerID())) != 0 {
		t.Fatal("creature entered early")
	}
	g.ResolveStack()
	p := g.FindPermanent(c.ID())
	if p == nil || p.CurrentPower(g) != 1 || p.CurrentToughness(g) != 1 {
		t.Fatal("creature entry or metadata")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err == nil {
		t.Fatal("summoning sickness ignored")
	}
	if p.Tapped || g.StackSize() != 0 {
		t.Fatal("failed activation spent cost")
	}
}

func TestRONLibraryInspectionRetargetAndControl(t *testing.T) {
	a := &ronLibraryViewer{BasePlayer: mage.NewBasePlayer("A")}
	b := &ronLibraryViewer{BasePlayer: mage.NewBasePlayer("B"), retarget: a.PlayerID()}
	g := mage.NewGame(a, b)
	for _, p := range []*ronLibraryViewer{a, b} {
		for range 3 {
			p.AddToLibrary(mage.NewLand(p.Name() + " card"))
		}
	}
	c := repositoryCard(t, "FEM", "Orcish Spy")
	c.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(c, a.PlayerID())
	source.RevokeBaseAttr(core.AttrSummonSick)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	cp := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true)
	if cp == nil || len(cp.Targets) != 1 || cp.Targets[0] != a.PlayerID() {
		t.Fatal("copy retarget failed")
	}
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.ResolveTopOfStack()
	if len(g.LibraryLooks(a.PlayerID())) != 0 || len(g.LibraryLooks(b.PlayerID())) != 3 || g.LibraryLooks(b.PlayerID())[0].Name != "A card" {
		t.Fatal("retarget or copy controller")
	}
	g.ResolveStack()
	if len(g.LibraryLooks(a.PlayerID())) != 3 || g.LibraryLooks(a.PlayerID())[0].Name != "B card" {
		t.Fatal("copy or control changed original target/viewer")
	}
}

func TestRONLibraryInspectionSorceryTiming(t *testing.T) {
	g, a, b := ronGame()
	g.SetStep(core.DeclareAttackers)
	c := repositoryCard(t, "LEG", "Visions")
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	a.ManaPool().Add(core.White, 1)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{b.PlayerID()}, 0); err == nil {
		t.Fatal("sorcery cast during combat")
	}
	if a.ManaPool().Count(core.White) != 1 || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("timing rejection spent resources")
	}
}
