package mage

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWholeHandDiscardInstructionDeclaration(t *testing.T) {
	for _, declaration := range []WholeHandDiscardInstruction{{Subject: You}, {Subjects: EachPlayer}} {
		got, ok := InspectWholeHandDiscardInstruction(declaration.Effect())
		if !ok || got != declaration {
			t.Fatal("lost declaration")
		}
	}
	for _, bad := range []WholeHandDiscardInstruction{{}, {Subject: You, Subjects: EachPlayer}} {
		if bad.Effect() != nil {
			t.Fatal("invalid subjects accepted")
		}
	}
	var absentPlayer *lifeInspectionPlayerTrap
	var absentPlayers *opponentsRef
	for _, bad := range []WholeHandDiscardInstruction{{Subject: absentPlayer}, {Subjects: absentPlayers}} {
		if bad.Effect() != nil {
			t.Fatal("typed-nil operand accepted")
		}
	}
	var singular *discardEntireHandEffect
	var plural *discardEachEffect
	for _, bad := range []Effect{nil, singular, plural, DiscardEntireHand(nil), DiscardEach(nil),
		DiscardEntireHand(You, DiscardAmountInto(BindNumber("count"))),
		DiscardEntireHand(You, DiscardAmountInto(nil)),
		DiscardEach(EachPlayer, DiscardTotalInto(BindNumber("count"))),
		DiscardEach(EachPlayer, DiscardTotalInto(nil)),
		DiscardMatchingHand(You, CardFilter{}), Sequence(DiscardEach(EachPlayer)), destructionInspectionTrap{},
	} {
		if _, ok := InspectWholeHandDiscardInstruction(bad); ok {
			t.Fatal("non-native or result declaration accepted")
		}
	}
	for _, declaration := range []WholeHandDiscardInstruction{{Subject: lifeInspectionPlayerTrap{}}, {Subjects: collectionDamagePlayerTrap{}}} {
		if _, ok := InspectWholeHandDiscardInstruction(declaration.Effect()); !ok {
			t.Fatal("inspector evaluated operands")
		}
	}
}

func TestWholeHandDiscardInstructionQuery(t *testing.T) {
	for _, state := range []string{"only", "single", "active", "prefix", "suffix", "empty", "opponents", "target", "foreign", "illegal player", "all illegal", "opaque player", "opaque players", "opaque suffix", "opaque illegal", "foreign illegal", "phase", "return"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := targetQueryFixture(t)
			declaration := WholeHandDiscardInstruction{Subjects: EachPlayer}
			player := DeclarePlayerTarget("player")
			want, known := false, true
			switch state {
			case "single":
				declaration = WholeHandDiscardInstruction{Subject: You}
			case "active":
				declaration = WholeHandDiscardInstruction{Subject: ActivePlayer}
			case "empty":
				declaration.Subjects = noDamagePlayers{}
			case "opponents":
				declaration.Subjects = OpponentsOf(ActivePlayer)
			case "target", "illegal player", "all illegal":
				declaration = WholeHandDiscardInstruction{Subject: player}
				obj.TargetSpecs = append(obj.TargetSpecs, player.targetSpec())
				obj.Targets = append(obj.Targets, obj.Controller)
				if state != "target" {
					obj.Targets[1] = uuid.New()
				}
			case "foreign", "foreign illegal":
				declaration = WholeHandDiscardInstruction{Subject: player}
				known = false
			case "opaque player", "opaque illegal":
				declaration = WholeHandDiscardInstruction{Subject: lifeInspectionPlayerTrap{}}
				known = false
			case "opaque players":
				declaration.Subjects = collectionDamagePlayerTrap{}
				known = false
			case "phase":
				g.PhaseOut(p.ID())
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, p.ControllerID())
			}
			discard := declaration.Effect()
			obj.Effects = []Effect{discard}
			switch state {
			case "prefix":
				obj.Effects = []Effect{Sequence(discard, Destroy(target))}
				known = false
			case "empty", "illegal player":
				obj.Effects = []Effect{Sequence(discard, Destroy(target))}
				want = true
			case "suffix":
				obj.Effects = []Effect{Sequence(Destroy(target), discard)}
				want = true
			case "opaque suffix":
				obj.Effects = []Effect{Sequence(Destroy(target), discard, destructionInspectionTrap{})}
				known = false
			case "all illegal", "opaque illegal", "foreign illegal":
				p.GrantBaseAttr(core.Shroud)
			case "phase", "return":
				obj.Effects = []Effect{Sequence(Destroy(target), discard)}
			}
			card := NewArtifact("Hand", "")
			card.SetOwner(obj.Controller)
			g.GetPlayer(obj.Controller).AddToHand(card)
			before := g.Clone()
			for _, game := range []*Game{g, g.Clone()} {
				for range 2 {
					certain, k := game.InspectSpellDestruction(game.StackPeek(), []PermanentVisit{{p.ID(), p.incarnationID}})
					if certain != want || k != known {
						t.Fatalf("query=%v/%v want=%v/%v", certain, k, want, known)
					}
				}
			}
			if !reflect.DeepEqual(before.random, g.random) || len(g.GetPlayer(obj.Controller).Hand()) != 1 || g.StackSize() != before.StackSize() || len(g.resolution.playerBindings) != 0 {
				t.Fatal("inspection changed state")
			}
		})
	}
}

