package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type outsideChooser struct {
	*BasePlayer
	calls  int
	choose func([]Card) Card
}

func (p *outsideChooser) ChooseCardFromOutsideGame(cards []Card, _ string, _ GameReader) Card {
	p.calls++
	if p.choose != nil {
		return p.choose(cards)
	}
	return cards[len(cards)-1]
}
func outsideCard(owner uuid.UUID, name string) Card {
	c := NewArtifact(name, "")
	c.SetOwner(owner)
	return c
}
func TestOutsideGameEntry(t *testing.T) {
	for _, answer := range []string{"last", "nil", "foreign", "forged"} {
		t.Run(answer, func(t *testing.T) {
			a := &outsideChooser{BasePlayer: NewBasePlayer("A")}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			first, second := outsideCard(a.PlayerID(), "First"), outsideCard(a.PlayerID(), "Second")
			if err := g.SetOutsideGameCards(a.PlayerID(), []Card{first, second}); err != nil {
				t.Fatal(err)
			}
			if g.FindCardAnywhere(first.ID()) != nil || g.CardZone(first.ID()) != core.ZoneAny {
				t.Fatal("outside card is in a game zone")
			}
			listed := g.OutsideGameCards(a.PlayerID())
			listed[0].SetOwner(b.PlayerID())
			a.choose = func(cards []Card) Card {
				switch answer {
				case "nil":
					return nil
				case "foreign":
					return outsideCard(b.PlayerID(), "Foreign")
				case "forged":
					c := outsideCard(b.PlayerID(), "Forged")
					c.SetID(second.ID())
					return c
				}
				return cards[1]
			}
			clone := g.Clone()
			events := 0
			watcher := NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
				if e.ToZone == core.ZoneHand {
					events++
					if len(a.Hand()) != 1 || len(g.OutsideGameCards(a.PlayerID())) != 1 {
						t.Fatal("event before entry")
					}
				}
				return false
			})
			g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(watcher)), a.PlayerID())
			if err := PutOutsideGameCardInHand(You, CardFilter{}).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			want := second.ID()
			if answer == "nil" || answer == "foreign" {
				want = first.ID()
			}
			if len(a.Hand()) != 1 || a.Hand()[0].ID() != want || a.Hand()[0].Owner() != a.PlayerID() || events != 1 || a.calls != 1 {
				t.Fatal("choice, owner, entry or event", answer)
			}
			if len(clone.OutsideGameCards(a.PlayerID())) != 2 || len(clone.GetPlayer(a.PlayerID()).Hand()) != 0 {
				t.Fatal("clone shared entry")
			}
			if err := PutOutsideGameCardInHand(You, CardFilter{}).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			if len(a.Hand()) != 1 || len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 {
				t.Fatal("clone entry changed original")
			}
		})
	}
}
func TestOutsideGameEligibility(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	valid := outsideCard(a.PlayerID(), "Valid")
	if err := g.SetOutsideGameCards(a.PlayerID(), []Card{valid}); err != nil {
		t.Fatal(err)
	}
	token := NewArtifact("Token", "")
	token.SetOwner(a.PlayerID())
	token.isToken = true
	hand := outsideCard(a.PlayerID(), "Hand")
	a.AddToHand(hand)
	phased := outsideCard(a.PlayerID(), "Phased")
	p := g.PutOnBattlefield(phased, a.PlayerID())
	g.PhaseOut(p.ID())
	for _, cards := range [][]Card{{nil}, {valid, valid}, {outsideCard(b.PlayerID(), "Foreign")}, {token}, {hand}, {phased}} {
		if err := g.SetOutsideGameCards(a.PlayerID(), cards); err == nil {
			t.Fatal("invalid pool accepted")
		}
		if len(g.OutsideGameCards(a.PlayerID())) != 1 {
			t.Fatal("failed setup changed pool")
		}
	}
	if err := g.SetOutsideGameCards(uuid.New(), nil); err == nil {
		t.Fatal("absent player accepted")
	}
	restricted := NewArtifact("Restricted", "", WithAnteDeckRestriction())
	restricted.SetOwner(a.PlayerID())
	if err := g.SetOutsideGameCards(a.PlayerID(), []Card{restricted, valid}); err != nil {
		t.Fatal(err)
	}
	if err := PutOutsideGameCardInHand(You, CardFilter{}).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 2 || a.Hand()[1].ID() != valid.ID() {
		t.Fatal("ante restriction ignored")
	}
	if err := PutOutsideGameCardInHand(You, CardFilter{}).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 2 {
		t.Fatal("restricted card entered")
	}
}
func TestOutsideGameEntryRechecks(t *testing.T) {
	for _, change := range []string{"replace-pool", "in-game", "filter"} {
		t.Run(change, func(t *testing.T) {
			a := &outsideChooser{BasePlayer: NewBasePlayer("A")}
			g := NewGame(a, NewBasePlayer("B"))
			card := outsideCard(a.PlayerID(), "Card")
			if err := g.SetOutsideGameCards(a.PlayerID(), []Card{card}); err != nil {
				t.Fatal(err)
			}
			matches := true
			a.choose = func(cards []Card) Card {
				switch change {
				case "replace-pool":
					if err := g.SetOutsideGameCards(a.PlayerID(), []Card{card}); err != nil {
						t.Fatal(err)
					}
				case "in-game":
					a.AddToHand(card)
				case "filter":
					matches = false
				}
				return cards[0]
			}
			e := PutOutsideGameCardInHand(You, NewCardFilter("live", func(Card) bool { return matches }))
			if err := e.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			want := 0
			if change == "in-game" {
				want = 1
			}
			if len(a.Hand()) != want {
				t.Fatal("stale choice entered")
			}
		})
	}
}
func TestOutsideGameEntryDefinition(t *testing.T) {
	c := NewInstant("Invalid", "", WithAbility(Spell(PutOutsideGameCardInHand(nil, CardFilter{}))))
	if ValidateCardDefinition(c.Name(), c) == nil {
		t.Fatal("missing player accepted")
	}
}

