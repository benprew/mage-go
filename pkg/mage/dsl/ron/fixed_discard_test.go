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

type ronDiscardPlayer struct {
	*mage.BasePlayer
	requests []int
}

func (p *ronDiscardPlayer) ChooseMayAbility(string) bool { return true }
func (p *ronDiscardPlayer) ChooseCardsFromHand(n int, _ string, _ mage.GameReader) []mage.Card {
	p.requests = append(p.requests, n)
	hand := p.Hand()
	return append([]mage.Card(nil), hand[len(hand)-min(n, len(hand)):]...)
}

func TestRONFixedDiscardRealCards(t *testing.T) {
	for _, tc := range []struct {
		set, name           string
		draw, discard, mana int
	}{
		{"ATQ", "Jalum Tome", 1, 1, 2},
		{"ARN", "Bazaar of Baghdad", 2, 3, 0},
	} {
		for _, change := range []string{"none", "departed", "returned", "control", "phased"} {
			for _, initial := range []int{0, 3} {
				t.Run(fmt.Sprintf("%s/%s/%d", tc.name, change, initial), func(t *testing.T) {
					a := &ronDiscardPlayer{BasePlayer: mage.NewBasePlayer("A")}
					b := mage.NewBasePlayer("B")
					g := mage.NewGame(a, b)
					fillRONLibrary(t, a.BasePlayer, 5)
					for range initial {
						c := repositoryCard(t, "LEB", "Forest")
						c.SetOwner(a.PlayerID())
						a.AddToHand(c)
					}
					card := repositoryCard(t, tc.set, tc.name)
					if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
						t.Fatal(err)
					}
					card.SetOwner(b.PlayerID())
					p := g.PutOnBattlefield(card, a.PlayerID())
					a.ManaPool().Add(core.Colorless, tc.mana)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if len(a.requests) != 0 || len(a.Hand()) != initial || !g.FindPermanent(p.ID()).Tapped || a.ManaPool().TotalMana() != 0 || g.GetStack().Size() != 1 {
						t.Fatal("discard happened during payment")
					}
					switch change {
					case "departed", "returned":
						g.DestroyPermanent(p)
						if change == "returned" {
							g.PutOnBattlefield(card, b.PlayerID())
						}
					case "control":
						if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ChangeController(mage.This, mage.You)), p.ID(), b.PlayerID(), nil); err != nil {
							t.Fatal(err)
						}
					case "phased":
						g.PhaseOut(p.ID())
					}
					clone := g.Clone()
					drawn := append([]mage.Card(nil), a.Library()[:tc.draw]...)
					g.ResolveStack()
					want := min(tc.discard, initial+tc.draw)
					if len(a.requests) != 1 || a.requests[0] != tc.discard || len(a.Hand()) != initial+tc.draw-want || len(a.Graveyard()) != want || len(b.Hand()) != 0 || len(a.Library()) != 5-tc.draw {
						t.Fatal("wrong discard count, order, or recipient")
					}
					for _, c := range drawn {
						found := false
						for _, discarded := range a.Graveyard() {
							found = found || discarded.ID() == c.ID()
						}
						if !found {
							t.Fatal("choice did not include the newly drawn card")
						}
					}
					if len(clone.GetPlayer(a.PlayerID()).Hand()) != initial {
						t.Fatal("resolution changed clone")
					}
					clone.ResolveStack()
					if len(clone.GetPlayer(a.PlayerID()).Hand()) != initial+tc.draw-want || len(clone.GetPlayer(a.PlayerID()).Graveyard()) != want {
						t.Fatal("clone lost discard context")
					}
				})
			}
		}
	}
}

func TestRONFixedDiscardInvalid(t *testing.T) {
	for _, action := range []string{
		`DiscardACard()`, `DiscardACard(Integer(1))`, `DiscardNumberCards`,
		`DiscardNumberCards(Integer(-1))`, `DiscardNumberCards(ValueX)`,
		`DiscardNumberCards(3)`, `DiscardNumberCards(Integer())`,
		`DiscardNumberCards(Integer(1), Integer(2))`,
		`PlayerAction(Ref_TargetPlayer, DiscardACard)`,
		`PlayerAction(ThisPermanent, DiscardACard)`,
		`PlayerAction(Trigger_ThatPlayer, DiscardACard)`,
	} {
		src := `Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Activated(PayMana([]), ActionList([` + action + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Errorf("accepted %s", action)
		}
	}
}

