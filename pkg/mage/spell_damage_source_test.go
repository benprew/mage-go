package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func spellSourceFixture(t *testing.T) (*Game, *BasePlayer, *BasePlayer, *StackObject) {
	t.Helper()
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	card := NewInstant("Source", "{R}", WithAbility(Spell(DealDamageFrom(This, You, Fixed(1)))))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	return g, a, b, g.StackPeek()
}

func TestSpellDamageSourceLiveAndResolving(t *testing.T) {
	g, a, b, obj := spellSourceFixture(t)
	obj.Controller = b.PlayerID()
	if !g.ChangeObjectColor(obj.ID, core.Blue) {
		t.Fatal("spell color change failed")
	}
	source := g.CaptureSpellDamageSource(obj.ID)
	if source.StackID() != obj.ID || source.ID() != obj.Card.ID() || source.Incarnation() != uuid.Nil || source.Controller() != b.PlayerID() || source.Owner() != a.PlayerID() || !source.HasColor(core.Blue) || source.HasColor(core.Red) || !source.HasType(core.TypeInstant) {
		t.Fatal("spell information did not use its live stack object")
	}
	if g.CaptureSpellDamageSource(obj.SourceID).Valid() || g.CaptureSpellDamageSource(uuid.New()).Valid() {
		t.Fatal("card ID or unknown ID was accepted as a stack reference")
	}
	ab := &StackObject{ID: uuid.New(), Card: obj.Card, SourceID: obj.SourceID, IsAbility: true}
	g.GetStack().Push(ab)
	if g.CaptureSpellDamageSource(ab.ID).Valid() {
		t.Fatal("ability was accepted as a spell")
	}
	g.GetStack().Pop()
	g.GetStack().Pop()
	end := g.resolution.Begin(obj)
	defer end()
	clearColor := g.resolution.SetColorOverride(obj.SourceID, obj.ColorOverride)
	defer clearColor()
	resolved := g.CaptureDamageSource(obj.SourceID)
	if !source.SameObject(resolved) || !resolved.HasColor(core.Blue) || resolved.Controller() != b.PlayerID() {
		t.Fatal("resolution lost stack identity or current spell information")
	}
	clone := g.Clone()
	if !source.SameObject(clone.CaptureDamageSource(obj.SourceID)) {
		t.Fatal("clone lost the resolving stack identity")
	}
	end()
	if g.CaptureDamageSource(obj.SourceID).Valid() || g.CaptureSpellDamageSource(obj.ID).Valid() {
		t.Fatal("cleared or departed spell was treated as live")
	}
}

func TestSpellDamageSourceSnapshotLifecycle(t *testing.T) {
	g, a, b, obj := spellSourceFixture(t)
	spell := DeclareSpellTarget("spell")
	saved := BindDamageSource("source")
	ctx := &EffectContext{Game: g, Controller: b.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{spell.targetDeclaration: {{id: obj.ID}}}}
	effect := SnapshotDamageSource(DamageSourceOfSpell(spell), saved)
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	before := FromDamageSource(saved).resolveDamageSource(ctx)
	if !before.SameObject(g.CaptureSpellDamageSource(obj.ID)) {
		t.Fatal("typed spell target did not supply the captured identity")
	}
	clone := g.Clone()
	spellBinding := BindSpell("saved spell")
	g.resolution.spellBindings = map[*SpellBinding]uuid.UUID{spellBinding: obj.ID}
	if !before.SameObject(DamageSourceOfSpell(FromSpell(spellBinding)).resolveDamageSource(ctx)) {
		t.Fatal("saved spell binding lost its stack identity")
	}
	g.ChangeObjectColor(obj.ID, core.Green)
	obj.Controller = b.PlayerID()
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	after := FromDamageSource(saved).resolveDamageSource(ctx)
	if !before.SameObject(after) || !after.HasColor(core.Green) || before.HasColor(core.Green) || after.Controller() != b.PlayerID() {
		t.Fatal("snapshot did not freeze characteristics independently of identity")
	}
	cloneCtx := *ctx
	cloneCtx.Game = clone
	if got := FromDamageSource(saved).resolveDamageSource(&cloneCtx); !got.HasColor(core.Red) || got.Controller() != a.PlayerID() {
		t.Fatal("snapshot mutation changed the clone")
	}
	g.GetStack().RemoveByID(obj.ID)
	if DamageSourceOfSpell(spell).resolveDamageSource(ctx).Valid() {
		t.Fatal("live spell reference read a departed object")
	}
	if !FromDamageSource(saved).resolveDamageSource(ctx).SameObject(after) {
		t.Fatal("departure removed saved source information")
	}
	a.AddToHand(obj.Card)
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), obj.Card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	newSource := g.CaptureSpellDamageSource(g.StackPeek().ID)
	if newSource.ID() != before.ID() || newSource.StackID() == before.StackID() || newSource.SameObject(before) {
		t.Fatal("a later cast reused the old source identity")
	}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(saved).resolveDamageSource(ctx).Valid() {
		t.Fatal("failed capture retained the previous binding value")
	}
}

