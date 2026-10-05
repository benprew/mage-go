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

func TestRONEachPlayerWheel(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, active := range []int{0, 1} {
			t.Run(fmt.Sprintf("%s/%d", set, active), func(t *testing.T) {
				g, a, b := ronGame()
				g.SetActivePlayerIndex(active)
				caster := a
				if active == 1 {
					caster = b
				}
				fillRONLibrary(t, a, 20)
				fillRONLibrary(t, b, 20)
				card := repositoryCard(t, set, "Wheel of Fortune")
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(caster.PlayerID())
				caster.AddToHand(card)
				caster.ManaPool().Add(core.Red, 1)
				if err := g.CastSpellByID(caster.PlayerID(), card.ID(), nil, 0); err == nil {
					t.Fatal("unpaid Wheel accepted")
				}
				if len(caster.Hand()) != 1 || caster.ManaPool().TotalMana() != 1 || g.StackSize() != 0 || len(a.Library()) != 20 || len(b.Library()) != 20 {
					t.Fatal("failed casting changed state")
				}
				caster.ManaPool().Add(core.Colorless, 2)
				if err := g.CastSpellByID(caster.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				for i, player := range []*mage.BasePlayer{a, b} {
					for range i + 1 {
						c := repositoryCard(t, "LEB", "Forest")
						c.SetOwner(player.PlayerID())
						player.AddToHand(c)
					}
				}
				clone := g.Clone()
				clone.ResolveStack()
				if len(a.Hand()) != 1 || len(b.Hand()) != 2 || len(a.Library()) != 20 {
					t.Fatal("clone changed pending original")
				}
				discards := 0
				var draws []uuid.UUID
				watcher := mage.NewTriggered(core.EvtDiscard, false).SetCondition(func(_ *core.GameEvent, _ mage.GameReader, _, _ uuid.UUID) bool {
					discards++
					if len(a.Hand()) != 0 || len(b.Hand()) != 0 || len(a.Library()) != 20 || len(b.Library()) != 20 {
						t.Error("discard event saw partial hands or early draws")
					}
					return false
				})
				g.PutOnBattlefield(mage.NewArtifact("Discard observer", "", mage.WithAbility(watcher)), a.PlayerID())
				drawWatcher := mage.NewTriggered(core.EvtCardDrawn, false).SetCondition(func(event *core.GameEvent, _ mage.GameReader, _, _ uuid.UUID) bool {
					draws = append(draws, event.PlayerID)
					return false
				})
				g.PutOnBattlefield(mage.NewArtifact("Draw observer", "", mage.WithAbility(drawWatcher)), a.PlayerID())
				g.ResolveStack()
				if discards != 3 || len(draws) != 14 || len(a.Hand()) != 7 || len(b.Hand()) != 7 || len(a.Library()) != 13 || len(b.Library()) != 13 {
					t.Fatalf("events or final zones: discards=%d draws=%d hands=%d/%d", discards, len(draws), len(a.Hand()), len(b.Hand()))
				}
				for i, id := range draws {
					want := caster.PlayerID()
					if i >= 7 {
						want = a.PlayerID()
						if active == 0 {
							want = b.PlayerID()
						}
					}
					if id != want {
						t.Fatal("draws did not use active-player order")
					}
				}
				wantA, wantB := 1, 2
				if active == 0 {
					wantA++
				} else {
					wantB++
				}
				if len(a.Graveyard()) != wantA || len(b.Graveyard()) != wantB || caster.ManaPool().TotalMana() != 0 {
					t.Fatal("wrong discarded cards, spell destination, or payment")
				}
			})
		}
	}
}

