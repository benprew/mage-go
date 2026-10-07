package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDamageSourcePropertyMatching(t *testing.T) {
	g, a, _, obj := spellSourceFixture(t)
	card := NewCreature("Artifact source", "{U}", 1, 2)
	card.AddType(core.TypeArtifact)
	p := g.PutOnBattlefield(card, a.PlayerID())
	spell, permanent := g.CaptureSpellDamageSource(obj.ID), g.CaptureDamageSource(p.ID())
	for _, tc := range []struct {
		filter           DamageSourceFilter
		spell, permanent bool
	}{
		{DamageSourceFilter{}, true, true},
		{DamageSourceHasColor(core.Red), true, false},
		{DamageSourceHasColor(core.Blue), false, true},
		{DamageSourceHasType(core.TypeArtifact), false, true},
		{DamageSourceHasType(core.TypeInstant), true, false},
		{AnyOfDamageSources(DamageSourceHasColor(core.Black), DamageSourceHasColor(core.Red)), true, false},
		{AllDamageSources(DamageSourceHasColor(core.Blue), DamageSourceHasType(core.TypeArtifact)), false, true},
	} {
		if tc.filter.Match(spell) != tc.spell || tc.filter.Match(permanent) != tc.permanent {
			t.Fatal("wrong property match", tc.filter)
		}
		if tc.filter.Match(DamageSource{}) {
			t.Fatal("missing source matched")
		}
	}
	if !g.ChangeObjectColor(obj.ID, core.Blue) {
		t.Fatal("color change failed")
	}
	if DamageSourceHasColor(core.Red).Match(g.CaptureSpellDamageSource(obj.ID)) || !DamageSourceHasColor(core.Red).Match(spell) {
		t.Fatal("live and saved spell information merged")
	}
	filters := []DamageSourceFilter{DamageSourceHasColor(core.Red)}
	composed := AnyOfDamageSources(filters...)
	filters[0] = DamageSourceHasColor(core.Blue)
	if !composed.Match(spell) || composed.Match(g.CaptureSpellDamageSource(obj.ID)) {
		t.Fatal("composition retained caller slice")
	}
}

func TestDamageSourcePropertyChoice(t *testing.T) {
	g, a, _, obj := spellSourceFixture(t)
	chooser := &sourceChoicePlayer{BasePlayer: a, index: 1}
	g.players[0] = chooser
	copied := g.CopyStackObjectDirect(obj, a.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy failed")
	}
	copied.Card.SetID(obj.Card.ID())
	copied.SourceID = obj.SourceID
	green := g.PutOnBattlefield(NewCreature("Green", "{G}", 1, 2), a.PlayerID())
	one, two, three, result := BindDamageSource("one"), BindDamageSource("two"), BindDamageSource("three"), BindDamageSource("result")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{one: g.CaptureSpellDamageSource(obj.ID), two: g.CaptureSpellDamageSource(copied.ID), three: g.CaptureDamageSource(green.ID())}
	refs := []DamageSourceRef{FromDamageSource(three), FromDamageSource(one), FromDamageSource(one), FromDamageSource(two)}
	effect := ChooseDamageSourceMatchingFrom(You, refs, DamageSourceHasColor(core.Red), result)
	refs[0] = nil
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	got := FromDamageSource(result).resolveDamageSource(ctx)
	if len(chooser.seen) != 2 || !got.SameObject(g.CaptureSpellDamageSource(copied.ID)) {
		t.Fatal("filter changed exact choice identities")
	}
	clone := g.Clone()
	g.resolution.damageSourceBindings[result] = DamageSource{}
	if !FromDamageSource(result).resolveDamageSource(&EffectContext{Game: clone}).SameObject(got) {
		t.Fatal("filtered result shared clone state")
	}
	before := chooser.calls
	if err := ChooseDamageSourceMatchingFrom(You, refs[1:], DamageSourceHasColor(core.White), result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != before || FromDamageSource(result).resolveDamageSource(ctx).Valid() {
		t.Fatal("empty filtered choice retained result or made a choice")
	}
}

func TestDamageSourcePropertyDeclarations(t *testing.T) {
	result := BindDamageSource("choice")
	valid := ChooseDamageSourceMatchingFrom(You, []DamageSourceRef{This}, AllDamageSources(DamageSourceHasColor(core.Red), DamageSourceHasType(core.TypeCreature)), result)
	if err := ValidateCardDefinition("Valid", NewArtifact("Valid", "", WithAbility(Activate(Costs(), valid)))); err != nil {
		t.Fatal(err)
	}
	for _, filter := range []DamageSourceFilter{
		DamageSourceHasColor(core.Colorless), DamageSourceHasColor(core.AnyColor), DamageSourceHasColor(core.Color(-1)),
		DamageSourceHasType(core.CardType(-1)), DamageSourceHasType(core.CardType(999)),
		AnyOfDamageSources(), AllDamageSources(), AnyOfDamageSources(DamageSourceHasColor(core.AnyColor)),
	} {
		effect := ChooseDamageSourceMatchingFrom(You, []DamageSourceRef{This}, filter, result)
		if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))); err == nil {
			t.Fatal("invalid filter accepted", filter)
		}
	}
	for _, effect := range []Effect{
		ChooseDamageSourceMatchingFrom(nil, []DamageSourceRef{This}, DamageSourceFilter{}, result),
		ChooseDamageSourceMatchingFrom(You, nil, DamageSourceFilter{}, result),
		ChooseDamageSourceMatchingFrom(You, []DamageSourceRef{nil}, DamageSourceFilter{}, result),
		ChooseDamageSourceMatchingFrom(You, []DamageSourceRef{This}, DamageSourceFilter{}, nil),
		ChooseDamageSourceMatchingFrom(You, []DamageSourceRef{FromDamageSource(result)}, DamageSourceFilter{}, result),
	} {
		if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))); err == nil {
			t.Fatal("invalid choice accepted")
		}
	}
}

