package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLiveDamageSourcesObjects(t *testing.T) {
	g, a, b, spell := spellSourceFixture(t)
	land := g.PutOnBattlefield(NewLand("Land"), a.PlayerID())
	creature := g.PutOnBattlefield(NewCreature("Shrouded", "{U}", 1, 2, WithKeyword(core.Shroud)), b.PlayerID())
	phased := g.PutOnBattlefield(NewArtifact("Phased", ""), a.PlayerID())
	g.PhaseOut(phased.ID())
	copySpell := g.CopyStackObjectDirect(spell, b.PlayerID(), false)
	if copySpell == nil {
		t.Fatal("copy failed")
	}
	copySpell.Card.SetID(spell.Card.ID())
	permanentCard := NewCreature("Permanent spell", "{G}", 3, 3)
	permanentCard.SetOwner(a.PlayerID())
	permanentSpell := &StackObject{ID: uuid.New(), SourceID: permanentCard.ID(), Card: permanentCard, Controller: a.PlayerID()}
	g.stack.Push(permanentSpell)
	g.stack.Push(&StackObject{ID: uuid.New(), SourceID: phased.ID(), IsAbility: true, Controller: a.PlayerID()})
	g.ApplyContinuousEffects()
	before := g.StackSize()
	sources := g.LiveDamageSources()
	if len(sources) != 5 || g.StackSize() != before {
		t.Fatalf("sources=%d, stack=%d", len(sources), g.StackSize())
	}
	want := []DamageSource{damageSourceFromPermanent(land), damageSourceFromPermanent(creature), g.CaptureSpellDamageSource(spell.ID), g.CaptureSpellDamageSource(copySpell.ID), g.CaptureSpellDamageSource(permanentSpell.ID)}
	for _, source := range want {
		if !slices.ContainsFunc(sources, source.SameObject) {
			t.Fatalf("missing %s (%s)", source.Name(), source.StackID())
		}
	}
	for _, source := range sources {
		if source.ID() == phased.ID() || !source.Valid() {
			t.Fatal("included phased permanent or ability")
		}
	}
	if !sources[len(sources)-1].SameObject(want[len(want)-1]) {
		t.Fatal("spell order changed")
	}
	if !g.ChangeObjectColor(spell.ID, core.Blue) {
		t.Fatal("color change failed")
	}
	fresh := g.LiveDamageSources()
	if !slices.ContainsFunc(fresh, func(s DamageSource) bool { return s.SameObject(want[2]) && s.HasColor(core.Blue) }) || !want[2].HasColor(core.Red) {
		t.Fatal("live and saved colors merged")
	}
	clone := g.Clone()
	g.stack.RemoveByID(copySpell.ID)
	g.BouncePermanentToHand(g.FindPermanent(creature.ID()))
	if len(g.LiveDamageSources()) != 3 || len(clone.LiveDamageSources()) != 5 {
		t.Fatal("departed objects remained live or clone changed")
	}
	for _, source := range sources {
		source.card.AddType(core.TypeEnchantment)
	}
	if slices.ContainsFunc(clone.LiveDamageSources(), func(s DamageSource) bool { return s.HasType(core.TypeEnchantment) }) {
		t.Fatal("query exposed mutable card state")
	}
}

func TestLiveDamageSourceChoiceResolution(t *testing.T) {
	g, a, _, spell := spellSourceFixture(t)
	chooser := &sourceChoicePlayer{BasePlayer: a}
	g.players[0] = chooser
	result := BindDamageSource("live choice")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	effect := ChooseDamageSourceFromLiveObjects(You, DamageSourceHasColor(core.Blue), result)
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 0 || FromDamageSource(result).resolveDamageSource(ctx).Valid() {
		t.Fatal("empty list made a choice")
	}
	g.ChangeObjectColor(spell.ID, core.Blue)
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	chosen := FromDamageSource(result).resolveDamageSource(ctx)
	if chooser.calls != 1 || len(chooser.seen) != 1 || !chosen.SameObject(g.CaptureSpellDamageSource(spell.ID)) {
		t.Fatal("choice did not query resolution state")
	}
	clone := g.Clone()
	g.stack.RemoveByID(spell.ID)
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 1 || FromDamageSource(result).resolveDamageSource(ctx).Valid() || !FromDamageSource(result).resolveDamageSource(&EffectContext{Game: clone}).SameObject(chosen) {
		t.Fatal("empty choice retained result or changed clone")
	}
	missing := ChooseDamageSourceFromLiveObjects(FromPlayer(BindPlayer("absent")), DamageSourceFilter{}, result)
	if err := missing.Apply(&EffectContext{Game: clone}); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(result).resolveDamageSource(&EffectContext{Game: clone}).Valid() {
		t.Fatal("absent player retained result")
	}
}

