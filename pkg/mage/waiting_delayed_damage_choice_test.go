package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedDamageChoiceOperands(t *testing.T) {
	for _, operand := range []string{"direct", "wrapped", "controller", "owner", "category-player", "filtered"} {
		for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "control"} {
			t.Run(operand+"/"+state, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				card := NewCreature("Source", "{U}", 1, 20)
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				saved := g.CaptureDamageSource(p.ID())
				binding := BindDamageSource("choice")
				var player PlayerRef = You
				refs := []DamageSourceRef{This}
				switch operand {
				case "wrapped":
					refs = []DamageSourceRef{DamageSourceOf(This), DamageSourceOf(This)}
				case "controller", "category-player":
					player, refs = ControllerOfWithLKI(This), []DamageSourceRef{DamageSourceOf(AttachedPermanent)}
				case "owner":
					player, refs = OwnerOf(This), []DamageSourceRef{DamageSourceOf(AttachedPermanent)}
				}
				body := ChooseDamageSourceFrom(player, refs, binding)
				if operand == "category-player" {
					body = ChooseDamageSourceFromLiveObjects(player, DamageSourceFilter{}, binding)
				}
				if operand == "filtered" {
					body = ChooseDamageSourceMatchingFrom(player, refs, DamageSourceHasColor(core.Red), binding)
				}
				delay := AtNext(EachEndStep, Sequence(body))
				if err := ValidateCardDefinition("Delay", NewArtifact("Delay", "", WithAbility(Activate(Costs(), delay)))); err != nil {
					t.Fatal(err)
				}
				if err := ApplyEffect(g, delay, p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				switch state {
				case "departed", "returned":
					g.BouncePermanentToHand(p)
					g.zones.ClearLKI()
					if state == "returned" {
						a.RemoveFromHand(card.ID())
						g.PutOnBattlefield(card, b.PlayerID())
						g.ChangeObjectColor(card.ID(), core.Red)
					}
				case "phased":
					g.PhaseOut(p.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				for range 3 {
					got := g.WaitingDelayedPermanentSources()
					if len(got) != 1 || !got[0].SameObject(saved) || !got[0].HasColor(core.Blue) {
						t.Fatal("choice omitted exact source operand", got)
					}
					if state == "control" && got[0].Controller() != b.PlayerID() {
						t.Fatal("lost current controller")
					}
					got[0].card.AddType(core.TypeEnchantment)
					if g.WaitingDelayedPermanentSources()[0].HasType(core.TypeEnchantment) {
						t.Fatal("query shared card facts")
					}
				}
				if len(g.resolution.damageSourceBindings) != 0 {
					t.Fatal("query wrote choice")
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

func TestWaitingDelayedDamageChoicePurityAndBoundaries(t *testing.T) {
	a := &sourceChoicePlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	p := g.PutOnBattlefield(NewCreature("Source", "", 1, 20), a.PlayerID())
	calls := 0
	binding := BindDamageSource("choice")
	player := waitingRedirectPlayer{PlayerRef: You, calls: &calls}
	sourceCalls := 0
	observed := waitingChoiceDamageRef{DamageSourceRef: This, calls: &sourceCalls}
	body := ChooseDamageSourceFrom(player, []DamageSourceRef{This, observed}, binding)
	if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || calls != 0 || sourceCalls != 0 || a.calls != 0 {
			t.Fatal("query resolved player or made choice")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if calls != 1 || sourceCalls != 1 || a.calls != 1 {
		t.Fatal("choice timing", calls, sourceCalls, a.calls)
	}
	if len(a.seen) != 1 || !a.seen[0].SameObject(g.CaptureDamageSource(p.ID())) {
		t.Fatal("choice did not deduplicate the exact source")
	}
	for _, effect := range []ResolvingEffect{
		ChooseDamageSourceFrom(You, []DamageSourceRef{DamageSourceOf(AttachedPermanent)}, binding),
		ChooseDamageSourceFrom(You, []DamageSourceRef{observed}, binding),
		ChooseDamageSourceFrom(FromPlayer(BindPlayer("unavailable")), []DamageSourceRef{observed}, binding),
		ChooseDamageSourceFromLiveObjects(You, DamageSourceFilter{}, binding),
		ChooseDamageSourceFromWaitingDelayedBindings(You, DamageSourceFilter{}, binding),
		AtNext(EachEndStep, ChooseDamageSourceFrom(You, []DamageSourceRef{This}, binding)),
	} {
		if err := ApplyEffect(g, AtNext(EachEndStep, effect), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("implicit category or future body inferred")
	}
}

type waitingChoiceDamageRef struct {
	DamageSourceRef
	calls *int
}

func (r waitingChoiceDamageRef) resolveDamageSource(ctx *EffectContext) DamageSource {
	*r.calls++
	return r.DamageSourceRef.resolveDamageSource(ctx)
}

func TestWaitingDelayedDamageChoiceInvalid(t *testing.T) {
	binding := BindDamageSource("choice")
	for _, body := range []ResolvingEffect{
		ChooseDamageSourceFrom(nil, []DamageSourceRef{This}, binding),
		ChooseDamageSourceFrom(You, []DamageSourceRef{nil}, binding),
		ChooseDamageSourceFrom(You, []DamageSourceRef{This}, nil),
		ChooseDamageSourceFrom(You, []DamageSourceRef{FromDamageSource(binding)}, binding),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid delayed choice accepted")
		}
	}
}
