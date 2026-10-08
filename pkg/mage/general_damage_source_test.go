package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestGeneralDamageSourceUnion(t *testing.T) {
	a := &sourceChoicePlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Referenced", "{U}", 1, 20)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	saved := g.CaptureDamageSource(p.ID())
	g.AddRegenerationShield(p.ID())
	if err := ApplyEffect(g, AtNext(EachEndStep, TapSubject(This)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	faceUp := NewArtifact("Command", "")
	faceUp.SetOwner(a.PlayerID())
	if _, err := g.AddCommandObject(faceUp, false); err != nil {
		t.Fatal(err)
	}
	if _, err := g.AddCommandObject(NewArtifact("Secret", "{R}"), true); err != nil {
		t.Fatal(err)
	}
	hidden := NewCreature("Unreferred hand", "{R}", 1, 1)
	a.AddToHand(hidden)
	for _, state := range []string{"live", "phased", "departed", "returned"} {
		switch state {
		case "phased":
			g.PhaseOut(p.ID())
		case "departed":
			g.PhaseIn([]uuid.UUID{p.ID()})
			g.BouncePermanentToHand(g.FindPermanent(p.ID()))
			g.zones.ClearLKI()
		case "returned":
			a.RemoveFromHand(card.ID())
			g.PutOnBattlefield(card, b.PlayerID())
			g.ChangeObjectColor(card.ID(), core.Red)
		}
		sources := g.DamageSourcesFor(a.PlayerID())
		want := 2
		if state == "returned" {
			want = 3
		}
		if len(sources) != want || a.calls != 0 || len(g.resolution.damageSourceBindings) != 0 {
			t.Fatal(state, "union, deduplication or query purity", len(sources))
		}
		old := slices.IndexFunc(sources, saved.SameObject)
		if old < 0 || !sources[old].HasColor(core.Blue) {
			t.Fatal(state, "lost exact saved source")
		}
		sources[old].card.AddType(core.TypeArtifact)
		if slices.ContainsFunc(g.DamageSourcesFor(a.PlayerID()), func(s DamageSource) bool { return s.SameObject(saved) && s.HasType(core.TypeArtifact) }) {
			t.Fatal("query exposed saved card")
		}
	}
	binding := BindDamageSource("general choice")
	effect := ChooseDamageSource(You, DamageSourceHasColor(core.Blue), binding)
	fixture := NewArtifact("Choice", "", WithAbility(Activate(Costs(), effect)))
	if err := ValidateCardDefinition(fixture.Name(), fixture); err != nil {
		t.Fatal(err)
	}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.calls != 1 || len(a.seen) != 1 || !FromDamageSource(binding).resolveDamageSource(ctx).SameObject(saved) {
		t.Fatal("general choice lost saved identity")
	}
	clone := g.Clone()
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.effects.ClearReplacementsEndOfTurn()
	if len(g.DamageSourcesFor(a.PlayerID())) != 2 || len(clone.DamageSourcesFor(a.PlayerID())) != 3 {
		t.Fatal("clone shares waiting lifetimes")
	}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.calls != 1 || FromDamageSource(binding).resolveDamageSource(ctx).Valid() {
		t.Fatal("empty choice did not clear old result")
	}
}

func TestGeneralDamageSourceDeclarations(t *testing.T) {
	for _, effect := range []ResolvingEffect{ChooseDamageSource(nil, DamageSourceFilter{}, BindDamageSource("choice")), ChooseDamageSource(You, DamageSourceFilter{}, nil), ChooseDamageSource(You, DamageSourceFilter{}, BindDamageSource(""))} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid general choice accepted")
		}
	}
}