func TestWholeHandDiscardInstructionSourceRead(t *testing.T) {
	for _, declaration := range []WholeHandDiscardInstruction{{Subject: ControllerOf(This)}, {Subjects: OpponentsOf(ControllerOf(This))}} {
		if !effectReadsSourcePermanent(declaration.Effect()) {
			t.Fatal("discard lost its source controller read")
		}
	}
	for _, declaration := range []WholeHandDiscardInstruction{{Subject: You}, {Subjects: EachPlayer}} {
		if effectReadsSourcePermanent(declaration.Effect()) {
			t.Fatal("player-only discard gained a source read")
		}
	}
}

func TestWholeHandDiscardInstructionExecution(t *testing.T) {
	for _, plural := range []bool{false, true} {
		for _, replacement := range []string{"none", "library", "prevented"} {
			t.Run(fmt.Sprintf("%v/%s", plural, replacement), func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				declaration := WholeHandDiscardInstruction{Subject: You}
				if plural {
					declaration = WholeHandDiscardInstruction{Subjects: EachPlayer}
				}
				for _, player := range []*BasePlayer{a, b} {
					for range 2 {
						card := NewArtifact("Hand", "")
						card.SetOwner(player.PlayerID())
						player.AddToHand(card)
					}
				}
				if replacement == "library" || replacement == "prevented" {
					g.AddReplacementEffect(&allDiscardToLibraryReplacement{})
				}
				if replacement == "prevented" {
					g.AddReplacementEffect(&discardToHandReplacement{})
				}
				seen := 0
				watcher := NewTriggered(core.EvtDiscard, false).SetCondition(func(_ *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
					seen++
					if len(a.Hand()) != 0 || plural && len(b.Hand()) != 0 {
						t.Error("event saw a partial discard")
					}
					return false
				})
				g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(watcher)), a.PlayerID())
				before := g.Clone()
				if err := ApplyEffect(g, Sequence(declaration.Effect(), GainLifeFor(You, Fixed(1))), uuid.Nil, a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if a.Life() != 21 || len(before.GetPlayer(a.PlayerID()).Hand()) != 2 {
					t.Fatal("continuation or clone isolation failed")
				}
				expected := 2
				if plural {
					expected = 4
				}
				if replacement == "prevented" {
					expected = 0
				}
				if seen != expected {
					t.Fatalf("discard events=%d want=%d", seen, expected)
				}
				for i, player := range []*BasePlayer{a, b} {
					selected := i == 0 || plural
					wantHand, wantGrave, wantLibrary := 2, 0, 0
					if selected && replacement != "prevented" {
						wantHand = 0
						if replacement == "library" {
							wantLibrary = 2
						} else {
							wantGrave = 2
						}
					}
					if len(player.Hand()) != wantHand || len(player.Graveyard()) != wantGrave || len(player.Library()) != wantLibrary {
						t.Fatal("discard destination or player changed")
					}
				}
			})
		}
	}
}
