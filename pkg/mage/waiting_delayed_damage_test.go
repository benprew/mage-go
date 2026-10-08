package mage

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedDamageFrames(t *testing.T) {
	for _, kind := range []string{"permanent", "spell"} {
		for _, change := range []string{"live", "color", "control", "departed", "returned", "phased", "cycle", "face-down"} {
			if kind == "spell" && (change == "phased" || change == "cycle" || change == "face-down") {
				continue
			}
			for _, mode := range []bool{false, true} {
				t.Run(kind+"/"+change+map[bool]string{false: "/body", true: "/modal"}[mode], func(t *testing.T) {
					g, a, b, spell := spellSourceFixture(t)
					card := NewCreature("Source", "{U}", 1, 4, WithSubTypes("Wizard"))
					card.SetOwner(a.PlayerID())
					p := g.PutOnBattlefield(card, a.PlayerID())
					saved := g.CaptureDamageSource(p.ID())
					if kind == "spell" {
						saved = g.CaptureSpellDamageSource(spell.ID)
					}
					binding := BindDamageSource("event")
					body := DealDamageFrom(FromDamageSource(binding), You, Fixed(2))
					delay := AtNext(EachEndStep, body)
					if mode {
						delay = AtNextModes(EachEndStep, ModeOf("Damage", body), ModeOf("Other", GainLifeFor(You, Fixed(1))))
					}
					observer := NewArtifact("Observer", "", WithAbility(WhenEvent(ControllerIsDealtDamage(BindNumber("amount")).WithSource(binding), delay)))
					if err := ValidateCardDefinition(observer.Name(), observer); err != nil {
						t.Fatal(err)
					}
					ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
					if err := (&eventDamageSourceCapture{binding: binding, source: saved}).Apply(ctx); err != nil {
						t.Fatal(err)
					}
					if err := delay.Apply(ctx); err != nil {
						t.Fatal(err)
					}
					g.resolution.Clear()
					before := g.Clone()
					if kind == "permanent" {
						switch change {
						case "color":
							g.ChangeObjectColor(p.ID(), core.Red)
						case "control":
							g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
						case "departed", "returned":
							g.ChangeObjectColor(p.ID(), core.Red)
							g.BouncePermanentToHand(g.FindPermanent(p.ID()))
							if change == "returned" {
								a.RemoveFromHand(card.ID())
								g.PutOnBattlefield(card, b.PlayerID())
							}
						case "phased", "cycle":
							ids := g.PhaseOut(p.ID())
							if change == "cycle" {
								g.PhaseIn(ids)
							}
						case "face-down":
							g.MutablePermanent(p.ID()).FaceDown = true
						}
					} else {
						if change == "color" || change == "departed" || change == "returned" {
							g.ChangeObjectColor(spell.ID, core.Blue)
						}
						if change == "control" {
							spell.Controller = b.PlayerID()
						}
						if change == "departed" || change == "returned" {
							g.CounterStackObject(spell.ID)
						}
						if change == "returned" {
							a.RemoveFromGraveyard(spell.Card.ID())
							a.AddToHand(spell.Card)
							a.ManaPool().Add(core.Red, 1)
							if err := g.CastSpellByID(a.PlayerID(), spell.Card.ID(), nil, 0); err != nil {
								t.Fatal(err)
							}
						}
					}
					got := g.WaitingDelayedDamageSources()
					if len(got) != 1 || !got[0].SameObject(saved) {
						t.Fatal("saved identity missing")
					}
					changed := change == "color" || change == "departed" || change == "returned"
					color := core.Red
					if kind == "spell" {
						color = core.Blue
					}
					if got[0].HasColor(color) != changed {
						t.Fatal("current or departure color")
					}
					if change == "control" && got[0].Controller() != b.PlayerID() {
						t.Fatal("current controller")
					}
					if change == "face-down" && (got[0].Name() != "" || got[0].HasColor(core.Blue) || got[0].HasSubType("Wizard")) {
						t.Fatal("hidden information")
					}
					got[0].card.AddType(core.TypeArtifact)
					if g.WaitingDelayedDamageSources()[0].HasType(core.TypeArtifact) || before.WaitingDelayedDamageSources()[0].HasType(core.TypeArtifact) {
						t.Fatal("mutable snapshot")
					}
					clone := g.Clone()
					for _, game := range []*Game{g, clone} {
						game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
						if len(game.WaitingDelayedDamageSources()) != 0 {
							t.Fatal("one-shot still waiting")
						}
						game.PutTriggersOnStack()
						if sources := game.StackBoundDamageSources(); len(sources) != 1 || !sources[0].SameObject(saved) || sources[0].HasColor(color) != changed {
							t.Fatal("placed frame lost source")
						}
						if game.CopyStackObjectDirect(game.StackPeek(), a.PlayerID(), false) == nil {
							t.Fatal("copy")
						}
						outer := BindDamageSource("outer")
						game.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{outer: saved}
						for range 2 {
							obj := game.StackPeek()
							if err := Sequence(obj.Effects...).Apply(&EffectContext{Game: game, Controller: a.PlayerID()}); err != nil {
								t.Fatal(err)
							}
							if len(game.resolution.damageSourceBindings) != 1 || !game.resolution.damageSourceBindings[outer].SameObject(saved) {
								t.Fatal("outer frame not restored")
							}
							game.GetStack().Pop()
						}
						if game.GetPlayer(a.PlayerID()).Life() != 16 {
							t.Fatal("saved source body or copy")
						}
					}
				})
			}
		}
	}
}