func TestGeneralDamageSourceStackAndPrivacy(t *testing.T) {
	g, a, b, spell := spellSourceFixture(t)
	source := g.CaptureSpellDamageSource(spell.ID)
	g.PushStack(boundSpellObject(source))
	g.PushStack(boundDamageObject(source))
	if len(g.DamageSourcesFor(a.PlayerID())) != 1 {
		t.Fatal("live and referred spell not deduplicated")
	}
	copied := g.CopyStackObjectDirect(spell, a.PlayerID(), false)
	if copied == nil || len(g.DamageSourcesFor(a.PlayerID())) != 2 {
		t.Fatal("copied spell identity merged")
	}
	g.GetStack().RemoveByID(spell.ID)
	g.GetStack().RemoveByID(copied.ID)
	if got := g.DamageSourcesFor(a.PlayerID()); len(got) != 1 || !got[0].SameObject(source) {
		t.Fatal("departed spell reference lost")
	}
	card := NewCreature("Private reference", "{R}", 1, 1)
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	visit := a.handVisit(card.ID())
	hand := g.CaptureHandDamageSourceFor(a.PlayerID(), visit)
	object := &StackObject{ID: uuid.New(), IsAbility: true, capturedCards: map[*CardBinding]capturedCard{BindCard("hand"): {card.ID(), visit, core.ZoneHand}}}
	g.PushStack(object)
	for _, departed := range []bool{false, true} {
		if departed {
			a.RemoveFromHand(card.ID())
			a.AddToHand(card)
		}
		own := g.DamageSourcesFor(a.PlayerID())
		other := g.DamageSourcesFor(b.PlayerID())
		i := slices.IndexFunc(own, hand.SameObject)
		j := slices.IndexFunc(other, hand.SameObject)
		if len(own) != 2 || len(other) != 2 || i < 0 || j < 0 || own[i].Name() != card.Name() || other[j].Name() != "" || other[j].HasColor(core.Red) || other[j].card != nil {
			t.Fatal("hand inspection or exact-visit privacy", departed)
		}
	}
	g.GetStack().RemoveByID(object.ID)
	damageObject := boundDamageObject(hand)
	g.PushStack(damageObject)
	result := BindDamageSource("private source")
	for _, viewer := range []*BasePlayer{a, b} {
		ctx := &EffectContext{Game: g, Controller: viewer.PlayerID()}
		if err := ChooseDamageSource(You, DamageSourceHasType(core.TypeCreature), result).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		got := FromDamageSource(result).resolveDamageSource(ctx)
		if viewer == a && !got.SameObject(hand) {
			t.Fatal("damage binding omitted private inspection rights")
		}
		if viewer == b && got.Valid() {
			t.Fatal("damage binding exposed private facts")
		}
	}
	g.GetStack().RemoveByID(damageObject.ID)
	if len(g.DamageSourcesFor(a.PlayerID())) != 1 {
		t.Fatal("unreferred hand remained available")
	}
}

func TestGeneralDamageSourceGraveyardAbility(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Graveyard ability source", "{B}", 1, 1)
	card.SetOwner(a.PlayerID())
	a.AddToGraveyard(card)
	visit := a.graveyardVisit(card.ID())
	saved := g.CaptureGraveyardDamageSource(visit)
	object := &StackObject{ID: uuid.New(), IsAbility: true, SourceID: card.ID(), abilitySourceVisit: visit, TargetSource: card.Copy()}
	g.PushStack(object)
	for _, state := range []string{"live", "departed", "returned"} {
		if state == "departed" {
			a.RemoveFromGraveyard(card.ID())
		}
		if state == "returned" {
			card.AddType(core.TypeArtifact)
			a.AddToGraveyard(card)
		}
		got := g.DamageSourcesFor(b.PlayerID())
		if len(got) != 1 || !got[0].SameObject(saved) || got[0].HasType(core.TypeArtifact) {
			t.Fatal(state, "graveyard ability lost exact source visit")
		}
	}
	clone := g.Clone()
	g.GetStack().RemoveByID(object.ID)
	if len(g.DamageSourcesFor(a.PlayerID())) != 0 || len(clone.DamageSourcesFor(a.PlayerID())) != 1 {
		t.Fatal("graveyard ability reference lifetime or clone")
	}
}
