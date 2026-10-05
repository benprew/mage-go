package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONLibraryRecycleCards(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"LEA", "Timetwister"}, {"LEB", "Timetwister"}, {"2ED", "Timetwister"}, {"CED", "Timetwister"}, {"CEI", "Timetwister"}, {"LEG", "Winds of Change"}} {
		for _, active := range []int{0, 1} {
			for _, copySpell := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/%d/%v", tc.set, tc.name, active, copySpell), func(t *testing.T) {
					g, a, b := ronGame()
					g.SetActivePlayerIndex(active)
					caster := a
					if active == 1 {
						caster = b
					}
					for i, player := range []*mage.BasePlayer{a, b} {
						fillRONLibrary(t, player, 12)
						for range i + 1 {
							c := mage.NewLand("Hand")
							c.SetOwner(player.PlayerID())
							player.AddToHand(c)
						}
						c := mage.NewLand("Graveyard")
						c.SetOwner(player.PlayerID())
						player.AddToGraveyard(c)
					}
					c := repositoryCard(t, tc.set, tc.name)
					if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
						t.Fatal(err)
					}
					c.SetOwner(caster.PlayerID())
					caster.AddToHand(c)
					if err := g.CastSpellByID(caster.PlayerID(), c.ID(), nil, 0); err == nil {
						t.Fatal("unpaid spell accepted")
					}
					if g.StackSize() != 0 || len(a.Library()) != 12 || len(b.Library()) != 12 {
						t.Fatal("failed cast changed state")
					}
					if tc.name == "Timetwister" {
						caster.ManaPool().Add(core.Blue, 1)
						caster.ManaPool().Add(core.Colorless, 2)
					} else {
						caster.ManaPool().Add(core.Red, 1)
					}
					if err := g.CastSpellByID(caster.PlayerID(), c.ID(), nil, 0); err != nil {
						t.Fatal(err)
					}
					if len(a.Hand()) != 1 || len(b.Hand()) != 2 || caster.ManaPool().TotalMana() != 0 {
						t.Fatal("announcement ran shuffle")
					}
					if copySpell && g.CopySpellOnStack(c.ID(), b.PlayerID(), true) == nil {
						t.Fatal("copy failed")
					}
					clone := g.Clone()
					clone.ResolveStack()
					if len(a.Hand()) != 1 || len(b.Hand()) != 2 || len(a.Library()) != 12 {
						t.Fatal("clone changed original")
					}
					var draws []uuid.UUID
					watcher := mage.NewTriggered(core.EvtCardDrawn, false).SetCondition(func(e *core.GameEvent, _ mage.GameReader, _, _ uuid.UUID) bool {
						draws = append(draws, e.PlayerID)
						return false
					})
					g.PutOnBattlefield(mage.NewArtifact("Observer", "", mage.WithAbility(watcher)), a.PlayerID())
					g.ResolveStack()
					wantA, wantB, wantLibraryA, wantLibraryB, wantDraws := 1, 2, 12, 12, 3
					wantGraveyardA, wantGraveyardB := 1, 1
					if tc.name == "Timetwister" {
						wantA, wantB, wantLibraryA, wantLibraryB, wantDraws = 7, 7, 7, 8, 14
						wantGraveyardA, wantGraveyardB = 0, 0
					}
					if copySpell {
						wantDraws *= 2
					}
					if active == 0 {
						wantGraveyardA++
					} else {
						wantGraveyardB++
					}
					if len(a.Hand()) != wantA || len(b.Hand()) != wantB || len(a.Library()) != wantLibraryA || len(b.Library()) != wantLibraryB || len(a.Graveyard()) != wantGraveyardA || len(b.Graveyard()) != wantGraveyardB || len(draws) != wantDraws {
						t.Fatalf("wrong zones/draws: hands %d/%d libraries %d/%d graveyards %d/%d draws %d", len(a.Hand()), len(b.Hand()), len(a.Library()), len(b.Library()), len(a.Graveyard()), len(b.Graveyard()), len(draws))
					}
					if draws[0] != caster.PlayerID() {
						t.Fatal("draw order")
					}
					for _, player := range []*mage.BasePlayer{a, b} {
						for _, zone := range [][]mage.Card{player.Hand(), player.Library(), player.Graveyard()} {
							for _, card := range zone {
								if card.Owner() != player.PlayerID() {
									t.Fatal("owner changed")
								}
							}
						}
						cp := clone.GetPlayer(player.PlayerID())
						if len(cp.Hand()) != len(player.Hand()) || len(cp.Library()) != len(player.Library()) {
							t.Fatal("clone lost counts")
						}
					}
					if !strings.Contains(caster.Graveyard()[len(caster.Graveyard())-1].Name(), tc.name) {
						t.Fatal("resolving spell entered shuffle")
					}
				})
			}
		}
	}
}

