package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDrawLifeNativeDomain(t *testing.T) {
	for _, tc := range []struct {
		name string
		rule ReplacementEffect
		want ReplacementActionDomains
	}{
		{"static draw", &scopedDrawReplacement{}, ReplacementDrawActions},
		{"future draw", &futureDrawReplacement{}, ReplacementDrawActions},
		{"life gain", &lifeGainDrawReplacement{}, ReplacementLifeGainActions},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, known := InspectReplacementActionDomains(tc.rule); !known || got != tc.want {
				t.Fatalf("domain %v known %v, want %v", got, known, tc.want)
			}
			for _, cycle := range []bool{false, true} {
				for _, state := range []string{"live", "phase", "phase cycle", "return", "regen", "no regen", "indestructible"} {
					t.Run(state+map[bool]string{false: "/persistent", true: "/cycle"}[cycle], func(t *testing.T) {
						a, b := NewBasePlayer("A"), NewBasePlayer("B")
						g := NewGame(a, b)
						p := g.PutOnBattlefield(NewCreature("Subject", "", 3, 3), a.PlayerID())
						visit := PermanentVisit{p.ID(), p.incarnationID}
						want := DestructionToGraveyard
						noRegen := state == "no regen"
						switch state {
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
						case "regen", "no regen":
							g.AddRegenerationShield(p.ID())
							if !noRegen {
								want = DestructionNoMove
							}
						case "indestructible":
							p.GrantBaseAttr(core.Indestructible)
							want = DestructionNoMove
						}
						if cycle {
							g.effects.cycleReplacements = append(g.effects.cycleReplacements, tc.rule.Clone())
						} else {
							g.effects.replacements = append(g.effects.replacements, tc.rule.Clone())
						}
						p.Damage = 1
						before, random := g.Clone(), g.random.Clone()
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
						if !reflect.DeepEqual(before.effects.replacements, after.effects.replacements) || !reflect.DeepEqual(before.effects.cycleReplacements, after.effects.cycleReplacements) || !reflect.DeepEqual(random, g.random) || p.Tapped || p.Damage != 1 || a.Life() != 20 || b.Life() != 20 {
							t.Fatal("inspection changed state")
						}
						if state == "return" {
							if got, known := g.InspectDestructionDestinations(PermanentVisit{p.ID(), p.incarnationID}, false); !known || got != DestructionToGraveyard {
								t.Fatal("new visit obscured")
							}
						}
					})
				}
			}
		})
	}
}

type scopedDrawDomainImpostor struct{ *scopedDrawReplacement }
type futureDrawDomainImpostor struct{ *futureDrawReplacement }
type lifeGainDomainImpostor struct{ *lifeGainDrawReplacement }

func TestDrawLifeNativeDomainNoCallbacks(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Subject", "", 3, 3), a.PlayerID())
	visit := PermanentVisit{p.ID(), p.incarnationID}
	for _, r := range []ReplacementEffect{
		&scopedDrawReplacement{rule: drawReplacementRule{body: drawLifeBodyTrap{}}},
		&futureDrawReplacement{body: drawLifeBodyTrap{}},
		&lifeGainDrawReplacement{subject: drawLifePlayerTrap{}},
	} {
		for _, cycle := range []bool{false, true} {
			g.effects.replacements, g.effects.cycleReplacements = nil, nil
			if cycle {
				g.effects.cycleReplacements = []ReplacementEffect{r}
			} else {
				g.effects.replacements = []ReplacementEffect{r}
			}
			if got, known := g.InspectDestructionDestinations(visit, false); !known || got != DestructionToGraveyard {
				t.Fatal("unrelated body obscures destination")
			}
			if proceeds, known := g.InspectDestructionProposal(visit, false); !known || !proceeds {
				t.Fatal("unrelated body obscures proposal")
			}
		}
	}
	var scoped *scopedDrawReplacement
	var future *futureDrawReplacement
	var gain *lifeGainDrawReplacement
	for _, r := range []ReplacementEffect{nil, scoped, future, gain, &scopedDrawDomainImpostor{}, &futureDrawDomainImpostor{}, &lifeGainDomainImpostor{}, &proposalTrap{}, &drawReplacementEffect{}} {
		if _, known := InspectReplacementActionDomains(r); known {
			t.Fatalf("%T supplies unchecked proof", r)
		}
		for _, cycle := range []bool{false, true} {
			game := g.Clone()
			if cycle {
				game.effects.cycleReplacements = append(game.effects.cycleReplacements, r)
			} else {
				game.effects.replacements = append(game.effects.replacements, r)
			}
			if _, known := game.InspectDestructionDestinations(visit, false); known {
				t.Fatalf("%T hidden by known domain", r)
			}
			if _, known := game.InspectDestructionProposal(visit, false); known {
				t.Fatalf("%T hidden by known domain", r)
			}
		}
	}
}

type drawLifeBodyTrap struct{ Effect }

func (drawLifeBodyTrap) Apply(*EffectContext) error { panic("replacement body called") }

type drawLifePlayerTrap struct{ PlayerRef }

func (drawLifePlayerTrap) resolvePlayer(*EffectContext) (uuid.UUID, bool) {
	panic("player operand called")
}
