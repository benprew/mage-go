package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func boundDamageObject(source DamageSource) *StackObject {
	return &StackObject{ID: uuid.New(), IsAbility: true, Effects: []Effect{Sequence(Sequence(&eventDamageSourceCapture{binding: BindDamageSource("event"), source: source}), GainLifeFor(You, Fixed(1)))}}
}

func TestStackBoundDamageSourcesPermanent(t *testing.T) {
	for _, change := range []string{"live", "departed", "returned", "phased", "cycle", "color", "control", "face-down", "shroud", "departed-color", "phased-color"} {
		t.Run(change, func(t *testing.T) {
			g, a, b, _ := spellSourceFixture(t)
			card := NewCreature("Event source", "{U}", 1, 4, WithSubTypes("Wizard"))
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			object := boundDamageObject(saved)
			g.PushStack(object)
			if g.CopyStackObjectDirect(object, b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			switch change {
			case "departed", "returned", "departed-color":
				if change == "departed-color" {
					g.ChangeObjectColor(p.ID(), core.Red)
				}
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				if change == "returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
					g.ChangeObjectColor(card.ID(), core.Red)
				}
			case "phased", "cycle", "phased-color":
				if change == "phased-color" {
					g.ChangeObjectColor(p.ID(), core.Red)
				}
				ids := g.PhaseOut(p.ID())
				if change == "cycle" {
					g.PhaseIn(ids)
				}
			case "color":
				g.ChangeObjectColor(p.ID(), core.Red)
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "face-down":
				g.MutablePermanent(p.ID()).FaceDown = true
			case "shroud":
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
			}
			size := g.StackSize()
			got := g.StackBoundDamageSources()
			if len(got) != 1 || !got[0].SameObject(saved) || g.StackSize() != size {
				t.Fatal("identity, deduplication or pure read")
			}
			if got[0].HasColor(core.Red) != (change == "color" || change == "departed-color" || change == "phased-color") {
				t.Fatal("live or retained color")
			}
			controller := a.PlayerID()
			if change == "control" {
				controller = b.PlayerID()
			}
			if got[0].Controller() != controller {
				t.Fatal("controller")
			}
			if change == "face-down" && (got[0].Name() != "" || got[0].HasColor(core.Blue) || got[0].HasSubType("Wizard")) {
				t.Fatal("hidden characteristics exposed")
			}
			clone := g.Clone()
			got[0].card.AddType(core.TypeArtifact)
			if g.StackBoundDamageSources()[0].HasType(core.TypeArtifact) || clone.StackBoundDamageSources()[0].HasType(core.TypeArtifact) {
				t.Fatal("mutable query result")
			}
			g.GetStack().RemoveByID(object.ID)
			if len(g.StackBoundDamageSources()) != 1 {
				t.Fatal("copy lost reference")
			}
			g.GetStack().Pop()
			if len(g.StackBoundDamageSources()) != 0 || len(clone.StackBoundDamageSources()) != 1 {
				t.Fatal("final reference or clone lifetime")
			}
		})
	}
}

func TestStackBoundDamageSourcesSpellDeparture(t *testing.T) {
	for _, removal := range []string{"live", "pop", "id", "source", "counter"} {
		t.Run(removal, func(t *testing.T) {
			g, a, b, spell := spellSourceFixture(t)
			saved := g.CaptureSpellDamageSource(spell.ID)
			object := boundDamageObject(saved)
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
			got := g.StackBoundDamageSources()
			if len(got) != 1 || !got[0].SameObject(saved) || !got[0].HasColor(core.Blue) || got[0].Controller() != b.PlayerID() {
				t.Fatal("live/departure information or recast identity")
			}
			clone.GetStack().RemoveByID(spell.ID)
			if !clone.StackBoundDamageSources()[0].HasColor(core.Red) {
				t.Fatal("departure changed clone")
			}
			capture := object.Effects[0].(*sequenceEffect).effects[0].(*sequenceEffect).effects[0].(*eventDamageSourceCapture)
			ctx := &EffectContext{Game: g}
			if err := capture.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if !FromDamageSource(capture.binding).resolveDamageSource(ctx).HasColor(core.Red) {
				t.Fatal("query departure data changed the event-time binding")
			}
			got[0].card.AddType(core.TypeArtifact)
			if g.StackBoundDamageSources()[0].HasType(core.TypeArtifact) {
				t.Fatal("query exposed snapshot")
			}
			cloneAfter := g.Clone()
			for _, obj := range slices.Clone(g.StackObjects()) {
				g.GetStack().RemoveByID(obj.ID)
			}
			if got := cloneAfter.StackBoundDamageSources(); len(got) != 1 || !got[0].HasColor(core.Blue) {
				t.Fatal("clone after departure lost query information")
			}
			if len(g.StackBoundDamageSources()) != 0 || len(clone.StackBoundDamageSources()) != 1 {
				t.Fatal("reference lifetime")
			}
		})
	}
}

