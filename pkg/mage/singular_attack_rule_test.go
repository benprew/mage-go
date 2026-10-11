package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type singularAttackPlayer struct {
	*BasePlayer
	offers int
}

func (*singularAttackPlayer) DeclareAttackers(*Game) []uuid.UUID { return nil }
func (p *singularAttackPlayer) ChooseMayAbility(string) bool     { p.offers++; return true }

func TestSingularAttackRuleLifetime(t *testing.T) {
	for _, change := range []string{"normal", "phase", "cycle", "return", "control", "remove-abilities", "source-departure", "source-phase", "cleanup", "restricted", "tapped", "sick", "tax"} {
		for _, native := range []bool{false, true} {
			t.Run(change+map[bool]string{true: "/native", false: "/explicit"}[native], func(t *testing.T) {
				a, b := &singularAttackPlayer{BasePlayer: NewBasePlayer("A")}, &singularAttackPlayer{BasePlayer: NewBasePlayer("B")}
				g := NewGame(a, b)
				g.SetTurn(1)
				source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
				options := []CardOption{}
				if change == "tax" {
					options = append(options, WithStaticAbility(SourceCantAttackUnlessPays(GenericCost(1))))
				}
				card := NewCreature("Subject", "", 2, 3, options...)
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				p.RevokeBaseAttr(core.AttrSummonSick)
				peer := g.PutOnBattlefield(NewCreature("Peer", "", 2, 3), a.PlayerID())
				peer.RevokeBaseAttr(core.AttrSummonSick)
				binding := BindPermanent("subject")
				g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{binding: {p.ID(), p.incarnationID}}
				if err := applyEffectWithCaptures(g, MustAttackThisTurn(From(binding)), source.ID(), a.PlayerID(), nil, nil, g.resolution.permanentBindings, nil); err != nil {
					t.Fatal(err)
				}
				clone := g.Clone()
				want := true
				actor := a.PlayerID()
				switch change {
				case "phase":
					g.PhaseOut(p.ID())
					want = false
				case "cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "return":
					g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
					p = g.PutOnBattlefield(card, a.PlayerID())
					p.RevokeBaseAttr(core.AttrSummonSick)
					want = false
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
					g.SetActivePlayerIndex(1)
					actor = b.PlayerID()
				case "remove-abilities":
					if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), p.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "source-departure":
					g.RemoveFromBattlefield(source)
				case "source-phase":
					g.PhaseOut(source.ID())
				case "cleanup":
					g.RunStepWithPriority(core.Cleanup)
					want = false
				case "restricted":
					if err := ApplyEffect(g, CantAttackMatchingThisTurn(IsCreature), source.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					want = false
				case "tapped":
					g.TapPermanent(g.FindPermanent(p.ID()))
					want = false
				case "sick":
					g.MutablePermanent(p.ID()).GrantBaseAttr(core.AttrSummonSick)
					want = false
				case "tax":
					a.ManaPool().Add(core.Colorless, 1)
					want = false
				}
				if native {
					g.RunStepWithPriority(core.DeclareAttackers)
				} else {
					g.ExecuteAttackers(actor, nil)
				}
				if g.GetCombat().IsAttacking(p.ID()) != want || g.GetCombat().IsAttacking(peer.ID()) {
					t.Fatal("singular requirement or attack legality")
				}
				if clone.GetCombat().IsAttacking(p.ID()) {
					t.Fatal("shared combat")
				}
				expectedMana := 1
				if native {
					expectedMana = 0
				}
				if change == "tax" && (a.ManaPool().TotalMana() != expectedMana || a.offers != 0) {
					t.Fatal("forced attack payment")
				}
				if !clone.hasTurnAttackRule(clone.FindPermanent(card.ID()), true) {
					t.Fatal("clone lost saved subject")
				}
				if want {
					g.RemoveFromCombat(p.ID())
					g.UntapPermanent(g.FindPermanent(p.ID()))
					g.SetStep(core.EndCombat)
					g.SetStep(core.BeginCombat)
					g.ExecuteAttackers(actor, nil)
					if !g.GetCombat().IsAttacking(p.ID()) {
						t.Fatal("extra combat lost requirement")
					}
				}
			})
		}
	}
}

func TestSingularAttackRuleDefinitionAndAbsent(t *testing.T) {
	target := DeclarePermanentTarget("subject", IsCreature)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "source", Card: NewCreature("Source", "", 1, 1, WithAbility(NewActivated(GenericCost(0), MustAttackThisTurn(This))))},
		{Name: "nil", Card: NewInstant("Nil", "", WithAbility(Spell(MustAttackThisTurn(nil)))), WantErr: true},
		{Name: "unbound", Card: NewInstant("Unbound", "", WithAbility(Spell(MustAttackThisTurn(target)))), WantErr: true},
	})
	a, b := &singularAttackPlayer{BasePlayer: NewBasePlayer("A")}, &singularAttackPlayer{BasePlayer: NewBasePlayer("B")}
	g := NewGame(a, b)
	binding := BindPermanent("absent")
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{binding: {uuid.New(), uuid.New()}}
	before := len(g.effects.effects)
	if err := applyEffectWithCaptures(g, MustAttackThisTurn(From(binding)), uuid.New(), a.PlayerID(), nil, nil, g.resolution.permanentBindings, nil); err != nil {
		t.Fatal(err)
	}
	if len(g.effects.effects) != before {
		t.Fatal("absent subject created rule")
	}
}

func TestSingularAttackRuleDelayedSource(t *testing.T) {
	for _, change := range []string{"normal", "phase", "cycle", "return", "departed"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.SetTurn(1)
			card := NewCreature("Source", "", 2, 3, WithAbility(NewActivated(GenericCost(0), AtNext(EachEndStep, MustAttackThisTurn(This)))))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			switch change {
			case "phase":
				g.PhaseOut(source.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "return":
				g.RemoveFromBattlefield(source)
				g.PutOnBattlefield(card, a.PlayerID())
			case "departed":
				g.RemoveFromBattlefield(source)
			}
			clone := g.Clone()
			for _, game := range []*Game{g, clone} {
				for range 3 {
					if len(game.WaitingDelayedPermanentSources()) != 1 {
						t.Fatal("waiting source reference lost")
					}
				}
				current := game.FindPermanentIncludingPhased(source.ID())
				if current != nil && game.hasTurnAttackRule(current, true) {
					t.Fatal("query installed requirement")
				}
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
				current = game.FindPermanentIncludingPhased(source.ID())
				want := change == "normal" || change == "cycle"
				if current != nil && game.hasTurnAttackRule(current, true) != want {
					t.Fatal("delay followed wrong source visit")
				}
				if len(game.WaitingDelayedPermanentSources()) != 0 {
					t.Fatal("consumed delay kept reference")
				}
			}
		})
	}
}