func TestRONFixedDiscardInventory(t *testing.T) {
	src := `Card(Name: "Discard", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Activated(PayMana([]), ActionList([DiscardNumberCards(Integer(3)), PlayerAction(You, DiscardNumberCards(Integer(2)))]))])`
	report, err := InspectFS(fstest.MapFS{"discard.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, entry := range report.Signatures {
		if entry.Name == "Integer" {
			seen++
			if entry.Role != "value" {
				t.Errorf("amount role = %s", entry.Role)
			}
		}
	}
	if seen != 2 {
		t.Fatalf("amount signatures = %d", seen)
	}
}

func TestRONFixedDiscardReplacementAndEvents(t *testing.T) {
	a := &ronDiscardPlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	fillRONLibrary(t, a.BasePlayer, 5)
	card := repositoryCard(t, "ARN", "Bazaar of Baghdad")
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	for range 2 {
		c := repositoryCard(t, "LEB", "Forest")
		c.SetOwner(a.PlayerID())
		a.AddToHand(c)
	}
	g.PutOnBattlefield(mage.NewArtifact("Replacement", "", mage.WithStaticAbility(mage.DiscardToLibraryReplacement())), a.PlayerID())
	seen := 0
	watcher := mage.NewTriggered(core.EvtDiscard, false).SetCondition(func(evt *core.GameEvent, _ mage.GameReader, _, _ uuid.UUID) bool {
		seen++
		if evt.PlayerID != a.PlayerID() || len(a.Hand()) != 1 || len(a.Library()) != 6 || len(a.Graveyard()) != 0 {
			t.Error("event saw partial replacement results")
		}
		return false
	})
	g.PutOnBattlefield(mage.NewArtifact("Watcher", "", mage.WithAbility(watcher)), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if seen != 3 || len(a.requests) != 1 {
		t.Fatal("discard bypassed choices or events")
	}
}

func TestRONFixedDiscardContexts(t *testing.T) {
	for _, tc := range []struct{ kind, rule string }{
		{"Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)], ActionList([PlayerAction(Ref_TargetPlayer, DiscardACard), DiscardACard])))`},
		{"Artifact", `Activated(PayMana([]), Targeted([TargetPlayer(AnyPlayer)], ActionList([PlayerAction(Ref_TargetPlayer, DiscardNumberCards(Integer(2))), DiscardACard])))`},
		{"Artifact", `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)), ActionList([DiscardACard]))`},
		{"Artifact", `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)), ActionList([DiscardACard]))`},
	} {
		t.Run(tc.rule, func(t *testing.T) {
			factory := permanentScopeFactory(t, tc.kind, tc.rule)
			first, second := factory(), factory()
			if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			for range 4 {
				x, y := repositoryCard(t, "LEB", "Forest"), repositoryCard(t, "LEB", "Forest")
				x.SetOwner(a.PlayerID())
				y.SetOwner(b.PlayerID())
				a.AddToHand(x)
				b.AddToHand(y)
			}
			first.SetOwner(a.PlayerID())
			wantA, wantB := 1, 0
			if tc.kind == "Instant" {
				if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
					t.Fatal("shared target state")
				}
				a.AddToHand(first)
				if err := g.CastSpellByID(a.PlayerID(), first.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
					t.Fatal(err)
				}
				if copied := g.CopySpellOnStack(first.ID(), b.PlayerID(), true); copied == nil || copied.Targets[0] != a.PlayerID() {
					t.Fatal("copy did not retarget")
				}
				wantA, wantB = 2, 2
			} else {
				p := g.PutOnBattlefield(first, a.PlayerID())
				if strings.HasPrefix(tc.rule, "Activated") {
					if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
						t.Fatal("shared target state")
					}
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
						t.Fatal(err)
					}
					wantB = 2
				} else if strings.Contains(tc.rule, "Upkeep") {
					g.SetActivePlayerIndex(1)
					g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
					g.ResolveStack()
					if len(a.Hand()) != 4 {
						t.Fatal("opponent upkeep discarded")
					}
					g.SetActivePlayerIndex(0)
					g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
				}
			}
			g.ResolveStack()
			if len(a.Hand()) != 4-wantA || len(b.Hand()) != 4-wantB {
				t.Fatalf("discard scopes: hands %d/%d", len(a.Hand()), len(b.Hand()))
			}
		})
	}
}

func TestRONFixedDiscardZeroEmptyAndContinuation(t *testing.T) {
	a := &ronDiscardPlayer{BasePlayer: mage.NewBasePlayer("A")}
	g := mage.NewGame(a, mage.NewBasePlayer("B"))
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]), ActionList([DiscardNumberCards(Integer(0)), DiscardACard, GainLife(Integer(1))]))`)
	card := factory()
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(a.requests) != 1 || a.requests[0] != 1 || a.Life() != 21 || len(a.Graveyard()) != 0 {
		t.Fatal("zero or empty discard stopped continuation")
	}
}

func TestRONFixedDiscardActivationRollback(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		mana      int
	}{{"ATQ", "Jalum Tome", 2}, {"ARN", "Bazaar of Baghdad", 0}} {
		for _, failure := range []string{"mana", "tapped", "phased"} {
			if failure == "mana" && tc.mana == 0 {
				continue
			}
			t.Run(tc.name+"/"+failure, func(t *testing.T) {
				g, a, _ := ronGame()
				fillRONLibrary(t, a, 3)
				card := repositoryCard(t, tc.set, tc.name)
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				mana := tc.mana
				if failure == "mana" {
					mana--
				}
				a.ManaPool().Add(core.Colorless, mana)
				if failure == "tapped" {
					g.TapPermanent(p)
				}
				var phased []uuid.UUID
				if failure == "phased" {
					phased = g.PhaseOut(p.ID())
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
					t.Fatal("illegal activation succeeded")
				}
				if !g.GetStack().IsEmpty() || len(a.Library()) != 3 || len(a.Hand()) != 0 || len(a.Graveyard()) != 0 || a.ManaPool().TotalMana() != mana {
					t.Fatal("failed activation changed state")
				}
				g.PhaseIn(phased)
				if g.FindPermanent(p.ID()).Tapped != (failure == "tapped") {
					t.Fatal("failed activation changed tap state")
				}
			})
		}
	}
}