func TestStackBoundDamageSourcesBoundaries(t *testing.T) {
	g, a, _, spell := spellSourceFixture(t)
	p := g.PutOnBattlefield(NewArtifact("P", ""), a.PlayerID())
	perm := g.CaptureDamageSource(p.ID())
	sp := g.CaptureSpellDamageSource(spell.ID)
	g.PushStack(boundDamageObject(sp))
	g.PushStack(boundDamageObject(perm))
	g.PushStack(boundDamageObject(sp))
	got := g.StackBoundDamageSources()
	if len(got) != 2 || !got[0].SameObject(sp) || !got[1].SameObject(perm) {
		t.Fatal("effect order or exact deduplication")
	}
	for _, obj := range slices.Clone(g.StackObjects()) {
		g.GetStack().RemoveByID(obj.ID)
	}
	unknown := perm
	unknown.incarnation = uuid.New()
	missing := perm
	missing.incarnation = uuid.Nil
	mixed := perm
	mixed.stackID = uuid.New()
	invalid := DamageSource{}
	for _, source := range []DamageSource{unknown, missing, mixed, invalid} {
		g.PushStack(boundDamageObject(source))
	}
	g.PushStack(&StackObject{ID: uuid.New(), Effects: []Effect{&eventDamageSourceCapture{source: perm}, SnapshotDamageSource(DamageSourceOf(This), BindDamageSource("future"))}})
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{BindDamageSource("resolution"): perm}
	if len(g.StackBoundDamageSources()) != 0 {
		t.Fatal("nil declarations, incompatible identities or future/global results admitted")
	}
}