func TestRONLibraryRecycleDeclarations(t *testing.T) {
	for _, body := range []string{
		`EachPlayerAction(AnyPlayer,ShuffleHandIntoLibrary)`,
		`EachPlayerAction(AnyPlayer,ShuffleHandIntoLibrary()),EachPlayerAction(AnyPlayer,DrawNumberCards(NumCardsShuffledIntoLibraryThisWay))`,
		`EachPlayerAction(AnyPlayer,DrawNumberCards(NumCardsShuffledIntoLibraryThisWay))`,
		`EachPlayerAction(Opponent,ShuffleHandIntoLibrary),EachPlayerAction(AnyPlayer,DrawNumberCards(NumCardsShuffledIntoLibraryThisWay))`,
		`EachPlayerAction(AnyPlayer,ShuffleHandIntoLibrary),EachPlayerAction(Opponent,DrawNumberCards(NumCardsShuffledIntoLibraryThisWay))`,
		`EachPlayerAction(AnyPlayer,ShuffleHandIntoLibrary),GainLife(Integer(1)),EachPlayerAction(AnyPlayer,DrawNumberCards(NumCardsShuffledIntoLibraryThisWay))`,
		`EachPlayerAction(AnyPlayer,ShuffleHandIntoLibrary),EachPlayerAction(AnyPlayer,DrawNumberCards(NumCardsShuffledIntoLibraryThisWay()))`,
		`EachPlayerAction(AnyPlayer,ShuffleHandAndGraveyardIntoLibrary()),EachPlayerAction(AnyPlayer,DrawNumberCards(Integer(7)))`,
		`EachPlayerAction(AnyPlayer,ShuffleHandAndGraveyardIntoLibrary),EachPlayerAction(AnyPlayer,DrawNumberCards(NumCardsShuffledIntoLibraryThisWay))`,
	} {
		text := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([` + body + `]))])`
		nodes, err := Parse(strings.NewReader(text), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid shuffle result accepted", body)
		}
	}
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "LEG.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range nodes {
		if node.Fields["Name"].String != "Winds of Change" {
			continue
		}
		factory, err := Compile(node)
		if err != nil {
			t.Fatal(err)
		}
		if factory().Abilities()[0] == factory().Abilities()[0] {
			t.Fatal("shared factory ability")
		}
	}
	_, err = LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([EachPlayerAction(AnyPlayer,DrawNumberCards(NumCardsShuffledIntoLibraryThisWay))]))])`)}}, "*.ron")
	if err == nil {
		t.Fatal("invalid collection accepted")
	}
}

