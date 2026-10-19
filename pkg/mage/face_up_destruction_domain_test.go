package mage

import (
	"reflect"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestFaceUpDestructionDomain(t *testing.T) {
	want := ReplacementDamageActions | ReplacementTapActions | ReplacementDamageAssignmentActions
	if got, known := InspectReplacementActionDomains(&faceUpActionReplacement{}); !known || got != want {
		t.Fatalf("face-up domain %v known %v, want %v", got, known, want)
	}
	for _, cycle := range []bool{false, true} {
		for _, state := range []string{"down", "up", "phase", "phase cycle", "return", "shield", "shield no regen"} {
			t.Run(state+map[bool]string{false: "/persistent", true: "/cycle"}[cycle], func(t *testing.T) {
				g, a, _, p := faceUpReplacementGame(t, true)
				visit := PermanentVisit{p.ID(), p.incarnationID}
				if cycle {
					g.effects.cycleReplacements = g.effects.replacements
					g.effects.replacements = nil
				}
				want := DestructionToGraveyard
				noRegen := state == "shield no regen"
				switch state {
				case "up":
					g.turnFaceUp(p)
				case "phase":
					g.PhaseOut(p.ID())
					want = DestructionNoMove
				case "phase cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "return":
					card := p.Card
					g.BouncePermanentToHand(p)
					a.RemoveFromHand(card.ID())
					p = g.PutOnBattlefield(card, a.PlayerID())
					want = DestructionNoMove
				case "shield", "shield no regen":
					g.AddRegenerationShield(p.ID())
					if !noRegen {
						want = DestructionNoMove
					}
				}
				p.Damage = 1
				before, random := g.Clone(), g.random.Clone()
				faceDown, tapped := p.FaceDown, p.Tapped
				for _, game := range []*Game{g, g.Clone()} {
					for range 2 {
						if got, known := game.InspectDestructionDestinations(visit, noRegen); !known || got != want {
							t.Fatalf("destinations %v known %v, want %v", got, known, want)
						}
						if proceeds, known := game.InspectDestructionProposal(visit, noRegen); !known || proceeds != (want == DestructionToGraveyard) {
							t.Fatalf("proposal proceeds %v known %v", proceeds, known)
						}
					}
				}
				after := g.Clone()
				if !reflect.DeepEqual(before.effects.replacements, after.effects.replacements) || !reflect.DeepEqual(before.effects.cycleReplacements, after.effects.cycleReplacements) || !reflect.DeepEqual(random, g.random) || p.FaceDown != faceDown || p.Tapped != tapped || p.Damage != 1 || g.StackSize() != 0 {
					t.Fatal("inspection changed state")
				}
				if state == "return" {
					visit = PermanentVisit{p.ID(), p.incarnationID}
					if got, known := g.InspectDestructionDestinations(visit, false); !known || got != DestructionToGraveyard {
						t.Fatal("stale replacement obscures new visit")
					}
				}
				for _, game := range []*Game{g.Clone(), g} {
					if state == "phase" {
						continue
					}
					game.destroyPermanents([]*Permanent{game.FindPermanent(p.ID())}, noRegen)
					if state == "shield" {
						live := game.FindPermanent(p.ID())
						if live == nil || live.FaceDown || !live.Tapped || live.Damage != 0 || game.HasRegenerationShield(p.ID()) {
							t.Fatal("native regeneration must tap and turn up only after inspection")
						}
					} else if game.FindPermanent(p.ID()) != nil || len(game.GetPlayer(a.PlayerID()).Graveyard()) != 1 {
						t.Fatal("native destruction disagrees")
					}
				}
			})
		}
	}
}

type faceUpDomainImpostor struct{ *faceUpActionReplacement }

func (*faceUpDomainImpostor) Matches(Action, GameReader) bool { panic("opaque match") }

func TestFaceUpDestructionDomainUnknown(t *testing.T) {
	g, _, _, p := faceUpReplacementGame(t, true)
	visit := PermanentVisit{p.ID(), p.incarnationID}
	var absent *faceUpActionReplacement
	for _, r := range []ReplacementEffect{nil, absent, &faceUpDomainImpostor{&faceUpActionReplacement{}}, &proposalTrap{}} {
		if _, known := InspectReplacementActionDomains(r); known {
			t.Fatalf("%T supplied an unchecked proof", r)
		}
		for _, cycle := range []bool{false, true} {
			game := g.Clone()
			if cycle {
				game.effects.cycleReplacements = append(game.effects.cycleReplacements, r)
			} else {
				game.effects.replacements = append(game.effects.replacements, r)
			}
			if _, known := game.InspectDestructionDestinations(visit, false); known {
				t.Fatalf("%T hidden by face-up proof", r)
			}
			if _, known := game.InspectDestructionProposal(visit, false); known {
				t.Fatalf("%T hidden by face-up proof", r)
			}
		}
	}
	if !g.FindPermanent(p.ID()).FaceDown || p.HasKeyword(core.CantRegenerate) {
		t.Fatal("inspection changed characteristics")
	}
}
