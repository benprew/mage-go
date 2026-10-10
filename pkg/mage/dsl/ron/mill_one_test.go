package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONMillOneChains(t *testing.T) {
	for _, copies := range []int{1, 2} {
		for _, hand := range []int{0, 1, 3} {
			for _, playerIndex := range []int{0, 1} {
				t.Run(fmt.Sprintf("%d-chains/%d-hand/player-%d", copies, hand, playerIndex), func(t *testing.T) {
					g, a, b := ronGame()
					g.SetTurn(3)
					g.SetStep(core.Upkeep)
					for range copies {
						g.PutOnBattlefield(repositoryCard(t, "LEG", "Chains of Mephistopheles"), a.PlayerID())
					}
					player, other := a, b
					if playerIndex == 1 {
						player, other = b, a
					}
					fillRONLibrary(t, player, 8)
					originalTop := player.Library()[0].ID()
					for range hand {
						player.AddToHand(mage.NewLand("Discard"))
					}
					g.PlayerDrawCard(player)
					wantHand, wantGrave, wantLibrary := 0, hand+1, 7
					if hand >= copies {
						wantHand, wantGrave = hand-copies+1, copies
					}
					if len(player.Hand()) != wantHand || len(player.Graveyard()) != wantGrave || len(player.Library()) != wantLibrary {
						t.Fatalf("hand/graveyard/library=%d/%d/%d want=%d/%d/%d", len(player.Hand()), len(player.Graveyard()), len(player.Library()), wantHand, wantGrave, wantLibrary)
					}
					if hand < copies && player.Graveyard()[wantGrave-1].ID() != originalTop {
						t.Fatal("did not mill the original top")
					}
					if hand >= copies && player.Hand()[wantHand-1].ID() != originalTop {
						t.Fatal("replacement draw lost the top")
					}
					if len(other.Graveyard()) != 0 || len(other.Hand()) != 0 || g.StackSize() != 0 {
						t.Fatal("wrong player or stack timing")
					}
				})
			}
		}
	}
}

func TestRONMillOneChainsDrawStep(t *testing.T) {
	g, a, b := ronGame()
	g.SetTurn(3)
	g.PutOnBattlefield(repositoryCard(t, "LEG", "Chains of Mephistopheles"), b.PlayerID())
	fillRONLibrary(t, a, 6)
	fillRONLibrary(t, b, 6)
	g.SetStep(core.Draw)
	g.PlayerDrawCard(a)
	if len(a.Hand()) != 1 || len(a.Graveyard()) != 0 {
		t.Fatal("first effect draw in draw step must be exempt")
	}
	g.PlayerDrawCard(a)
	if len(a.Hand()) != 1 || len(a.Graveyard()) != 1 {
		t.Fatal("second draw must discard then draw")
	}
	g.PlayerDrawCard(b)
	if len(b.Hand()) != 0 || len(b.Graveyard()) != 1 {
		t.Fatal("nonactive player's draw must mill")
	}
	g.SetStep(core.PrecombatMain)
	g.SetStep(core.Draw)
	g.PlayerDrawCard(a)
	if len(a.Hand()) != 2 || len(a.Graveyard()) != 1 {
		t.Fatal("new draw step must reset the exemption")
	}
}

type millOnePlayer struct {
	*mage.BasePlayer
	mode    string
	choices int
	offers  int
}

func (p *millOnePlayer) ChooseCardsFromHand(_ int, _ string, _ mage.GameReader) []mage.Card {
	p.choices++
	c := p.Hand()[0]
	switch p.mode {
	case "invalid":
		return []mage.Card{mage.NewLand("Foreign")}
	case "returned":
		p.RemoveFromHand(c.ID())
		p.AddToHand(c)
	}
	return []mage.Card{c}
}
func (p *millOnePlayer) ChooseMayAbility(_ string) bool { p.offers++; return true }

