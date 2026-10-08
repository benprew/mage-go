package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestStackBoundPermanentSources(t *testing.T) {
	for _, change := range []string{"live", "departed", "returned", "phased", "phase-cycle", "recolored", "control", "face-down", "shroud"} {
		t.Run(change, func(t *testing.T) {
			g, a, b, _ := spellSourceFixture(t)
			card := NewCreature("Bound", "{U}", 1, 4, WithSubTypes("Wizard"))
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			original := g.CaptureDamageSource(p.ID())
			first, second := BindPermanent("first"), BindPermanent("second")
			object := &StackObject{ID: uuid.New(), IsAbility: true, Controller: a.PlayerID(), capturedPermanents: map[*PermanentBinding]capturedPermanent{first: {p.ID(), p.incarnationID}, second: {p.ID(), p.incarnationID}}}
			g.PushStack(object)
			if g.CopyStackObjectDirect(object, b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			switch change {
			case "departed", "returned":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				if change == "returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
					g.ChangeObjectColor(card.ID(), core.Red)
				}
			case "phased", "phase-cycle":
				ids := g.PhaseOut(p.ID())
				if change == "phase-cycle" {
					g.PhaseIn(ids)
				}
			case "recolored":
				g.ChangeObjectColor(p.ID(), core.Red)
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "face-down":
				g.MutablePermanent(p.ID()).FaceDown = true
			case "shroud":
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
			}
			size := g.StackSize()
			got := g.StackBoundPermanentSources()
			if len(got) != 1 || !got[0].SameObject(original) || g.StackSize() != size {
				t.Fatal("identity, deduplication or pure query failed")
			}
			if change == "face-down" {
				if got[0].Name() != "" || got[0].HasColor(core.Blue) || got[0].HasSubType("Wizard") {
					t.Fatal("hidden information exposed")
				}
			} else if got[0].HasColor(core.Red) != (change == "recolored") {
				t.Fatal("wrong live or retained color")
			}
			controller := a.PlayerID()
			if change == "control" {
				controller = b.PlayerID()
			}
			if got[0].Controller() != controller {
				t.Fatal("wrong live or retained controller")
			}
			clone := g.Clone()
			got[0].card.AddType(core.TypeArtifact)
			if g.StackBoundPermanentSources()[0].HasType(core.TypeArtifact) || clone.StackBoundPermanentSources()[0].HasType(core.TypeArtifact) {
				t.Fatal("query exposed mutable card data")
			}
			g.GetStack().RemoveByID(object.ID)
			if len(g.StackBoundPermanentSources()) != 1 {
				t.Fatal("copy lost reference")
			}
			g.GetStack().Pop()
			if len(g.StackBoundPermanentSources()) != 0 || len(clone.StackBoundPermanentSources()) != 1 {
				t.Fatal("final reference or clone lifetime failed")
			}
		})
	}
}

func TestStackBoundPermanentSourceOrderAndBoundaries(t *testing.T) {
	g, a, b, _ := spellSourceFixture(t)
	card := NewArtifact("Repeated", "")
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	old := g.CaptureDamageSource(p.ID())
	binding := BindPermanent("same declaration")
	first := &StackObject{ID: uuid.New(), capturedPermanents: map[*PermanentBinding]capturedPermanent{binding: {p.ID(), p.incarnationID}}}
	g.PushStack(first)
	g.BouncePermanentToHand(p)
	a.RemoveFromHand(card.ID())
	p = g.PutOnBattlefield(card, b.PlayerID())
	q := g.PutOnBattlefield(NewArtifact("Other", ""), b.PlayerID())
	second := &StackObject{ID: uuid.New(), capturedPermanents: map[*PermanentBinding]capturedPermanent{binding: {p.ID(), p.incarnationID}, BindPermanent("other"): {q.ID(), q.incarnationID}}}
	g.PushStack(second)
	got := g.StackBoundPermanentSources()
	if len(got) != 3 || !got[0].SameObject(old) {
		t.Fatal("stack insertion order or incarnation separation failed")
	}
	sorted := []uuid.UUID{p.ID(), q.ID()}
	slices.SortFunc(sorted, func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) })
	if got[1].ID() != sorted[0] || got[2].ID() != sorted[1] {
		t.Fatal("binding map order is not stable")
	}
	for range 20 {
		next := g.StackBoundPermanentSources()
		for i := range got {
			if !next[i].SameObject(got[i]) {
				t.Fatal("query order changed")
			}
		}
	}
	g.GetStack().RemoveByID(first.ID)
	g.GetStack().RemoveByID(second.ID)
	invalid := map[*PermanentBinding]capturedPermanent{
		nil: {p.ID(), p.incarnationID}, BindPermanent("missing"): {p.ID(), uuid.Nil}, BindPermanent("unknown"): {p.ID(), uuid.New()}, BindPermanent("player"): {b.PlayerID(), uuid.New()}, BindPermanent("stack"): {g.StackPeek().ID, uuid.New()},
	}
	g.PushStack(&StackObject{ID: uuid.New(), capturedPermanents: invalid})
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{binding: {p.ID(), p.incarnationID}}
	if len(g.StackBoundPermanentSources()) != 0 {
		t.Fatal("invalid identities or resolution bindings admitted")
	}
}

