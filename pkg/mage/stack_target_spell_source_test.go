package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestStackTargetSpellSources(t *testing.T) {
	for _, removal := range []string{"live", "pop", "id", "source"} {
		t.Run(removal, func(t *testing.T) {
			g, a, b, spell := spellSourceFixture(t)
			original := g.CaptureSpellDamageSource(spell.ID)
			referer := &StackObject{ID: uuid.New(), IsAbility: true, Targets: []uuid.UUID{spell.ID, spell.ID, spell.SourceID, b.PlayerID()}}
			g.PushStack(referer)
			g.ChangeObjectColor(spell.ID, core.Blue)
			spell.Controller = b.PlayerID()
			cloneBefore := g.Clone()
			if removal != "live" {
				switch removal {
				case "pop":
					g.stack.objects[0], g.stack.objects[1] = g.stack.objects[1], g.stack.objects[0]
					g.GetStack().Pop()
				case "id":
					g.GetStack().RemoveByID(spell.ID)
				case "source":
					g.GetStack().RemoveBySourceID(spell.SourceID)
				}
				a.AddToHand(spell.Card)
				a.ManaPool().Add(core.Red, 1)
				if err := g.CastSpellByID(a.PlayerID(), spell.Card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
			}
			copied := g.CopyStackObjectDirect(referer, a.PlayerID(), false)
			if copied == nil {
				t.Fatal("copy failed")
			}
			size := g.StackSize()
			sources := g.StackTargetSpellSources()
			if len(sources) != 1 || !sources[0].SameObject(original) || !sources[0].HasColor(core.Blue) || sources[0].Controller() != b.PlayerID() || g.StackSize() != size {
				t.Fatalf("wrong exact live or departed source: %+v", sources)
			}
			if !sources[0].HasType(core.TypeInstant) || sources[0].Owner() != a.PlayerID() {
				t.Fatal("source characteristics lost")
			}
			sources[0].card.AddType(core.TypeArtifact)
			if g.StackTargetSpellSources()[0].HasType(core.TypeArtifact) || cloneBefore.StackTargetSpellSources()[0].HasType(core.TypeArtifact) {
				t.Fatal("query exposed mutable card data")
			}
			cloneAfter := g.Clone()
			g.GetStack().RemoveByID(referer.ID)
			g.GetStack().RemoveByID(copied.ID)
			if len(g.StackTargetSpellSources()) != 0 || len(cloneAfter.StackTargetSpellSources()) != 1 {
				t.Fatal("last reference or clone lifetime failed")
			}
		})
	}
}

func TestStackTargetSpellSourceDepartureAndCopy(t *testing.T) {
	g, a, _, spell := spellSourceFixture(t)
	original := g.CaptureSpellDamageSource(spell.ID)
	referer := &StackObject{ID: uuid.New(), IsAbility: true, Targets: []uuid.UUID{spell.ID}}
	g.PushStack(referer)
	clone := g.Clone()
	g.ChangeObjectColor(spell.ID, core.Green)
	g.GetStack().RemoveByID(spell.ID)
	clone.GetStack().RemoveByID(spell.ID)
	if !g.StackTargetSpellSources()[0].HasColor(core.Green) || !clone.StackTargetSpellSources()[0].HasColor(core.Red) {
		t.Fatal("departure snapshot changed clone")
	}
	secondCard := NewArtifact("Permanent spell", "{U}")
	secondCard.SetOwner(a.PlayerID())
	second := &StackObject{ID: uuid.New(), Card: secondCard, SourceID: secondCard.ID(), Controller: a.PlayerID()}
	g.PushStack(second)
	target := DeclareSpellTarget("spell")
	ability := Activate(Costs(), Counter(target), WithTargets(target))
	referer.TargetSpecs = ability.Targets()
	chooser := newRecordingPlayer("Copy controller")
	chooser.chooseQueue = [][]uuid.UUID{{second.ID}}
	g.players[1] = chooser
	copied := g.CopyStackObjectDirect(referer, chooser.PlayerID(), true)
	if copied == nil || copied.Targets[0] != second.ID {
		t.Fatal("native retarget failed")
	}
	got := g.StackTargetSpellSources()
	if len(got) != 2 || !got[0].SameObject(original) || got[1].StackID() != second.ID {
		t.Fatal("retarget, insertion order or permanent spell category failed")
	}
	g.GetStack().RemoveByID(referer.ID)
	if got = g.StackTargetSpellSources(); len(got) != 1 || got[0].StackID() != second.ID {
		t.Fatal("unused copied snapshot added a source")
	}
}

func TestStackTargetSpellSourceBoundaries(t *testing.T) {
	g, a, _, spell := spellSourceFixture(t)
	permanent := g.PutOnBattlefield(NewArtifact("Permanent", ""), a.PlayerID())
	ability := &StackObject{ID: uuid.New(), IsAbility: true, Card: spell.Card}
	g.PushStack(ability)
	g.PushStack(&StackObject{ID: uuid.New(), Targets: []uuid.UUID{ability.ID, permanent.ID(), spell.Card.ID(), uuid.New()}})
	if len(g.StackTargetSpellSources()) != 0 {
		t.Fatal("non-spell target accepted")
	}
	g.PushStack(&StackObject{ID: uuid.New(), Targets: []uuid.UUID{spell.ID}})
	g.GetStack().RemoveByID(spell.ID)
	g.ResolveTopOfStack()
	if len(g.StackTargetSpellSources()) != 0 {
		t.Fatal("resolving referer stayed in category")
	}
}

func TestStackTargetSpellSourceChoice(t *testing.T) {
	g, a, b, spell := spellSourceFixture(t)
	chooser := &sourceChoicePlayer{BasePlayer: a}
	g.players[0] = chooser
	referer := &StackObject{ID: uuid.New(), IsAbility: true, Targets: []uuid.UUID{spell.ID}}
	g.PushStack(referer)
	one, two := BindDamageSource("one"), BindDamageSource("two")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	choice := ChooseDamageSourceFromStackSpellTargets(You, DamageSourceHasColor(core.Red), one)
	if err := choice.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 1 || !FromDamageSource(one).resolveDamageSource(ctx).SameObject(g.CaptureSpellDamageSource(spell.ID)) {
		t.Fatal("choice lost identity")
	}
	if err := ChooseDamageSourceFromStackSpellTargets(You, DamageSourceHasColor(core.Blue), two).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 1 || FromDamageSource(two).resolveDamageSource(ctx).Valid() || !FromDamageSource(one).resolveDamageSource(ctx).Valid() {
		t.Fatal("empty choice changed another binding")
	}
	clone := g.Clone()
	g.GetStack().RemoveByID(referer.ID)
	if err := choice.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(one).resolveDamageSource(ctx).Valid() || !FromDamageSource(one).resolveDamageSource(&EffectContext{Game: clone}).Valid() {
		t.Fatal("empty clearing or frame clone failed")
	}
	if err := ChooseDamageSourceFromStackSpellTargets(FromPlayer(BindPlayer("absent")), DamageSourceFilter{}, one).Apply(&EffectContext{Game: clone, Controller: b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(one).resolveDamageSource(&EffectContext{Game: clone}).Valid() {
		t.Fatal("absent chooser retained result")
	}
}

func TestStackTargetSpellSourceDeclarations(t *testing.T) {
	binding := BindDamageSource("source")
	valid := Sequence(ChooseDamageSourceFromStackSpellTargets(You, DamageSourceFilter{}, binding), DealDamageFrom(FromDamageSource(binding), You, Fixed(1)))
	if err := ValidateCardDefinition("Valid", NewArtifact("Valid", "", WithAbility(Activate(Costs(), valid)))); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{
		ChooseDamageSourceFromStackSpellTargets(nil, DamageSourceFilter{}, binding),
		ChooseDamageSourceFromStackSpellTargets(You, DamageSourceFilter{}, nil),
		ChooseDamageSourceFromStackSpellTargets(You, DamageSourceFilter{}, BindDamageSource("")),
		ChooseDamageSourceFromStackSpellTargets(You, DamageSourceHasColor(core.AnyColor), binding),
		Sequence(DealDamageFrom(FromDamageSource(binding), You, Fixed(1)), ChooseDamageSourceFromStackSpellTargets(You, DamageSourceFilter{}, binding)),
	} {
		if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestStackTargetSpellSourceCopiesAndRemoval(t *testing.T) {
	for _, removal := range []string{"resolve", "counter-copy", "ante"} {
		t.Run(removal, func(t *testing.T) {
			g, a, b, spell := spellSourceFixture(t)
			copied := g.CopyStackObjectDirect(spell, b.PlayerID(), false)
			if copied == nil {
				t.Fatal("spell copy failed")
			}
			referer := &StackObject{ID: uuid.New(), IsAbility: true, Targets: []uuid.UUID{copied.ID, spell.ID, copied.ID}}
			g.PushStack(referer)
			got := g.StackTargetSpellSources()
			if len(got) != 2 || got[0].StackID() != copied.ID || got[1].StackID() != spell.ID {
				t.Fatal("spell copies or target order merged")
			}
			g.ChangeObjectColor(spell.ID, core.Blue)
			g.ChangeObjectColor(copied.ID, core.Green)
			switch removal {
			case "resolve":
				g.stack.objects[1], g.stack.objects[2] = g.stack.objects[2], g.stack.objects[1]
				g.ResolveTopOfStack()
				if b.Life() != 19 {
					t.Fatal("native copy did not resolve")
				}
			case "counter-copy":
				g.CounterStackObject(copied.ID)
			case "ante":
				g.anteEnabled = true
				if err := g.MoveToAnte(a.PlayerID(), spell.Card.ID()); err != nil {
					t.Fatal(err)
				}
			}
			got = g.StackTargetSpellSources()
			if len(got) != 2 || !got[0].HasColor(core.Green) || !got[1].HasColor(core.Blue) {
				t.Fatal("removal lost current departure information")
			}
			clone := g.Clone()
			g.GetStack().RemoveByID(referer.ID)
			if len(g.StackTargetSpellSources()) != 0 || len(clone.StackTargetSpellSources()) != 2 {
				t.Fatal("copy reference or clone lifetime failed")
			}
		})
	}
}
