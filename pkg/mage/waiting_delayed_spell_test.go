package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedSpellFrames(t *testing.T) {
	for _, mode := range []bool{false, true} {
		for _, change := range []string{"live", "color", "control", "departed", "recast"} {
			t.Run(change+map[bool]string{false: "/body", true: "/modal"}[mode], func(t *testing.T) {
				g, a, b, spell := spellSourceFixture(t)
				binding := BindSpell("event")
				body := Counter(FromSpell(binding))
				var delay = AtNext(EachEndStep, body)
				if mode {
					delay = AtNextModes(EachEndStep, ModeOf("Counter", body), ModeOf("Life", GainLifeFor(You, Fixed(1))))
				}
				observer := NewArtifact("Observer", "", WithAbility(WhenEvent(SpellCast().WithSpell(binding), delay)))
				if err := ValidateCardDefinition(observer.Name(), observer); err != nil {
					t.Fatal(err)
				}
				source := g.PutOnBattlefield(observer, a.PlayerID())
				ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: source.ID()}
				if err := (&eventSpellCapture{binding: binding, id: spell.ID, source: g.CaptureSpellDamageSource(spell.ID)}).Apply(ctx); err != nil {
					t.Fatal(err)
				}
				if err := delay.Apply(ctx); err != nil {
					t.Fatal(err)
				}
				saved := g.CaptureSpellDamageSource(spell.ID)
				g.resolution.Clear()
				cloneBefore := g.Clone()
				if change == "color" || change == "departed" || change == "recast" {
					g.ChangeObjectColor(spell.ID, core.Blue)
				}
				if change == "control" {
					spell.Controller = b.PlayerID()
				}
				if change == "departed" || change == "recast" {
					g.CounterStackObject(spell.ID)
				}
				var recastID uuid.UUID
				if change == "recast" {
					g.BouncePermanentToHand(source)
					a.RemoveFromGraveyard(spell.Card.ID())
					a.AddToHand(spell.Card)
					a.ManaPool().Add(core.Red, 1)
					if err := g.CastSpellByID(a.PlayerID(), spell.Card.ID(), nil, 0); err != nil {
						t.Fatal(err)
					}
					recastID = g.StackPeek().ID
				}
				got := g.WaitingDelayedSpellSources()
				if len(got) != 1 || !got[0].SameObject(saved) {
					t.Fatal("saved identity missing")
				}
				if got[0].HasColor(core.Blue) != (change == "color" || change == "departed" || change == "recast") {
					t.Fatal("current or departure information")
				}
				if change == "control" && got[0].Controller() != b.PlayerID() {
					t.Fatal("current controller")
				}
				got[0].card.AddType(core.TypeArtifact)
				if g.WaitingDelayedSpellSources()[0].HasType(core.TypeArtifact) {
					t.Fatal("mutable query result")
				}
				if !cloneBefore.WaitingDelayedSpellSources()[0].HasColor(core.Red) {
					t.Fatal("clone before departure changed")
				}
				clone := g.Clone()
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				if len(g.WaitingDelayedSpellSources()) != 0 || len(clone.WaitingDelayedSpellSources()) != 1 {
					t.Fatal("waiting lifetime")
				}
				g.PutTriggersOnStack()
				if len(g.StackBoundSpellSources()) != 1 {
					t.Fatal("stack frame lost spell reference")
				}
				if !mode {
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
					g.ResolveTopOfStack()
					g.ResolveTopOfStack()
					if change == "recast" {
						if g.stack.FindByID(recastID) == nil {
							t.Fatal("old frame countered recast")
						}
					} else if g.stack.FindByID(spell.ID) != nil {
						t.Fatal("saved frame did not counter")
					}
				}
			})
		}
	}
}

