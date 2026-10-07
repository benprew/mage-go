package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentSetComposition(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	own := NewArtifact("Owned", "")
	own.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(own, b.PlayerID())
	other := NewArtifact("Other", "")
	other.SetOwner(b.PlayerID())
	q := g.PutOnBattlefield(other, a.PlayerID())
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	owned := PermanentsOwnedBy(You, IsArtifact)
	controlled := PermanentsControlledBy(You, IsArtifact)
	if got := IntersectPermanents(owned, controlled).resolvePermanents(ctx); len(got) != 0 {
		t.Fatal("owner and controller merged")
	}
	inputs := []PermanentSet{owned, controlled, owned}
	union := UnionPermanents(inputs...)
	inputs[0] = MatchingPermanents(IsLand)
	if got := union.resolvePermanents(ctx); len(got) != 2 {
		t.Fatal("union lost a member or repeated it", len(got))
	}
	clone := g.Clone()
	if err := ApplyEffect(clone, ReturnEachToHand(union), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if clone.FindPermanent(p.ID()) != nil || clone.FindPermanent(q.ID()) != nil || g.FindPermanent(p.ID()) == nil {
		t.Fatal("clone selection crossed games")
	}
	g.PhaseOut(p.ID())
	if got := union.resolvePermanents(ctx); len(got) != 1 || got[0].ID() != q.ID() {
		t.Fatal("phased member selected")
	}
	g.PhaseIn(g.PhaseOut(q.ID()))
	g.PhaseIn([]uuid.UUID{p.ID()})
	g.AddControlEffect(ControlEffectSpec{SourceID: q.ID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
	if got := IntersectPermanents(owned, controlled).resolvePermanents(ctx); len(got) != 1 || got[0].ID() != p.ID() {
		t.Fatal("current control not read")
	}
}

func TestPermanentSetAttachments(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host := g.PutOnBattlefield(NewCreature("Host", "", 1, 4), a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 4), b.PlayerID())
	aura := g.PutOnBattlefield(NewEnchantment("Aura", "", WithSubTypes("Aura")), b.PlayerID())
	g.Attach(aura.ID(), host.ID())
	set := AttachmentsOn(PermanentsControlledBy(You, IsCreature))
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if got := set.resolvePermanents(ctx); len(got) != 1 || got[0].ID() != aura.ID() {
		t.Fatal("attachment controller replaced host controller")
	}
	clone := g.Clone()
	clone.Attach(aura.ID(), other.ID())
	if got := set.resolvePermanents(&EffectContext{Game: clone, Controller: a.PlayerID()}); len(got) != 0 {
		t.Fatal("attachment not read live")
	}
	if got := set.resolvePermanents(ctx); len(got) != 1 {
		t.Fatal("clone changed attachment")
	}
	g.PhaseOut(host.ID())
	if got := set.resolvePermanents(ctx); len(got) != 0 {
		t.Fatal("phased host selected")
	}
	g.PhaseIn([]uuid.UUID{host.ID()})
	g.MutablePermanent(aura.ID()).attachedIncarnation = uuid.New()
	if got := set.resolvePermanents(ctx); len(got) != 0 {
		t.Fatal("stale attachment incarnation selected")
	}
}

func TestPermanentSetCompositionDeclarations(t *testing.T) {
	unbound := DeclarePlayerTarget("missing")
	for _, set := range []PermanentSet{UnionPermanents(), IntersectPermanents(), UnionPermanents(nil), IntersectPermanents(MatchingPermanents(IsCreature), nil), AttachmentsOn(nil), UnionPermanents(PermanentsOwnedBy(unbound, IsArtifact)), AttachmentsOn(PermanentsControlledBy(unbound, IsCreature))} {
		card := NewInstant("Invalid", "", WithAbility(Spell(ReturnEachToHand(set))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid collection accepted")
		}
	}
}

func TestPermanentSetPlayerControllers(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Other", "", 1, 4), b.PlayerID())
	q := g.PutOnBattlefield(NewCreature("Own", "", 1, 4), a.PlayerID())
	set := PermanentsControlledByPlayers(OpponentsOf(You), IsCreature)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	for range 3 {
		got := set.resolvePermanents(ctx)
		if len(got) != 1 || got[0].ID() != p.ID() || g.StackSize() != 0 || a.ManaPool().TotalMana() != 0 {
			t.Fatal("query changed state or selected wrong player")
		}
	}
	if got := set.resolvePermanents(&EffectContext{Game: g, Controller: b.PlayerID()}); len(got) != 1 || got[0].ID() != q.ID() {
		t.Fatal("controller context shared")
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	if got := set.resolvePermanents(ctx); len(got) != 0 {
		t.Fatal("LKI restored departed member")
	}
	b.RemoveFromHand(p.ID())
	g.PutOnBattlefield(p.Card, b.PlayerID())
	if got := set.resolvePermanents(ctx); len(got) != 1 {
		t.Fatal("live collection ignored new incarnation")
	}
	for _, invalid := range []PermanentSet{PermanentsControlledByPlayers(nil, IsCreature), PermanentsControlledByPlayers(OpponentsOf(nil), IsCreature)} {
		card := NewInstant("Invalid", "", WithAbility(Spell(ReturnEachToHand(invalid))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid player set accepted")
		}
	}
}