func TestWaitingDelayedDamageBoundaries(t *testing.T) {
	g, a, _, spell := spellSourceFixture(t)
	saved := g.CaptureSpellDamageSource(spell.ID)
	one, two := BindDamageSource("one"), BindDamageSource("two")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{one: saved, two: saved}
	delay := WhenLeavesThisTurn(This, DealDamageFrom(FromDamageSource(one), You, Fixed(1)))
	p := g.PutOnBattlefield(NewArtifact("Observer", ""), a.PlayerID())
	ctx.SourceID = p.ID()
	if err := delay.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	chooser := &sourceChoicePlayer{BasePlayer: a, index: -1}
	g.players[0] = chooser
	output, other := BindDamageSource("choice"), BindDamageSource("other")
	choice := ChooseDamageSourceFromWaitingDelayedDamageBindings(You, DamageSourceHasColor(core.Red), output)
	if err := choice.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 1 || !FromDamageSource(output).resolveDamageSource(ctx).SameObject(saved) || len(g.WaitingDelayedDamageSources()) != 1 {
		t.Fatal("choice or deduplication")
	}
	if err := ChooseDamageSourceFromWaitingDelayedDamageBindings(You, DamageSourceHasColor(core.Blue), other).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(other).resolveDamageSource(ctx).Valid() || !FromDamageSource(output).resolveDamageSource(ctx).Valid() {
		t.Fatal("independent choices")
	}
	g.BouncePermanentToHand(p)
	g.PutTriggersOnStack()
	g.ResolveStack()
	if len(g.WaitingDelayedDamageSources()) != 1 {
		t.Fatal("persistent lifetime")
	}
	g.RunStepWithPriority(core.Cleanup)
	if len(g.WaitingDelayedDamageSources()) != 0 {
		t.Fatal("cleanup lifetime")
	}
	if err := choice.Apply(ctx); err != nil || FromDamageSource(output).resolveDamageSource(ctx).Valid() {
		t.Fatal("empty list clearing")
	}
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{output: saved}
	ctx.Controller = uuid.New()
	if err := choice.Apply(ctx); err != nil || FromDamageSource(output).resolveDamageSource(ctx).Valid() {
		t.Fatal("absent player clearing")
	}
	for _, effect := range []ResolvingEffect{ChooseDamageSourceFromWaitingDelayedDamageBindings(nil, DamageSourceFilter{}, output), ChooseDamageSourceFromWaitingDelayedDamageBindings(You, DamageSourceFilter{}, nil), AtNext(EachEndStep, DealDamageFrom(FromDamageSource(BindDamageSource("unbound")), You, Fixed(1)))} {
		if err := ValidateCardDefinition("Invalid", NewInstant("Invalid", "", Spell(effect))); err == nil {
			t.Fatal("invalid declaration")
		}
	}
	g.RegisterDelayedTrigger(&DelayedTrigger{EventType: core.EvtEndStep, Effects: []Effect{AtNext(EachEndStep, GainLifeFor(You, Fixed(1)))}})
	g.RegisterDelayedTrigger(&DelayedTrigger{EventType: core.EvtEndStep, SourceID: spell.SourceID})
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{one: saved}
	if len(g.WaitingDelayedDamageSources()) != 0 {
		t.Fatal("future producer or global frame admitted")
	}
}

func TestWaitingDelayedDamageIndependentEvents(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	observers := make([]*Permanent, 0, 2)
	for _, player := range []*BasePlayer{a, b} {
		source, amount := BindDamageSource("event"), BindNumber("amount")
		card := NewArtifact("Observer", "", WithAbility(WhenEvent(ControllerIsDealtDamage(amount).WithSource(source), AtNext(EachEndStep, DealDamageFrom(FromDamageSource(source), You, FromNumber(amount))))))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		observers = append(observers, g.PutOnBattlefield(card, player.PlayerID()))
	}
	first := g.PutOnBattlefield(NewCreature("First", "{U}", 1, 4), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "{R}", 1, 4), b.PlayerID())
	sources := []DamageSource{g.CaptureDamageSource(first.ID()), g.CaptureDamageSource(second.ID())}
	g.DealSimultaneousDamage(PlayerDamageAssignment(a, 1, sources[0]), PlayerDamageAssignment(a, 2, sources[1]), PlayerDamageAssignment(b, 1, sources[0]), PlayerDamageAssignment(b, 2, sources[1]))
	if len(g.WaitingDelayedDamageSources()) != 0 {
		t.Fatal("pending trigger became a waiting delay")
	}
	g.PutTriggersOnStack()
	g.ResolveStack()
	got := g.WaitingDelayedDamageSources()
	if len(got) != 2 || len(g.DelayedTriggers()) != 4 {
		t.Fatal("event frames or exact deduplication")
	}
	for _, saved := range sources {
		if !got[0].SameObject(saved) && !got[1].SameObject(saved) {
			t.Fatal("independent event lost")
		}
	}
	for _, p := range observers {
		g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		objects := game.StackObjects()
		if len(objects) != 4 || objects[0].Controller != a.PlayerID() || objects[2].Controller != b.PlayerID() {
			t.Fatal("APNAP placement")
		}
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 14 || game.GetPlayer(b.PlayerID()).Life() != 14 {
			t.Fatal("event amount/source replay or clone")
		}
	}
}