func TestWaitingDelayedSpellChoicesAndBoundaries(t *testing.T) {
	g, a, _, spell := spellSourceFixture(t)
	chooser := &sourceChoicePlayer{BasePlayer: a, index: -1}
	g.players[0] = chooser
	binding := BindSpell("saved")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := (&eventSpellCapture{binding: binding, id: spell.ID}).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	delay := AtNext(EachEndStep, Counter(FromSpell(binding)))
	if err := delay.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	saved := g.CaptureSpellDamageSource(spell.ID)
	g.resolution.Clear()
	one, two := BindDamageSource("one"), BindDamageSource("two")
	choice := ChooseDamageSourceFromWaitingDelayedSpellBindings(You, DamageSourceHasColor(core.Red), one)
	if err := choice.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 1 || !FromDamageSource(one).resolveDamageSource(ctx).SameObject(saved) {
		t.Fatal("filtered choice")
	}
	if err := ChooseDamageSourceFromWaitingDelayedSpellBindings(You, DamageSourceHasColor(core.Blue), two).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(two).resolveDamageSource(ctx).Valid() || !FromDamageSource(one).resolveDamageSource(ctx).Valid() {
		t.Fatal("independent outputs")
	}
	outer := BindSpell("outer")
	g.resolution.spellBindings = map[*SpellBinding]uuid.UUID{outer: uuid.New()}
	frame := g.DelayedTriggers()[0].Effects[0]
	if err := frame.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(g.resolution.spellBindings) != 1 || g.resolution.spellBindings[outer] == uuid.Nil {
		t.Fatal("outer frame not restored")
	}
	g.triggers.SetDelayed(nil)
	if err := choice.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(one).resolveDamageSource(ctx).Valid() || chooser.calls != 1 {
		t.Fatal("empty list reused choice")
	}
	for _, invalid := range []ResolvingEffect{ChooseDamageSourceFromWaitingDelayedSpellBindings(nil, DamageSourceFilter{}, one), ChooseDamageSourceFromWaitingDelayedSpellBindings(You, DamageSourceFilter{}, nil), AtNext(EachEndStep, Counter(FromSpell(BindSpell("unbound"))))} {
		if err := ValidateCardDefinition("Invalid", NewInstant("Invalid", "", Spell(invalid))); err == nil {
			t.Fatal("invalid declaration")
		}
	}
	g.RegisterDelayedTrigger(&DelayedTrigger{EventType: core.EvtEndStep, Effects: []Effect{AtNext(EachEndStep, GainLifeFor(You, Fixed(1)))}})
	g.RegisterDelayedTrigger(&DelayedTrigger{EventType: core.EvtEndStep, SourceID: spell.SourceID})
	invalid := &capturedDelayedBody{body: GainLifeFor(You, Fixed(1)), spells: map[*SpellBinding]uuid.UUID{binding: uuid.New()}, spellSources: map[*SpellBinding]DamageSource{binding: saved, nil: saved}}
	g.RegisterDelayedTrigger(&DelayedTrigger{EventType: core.EvtEndStep, Effects: []Effect{invalid}})
	if len(g.WaitingDelayedSpellSources()) != 0 {
		t.Fatal("future body, source ID, or invalid frame admitted")
	}
	g.resolution.damageSourceBindings[one] = saved
	ctx.Controller = uuid.New()
	if err := choice.Apply(ctx); err != nil || FromDamageSource(one).resolveDamageSource(ctx).Valid() {
		t.Fatal("absent player clearing")
	}
}

func TestWaitingDelayedSpellIndependentEvents(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	for _, player := range []*BasePlayer{a, b} {
		binding := BindSpell("event")
		observer := NewArtifact("Observer", "", WithAbility(WhenEvent(SpellCast().WithSpell(binding), AtNext(EachEndStep, Counter(FromSpell(binding))))))
		if err := ValidateCardDefinition(observer.Name(), observer); err != nil {
			t.Fatal(err)
		}
		g.PutOnBattlefield(observer, player.PlayerID())
	}
	saved := make([]DamageSource, 0, 2)
	for range 2 {
		card := NewInstant("Spell", "", Spell(GainLifeFor(You, Fixed(1))))
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		saved = append(saved, g.CaptureSpellDamageSource(g.StackPeek().ID))
		g.PutTriggersOnStack()
		g.ResolveTopOfStack()
		g.ResolveTopOfStack()
	}
	got := g.WaitingDelayedSpellSources()
	if len(got) != 2 || !got[0].SameObject(saved[0]) || !got[1].SameObject(saved[1]) || len(g.DelayedTriggers()) != 4 {
		t.Fatal("independent event frames or deduplication")
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		objects := game.StackObjects()
		if len(objects) != 6 || objects[2].Controller != a.PlayerID() || objects[4].Controller != b.PlayerID() {
			t.Fatal("APNAP placement")
		}
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.StackSize() != 0 || len(game.WaitingDelayedSpellSources()) != 0 {
			t.Fatal("saved event replay or clone")
		}
	}
}