func TestStackBoundPermanentSourceChoice(t *testing.T) {
	g, a, _, _ := spellSourceFixture(t)
	chooser := &sourceChoicePlayer{BasePlayer: a, index: -1}
	g.players[0] = chooser
	p := g.PutOnBattlefield(NewArtifact("Bound", ""), a.PlayerID())
	object := &StackObject{ID: uuid.New(), capturedPermanents: map[*PermanentBinding]capturedPermanent{BindPermanent("bound"): {p.ID(), p.incarnationID}}}
	g.PushStack(object)
	one, two := BindDamageSource("one"), BindDamageSource("two")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	choice := ChooseDamageSourceFromStackBindings(You, DamageSourceHasType(core.TypeArtifact), one)
	if err := choice.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 1 || !FromDamageSource(one).resolveDamageSource(ctx).SameObject(g.CaptureDamageSource(p.ID())) {
		t.Fatal("filtered choice or invalid-answer fallback failed")
	}
	chooser.seen[0].card.AddType(core.TypeCreature)
	if FromDamageSource(one).resolveDamageSource(ctx).HasType(core.TypeCreature) {
		t.Fatal("choice exposed saved result")
	}
	if err := ChooseDamageSourceFromStackBindings(You, DamageSourceHasColor(core.Blue), two).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 1 || FromDamageSource(two).resolveDamageSource(ctx).Valid() || !FromDamageSource(one).resolveDamageSource(ctx).Valid() {
		t.Fatal("bindings crossed")
	}
	clone := g.Clone()
	g.GetStack().RemoveByID(object.ID)
	if err := choice.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(one).resolveDamageSource(ctx).Valid() || !FromDamageSource(one).resolveDamageSource(&EffectContext{Game: clone}).Valid() {
		t.Fatal("empty clearing or clone isolation failed")
	}
	if err := ChooseDamageSourceFromStackBindings(FromPlayer(BindPlayer("absent")), DamageSourceFilter{}, one).Apply(&EffectContext{Game: clone}); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(one).resolveDamageSource(&EffectContext{Game: clone}).Valid() {
		t.Fatal("absent player kept old choice")
	}
}

