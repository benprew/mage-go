package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCastResultIdentity(t *testing.T) {
	for _, scenario := range []string{"resolve", "decline", "stale", "tax", "counter", "recast", "return", "phase", "phase-cycle", "control", "copy", "clone", "clone-after", "face-up"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			chooser := &optionalFaceDownPlayer{BasePlayer: a, accept: scenario != "decline"}
			g.SetPlayerAt(0, chooser)
			card := NewCreature("Private", "{U}", 5, 5)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			hand := BindCard("hand")
			result := BindDamageSource("cast result")
			g.resolution.cardBindings = map[*CardBinding]capturedCard{hand: {id: card.ID(), visit: g.handVisit(card.ID()), zone: core.ZoneHand}}
			if scenario == "stale" {
				a.RemoveFromHand(card.ID())
				a.AddToHand(card)
			}
			if scenario == "tax" {
				g.PutOnBattlefield(NewArtifact("Tax", "", WithAbility(StaticAbility(IncreaseSpellCostStatic(func(*Game, uuid.UUID, Card, uuid.UUID) bool { return true }, core.ManaCost{Generic: 1}, false)))), b.PlayerID())
			}
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			effect := MayCastHandCardFaceDownWithoutPayingInto(You, FromCard(hand), result)
			if err := effect.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			ref := PermanentFromResolvedSpell(FromDamageSource(result))
			if _, ok := ref.resolvePermanent(ctx); ok {
				t.Fatal("result became a permanent before resolution")
			}
			cast := scenario != "decline" && scenario != "stale" && scenario != "tax"
			if g.resolution.damageSourceBindings[result].Valid() != cast {
				t.Fatal("incorrect cast result")
			}
			if !cast {
				return
			}
			saved := g.resolution.damageSourceBindings[result]
			spell := g.StackPeek()
			if g.resolution.damageSourceBindings[result].StackID() != spell.ID {
				t.Fatal("wrong spell visit")
			}
			if err := AtNext(EachEndStep, TapSubject(ref)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			var clone *Game
			if scenario == "clone" {
				clone = g.Clone()
			}
			if scenario == "counter" || scenario == "recast" {
				g.CounterStackObject(spell.ID)
			}
			if scenario == "recast" {
				a.RemoveFromGraveyard(card.ID())
				a.AddToHand(card)
				if _, err := g.castHandCardFaceDownWithoutPaying(a.PlayerID(), card.ID()); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "copy" {
				copied := g.CopyStackObjectDirect(spell, b.PlayerID(), false)
				g.ResolveTopOfStack()
				g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{result: saved}
				if p, ok := ref.resolvePermanent(ctx); ok || p != nil {
					t.Fatal("copy supplied original result")
				}
				g.CounterStackObject(spell.ID)
				if g.FindPermanent(copied.Card.ID()) == nil {
					t.Fatal("missing copy")
				}
			}
			g.ResolveStack()
			// A later reader uses the saved result in its own resolution frame.
			g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{result: saved}
			p := g.FindPermanent(card.ID())
			if scenario == "clone-after" {
				cloned := g.Clone()
				cloned.TapPermanent(cloned.FindPermanent(p.ID()))
				cloneCtx := &EffectContext{Game: cloned, Controller: a.PlayerID()}
				if _, ok := ref.resolvePermanent(cloneCtx); !ok {
					t.Fatal("copy-on-write lost identity")
				}
				if g.FindPermanent(p.ID()).Tapped {
					t.Fatal("clone tap changed original")
				}
			}
			if scenario == "face-up" {
				g.turnFaceUp(p)
			}
			if scenario == "return" {
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			}
			if scenario == "phase" || scenario == "phase-cycle" {
				g.PhaseOut(p.ID())
			}
			if scenario == "phase-cycle" {
				g.PhaseIn([]uuid.UUID{p.ID()})
			}
			if scenario == "control" {
				g.AddControlEffect(ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			want := scenario != "counter" && scenario != "recast" && scenario != "return" && scenario != "phase" && scenario != "copy"
			if _, ok := ref.resolvePermanent(ctx); ok != want {
				t.Fatalf("reference present=%v want=%v", ok, want)
			}
			if clone != nil {
				clone.CounterStackObject(spell.ID)
				clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				clone.PutTriggersOnStack()
				clone.ResolveStack()
				if g.FindPermanent(card.ID()).Tapped {
					t.Fatal("clone mutated original")
				}
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			p = g.FindPermanentIncludingPhased(card.ID())
			if p != nil && p.Tapped != want {
				t.Fatal("delayed result lost identity")
			}
		})
	}
}

func TestCastResultIdentityClearsFailure(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Private", "", 2, 2)
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	hand := BindCard("hand")
	result := BindDamageSource("result")
	g.resolution.cardBindings = map[*CardBinding]capturedCard{hand: {id: card.ID(), visit: g.handVisit(card.ID()), zone: core.ZoneHand}}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	effect := CastHandCardFaceDownWithoutPayingInto(You, FromCard(hand), result)
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if !g.resolution.damageSourceBindings[result].Valid() {
		t.Fatal("missing successful result")
	}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.resolution.damageSourceBindings[result].Valid() {
		t.Fatal("failed cast retained earlier result")
	}
}

func TestCastResultIdentityDefinition(t *testing.T) {
	hand := BindCard("hand")
	result := BindDamageSource("result")
	good := ChooseHandCardOrElse(You, CardFilter{}, hand, Sequence(MayCastHandCardFaceDownWithoutPayingInto(You, FromCard(hand), result), AtNext(EachEndStep, TapSubject(PermanentFromResolvedSpell(FromDamageSource(result))))), GainLifeFor(You, Fixed(0)))
	if err := ValidateCardDefinition("Good", NewSorcery("Good", "", good)); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{
		CastHandCardFaceDownWithoutPayingInto(You, FromCard(hand), nil),
		MayCastHandCardFaceDownWithoutPayingInto(You, FromCard(hand), nil),
		TapSubject(PermanentFromResolvedSpell(nil)),
		TapSubject(PermanentFromResolvedSpell(FromDamageSource(result))),
	} {
		if err := ValidateCardDefinition("Bad", NewSorcery("Bad", "", ChooseHandCardOrElse(You, CardFilter{}, hand, effect, GainLifeFor(You, Fixed(0))))); err == nil {
			t.Fatal("accepted invalid result declaration")
		}
	}
}

func TestCastResultIdentityInformation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Private", "", 5, 5)
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	hand := BindCard("hand")
	result := BindDamageSource("cast")
	g.resolution.cardBindings = map[*CardBinding]capturedCard{hand: {id: card.ID(), visit: g.handVisit(card.ID()), zone: core.ZoneHand}}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := CastHandCardFaceDownWithoutPayingInto(You, FromCard(hand), result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	saved := g.resolution.damageSourceBindings[result]
	g.ResolveStack()
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{result: saved}
	ref := PermanentFromResolvedSpell(FromDamageSource(result))
	if got := PowerOfWithLKI(ref).resolveValue(ctx); got != 2 {
		t.Fatalf("power=%d", got)
	}
	p := g.FindPermanent(card.ID())
	g.AddControlEffect(ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.PhaseOut(p.ID())
	if owner, ok := ControllerOfWithLKI(ref).resolvePlayer(ctx); !ok || owner != b.PlayerID() {
		t.Fatal("phased information lost")
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	a.RemoveFromHand(card.ID())
	g.PutOnBattlefield(card, a.PlayerID())
	if got := PowerOfWithLKI(ref).resolveValue(ctx); got != 2 {
		t.Fatalf("LKI power=%d", got)
	}
	if controller, ok := ControllerOfWithLKI(ref).resolvePlayer(ctx); !ok || controller != b.PlayerID() {
		t.Fatal("LKI controller lost")
	}
	if _, ok := ref.resolvePermanent(ctx); ok {
		t.Fatal("LKI supplied action subject")
	}
}
