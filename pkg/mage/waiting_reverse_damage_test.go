package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingReverseDamageReferences(t *testing.T) {
	for _, state := range []string{"live", "depart", "return", "phase", "cycle", "control", "color", "consume", "cleanup"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Subject", "{R}", 1, 20)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			g.AddReverseDamageShield(b.PlayerID(), p.ID())
			before := g.Clone()
			switch state {
			case "depart", "return":
				g.RemoveFromBattlefield(p)
				if state == "return" {
					g.PutOnBattlefield(card, a.PlayerID())
				}
				g.zones.ClearLKI()
			case "phase":
				g.PhaseOut(p.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "color":
				if err := ApplyEffect(g, Until(core.Indefinite, SetColors(This, core.Blue)), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "consume":
				g.DealDamageToPlayerFromSource(b, 4, saved)
				if b.Life() != 24 || g.PlayerLifeGainedThisTurn(b.PlayerID()) != 4 || g.HasDealtDamageToPlayer(p.ID(), b.PlayerID()) {
					t.Fatal("reverse damage outcome or history")
				}
			case "cleanup":
				g.effects.ClearReplacementsEndOfTurn()
			}
			for _, game := range []*Game{g, g.Clone()} {
				refs := game.WaitingPreventionPermanentSources()
				want := 1
				if state == "consume" || state == "cleanup" {
					want = 0
				}
				if len(refs) != want {
					t.Fatalf("sources=%d want=%d", len(refs), want)
				}
				if want == 1 {
					if !refs[0].SameObject(saved) {
						t.Fatal("wrong incarnation")
					}
					if state == "color" && (!refs[0].HasColor(core.Blue) || refs[0].HasColor(core.Red)) {
						t.Fatal("stale colors")
					}
					if state == "control" && refs[0].Controller() != b.PlayerID() {
						t.Fatal("stale controller")
					}
					refs[0].card.SetOwner(b.PlayerID())
					if game.WaitingPreventionPermanentSources()[0].Owner() != a.PlayerID() {
						t.Fatal("query changed stored data")
					}
					life := game.GetPlayer(b.PlayerID()).Life()
					if state == "return" {
						game.DealDamageToPlayerFromSource(game.GetPlayer(b.PlayerID()), 4, game.CaptureDamageSource(p.ID()))
						if game.GetPlayer(b.PlayerID()).Life() != life-4 {
							t.Fatal("returned source inherited shield")
						}
						life -= 4
					}
					game.DealDamageToPlayerFromSource(game.GetPlayer(b.PlayerID()), 4, saved)
					if game.GetPlayer(b.PlayerID()).Life() != life+4 {
						t.Fatal("original source lost reverse shield")
					}
					if len(game.WaitingPreventionPermanentSources()) != 0 {
						t.Fatal("shield not consumed")
					}
				}
			}
			if len(before.WaitingPreventionPermanentSources()) != 1 || before.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("clone shared lifetime")
			}
		})
	}
}

func TestWaitingReverseDamageBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Subject", "{R}", 1, 20), a.PlayerID())
	saved := g.CaptureDamageSource(p.ID())
	g.AddReverseDamageShield(b.PlayerID(), p.ID())
	rule := g.effects.replacements[0]
	for _, amount := range []int{0, -1} {
		if rule.Matches(NewDamageToPlayerAction(p.ID(), b.PlayerID(), amount, false).withDamageSource(saved), g) {
			t.Fatal("nonpositive damage matches")
		}
	}
	if rule.Matches(NewDamageToPlayerAction(p.ID(), a.PlayerID(), 1, false).withDamageSource(saved), g) {
		t.Fatal("wrong recipient matches")
	}
	g.RemoveFromBattlefield(p)
	g.PutOnBattlefield(p.Card, a.PlayerID())
	g.DealDamageToPlayer(b, 3, p.ID())
	if b.Life() != 17 {
		t.Fatal("returned source inherited reverse shield")
	}
	g.DealDamageToPlayerFromSource(b, 2, saved)
	if b.Life() != 19 || len(g.WaitingPreventionPermanentSources()) != 0 {
		t.Fatal("original source lost reverse shield")
	}
	if rule.Matches(NewDamageToPlayerAction(p.ID(), b.PlayerID(), 1, false).withDamageSource(saved), g) {
		t.Fatal("consumed rule matches")
	}
}

func TestWaitingReverseDamageDeduplication(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Subject", "{R}", 1, 20), a.PlayerID())
	g.AddReverseDamageShield(b.PlayerID(), p.ID())
	g.effects.cycleReplacements = append(g.effects.cycleReplacements, g.effects.replacements[0].Clone())
	unknown := uuid.New()
	g.AddReverseDamageShield(b.PlayerID(), unknown)
	g.AddReverseDamageShield(b.PlayerID(), uuid.Nil)
	if len(g.WaitingPreventionPermanentSources()) != 1 || len(g.effects.replacements) != 2 {
		t.Fatal("duplicate, unknown or nil reference")
	}
	g.DealDamageToPlayer(b, 2, unknown)
	if b.Life() != 22 {
		t.Fatal("legacy unknown-ID matching changed")
	}
}
