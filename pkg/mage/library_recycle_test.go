package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLibraryRecycleBatch(t *testing.T) {
	for _, graveyard := range []bool{false, true} {
		for _, active := range []int{0, 1} {
			t.Run(string(rune('0'+active))+map[bool]string{false: "/hand", true: "/graveyard"}[graveyard], func(t *testing.T) {
				a := &librarySearchPlayer{BasePlayer: NewBasePlayer("A")}
				b := &librarySearchPlayer{BasePlayer: NewBasePlayer("B")}
				g := NewGame(a, b)
				g.SetActivePlayerIndex(active)
				for i, p := range []*librarySearchPlayer{a, b} {
					for range i + 1 {
						c := NewLand("Hand")
						c.SetOwner(p.PlayerID())
						p.AddToHand(c)
					}
					c := NewLand("Graveyard")
					c.SetOwner(p.PlayerID())
					p.AddToGraveyard(c)
					p.onShuffle = func() {
						if len(a.Hand()) != 0 || len(b.Hand()) != 0 {
							t.Error("shuffle saw an unmoved hand")
						}
						if graveyard && (len(a.Graveyard()) != 0 || len(b.Graveyard()) != 0) {
							t.Error("shuffle saw an unmoved graveyard")
						}
					}
				}
				var draws []uuid.UUID
				batches, zones, discards := 0, 0, 0
				observer := NewTriggered(core.EvtCardDrawn, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
					draws = append(draws, e.PlayerID)
					if a.shuffles != 1 || b.shuffles != 1 {
						t.Error("draw before both shuffles")
					}
					return false
				})
				zoneObserver := NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
					if e.ToZone == core.ZoneLibrary {
						zones++
						if len(a.Hand()) != 0 || len(b.Hand()) != 0 {
							t.Error("zone event saw partial batch")
						}
					}
					return false
				})
				batchObserver := NewTriggered(core.EvtCardsLeftGraveyard, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
					batches++
					if e.Amount != 1 || len(a.Graveyard()) != 0 || len(b.Graveyard()) != 0 {
						t.Error("graveyard batch event")
					}
					return false
				})
				discardObserver := NewTriggered(core.EvtDiscard, false).SetCondition(func(*core.GameEvent, GameReader, uuid.UUID, uuid.UUID) bool { discards++; return false })
				g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(observer), WithAbility(zoneObserver), WithAbility(batchObserver), WithAbility(discardObserver)), a.PlayerID())
				effect := ShuffleHandsIntoLibrariesAndDraw(EachPlayer)
				if graveyard {
					effect = Sequence(ShuffleHandsAndGraveyardsIntoLibraries(EachPlayer), DrawEach(EachPlayer, Fixed(1)))
				}
				if err := ApplyEffect(g, effect, uuid.Nil, b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				wantDraws, wantZones, wantBatches := 3, 3, 0
				if graveyard {
					wantDraws, wantZones, wantBatches = 2, 5, 2
				}
				if len(draws) != wantDraws || zones != wantZones || batches != wantBatches || discards != 0 {
					t.Fatalf("draws/zones/batches/discards: %d/%d/%d/%d", len(draws), zones, batches, discards)
				}
				first := a.PlayerID()
				if active == 1 {
					first = b.PlayerID()
				}
				if draws[0] != first || draws[len(draws)-1] == first {
					t.Fatal("draw order")
				}
			})
		}
	}
}

