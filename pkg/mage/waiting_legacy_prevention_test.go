package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingLegacyPreventionReferences(t *testing.T) {
	for _, half := range []bool{false, true} {
		for _, state := range []string{"live", "depart", "return", "phase", "cycle", "control", "color", "consume", "cleanup"} {
			t.Run(state+map[bool]string{false: "/finite", true: "/half"}[half], func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				card := NewCreature("Subject", "{R}", 1, 20)
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				saved := g.CaptureDamageSource(p.ID())
				if half {
					g.AddHalfDamageFromSourcePreventionShield(b.PlayerID(), p.ID())
				} else {
					g.AddSourcePreventionShield(b.PlayerID(), p.ID(), 2)
				}
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
						if game.GetPlayer(b.PlayerID()).Life() != life-2 {
							t.Fatal("original source lost prevention")
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
}

func TestWaitingLegacyPreventionBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Subject", "{R}", 1, 20), a.PlayerID())
	saved := g.CaptureDamageSource(p.ID())
	g.AddSourcePreventionShield(b.PlayerID(), p.ID(), 3)
	g.AddHalfDamageFromSourcePreventionShield(b.PlayerID(), p.ID())
	g.AddSourcePreventionShield(b.PlayerID(), uuid.New(), 1)
	g.AddHalfDamageFromSourcePreventionShield(b.PlayerID(), uuid.New())
	g.AddSourcePreventionShield(b.PlayerID(), p.ID(), 0)
	g.effects.cycleReplacements = append(g.effects.cycleReplacements, g.effects.replacements[0].Clone())
	if len(g.WaitingPreventionPermanentSources()) != 1 {
		t.Fatal("duplicate or unknown reference")
	}
	for _, rule := range g.effects.replacements[:2] {
		for _, amount := range []int{0, -1} {
			if rule.Matches(NewDamageToPlayerAction(p.ID(), b.PlayerID(), amount, false).withDamageSource(saved), g) {
				t.Fatal("nonpositive damage matches")
			}
		}
		if rule.Matches(NewDamageToPlayerAction(p.ID(), a.PlayerID(), 1, false).withDamageSource(saved), g) {
			t.Fatal("wrong recipient matches")
		}
	}
	g.effects.cycleReplacements = nil
	g.effects.replacements = g.effects.replacements[:1]
	g.DealDamageToPlayerFromSource(b, 1, saved)
	if b.Life() != 20 || len(g.WaitingPreventionPermanentSources()) != 1 {
		t.Fatal("finite shield lost remainder")
	}
	g.DealDamageToPlayerFromSource(b, 3, saved)
	if b.Life() != 19 || len(g.WaitingPreventionPermanentSources()) != 0 {
		t.Fatal("finite shield remainder")
	}
}

func TestWaitingLegacyPreventionDamageContracts(t *testing.T) {
	t.Run("finite simultaneous", func(t *testing.T) {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		p := g.PutOnBattlefield(NewCreature("Subject", "{R}", 1, 20), a.PlayerID())
		saved := g.CaptureDamageSource(p.ID())
		g.AddSourcePreventionShield(b.PlayerID(), p.ID(), 3)
		before := g.Clone()
		g.DealSimultaneousDamage(PlayerDamageAssignment(b, 2, saved), PlayerDamageAssignment(b, 2, saved))
		if b.Life() != 19 || len(g.WaitingPreventionPermanentSources()) != 0 {
			t.Fatal("finite batch consumption")
		}
		if before.GetPlayer(b.PlayerID()).Life() != 20 || len(before.WaitingPreventionPermanentSources()) != 1 {
			t.Fatal("batch changed clone")
		}
	})
	t.Run("half odd amount", func(t *testing.T) {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		p := g.PutOnBattlefield(NewCreature("Subject", "{R}", 1, 20), a.PlayerID())
		saved := g.CaptureDamageSource(p.ID())
		g.AddHalfDamageFromSourcePreventionShield(b.PlayerID(), p.ID())
		g.DealDamageToPlayerFromSource(b, 5, saved)
		if b.Life() != 17 || len(g.WaitingPreventionPermanentSources()) != 0 {
			t.Fatal("half rounding or consumption")
		}
		g.DealDamageToPlayerFromSource(b, 2, saved)
		if b.Life() != 15 {
			t.Fatal("half shield applied twice")
		}
	})
}
