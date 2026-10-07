package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestStackTargetPermanentSources(t *testing.T) {
	for _, ability := range []bool{false, true} {
		for _, change := range []string{"live", "departed", "returned", "phased", "phase-cycle", "recolored", "face-down", "shroud", "control"} {
			t.Run(change+map[bool]string{false: "/spell", true: "/ability"}[ability], func(t *testing.T) {
				g, a, b, _ := spellSourceFixture(t)
				card := NewCreature("Target", "{U}", 4, 4, WithSubTypes("Wizard"))
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				original := g.CaptureDamageSource(p.ID())
				object := &StackObject{ID: uuid.New(), IsAbility: ability, Targets: []uuid.UUID{p.ID(), b.PlayerID(), p.ID()}}
				g.PushStack(object)
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
				case "face-down":
					g.MutablePermanent(p.ID()).FaceDown = true
				case "shroud":
					g.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				if g.CopyStackObjectDirect(object, a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				before := g.StackSize()
				sources := g.StackTargetPermanentSources()
				if len(sources) != 1 || !sources[0].SameObject(original) || g.StackSize() != before {
					t.Fatal("target identity, deduplication or pure query failed")
				}
				if change == "face-down" {
					if sources[0].Name() != "" || sources[0].HasColor(core.Blue) || sources[0].HasSubType("Wizard") {
						t.Fatal("hidden information exposed")
					}
				} else if sources[0].HasColor(core.Red) != (change == "recolored") {
					t.Fatal("wrong current or retained color")
				}
				wantController := a.PlayerID()
				if change == "control" {
					wantController = b.PlayerID()
				}
				if sources[0].Controller() != wantController {
					t.Fatal("wrong current or retained controller")
				}
				clone := g.Clone()
				sources[0].card.AddType(core.TypeArtifact)
				if g.StackTargetPermanentSources()[0].HasType(core.TypeArtifact) || clone.StackTargetPermanentSources()[0].HasType(core.TypeArtifact) {
					t.Fatal("query exposed mutable card data")
				}
				g.GetStack().RemoveByID(object.ID)
				g.GetStack().Pop()
				if len(g.StackTargetPermanentSources()) != 0 || len(clone.StackTargetPermanentSources()) != 1 {
					t.Fatal("last reference or clone lifecycle failed")
				}
			})
		}
	}
}

func TestStackTargetPermanentSourceIdentity(t *testing.T) {
	g, a, b, _ := spellSourceFixture(t)
	card := NewArtifact("Target", "")
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	old := g.CaptureDamageSource(p.ID())
	first := &StackObject{ID: uuid.New(), Targets: []uuid.UUID{p.ID()}}
	g.PushStack(first)
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	a.RemoveFromHand(card.ID())
	p = g.PutOnBattlefield(card, b.PlayerID())
	second := &StackObject{ID: uuid.New(), Targets: []uuid.UUID{p.ID()}}
	g.PushStack(second)
	got := g.StackTargetPermanentSources()
	if len(got) != 2 || !got[0].SameObject(old) || !got[1].SameObject(g.CaptureDamageSource(p.ID())) {
		t.Fatal("separate visits or insertion order lost")
	}
	g.GetStack().RemoveByID(first.ID)
	if got = g.StackTargetPermanentSources(); len(got) != 1 || got[0].Controller() != b.PlayerID() {
		t.Fatal("removing old reference changed new visit")
	}
	g.GetStack().RemoveByID(second.ID)
	for _, incarnation := range []uuid.UUID{uuid.Nil, uuid.New()} {
		g.GetStack().Push(&StackObject{ID: uuid.New(), Targets: []uuid.UUID{p.ID()}, targetIncarnations: map[uuid.UUID]uuid.UUID{p.ID(): incarnation}})
	}
	g.GetStack().Push(&StackObject{ID: uuid.New(), targetIncarnations: map[uuid.UUID]uuid.UUID{p.ID(): p.incarnationID}})
	if len(g.StackTargetPermanentSources()) != 0 {
		t.Fatal("missing, stale or unused identity accepted")
	}
	grave := NewCreature("Graveyard target", "", 1, 1)
	grave.SetOwner(a.PlayerID())
	a.AddToGraveyard(grave)
	g.PushStack(&StackObject{ID: uuid.New(), Targets: []uuid.UUID{grave.ID(), b.PlayerID(), g.StackPeek().ID}})
	if len(g.StackTargetPermanentSources()) != 0 {
		t.Fatal("other target categories accepted as permanents")
	}
}

func TestStackTargetPermanentSourceChoice(t *testing.T) {
	g, a, _, _ := spellSourceFixture(t)
	chooser := &sourceChoicePlayer{BasePlayer: a}
	g.players[0] = chooser
	p := g.PutOnBattlefield(NewArtifact("Target", ""), a.PlayerID())
	object := &StackObject{ID: uuid.New(), Targets: []uuid.UUID{p.ID()}}
	g.PushStack(object)
	one, two := BindDamageSource("one"), BindDamageSource("two")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	choice := ChooseDamageSourceFromStackTargets(You, DamageSourceHasType(core.TypeArtifact), one)
	if err := choice.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 1 || len(chooser.seen) != 1 || !FromDamageSource(one).resolveDamageSource(ctx).SameObject(g.CaptureDamageSource(p.ID())) {
		t.Fatal("filtered target choice failed")
	}
	if err := ChooseDamageSourceFromStackTargets(You, DamageSourceHasColor(core.Blue), two).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 1 || FromDamageSource(two).resolveDamageSource(ctx).Valid() || !FromDamageSource(one).resolveDamageSource(ctx).Valid() {
		t.Fatal("empty choice changed separate binding")
	}
	clone := g.Clone()
	g.GetStack().RemoveByID(object.ID)
	if err := choice.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(one).resolveDamageSource(ctx).Valid() || !FromDamageSource(one).resolveDamageSource(&EffectContext{Game: clone}).Valid() {
		t.Fatal("empty clearing or clone isolation failed")
	}
	if err := ChooseDamageSourceFromStackTargets(FromPlayer(BindPlayer("absent")), DamageSourceFilter{}, one).Apply(&EffectContext{Game: clone}); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(one).resolveDamageSource(&EffectContext{Game: clone}).Valid() {
		t.Fatal("absent chooser retained result")
	}
}

func TestStackTargetPermanentSourceDeclarations(t *testing.T) {
	binding := BindDamageSource("source")
	valid := Sequence(ChooseDamageSourceFromStackTargets(You, DamageSourceFilter{}, binding), DealDamageFrom(FromDamageSource(binding), You, Fixed(1)))
	if err := ValidateCardDefinition("Valid", NewArtifact("Valid", "", WithAbility(Activate(Costs(), valid)))); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{
		ChooseDamageSourceFromStackTargets(nil, DamageSourceFilter{}, binding),
		ChooseDamageSourceFromStackTargets(You, DamageSourceFilter{}, nil),
		ChooseDamageSourceFromStackTargets(You, DamageSourceFilter{}, BindDamageSource("")),
		ChooseDamageSourceFromStackTargets(You, DamageSourceHasColor(core.AnyColor), binding),
		Sequence(DealDamageFrom(FromDamageSource(binding), You, Fixed(1)), ChooseDamageSourceFromStackTargets(You, DamageSourceFilter{}, binding)),
	} {
		if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestStackTargetPermanentSourceRetarget(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first := g.PutOnBattlefield(NewCreature("First", "", 4, 4), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 4, 4), b.PlayerID())
	third := g.PutOnBattlefield(NewArtifact("Third", ""), b.PlayerID())
	target := DeclarePermanentTarget("creature", IsCreature)
	ability := Activate(Costs(), TapSubject(target), WithTargets(target))
	original := &StackObject{ID: uuid.New(), IsAbility: true, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.PushStack(original)
	b.chooseQueue = [][]uuid.UUID{{second.ID()}}
	copied := g.CopyStackObjectDirect(original, b.PlayerID(), true)
	if copied == nil || copied.Targets[0] != second.ID() || original.Targets[0] != first.ID() {
		t.Fatal("native retarget failed")
	}
	clone := g.Clone()
	got := g.StackTargetPermanentSources()
	if len(got) != 2 || got[0].ID() != first.ID() || got[1].ID() != second.ID() {
		t.Fatal("copy target replaced original or changed order")
	}
	g.GetStack().RemoveByID(original.ID)
	g.PushStack(&StackObject{ID: uuid.New(), Targets: []uuid.UUID{third.ID(), second.ID()}, targetIncarnations: map[uuid.UUID]uuid.UUID{first.ID(): first.incarnationID}})
	got = g.StackTargetPermanentSources()
	if len(got) != 2 || got[0].ID() != second.ID() || got[1].ID() != third.ID() {
		t.Fatal("target-list order, deduplication or stale map entry failed")
	}
	if got = clone.StackTargetPermanentSources(); len(got) != 2 || got[0].ID() != first.ID() || got[1].ID() != second.ID() {
		t.Fatal("retarget state changed clone")
	}
	g.GetStack().Pop()
	g.ResolveTopOfStack()
	if len(g.StackTargetPermanentSources()) != 0 || !g.FindPermanent(second.ID()).Tapped || g.FindPermanent(first.ID()).Tapped {
		t.Fatal("native copy resolution or reference lifetime failed")
	}
}
