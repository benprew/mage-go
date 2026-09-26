package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestOpponentCreatureDiesFormerController(t *testing.T) {
	for _, tc := range []struct {
		name                string
		ownerB, controllerB bool
		want                int
	}{
		{"opponent", true, true, 1},
		{"stolen from opponent", true, false, 0},
		{"opponent controls your creature", false, true, 1},
		{"your creature", false, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			former := BindPlayer("former")
			source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(OpponentCreatureDies().WithController(former), GainLifeFor(FromPlayer(former), Fixed(1)))))
			source.SetOwner(a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			victim := NewCreature("Victim", "", 1, 1)
			owner := a.PlayerID()
			if tc.ownerB {
				owner = b.PlayerID()
			}
			victim.SetOwner(owner)
			g.PutOnBattlefield(victim, owner)
			if tc.controllerB != tc.ownerB {
				controller := a.PlayerID()
				if tc.controllerB {
					controller = b.PlayerID()
				}
				g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: victim.ID(), ControllerID: controller, Duration: core.Indefinite})
			}
			g.DestroyPermanent(g.FindPermanent(victim.ID()))
			g.PutTriggersOnStack()
			if g.StackSize() != tc.want {
				t.Fatalf("triggers = %d, want %d", g.StackSize(), tc.want)
			}
			if tc.want == 1 {
				clone := g.Clone()
				clone.ResolveStack()
				if b.Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 21 {
					t.Fatal("clone shared trigger state")
				}
			}
			g.ResolveStack()
			if b.Life() != 20+tc.want {
				t.Fatalf("opponent life = %d", b.Life())
			}
		})
	}
}

func TestOpponentCreatureDiesLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*Game, *Permanent, *Permanent)
		want  int
	}{
		{"ordinary death", nil, 1},
		{"phased source", func(g *Game, s, _ *Permanent) { g.PhaseOut(s.ID()) }, 0},
		{"exiled creature", func(g *Game, _, v *Permanent) { g.ExilePermanent(v) }, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(OpponentCreatureDies(), GainLifeFor(You, Fixed(1)))))
			source.SetOwner(a.PlayerID())
			victim := NewCreature("Victim", "", 1, 1)
			victim.SetOwner(b.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			g.PutOnBattlefield(victim, b.PlayerID())
			if tc.setup != nil {
				tc.setup(g, g.FindPermanent(source.ID()), g.FindPermanent(victim.ID()))
			}
			if tc.name != "exiled creature" {
				g.DestroyPermanent(g.FindPermanent(victim.ID()))
			}
			g.PutTriggersOnStack()
			if g.StackSize() != tc.want {
				t.Fatalf("triggers = %d, want %d", g.StackSize(), tc.want)
			}
		})
	}
}

func TestOpponentCreatureDiesDeclaration(t *testing.T) {
	former := BindPlayer("former")
	pattern := OpponentCreatureDies().WithController(former)
	if info := InspectDefinition(pattern); info.Role != "event-pattern" || info.Text != "a creature an opponent controls dies" {
		t.Fatalf("inspection = %+v", info)
	}
	for _, tc := range []struct {
		name    string
		pattern EventPattern
		body    ResolvingEffect
		want    string
	}{
		{"valid", pattern, GainLifeFor(FromPlayer(former), Fixed(1)), ""},
		{"nil", OpponentCreatureDies().WithController(nil), GainLifeFor(You, Fixed(1)), "binding"},
		{"empty", OpponentCreatureDies().WithController(BindPlayer("")), GainLifeFor(You, Fixed(1)), "name"},
		{"unbound", OpponentCreatureDies(), GainLifeFor(FromPlayer(former), Fixed(1)), "producer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewEnchantment("Watcher", "", WithAbility(WhenEvent(tc.pattern, tc.body)))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestOpponentCreatureDiesSimultaneousAndToken(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(OpponentCreatureDies(), GainLifeFor(You, Fixed(1)))))
	source.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 1, 1)
	victim.SetOwner(b.PlayerID())
	token := NewToken("Token", 1, 1, []core.CardType{core.TypeCreature}, nil)
	token.SetOwner(b.PlayerID())
	artifact := NewArtifact("Artifact", "")
	artifact.SetOwner(b.PlayerID())
	for _, card := range []Card{source, victim, token, artifact} {
		g.PutOnBattlefield(card, card.Owner())
	}
	g.DestroyPermanents([]*Permanent{g.FindPermanent(source.ID()), g.FindPermanent(victim.ID()), g.FindPermanent(token.ID()), g.FindPermanent(artifact.ID())})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("triggers = %d, want 2", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatalf("life = %d", a.Life())
	}
}

func TestOpponentCreatureDiesPendingCopyAndReturn(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	former := BindPlayer("former")
	source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(OpponentCreatureDies().WithController(former), GainLifeFor(FromPlayer(former), Fixed(1)))))
	source.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 1, 1)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.DestroyPermanent(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers = %d", g.StackSize())
	}
	obj := g.StackPeek()
	if len(obj.Targets) != 0 {
		t.Fatal("event controller became a target")
	}
	if g.CopyStackObjectDirect(obj, a.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	clone := g.Clone()
	clone.ResolveStack()
	if b.Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 22 {
		t.Fatal("copy or clone lost event-time controller")
	}
	g.ResolveStack()
	if b.Life() != 22 {
		t.Fatalf("life = %d", b.Life())
	}
}
