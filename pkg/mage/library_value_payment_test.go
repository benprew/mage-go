package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLibraryValuePaymentContract(t *testing.T) {
	for _, scenario := range []string{"less", "equal", "paid", "declined", "poor", "no-find", "no-sacrifice", "phased", "clone"} {
		t.Run(scenario, func(t *testing.T) {
			a := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: scenario != "declined"}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			result := BindNumber("sacrificed mana value")
			filter := NewCardFilter("artifact", func(c Card) bool { return c.HasType(core.TypeArtifact) })
			effect := IfSacrificeWithManaValue(You, IsArtifact, result, SearchLibraryWithManaDifference(You, filter, FromNumber(result)))
			definition := NewSorcery("Exchange", "", Spell(effect))
			if err := ValidateCardDefinition("Exchange", definition); err != nil {
				t.Fatal(err)
			}
			old := NewArtifact("Old", "{2}")
			old.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(old, a.PlayerID())
			if scenario == "no-sacrifice" {
				g.ExilePermanent(p)
			}
			if scenario == "phased" {
				p.PhasedOut = true
			}
			cost := "{5}"
			if scenario == "less" {
				cost = "{1}"
			}
			if scenario == "equal" {
				cost = "{2}"
			}
			found := NewArtifact("Found", cost)
			found.SetOwner(a.PlayerID())
			if scenario != "no-find" {
				a.AddToLibrary(found)
			}
			mana := 3
			if scenario == "poor" {
				mana = 2
			}
			a.ManaPool().Add(core.Colorless, mana)
			if scenario == "clone" {
				clone := g.Clone()
				if err := ApplyEffect(clone, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if g.FindPermanent(old.ID()) == nil || len(a.Library()) != 1 || a.ManaPool().TotalMana() != mana {
					t.Fatal("clone changed original")
				}
			}
			if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			entered := scenario == "less" || scenario == "equal" || scenario == "paid" || scenario == "clone"
			if (g.FindPermanent(found.ID()) != nil) != entered {
				t.Fatal("wrong battlefield destination")
			}
			if entered && g.FindPermanent(found.ID()).ControllerID() != a.PlayerID() {
				t.Fatal("wrong controller")
			}
			grave := scenario == "declined" || scenario == "poor"
			if (len(a.Graveyard()) == 2) != grave {
				t.Fatal("unpaid card must use owner's graveyard")
			}
			expectedMana := mana
			if scenario == "paid" || scenario == "clone" {
				expectedMana -= 3
			}
			if a.ManaPool().TotalMana() != expectedMana {
				t.Fatal("wrong difference payment")
			}
			if scenario == "no-sacrifice" || scenario == "phased" {
				if len(a.Library()) != 1 {
					t.Fatal("searched without sacrifice")
				}
			}
			if len(g.PublicReveals()) != 0 {
				t.Fatal("search revealed the card")
			}
		})
	}
}

