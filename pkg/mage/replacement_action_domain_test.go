package mage

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestReplacementActionDomain(t *testing.T) {
	damage := []ReplacementEffect{
		&preventionShieldReplacement{}, &sourcePreventionShieldReplacement{}, &halfDamagePreventionShieldReplacement{},
		&fogReplacement{}, &creatureCombatDamagePreventionReplacement{}, &combatSourcePreventionReplacement{},
		&playerDamageFromFlyingCreaturesPreventionReplacement{}, &forcefieldReplacement{}, &colorPreventionReplacement{},
		&sourcePreventionReplacement{}, &typePreventionReplacement{}, &reverseDamageReplacement{}, &bodyguardReplacement{},
		&playerDamageRedirectReplacement{}, &artifactDamageRedirectReplacement{}, &creatureDamageRedirectReplacement{},
		&attackerDamageRedirectReplacement{}, &damagePreventionRuleReplacement{}, &counterDamagePreventionReplacement{},
		&instructionDamagePrevention{}, &continuedDamageReplacement{}, &selectedSourceRedirectReplacement{},
		&finiteDamageRedirectReplacement{}, &staticDamagePreventionReplacement{}, &staticDamageRedirectReplacement{},
	}
	for _, r := range damage {
		t.Run(fmt.Sprintf("%T", r), func(t *testing.T) {
			domain, known := InspectReplacementActionDomains(r)
			if !known || domain != ReplacementDamageActions {
				t.Fatalf("%T: domain %v known %v", r, domain, known)
			}
			for _, cycle := range []bool{false, true} {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				p := g.PutOnBattlefield(NewCreature("Subject", "", 3, 3), a.PlayerID())
				if cycle {
					g.effects.cycleReplacements = []ReplacementEffect{r}
				} else {
					g.effects.replacements = []ReplacementEffect{r}
				}
				visit := PermanentVisit{p.ID(), p.incarnationID}
				for _, game := range []*Game{g, g.Clone()} {
					before := r.Clone()
					if proceeds, known := game.InspectDestructionProposal(visit, false); !known || !proceeds {
						t.Fatalf("%T blocks unrelated destruction", r)
					}
					game.AddRegenerationShield(p.ID())
					if proceeds, known := game.InspectDestructionProposal(visit, false); !known || proceeds {
						t.Fatalf("%T hides regeneration", r)
					}
					if proceeds, known := game.InspectDestructionProposal(visit, true); !known || !proceeds {
						t.Fatalf("%T hides cannot-regenerate destruction", r)
					}
					if !reflect.DeepEqual(before, r) {
						t.Fatalf("%T was changed by inspection", r)
					}
				}
			}
		})
	}
	for _, r := range []ReplacementEffect{&regenerationReplacement{}, &staticRegenerationReplacement{}, &nextDestructionReplacement{}} {
		if domain, known := InspectReplacementActionDomains(r); !known || domain != ReplacementDestructionActions {
			t.Fatalf("%T lost destruction domain", r)
		}
	}
	var absent *fogReplacement
	for _, r := range []ReplacementEffect{nil, absent, &proposalTrap{}, &temporaryDeathExileReplacement{}, &deathReturnHandReplacement{}, &faceUpDomainImpostor{&faceUpActionReplacement{}}, &drawReplacementEffect{}} {
		if _, known := InspectReplacementActionDomains(r); known {
			t.Fatalf("%T supplied unsupported domain", r)
		}
	}
}

func TestReplacementActionDomainNoCallbacks(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Subject", "", 3, 3), a.PlayerID())
	trap := NewPermanentFilter("trap", func(*Permanent, *Game) bool { panic("filter callback") })
	r := &damagePreventionRuleReplacement{from: trap, to: trap}
	g.effects.AddReplacement(r)
	if proceeds, known := g.InspectDestructionProposal(PermanentVisit{p.ID(), p.incarnationID}, false); !known || !proceeds {
		t.Fatal("damage filters obscured destruction")
	}
	if r.consumed || p.Tapped || p.Damage != 0 || p.HasKeyword(core.CantRegenerate) {
		t.Fatal("inspection changed state")
	}
	g.effects.AddReplacement(&proposalTrap{})
	if _, known := g.InspectDestructionProposal(PermanentVisit{p.ID(), p.incarnationID}, false); known {
		t.Fatal("opaque replacement ignored")
	}
}

type domainImpostor struct{ *fogReplacement }

func (*domainImpostor) Matches(Action, GameReader) bool { panic("impostor callback") }

func TestReplacementActionDomainClosedTypes(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Subject", "", 3, 3), a.PlayerID())
	var absent *regenerationReplacement
	for _, r := range []ReplacementEffect{nil, absent, &domainImpostor{&fogReplacement{}}} {
		g.effects.replacements = []ReplacementEffect{r}
		if _, known := g.InspectDestructionProposal(PermanentVisit{p.ID(), p.incarnationID}, false); known {
			t.Fatalf("%T accepted as proof", r)
		}
	}
}
