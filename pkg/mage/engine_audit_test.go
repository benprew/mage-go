package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func auditStateGame() (*Game, *BasePlayer, *Permanent) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewLand("Hand"))
	a.AddToLibrary(NewLand("Library"))
	ability := NewStateTriggered(false, GainLife(1)).SetCondition(func(_ *GameEvent, g GameReader, _, controller uuid.UUID) bool {
		return len(g.GetPlayer(controller).Hand()) == 0
	})
	card := NewCreature("Watcher", "", 1, 1, WithAbility(ability))
	g.PutOnBattlefield(card, a.PlayerID())
	return g, a, g.FindPermanent(card.ID())
}

func TestAuditStateTriggerTransient(t *testing.T) {
	g, a, p := auditStateGame()
	g.CheckStateTriggers()
	if err := ApplyEffect(g, CompositeEffects("", DiscardHand(), DrawCards(Fixed(1))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.CheckStateTriggers()
	if len(g.triggers.Pending()) != 1 {
		t.Fatalf("pending = %d, want 1", len(g.triggers.Pending()))
	}
}

func TestAuditStateTriggerLifecycle(t *testing.T) {
	for _, finish := range []string{"resolve", "counter", "remove"} {
		t.Run(finish, func(t *testing.T) {
			g, a, p := auditStateGame()
			if err := ApplyEffect(g, DiscardHand(), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			a.AddToHand(NewLand("Temporary"))
			g.CheckStateTriggers()
			if err := ApplyEffect(g, DiscardHand(), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.CheckStateTriggers()
			if len(g.triggers.Pending()) != 0 {
				t.Fatal("trigger repeated before leaving stack")
			}
			switch finish {
			case "resolve":
				g.ResolveStackObject(g.stack.Pop())
			case "counter":
				g.CounterSpellOnStack(p.ID())
			case "remove":
				g.stack.RemoveByID(g.stack.Peek().ID)
			}
			g.CheckStateTriggers()
			if len(g.triggers.Pending())+len(g.stack.Objects()) != 1 {
				t.Fatal("trigger did not rearm")
			}
		})
	}
}

func TestAuditSacrificeSourceController(t *testing.T) {
	for _, effect := range []Effect{SacrificeSource(), SacrificeSourceStep()} {
		t.Run(effect.Text(), func(t *testing.T) {
			g, a, p := auditStateGame()
			b := g.players[1]
			if err := ApplyEffect(g, GainControl().Targeting(ToTarget()).Until(EndOfTurn), uuid.Nil, b.PlayerID(), []uuid.UUID{p.ID()}); err != nil {
				t.Fatal(err)
			}
			if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(p.ID()) == nil {
				t.Fatal("former controller sacrificed opponent's permanent")
			}
		})
	}
}

type auditChoicePlayer struct {
	*BasePlayer
	choice int
	calls  int
}

func (p *auditChoicePlayer) ChooseMode(_ []string, _ string) int { p.calls++; return p.choice }

type auditReplacement struct {
	replacementBase
	multiply, subtract int
}

func (r *auditReplacement) Matches(a Action, _ GameReader) bool {
	_, ok := a.(*DamageToPlayerAction)
	return ok
}
func (r *auditReplacement) IsActive(GameReader) bool { return true }
func (r *auditReplacement) Clone() ReplacementEffect { cp := *r; return &cp }
func (r *auditReplacement) Replace(a Action, _ *Game) Action {
	d := a.(*DamageToPlayerAction)
	return d.WithAmount(d.Amount()*r.multiply - r.subtract)
}
func TestAuditReplacementAffectedPlayerChooses(t *testing.T) {
	for _, choice := range []int{0, 1} {
		t.Run(string(rune('0'+choice)), func(t *testing.T) {
			a := &auditChoicePlayer{BasePlayer: NewBasePlayer("A")}
			b := &auditChoicePlayer{BasePlayer: NewBasePlayer("B"), choice: choice}
			g := NewGame(a, b)
			g.effects.AddReplacement(&auditReplacement{multiply: 2})
			g.effects.AddReplacement(&auditReplacement{multiply: 1, subtract: 1})
			result := g.effects.ApplyReplacements(NewDamageToPlayerAction(a.PlayerID(), b.PlayerID(), 3, false), g).(*DamageToPlayerAction)
			want := []int{5, 4}[choice]
			if result.Amount() != want || b.calls != 1 || a.calls != 0 {
				t.Fatalf("amount=%d, calls A=%d B=%d; want %d, 0, 1", result.Amount(), a.calls, b.calls, want)
			}
		})
	}
}

func TestAuditStateTriggerCloneAndReentry(t *testing.T) {
	g, a, p := auditStateGame()
	if err := ApplyEffect(g, DiscardHand(), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	clone := g.Clone()
	clone.stack.Pop()
	clone.CheckStateTriggers()
	g.CheckStateTriggers()
	if len(clone.triggers.Pending()) != 1 || len(g.triggers.Pending()) != 0 {
		t.Fatal("clone changed original trigger state")
	}
	card := p.Card
	g.RemoveFromBattlefield(p)
	g.PutOnBattlefield(card, a.PlayerID())
	g.CheckStateTriggers()
	if len(g.triggers.Pending()) != 1 {
		t.Fatal("old stack instance suppressed new permanent")
	}
}

func TestAuditSimultaneousDeathTriggers(t *testing.T) {
	for _, mode := range []string{"wipe", "lethal", "zero"} {
		t.Run(mode, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := NewCreature("Watcher", "", 1, 1, WithAbility(AnyCreatureDiesTrigger(GainLife(1), false)))
			other := NewCreature("Other", "", 1, 1)
			watcher.SetOwner(a.PlayerID())
			other.SetOwner(b.PlayerID())
			g.PutOnBattlefield(watcher, a.PlayerID())
			g.PutOnBattlefield(other, b.PlayerID())
			switch mode {
			case "wipe":
				if err := ApplyEffect(g, DestroyAllCreatures(), uuid.Nil, a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "lethal":
				g.MutablePermanent(watcher.ID()).Damage = 1
				g.MutablePermanent(other.ID()).Damage = 1
				g.CheckStateBasedActions()
			case "zero":
				g.MutablePermanent(watcher.ID()).AddCounter(M1M1, 1)
				g.MutablePermanent(other.ID()).AddCounter(M1M1, 1)
				g.CheckStateBasedActions()
			}
			g.ResolveStack()
			if a.Life() != 22 {
				t.Fatalf("life = %d, want 22", a.Life())
			}
		})
	}
}

func TestAuditStateTriggerRearmsBeforeStateBasedActions(t *testing.T) {
	g, a, p := auditStateGame()
	if err := ApplyEffect(g, DiscardHand(), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	obj := g.stack.Pop()
	obj.Effects = []Effect{FuncEffect("mark damage", EffectProperties{}, func(g *Game, source, _ uuid.UUID, _ []uuid.UUID) error {
		g.MutablePermanent(source).Damage = 1
		return nil
	})}
	g.ResolveStackObject(obj)
	if g.FindPermanent(p.ID()) != nil {
		t.Fatal("source survived lethal damage")
	}
	if len(g.triggers.Pending())+len(g.stack.Objects()) != 1 {
		t.Fatal("state-based action removed source before its trigger rearmed")
	}
}
