package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedSourceBodies(t *testing.T) {
	for _, action := range []struct {
		name string
		body ResolvingEffect
	}{
		{"destroy", Destroy(This)}, {"no-regen", DestroyWithoutRegeneration(This)},
		{"sacrifice", Sacrifice(This)}, {"return", ReturnToHand(This)},
	} {
		for _, change := range []string{"live", "departed", "returned", "phased", "cycle", "color", "control", "face-down", "shroud"} {
			t.Run(action.name+"/"+change, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				card := NewArtifact("Source", "{U}", WithAbility(Activate(Costs(), AtNext(EachEndStep, Sequence(GainLifeFor(You, Fixed(1)), Sequence(action.body))))))
				if err := ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				original := g.CaptureDamageSource(p.ID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if len(g.WaitingDelayedPermanentSources()) != 0 {
					t.Fatal("delay created before resolution")
				}
				g.ResolveStack()
				switch change {
				case "departed", "returned":
					g.BouncePermanentToHand(p)
					if change == "returned" {
						a.RemoveFromHand(card.ID())
						g.PutOnBattlefield(card, b.PlayerID())
						g.ChangeObjectColor(card.ID(), core.Red)
					}
				case "phased", "cycle":
					ids := g.PhaseOut(p.ID())
					if change == "cycle" {
						g.PhaseIn(ids)
					}
				case "color":
					g.ChangeObjectColor(p.ID(), core.Red)
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "face-down":
					g.MutablePermanent(p.ID()).FaceDown = true
				case "shroud":
					g.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
				}
				got := g.WaitingDelayedPermanentSources()
				if len(got) != 1 || !got[0].SameObject(original) {
					t.Fatal("source body lost original incarnation")
				}
				if got[0].HasColor(core.Red) != (change == "color") {
					t.Fatal("wrong source information")
				}
				if change == "control" && got[0].Controller() != b.PlayerID() {
					t.Fatal("wrong current control")
				}
				if change == "face-down" && got[0].Name() != "" {
					t.Fatal("hidden information exposed")
				}
				got[0].card.AddType(core.TypeEnchantment)
				if g.WaitingDelayedPermanentSources()[0].HasType(core.TypeEnchantment) {
					t.Fatal("query exposed mutable card")
				}
				clone := g.Clone()
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
				if len(g.WaitingDelayedPermanentSources()) != 0 || len(clone.WaitingDelayedPermanentSources()) != 1 {
					t.Fatal("waiting lifetime or clone isolation")
				}
				g.PutTriggersOnStack()
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				if len(g.WaitingDelayedPermanentSources()) != 0 {
					t.Fatal("stack copy created waiting ability")
				}
				g.ResolveStack()
				clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				clone.PutTriggersOnStack()
				clone.ResolveStack()
				survives := change == "returned" || change == "phased"
				for _, game := range []*Game{g, clone} {
					wantSurvives := survives || (game == clone && action.name == "sacrifice" && change == "control")
					if (game.FindPermanentIncludingPhased(p.ID()) != nil) != wantSurvives {
						t.Fatal("action lost original identity")
					}
				}
				if a.Life() != 21 || b.Life() != 21 || clone.GetPlayer(a.PlayerID()).Life() != 21 {
					t.Fatal("delayed body or copied controller")
				}
			})
		}
	}
}

func TestWaitingDelayedSourceBodyBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	for _, body := range []ResolvingEffect{GainLifeFor(You, Fixed(1)), Destroy(From(BindPermanent("other"))), AtNext(EachEndStep, Destroy(This))} {
		g.triggers.SetDelayed(nil)
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(g.WaitingDelayedPermanentSources()) != 0 {
			t.Fatal("source ID or future body supplied reference")
		}
	}
	g.triggers.SetDelayed(nil)
	effect := AtNextModes(EachEndStep, ModeOf("Destroy", Destroy(This)), ModeOf("Return", ReturnToHand(This)))
	if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.WaitingDelayedPermanentSources()) != 1 {
		t.Fatal("modal source reference absent")
	}
	g.triggers.SetDelayed(nil)
	g.RegisterDelayedTrigger(&DelayedTrigger{SourceID: p.ID(), sourceIncarnation: p.incarnationID, Effects: []Effect{Destroy(This)}})
	for _, inc := range []uuid.UUID{uuid.Nil, uuid.New()} {
		g.RegisterDelayedTrigger(&DelayedTrigger{SourceID: p.ID(), sourceCaptured: true, sourceIncarnation: inc, Effects: []Effect{Destroy(This)}})
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("invalid source identity admitted")
	}
	for _, body := range []ResolvingEffect{Destroy(nil), Sacrifice(nil), ReturnToHand(nil)} {
		if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))); err == nil {
			t.Fatal("invalid source body accepted")
		}
	}
}

func TestWaitingDelayedSourceBodyChoicesAndInstances(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	chooser := &sourceChoicePlayer{BasePlayer: a, index: -1}
	g := NewGame(chooser, b)
	originals := make([]DamageSource, 0, 2)
	for _, player := range []*BasePlayer{a, b} {
		card := NewArtifact("Source", "{U}", WithAbility(Activate(Costs(), AtNextModes(EachEndStep, ModeOf("Return", ReturnToHand(This)), ModeOf("Destroy", Destroy(This))))))
		card.SetOwner(player.PlayerID())
		p := g.PutOnBattlefield(card, player.PlayerID())
		originals = append(originals, g.CaptureDamageSource(p.ID()))
		if err := g.ActivateAbilityByIndex(player.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	got := g.WaitingDelayedPermanentSources()
	if len(got) != 2 || !got[0].SameObject(originals[0]) || !got[1].SameObject(originals[1]) {
		t.Fatal("instance identity or registration order")
	}
	one, two := BindDamageSource("one"), BindDamageSource("two")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	choice := ChooseDamageSourceFromWaitingDelayedBindings(You, DamageSourceHasColor(core.Blue), one)
	if err := choice.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 1 || !FromDamageSource(one).resolveDamageSource(ctx).SameObject(originals[0]) {
		t.Fatal("filtered fallback choice")
	}
	if err := ChooseDamageSourceFromWaitingDelayedBindings(You, DamageSourceHasColor(core.Red), two).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(two).resolveDamageSource(ctx).Valid() || !FromDamageSource(one).resolveDamageSource(ctx).Valid() {
		t.Fatal("independent filtered outputs")
	}
	clone := g.Clone()
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 || g.StackObjects()[0].Controller != a.PlayerID() || g.StackObjects()[1].Controller != b.PlayerID() {
		t.Fatal("modal APNAP placement")
	}
	if err := choice.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(one).resolveDamageSource(ctx).Valid() || len(clone.WaitingDelayedPermanentSources()) != 2 {
		t.Fatal("empty clearing or clone lifetime")
	}
	g.ResolveStack()
	for _, original := range originals {
		if g.FindPermanent(original.ID()) != nil || clone.FindPermanent(original.ID()) == nil {
			t.Fatal("modal native action or clone state")
		}
	}
	g = clone
	if err := ChooseDamageSourceFromWaitingDelayedBindings(FromPlayer(BindPlayer("absent")), DamageSourceFilter{}, one).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(one).resolveDamageSource(&EffectContext{Game: g}).Valid() {
		t.Fatal("absent player retained result")
	}
}