func TestRONMillOneChainsDiscardResults(t *testing.T) {
	for _, mode := range []string{"normal", "invalid", "returned", "library"} {
		t.Run(mode, func(t *testing.T) {
			a := &millOnePlayer{BasePlayer: mage.NewBasePlayer("A"), mode: mode}
			g := mage.NewGame(a, mage.NewBasePlayer("B"))
			g.SetTurn(3)
			g.SetStep(core.Upkeep)
			g.PutOnBattlefield(repositoryCard(t, "LEG", "Chains of Mephistopheles"), a.PlayerID())
			if mode == "library" {
				g.PutOnBattlefield(mage.NewArtifact("Effect discard replacement", "", mage.WithStaticAbility(mage.DiscardToLibraryReplacement())), a.PlayerID())
			}
			fillRONLibrary(t, a.BasePlayer, 3)
			top := a.Library()[0].ID()
			discard := mage.NewLand("Discard")
			discard.SetOwner(a.PlayerID())
			a.AddToHand(discard)
			g.PlayerDrawCard(a)
			if len(a.Hand()) != 1 || a.choices != 1 || a.Life() != 20 {
				t.Fatal("discard timing or mandatory choice")
			}
			if mode == "library" {
				if a.Hand()[0].ID() != discard.ID() || len(a.Graveyard()) != 0 || len(a.Library()) != 3 || a.offers != 1 {
					t.Fatal("changed discard destination must still draw")
				}
			} else {
				grave := discard.ID()
				if mode != "normal" {
					grave = top
				}
				if len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != grave || len(a.Library()) != 2 || a.offers != 0 {
					t.Fatal("actual discard result did not select the correct branch")
				}
				if mode != "normal" && a.Hand()[0].ID() != discard.ID() {
					t.Fatal("invalid visit was discarded")
				}
			}
		})
	}
}

func TestRONMillOneChainsLifecycle(t *testing.T) {
	for _, state := range []string{"live", "control", "phase", "phase-cycle", "depart", "return", "remove-ability", "clone"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			g.SetTurn(3)
			g.SetStep(core.Upkeep)
			card := repositoryCard(t, "LEG", "Chains of Mephistopheles")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			fillRONLibrary(t, b, 4)
			switch state {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phase":
				g.PhaseOut(source.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "depart", "return":
				g.BouncePermanentToHand(source)
				if state == "return" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "remove-ability":
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
			}
			run := g
			if state == "clone" {
				run = g.Clone()
			}
			run.PlayerDrawCard(run.GetPlayer(b.PlayerID()))
			wantHand, wantGrave := 0, 1
			if state == "phase" || state == "depart" || state == "remove-ability" {
				wantHand, wantGrave = 1, 0
			}
			player := run.GetPlayer(b.PlayerID())
			if len(player.Hand()) != wantHand || len(player.Graveyard()) != wantGrave || len(player.Library()) != 3 {
				t.Fatal("source lifecycle or player binding")
			}
			if state == "clone" && (len(b.Library()) != 4 || len(b.Graveyard()) != 0) {
				t.Fatal("clone changed original")
			}
		})
	}
}

