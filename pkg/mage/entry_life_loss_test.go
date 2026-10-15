package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestEntryLifeLossTiming(t *testing.T) {
	for _, life := range []int{20, 7, 0, -3} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		a.SetLife(life)
		losses, entries := 0, 0
		card := NewArtifact("Entry loss", "", WithAbility(LoseLifeAsEnters(LifeTotalOf(You))), WithAbility(WheneverYouLoseLifeTrigger(GainLifeFor(You, Fixed(5)), false)))
		card.SetOwner(b.PlayerID())
		g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(NewTriggered(core.EvtLifeLost, false).SetCondition(func(event *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
			losses++
			if event.PlayerID != a.PlayerID() || event.Amount != life || a.Life() != 0 {
				t.Fatal("wrong life loss event")
			}
			for _, p := range reader.AllBattlefield() {
				if p.ID() == card.ID() {
					t.Fatal("loss occurred after entry")
				}
			}
			return false
		})), WithAbility(NewTriggered(core.EvtZoneChange, false).SetCondition(func(event *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
			if event.SourceID == card.ID() && event.ToZone == core.ZoneBattlefield {
				entries++
				if a.Life() != min(life, 0) {
					t.Fatal("entry observed old life")
				}
			}
			return false
		}))), a.PlayerID())
		g.AddPreventionShield(a.PlayerID(), 100)
		p := g.PutOnBattlefield(card, a.PlayerID())
		wantLosses := 0
		if life > 0 {
			wantLosses = 1
		}
		if p == nil || entries != 1 || losses != wantLosses || a.Life() != min(life, 0) || b.Life() != 20 || g.StackSize() != 0 {
			t.Fatal("entry loss timing or controller", entries, losses, a.Life())
		}
	}
}

func TestEntryLifeLossReplacementOrder(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		for _, mode := range []int{0, 1} {
			a := &entryLifePlayer{BasePlayer: NewBasePlayer("A"), mode: mode}
			g := NewGame(a, NewBasePlayer("B"))
			seen := -1
			g.effects.AddReplacement(&testPrioritizedReplacement{matchFunc: func(action Action) bool { _, ok := action.(*battlefieldEntryAction); return ok }, replaceFunc: func(action Action, _ *Game) Action {
				seen = a.Life()
				if cancel {
					return nil
				}
				a.SetLife(a.Life() + 3)
				return action
			}})
			p := g.PutOnBattlefield(NewArtifact("Loss", "", WithAbility(LoseLifeAsEnters(LifeTotalOf(You)))), a.PlayerID())
			want, wantSeen := 0, 20
			if mode == 1 {
				want, wantSeen = 3, 0
			}
			if cancel {
				want = wantSeen
			}
			if a.Life() != want || seen != wantSeen || (p == nil) != cancel || a.orders != 1 {
				t.Fatal("replacement order, current value or cancellation", mode, cancel, a.Life(), seen, a.orders)
			}
		}
	}
}

func TestEntryLifeLossCopyReturnAndClone(t *testing.T) {
	a := &optionalEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	plan := entryCopyPlan(t, "Loss donor", func() []CardOption { return []CardOption{WithAbility(LoseLifeAsEnters(Fixed(3)))} })
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	donor := g.PutOnBattlefield(card, b.PlayerID())
	copyPlan := entryCopyPlan(t, "Copy", func() []CardOption {
		return []CardOption{WithAbility(MayCopyAsEnters(IsCreature)), WithAbility(LoseLifeAsEnters(Fixed(8)))}
	})
	copyCard, err := copyPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(copyCard, a.PlayerID())
	if p.Name() != donor.Name() || a.Life() != 17 || b.Life() != 17 {
		t.Fatal("copy did not replace entry loss")
	}
	g.PhaseIn(g.PhaseOut(p.ID()))
	if a.Life() != 17 {
		t.Fatal("phase cycle repeated loss")
	}
	clone := g.Clone()
	g.BouncePermanentToHand(p)
	a.RemoveFromHand(copyCard.ID())
	if g.PutOnBattlefield(copyCard, a.PlayerID()) == nil || a.Life() != 14 || clone.GetPlayer(a.PlayerID()).Life() != 17 {
		t.Fatal("return or clone isolation")
	}
}

func TestEntryLifeLossInvalid(t *testing.T) {
	for _, card := range []Card{
		NewInstant("Bad", "", WithAbility(LoseLifeAsEnters(Fixed(2)))),
		NewArtifact("Bad", "", WithAbility(LoseLifeAsEnters(nil))),
		NewArtifact("Bad", "", WithAbility(LoseLifeAsEnters(LifeTotalOf(nil)))),
		NewArtifact("Bad", "", WithAbility(LoseLifeAsEnters(LifeTotalOf(FromPlayer(BindPlayer("unbound")))))),
	} {
		if err := ValidateCardDefinition("Bad", card); err == nil {
			t.Fatal("invalid entry loss accepted")
		}
	}
}

func TestEntryLifeLossBatchAndIndependentRules(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := NewCreature("First", "", 2, 2, WithAbility(LoseLifeAsEnters(Fixed(2))), WithAbility(LoseLifeAsEnters(Fixed(3))))
	second := NewCreature("Second", "", 2, 2, WithAbility(LoseLifeAsEnters(Fixed(4))))
	first.SetOwner(a.PlayerID())
	second.SetOwner(b.PlayerID())
	a.AddToGraveyard(first)
	b.AddToGraveyard(second)
	entries := 0
	g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(NewTriggered(core.EvtZoneChange, false).SetCondition(func(event *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		if event.ToZone == core.ZoneBattlefield && (event.SourceID == first.ID() || event.SourceID == second.ID()) {
			entries++
			if reader.FindPermanent(first.ID()) == nil || reader.FindPermanent(second.ID()) == nil || a.Life() != 15 || b.Life() != 16 {
				t.Fatal("batch observation preceded losses or registration")
			}
		}
		return false
	}))), a.PlayerID())
	if err := ApplyEffect(g, PutGraveyardCreaturesOntoBattlefieldUnderOwnersControl(), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if entries != 2 || a.Life() != 15 || b.Life() != 16 {
		t.Fatal("batch or independent loss rules")
	}
}
