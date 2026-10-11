package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTemporaryDeathExileRoutes(t *testing.T) {
	for _, route := range []string{"destroy", "sacrifice", "graveyard", "zero", "indestructible", "phase", "return", "expiry", "clone", "token", "indestructible-sacrifice"} {
		t.Run(route, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(NewCreature("Subject", "", 2, 3), b.PlayerID())
			p.Card.SetOwner(b.PlayerID())
			if err := ApplyEffect(g, ExileIfWouldDieThisTurn(This), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			switch route {
			case "zero":
				g.AddCountersWithReplacement(p, core.M1M1, 3, uuid.Nil, false)
			case "indestructible", "indestructible-sacrifice":
				p.GrantBaseAttr(core.Indestructible)
			case "token":
				p.IsToken = true
			case "phase":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.BouncePermanentToHand(p)
				p = g.PutOnBattlefield(p.Card, b.PlayerID())
			case "expiry":
				g.GetEffects().ClearReplacementsEndOfTurn()
			case "clone":
				g = g.Clone()
			}
			switch route {
			case "sacrifice", "indestructible-sacrifice":
				g.DoSacrifice(g.FindPermanent(p.ID()))
			case "graveyard":
				g.PutPermanentIntoGraveyard(g.FindPermanent(p.ID()))
			case "zero":
				g.CheckStateBasedActions()
			default:
				g.DestroyPermanent(g.FindPermanent(p.ID()))
			}
			if route == "indestructible" {
				if g.FindPermanent(p.ID()) == nil {
					t.Fatal("indestructible destroyed")
				}
				return
			}
			if g.FindPermanent(p.ID()) != nil {
				t.Fatal("subject did not leave")
			}
			wantExile := route != "return" && route != "expiry" && route != "token"
			if (g.FindExiledCard(p.Card.ID()) != nil) != wantExile {
				t.Fatalf("exile route %s", route)
			}
			if (wantExile || route == "token") && g.CreatureDeaths() != 0 {
				t.Fatal("exile counted as death")
			}
			if route == "clone" && len(b.Graveyard()) != 0 {
				t.Fatal("clone changed player")
			}
		})
	}
}

func TestTemporaryDeathExileDeclarations(t *testing.T) {
	card := NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), ExileIfWouldDieThisTurn(nil))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("nil subject accepted")
	}
}

type deathExileOrderPlayer struct {
	*BasePlayer
	index   int
	choices int
}

func (p *deathExileOrderPlayer) ChooseMode(_ []string, _ string) int { p.choices++; return p.index }

func TestTemporaryDeathExileOrder(t *testing.T) {
	for _, index := range []int{0, 1} {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			a := NewBasePlayer("A")
			b := &deathExileOrderPlayer{BasePlayer: NewBasePlayer("B"), index: index}
			g := NewGame(a, b)
			p := g.PutOnBattlefield(NewCreature("Subject", "", 1, 3), b.PlayerID())
			g.AddRegenerationShield(p.ID())
			if err := ApplyEffect(g, ExileIfWouldDieThisTurn(This), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.DestroyPermanent(p)
			if b.choices != 1 || (g.FindPermanent(p.ID()) != nil) != (index == 0) || (g.FindExiledCard(p.Card.ID()) != nil) != (index == 1) {
				t.Fatal("affected controller replacement order")
			}
		})
	}
}

type deathExileBatchProbe struct {
	replacementBase
	first, second uuid.UUID
	t             *testing.T
	calls         int
}

func (*deathExileBatchProbe) IsActive(GameReader) bool { return true }
func (r *deathExileBatchProbe) Matches(a Action, _ GameReader) bool {
	switch a.(type) {
	case *DestroyPermanentAction, *permanentGraveyardAction:
		return true
	}
	return false
}
func (r *deathExileBatchProbe) Replace(a Action, g *Game) Action {
	r.calls++
	if g.FindPermanent(r.first) == nil || g.FindPermanent(r.second) == nil {
		r.t.Fatal("replacement choices after batch departure")
	}
	return a
}
func (r *deathExileBatchProbe) Clone() ReplacementEffect { c := *r; return &c }

func TestTemporaryDeathExileMixedStateBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("Lethal", "", 1, 3), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Zero", "", 1, 3), b.PlayerID())
	probe := &deathExileBatchProbe{replacementBase: replacementBase{duration: core.EndOfTurn}, first: first.ID(), second: second.ID(), t: t}
	g.AddReplacementEffect(probe)
	for _, p := range []*Permanent{first, second} {
		if err := ApplyEffect(g, ExileIfWouldDieThisTurn(This), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	first.Damage = 3
	g.AddCountersWithReplacement(second, core.M1M1, 3, uuid.Nil, false)
	g.CheckStateBasedActions()
	if probe.calls != 2 || g.CreatureDeaths() != 0 || g.FindExiledCard(first.Card.ID()) == nil || g.FindExiledCard(second.Card.ID()) == nil {
		t.Fatal("mixed simultaneous death batch")
	}
}

func TestTemporaryDeathExileWaitingIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Subject", "", 2, 3), b.PlayerID())
	p.Card.SetOwner(b.PlayerID())
	if err := ApplyEffect(g, ExileIfWouldDieThisTurn(This), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	saved := g.WaitingExileSources()
	if len(saved) != 1 || saved[0].Incarnation() != p.incarnationID {
		t.Fatal("missing exact waiting reference")
	}
	g.BouncePermanentToHand(p)
	returned := g.PutOnBattlefield(p.Card, b.PlayerID())
	refs := g.WaitingExileSources()
	if len(refs) != 1 || !refs[0].SameObject(saved[0]) || refs[0].Incarnation() == returned.incarnationID {
		t.Fatal("waiting reference followed return")
	}
	clone := g.Clone()
	g.GetEffects().ClearReplacementsEndOfTurn()
	if len(g.WaitingExileSources()) != 0 || len(clone.WaitingExileSources()) != 1 {
		t.Fatal("cleanup or clone references")
	}
}
