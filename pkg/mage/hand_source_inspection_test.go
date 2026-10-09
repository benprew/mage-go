package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func inspectableHandFixture() (*Game, *sourceChoicePlayer, *BasePlayer, *BaseCard, uuid.UUID) {
	a := &sourceChoicePlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Private source", "{R}", 1, 1)
	a.AddToHand(card)
	visit := a.handVisit(card.ID())
	g.PushStack(&StackObject{ID: uuid.New(), IsAbility: true, capturedCards: map[*CardBinding]capturedCard{BindCard("hand reference"): {id: card.ID(), visit: visit, zone: core.ZoneHand}}})
	return g, a, b, card, visit
}

func TestHandSourceInspectionInformation(t *testing.T) {
	g, a, b, card, visit := inspectableHandFixture()
	checkHidden := func(source DamageSource) {
		t.Helper()
		if source.Name() != "" || source.card != nil || source.HasColor(core.Red) || source.Controller() != uuid.Nil {
			t.Fatal("private characteristics escaped")
		}
	}
	saved := g.CaptureHandDamageSourceFor(a.PlayerID(), visit)
	if !saved.HasColor(core.Red) || !saved.HasType(core.TypeCreature) || saved.Name() != card.Name() || saved.Owner() != a.PlayerID() {
		t.Fatal("hand player could not inspect their source")
	}
	for _, viewer := range []uuid.UUID{b.PlayerID(), uuid.Nil, uuid.New()} {
		checkHidden(g.CaptureHandDamageSourceFor(viewer, visit))
		checkHidden(g.StackHandSourcesFor(viewer)[0])
	}
	checkHidden(g.StackHandSources()[0])
	if g.CaptureHandDamageSourceFor(a.PlayerID(), card.ID()).Valid() || g.CaptureHandDamageSourceFor(a.PlayerID(), uuid.Nil).Valid() {
		t.Fatal("invalid visit supplied information")
	}
	sources := g.StackHandSourcesFor(a.PlayerID())
	sources[0].card.AddType(core.TypeLand)
	if g.StackHandSourcesFor(a.PlayerID())[0].HasType(core.TypeLand) {
		t.Fatal("query changed source facts")
	}
	clone := g.Clone()
	card.AddType(core.TypeArtifact)
	if clone.StackHandSourcesFor(a.PlayerID())[0].HasType(core.TypeArtifact) {
		t.Fatal("live clone shared source facts")
	}
	a.RemoveFromHand(card.ID())
	card.SetColorOverride([]core.Color{core.Blue})
	a.AddToHand(card)
	if g.CaptureHandDamageSourceFor(a.PlayerID(), visit).Valid() {
		t.Fatal("departed source remained live")
	}
	departed := g.StackHandSourcesFor(a.PlayerID())[0]
	if !departed.SameObject(saved) || !departed.HasColor(core.Red) || !departed.HasType(core.TypeArtifact) {
		t.Fatal("departure information followed the returned card")
	}
	checkHidden(g.StackHandSourcesFor(b.PlayerID())[0])
	checkHidden(g.damageReferenceInformation(saved))
	departed.card.AddType(core.TypeLand)
	if g.StackHandSourcesFor(a.PlayerID())[0].HasType(core.TypeLand) {
		t.Fatal("departure query changed stored facts")
	}
	after := g.Clone()
	g.GetStack().Pop()
	if len(g.StackHandSourcesFor(a.PlayerID())) != 0 || len(after.StackHandSourcesFor(a.PlayerID())) != 1 {
		t.Fatal("clone or final reference lifetime failed")
	}
	if !after.StackHandSourcesFor(a.PlayerID())[0].HasColor(core.Red) {
		t.Fatal("wrapped player clone lost private departure facts")
	}
}