func TestWaitingDelayedDamageSavedFacts(t *testing.T) {
	g, a, b, spell := spellSourceFixture(t)
	g.CounterStackObject(spell.ID)
	p := g.PutOnBattlefield(NewCreature("Source", "{U}", 1, 4, WithKeyword(core.Lifelink)), a.PlayerID())
	saved := g.CaptureDamageSource(p.ID())
	binding := BindDamageSource("event")
	ctx := &EffectContext{Game: g, Controller: b.PlayerID()}
	if err := (&eventDamageSourceCapture{binding: binding, source: saved}).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if err := AtNext(EachEndStep, DealDamageFrom(FromDamageSource(binding), You, Fixed(2))).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.resolution.damageSourceBindings[binding].card.AddType(core.TypeArtifact)
	g.resolution.Clear()
	g.ChangeObjectColor(p.ID(), core.Red)
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	got := g.WaitingDelayedDamageSources()
	if len(got) != 1 || !got[0].HasColor(core.Red) || got[0].Controller() != b.PlayerID() || got[0].HasType(core.TypeArtifact) {
		t.Fatal("current query or capture isolation")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 18 {
		t.Fatalf("body did not retain event-time lifelink controller: %d/%d", a.Life(), b.Life())
	}
}

func TestWaitingDelayedDamageInvalidFrames(t *testing.T) {
	g, a, _, spell := spellSourceFixture(t)
	binding := BindDamageSource("saved")
	source := g.CaptureSpellDamageSource(spell.ID)
	invalid := []DamageSource{{}, {id: uuid.New()}, {id: uuid.New(), incarnation: uuid.New()}, {id: source.ID(), stackID: source.StackID(), incarnation: uuid.New()}}
	for _, saved := range invalid {
		g.RegisterDelayedTrigger(&DelayedTrigger{EventType: core.EvtEndStep, Effects: []Effect{&capturedDelayedBody{body: GainLifeFor(You, Fixed(1)), damageSources: map[*DamageSourceBinding]DamageSource{binding: saved, nil: source}}}})
	}
	if len(g.WaitingDelayedDamageSources()) != 0 {
		t.Fatal("invalid reference admitted")
	}
	p := g.PutOnBattlefield(NewCreature("Source", "", 1, 4), a.PlayerID())
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	saved := g.CaptureDamageSource(p.ID())
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{binding: saved}
	if err := AtNext(EachEndStep, GainLifeFor(You, Fixed(1))).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	got := g.WaitingDelayedDamageSources()
	if len(got) != 1 || !got[0].SameObject(saved) {
		t.Fatal("invalid frame masked valid reference")
	}
}

func TestWaitingDelayedDamageErrorRestoration(t *testing.T) {
	g, a, _, spell := spellSourceFixture(t)
	saved := g.CaptureSpellDamageSource(spell.ID)
	binding, outer := BindDamageSource("saved"), BindDamageSource("outer")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{binding: saved}
	failure := errors.New("body failure")
	body := FuncEffect("check frame", EffectProperties{}, func(game *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
		source := game.resolution.damageSourceBindings[binding]
		if !source.SameObject(saved) || len(game.resolution.damageSourceBindings) != 1 {
			t.Fatal("wrong inner frame")
		}
		source.card.AddType(core.TypeArtifact)
		return failure
	})
	if err := AtNext(EachEndStep, body).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{outer: saved}
	frame := g.DelayedTriggers()[0].Effects[0]
	for range 2 {
		if err := frame.Apply(ctx); !errors.Is(err, failure) {
			t.Fatal("lost body error")
		}
		if len(g.resolution.damageSourceBindings) != 1 || !g.resolution.damageSourceBindings[outer].SameObject(saved) || g.resolution.damageSourceBindings[outer].HasType(core.TypeArtifact) {
			t.Fatal("outer frame changed on failure")
		}
		if g.WaitingDelayedDamageSources()[0].HasType(core.TypeArtifact) {
			t.Fatal("body changed saved snapshot")
		}
	}
}