func TestSpellDamageSourceReplacementResults(t *testing.T) {
	for _, redirect := range []bool{false, true} {
		t.Run(map[bool]string{false: "player", true: "redirected"}[redirect], func(t *testing.T) {
			g, _, b, obj := spellSourceFixture(t)
			source := g.CaptureSpellDamageSource(obj.ID)
			target := g.PutOnBattlefield(NewCreature("Recipient", "", 1, 5), b.PlayerID())
			if redirect {
				g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: target.ID()})
				g.AddPreventionShield(target.ID(), 1)
			} else {
				g.AddPreventionShield(b.PlayerID(), 1)
			}
			g.GetStack().Pop()
			end := g.resolution.Begin(obj)
			defer end()
			saved := BindDamageSource("actual source")
			amount := BindNumber("actual amount")
			effect := DealDamageFrom(This, You, Fixed(3), DamageSourceInto(saved), DamageAmountInto(amount))
			if err := effect.Apply(&EffectContext{Game: g, SourceID: obj.SourceID, Controller: b.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			if !source.SameObject(g.resolution.damageSourceBindings[saved]) || g.resolution.numberBindings[amount] != 2 {
				t.Fatal("replacement or damage result lost spell identity or actual amount")
			}
			if redirect && (target.Damage != 2 || b.Life() != 20) || !redirect && (b.Life() != 18 || target.Damage != 0) {
				t.Fatal("replacement damage outcome changed")
			}
		})
	}
}

func TestSpellDamageSourceCopiesAndPermanentBoundary(t *testing.T) {
	g, a, b, obj := spellSourceFixture(t)
	original := g.CaptureSpellDamageSource(obj.ID)
	copyObj := g.CopyStackObjectDirect(obj, b.PlayerID(), false)
	if copyObj == nil {
		t.Fatal("copy failed")
	}
	copied := g.CaptureSpellDamageSource(copyObj.ID)
	if copied.StackID() == original.StackID() || copied.SameObject(original) || copied.Controller() != b.PlayerID() || !copied.HasColor(core.Red) {
		t.Fatal("copy reused original identity or controller")
	}
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 19 {
		t.Fatal("copy or original did not resolve")
	}
	card := NewCreature("Permanent spell", "{R}", 2, 2)
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	spellSource := g.CaptureSpellDamageSource(g.StackPeek().ID)
	if !spellSource.HasType(core.TypeCreature) {
		t.Fatal("permanent spell was not accepted")
	}
	g.ResolveStack()
	permanentSource := g.CaptureDamageSource(card.ID())
	if spellSource.SameObject(permanentSource) || permanentSource.StackID() != uuid.Nil || !permanentSource.SameObject(permanentSource) {
		t.Fatal("stack identity crossed the permanent incarnation boundary")
	}
	g.PhaseIn(g.PhaseOut(card.ID()))
	if !permanentSource.SameObject(g.CaptureDamageSource(card.ID())) {
		t.Fatal("phasing changed permanent identity")
	}
	g.BouncePermanentToHand(g.FindPermanent(card.ID()))
	a.RemoveFromHand(card.ID())
	g.PutOnBattlefield(card, a.PlayerID())
	if permanentSource.SameObject(g.CaptureDamageSource(card.ID())) || (DamageSource{}).SameObject(DamageSource{}) {
		t.Fatal("returned or missing source reused an identity")
	}
}

func TestSpellDamageSourceDefinitionContract(t *testing.T) {
	spell := DeclareSpellTarget("spell")
	saved := BindDamageSource("source")
	valid := NewInstant("Valid", "", WithAbility(Spell(Sequence(SnapshotDamageSource(DamageSourceOfSpell(spell), saved), DealDamageFrom(FromDamageSource(saved), You, Fixed(1))), WithTargets(spell))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	self := NewInstant("Self", "", WithAbility(Spell(Sequence(SnapshotDamageSource(This, saved), DealDamageFrom(FromDamageSource(saved), You, Fixed(1))))))
	if err := ValidateCardDefinition(self.Name(), self); err != nil {
		t.Fatal(err)
	}
	for _, body := range []ResolvingEffect{
		SnapshotDamageSource(nil, saved),
		SnapshotDamageSource(DamageSourceOfSpell(nil), saved),
		SnapshotDamageSource(This, nil),
		SnapshotDamageSource(This, BindDamageSource("")),
		SnapshotDamageSource(DamageSourceOfSpell(spell), saved),
		DealDamageFrom(FromDamageSource(saved), You, Fixed(1)),
		Sequence(DealDamageFrom(FromDamageSource(saved), You, Fixed(1)), SnapshotDamageSource(This, saved)),
		DealDamageFrom(DamageSourceOfSpell(nil), You, Fixed(1)),
		DealDamageFrom(DamageSourceOfSpell(spell), You, Fixed(1)),
	} {
		card := NewInstant("Invalid", "", WithAbility(Spell(body)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatalf("invalid source declaration accepted: %s", body.Text())
		}
	}
	ability := DeclareActivatedAbilityTarget("ability", core.TypeArtifact)
	wrongKind := NewInstant("Wrong kind", "", WithAbility(Spell(SnapshotDamageSource(DamageSourceOfSpell(ability), saved), WithTargets(ability))))
	if ValidateCardDefinition(wrongKind.Name(), wrongKind) == nil {
		t.Fatal("ability reference was accepted as a spell source")
	}
}
