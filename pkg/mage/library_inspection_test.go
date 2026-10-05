package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type libraryInspectionPlayer struct {
	*BasePlayer
	order            []uuid.UUID
	offered, ordered int
	shuffles         int
	observed         []Card
	accept           bool
}

func (p *libraryInspectionPlayer) ObserveLibraryLook(cards []Card, _ GameReader) { p.observed = cards }

func (p *libraryInspectionPlayer) ChooseScryPlacement(top []Card, _ string, _ GameReader) ([]uuid.UUID, []uuid.UUID) {
	p.ordered++
	return nil, p.order
}
func (p *libraryInspectionPlayer) ShuffleLibrary() { p.shuffles++; p.BasePlayer.ShuffleLibrary() }

func (p *libraryInspectionPlayer) ChooseMayAbility(string) bool { p.offered++; return p.accept }

func TestLibraryInspectionPrivacyOrderAndClone(t *testing.T) {
	for _, reorder := range []bool{false, true} {
		a := &libraryInspectionPlayer{BasePlayer: NewBasePlayer("A")}
		b := &libraryInspectionPlayer{BasePlayer: NewBasePlayer("B")}
		g := NewGame(a, b)
		g.SetActivePlayerIndex(1)
		cards := []Card{NewLand("First"), NewLand("Second"), NewLand("Third"), NewLand("Fourth")}
		for _, c := range cards {
			c.SetOwner(b.PlayerID())
			b.AddToLibrary(c)
		}
		a.order = []uuid.UUID{cards[2].ID(), cards[0].ID(), cards[1].ID()}
		owner := BindPlayer("owner")
		effect := Sequence(SnapshotPlayer(ActivePlayer, owner), LookAtLibraryTop(You, FromPlayer(owner), 3, reorder))
		clone := g.Clone()
		if err := ApplyEffect(clone, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(g.LibraryLooks(a.PlayerID())) != 0 || b.Library()[0].ID() != cards[0].ID() {
			t.Fatal("clone changed original")
		}
		if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(a.observed) != 3 || len(b.observed) != 0 {
			t.Fatal("private observer identity")
		}
		a.observed[0].SetOwner(a.PlayerID())
		if cards[0].Owner() != b.PlayerID() {
			t.Fatal("observer aliases library card")
		}
		seen := g.LibraryLooks(a.PlayerID())
		if len(seen) != 3 || len(g.LibraryLooks(b.PlayerID())) != 0 || len(g.PublicReveals()) != 0 {
			t.Fatal("private inspection visibility")
		}
		for i, c := range cards[:3] {
			if seen[i].CardID != c.ID() {
				t.Fatal("inspection order")
			}
		}
		want := cards[0].ID()
		if reorder {
			want = cards[2].ID()
		}
		if b.Library()[0].ID() != want || b.Library()[3].ID() != cards[3].ID() || len(b.Hand()) != 0 || len(b.Graveyard()) != 0 {
			t.Fatal("order or zone changed")
		}
		if b.ordered != 0 || a.ordered != map[bool]int{false: 0, true: 1}[reorder] {
			t.Fatal("wrong chooser")
		}
		seen[0].Card.SetOwner(a.PlayerID())
		seen[0].Types[0] = core.TypeCreature
		if g.LibraryLooks(a.PlayerID())[0].Card.Owner() != b.PlayerID() || g.LibraryLooks(a.PlayerID())[0].Types[0] != core.TypeLand {
			t.Fatal("history aliases caller")
		}
		if err := ApplyEffect(clone, LookAtLibraryTop(You, You, 3, false), uuid.Nil, b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(g.LibraryLooks(b.PlayerID())) != 0 {
			t.Fatal("history aliases clone")
		}
	}
}

func TestLibraryInspectionBoundsInvalidOrderAndShuffle(t *testing.T) {
	for _, n := range []int{0, 1, 3} {
		for _, accept := range []bool{false, true} {
			a := &libraryInspectionPlayer{BasePlayer: NewBasePlayer("A"), accept: accept}
			b := &libraryInspectionPlayer{BasePlayer: NewBasePlayer("B")}
			g := NewGame(a, b)
			g.SetActivePlayerIndex(1)
			for range n {
				b.AddToLibrary(NewLand("Card"))
			}
			before := slices.Clone(b.Library())
			a.order = []uuid.UUID{uuid.New(), uuid.New()}
			if err := ApplyEffect(g, LookAtLibraryTop(You, ActivePlayer, 5, true), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if len(g.LibraryLooks(a.PlayerID())) != n || !slices.Equal(before, b.Library()) {
				t.Fatal("invalid order lost or moved cards")
			}
			if err := ApplyEffect(g, MayShuffleLibrary(You, ActivePlayer), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if b.shuffles != map[bool]int{false: 0, true: 1}[accept] {
				t.Fatal("optional shuffle did not follow choice")
			}
			if a.offered != 1 || b.offered != 0 || len(b.Library()) != n {
				t.Fatal("shuffle chooser or card conservation")
			}
			for _, c := range before {
				if !slices.Contains(b.Library(), c) {
					t.Fatal("shuffle lost card")
				}
			}
		}
	}
}

func TestLibraryInspectionDeclarationsAndMissingPlayers(t *testing.T) {
	for _, e := range []Effect{LookAtLibraryTop(nil, You, 3, false), LookAtLibraryTop(You, nil, 3, false), LookAtLibraryTop(You, You, -1, false), LookAtLibraryTop(You, DeclarePlayerTarget("missing"), 3, false), MayShuffleLibrary(nil, You), MayShuffleLibrary(You, nil)} {
		c := NewSorcery("Invalid", "", WithAbility(Spell(e)))
		if ValidateCardDefinition(c.Name(), c) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	a := &libraryInspectionPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	for _, e := range []Effect{LookAtLibraryTop(You, ControllerOf(This), 3, true), MayShuffleLibrary(You, ControllerOf(This)), LookAtLibraryTop(You, You, 0, true)} {
		if err := ApplyEffect(g, e, uuid.New(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if a.offered != 0 || a.ordered != 0 || len(g.LibraryLooks(a.PlayerID())) != 0 {
		t.Fatal("missing or empty inspection made decisions")
	}
}

func TestLibraryInspectionOrderValidation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		order []int
		want  []int
	}{
		{"complete", []int{2, 0, 1}, []int{2, 0, 1}},
		{"duplicate", []int{0, 0, 2}, []int{0, 1, 2}},
		{"missing", []int{1, 0}, []int{0, 1, 2}},
		{"extra", []int{2, 1, 0, 2}, []int{0, 1, 2}},
		{"unknown", []int{0, -1, 2}, []int{0, 1, 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &libraryInspectionPlayer{BasePlayer: NewBasePlayer("A")}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			g.SetActivePlayerIndex(1)
			cards := []Card{NewLand("First"), NewLand("Second"), NewLand("Third"), NewLand("Tail")}
			b.SetLibrary(cards)
			for _, i := range tc.order {
				id := uuid.New()
				if i >= 0 {
					id = cards[i].ID()
				}
				a.order = append(a.order, id)
			}
			if err := ApplyEffect(g, LookAtLibraryTop(You, ActivePlayer, 3, true), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			for i, index := range tc.want {
				if b.Library()[i].ID() != cards[index].ID() {
					t.Fatal("invalid permutation handling")
				}
			}
			if b.Library()[3].ID() != cards[3].ID() {
				t.Fatal("uninspected suffix changed")
			}
		})
	}
}
