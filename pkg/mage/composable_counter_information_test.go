package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestCounterInformationLifecycle(t *testing.T) {
	for _, reference := range []string{"source", "target", "saved", "attachment"} {
		for _, change := range []string{"live", "change", "leave", "return", "second-departure", "phase", "phase-cycle", "unknown"} {
			t.Run(reference+"/"+change, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				c := NewCreature("Subject", "", 2, 8)
				c.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(c, a.PlayerID())
				p.AddCounter(Charge, 3)
				p.AddCounter(Time, 9)
				target := DeclarePermanentTarget("subject", AnyPermanent)
				saved := BindPermanent("saved")
				ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: p.ID(), sourceCaptured: true, sourceIncarnation: p.incarnationID, targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{p.ID(), p.incarnationID}}}}
				if err := Snapshot(target, saved).Apply(ctx); err != nil {
					t.Fatal(err)
				}
				var subject PermanentRef = This
				switch reference {
				case "target":
					subject = target
				case "saved":
					subject = From(saved)
				case "attachment":
					aura := g.PutOnBattlefield(NewEnchantment("Aura", "", WithSubTypes("Aura")), a.PlayerID())
					g.Attach(aura.ID(), p.ID())
					ctx.SourceID, ctx.sourceIncarnation = aura.ID(), aura.incarnationID
					subject = AttachedPermanent
				}
				want := 3
				switch change {
				case "change":
					g.MutablePermanent(p.ID()).AddCounter(Charge, 2)
					want = 5
				case "leave", "return", "second-departure":
					g.BouncePermanentToHand(g.FindPermanent(p.ID()))
					if change != "leave" {
						a.RemoveFromHand(c.ID())
						q := g.PutOnBattlefield(c, b.PlayerID())
						q.AddCounter(Charge, 11)
						if change == "second-departure" {
							g.BouncePermanentToHand(q)
						}
					}
				case "phase":
					g.PhaseOut(p.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
					g.MutablePermanent(p.ID()).AddCounter(Charge, 2)
					want = 5
				case "unknown":
					ctx.sourceIncarnation = uuid.New()
					ctx.targetBindings[target.targetDeclaration][0].incarnation = uuid.New()
					g.resolution.permanentBindings[saved] = capturedPermanent{p.ID(), uuid.New()}
					want = 0
				}
				clone := g.Clone()
				for _, game := range []*Game{g, clone} {
					local := *ctx
					local.Game = game
					if got := CountersOnWithLKI(subject, Charge).resolveValue(&local); got != want {
						t.Fatalf("count=%d want=%d", got, want)
					}
					if got := CountersOnWithLKI(subject, Time).resolveValue(&local); got != 9 && change != "unknown" {
						t.Fatalf("other count=%d", got)
					}
					if change == "leave" || change == "return" || change == "second-departure" || change == "phase" {
						if CountersOn(subject, Charge).resolveValue(&local) != 0 {
							t.Fatal("live-only value changed")
						}
						if _, ok := subject.resolvePermanent(&local); ok {
							t.Fatal("information restored absent subject")
						}
					}
				}
				if change == "live" {
					g.MutablePermanent(p.ID()).AddCounter(Charge, 1)
					local := *ctx
					local.Game = clone
					if CountersOnWithLKI(subject, Charge).resolveValue(&local) != 3 {
						t.Fatal("clone counters shared")
					}
				}
			})
		}
	}
}

func TestCounterInformationDeclaration(t *testing.T) {
	for _, subject := range []PermanentRef{nil, From(BindPermanent("missing")), DeclarePermanentTarget("missing", AnyPermanent)} {
		c := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), GainLifeFor(You, CountersOnWithLKI(subject, Charge)))))
		if err := ValidateCardDefinition(c.Name(), c); err == nil {
			t.Fatal("invalid information subject accepted")
		}
	}
	for _, counter := range []CounterType{CounterType(-1), NumCounters} {
		c := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), GainLifeFor(You, CountersOnWithLKI(This, counter)))))
		if err := ValidateCardDefinition(c.Name(), c); err == nil {
			t.Fatal("invalid counter accepted")
		}
	}
}

func TestCounterInformationBoundsAndPurity(t *testing.T) {
	for _, counter := range []CounterType{Charge, P1P1, Pin, Storage, Wind, Credit, Doom} {
		for _, count := range []uint8{0, 1, 255} {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(NewArtifact("Subject", ""), a.PlayerID())
			g.MutablePermanent(p.ID()).Counters[counter] = count
			ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: p.ID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
			value := CountersOnWithLKI(This, counter)
			for range 3 {
				if value.resolveValue(ctx) != int(count) {
					t.Fatal("counter count wrapped or changed")
				}
			}
			if g.FindPermanent(p.ID()).Counters[counter] != count || g.StackSize() != 0 || a.Life() != 20 {
				t.Fatal("query changed game")
			}
		}
	}
}
