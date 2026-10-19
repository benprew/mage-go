package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDestructionProposal(t *testing.T) {
	for _, state := range []string{"plain", "shield", "two shields", "other shield", "cycle shield", "prohibited", "no regeneration", "indestructible", "phase", "return", "static", "static face down", "static removed", "static granted", "static prohibited", "static no regeneration", "cancel", "cancel no regeneration", "competing", "cleanup", "damage prevention", "fog"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Subject", "", 3, 3)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			visit := PermanentVisit{p.ID(), p.incarnationID}
			noRegen, want := false, true
			switch state {
			case "shield", "two shields", "cycle shield", "prohibited", "no regeneration":
				g.AddRegenerationShield(p.ID())
				want = false
				if state == "cycle shield" {
					g.effects.cycleReplacements = g.effects.replacements
					g.effects.replacements = nil
				}
				if state == "two shields" {
					g.AddRegenerationShield(p.ID())
				}
				if state == "prohibited" {
					p.GrantBaseAttr(core.CantRegenerate)
					want = true
				}
				if state == "no regeneration" {
					noRegen, want = true, true
				}
			case "other shield":
				other := g.PutOnBattlefield(NewCreature("Other", "", 3, 3), a.PlayerID())
				g.AddRegenerationShield(other.ID())
			case "indestructible":
				p.GrantBaseAttr(core.Indestructible)
				want = false
			case "phase":
				g.PhaseOut(p.ID())
				want = false
			case "return":
				g.AddRegenerationShield(p.ID())
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(card, a.PlayerID())
				want = false
			case "static", "static face down", "static removed", "static granted", "static prohibited", "static no regeneration":
				ability := RegenerateInsteadOfDestroy()
				if state == "static granted" {
					ability = WrapGrantedAbility(ability)
				}
				p.RuntimeAbilities = append(p.RuntimeAbilities, ability)
				want = false
				if state == "static prohibited" {
					p.GrantBaseAttr(core.CantRegenerate)
					want = true
				}
				if state == "static no regeneration" {
					noRegen, want = true, true
				}
				if state == "static face down" {
					p.FaceDown = true
					want = true
				}
				if state == "static removed" {
					p.RuntimeAbilities = nil
					want = true
				}
			case "cancel", "cancel no regeneration", "competing", "cleanup":
				g.effects.AddReplacement(&nextDestructionReplacement{replacementBase: replacementBase{duration: core.EndOfTurn}, subject: p.ID(), incarnation: p.incarnationID})
				want = false
				if state == "cancel no regeneration" {
					noRegen = true
				}
				if state == "competing" {
					g.AddRegenerationShield(p.ID())
				}
				if state == "cleanup" {
					g.effects.ClearReplacementsEndOfTurn()
					want = true
				}
			case "fog":
				g.effects.AddReplacement(&fogReplacement{})
			case "damage prevention":
				g.AddPreventionShield(p.ID(), 3)
			}
			p.Damage = 2
			random := g.random.Clone()
			for range 2 {
				got, known := g.InspectDestructionProposal(visit, noRegen)
				if !known || got != want {
					t.Fatalf("proceeds %v known %v, want %v", got, known, want)
				}
			}
			if p.Damage != 2 || p.Tapped || !reflect.DeepEqual(random, g.random) {
				t.Fatal("inspection changed permanent")
			}
			clone := g.Clone()
			if got, known := clone.InspectDestructionProposal(visit, noRegen); !known || got != want {
				t.Fatal("clone changed inspection")
			}
			if state == "phase" {
				return
			}
			if state == "return" {
				current := g.FindPermanent(p.ID())
				if got, known := g.InspectDestructionProposal(PermanentVisit{current.ID(), current.incarnationID}, false); !known || !got {
					t.Fatal("old visit protected return")
				}
				return
			}
			for _, game := range []*Game{clone, g} {
				game.destroyPermanents([]*Permanent{game.FindPermanent(p.ID())}, noRegen)
				if (game.FindPermanent(p.ID()) == nil) != want {
					t.Fatal("inspection disagrees with native destruction")
				}
			}
			if state == "two shields" {
				if got, known := g.InspectDestructionProposal(visit, false); !known || got {
					t.Fatal("second shield lost")
				}
			}
			if state == "shield" || state == "cancel" {
				if got, known := g.InspectDestructionProposal(visit, false); !known || !got {
					t.Fatal("consumed replacement still applies")
				}
			}
		})
	}
}

type proposalTrap struct{ replacementBase }

func (*proposalTrap) Matches(Action, GameReader) bool { panic("Matches callback") }
func (*proposalTrap) Replace(Action, *Game) Action    { panic("Replace callback") }
func (*proposalTrap) IsActive(GameReader) bool        { panic("IsActive callback") }
func (r *proposalTrap) Clone() ReplacementEffect      { c := *r; return &c }

