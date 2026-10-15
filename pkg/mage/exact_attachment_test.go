package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestExactAttachmentContracts(t *testing.T) {
	for _, mode := range []string{"normal", "shroud", "hexproof", "protection", "restriction", "aura-phase", "host-phase", "phase-cycle", "aura-return", "host-return", "self", "same", "nonattachment"} {
		t.Run(mode, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			old := g.PutOnBattlefield(NewCreature("Old", "", 2, 4), a.PlayerID())
			guestCard := NewCreature("Guest", "", 3, 4)
			guest := g.PutOnBattlefield(guestCard, b.PlayerID())
			auraCard := NewAura("Aura", "{B}", WithCastTarget(TargetCreature()))
			if mode == "nonattachment" {
				auraCard = NewArtifact("Artifact", "")
			}
			aura := g.PutOnBattlefield(auraCard, a.PlayerID())
			g.Attach(aura.ID(), old.ID())
			savedAura, savedHost := BindPermanent("aura"), BindPermanent("host")
			ctx := &EffectContext{Game: g, SourceID: aura.ID(), Controller: a.PlayerID()}
			if err := Snapshot(This, savedAura).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			ctx.SourceID = guest.ID()
			if err := Snapshot(This, savedHost).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			want := guest.ID()
			switch mode {
			case "shroud":
				guest.GrantBaseAttr(core.Shroud)
			case "hexproof":
				guest.GrantBaseAttr(core.Hexproof)
			case "protection":
				if err := ApplyEffect(g, Until(core.EndOfTurn, GrantProtectionColors(This, core.Black)), guest.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				want = old.ID()
			case "restriction":
				guest.GrantBaseAttr(core.AttrCantBeEnchanted)
				want = old.ID()
			case "aura-phase":
				g.PhaseOut(aura.ID())
				want = old.ID()
			case "host-phase":
				g.PhaseOut(guest.ID())
				want = old.ID()
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(guest.ID()))
			case "aura-return":
				g.RemoveFromBattlefield(aura)
				returned := g.PutOnBattlefield(auraCard, a.PlayerID())
				want = returned.AttachedTo
			case "host-return":
				g.RemoveFromBattlefield(guest)
				g.PutOnBattlefield(guestCard, b.PlayerID())
				want = old.ID()
			case "self":
				savedHost = savedAura
				want = old.ID()
			case "same":
				g.Attach(aura.ID(), guest.ID())
			case "nonattachment":
				want = aura.AttachedTo
			}
			body := AttachToPermanent(From(savedAura), From(savedHost))
			clone := g.Clone()
			for _, game := range []*Game{g, clone} {
				if err := body.Apply(&EffectContext{Game: game, Controller: a.PlayerID()}); err != nil {
					t.Fatal(err)
				}
				if p := game.FindPermanentIncludingPhased(aura.ID()); p == nil || p.AttachedTo != want {
					t.Fatalf("wrong attachment: %v want %v", p, want)
				}
			}
		})
	}
}

func TestExactAttachmentDefinition(t *testing.T) {
	for _, body := range []ResolvingEffect{AttachToPermanent(nil, This), AttachToPermanent(This, nil), AttachToPermanent(This, From(BindPermanent("missing")))} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), body)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid attachment accepted")
		}
	}
	card := NewArtifact("Valid", "", WithAbility(Activate(Costs(), AttachToPermanent(This, This))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestExactAttachmentWaitingAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	old := g.PutOnBattlefield(NewCreature("Old", "", 2, 4), a.PlayerID())
	guestCard := NewCreature("Guest", "", 3, 4)
	guest := g.PutOnBattlefield(guestCard, b.PlayerID())
	aura := g.PutOnBattlefield(NewAura("Aura", "{B}", WithCastTarget(TargetCreature())), a.PlayerID())
	g.Attach(aura.ID(), old.ID())
	saved := BindPermanent("guest")
	ctx := &EffectContext{Game: g, SourceID: guest.ID(), Controller: a.PlayerID()}
	if err := Snapshot(This, saved).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	ctx.SourceID = aura.ID()
	if err := AtNext(EachEndStep, AttachToPermanent(This, From(saved))).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if refs := g.WaitingDelayedPermanentSources(); len(refs) != 2 {
			t.Fatalf("waiting sources=%d want 2", len(refs))
		}
		if g.FindPermanent(aura.ID()).AttachedTo != old.ID() {
			t.Fatal("waiting query moved attachment")
		}
	}
	clone := g.Clone()
	g.RemoveFromBattlefield(guest)
	g.PutOnBattlefield(guestCard, b.PlayerID())
	for _, game := range []*Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
	}
	if g.FindPermanent(aura.ID()).AttachedTo != old.ID() || clone.FindPermanent(aura.ID()).AttachedTo != guest.ID() {
		t.Fatal("delayed attachment lost exact visits or clone isolation")
	}
}
