package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type boundedSearchPlayer struct {
	*BasePlayer
	accept                    bool
	limit                     int
	choices, offers, shuffles int
	choose                    func([]Card) Card
	onShuffle                 func()
}

func (p *boundedSearchPlayer) ChooseMayAbility(string) bool { p.offers++; return p.accept }
func (p *boundedSearchPlayer) ChooseCardFromLibrary(cards []Card, _ string, _ GameReader) Card {
	p.choices++
	if p.choose != nil {
		return p.choose(cards)
	}
	if p.choices > p.limit {
		return nil
	}
	return cards[0]
}
func (p *boundedSearchPlayer) ShuffleLibrary() {
	p.shuffles++
	if p.onShuffle != nil {
		p.onShuffle()
	}
	p.BasePlayer.ShuffleLibrary()
}
func boundedSearchFilter() CardFilter {
	return NewCardFilter("basic land", func(c Card) bool { return c.HasType(core.TypeLand) && c.HasSuperType(core.SuperBasic) })
}
func TestBoundedLibrarySearchChoices(t *testing.T) {
	for _, maximum := range []int{0, 1, 3, 6} {
		for _, count := range []int{0, 1, 2, 3, 4} {
			for _, accept := range []bool{false, true} {
				t.Run(fmt.Sprintf("%d/%d/%t", maximum, count, accept), func(t *testing.T) {
					a := &boundedSearchPlayer{BasePlayer: NewBasePlayer("A"), accept: accept, limit: count}
					b := NewBasePlayer("B")
					g := NewGame(a, b)
					for range 4 {
						c := NewLand("Basic", WithSuperTypes(core.SuperBasic))
						c.SetOwner(a.PlayerID())
						a.AddToLibrary(c)
					}
					a.AddToLibrary(NewLand("Nonbasic", WithSubTypes("Forest")))
					b.AddToLibrary(NewLand("Other library"))
					want := 0
					if accept {
						want = min(maximum, count, 4)
					}
					events := 0
					observer := NewTriggered(core.EvtZoneChange, false).SetCondition(func(event *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
						if event.Type == core.EvtZoneChange && event.FromZone == core.ZoneLibrary && event.ToZone == core.ZoneHand {
							events++
							if len(a.Hand()) != want || len(g.PublicReveals()) != want {
								t.Error("batch must finish before zone observations")
							}
						}
						return false
					})
					g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(observer)), a.PlayerID())
					a.onShuffle = func() {
						if len(a.Hand()) != want || len(g.PublicReveals()) != want {
							t.Error("reveal and movement must precede shuffle")
						}
					}
					ApplyEffect(g, MaySearchLibraryToHand(You, maximum, boundedSearchFilter()), a.PlayerID(), a.PlayerID(), nil)
					if len(a.Hand()) != want || len(a.Library()) != 5-want || events != want || len(b.Library()) != 1 {
						t.Fatal("bounded search count or destination")
					}
					if a.offers != 1 || a.shuffles != boolToCount(accept) {
						t.Fatal("optional search or shuffle")
					}
					for _, c := range a.Hand() {
						if !boundedSearchFilter().Match(c) {
							t.Fatal("found a nonbasic card")
						}
					}
				})
			}
		}
	}
}
func boolToCount(v bool) int {
	if v {
		return 1
	}
	return 0
}

func TestBoundedLibrarySearchVisits(t *testing.T) {
	for _, change := range []string{"foreign", "duplicate", "return", "later-return", "forged", "owner"} {
		t.Run(change, func(t *testing.T) {
			a := &boundedSearchPlayer{BasePlayer: NewBasePlayer("A"), accept: true, limit: 3}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			first := NewLand("First", WithSuperTypes(core.SuperBasic))
			first.SetOwner(a.PlayerID())
			second := NewLand("Second", WithSuperTypes(core.SuperBasic))
			second.SetOwner(a.PlayerID())
			a.AddToLibrary(first)
			a.AddToLibrary(second)
			a.choose = func(cards []Card) Card {
				switch change {
				case "foreign":
					return NewLand("Foreign")
				case "duplicate":
					return first
				case "return":
					a.SetLibrary([]Card{second})
					a.AddToLibrary(first)
					return first
				case "later-return":
					if a.choices == 1 {
						return first
					}
					a.SetLibrary([]Card{second})
					a.AddToLibrary(first)
					return nil
				case "forged":
					c := NewLand("Forged")
					c.SetID(first.ID())
					return c
				case "owner":
					first.SetOwner(b.PlayerID())
					return first
				}
				return nil
			}
			ApplyEffect(g, MaySearchLibraryToHand(You, 3, boundedSearchFilter()), first.ID(), a.PlayerID(), nil)
			want := 1
			if change == "foreign" || change == "return" || change == "later-return" {
				want = 0
			}
			if len(a.Hand())+len(b.Hand()) != want || len(g.PublicReveals()) != want || a.shuffles != 1 {
				t.Fatal("saved search visit or choice validation")
			}
			if change == "forged" && a.Hand()[0].Name() != "First" {
				t.Fatal("used a forged choice instead of the library card")
			}
			if change == "owner" && len(b.Hand()) != 1 {
				t.Fatal("hand destination must follow owner")
			}
		})
	}
}

