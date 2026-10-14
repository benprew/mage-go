package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestBandedSetIdentity(t *testing.T) {
	for _, change := range []string{"none", "ability-loss", "source-return", "member-return", "source-phase", "member-phase-cycle", "remove", "stale-member", "stale-source"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			sourceCard := NewCreature("Source", "", 1, 10, WithKeyword(core.Banding))
			memberCard := NewCreature("Member", "", 1, 10)
			source := g.PutOnBattlefield(sourceCard, a.PlayerID())
			member := g.PutOnBattlefield(memberCard, a.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other", "", 1, 10), a.PlayerID())
			for _, p := range []*Permanent{source, member, other} {
				g.combat.AddAttackerIncarnation(p.ID(), p.incarnationID, b.PlayerID())
			}
			if err := g.FormAttackingBand([]uuid.UUID{source.ID(), member.ID()}); err != nil {
				t.Fatal(err)
			}
			ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceIncarnation: source.incarnationID, sourceCaptured: true}
			set := BandedWith(This)
			clone := g.Clone()
			switch change {
			case "ability-loss":
				g.MutablePermanent(source.ID()).RevokeBaseAttr(core.Banding)
			case "source-return":
				g.RemoveFromBattlefield(source)
				g.PutOnBattlefield(sourceCard, a.PlayerID())
			case "member-return":
				g.RemoveFromBattlefield(member)
				g.PutOnBattlefield(memberCard, a.PlayerID())
			case "source-phase":
				g.PhaseOut(source.ID())
			case "member-phase-cycle":
				g.PhaseIn(g.PhaseOut(member.ID()))
			case "remove":
				g.RemoveFromCombat(member.ID())
			case "stale-member":
				g.combat.AttackerIncarnations[member.ID()] = uuid.New()
			case "stale-source":
				g.combat.AttackerIncarnations[source.ID()] = uuid.New()
			}
			want := 2
			if change == "source-return" || change == "source-phase" || change == "stale-source" {
				want = 0
			} else if change != "none" && change != "ability-loss" {
				want = 1
			}
			got := set.resolvePermanents(ctx)
			if len(got) != want {
				t.Fatalf("members=%d want=%d", len(got), want)
			}
			for _, p := range got {
				if p.ID() == other.ID() {
					t.Fatal("unrelated attacker selected")
				}
			}
			cloneCtx := *ctx
			cloneCtx.Game = clone
			if len(set.resolvePermanents(&cloneCtx)) != 2 {
				t.Fatal("original changed clone membership")
			}
		})
	}
}

func TestBandedSetDefinitions(t *testing.T) {
	for _, ref := range []PermanentRef{nil, From(BindPermanent("unbound"))} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), TapEach(BandedWith(ref)))))
		if err := ValidateCardDefinition("Invalid", card); err == nil {
			t.Fatal("invalid band subject accepted")
		}
	}
	if !permanentSetReadsSourcePermanent(BandedWith(This)) {
		t.Fatal("waiting query lost explicit source read")
	}
	if len(BandedWith(This).resolvePermanents(nil)) != 0 {
		t.Fatal("missing context supplied members")
	}
}

func TestBandedSetWaiting(t *testing.T) {
	g, a, b := newSpellCopyGame()
	card := NewCreature("Source", "", 1, 10, WithKeyword(core.Banding))
	p := g.PutOnBattlefield(card, a.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID()}
	if len(BandedWith(This).resolvePermanents(ctx)) != 0 {
		t.Fatal("nonattacker supplied a band")
	}
	g.combat.AddAttackerIncarnation(p.ID(), p.incarnationID, b.PlayerID())
	if len(BandedWith(This).resolvePermanents(ctx)) != 0 {
		t.Fatal("unbanded attacker supplied a band")
	}
	if err := g.FormAttackingBand([]uuid.UUID{p.ID()}); err != nil {
		t.Fatal(err)
	}
	if len(BandedWith(This).resolvePermanents(ctx)) != 1 {
		t.Fatal("one-member band lost its member")
	}
	before := g.CaptureDamageSource(p.ID())
	if err := ApplyEffect(g, AtNext(EachEndStep, TapEach(BandedWith(This))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		got := g.WaitingDelayedPermanentSources()
		if len(got) != 1 || !got[0].SameObject(before) || g.FindPermanent(p.ID()).Tapped {
			t.Fatal("waiting query ran the action or lost its source")
		}
	}
	g.RemoveFromBattlefield(p)
	returned := g.PutOnBattlefield(card, b.PlayerID())
	clone := g.Clone()
	if len(clone.WaitingDelayedPermanentSources()) != 1 || !clone.WaitingDelayedPermanentSources()[0].SameObject(before) {
		t.Fatal("waiting source followed a returned visit")
	}
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if clone.FindPermanent(returned.ID()).Tapped || len(clone.WaitingDelayedPermanentSources()) != 0 || len(g.WaitingDelayedPermanentSources()) != 1 {
		t.Fatal("delayed action or clone lifetime followed the wrong visit")
	}
}
