package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func boundSpellObject(source DamageSource) *StackObject {
	return &StackObject{ID: uuid.New(), IsAbility: true, Effects: []Effect{Sequence(Sequence(&eventSpellCapture{binding: BindSpell("event"), id: source.StackID(), source: source}), GainLifeFor(You, Fixed(1)))}}
}

func TestStackBoundSpellSourcesDeparture(t *testing.T) {
	for _, removal := range []string{"live", "pop", "id", "source", "counter"} {
		t.Run(removal, func(t *testing.T) {
			g, a, b, spell := spellSourceFixture(t)
			saved := g.CaptureSpellDamageSource(spell.ID)
			object := boundSpellObject(saved)
			g.PushStack(object)
			clone := g.Clone()
			g.ChangeObjectColor(spell.ID, core.Blue)
			spell.Controller = b.PlayerID()
			switch removal {
			case "pop":
				g.stack.objects[0], g.stack.objects[1] = g.stack.objects[1], g.stack.objects[0]
				g.GetStack().Pop()
			case "id":
				g.GetStack().RemoveByID(spell.ID)
			case "source":
				g.GetStack().RemoveBySourceID(spell.SourceID)
			case "counter":
				g.CounterStackObject(spell.ID)
			}
			if removal != "live" {
				a.AddToHand(spell.Card)
				a.ManaPool().Add(core.Red, 1)
				if err := g.CastSpellByID(a.PlayerID(), spell.Card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
			}
			if g.CopyStackObjectDirect(object, a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			got := g.StackBoundSpellSources()
			if len(got) != 1 || !got[0].SameObject(saved) || !got[0].HasColor(core.Blue) || got[0].Controller() != b.PlayerID() {
				t.Fatal("live/departure information or recast identity")
			}
			clone.GetStack().RemoveByID(spell.ID)
			if !clone.StackBoundSpellSources()[0].HasColor(core.Red) {
				t.Fatal("departure changed clone")
			}
			capture := object.Effects[0].(*sequenceEffect).effects[0].(*sequenceEffect).effects[0].(*eventSpellCapture)
			ctx := &EffectContext{Game: g}
			if err := capture.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if g.resolution.spellBindings[capture.binding] != spell.ID {
				t.Fatal("query departure data changed the event-time binding")
			}
			got[0].card.AddType(core.TypeArtifact)
			if g.StackBoundSpellSources()[0].HasType(core.TypeArtifact) {
				t.Fatal("query exposed snapshot")
			}
			cloneAfter := g.Clone()
			for _, obj := range slices.Clone(g.StackObjects()) {
				g.GetStack().RemoveByID(obj.ID)
			}
			if got := cloneAfter.StackBoundSpellSources(); len(got) != 1 || !got[0].HasColor(core.Blue) {
				t.Fatal("clone after departure lost query information")
			}
			if len(g.StackBoundSpellSources()) != 0 || len(clone.StackBoundSpellSources()) != 1 {
				t.Fatal("reference lifetime")
			}
		})
	}
}

func TestStackBoundSpellSourcesChoiceAndDeclarations(t *testing.T) {
	g, a, _, spell := spellSourceFixture(t)
	chooser := &sourceChoicePlayer{BasePlayer: a, index: -1}
	g.players[0] = chooser
	g.PushStack(boundSpellObject(g.CaptureSpellDamageSource(spell.ID)))
	one, two := BindDamageSource("one"), BindDamageSource("two")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	choice := ChooseDamageSourceFromStackSpellBindings(You, DamageSourceHasColor(core.Red), one)
	if err := choice.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 1 || !FromDamageSource(one).resolveDamageSource(ctx).SameObject(g.CaptureSpellDamageSource(spell.ID)) {
		t.Fatal("filtered choice or fallback")
	}
	chooser.seen[0].card.AddType(core.TypeArtifact)
	if FromDamageSource(one).resolveDamageSource(ctx).HasType(core.TypeArtifact) {
		t.Fatal("choice exposed result")
	}
	if err := ChooseDamageSourceFromStackSpellBindings(You, DamageSourceHasColor(core.Blue), two).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 1 || FromDamageSource(two).resolveDamageSource(ctx).Valid() || !FromDamageSource(one).resolveDamageSource(ctx).Valid() {
		t.Fatal("independent outputs")
	}
	clone := g.Clone()
	g.GetStack().Pop()
	if err := choice.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(one).resolveDamageSource(ctx).Valid() || !FromDamageSource(one).resolveDamageSource(&EffectContext{Game: clone}).Valid() {
		t.Fatal("empty clearing or cloned result")
	}
	if err := ChooseDamageSourceFromStackSpellBindings(FromPlayer(BindPlayer("absent")), DamageSourceFilter{}, one).Apply(&EffectContext{Game: clone}); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(one).resolveDamageSource(&EffectContext{Game: clone}).Valid() {
		t.Fatal("absent-player clearing")
	}
	body := Sequence(choice, DealDamageFrom(FromDamageSource(one), You, Fixed(1)))
	if err := ValidateCardDefinition("Valid", NewArtifact("Valid", "", WithAbility(Activate(Costs(), body)))); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{ChooseDamageSourceFromStackSpellBindings(nil, DamageSourceFilter{}, one), ChooseDamageSourceFromStackSpellBindings(You, DamageSourceFilter{}, nil), ChooseDamageSourceFromStackSpellBindings(You, DamageSourceFilter{}, BindDamageSource("")), ChooseDamageSourceFromStackSpellBindings(You, DamageSourceHasColor(core.AnyColor), one), Sequence(DealDamageFrom(FromDamageSource(one), You, Fixed(1)), choice)} {
		if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestStackBoundSpellSourcesBoundaries(t *testing.T) {
	g, a, _, spell := spellSourceFixture(t)
	saved := g.CaptureSpellDamageSource(spell.ID)
	other := NewArtifact("Other", "")
	other.SetOwner(a.PlayerID())
	second := &StackObject{ID: uuid.New(), SourceID: other.ID(), Card: other, Controller: a.PlayerID()}
	g.PushStack(second)
	g.PushStack(&StackObject{ID: uuid.New(), Effects: []Effect{Sequence(boundSpellObject(g.CaptureSpellDamageSource(second.ID)).Effects...), Sequence(boundSpellObject(saved).Effects...)}})
	got := g.StackBoundSpellSources()
	if len(got) != 2 || got[0].StackID() != second.ID || !got[1].SameObject(saved) {
		t.Fatal("capture traversal order")
	}
	for _, obj := range slices.Clone(g.StackObjects()) {
		g.GetStack().RemoveByID(obj.ID)
	}
	p := g.PutOnBattlefield(other, a.PlayerID())
	wrong := saved
	wrong.incarnation = uuid.New()
	for _, value := range []DamageSource{{}, wrong, g.CaptureDamageSource(p.ID())} {
		g.PushStack(boundSpellObject(value))
	}
	g.PushStack(&StackObject{ID: uuid.New(), Effects: []Effect{&eventSpellCapture{id: spell.ID, source: saved}, &eventSpellCapture{binding: BindSpell("mismatch"), id: uuid.New(), source: saved}, SnapshotDamageSource(DamageSourceOfSpell(FromSpell(BindSpell("future"))), BindDamageSource("future"))}})
	g.resolution.spellBindings = map[*SpellBinding]uuid.UUID{BindSpell("global"): spell.ID}
	if len(g.StackBoundSpellSources()) != 0 {
		t.Fatal("nil, incompatible, mismatched, future or global binding admitted")
	}
}

func TestStackBoundSpellSourcesPendingAndResolving(t *testing.T) {
	g, a, _, spell := spellSourceFixture(t)
	binding := BindSpell("cast")
	card := NewEnchantment("Observer", "", WithAbility(WhenEvent(SpellCast().WithSpell(binding), Counter(FromSpell(binding)))))
	card.SetOwner(a.PlayerID())
	observer := g.PutOnBattlefield(card, a.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtSpellCast, SourceID: spell.SourceID, TargetID: spell.ID, PlayerID: a.PlayerID()})
	if len(g.StackBoundSpellSources()) != 0 {
		t.Fatal("pending trigger admitted")
	}
	g.GetStack().RemoveByID(spell.ID)
	g.DestroyPermanent(observer)
	g.PutTriggersOnStack()
	got := g.StackBoundSpellSources()
	if len(got) != 1 || got[0].StackID() != spell.ID {
		t.Fatal("detection snapshot lost before placement")
	}
	clone := g.Clone()
	g.ResolveTopOfStack()
	if len(g.StackBoundSpellSources()) != 0 || len(clone.StackBoundSpellSources()) != 1 {
		t.Fatal("resolving or cloned reference lifetime")
	}
}

func TestStackBoundSpellSourcesPermanentSpellAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	card := NewArtifact("Permanent spell", "{U}")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	copyObject := g.CopyStackObjectDirect(original, b.PlayerID(), false)
	if copyObject == nil {
		t.Fatal("copy failed")
	}
	first, second := g.CaptureSpellDamageSource(original.ID), g.CaptureSpellDamageSource(copyObject.ID)
	refOne, refTwo := boundSpellObject(first), boundSpellObject(second)
	g.PushStack(refOne)
	g.PushStack(refTwo)
	if got := g.StackBoundSpellSources(); len(got) != 2 || !got[0].SameObject(first) || !got[1].SameObject(second) {
		t.Fatal("spell copy identity or order")
	}
	g.stack.objects = []*StackObject{refOne, refTwo, original, copyObject}
	g.ResolveTopOfStack()
	g.ResolveTopOfStack()
	if g.FindPermanent(card.ID()) == nil {
		t.Fatal("permanent spell did not enter")
	}
	g.ChangeObjectColor(card.ID(), core.Red)
	got := g.StackBoundSpellSources()
	if len(got) != 2 || !got[0].SameObject(first) || !got[1].SameObject(second) || got[0].Incarnation() != uuid.Nil || !got[0].HasColor(core.Blue) || !got[1].HasType(core.TypeArtifact) || got[1].Controller() != b.PlayerID() {
		t.Fatal("entry replaced a saved spell or copy")
	}
	clone := g.Clone()
	g.GetStack().RemoveByID(refOne.ID)
	g.GetStack().RemoveByID(refTwo.ID)
	if len(g.StackBoundSpellSources()) != 0 || len(clone.StackBoundSpellSources()) != 2 {
		t.Fatal("clone lost departure maps")
	}
}
