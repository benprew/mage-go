package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func deathCardEffect(subject PermanentRef, card *CardBinding, body ResolvingEffect) ResolvingEffect {
	return WhenDiesThisTurnWithCard(subject, BindPermanent("dead"), BindPlayer("owner"), BindPlayer("controller"), card, body)
}

func TestDelayedDeathCardVisits(t *testing.T) {
	for _, change := range []string{"normal", "waiting-clone", "pending-clone", "stack-clone", "returned-pending", "returned-stack", "battlefield-return", "phase-cycle", "replaced-death", "expired", "token", "simultaneous"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			card := NewCreature("Watched", "", 2, 3)
			card.SetOwner(b.PlayerID())
			watched := g.PutOnBattlefield(card, a.PlayerID())
			saved := BindCard("dead card")
			body := Sequence(ExileGraveyardCard(FromCard(saved)), GainLifeFor(You, Fixed(1)))
			if err := ApplyEffect(g, deathCardEffect(This, saved, body), watched.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if change == "token" {
				g.MutablePermanent(watched.ID()).IsToken = true
			}
			if change == "phase-cycle" {
				g.PhaseIn(g.PhaseOut(watched.ID()))
			}
			if change == "replaced-death" {
				g.AddExileIfWouldGoToGraveyardThisTurn(watched.ID(), source.ID())
			}
			if change == "expired" {
				g.RunStepWithPriority(core.Cleanup)
			}
			var clone *Game
			if change == "waiting-clone" {
				clone = g.Clone()
			}
			if change == "simultaneous" {
				g.DestroyPermanents([]*Permanent{g.FindPermanent(watched.ID()), source})
			} else {
				g.PutPermanentIntoGraveyard(g.FindPermanent(watched.ID()))
			}
			if g.StackSize() != 0 {
				t.Fatal("death body resolved before priority")
			}
			if change == "pending-clone" {
				clone = g.Clone()
			}
			if change == "returned-pending" {
				b.RemoveFromGraveyard(card.ID())
				b.AddToGraveyard(card)
			}
			g.PutTriggersOnStack()
			if change == "stack-clone" {
				clone = g.Clone()
			}
			if change == "returned-stack" {
				b.RemoveFromGraveyard(card.ID())
				b.AddToGraveyard(card)
			}
			if change == "battlefield-return" {
				b.RemoveFromGraveyard(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			}
			g.ResolveStack()
			wantExile := change != "returned-pending" && change != "returned-stack" && change != "battlefield-return" && change != "expired" && change != "token"
			if (g.FindExiledCard(card.ID()) != nil) != wantExile {
				t.Fatal("wrong graveyard visit moved")
			}
			wantLife := 21
			if change == "expired" || change == "replaced-death" {
				wantLife = 20
			}
			if a.Life() != wantLife {
				t.Fatalf("life=%d want=%d", a.Life(), wantLife)
			}
			if len(g.resolution.cardBindings) != 0 {
				t.Fatal("card binding leaked")
			}
			if clone != nil {
				if clone.FindExiledCard(card.ID()) != nil {
					t.Fatal("original changed clone")
				}
				if change == "waiting-clone" {
					clone.PutPermanentIntoGraveyard(clone.FindPermanent(watched.ID()))
				}
				clone.PutTriggersOnStack()
				clone.ResolveStack()
				if clone.FindExiledCard(card.ID()) == nil || clone.GetPlayer(a.PlayerID()).Life() != 21 {
					t.Fatal("clone lost death card")
				}
			}
		})
	}
}

func TestDelayedDeathCardDefinition(t *testing.T) {
	saved := BindCard("dead card")
	valid := deathCardEffect(This, saved, ExileGraveyardCard(FromCard(saved)))
	if err := ValidateCardDefinition("Valid", NewArtifact("Valid", "", WithAbility(Activate(Costs(), valid)))); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{
		deathCardEffect(This, nil, GainLifeFor(You, Fixed(1))),
		deathCardEffect(This, BindCard(""), GainLifeFor(You, Fixed(1))),
		Sequence(valid, ExileGraveyardCard(FromCard(saved))),
		deathCardEffect(This, saved, ExileGraveyardCard(FromCard(BindCard("unbound")))),
		deathCardEffect(nil, saved, GainLifeFor(You, Fixed(1))),
		deathCardEffect(This, saved, nil),
	} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))) == nil {
			t.Fatal("invalid binding accepted")
		}
	}
}

type deathCardMoveReplacement struct {
	replacementBase
	cancel bool
	calls  int
}