func TestLibraryValuePaymentDeclarations(t *testing.T) {
	filter := NewCardFilter("artifact", func(c Card) bool { return c.HasType(core.TypeArtifact) })
	for _, effect := range []ResolvingEffect{
		SearchLibraryWithManaDifference(nil, filter, Fixed(0)),
		SearchLibraryWithManaDifference(You, CardFilter{}, Fixed(0)),
		SearchLibraryWithManaDifference(You, filter, nil),
		SearchLibraryWithManaDifference(You, filter, FromNumber(BindNumber("missing"))),
		IfSacrificeWithManaValue(You, PermanentFilter{}, BindNumber("value"), GainLifeFor(You, Fixed(1))),
		IfSacrificeWithManaValue(You, IsArtifact, nil, GainLifeFor(You, Fixed(1))),
		IfSacrificeWithManaValue(You, IsArtifact, BindNumber("value"), nil),
	} {
		if ValidateCardDefinition("Invalid", NewSorcery("Invalid", "", Spell(effect))) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

type libraryValuePlayer struct {
	*BasePlayer
	accept   bool
	shuffles int
	choose   func([]Card) Card
	may      func()
}

func (p *libraryValuePlayer) ChooseMayAbility(string) bool {
	if p.may != nil {
		p.may()
	}
	return p.accept
}
func (p *libraryValuePlayer) ChooseCardFromLibrary(cards []Card, _ string, _ GameReader) Card {
	if p.choose != nil {
		return p.choose(cards)
	}
	if len(cards) == 0 {
		return nil
	}
	return cards[0]
}
func (p *libraryValuePlayer) ShuffleLibrary() { p.shuffles++; p.BasePlayer.ShuffleLibrary() }

func TestLibraryValuePaymentVisitsAndReplacements(t *testing.T) {
	for _, state := range []string{"return-at-choice", "return-at-payment", "invalid-choice", "no-match", "decline-find", "unpaid-exile", "unpaid-cancel", "sacrifice-exile", "sacrifice-replaced-no-move", "mana-source", "enters-tapped", "entry-fails"} {
		t.Run(state, func(t *testing.T) {
			a := &libraryValuePlayer{BasePlayer: NewBasePlayer("A"), accept: state != "unpaid-exile" && state != "unpaid-cancel"}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			old := NewArtifact("Old", "{2}")
			old.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(old, a.PlayerID())
			cost := "{3}"
			var options []CardOption
			if state == "enters-tapped" {
				options = append(options, WithKeyword(core.EntersTapped))
			}
			found := NewArtifact("Found", cost, options...)
			found.SetOwner(a.PlayerID())
			a.AddToLibrary(found)
			if state == "no-match" {
				a.SetLibrary([]Card{NewLand("Land")})
			}
			a.ManaPool().Add(core.Colorless, 1)
			if state == "mana-source" {
				a.ManaPool().Clear()
				g.PutOnBattlefield(NewLand("Mana", WithManaAbility(core.Green)), a.PlayerID())
			}
			cycle := func() { a.SetLibrary(nil); a.AddToLibrary(found) }
			switch state {
			case "return-at-choice":
				a.choose = func([]Card) Card { cycle(); return found }
			case "return-at-payment":
				a.may = cycle
			case "invalid-choice":
				a.choose = func([]Card) Card { return NewArtifact("Invalid", "") }
			case "decline-find":
				a.choose = func([]Card) Card { return nil }
			case "sacrifice-exile":
				g.AddExileIfWouldGoToGraveyardThisTurn(p.ID(), p.ID())
			case "sacrifice-replaced-no-move":
				g.effects.AddReplacement(&testPrioritizedReplacement{matchFunc: func(a Action) bool { _, ok := a.(*SacrificePermanentAction); return ok }, replaceFunc: func(Action, *Game) Action { return nil }})
			case "unpaid-exile", "unpaid-cancel":
				g.effects.AddReplacement(&testPrioritizedReplacement{matchFunc: func(a Action) bool {
					move, ok := a.(*CardZoneMoveAction)
					return ok && move.Destination() == core.ZoneGraveyard
				}, replaceFunc: func(a Action, _ *Game) Action {
					if state == "unpaid-cancel" {
						return nil
					}
					return a.(*CardZoneMoveAction).WithExile(p.ID())
				}})
			case "entry-fails":
				g.effects.AddReplacement(&testPrioritizedReplacement{matchFunc: func(a Action) bool { _, ok := a.(*battlefieldEntryAction); return ok }, replaceFunc: func(Action, *Game) Action { return nil }})
			}
			result := BindNumber("value")
			filter := NewCardFilter("artifact", func(c Card) bool { return c.HasType(core.TypeArtifact) })
			effect := IfSacrificeWithManaValue(You, IsArtifact, result, SearchLibraryWithManaDifference(You, filter, FromNumber(result)))
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			g.resolution.numberBindings = map[*NumberBinding]int{result: 99}
			if err := effect.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if FromNumber(result).resolveValue(ctx) != 99 {
				t.Fatal("result escaped continuation")
			}
			entered := state == "sacrifice-replaced-no-move" || state == "sacrifice-exile" || state == "mana-source" || state == "enters-tapped"
			if (g.FindPermanent(found.ID()) != nil) != entered {
				t.Fatal("wrong visit or entry result")
			}
			if state == "enters-tapped" && !g.FindPermanent(found.ID()).Tapped {
				t.Fatal("entry replacement omitted")
			}
			if state == "unpaid-exile" && g.FindExiledCard(found.ID()) == nil {
				t.Fatal("graveyard replacement omitted")
			}
			if state == "entry-fails" && len(a.Graveyard()) != 2 {
				t.Fatal("failed entry lost card")
			}
			expectedShuffles := 1
			if state == "sacrifice-replaced-no-move" {
				if g.FindPermanent(old.ID()) == nil {
					t.Fatal("failed sacrifice moved card")
				}
			}
			if a.shuffles != expectedShuffles {
				t.Fatal("shuffle ordering or count")
			}
			expectedMana := 1
			if entered || state == "entry-fails" || state == "return-at-payment" || state == "mana-source" {
				expectedMana = 0
			}
			if a.ManaPool().TotalMana() != expectedMana {
				t.Fatal("payment rollback or amount")
			}
		})
	}
}

func TestLibraryValuePaymentZoneEvents(t *testing.T) {
	for _, state := range []string{"entry", "unpaid", "entry-fails"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewArtifact("Find", "{1}")
			a.AddToLibrary(card)
			destination := core.ZoneBattlefield
			if state != "entry" {
				destination = core.ZoneGraveyard
			}
			events := 0
			observer := NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
				if e.SourceID == card.ID() {
					events++
					if e.FromZone != core.ZoneLibrary || e.ToZone != destination || len(a.Library()) != 0 {
						t.Error("wrong zone origin, destination or observation timing")
					}
				}
				return false
			})
			g.PutOnBattlefield(NewCreature("Observer", "", 1, 1, WithAbility(observer)), a.PlayerID())
			if state == "entry-fails" {
				g.effects.AddReplacement(&testPrioritizedReplacement{matchFunc: func(a Action) bool { _, ok := a.(*battlefieldEntryAction); return ok }, replaceFunc: func(Action, *Game) Action { return nil }})
			}
			allowance := 1
			if state == "unpaid" {
				allowance = 0
			}
			if err := ApplyEffect(g, SearchLibraryWithManaDifference(You, NewCardFilter("artifact", func(c Card) bool { return c.HasType(core.TypeArtifact) }), Fixed(allowance)), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if events != 1 {
				t.Fatalf("zone events %d, want 1", events)
			}
		})
	}
}