func TestOutsideGameDrawReplacementCompetition(t *testing.T) {
	for _, choice := range []int{0, 1} {
		t.Run(string(rune('0'+choice)), func(t *testing.T) {
			a := &drawReplacementChooser{BasePlayer: NewBasePlayer("A"), choice: choice}
			g := NewGame(a, NewBasePlayer("B"))
			g.SetTurn(3)
			card := outsideCard(a.PlayerID(), "Outside")
			if err := g.SetOutsideGameCards(a.PlayerID(), []Card{card}); err != nil {
				t.Fatal(err)
			}
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			for _, body := range []ResolvingEffect{PutOutsideGameCardInHand(You, CardFilter{}), GainLifeFor(You, Fixed(3))} {
				if err := ReplaceNextDrawThisTurn(You, body).Apply(ctx); err != nil {
					t.Fatal(err)
				}
			}
			g.PlayerDrawCard(a)
			if a.offers != 1 || a.DrewFromEmpty() {
				t.Fatal("missing affected-player order or draw was not replaced")
			}
			if choice == 0 {
				if len(a.Hand()) != 1 || a.Life() != 20 {
					t.Fatal("outside replacement did not win")
				}
			} else if len(a.Hand()) != 0 || a.Life() != 23 {
				t.Fatal("other replacement did not win")
			}
			g.PlayerDrawCard(a)
			if len(a.Hand()) != 1 || a.Life() != 23 {
				t.Fatal("unchosen replacement was consumed")
			}
		})
	}
}

type cancelOutsideEntry struct {
	replacementBase
	calls int
}

func (r *cancelOutsideEntry) IsActive(GameReader) bool { return true }
func (r *cancelOutsideEntry) Matches(a Action, _ GameReader) bool {
	_, ok := a.(*OutsideGameEntryAction)
	return ok
}
func (r *cancelOutsideEntry) Replace(_ Action, _ *Game) Action { r.calls++; return nil }
func (r *cancelOutsideEntry) Clone() ReplacementEffect         { c := *r; return &c }
func TestOutsideGameEntryReplacement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := outsideCard(a.PlayerID(), "Outside")
	if err := g.SetOutsideGameCards(a.PlayerID(), []Card{c}); err != nil {
		t.Fatal(err)
	}
	rule := &cancelOutsideEntry{replacementBase: replacementBase{duration: core.EndOfTurn}}
	g.effects.AddReplacement(rule)
	if err := PutOutsideGameCardInHand(You, CardFilter{}).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 0 || len(g.OutsideGameCards(a.PlayerID())) != 1 || rule.calls != 1 {
		t.Fatal("cancelled entry moved the card")
	}
}

func TestOutsideGameAnteAndExplicitPlayer(t *testing.T) {
	for _, ante := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		if ante {
			var err error
			g, err = NewGameWithAnte(a, b, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
		}
		c := NewArtifact("Restricted", "", WithAnteDeckRestriction())
		c.SetOwner(b.PlayerID())
		if err := g.SetOutsideGameCards(b.PlayerID(), []Card{c}); err != nil {
			t.Fatal(err)
		}
		bound := BindPlayer("recipient")
		g.resolution.playerBindings = map[*PlayerBinding]uuid.UUID{bound: b.PlayerID()}
		if err := PutOutsideGameCardInHand(FromPlayer(bound), CardFilter{}).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		want := 0
		if ante {
			want = 1
		}
		if len(b.Hand()) != want || len(a.Hand()) != 0 {
			t.Fatal("ante rule or explicit recipient")
		}
	}
}