func (*deathCardMoveReplacement) IsActive(GameReader) bool { return true }
func (*deathCardMoveReplacement) Matches(action Action, _ GameReader) bool {
	a, ok := action.(*CardZoneMoveAction)
	return ok && a.FromZone() == core.ZoneGraveyard && a.Destination() == core.ZoneExile
}
func (r *deathCardMoveReplacement) Replace(action Action, _ *Game) Action {
	r.calls++
	if r.cancel {
		return nil
	}
	a := *(action.(*CardZoneMoveAction))
	a.destination = core.ZoneHand
	return &a
}
func (r *deathCardMoveReplacement) Clone() ReplacementEffect { c := *r; return &c }

func TestDelayedDeathCardDestinationReplacement(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		card := NewCreature("Watched", "", 2, 3)
		card.SetOwner(b.PlayerID())
		p := g.PutOnBattlefield(card, a.PlayerID())
		saved := BindCard("dead card")
		if err := ApplyEffect(g, deathCardEffect(This, saved, ExileGraveyardCard(FromCard(saved))), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		replacement := &deathCardMoveReplacement{cancel: cancel}
		g.AddReplacementEffect(replacement)
		observations := 0
		watch := NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
			if e.SourceID == card.ID() && e.FromZone == core.ZoneGraveyard {
				observations++
				if e.ToZone != core.ZoneHand || len(b.Hand()) != 1 {
					t.Fatal("zone event preceded final movement")
				}
			}
			return false
		})
		g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(watch)), a.PlayerID())
		g.PutPermanentIntoGraveyard(p)
		g.PutTriggersOnStack()
		g.ResolveStack()
		if replacement.calls != 1 || g.FindExiledCard(card.ID()) != nil {
			t.Fatal("exile bypassed replacement")
		}
		if cancel {
			if len(b.Graveyard()) != 1 || observations != 0 {
				t.Fatal("canceled move observed")
			}
		} else if len(b.Hand()) != 1 || observations != 1 {
			t.Fatal("replacement did not move to owner hand")
		}
	}
}

func TestDelayedDeathCardIndependentFrames(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	saved := BindCard("dead card")
	permanent, owner, controller := BindPermanent("dead"), BindPlayer("owner"), BindPlayer("controller")
	body := Sequence(ExileGraveyardCard(FromCard(saved)), GainLifeFor(FromPlayer(owner), ToughnessOfWithLKI(From(permanent))), GainLifeFor(FromPlayer(controller), Fixed(1)))
	subjects := make([]*Permanent, 0, 2)
	for i := range 2 {
		card := NewCreature("Watched", "", 2, 3+i)
		card.SetOwner(b.PlayerID())
		p := g.PutOnBattlefield(card, a.PlayerID())
		subjects = append(subjects, p)
		if err := ApplyEffect(g, WhenDiesThisTurnWithCard(This, permanent, owner, controller, saved, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	g.DestroyPermanents(subjects)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("simultaneous deaths lost pending bodies")
	}
	g.RunStepWithPriority(core.Cleanup)
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 27 || len(g.GetExile()) != 2 {
		t.Fatal("independent death facts overwritten", a.Life(), b.Life())
	}
	if len(g.resolution.cardBindings) != 0 || len(g.resolution.playerBindings) != 0 || len(g.resolution.permanentBindings) != 0 {
		t.Fatal("death frame escaped")
	}
}

func TestDelayedDeathCardReplacementOrder(t *testing.T) {
	for _, index := range []int{0, 1} {
		a := NewBasePlayer("A")
		b := &deathExileOrderPlayer{BasePlayer: NewBasePlayer("B"), index: index}
		g := NewGame(a, b)
		card := NewCreature("Watched", "", 2, 3)
		card.SetOwner(b.PlayerID())
		p := g.PutOnBattlefield(card, a.PlayerID())
		saved := BindCard("dead card")
		if err := ApplyEffect(g, deathCardEffect(This, saved, ExileGraveyardCard(FromCard(saved))), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		cancel, hand := &deathCardMoveReplacement{cancel: true}, &deathCardMoveReplacement{}
		g.AddReplacementEffect(cancel)
		g.AddReplacementEffect(hand)
		g.PutPermanentIntoGraveyard(p)
		g.PutTriggersOnStack()
		g.ResolveStack()
		if b.choices != 1 || cancel.calls+hand.calls != 1 {
			t.Fatal("replacement order did not use affected owner")
		}
		if (len(b.Hand()) == 1) != (index == 1) || (len(b.Graveyard()) == 1) != (index == 0) {
			t.Fatal("wrong replacement chosen")
		}
	}
}
