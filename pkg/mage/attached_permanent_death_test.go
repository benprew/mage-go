package mage

import (
	"testing"
)

func TestAttachedPermanentDiesCaptures(t *testing.T) {
	for _, kind := range []string{"creature", "land", "artifact"} {
		for _, simultaneous := range []bool{false, true} {
			t.Run(kind+map[bool]string{true: "/batch", false: "/single"}[simultaneous], func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				controller, owner := BindPlayer("controller"), BindPlayer("owner")
				card, permanent := BindCard("graveyard card"), BindPermanent("departing permanent")
				var host Card
				switch kind {
				case "land":
					host = NewLand("Host")
				case "artifact":
					host = NewArtifact("Host", "")
				default:
					host = NewCreature("Host", "", 2, 3)
				}
				host.SetOwner(a.PlayerID())
				hp := g.PutOnBattlefield(host, b.PlayerID())
				aura := NewAura("Watcher", "", WithCastTarget(TargetPermanent(AnyPermanent)), WithAbility(WhenEvent(AttachedPermanentDies().WithController(controller).WithOwner(owner).WithCard(card).WithPermanent(permanent),
					Sequence(GainLifeFor(FromPlayer(controller), Fixed(2)), GainLifeFor(FromPlayer(owner), Fixed(1)), ReturnCardToHand(FromCard(card))))))
				aura.SetOwner(a.PlayerID())
				ap := g.PutOnBattlefield(aura, a.PlayerID())
				g.Attach(ap.ID(), hp.ID())
				if err := ValidateCardDefinition(aura.Name(), aura); err != nil {
					t.Fatal(err)
				}
				if simultaneous {
					g.DestroyPermanents([]*Permanent{g.FindPermanent(ap.ID()), g.FindPermanent(hp.ID())})
				} else {
					g.DestroyPermanent(hp)
				}
				g.PutTriggersOnStack()
				if g.StackSize() != 1 {
					t.Fatalf("stack=%d", g.StackSize())
				}
				clone := g.Clone()
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).Life() != 21 || clone.GetPlayer(b.PlayerID()).Life() != 22 || len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 {
					t.Fatal("lost death capture")
				}
				if a.Life() != 20 || b.Life() != 20 || len(a.Hand()) != 0 {
					t.Fatal("clone shared state")
				}
				g.ResolveStack()
				if a.Life() != 21 || b.Life() != 22 || len(a.Hand()) != 1 {
					t.Fatal("lost original capture")
				}
			})
		}
	}
}

func TestAttachedPermanentDiesVisitAndPhase(t *testing.T) {
	for _, change := range []string{"phase", "exile", "return", "visit"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := BindCard("card")
			host := NewArtifact("Host", "")
			host.SetOwner(b.PlayerID())
			hp := g.PutOnBattlefield(host, b.PlayerID())
			aura := NewAura("Watcher", "", WithCastTarget(TargetPermanent(AnyPermanent)), WithAbility(WhenEvent(AttachedPermanentDies().WithCard(card), ReturnCardToHand(FromCard(card)))))
			ap := g.PutOnBattlefield(aura, a.PlayerID())
			g.Attach(ap.ID(), hp.ID())
			switch change {
			case "phase":
				g.PhaseOut(hp.ID())
			case "exile":
				g.ExilePermanent(hp)
			case "return":
				g.RemoveFromBattlefield(hp)
				g.PutOnBattlefield(host, b.PlayerID())
				g.DestroyPermanent(g.FindPermanent(host.ID()))
			case "visit":
				g.DestroyPermanent(hp)
				b.RemoveFromGraveyard(host.ID())
				b.AddToGraveyard(host)
			}
			g.PutTriggersOnStack()
			want := 0
			if change == "visit" {
				want = 1
			}
			if g.StackSize() != want {
				t.Fatalf("stack=%d want=%d", g.StackSize(), want)
			}
			g.ResolveStack()
			if len(b.Hand()) != 0 {
				t.Fatal("moved wrong graveyard visit")
			}
		})
	}
}

func TestAttachedPermanentDiesInvalid(t *testing.T) {
	card := NewAura("Invalid", "", WithAbility(WhenEvent(AttachedPermanentDies().WithPermanent(nil), GainLifeFor(You, Fixed(1)))))
	if ValidateCardDefinition(card.Name(), card) == nil {
		t.Fatal("nil binding accepted")
	}
}