func TestDamageSourcePropertyLiveAndRetained(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	chooser := &sourceChoicePlayer{BasePlayer: a}
	g := NewGame(chooser, b)
	card := NewCreature("Source", "{U}", 1, 2, WithKeyword(core.Shroud))
	source := g.PutOnBattlefield(card, a.PlayerID())
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: source.ID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	saved, result := BindDamageSource("saved"), BindDamageSource("result")
	if err := SnapshotDamageSource(This, saved).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	blue := DamageSourceHasColor(core.Blue)
	choose := ChooseDamageSourceMatchingFrom(You, []DamageSourceRef{This}, blue, result)
	if err := choose.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	original := FromDamageSource(result).resolveDamageSource(ctx)
	if !original.Valid() || !original.HasKeyword(core.Shroud) {
		t.Fatal("filtered choice treated shroud as targeting")
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, SetColors(This, core.Red)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	calls := chooser.calls
	if err := choose.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != calls || FromDamageSource(result).resolveDamageSource(ctx).Valid() {
		t.Fatal("live choice did not read current color")
	}
	retained := ChooseDamageSourceMatchingFrom(You, []DamageSourceRef{FromDamageSource(saved)}, blue, result)
	if err := retained.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if !FromDamageSource(result).resolveDamageSource(ctx).SameObject(original) {
		t.Fatal("saved choice lost earlier information")
	}
	g.PhaseOut(source.ID())
	if err := ChooseDamageSourceMatchingFrom(You, []DamageSourceRef{This}, DamageSourceHasColor(core.Red), result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if !FromDamageSource(result).resolveDamageSource(ctx).SameObject(original) {
		t.Fatal("explicit phase information lost identity")
	}
	g.PhaseIn([]uuid.UUID{source.ID()})
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if err := choose.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(result).resolveDamageSource(ctx).Valid() {
		t.Fatal("returned blue object replaced red LKI")
	}
	if err := ChooseDamageSourceMatchingFrom(You, []DamageSourceRef{This}, DamageSourceHasColor(core.Red), result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	got := FromDamageSource(result).resolveDamageSource(ctx)
	if !got.SameObject(original) || got.SameObject(g.CaptureDamageSource(returned.ID())) {
		t.Fatal("LKI replaced original incarnation")
	}
	overlap := Sequence(SnapshotDamageSource(This, saved), ChooseDamageSourceMatchingFrom(You, []DamageSourceRef{FromDamageSource(saved)}, DamageSourceHasColor(core.Red), saved))
	if err := ValidateCardDefinition("Overlap", NewArtifact("Overlap", "", WithAbility(Activate(Costs(), overlap)))); err != nil {
		t.Fatal(err)
	}
	if err := overlap.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if !FromDamageSource(saved).resolveDamageSource(ctx).SameObject(original) {
		t.Fatal("filtered overlap cleared input before selection")
	}
}
