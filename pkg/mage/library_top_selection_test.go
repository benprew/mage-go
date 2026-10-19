package mage

import (
	"slices"
	"strconv"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type topSelectionPlayer struct {
	onChoice func()
	*BasePlayer
	choice   int
	offered  int
	observed []Card
}

func (p *topSelectionPlayer) ObserveLibraryLook(cards []Card, _ GameReader) { p.observed = cards }
func (p *topSelectionPlayer) ChooseCardFromLibrary(cards []Card, _ string, _ GameReader) Card {
	p.offered++
	if p.onChoice != nil {
		p.onChoice()
	}
	if p.choice < 0 {
		return NewLand("Foreign")
	}
	if p.choice >= len(cards) {
		return nil
	}
	return cards[p.choice]
}

func TestLibraryTopSelection(t *testing.T) {
	for _, choice := range []int{0, 1, -1, 99} {
		t.Run(strconv.Itoa(choice), func(t *testing.T) {
			a := &topSelectionPlayer{BasePlayer: NewBasePlayer("A"), choice: choice}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			cards := []Card{NewLand("One"), NewLand("Two"), NewLand("Three"), NewLand("Tail")}
			for _, c := range cards {
				c.SetOwner(b.PlayerID())
				b.AddToLibrary(c)
			}
			tailVisit := b.libraryVisit(cards[3].ID())
			chosenIndex := 0
			if choice == 1 {
				chosenIndex = 1
			}
			chosenVisit := b.libraryVisit(cards[chosenIndex].ID())
			effect := SelectLibraryTopAndBottomRestRandomly(You, ActivePlayer, Fixed(3))
			g.SetActivePlayerIndex(1)
			if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			selected := 0
			if choice == 1 {
				selected = 1
			}
			if b.Library()[0].ID() != cards[selected].ID() || b.Library()[1].ID() != cards[3].ID() || len(b.Library()) != 4 {
				t.Fatal("selection or untouched tail")
			}
			if b.libraryVisit(cards[selected].ID()) != chosenVisit {
				t.Fatal("retained chosen card lost its visit")
			}
			for _, c := range cards {
				if !slices.ContainsFunc(b.Library(), func(x Card) bool { return x.ID() == c.ID() }) {
					t.Fatal("card lost")
				}
			}
			if len(g.LibraryLooks(a.PlayerID())) != 3 || len(g.LibraryLooks(b.PlayerID())) != 0 || len(g.PublicReveals()) != 0 || a.offered != 1 {
				t.Fatal("privacy or choice count")
			}
			a.observed[0].SetOwner(a.PlayerID())
			if cards[0].Owner() != b.PlayerID() {
				t.Fatal("observer aliases card")
			}
			if len(b.Hand()) != 0 || len(b.Graveyard()) != 0 || b.libraryVisit(cards[3].ID()) != tailVisit {
				t.Fatal("zone or tail visit changed")
			}
		})
	}
}

func TestLibraryTopSelectionBoundsAndDeclarations(t *testing.T) {
	for _, count := range []int{0, 1, 8} {
		a := &topSelectionPlayer{BasePlayer: NewBasePlayer("A"), choice: 1}
		g := NewGame(a, NewBasePlayer("B"))
		a.AddToLibrary(NewLand("One"))
		a.AddToLibrary(NewLand("Two"))
		clone := g.Clone()
		if err := ApplyEffect(clone, SelectLibraryTopAndBottomRestRandomly(You, You, Fixed(count)), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(g.LibraryLooks(a.PlayerID())) != 0 || a.Library()[0].Name() != "One" {
			t.Fatal("clone changed original")
		}
		if err := ApplyEffect(g, SelectLibraryTopAndBottomRestRandomly(You, You, Fixed(count)), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(g.LibraryLooks(a.PlayerID())) != min(count, 2) {
			t.Fatal("count bounds")
		}
		a.SetLibrary(nil)
		before := a.offered
		if err := ApplyEffect(g, SelectLibraryTopAndBottomRestRandomly(You, You, Fixed(count)), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if a.offered != before {
			t.Fatal("empty library choice")
		}
	}
	for _, e := range []Effect{
		SelectLibraryTopAndBottomRestRandomly(nil, You, Fixed(1)),
		SelectLibraryTopAndBottomRestRandomly(You, nil, Fixed(1)),
		SelectLibraryTopAndBottomRestRandomly(You, You, nil),
		SelectLibraryTopAndBottomRestRandomly(You, You, Fixed(-1)),
	} {
		c := NewInstant("Bad", "", WithAbility(Spell(e)))
		if ValidateCardDefinition(c.Name(), c) == nil {
			t.Fatal("invalid selection declaration")
		}
	}
}

func TestLibraryTopSelectionPositiveActivationX(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	action := NewAction(ActionActivated, WithCost(ManaCostOf("{X}")), WithCost(Tap()), WithEffect(GainLifeFor(You, ValueAtResolution(XValue()))), WithActivationMinimumX(1))
	p := g.PutOnBattlefield(NewArtifact("Variable", "", WithAbility(action)), a.PlayerID())
	a.ManaPool().Add(core.Colorless, 2)
	for _, x := range []int{-1, 0, 3} {
		g.SetXValue(x)
		if g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil) == nil {
			t.Fatal("invalid X accepted")
		}
		if a.ManaPool().TotalMana() != 2 || p.Tapped || g.StackSize() != 0 {
			t.Fatal("failed payment changed state")
		}
	}
	g.SetXValue(2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.SetXValue(9)
	g.ResolveStack()
	if a.Life() != 22 || !p.Tapped || a.ManaPool().TotalMana() != 0 {
		t.Fatal("announced X lost")
	}
	for _, opts := range [][]ActionOption{
		{WithActivationMinimumX(0)},
		{WithActivationMinimumX(1), WithActivationMinimumX(1)},
	} {
		c := NewArtifact("Bad", "", WithAbility(NewAction(ActionActivated, append(opts, WithCost(ManaCostOf("{X}")), WithEffect(GainLifeFor(You, Fixed(1))))...)))
		if ValidateCardDefinition(c.Name(), c) == nil {
			t.Fatal("invalid minimum X")
		}
	}
	c := NewArtifact("Bad", "", WithAbility(NewAction(ActionActivated, WithActivationMinimumX(1), WithEffect(GainLifeFor(You, Fixed(1))))))
	if ValidateCardDefinition(c.Name(), c) == nil {
		t.Fatal("minimum without X")
	}
}

func TestLibraryTopSelectionReplacementOrder(t *testing.T) {
	for _, choice := range []int{0, 1} {
		a := &topSelectionPlayer{BasePlayer: NewBasePlayer("A"), choice: 1}
		chooser := &topSelectionReplacementChooser{topSelectionPlayer: a, order: choice}
		g := NewGame(chooser, NewBasePlayer("B"))
		g.SetTurn(3)
		cards := []Card{NewLand("One"), NewLand("Two"), NewLand("Three"), NewLand("Four"), NewLand("Five"), NewLand("Six")}
		for _, c := range cards {
			c.SetOwner(a.PlayerID())
			a.AddToLibrary(c)
		}
		for _, n := range []int{2, 3} {
			body := Sequence(SelectLibraryTopAndBottomRestRandomly(You, You, Fixed(n)), DrawFor(You, Fixed(1)))
			if err := ApplyEffect(g, ReplaceNextDrawThisTurn(You, body), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
		}
		g.PlayerDrawCard(chooser)
		if len(a.Hand()) != 1 || a.offered != 2 || len(g.LibraryLooks(a.PlayerID())) != 5 || chooser.orders != 1 {
			t.Fatal("replacement chain or order")
		}
		want := cards[2].ID()
		if choice == 1 {
			want = cards[3].ID()
		}
		if a.Hand()[0].ID() != want {
			t.Fatal("second replacement did not replace first replacement's draw")
		}
	}
}

type topSelectionReplacementChooser struct {
	*topSelectionPlayer
	order, orders int
}

func (p *topSelectionReplacementChooser) ChooseMode([]string, string) int { p.orders++; return p.order }

func TestLibraryTopSelectionMinimumXAvailability(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	c := NewArtifact("Variable", "", WithAbility(NewAction(ActionActivated, WithCost(ManaCostOf("{X}")), WithEffect(GainLifeFor(You, Fixed(1))), WithActivationMinimumX(1))))
	p := g.PutOnBattlefield(c, a.PlayerID())
	action := UnwrapAbility(p.RuntimeAbilities[0]).(*ActionDefinition)
	if action.CanActivate(a.PlayerID(), g) {
		t.Fatal("minimum X affordability ignored")
	}
	a.ManaPool().Add(core.Colorless, 1)
	if !action.CanActivate(a.PlayerID(), g) {
		t.Fatal("unannounced X hid legal activation")
	}
	if a.ManaPool().TotalMana() != 1 || g.XValue() != 0 || g.StackSize() != 0 {
		t.Fatal("availability query changed state")
	}
	bad := NewInstant("Spell", "{X}", WithAbility(NewAction(ActionSpell, WithActivationMinimumX(1), WithEffect(GainLifeFor(You, Fixed(1))))))
	if ValidateCardDefinition(bad.Name(), bad) == nil {
		t.Fatal("activation minimum accepted on spell")
	}
}

func TestLibraryTopSelectionReturnedVisit(t *testing.T) {
	a := &topSelectionPlayer{BasePlayer: NewBasePlayer("A"), choice: 1}
	g := NewGame(a, NewBasePlayer("B"))
	cards := []Card{NewLand("One"), NewLand("Two"), NewLand("Three")}
	for _, c := range cards {
		a.AddToLibrary(c)
	}
	a.onChoice = func() { a.SetLibrary(cards[1:]); a.SetLibrary(cards) }
	if err := ApplyEffect(g, SelectLibraryTopAndBottomRestRandomly(You, You, Fixed(2)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Library()[0].ID() != cards[0].ID() || a.Library()[1].ID() != cards[1].ID() {
		t.Fatal("returned library visit supplied saved selection")
	}
}