func TestHandSourceInspectionChoiceAndRechecks(t *testing.T) {
	for _, state := range []string{"live", "departed", "returned", "clone"} {
		t.Run(state, func(t *testing.T) {
			g, a, b, card, visit := inspectableHandFixture()
			saved := g.CaptureHandDamageSource(visit)
			chosen := BindDamageSource("inspectable source")
			body := Sequence(ChooseDamageSourceFromInspectableStackHandReferences(You, DamageSourceHasColor(core.Red), chosen), PreventNextDamageFromSelectedSource(FromDamageSource(chosen), You, DamageSourceHasColor(core.Red)))
			if err := ValidateCardDefinition("Inspection", NewArtifact("Inspection", "", WithAbility(Activate(Costs(), body)))); err != nil {
				t.Fatal(err)
			}
			if err := ApplyEffect(g, body, uuid.Nil, b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.DealDamageToPlayerFromSource(b, 1, saved)
			if b.Life() != 19 {
				t.Fatal("opponent selected hidden color")
			}
			if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if a.calls != 1 || !a.seen[0].HasColor(core.Red) {
				t.Fatal("owner did not receive private choice information")
			}
			a.seen[0].card.AddType(core.TypeLand)
			if g.resolution.damageSourceBindings[chosen].HasType(core.TypeLand) {
				t.Fatal("chooser mutated saved binding")
			}
			if state == "live" {
				card.SetColorOverride([]core.Color{core.Blue})
				g.DealDamageToPlayerFromSource(a, 1, saved)
				if a.Life() != 19 {
					t.Fatal("recheck used stale private color")
				}
				card.SetColorOverride([]core.Color{core.Red})
			} else {
				a.RemoveFromHand(card.ID())
				card.SetColorOverride([]core.Color{core.Blue})
				if state == "returned" {
					a.AddToHand(card)
					returned := g.CaptureHandDamageSource(a.handVisit(card.ID()))
					g.DealDamageToPlayerFromSource(a, 1, returned)
					if a.Life() != 19 {
						t.Fatal("return inherited private shield")
					}
				}
			}
			if state == "clone" {
				clone := g.Clone()
				clone.DealDamageToPlayerFromSource(clone.GetPlayer(a.PlayerID()), 2, saved)
				clone.DealDamageToPlayerFromSource(clone.GetPlayer(a.PlayerID()), 1, saved)
				if clone.GetPlayer(a.PlayerID()).Life() != 19 {
					t.Fatal("clone private shield or consumption failed")
				}
			}
			before := a.Life()
			outcomes := g.dealSimultaneousDamageOutcomes(PlayerDamageAssignment(a, 2, saved), PlayerDamageAssignment(a, 3, saved))
			if a.Life() != before || len(outcomes) != 0 {
				t.Fatal("private recheck lost simultaneous prevention")
			}
			g.DealDamageToPlayerFromSource(a, 1, saved)
			if a.Life() != before-1 {
				t.Fatal("private shield was not consumed")
			}
			g.GetStack().Pop()
			if err := ApplyEffect(g, Sequence(body, DealDamageFrom(FromDamageSource(chosen), You, Fixed(1))), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if a.Life() != before-1 {
				t.Fatal("empty category retained private result")
			}
		})
	}
}

func TestHandSourceInspectionVisibilityAndInvalid(t *testing.T) {
	g, a, b, card, visit := inspectableHandFixture()
	public := g.PutOnBattlefield(NewEnchantment("Public hand", "", WithAbility(Static(RevealAllHands()))), a.PlayerID())
	if !g.StackHandSourcesFor(b.PlayerID())[0].HasColor(core.Red) {
		t.Fatal("public hand could not be inspected")
	}
	g.PhaseOut(public.ID())
	g.ApplyContinuousEffects()
	if g.StackHandSourcesFor(b.PlayerID())[0].HasColor(core.Red) || !g.StackHandSourcesFor(a.PlayerID())[0].HasColor(core.Red) {
		t.Fatal("phasing changed private inspection permission")
	}
	a.RemoveFromHand(card.ID())
	g.PhaseIn([]uuid.UUID{public.ID()})
	g.ApplyContinuousEffects()
	if g.StackHandSourcesFor(b.PlayerID())[0].HasColor(core.Red) || !g.StackHandSourcesFor(a.PlayerID())[0].HasColor(core.Red) {
		t.Fatal("later visibility changed departure permission")
	}
	if g.CaptureHandDamageSourceFor(a.PlayerID(), visit).Valid() {
		t.Fatal("departure was live")
	}
	for _, effect := range []ResolvingEffect{
		ChooseDamageSourceFromInspectableStackHandReferences(nil, DamageSourceFilter{}, BindDamageSource("choice")),
		ChooseDamageSourceFromInspectableStackHandReferences(You, DamageSourceFilter{}, nil),
		ChooseDamageSourceFromInspectableStackHandReferences(You, DamageSourceHasColor(core.Colorless), BindDamageSource("choice")),
	} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))) == nil {
			t.Fatal("invalid inspection declaration accepted")
		}
	}
	binding := BindDamageSource("later")
	if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), Sequence(DealDamageFrom(FromDamageSource(binding), You, Fixed(1)), ChooseDamageSourceFromInspectableStackHandReferences(You, DamageSourceFilter{}, binding)))))) == nil {
		t.Fatal("premature private result accepted")
	}
}

func TestHandSourceInspectionPlayerAndRedirect(t *testing.T) {
	g, a, b, _, visit := inspectableHandFixture()
	saved := g.CaptureHandDamageSource(visit)
	chosen := BindDamageSource("opponent's own hand")
	player := BindPlayer("hand player")
	g.resolution.playerBindings = map[*PlayerBinding]uuid.UUID{player: a.PlayerID()}
	if err := ChooseDamageSourceFromInspectableStackHandReferences(FromPlayer(player), DamageSourceHasColor(core.Red), chosen).Apply(&EffectContext{Game: g, Controller: b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if a.calls != 1 || !a.seen[0].HasColor(core.Red) {
		t.Fatal("inspection used the effect controller instead of the choosing player")
	}
	subject := g.PutOnBattlefield(NewCreature("Subject", "", 3, 3), a.PlayerID())
	host := BindPermanent("redirect subject")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{host: {subject.ID(), subject.incarnationID}}
	if err := RedirectNextDamageFromSelectedSource(FromDamageSource(chosen), From(host), You, DamageSourceHasColor(core.Red)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.StackBoundDamageSources() != nil {
		t.Fatal("resolution-only choice supplied a stack category")
	}
	g.GetStack().Pop()
	clone := g.Clone()
	g.DealDamageToPermanentFromSource(subject, 2, saved)
	if a.Life() != 18 || g.FindPermanent(subject.ID()).Damage != 0 {
		t.Fatal("private source did not redirect damage")
	}
	g.DealDamageToPermanentFromSource(subject, 1, saved)
	if a.Life() != 18 || g.FindPermanent(subject.ID()).Damage != 1 {
		t.Fatal("private redirect was not consumed")
	}
	clone.DealDamageToPermanentFromSource(clone.FindPermanent(subject.ID()), 2, saved)
	if clone.GetPlayer(a.PlayerID()).Life() != 18 {
		t.Fatal("clone lost private redirect")
	}
}