func TestRONLibraryRecycleReplacementsAndEmpty(t *testing.T) {
	for _, name := range []string{"Timetwister", "Winds of Change"} {
		for _, empty := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%v", name, empty), func(t *testing.T) {
				g, a, b := ronGame()
				if !empty {
					fillRONLibrary(t, a, 12)
					fillRONLibrary(t, b, 12)
					for i, p := range []*mage.BasePlayer{a, b} {
						for range i + 1 {
							c := mage.NewLand("Hand")
							c.SetOwner(p.PlayerID())
							p.AddToHand(c)
						}
					}
					g.AddReplacementEffect(&ronPreventPlayerDraw{player: a.PlayerID()})
				}
				set := "LEG"
				if name == "Timetwister" {
					set = "LEB"
					a.ManaPool().Add(core.Blue, 1)
					a.ManaPool().Add(core.Colorless, 2)
				} else {
					a.ManaPool().Add(core.Red, 1)
				}
				c := repositoryCard(t, set, name)
				c.SetOwner(a.PlayerID())
				a.AddToHand(c)
				if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				if empty {
					if a.IsAlive() != (name == "Winds of Change") || b.IsAlive() != (name == "Winds of Change") {
						t.Fatal("empty hands attempted wrong draws")
					}
				} else {
					want := 2
					if name == "Timetwister" {
						want = 7
					}
					if len(a.Hand()) != 0 || len(b.Hand()) != want || len(a.Library()) != 13 {
						t.Fatal("replacement changed another player's amount or library")
					}
				}
			})
		}
	}
}

func TestRONLibraryRecycleCountsBeforeDraws(t *testing.T) {
	g, a, b := ronGame()
	for i, p := range []*mage.BasePlayer{a, b} {
		fillRONLibrary(t, p, 10)
		for range i + 1 {
			c := mage.NewLand("Hand")
			c.SetOwner(p.PlayerID())
			p.AddToHand(c)
		}
	}
	changed := false
	watcher := mage.NewTriggered(core.EvtCardDrawn, false).SetCondition(func(e *core.GameEvent, _ mage.GameReader, _, _ uuid.UUID) bool {
		if e.PlayerID == a.PlayerID() && !changed {
			changed = true
			for range 3 {
				c := mage.NewLand("Added during draw")
				c.SetOwner(b.PlayerID())
				b.AddToHand(c)
			}
		}
		return false
	})
	g.PutOnBattlefield(mage.NewArtifact("Observer", "", mage.WithAbility(watcher)), a.PlayerID())
	c := repositoryCard(t, "LEG", "Winds of Change")
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if !changed || len(a.Hand()) != 1 || len(b.Hand()) != 5 || len(a.Library()) != 10 || len(b.Library()) != 10 {
		t.Fatal("later hand changes replaced saved shuffle counts")
	}
}

func TestRONLibraryRecycleContexts(t *testing.T) {
	body := `EachPlayerAction(AnyPlayer,ShuffleHandIntoLibrary),EachPlayerAction(AnyPlayer,DrawNumberCards(NumCardsShuffledIntoLibraryThisWay))`
	for _, change := range []string{"leave", "return", "phase", "control"} {
		for _, rules := range []string{
			`Activated(TapPermanent(ThisPermanent),ActionList([` + body + `]))`,
			`Activated(TapPermanent(ThisPermanent),Modal_ChooseOne([ActionList([` + body + `]),ActionList([GainLife(Integer(1))])]))`,
		} {
			t.Run(change+rules, func(t *testing.T) {
				g, a, b := ronGame()
				first := permanentScopeFactory(t, "Artifact", rules)()
				first.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(first, a.PlayerID())
				for i, player := range []*mage.BasePlayer{a, b} {
					for range i + 1 {
						c := mage.NewLand("Hand")
						c.SetOwner(player.PlayerID())
						player.AddToHand(c)
					}
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				switch change {
				case "leave", "return":
					g.RemoveFromBattlefield(p)
					if change == "return" {
						g.PutOnBattlefield(first, b.PlayerID())
					}
				case "phase":
					g.PhaseOut(p.ID())
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
				clone := g.Clone()
				clone.ResolveStack()
				g.ResolveStack()
				if len(a.Hand()) != 1 || len(b.Hand()) != 2 || len(a.Library()) != 0 || len(b.Library()) != 0 || len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 {
					t.Fatal("source or copy changed counts")
				}
			})
		}
	}
}
