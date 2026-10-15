package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSacrificeForContracts(t *testing.T) {
	for _, mode := range []string{"normal", "wrong-player", "changed", "phased", "cycle", "returned", "indestructible", "regeneration"} {
		t.Run(mode, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Guest", "", 3, 4)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, b.PlayerID())
			saved := BindPermanent("guest")
			ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID()}
			if err := Snapshot(This, saved).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			player := ControllerOfWithLKI(From(saved))
			switch mode {
			case "wrong-player":
				player = You
			case "changed":
				g.AddControlEffect(ControlEffectSpec{TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "phased":
				g.PhaseOut(p.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "returned":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(card, b.PlayerID())
			case "indestructible":
				p.GrantBaseAttr(core.Indestructible)
			case "regeneration":
				g.AddRegenerationShield(p.ID())
			}
			effect := SacrificeFor(player, From(saved))
			clone := g.Clone()
			for _, game := range []*Game{g, clone} {
				if err := effect.Apply(&EffectContext{Game: game, Controller: a.PlayerID()}); err != nil {
					t.Fatal(err)
				}
				want := mode == "wrong-player" || mode == "phased" || mode == "returned"
				if (game.FindPermanentIncludingPhased(p.ID()) != nil) != want {
					t.Fatal("sacrifice violated player or visit")
				}
				if !want && len(game.GetPlayer(a.PlayerID()).Graveyard()) != 1 {
					t.Fatal("sacrifice lost owner")
				}
			}
		})
	}
}

func TestSacrificeForDefinition(t *testing.T) {
	for _, body := range []ResolvingEffect{SacrificeFor(nil, This), SacrificeFor(You, nil), SacrificeFor(FromPlayer(BindPlayer("missing")), This), SacrificeFor(You, From(BindPermanent("missing")))} {
		card := NewArtifact("Bad", "", WithAbility(Activate(Costs(), body)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	card := NewArtifact("Good", "", WithAbility(Activate(Costs(), SacrificeFor(You, This))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestSacrificeForDelayedReads(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	target := BindPermanent("guest")
	guest := g.PutOnBattlefield(NewArtifact("Guest", ""), a.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: guest.ID(), Controller: a.PlayerID()}
	if err := Snapshot(This, target).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	ctx.SourceID = source.ID()
	body := AtNext(EachEndStep, SacrificeFor(ControllerOfWithLKI(This), From(target)))
	if err := body.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		refs := g.WaitingDelayedPermanentSources()
		if len(refs) != 2 || g.FindPermanent(guest.ID()) == nil {
			t.Fatalf("query lost source or made sacrifice: %d references", len(refs))
		}
	}
	g.AddControlEffect(ControlEffectSpec{TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	clone := g.Clone()
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.ResolveStack()
	if g.FindPermanent(guest.ID()) == nil {
		t.Fatal("wrong player sacrificed")
	}
	clone.AddControlEffect(ControlEffectSpec{TargetID: guest.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	clone.ResolveStack()
	if clone.FindPermanent(guest.ID()) != nil {
		t.Fatal("clone lost player read")
	}
}
