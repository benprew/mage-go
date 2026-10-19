package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAttackerDefendingPlayer(t *testing.T) {
	for _, state := range []string{"attacking", "not-attacking", "phase", "phase-cycle", "return", "control", "combat-end", "planeswalker", "planeswalker-exit"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			g.SetStep(core.DeclareAttackers)
			card := NewCreature("Attacker", "", 2, 8)
			p := g.PutOnBattlefield(card, a.PlayerID())
			defender := b.PlayerID()
			var walker *Permanent
			if state == "planeswalker" || state == "planeswalker-exit" {
				walker = g.PutOnBattlefield(NewArtifact("Walker", "", WithCardType(core.TypePlaneswalker)), b.PlayerID())
				defender = walker.ID()
			}
			g.GetCombat().AddAttackerIncarnation(p.ID(), p.incarnationID, defender)
			ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
			clone := g.Clone()
			switch state {
			case "not-attacking":
				g.GetCombat().RemoveFromCombat(p.ID())
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.RemoveFromBattlefield(p)
				returned := g.PutOnBattlefield(card, a.PlayerID())
				g.GetCombat().AddAttackerIncarnation(returned.ID(), returned.incarnationID, b.PlayerID())
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "combat-end":
				g.SetStep(core.PostcombatMain)
			case "planeswalker-exit":
				g.RemoveFromBattlefield(walker)
			}
			ref := DefendingPlayerOf(This)
			want := state == "attacking" || state == "planeswalker" || state == "planeswalker-exit"
			id, ok := ref.resolvePlayer(ctx)
			if ok != want || (ok && id != b.PlayerID()) || (!ok && id != uuid.Nil) {
				t.Fatalf("defender=%v available=%v want=%v", id, ok, want)
			}
			if recipient, available := ref.resolveRecipient(ctx); recipient != id || available != ok {
				t.Fatal("recipient differs from player")
			}
			local := *ctx
			local.Game = clone
			if id, ok := ref.resolvePlayer(&local); !ok || id != b.PlayerID() {
				t.Fatal("original mutation changed clone query")
			}
		})
	}
}

func TestAttackerDefendingPlayerValidation(t *testing.T) {
	for _, ref := range []PermanentRef{nil, From(BindPermanent("unbound")), DeclarePermanentTarget("unbound", AnyPermanent)} {
		card := NewInstant("Invalid", "", Spell(GainLifeFor(DefendingPlayerOf(ref), Fixed(1))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	if !playerRefReadsSourcePermanent(DefendingPlayerOf(This)) {
		t.Fatal("waiting query lost source read")
	}
}

func TestAttackerDefendingPlayerWaiting(t *testing.T) {
	g, a, b := newSpellCopyGame()
	g.SetStep(core.DeclareAttackers)
	card := NewCreature("Observer", "", 2, 8, WithAbility(Activate(Costs(), AtNext(EachEndStep, GainLifeFor(DefendingPlayerOf(This), Fixed(2))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefieldAttacking(card, a.PlayerID(), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	for range 2 {
		if got := g.WaitingDelayedPermanentSources(); len(got) != 1 || got[0].ID() != p.ID() {
			t.Fatal("waiting query lost source")
		}
		if a.Life() != 20 || b.Life() != 20 {
			t.Fatal("waiting query executed body")
		}
	}
	clone := g.Clone()
	g.PhaseOut(p.ID())
	g.SetStep(core.EndStep)
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if b.Life() != 20 {
		t.Fatal("absent attacker supplied a player")
	}
	if len(clone.WaitingDelayedPermanentSources()) != 1 {
		t.Fatal("original consumed clone trigger")
	}
}
