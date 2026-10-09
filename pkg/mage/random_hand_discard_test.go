package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRandomHandDiscardNamedBatch(t *testing.T) {
	for _, mode := range []string{"normal", "prevent", "library"} {
		t.Run(mode, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			first, second := NewArtifact("Cecily, Haunted Mage", ""), NewArtifact("Eleven, the Mage", "")
			for _, c := range []Card{first, second} {
				c.SetOwner(a.PlayerID())
				a.AddToHand(c)
			}
			if mode != "normal" {
				g.effects.AddReplacement(&handOperationReplacement{card: second.ID(), prevent: mode == "prevent"})
			}
			events := 0
			watcher := NewTriggered(core.EvtDiscard, false).SetCondition(func(_ *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
				events++
				want := 0
				if mode == "prevent" {
					want = 1
				}
				if len(a.Hand()) != want || len(g.PublicReveals()) != 2 {
					t.Error("event preceded complete reveal and movement")
				}
				return false
			})
			g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(watcher)), b.PlayerID())
			name := BindCardName("chosen")
			g.resolution.cardNameBindings = map[*CardNameBinding]string{name: "Eleven, the Mage"}
			g.SetRandomResults([]int{1, 0, 8})
			effect := RevealRandomHandAndDiscardMatchingName(You, Fixed(99), MatchesChosenCardName(FromCardName(name)))
			if err := effect.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			wantEvents := 2
			if mode == "prevent" {
				wantEvents = 1
			}
			if events != wantEvents || g.RandIntn(10) != 8 || len(g.PublicReveals()) != 2 {
				t.Fatal("batch, short-hand selection or randomness failed")
			}
			if mode == "library" && (len(a.Library()) != 1 || a.Library()[0].ID() != second.ID()) {
				t.Fatal("replacement lost selected card")
			}
		})
	}
}

func TestRandomHandDiscardFiltered(t *testing.T) {
	for _, count := range []int{0, 1, 3} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		for i := range count {
			var c Card = NewArtifact("Other", "")
			if i == 1 {
				c = NewCreature("Creature", "", 1, 1)
			}
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
		}
		g.SetRandomResults([]int{0, 8})
		if err := ApplyEffect(g, RevealHandAndDiscardRandomMatching(You, IsCreatureCard), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		want := 0
		if count == 3 {
			want = 1
		}
		if len(g.PublicReveals()) != count || len(a.Graveyard()) != want || len(a.Hand()) != count-want {
			t.Fatal("filtered selection or full reveal failed")
		}
		if want == 0 && g.RandIntn(10) != 0 || want == 1 && g.RandIntn(10) != 8 {
			t.Fatal("empty candidate set used randomness")
		}
	}
}

func TestRandomHandDiscardDeclarationsAndNoop(t *testing.T) {
	name := BindCardName("name")
	matcher := MatchesChosenCardName(FromCardName(name))
	for _, e := range []Effect{RevealRandomHandAndDiscardMatchingName(nil, Fixed(1), matcher), RevealRandomHandAndDiscardMatchingName(You, nil, matcher), RevealRandomHandAndDiscardMatchingName(You, Fixed(1), nil), RevealRandomHandAndDiscardMatchingName(You, Fixed(1), matcher), RevealHandAndDiscardRandomMatching(nil, IsCreatureCard)} {
		c := NewInstant("Invalid", "", Spell(e))
		if ValidateCardDefinition(c.Name(), c) == nil {
			t.Fatal("invalid definition accepted")
		}
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewArtifact("Held", "")
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	g.SetRandomResults([]int{7})
	for _, e := range []Effect{RevealRandomHandAndDiscardMatchingName(You, Fixed(0), matcher), RevealRandomHandAndDiscardMatchingName(You, Fixed(-1), matcher), RevealRandomHandAndDiscardMatchingName(FromPlayer(BindPlayer("absent")), Fixed(2), matcher), RevealHandAndDiscardRandomMatching(FromPlayer(BindPlayer("absent")), IsCreatureCard)} {
		if err := e.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.PublicReveals()) != 0 || len(a.Hand()) != 1 || g.RandIntn(10) != 7 {
		t.Fatal("noop used hand or randomness")
	}
}

func TestRandomHandDiscardWaitingAndCurrentHand(t *testing.T) {
	for _, state := range []string{"departed", "returned", "phased"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			body := RevealHandAndDiscardRandomMatching(ControllerOfWithLKI(This), IsCreatureCard)
			if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if state == "phased" {
				g.PhaseOut(p.ID())
			} else {
				g.BouncePermanentToHand(p)
				if state == "returned" {
					a.RemoveFromHand(p.ID())
					g.PutOnBattlefield(p.Card, b.PlayerID())
				}
			}
			c := NewCreature("Late hand card", "", 1, 1)
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			g.SetRandomResults([]int{0, 8})
			for range 2 {
				refs := g.WaitingDelayedPermanentSources()
				if len(refs) != 1 || len(g.PublicReveals()) != 0 || len(a.Graveyard()) != 0 {
					t.Fatal("waiting query executed body or lost source")
				}
			}
			clone := g.Clone()
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			if len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != c.ID() || len(b.Graveyard()) != 0 || g.RandIntn(10) != 8 {
				t.Fatal("delay lost current hand or exact source LKI")
			}
			if len(clone.PublicReveals()) != 0 || len(clone.GetPlayer(a.PlayerID()).Graveyard()) != 0 {
				t.Fatal("clone shared execution")
			}
		})
	}
}