func TestStackBoundDamageSourcesChoiceAndDeclarations(t *testing.T) {
	g, a, _, spell := spellSourceFixture(t)
	chooser := &sourceChoicePlayer{BasePlayer: a, index: -1}
	g.players[0] = chooser
	g.PushStack(boundDamageObject(g.CaptureSpellDamageSource(spell.ID)))
	one, two := BindDamageSource("one"), BindDamageSource("two")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	choice := ChooseDamageSourceFromStackDamageBindings(You, DamageSourceHasColor(core.Red), one)
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
	if err := ChooseDamageSourceFromStackDamageBindings(You, DamageSourceHasColor(core.Blue), two).Apply(ctx); err != nil {
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
	if err := ChooseDamageSourceFromStackDamageBindings(FromPlayer(BindPlayer("absent")), DamageSourceFilter{}, one).Apply(&EffectContext{Game: clone}); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(one).resolveDamageSource(&EffectContext{Game: clone}).Valid() {
		t.Fatal("absent-player clearing")
	}
	body := Sequence(choice, DealDamageFrom(FromDamageSource(one), You, Fixed(1)))
	if err := ValidateCardDefinition("Valid", NewArtifact("Valid", "", WithAbility(Activate(Costs(), body)))); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{ChooseDamageSourceFromStackDamageBindings(nil, DamageSourceFilter{}, one), ChooseDamageSourceFromStackDamageBindings(You, DamageSourceFilter{}, nil), ChooseDamageSourceFromStackDamageBindings(You, DamageSourceFilter{}, BindDamageSource("")), ChooseDamageSourceFromStackDamageBindings(You, DamageSourceHasColor(core.AnyColor), one), Sequence(DealDamageFrom(FromDamageSource(one), You, Fixed(1)), choice)} {
		if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestStackBoundDamageSourcesNativeBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	event := BindDamageSource("event")
	amount := BindNumber("amount")
	for _, player := range []*BasePlayer{a, b} {
		observer := NewEnchantment("Observer", "", WithAbility(WhenEvent(ControllerIsDealtDamage(amount).WithSource(event), GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(observer.Name(), observer); err != nil {
			t.Fatal(err)
		}
		g.PutOnBattlefield(observer, player.PlayerID())
	}
	p := g.PutOnBattlefield(NewCreature("P", "{U}", 2, 4), a.PlayerID())
	q := g.PutOnBattlefield(NewCreature("Q", "{R}", 2, 4), b.PlayerID())
	first, second := g.CaptureDamageSource(p.ID()), g.CaptureDamageSource(q.ID())
	g.DealSimultaneousDamage(PlayerDamageAssignment(a, 1, first), PlayerDamageAssignment(b, 2, second))
	if len(g.StackBoundDamageSources()) != 0 {
		t.Fatal("pending triggers admitted")
	}
	g.PutTriggersOnStack()
	got := g.StackBoundDamageSources()
	if g.StackSize() != 2 || len(got) != 2 || !got[0].SameObject(first) || !got[1].SameObject(second) {
		t.Fatal("simultaneous events or APNAP placement")
	}
	if g.StackObjects()[0].Controller != a.PlayerID() || g.StackObjects()[1].Controller != b.PlayerID() {
		t.Fatal("APNAP controllers")
	}
	clone := g.Clone()
	for _, obj := range slices.Clone(g.StackObjects()) {
		g.CounterStackObject(obj.ID)
	}
	if len(g.StackBoundDamageSources()) != 0 || len(clone.StackBoundDamageSources()) != 2 {
		t.Fatal("countered references or clone")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 19 || len(clone.StackBoundDamageSources()) != 0 {
		t.Fatal("native bodies or final reference")
	}
}

func TestStackBoundDamageSourcesActivationAndResolving(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	event, chosen := BindDamageSource("event"), BindDamageSource("chosen")
	observer := NewEnchantment("Observer", "", WithAbility(WhenEvent(ActivatedAbilityEvent().WithSource(event), Sequence(ChooseDamageSourceFromStackDamageBindings(You, DamageSourceFilter{}, chosen), DealDamageFrom(FromDamageSource(chosen), You, Fixed(2))))))
	if err := ValidateCardDefinition(observer.Name(), observer); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(observer, b.PlayerID())
	card := NewArtifact("Sacrificed", "{U}", WithAbility(Activate(Costs(ManaCostOf("{1}"), SacrificeSourceCost()), GainLifeFor(You, Fixed(1)))))
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	saved := g.CaptureDamageSource(p.ID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil || g.StackSize() != 0 || len(g.StackBoundDamageSources()) != 0 || g.FindPermanent(p.ID()) == nil {
		t.Fatal("failed payment changed state")
	}
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if len(g.StackBoundDamageSources()) != 0 {
		t.Fatal("pending activation event admitted")
	}
	g.PutTriggersOnStack()
	got := g.StackBoundDamageSources()
	if len(got) != 1 || !got[0].SameObject(saved) {
		t.Fatal("activation capture after sacrifice")
	}
	g.ResolveTopOfStack()
	if b.Life() != 20 || len(g.StackBoundDamageSources()) != 0 {
		t.Fatal("resolving referer remained eligible")
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("paid activation body")
	}
}

func TestStackBoundDamageSourcesPermanentSpellAndCopy(t *testing.T) {
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
	refOne, refTwo := boundDamageObject(first), boundDamageObject(second)
	g.PushStack(refOne)
	g.PushStack(refTwo)
	if got := g.StackBoundDamageSources(); len(got) != 2 || !got[0].SameObject(first) || !got[1].SameObject(second) {
		t.Fatal("spell copy identity or order")
	}
	g.stack.objects = []*StackObject{refOne, refTwo, original, copyObject}
	g.ResolveTopOfStack()
	g.ResolveTopOfStack()
	if g.FindPermanent(card.ID()) == nil {
		t.Fatal("permanent spell did not enter")
	}
	g.ChangeObjectColor(card.ID(), core.Red)
	got := g.StackBoundDamageSources()
	if len(got) != 2 || !got[0].SameObject(first) || !got[1].SameObject(second) || got[0].Incarnation() != uuid.Nil || !got[0].HasColor(core.Blue) || !got[1].HasType(core.TypeArtifact) || got[1].Controller() != b.PlayerID() {
		t.Fatal("entry replaced a saved spell or copy")
	}
	clone := g.Clone()
	g.GetStack().RemoveByID(refOne.ID)
	g.GetStack().RemoveByID(refTwo.ID)
	if len(g.StackBoundDamageSources()) != 0 || len(clone.StackBoundDamageSources()) != 2 {
		t.Fatal("clone lost departure maps")
	}
}