func TestBoundedLibrarySearchDeclarationsAndEmpty(t *testing.T) {
	for _, effect := range []Effect{MaySearchLibraryToHand(nil, 3, boundedSearchFilter()), MaySearchLibraryToHand(You, -1, boundedSearchFilter())} {
		c := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))
		if ValidateCardDefinition(c.Name(), c) == nil {
			t.Fatal("accepted invalid search declaration")
		}
	}
	a := &boundedSearchPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	ApplyEffect(g, MaySearchLibraryToHand(You, 3, boundedSearchFilter()), a.PlayerID(), a.PlayerID(), nil)
	if a.offers != 1 || a.choices != 0 || a.shuffles != 1 {
		t.Fatal("accepted empty search must shuffle")
	}
}

func TestBoundedLibrarySearchWaitingAndClone(t *testing.T) {
	a := &boundedSearchPlayer{BasePlayer: NewBasePlayer("A"), accept: true, limit: 3}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	saved := g.CaptureDamageSource(source.ID())
	first := NewLand("Found", WithSuperTypes(core.SuperBasic))
	first.SetOwner(a.PlayerID())
	a.AddToLibrary(first)
	if err := ApplyEffect(g, AtNext(EachEndStep, MaySearchLibraryToHand(ControllerOfWithLKI(This), 3, boundedSearchFilter())), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	check := func() {
		t.Helper()
		refs := g.WaitingDelayedPermanentSources()
		if len(refs) != 1 || !refs[0].SameObject(saved) || a.offers != 0 || a.choices != 0 || a.shuffles != 0 {
			t.Fatal("waiting query must preserve explicit source without a choice")
		}
	}
	check()
	g.PhaseIn(g.PhaseOut(source.ID()))
	check()
	g.BouncePermanentToHand(source)
	check()
	a.RemoveFromHand(source.ID())
	g.PutOnBattlefield(source.Card, b.PlayerID())
	check()
	clone := g.Clone()
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if len(a.Hand()) != 1 || len(b.Hand()) != 0 || len(g.PublicReveals()) != 1 || len(clone.GetPlayer(a.PlayerID()).Library()) != 1 || len(clone.WaitingDelayedPermanentSources()) != 1 {
		t.Fatal("saved departure controller or clone isolation")
	}
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 || len(clone.PublicReveals()) != 1 {
		t.Fatal("cloned delayed search")
	}
	reveals := clone.PublicReveals()
	reveals[0].Card.SetOwner(b.PlayerID())
	if clone.PublicReveals()[0].Card.Owner() != a.PlayerID() || g.PublicReveals()[0].Card.Owner() != a.PlayerID() {
		t.Fatal("mutable reveal history")
	}
}

func TestBoundedLibrarySearchNoMatchAndAbsent(t *testing.T) {
	a := &boundedSearchPlayer{BasePlayer: NewBasePlayer("A"), accept: true, limit: 3}
	g := NewGame(a, NewBasePlayer("B"))
	a.AddToLibrary(NewLand("Nonbasic", WithSubTypes("Forest")))
	if err := ApplyEffect(g, MaySearchLibraryToHand(You, 3, boundedSearchFilter()), a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 0 || len(a.Library()) != 1 || a.choices != 0 || a.shuffles != 1 {
		t.Fatal("quality search with no match")
	}
	if err := ApplyEffect(g, MaySearchLibraryToHand(FromPlayer(BindPlayer("missing")), 3, boundedSearchFilter()), a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.offers != 1 {
		t.Fatal("missing player prompted")
	}
	invalid := NewArtifact("Unbound", "", WithAbility(Activate(Costs(), MaySearchLibraryToHand(FromPlayer(BindPlayer("unbound")), 3, boundedSearchFilter()))))
	if ValidateCardDefinition(invalid.Name(), invalid) == nil {
		t.Fatal("accepted an unbound player")
	}
}

func TestBoundedLibrarySearchSimulationChoice(t *testing.T) {
	player := NewSearchPlayer(NewBasePlayer("Search"))
	if !player.ChooseMayAbility(MaySearchLibraryToHand(You, 3, boundedSearchFilter()).Text()) {
		t.Fatal("simulation declined a beneficial library search")
	}
}