func TestRandomHandDiscardDelayedNameFrame(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	name := BindCardName("saved")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	g.resolution.cardNameBindings = map[*CardNameBinding]string{name: "Island"}
	body := RevealRandomHandAndDiscardMatchingName(You, Fixed(1), MatchesChosenCardName(FromCardName(name)))
	if err := AtNext(EachEndStep, body).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.resolution.cardNameBindings[name] = "Mountain"
	if err := g.triggers.Delayed()[0].Effects[0].Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.resolution.cardNameBindings[name] != "Mountain" {
		t.Fatal("delayed frame changed outer name")
	}
	card := NewLand("Island")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		if len(game.GetPlayer(a.PlayerID()).Graveyard()) != 1 {
			t.Fatal("delayed body lost its saved name")
		}
	}
}

func TestRandomHandDiscardOnlyRevealedNames(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	cards := []Card{NewLand("Island"), NewLand("Island"), NewLand("Island")}
	for _, c := range cards {
		c.SetOwner(a.PlayerID())
		a.AddToHand(c)
	}
	name := BindCardName("name")
	g.resolution.cardNameBindings = map[*CardNameBinding]string{name: "Island"}
	g.SetRandomResults([]int{2, 0, 7})
	if err := RevealRandomHandAndDiscardMatchingName(You, Fixed(2), MatchesChosenCardName(FromCardName(name))).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 1 || a.Hand()[0].ID() != cards[1].ID() || len(a.Graveyard()) != 2 || len(g.PublicReveals()) != 2 || g.PublicReveals()[0].CardID != cards[2].ID() || g.PublicReveals()[1].CardID != cards[0].ID() || g.RandIntn(10) != 7 {
		t.Fatal("unrevealed matching card was discarded or selection repeated a card")
	}
}

type randomHandOrderPlayer struct {
	*BasePlayer
	orders int
	test   *testing.T
}

func (p *randomHandOrderPlayer) ChooseCardFromLibrary(cards []Card, _ string, g GameReader) Card {
	p.orders++
	if len(p.Hand()) != 3 || len(p.Graveyard()) != 0 || len(g.(*Game).PublicReveals()) != 3 {
		p.test.Error("graveyard order choice did not see complete revealed hand before moves")
	}
	return cards[len(cards)-1]
}

func TestRandomHandDiscardOwnerOrder(t *testing.T) {
	for _, replace := range []bool{false, true} {
		base, b := NewBasePlayer("A"), NewBasePlayer("B")
		a := &randomHandOrderPlayer{BasePlayer: base, test: t}
		g := NewGame(a, b)
		cards := []Card{NewLand("Island"), NewLand("Island"), NewLand("Island")}
		for _, c := range cards {
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
		}
		if replace {
			g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
		}
		name := BindCardName("name")
		g.resolution.cardNameBindings = map[*CardNameBinding]string{name: "Island"}
		g.SetRandomResults([]int{0, 0, 0})
		if err := RevealRandomHandAndDiscardMatchingName(You, Fixed(3), MatchesChosenCardName(FromCardName(name))).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		if replace {
			if a.orders != 0 || len(a.Library()) != 3 {
				t.Fatal("hidden destinations offered a graveyard order")
			}
		} else if a.orders != 2 || len(a.Graveyard()) != 3 || a.Graveyard()[0].ID() != cards[2].ID() || a.Graveyard()[2].ID() != cards[0].ID() {
			t.Fatal("owner could not order simultaneous discards")
		}
	}
}