func TestDestructionProposalUnknown(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Subject", "", 3, 3), a.PlayerID())
	visit := PermanentVisit{p.ID(), p.incarnationID}
	for _, cycle := range []bool{false, true} {
		g.effects.replacements = nil
		g.effects.cycleReplacements = nil
		trap := &proposalTrap{}
		if cycle {
			g.effects.cycleReplacements = []ReplacementEffect{trap}
		} else {
			g.effects.AddReplacement(trap)
		}
		for _, shield := range []bool{false, true} {
			if shield {
				g.AddRegenerationShield(p.ID())
			}
			if _, known := g.InspectDestructionProposal(visit, false); known {
				t.Fatal("opaque replacement accepted")
			}
		}
	}
	p.GrantBaseAttr(core.Indestructible)
	if got, known := g.InspectDestructionProposal(visit, false); !known || got {
		t.Fatal("indestructible reached replacement pipeline")
	}
	if _, known := g.InspectDestructionProposal(PermanentVisit{ID: p.ID()}, false); known {
		t.Fatal("missing incarnation accepted")
	}
	var absent *Game
	if _, known := absent.InspectDestructionProposal(visit, false); known {
		t.Fatal("nil game accepted")
	}
	if _, known := g.InspectDestructionProposal(PermanentVisit{ID: uuid.New(), Incarnation: uuid.New()}, false); !known {
		t.Fatal("absent visit is not known")
	}
}

func TestDestructionProposalDescriptor(t *testing.T) {
	for _, instruction := range []DirectDestructionInstruction{{}, {Subject: This, Subjects: MatchingPermanents(AnyPermanent)}} {
		if instruction.Effect() != nil {
			t.Fatal("invalid descriptor built")
		}
	}
	for _, instruction := range []DirectDestructionInstruction{{Subject: This}, {Subject: This, CannotRegenerate: true}, {Subjects: MatchingPermanents(AnyPermanent)}, {Subjects: MatchingPermanents(AnyPermanent), CannotRegenerate: true}} {
		effect := instruction.Effect()
		plan, ok := InspectDirectDestruction(effect)
		if !ok || len(plan) != 1 || plan[0].CannotRegenerate != instruction.CannotRegenerate || plan[0].Subject != instruction.Subject || (plan[0].Subjects != nil) != (instruction.Subjects != nil) {
			t.Fatal("descriptor lost semantics")
		}
	}
}

func TestDestructionProposalLocalAndLiveUnknown(t *testing.T) {
	for _, state := range []string{"local death", "live prohibition"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(NewCreature("Subject", "", 3, 3), a.PlayerID())
			visit := PermanentVisit{p.ID(), p.incarnationID}
			if state == "local death" {
				p.RuntimeAbilities = append(p.RuntimeAbilities, ReturnToHandInsteadOfDyingWithPlayLock())
			} else {
				g.AddRegenerationShield(p.ID())
				g.effects.effects = append(g.effects.effects, &liveDamageRegenerationRule{
					funcContinuousEffect: FuncContinuousEffect(core.LayerAbility, core.EndOfTurn, func(*Game, uuid.UUID) error { panic("apply callback") }, func(*Game, uuid.UUID) bool { panic("active callback") }).(*funcContinuousEffect),
					filter:               NewPermanentFilter("opaque", func(*Permanent, *Game) bool { panic("filter callback") }),
				})
			}
			if _, known := g.InspectDestructionProposal(visit, false); known {
				t.Fatal("unsupported local or live replacement accepted")
			}
		})
	}
}

type proposalChoiceTrap struct{ *BasePlayer }

func (*proposalChoiceTrap) ChooseMode([]string, string) int { panic("replacement choice") }

func TestDestructionProposalNoChoices(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		a := &proposalChoiceTrap{NewBasePlayer("A")}
		g := NewGame(a, NewBasePlayer("B"))
		p := g.PutOnBattlefield(NewCreature("Subject", "", 3, 3), a.PlayerID())
		g.AddRegenerationShield(p.ID())
		cancellation := &nextDestructionReplacement{subject: p.ID(), incarnation: p.incarnationID}
		g.effects.AddReplacement(cancellation)
		if reverse {
			g.effects.replacements[0], g.effects.replacements[1] = g.effects.replacements[1], g.effects.replacements[0]
		}
		visit := PermanentVisit{p.ID(), p.incarnationID}
		if proceeds, known := g.InspectDestructionProposal(visit, false); !known || proceeds {
			t.Fatal("competing terminal replacements changed the result")
		}
		if cancellation.consumed || !g.HasRegenerationShield(p.ID()) {
			t.Fatal("inspection consumed a competing replacement")
		}
	}
}