func TestStackBoundPermanentSourceDeclarations(t *testing.T) {
	binding := BindDamageSource("source")
	choice := ChooseDamageSourceFromStackBindings(You, DamageSourceFilter{}, binding)
	if err := ValidateCardDefinition("Valid", NewArtifact("Valid", "", WithAbility(Activate(Costs(), Sequence(choice, DealDamageFrom(FromDamageSource(binding), You, Fixed(1))))))); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{
		ChooseDamageSourceFromStackBindings(nil, DamageSourceFilter{}, binding),
		ChooseDamageSourceFromStackBindings(You, DamageSourceFilter{}, nil),
		ChooseDamageSourceFromStackBindings(You, DamageSourceFilter{}, BindDamageSource("")),
		ChooseDamageSourceFromStackBindings(You, DamageSourceHasColor(core.AnyColor), binding),
		Sequence(DealDamageFrom(FromDamageSource(binding), You, Fixed(1)), choice),
	} {
		if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestStackBoundPermanentSourceNativeCaptures(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	bound := BindPermanent("dead")
	for _, player := range []*BasePlayer{a, b} {
		observer := NewEnchantment("Observer", "", WithAbility(WhenEvent(CreatureDies().WithPermanent(bound), DealDamageFrom(DamageSourceOf(From(bound)), You, Fixed(1)))))
		if err := ValidateCardDefinition(observer.Name(), observer); err != nil {
			t.Fatal(err)
		}
		g.PutOnBattlefield(observer, player.PlayerID())
	}
	firstCard, secondCard := NewCreature("First", "{U}", 1, 1), NewCreature("Second", "{R}", 1, 1)
	firstCard.SetOwner(a.PlayerID())
	secondCard.SetOwner(b.PlayerID())
	first := g.PutOnBattlefield(firstCard, a.PlayerID())
	second := g.PutOnBattlefield(secondCard, b.PlayerID())
	originals := []DamageSource{g.CaptureDamageSource(first.ID()), g.CaptureDamageSource(second.ID())}
	if err := ApplyEffect(g, DestroyEach(MatchingPermanents(IsCreature)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.StackBoundPermanentSources()) != 0 {
		t.Fatal("pending triggers admitted before stack placement")
	}
	g.PutTriggersOnStack()
	got := g.StackBoundPermanentSources()
	if g.StackSize() != 4 || len(got) != 2 {
		t.Fatal("simultaneous death captures missing or duplicated")
	}
	for _, original := range originals {
		if !slices.ContainsFunc(got, original.SameObject) {
			t.Fatal("distinct pending events lost identity")
		}
	}
	objects := slices.Clone(g.StackObjects())
	if objects[0].Controller != a.PlayerID() || objects[1].Controller != a.PlayerID() || objects[2].Controller != b.PlayerID() || objects[3].Controller != b.PlayerID() {
		t.Fatal("APNAP stack placement changed")
	}
	clone := g.Clone()
	for _, object := range objects {
		g.CounterStackObject(object.ID)
	}
	if len(g.StackBoundPermanentSources()) != 0 || len(clone.StackBoundPermanentSources()) != 2 {
		t.Fatal("countering changed clone or retained reference")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 18 || clone.GetPlayer(b.PlayerID()).Life() != 18 || len(clone.StackBoundPermanentSources()) != 0 {
		t.Fatal("event body or final reference failed")
	}
}

func TestStackBoundPermanentSourceNativePayment(t *testing.T) {
	for _, spell := range []bool{false, true} {
		t.Run(map[bool]string{false: "activation", true: "spell"}[spell], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			paid := BindPermanent("paid")
			body := DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(2))
			sacrifice := SacrificeMatchingCostWithPermanent(IsCreature, "Sacrifice a creature", paid)
			var card Card
			if spell {
				card = NewInstant("Offering", "{1}", WithAbility(Spell(body)), WithAdditionalCost(sacrifice))
			} else {
				card = NewArtifact("Offering", "", WithAbility(Activate(Costs(ManaCostOf("{1}"), sacrifice), body)))
			}
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			if spell {
				a.AddToHand(card)
			} else {
				g.PutOnBattlefield(card, a.PlayerID())
			}
			creature := NewCreature("Paid", "{U}", 1, 1, WithKeyword(core.Lifelink))
			creature.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(creature, a.PlayerID())
			original := g.CaptureDamageSource(p.ID())
			activate := func() error {
				if spell {
					return g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0)
				}
				return g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil)
			}
			if err := activate(); err == nil || g.FindPermanent(p.ID()) == nil || g.StackSize() != 0 || len(g.StackBoundPermanentSources()) != 0 {
				t.Fatal("unpaid operation changed sacrifice or source list")
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := activate(); err != nil {
				t.Fatal(err)
			}
			got := g.StackBoundPermanentSources()
			if len(got) != 1 || !got[0].SameObject(original) || g.FindPermanent(p.ID()) != nil {
				t.Fatal("native payment capture missing")
			}
			a.RemoveFromGraveyard(creature.ID())
			g.PutOnBattlefield(creature, b.PlayerID())
			g.ChangeObjectColor(creature.ID(), core.Red)
			if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			if got := g.StackBoundPermanentSources(); len(got) != 1 || got[0].Controller() != a.PlayerID() || !got[0].HasColor(core.Blue) {
				t.Fatal("payment reference followed returned object")
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			for _, game := range []*Game{g, clone} {
				if game.GetPlayer(a.PlayerID()).Life() != 22 || game.GetPlayer(b.PlayerID()).Life() != 18 || len(game.StackBoundPermanentSources()) != 0 {
					t.Fatal("paid source body, copy controller or clone failed")
				}
			}
		})
	}
}

func TestStackBoundPermanentSourceResolvingLifetime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	chooser := &sourceChoicePlayer{BasePlayer: a}
	g := NewGame(chooser, b)
	event, choice := BindPermanent("event"), BindDamageSource("choice")
	observer := NewEnchantment("Observer", "", WithAbility(WhenEvent(CreatureDies().WithPermanent(event), Sequence(
		ChooseDamageSourceFromStackBindings(You, DamageSourceFilter{}, choice), DealDamageFrom(FromDamageSource(choice), You, Fixed(3))))))
	if err := ValidateCardDefinition(observer.Name(), observer); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(observer, a.PlayerID())
	card := NewCreature("Dead", "", 1, 1)
	card.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(card, b.PlayerID())
	saved := g.CaptureDamageSource(p.ID())
	g.DestroyPermanent(p)
	g.PutTriggersOnStack()
	if len(g.StackBoundPermanentSources()) != 1 {
		t.Fatal("capture missing")
	}
	g.ResolveStack()
	if chooser.calls != 1 || a.Life() != 17 || len(g.StackBoundPermanentSources()) != 0 || !chooser.seen[0].SameObject(saved) {
		t.Fatal("resolving event reference or final frame lifetime failed")
	}
}

func TestStackBoundPermanentSourceAnnouncement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	saved := BindPermanent("announcement")
	card := NewArtifact("Source", "{U}", WithAbility(Activate(Costs(SacrificeSourceCost()),
		DealDamageFrom(DamageSourceOf(From(saved)), You, Fixed(2)), CapturePermanentBeforePayment(This, saved))))
	card.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(card, a.PlayerID())
	original := g.CaptureDamageSource(p.ID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if got := g.StackBoundPermanentSources(); len(got) != 1 || !got[0].SameObject(original) {
		t.Fatal("pre-payment capture lost sacrificed source")
	}
	g.ResolveStack()
	if a.Life() != 18 || len(g.StackBoundPermanentSources()) != 0 {
		t.Fatal("announcement body or reference lifetime failed")
	}
}