func TestRONMillOneScopesAndTiming(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(1)]),Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,MillACard),MillACard])))`)
	g, a, b := ronGame()
	g.SetTurn(3)
	fillRONLibrary(t, a, 4)
	fillRONLibrary(t, b, 4)
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err == nil {
		t.Fatal("unpaid activation succeeded")
	}
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{uuid.New()}); err == nil || a.ManaPool().TotalMana() != 1 {
		t.Fatal("illegal target spent mana")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if len(a.Library()) != 4 || len(b.Library()) != 4 {
		t.Fatal("mill ran at announcement")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	g.ResolveStack()
	if len(a.Graveyard()) != 1 || len(b.Graveyard()) != 3 {
		t.Fatal("lexical player or copy controller")
	}
	if len(clone.GetPlayer(b.PlayerID()).Library()) != 4 {
		t.Fatal("clone mutated")
	}
	clone.ResolveStack()
	if len(clone.GetPlayer(a.PlayerID()).Graveyard()) != 1 || len(clone.GetPlayer(b.PlayerID()).Graveyard()) != 3 {
		t.Fatal("clone lost bindings")
	}
}

func TestRONMillOneInvalid(t *testing.T) {
	for _, body := range []string{
		`MillACard()`, `MillACard(Integer(1))`, `PlayerAction(Ref_TargetPlayer,MillACard)`, `PlayerAction(WouldDrawACard_ThatPlayer,MillACard)`, `PlayerAction(Ref_TargetPermanent,MillACard)`,
		`MayCost(MillACard),If(CostWasPaid,[GainLife(Integer(1))])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([`+body+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", body)
		}
	}
}

func TestRONMillOneChainsReplacementOrder(t *testing.T) {
	for _, index := range []int{0, 1} {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			a := &counterReplacementChooser{BasePlayer: mage.NewBasePlayer("A"), index: index}
			b := &counterReplacementChooser{BasePlayer: mage.NewBasePlayer("B"), index: 1 - index}
			g := mage.NewGame(a, b)
			g.SetTurn(3)
			g.SetStep(core.Upkeep)
			g.PutOnBattlefield(repositoryCard(t, "LEG", "Chains of Mephistopheles"), b.PlayerID())
			other := permanentScopeFactory(t, "Enchantment", `ReplaceWouldDraw(APlayerWouldDrawExceptFirstDrawStepDraw(AnyPlayer),[PlayerAction(WouldDrawACard_ThatPlayer,GainLife(Integer(5)))])`)
			g.PutOnBattlefield(other(), b.PlayerID())
			fillRONLibrary(t, a.BasePlayer, 3)
			a.AddToHand(mage.NewLand("Discard"))
			g.PlayerDrawCard(a)
			wantHand, wantGrave := index, 1-index
			if len(a.Hand()) != wantHand || len(a.Graveyard()) != wantGrave || len(a.Library()) != 3 || a.Life() != 25 || a.choices != 1 || b.choices != 0 {
				t.Fatalf("replacement order: hand/graveyard=%d/%d choices=%d/%d life=%d", len(a.Hand()), len(a.Graveyard()), a.choices, b.choices, a.Life())
			}
		})
	}
}

func TestRONMillOneEmptyAndModifiers(t *testing.T) {
	for _, count := range []int{0, 1, 3} {
		for _, modified := range []int{0, 2} {
			t.Run(fmt.Sprintf("%d/%d", count, modified), func(t *testing.T) {
				g, a, b := ronGame()
				g.SetTurn(3)
				g.SetStep(core.Upkeep)
				source := g.PutOnBattlefield(repositoryCard(t, "LEG", "Chains of Mephistopheles"), a.PlayerID())
				fillRONLibrary(t, b, count)
				g.AddMillReplacement(source.ID(), func(_ *mage.Game, _ uuid.UUID, n int) int { return n * modified })
				g.PlayerDrawCard(b)
				if len(b.Hand()) != 0 || len(b.Graveyard()) != min(count, modified) || len(b.Library()) != max(0, count-modified) {
					t.Fatal("mill amount or empty library")
				}
				g.CheckStateBasedActions()
				if g.IsGameOver() {
					t.Fatal("a replaced draw must not lose to an empty library")
				}
			})
		}
	}
}

func TestRONMillOneDelayedSource(t *testing.T) {
	for _, state := range []string{"live", "depart", "return", "phase"} {
		t.Run(state, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PlayerAction(ControllerOfPermanent(ThisPermanent),MillACard)]))]))`)
			g, a, b := ronGame()
			g.SetTurn(3)
			card := factory()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(source.ID())
			fillRONLibrary(t, a, 3)
			fillRONLibrary(t, b, 3)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			switch state {
			case "depart", "return":
				g.DestroyPermanent(source)
				if state == "return" {
					a.RemoveFromGraveyard(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phase":
				g.PhaseOut(source.ID())
			}
			for range 2 {
				refs := g.WaitingDelayedPermanentSources()
				if len(refs) != 1 || !refs[0].SameObject(saved) || len(a.Library()) != 3 || len(b.Library()) != 3 {
					t.Fatal("waiting query lost exact source or moved cards")
				}
			}
			clone := g.Clone()
			clone.RunStepWithPriority(core.EndStep)
			clone.ResolveStack()
			if len(clone.GetPlayer(a.PlayerID()).Library()) != 2 || len(clone.GetPlayer(b.PlayerID()).Library()) != 3 || len(a.Library()) != 3 {
				t.Fatal("delayed mill lost controller LKI or clone isolation")
			}
		})
	}
}
