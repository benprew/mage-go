package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedRedirectOperands(t *testing.T) {
	for name, body := range map[string]ResolvingEffect{
		"selected-source":         RedirectNextDamageFromSelectedSource(This, AttachedPermanent, You, DamageSourceFilter{}),
		"selected-wrapped-source": RedirectNextDamageFromSelectedSource(DamageSourceOf(This), AttachedPermanent, You, DamageSourceFilter{}),
		"selected-subject":        RedirectNextDamageFromSelectedSource(DamageSourceOf(AttachedPermanent), This, You, DamageSourceFilter{}),
		"selected-controller":     RedirectNextDamageFromSelectedSource(DamageSourceOf(AttachedPermanent), AttachedPermanent, ControllerOfWithLKI(This), DamageSourceFilter{}),
		"selected-owner":          RedirectNextDamageFromSelectedSource(DamageSourceOf(AttachedPermanent), AttachedPermanent, OwnerOf(This), DamageSourceFilter{}),
		"permanent-source":        RedirectDamageFromPermanentThisTurn(This, You, AttachedPermanent),
		"permanent-destination":   RedirectDamageFromPermanentThisTurn(AttachedPermanent, You, This),
		"permanent-player":        RedirectDamageFromPermanentThisTurn(AttachedPermanent, ControllerOfWithLKI(This), AttachedPermanent),
	} {
		for _, state := range []string{"live", "phased", "cycle", "departed", "returned", "control"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				card := NewCreature("Source", "{U}", 3, 20)
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				saved := g.CaptureDamageSource(p.ID())
				delay := AtNext(EachEndStep, Sequence(body))
				if err := ValidateCardDefinition("Delay", NewArtifact("Delay", "", WithAbility(Activate(Costs(), delay)))); err != nil {
					t.Fatal(err)
				}
				if err := ApplyEffect(g, delay, p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				switch state {
				case "phased":
					g.PhaseOut(p.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "departed", "returned":
					g.BouncePermanentToHand(p)
					g.zones.ClearLKI()
					if state == "returned" {
						a.RemoveFromHand(card.ID())
						g.PutOnBattlefield(card, b.PlayerID())
						g.ChangeObjectColor(card.ID(), core.Red)
					}
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				for range 3 {
					got := g.WaitingDelayedPermanentSources()
					if len(got) != 1 || !got[0].SameObject(saved) || !got[0].HasColor(core.Blue) {
						t.Fatal("missing exact redirection operand", got)
					}
					if state == "control" && got[0].Controller() != b.PlayerID() {
						t.Fatal("lost current controller")
					}
					got[0].card.AddType(core.TypeEnchantment)
					if g.WaitingDelayedPermanentSources()[0].HasType(core.TypeEnchantment) {
						t.Fatal("query shares mutable facts")
					}
				}
				if len(g.WaitingRedirectionPermanentSources()) != 0 || a.Life() != 20 {
					t.Fatal("query installed redirection")
				}
				clone := g.Clone()
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				if len(g.WaitingDelayedPermanentSources()) != 0 || len(clone.WaitingDelayedPermanentSources()) != 1 {
					t.Fatal("waiting lifetime shared")
				}
				g.PutTriggersOnStack()
				g.ResolveStack()
			})
		}
	}
}

func TestWaitingDelayedRedirectScopeAndPurity(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	p := g.PutOnBattlefield(NewCreature("Source", "", 1, 20), a.PlayerID())
	calls := 0
	player := waitingRedirectPlayer{PlayerRef: You, calls: &calls}
	filter := DamageSourceFilter{}
	if err := ApplyEffect(g, AtNext(EachEndStep, RedirectNextDamageFromSelectedSource(This, This, player, filter)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || calls != 0 {
			t.Fatal("query resolved player")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if calls != 1 {
		t.Fatal("player not resolved at installation", calls)
	}
	for _, body := range []ResolvingEffect{
		RedirectNextDamageFromSelectedSource(DamageSourceOf(AttachedPermanent), AttachedPermanent, You, DamageSourceFilter{}),
		RedirectDamageFromPermanentThisTurn(AttachedPermanent, You, AttachedPermanent),
		AtNext(EachEndStep, RedirectNextDamageFromSelectedSource(This, This, You, DamageSourceFilter{})),
	} {
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("implicit or future reference inferred")
	}
}

type waitingRedirectPlayer struct {
	PlayerRef
	calls *int
}

func (p waitingRedirectPlayer) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	*p.calls++
	return p.PlayerRef.resolvePlayer(ctx)
}
