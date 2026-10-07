package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type sourceChoicePlayer struct {
	*BasePlayer
	index int
	calls int
	seen  []DamageSource
}

func (p *sourceChoicePlayer) ChooseDamageSource(sources []DamageSource, _ *Game) int {
	p.calls++
	p.seen = sources
	return p.index
}

func TestDamageSourceChoiceIdentity(t *testing.T) {
	g, a, _, obj := spellSourceFixture(t)
	chooser := &sourceChoicePlayer{BasePlayer: a, index: 1}
	g.players[0] = chooser
	copyObj := g.CopyStackObjectDirect(obj, a.PlayerID(), false)
	if copyObj == nil {
		t.Fatal("copy failed")
	}
	copyObj.Card.SetID(obj.Card.ID())
	copyObj.SourceID = obj.SourceID
	original, copied, result := BindDamageSource("original"), BindDamageSource("copy"), BindDamageSource("choice")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{original: g.CaptureSpellDamageSource(obj.ID), copied: g.CaptureSpellDamageSource(copyObj.ID)}
	refs := []DamageSourceRef{FromDamageSource(original), FromDamageSource(original), FromDamageSource(copied)}
	effect := ChooseDamageSourceFrom(You, refs, result)
	refs[0] = nil
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	got := FromDamageSource(result).resolveDamageSource(ctx)
	if len(chooser.seen) != 2 || chooser.calls != 1 || !got.SameObject(g.CaptureSpellDamageSource(copyObj.ID)) {
		t.Fatal("choice merged stack identities or retained caller slice")
	}
	if !got.HasColor(core.Red) || !got.HasType(core.TypeInstant) {
		t.Fatal("choice lost source information")
	}
	clone := g.Clone()
	g.GetStack().Pop()
	g.GetStack().Pop()
	if !FromDamageSource(result).resolveDamageSource(&EffectContext{Game: clone}).SameObject(got) {
		t.Fatal("clone lost binding")
	}
	g.resolution.damageSourceBindings[result] = DamageSource{}
	if !FromDamageSource(result).resolveDamageSource(&EffectContext{Game: clone}).SameObject(got) {
		t.Fatal("clone shared binding map")
	}
	chooser.seen[1].card.AddType(core.TypeArtifact)
	if got.HasType(core.TypeArtifact) {
		t.Fatal("choice options shared card data with the saved result")
	}
	if FromDamageSource(result).resolveDamageSource(&EffectContext{Game: clone}).HasType(core.TypeArtifact) {
		t.Fatal("choice exposed saved card state")
	}
}

func TestDamageSourceChoicePermanentAndAbsence(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	player := &sourceChoicePlayer{BasePlayer: a}
	g := NewGame(player, b)
	card := NewCreature("Source", "{G}", 2, 3, WithKeyword(core.Shroud))
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: p.ID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
	result := BindDamageSource("chosen")
	effect := ChooseDamageSourceFrom(You, []DamageSourceRef{This}, result)
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	original := FromDamageSource(result).resolveDamageSource(ctx)
	if !original.HasKeyword(core.Shroud) || original.Incarnation() != p.incarnationID {
		t.Fatal("choice treated shroud as targeting or lost incarnation")
	}
	g.PhaseOut(p.ID())
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if !FromDamageSource(result).resolveDamageSource(ctx).SameObject(original) {
		t.Fatal("explicit phased reference lost identity")
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	got := FromDamageSource(result).resolveDamageSource(ctx)
	if !got.SameObject(original) || got.SameObject(damageSourceFromPermanent(returned)) {
		t.Fatal("returned permanent replaced explicit LKI source")
	}
	missing := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: uuid.New()}
	before := player.calls
	if err := effect.Apply(missing); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(result).resolveDamageSource(missing).Valid() || player.calls != before {
		t.Fatal("empty choice retained binding or asked player")
	}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	absent := ChooseDamageSourceFrom(FromPlayer(BindPlayer("absent")), []DamageSourceRef{This}, result)
	if err := absent.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(result).resolveDamageSource(ctx).Valid() {
		t.Fatal("missing chooser retained binding")
	}
}

