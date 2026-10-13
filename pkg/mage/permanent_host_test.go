package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentHost(t *testing.T) {
	for _, change := range []string{"current", "move", "detach", "attachment-leaves", "attachment-returns", "host-leaves", "host-returns", "both-leave", "both-return", "host-phase", "phase-cycle", "unknown"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			hostCard := NewCreature("Host", "", 3, 8)
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other", "", 7, 8), b.PlayerID())
			attachmentCard := NewArtifact("Attachment", "", WithSubTypes("Equipment"))
			attachment := g.PutOnBattlefield(attachmentCard, a.PlayerID())
			g.Attach(attachment.ID(), host.ID())
			target := DeclarePermanentTarget("attachment", AnyPermanent)
			ctx := &EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{attachment.ID(), attachment.incarnationID}}}}
			ref := HostOfPermanent(target)
			wantPower, wantPresent := 3, true
			switch change {
			case "move":
				g.Attach(attachment.ID(), other.ID())
				wantPower = 7
			case "detach":
				g.MutablePermanent(attachment.ID()).AttachedTo = uuid.Nil
				g.MutablePermanent(attachment.ID()).attachedIncarnation = uuid.Nil
				wantPower, wantPresent = 0, false
			case "attachment-leaves", "attachment-returns":
				g.RemoveFromBattlefield(attachment)
				if change == "attachment-returns" {
					returned := g.PutOnBattlefield(attachmentCard, a.PlayerID())
					g.Attach(returned.ID(), other.ID())
				}
			case "host-leaves", "host-returns":
				g.RemoveFromBattlefield(host)
				wantPower, wantPresent = 0, false
				if change == "host-returns" {
					g.PutOnBattlefield(hostCard, b.PlayerID()).AddCounter(core.P1P1, 4)
				}
			case "both-leave", "both-return":
				g.RemoveFromBattlefield(attachment)
				g.RemoveFromBattlefield(host)
				wantPresent = false
				if change == "both-return" {
					returned := g.PutOnBattlefield(attachmentCard, a.PlayerID())
					g.PutOnBattlefield(hostCard, b.PlayerID()).AddCounter(core.P1P1, 4)
					g.Attach(returned.ID(), other.ID())
				}
			case "host-phase":
				g.PhaseOut(host.ID())
				wantPresent = false
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "unknown":
				ctx.targetBindings[target.targetDeclaration] = []capturedPermanent{{attachment.ID(), uuid.New()}}
				wantPower, wantPresent = 0, false
			}
			for _, game := range []*Game{g, g.Clone()} {
				local := *ctx
				local.Game = game
				game.SetRandomResults([]int{6})
				if got := PowerOfWithLKI(ref).resolveValue(&local); got != wantPower {
					t.Fatalf("power = %d, want %d", got, wantPower)
				}
				if _, ok := ref.resolvePermanent(&local); ok != wantPresent {
					t.Fatalf("present = %v, want %v", ok, wantPresent)
				}
				if game.RandIntn(10) != 6 {
					t.Fatal("host query used RNG")
				}
			}
		})
	}
}

func TestPermanentHostDeclarations(t *testing.T) {
	target := DeclarePermanentTarget("attachment", AnyPermanent)
	for _, ref := range []PermanentRef{nil, From(BindPermanent("missing")), target} {
		card := NewInstant("Invalid", "", Spell(TapSubject(HostOfPermanent(ref))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid host reference accepted")
		}
	}
	card := NewInstant("Valid", "", Spell(TapSubject(HostOfPermanent(target)), WithTargets(target)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestPermanentHostReferencesAndWaiting(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host := g.PutOnBattlefield(NewCreature("Host", "", 3, 8), b.PlayerID())
	attachment := g.PutOnBattlefield(NewArtifact("Attachment", "", WithSubTypes("Equipment")), a.PlayerID())
	g.Attach(attachment.ID(), host.ID())
	target := DeclarePermanentTarget("attachment", AnyPermanent)
	saved := BindPermanent("attachment")
	ctx := &EffectContext{Game: g, SourceID: attachment.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: attachment.incarnationID, targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{attachment.ID(), attachment.incarnationID}}}}
	if err := Snapshot(target, saved).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	for _, ref := range []PermanentRef{target, This, From(saved)} {
		if got := PowerOfWithLKI(HostOfPermanent(ref)).resolveValue(ctx); got != 3 {
			t.Fatalf("host power = %d", got)
		}
	}
	reads := 0
	filter := NewPermanentFilter("query guard", func(*Permanent, *Game) bool { reads++; return true })
	body := TapEach(PermanentsSharingCardTypes(HostOfPermanent(This), MatchingPermanents(filter)))
	if err := AtNext(EachEndStep, body).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 || host.Tapped {
		t.Fatal("waiting query lost source or executed host body")
	}
}