func TestLibraryRecycleEmptyAndDeclaration(t *testing.T) {
	a := &librarySearchPlayer{BasePlayer: NewBasePlayer("A")}
	b := &librarySearchPlayer{BasePlayer: NewBasePlayer("B")}
	g := NewGame(a, b)
	if err := ApplyEffect(g, ShuffleHandsIntoLibrariesAndDraw(EachPlayer), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.shuffles != 1 || b.shuffles != 1 || !a.IsAlive() || !b.IsAlive() {
		t.Fatal("empty shuffle")
	}
	for _, effect := range []ResolvingEffect{ShuffleHandsIntoLibrariesAndDraw(nil), ShuffleHandsAndGraveyardsIntoLibraries(nil)} {
		c := NewSorcery("Invalid", "", Spell(effect))
		if err := ValidateCardDefinition(c.Name(), c); err == nil {
			t.Fatal("nil collection accepted")
		}
	}
}

func TestLibraryRecycleLegacyOrder(t *testing.T) {
	a := &librarySearchPlayer{BasePlayer: NewBasePlayer("A")}
	b := &librarySearchPlayer{BasePlayer: NewBasePlayer("B")}
	g := NewGame(a, b)
	g.SetActivePlayerIndex(1)
	for _, p := range []*librarySearchPlayer{a, b} {
		card := NewLand("Hand")
		card.SetOwner(p.PlayerID())
		p.AddToHand(card)
	}
	var draws []uuid.UUID
	watcher := NewTriggered(core.EvtCardDrawn, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		draws = append(draws, e.PlayerID)
		if a.shuffles != 1 || b.shuffles != 1 {
			t.Error("legacy draw before both shuffles")
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(watcher)), a.PlayerID())
	if err := ApplyEffect(g, ShuffleHandAndGraveyardIntoLibraryAndDraw(1), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(draws) != 2 || draws[0] != b.PlayerID() {
		t.Fatal("legacy APNAP order")
	}
}

type libraryRecyclePreventDraw struct {
	replacementBase
	player uuid.UUID
}

func (r *libraryRecyclePreventDraw) Matches(action Action, _ GameReader) bool {
	draw, ok := action.(*DrawCardAction)
	return ok && draw.PlayerID() == r.player
}
func (*libraryRecyclePreventDraw) Replace(Action, *Game) Action { return nil }
func (*libraryRecyclePreventDraw) IsActive(GameReader) bool     { return true }
func (r *libraryRecyclePreventDraw) Clone() ReplacementEffect   { cloned := *r; return &cloned }

func TestLibraryRecycleCloneAndReplacement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	var cards []Card
	for i, p := range []*BasePlayer{a, b} {
		for range i + 1 {
			c := NewLand("Hand")
			c.SetOwner(p.PlayerID())
			p.AddToHand(c)
			cards = append(cards, c)
		}
		c := NewLand("Library")
		c.SetOwner(p.PlayerID())
		p.AddToLibrary(c)
		cards = append(cards, c)
	}
	clone := g.Clone()
	clone.AddReplacementEffect(&libraryRecyclePreventDraw{player: a.PlayerID()})
	effect := ShuffleHandsIntoLibrariesAndDraw(EachPlayer)
	if err := ApplyEffect(clone, effect, uuid.Nil, b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 1 || len(b.Hand()) != 2 || len(a.Library()) != 1 || len(b.Library()) != 1 {
		t.Fatal("clone changed original zones")
	}
	if len(clone.GetPlayer(a.PlayerID()).Hand()) != 0 || len(clone.GetPlayer(a.PlayerID()).Library()) != 2 || len(clone.GetPlayer(b.PlayerID()).Hand()) != 2 {
		t.Fatal("replacement changed another player's count")
	}
	for range 2 {
		if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(a.Hand()) != 1 || len(b.Hand()) != 2 || len(a.Library()) != 1 || len(b.Library()) != 1 {
			t.Fatal("repeated invocation lost counts")
		}
	}
	for _, game := range []*Game{g, clone} {
		seen := map[uuid.UUID]bool{}
		for _, p := range game.AllPlayers() {
			for _, zone := range [][]Card{p.Hand(), p.Library()} {
				for _, c := range zone {
					if seen[c.ID()] || c.Owner() != p.PlayerID() {
						t.Fatal("duplicate card or wrong owner")
					}
					seen[c.ID()] = true
				}
			}
		}
		for _, c := range cards {
			if !seen[c.ID()] {
				t.Fatal("lost card")
			}
		}
	}
}