func TestLiveDamageSourceChoiceDeclarations(t *testing.T) {
	result := BindDamageSource("live choice")
	choice := ChooseDamageSourceFromLiveObjects(You, DamageSourceFilter{}, result)
	valid := Sequence(choice, DealDamageFrom(FromDamageSource(result), You, Fixed(1)))
	if err := ValidateCardDefinition("Valid", NewArtifact("Valid", "", WithAbility(Activate(Costs(), valid)))); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{
		ChooseDamageSourceFromLiveObjects(nil, DamageSourceFilter{}, result),
		ChooseDamageSourceFromLiveObjects(You, DamageSourceFilter{}, nil),
		ChooseDamageSourceFromLiveObjects(You, DamageSourceFilter{}, BindDamageSource("")),
		ChooseDamageSourceFromLiveObjects(You, DamageSourceHasColor(core.AnyColor), result),
		ChooseDamageSourceFromLiveObjects(You, AnyOfDamageSources(), result),
		Sequence(DealDamageFrom(FromDamageSource(result), You, Fixed(1)), choice),
	} {
		if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestLiveDamageSourcesFaceDown(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Hidden name", "{U}", 4, 4, WithSubTypes("Dragon"), WithKeyword(core.Flying))
	p := g.PutOnBattlefield(card, a.PlayerID())
	p.FaceDown = true
	g.ApplyContinuousEffects()
	sources := g.LiveDamageSources()
	if len(sources) != 1 || sources[0].Name() != "" || sources[0].HasColor(core.Blue) || sources[0].HasKeyword(core.Flying) || sources[0].HasSubType("Dragon") || !sources[0].HasType(core.TypeCreature) {
		t.Fatal("source choice exposed face-down characteristics")
	}
	g.BouncePermanentToHand(p)
	lki := damageSourceFromLKI(g.zones.LKIByIncarnation(sources[0].Incarnation()))
	if !lki.SameObject(sources[0]) || lki.Name() != "" || lki.HasColor(core.Blue) || lki.HasKeyword(core.Flying) || lki.HasSubType("Dragon") {
		t.Fatal("retained source exposed face-down characteristics")
	}
}

func TestLiveDamageSourceChoiceIndependentBindings(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	chooser := &sourceChoicePlayer{BasePlayer: a}
	g := NewGame(chooser, b)
	first := g.PutOnBattlefield(NewCreature("First", "{R}", 1, 1), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "{R}", 1, 1), b.PlayerID())
	one, two := BindDamageSource("one"), BindDamageSource("two")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := ChooseDamageSourceFromLiveObjects(You, DamageSourceHasColor(core.Red), one).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	chooser.index = 1
	if err := ChooseDamageSourceFromLiveObjects(You, DamageSourceHasColor(core.Red), two).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if !FromDamageSource(one).resolveDamageSource(ctx).SameObject(g.CaptureDamageSource(first.ID())) || !FromDamageSource(two).resolveDamageSource(ctx).SameObject(g.CaptureDamageSource(second.ID())) {
		t.Fatal("choices shared a result")
	}
	clone := g.Clone()
	g.BouncePermanentToHand(g.FindPermanent(first.ID()))
	chooser.index = -1
	if err := ChooseDamageSourceFromLiveObjects(You, DamageSourceFilter{}, one).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if !FromDamageSource(one).resolveDamageSource(ctx).SameObject(g.CaptureDamageSource(second.ID())) || !FromDamageSource(one).resolveDamageSource(&EffectContext{Game: clone}).SameObject(clone.CaptureDamageSource(first.ID())) {
		t.Fatal("repeated choice changed clone result")
	}
}