func TestRONEachPlayerInvalid(t *testing.T) {
	for _, action := range []string{
		`EachPlayerAction`, `EachPlayerAction()`, `EachPlayerAction(AnyPlayer)`,
		`EachPlayerAction(AnyPlayer,DiscardHand,DiscardHand)`, `EachPlayerAction(AnyPlayer(),DiscardHand)`,
		`EachPlayerAction(SinglePlayer(You),DiscardHand)`, `EachPlayerAction(ActionPlayer,DiscardHand)`,
		`EachPlayerAction(AnyPlayer,DiscardHand())`, `EachPlayerAction(AnyPlayer,ActionList([DiscardHand]))`,
		`EachPlayerAction(AnyPlayer,DrawNumberCards)`, `EachPlayerAction(AnyPlayer,DrawNumberCards())`,
		`EachPlayerAction(AnyPlayer,DrawNumberCards(7))`, `EachPlayerAction(AnyPlayer,DrawNumberCards(Integer(-1)))`,
		`EachPlayerAction(AnyPlayer,DrawNumberCards(Integer(1,2)))`, `EachPlayerAction(AnyPlayer,DrawNumberCards(ValueX))`,
		`EachPlayerAction(AnyPlayer,DrawNumberCards(NumCardsShuffledIntoLibraryThisWay))`,
		`EachPlayerAction(AnyPlayer,PlayerAction(ActionPlayer,DrawACard))`,
		`EachPlayerAction(AnyPlayer,ShuffleHandAndGraveyardIntoLibrary())`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([`+action+`]))])`), "invalid-each.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
}

func TestRONEachPlayerInventory(t *testing.T) {
	src := `Card(Name:"All",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([EachPlayerAction(AnyPlayer,DiscardHand),EachPlayerAction(AnyPlayer,DrawNumberCards(Integer(7)))]))])`
	report, err := InspectFS(fstest.MapFS{"all.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, signature := range report.Signatures {
		want := map[string]string{"AnyPlayer": "player_filter", "DiscardHand": "player_action", "DrawNumberCards": "player_action", "Integer": "value"}[signature.Name]
		if want != "" && signature.Role != want {
			t.Errorf("%s role = %s, want %s", signature.Name, signature.Role, want)
		}
	}
}

type ronPreventPlayerDraw struct{ player uuid.UUID }

func (r *ronPreventPlayerDraw) Matches(action mage.Action, _ mage.GameReader) bool {
	draw, ok := action.(*mage.DrawCardAction)
	return ok && draw.PlayerID() == r.player
}
func (*ronPreventPlayerDraw) Replace(mage.Action, *mage.Game) mage.Action { return nil }
func (*ronPreventPlayerDraw) IsActive(mage.GameReader) bool               { return true }
func (*ronPreventPlayerDraw) SourceID() uuid.UUID                         { return uuid.Nil }
func (*ronPreventPlayerDraw) GetDuration() core.Duration                  { return core.EndOfTurn }
func (r *ronPreventPlayerDraw) Clone() mage.ReplacementEffect             { cloned := *r; return &cloned }

func TestRONEachPlayerReplacements(t *testing.T) {
	a := &ronDiscardPlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	fillRONLibrary(t, a.BasePlayer, 10)
	fillRONLibrary(t, b, 10)
	for _, player := range []*mage.BasePlayer{a.BasePlayer, b} {
		c := repositoryCard(t, "LEB", "Forest")
		c.SetOwner(player.PlayerID())
		player.AddToHand(c)
	}
	g.PutOnBattlefield(mage.NewArtifact("Discard replacement", "", mage.WithStaticAbility(mage.DiscardToLibraryReplacement())), a.PlayerID())
	g.AddReplacementEffect(&ronPreventPlayerDraw{player: a.PlayerID()})
	card := repositoryCard(t, "LEB", "Wheel of Fortune")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Colorless, 2)
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	seen := 0
	watcher := mage.NewTriggered(core.EvtDiscard, false).SetCondition(func(_ *core.GameEvent, _ mage.GameReader, _, _ uuid.UUID) bool {
		seen++
		if len(a.Hand()) != 0 || len(b.Hand()) != 0 || len(a.Library()) != 11 || len(b.Graveyard()) != 1 {
			t.Error("discard event saw partial replacement outcomes")
		}
		return false
	})
	g.PutOnBattlefield(mage.NewArtifact("Observer", "", mage.WithAbility(watcher)), b.PlayerID())
	g.ResolveStack()
	if seen != 2 || len(a.Hand()) != 0 || len(a.Library()) != 11 || len(a.Graveyard()) != 1 || len(b.Hand()) != 7 || len(b.Library()) != 3 || len(b.Graveyard()) != 1 {
		t.Fatal("discard or draw replacements stopped later operations")
	}
}

func TestRONEachPlayerCopies(t *testing.T) {
	g, a, b := ronGame()
	fillRONLibrary(t, a, 20)
	fillRONLibrary(t, b, 20)
	card := repositoryCard(t, "LEB", "Wheel of Fortune")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Colorless, 2)
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), true); copied == nil || len(copied.Targets) != 0 {
		t.Fatal("copy failed or created targets")
	}
	g.ResolveTopOfStack()
	if len(a.Hand()) != 7 || len(b.Hand()) != 7 || len(a.Graveyard()) != 0 || len(b.Graveyard()) != 0 {
		t.Fatal("copy did not draw for both players")
	}
	g.ResolveTopOfStack()
	if len(a.Hand()) != 7 || len(b.Hand()) != 7 || len(a.Library()) != 6 || len(b.Library()) != 6 || len(a.Graveyard()) != 8 || len(b.Graveyard()) != 7 {
		t.Fatal("original did not discard the resolution-time hands")
	}
}

func TestRONEachPlayerContexts(t *testing.T) {
	for _, change := range []string{"none", "departed", "returned", "phased", "control"} {
		t.Run(change, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `Activated(TapPermanent(ThisPermanent),ActionList([EachPlayerAction(AnyPlayer,DrawNumberCards(Integer(1)))]))`)
			first, second := factory(), factory()
			if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
				t.Fatal(err)
			}
			if first.Abilities()[0] == second.Abilities()[0] {
				t.Fatal("shared ability instance")
			}
			g, a, b := ronGame()
			fillRONLibrary(t, a, 2)
			fillRONLibrary(t, b, 2)
			first.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(first, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if len(a.Hand()) != 0 || len(b.Hand()) != 0 {
				t.Fatal("draw ran during payment")
			}
			switch change {
			case "departed", "returned":
				g.DestroyPermanent(source)
				if change == "returned" {
					g.PutOnBattlefield(first, b.PlayerID())
				}
			case "phased":
				g.PhaseOut(source.ID())
			case "control":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ChangeController(mage.This, mage.You)), source.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			g.ResolveStack()
			if len(a.Hand()) != 1 || len(b.Hand()) != 1 {
				t.Fatal("source change affected all-player collection")
			}
		})
	}
	for _, rule := range []string{
		`TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([EachPlayerAction(AnyPlayer,DrawNumberCards(Integer(1)))]))`,
		`Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([EachPlayerAction(AnyPlayer,DrawNumberCards(Integer(1)))]))]))`,
	} {
		g, a, b := ronGame()
		fillRONLibrary(t, a, 2)
		fillRONLibrary(t, b, 2)
		card := permanentScopeFactory(t, "Artifact", rule)()
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		source := g.PutOnBattlefield(card, a.PlayerID())
		if strings.HasPrefix(rule, "Activated") {
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if len(a.Hand()) != 0 || len(b.Hand()) != 0 {
				t.Fatal("delayed action ran early")
			}
			g.RunStepWithPriority(core.EndStep)
		}
		g.ResolveStack()
		if len(a.Hand()) != 1 || len(b.Hand()) != 1 {
			t.Fatal("trigger context failed")
		}
	}
}

func TestRONEachPlayerZeroAndTargetScope(t *testing.T) {
	g, a, b := ronGame()
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([EachPlayerAction(AnyPlayer,DiscardHand),EachPlayerAction(AnyPlayer,DrawNumberCards(Integer(0))),PlayerAction(Ref_TargetPlayer,GainLife(Integer(2)))])))`)
	card := factory()
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 22 || len(a.Hand()) != 0 || len(b.Hand()) != 0 {
		t.Fatal("empty hands or zero draws stopped the action sequence or changed target scope")
	}
	nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([EachPlayerAction(AnyPlayer,DrawNumberCards(Integer(1))),PlayerAction(ActionPlayer,GainLife(Integer(1)))]))])`), "leak.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(nodes[0]); err == nil {
		t.Fatal("all-player action leaked a loop reference")
	}
}