func TestDamageSourceChoiceDeclaration(t *testing.T) {
	saved := BindDamageSource("choice")
	valid := Sequence(ChooseDamageSourceFrom(You, []DamageSourceRef{This}, saved), DealDamageFrom(FromDamageSource(saved), You, Fixed(1)))
	if err := ValidateCardDefinition("Valid", NewInstant("Valid", "", WithAbility(Spell(valid)))); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []Effect{
		ChooseDamageSourceFrom(nil, []DamageSourceRef{This}, saved),
		ChooseDamageSourceFrom(You, nil, saved),
		ChooseDamageSourceFrom(You, []DamageSourceRef{nil}, saved),
		ChooseDamageSourceFrom(You, []DamageSourceRef{DamageSourceOf(nil)}, saved),
		ChooseDamageSourceFrom(You, []DamageSourceRef{This}, nil),
		ChooseDamageSourceFrom(You, []DamageSourceRef{This}, BindDamageSource("")),
		ChooseDamageSourceFrom(You, []DamageSourceRef{FromDamageSource(saved)}, saved),
		Sequence(DealDamageFrom(FromDamageSource(saved), You, Fixed(1)), ChooseDamageSourceFrom(You, []DamageSourceRef{This}, saved)),
		ChooseDamageSourceFrom(You, []DamageSourceRef{DamageSourceOfSpell(DeclareSpellTarget("undeclared", IsInstantCard))}, saved),
	} {
		if err := ValidateCardDefinition("Invalid", NewInstant("Invalid", "", WithAbility(Spell(effect)))); err == nil {
			t.Fatalf("accepted invalid declaration: %T", effect)
		}
	}
}

func TestDamageSourceChoiceInvalidAnswersAndRepeatedUse(t *testing.T) {
	for _, answer := range []int{-1, 2, 99} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		player := &sourceChoicePlayer{BasePlayer: a, index: answer}
		g := NewGame(player, b)
		p := g.PutOnBattlefield(NewCreature("First", "{R}", 1, 1), a.PlayerID())
		other := g.PutOnBattlefield(NewCreature("Second", "{G}", 1, 1), b.PlayerID())
		first, second, result := BindDamageSource("first"), BindDamageSource("second"), BindDamageSource("result")
		ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
		g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{first: damageSourceFromPermanent(p), second: damageSourceFromPermanent(other)}
		effect := ChooseDamageSourceFrom(You, []DamageSourceRef{FromDamageSource(first), FromDamageSource(second)}, result)
		if err := effect.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if !FromDamageSource(result).resolveDamageSource(ctx).SameObject(damageSourceFromPermanent(p)) {
			t.Fatal("invalid answer did not use legal first choice")
		}
		player.index = 1
		if err := effect.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if !FromDamageSource(result).resolveDamageSource(ctx).SameObject(damageSourceFromPermanent(other)) {
			t.Fatal("repeated choice retained former answer")
		}
		delete(g.resolution.damageSourceBindings, first)
		delete(g.resolution.damageSourceBindings, second)
		if err := effect.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if FromDamageSource(result).resolveDamageSource(ctx).Valid() || player.calls != 2 {
			t.Fatal("empty repeated choice retained former answer")
		}
	}
}

type numberedSourcePlayer struct {
	*BasePlayer
	reason string
}

func (p *numberedSourcePlayer) ChooseNumber(minimum, maximum int, reason string) int {
	p.reason = reason
	return maximum
}

func TestDamageSourceChoiceLegacyPlayer(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	player := &numberedSourcePlayer{BasePlayer: a}
	g := NewGame(player, b)
	first := g.PutOnBattlefield(NewArtifact("First", ""), a.PlayerID())
	second := g.PutOnBattlefield(NewArtifact("Second", ""), b.PlayerID())
	one, two, result := BindDamageSource("one"), BindDamageSource("two"), BindDamageSource("choice")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{one: damageSourceFromPermanent(first), two: damageSourceFromPermanent(second)}
	if err := ChooseDamageSourceFrom(You, []DamageSourceRef{FromDamageSource(one), FromDamageSource(two)}, result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if !FromDamageSource(result).resolveDamageSource(ctx).SameObject(damageSourceFromPermanent(second)) || player.reason != "Choose choice (1: First, 2: Second)" {
		t.Fatal("legacy player could not distinguish source options")
	}
}

func TestDamageSourceChoiceInputOutputOverlap(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	saved := BindDamageSource("source")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: p.ID()}
	body := Sequence(SnapshotDamageSource(This, saved), ChooseDamageSourceFrom(You, []DamageSourceRef{FromDamageSource(saved)}, saved))
	if err := ValidateCardDefinition("Overlap", NewArtifact("Overlap", "", WithAbility(Activate(Costs(), body)))); err != nil {
		t.Fatal(err)
	}
	if err := body.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if !FromDamageSource(saved).resolveDamageSource(ctx).SameObject(damageSourceFromPermanent(p)) {
		t.Fatal("choice cleared its input before saving the result")
	}
}
