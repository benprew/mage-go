package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAttackGroupDefender(t *testing.T) {
	for _, state := range []string{"normal", "leave", "phase-before", "phase-after", "remove-before", "control", "copy"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			defender, count := BindPlayer("defender"), BindNumber("count")
			card := NewEnchantment("Observer", "", WithAbility(WhenEvent(OneOrMoreCreaturesAttack().WithDefendingPlayer(defender).WithCount(count), GainLifeFor(FromPlayer(defender), FromNumber(count)))))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			if state == "phase-before" {
				g.PhaseOut(source.ID())
			}
			if state == "remove-before" {
				g.AddContinuousEffect(RemoveAllAbilities(source.ID(), core.Indefinite))
			}
			first := g.PutOnBattlefield(NewCreature("First", "", 1, 8), a.PlayerID())
			second := g.PutOnBattlefield(NewCreature("Second", "", 1, 8), a.PlayerID())
			first.RevokeBaseAttr(core.AttrSummonSick)
			second.RevokeBaseAttr(core.AttrSummonSick)
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{first.ID(), second.ID()})
			g.PutTriggersOnStack()
			if state == "phase-before" || state == "remove-before" {
				if g.StackSize() != 0 {
					t.Fatal("inactive source triggered")
				}
				return
			}
			if g.StackSize() != 1 || b.Life() != 20 {
				t.Fatal("aggregate timing")
			}
			g.GetCombat().Reset()
			g.FireEvent(core.GameEvent{Type: core.EvtAttackersDeclared, PlayerID: b.PlayerID(), Amount: 3})
			g.PutTriggersOnStack()
			switch state {
			case "leave":
				g.BouncePermanentToHand(source)
			case "phase-after":
				g.PhaseOut(source.ID())
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			wantA := 23
			if state == "copy" {
				wantA = 26
			}
			for _, game := range []*Game{g, clone} {
				if game.GetPlayer(a.PlayerID()).Life() != wantA || game.GetPlayer(b.PlayerID()).Life() != 22 {
					t.Fatal("saved group defender or count changed")
				}
			}
		})
	}
}

func TestAttackGroupDefenderInvalid(t *testing.T) {
	p := BindPlayer("defender")
	for _, pattern := range []EventPattern{
		OneOrMoreCreaturesAttack().WithDefendingPlayer(nil),
		OneOrMoreCreaturesAttack().WithDefendingPlayer(p).WithDefendingPlayer(p),
	} {
		card := NewEnchantment("Invalid", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid capture accepted")
		}
	}
}

type attackGroupPlayer struct {
	*BasePlayer
	attackers []uuid.UUID
}

func (p *attackGroupPlayer) DeclareAttackers(*Game) []uuid.UUID { return p.attackers }

func TestAttackGroupDefenderNativeDeclaration(t *testing.T) {
	a := &attackGroupPlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(1)
	defender := BindPlayer("defender")
	card := NewEnchantment("Observer", "", WithAbility(WhenEvent(OneOrMoreCreaturesYouControlAttack().WithDefendingPlayer(defender), GainLifeFor(FromPlayer(defender), Fixed(2)))))
	g.PutOnBattlefield(card, a.PlayerID())
	for range 2 {
		p := g.PutOnBattlefield(NewCreature("Attacker", "", 1, 8), a.PlayerID())
		p.RevokeBaseAttr(core.AttrSummonSick)
		a.attackers = append(a.attackers, p.ID())
	}
	g.RunStepWithPriority(core.DeclareAttackers)
	if b.Life() != 22 || a.Life() != 20 {
		t.Fatal("native declaration defender or multiplicity")
	}
}
