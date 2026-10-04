package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTemporaryAnimationFactory(t *testing.T) {
	for _, change := range []string{"none", "return", "phase", "phase-in", "control"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "ATQ", "Mishra's Factory")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			if !g.CanAfford(a.PlayerID(), core.ParseManaCost("{1}"), nil) {
				t.Fatal("mana solver omitted Factory")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if !g.GetStack().IsEmpty() || a.ManaPool().TotalMana() != 1 {
				t.Fatal("mana ability did not resolve immediately")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(p.ID()).HasType(core.TypeCreature) {
				t.Fatal("animation started at announcement")
			}
			clone := g.Clone()
			id := p.ID()
			switch change {
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(id))
				a.RemoveFromHand(card.ID())
				id = g.PutOnBattlefield(card, a.PlayerID()).ID()
			case "phase":
				g.PhaseOut(id)
			case "phase-in":
				g.PhaseIn(g.PhaseOut(id))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: id, TargetID: id, ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.ResolveStack()
			g.PhaseIn([]uuid.UUID{id})
			animated := change != "return" && change != "phase"
			p = g.FindPermanent(id)
			if !p.HasType(core.TypeLand) || p.HasType(core.TypeCreature) != animated || p.HasType(core.TypeArtifact) != animated || p.HasSubType("Assembly-Worker") != animated {
				t.Fatal("wrong animation identity or types")
			}
			if animated && (p.CurrentPower(g) != 2 || p.CurrentToughness(g) != 2) {
				t.Fatal("wrong animation base values")
			}
			clone.ResolveStack()
			if !clone.FindPermanent(card.ID()).HasType(core.TypeCreature) {
				t.Fatal("clone lost pending animation")
			}
			clone.MutablePermanent(card.ID()).Tapped = false
			if err := clone.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err == nil {
				t.Fatal("newly animated creature ignored summoning sickness")
			}
			clone.MutablePermanent(card.ID()).RevokeBaseAttr(core.AttrSummonSick)
			if err := clone.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err != nil {
				t.Fatal("animation removed the mana ability", err)
			}
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			if g.FindPermanent(id).HasType(core.TypeCreature) || g.FindPermanent(id).HasType(core.TypeArtifact) || g.FindPermanent(id).HasSubType("Assembly-Worker") || !clone.FindPermanent(card.ID()).HasType(core.TypeCreature) {
				t.Fatal("wrong animation cleanup or clone isolation")
			}
		})
	}
}

func TestRONTemporaryAnimationFactoryBoostAndRepeat(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "ATQ", "Mishra's Factory")
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	otherCard := repositoryCard(t, "ATQ", "Mishra's Factory")
	otherCard.SetOwner(b.PlayerID())
	other := g.PutOnBattlefield(otherCard, b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, []uuid.UUID{other.ID()}); err == nil {
		t.Fatal("noncreature target accepted")
	}
	if g.FindPermanent(p.ID()).Tapped || !g.GetStack().IsEmpty() {
		t.Fatal("invalid target paid tap cost")
	}
	b.ManaPool().Add(core.Colorless, 2)
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.AddCardType(mage.This, core.TypeCreature), mage.AddSubtypeFamily(mage.This, mage.SubtypeCreature, "Human"), mage.SetBasePowerToughness(mage.This, mage.Fixed(4), mage.Fixed(4))), other.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), other.ID(), 1, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, []uuid.UUID{other.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(other.ID()).HasSubType("Human") {
		t.Fatal("Factory retained a prior creature type")
	}
	if other = g.FindPermanent(other.ID()); other.CurrentPower(g) != 3 || other.CurrentToughness(g) != 3 {
		t.Fatal("Factory boost missed opponent Assembly-Worker")
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), other.ID(), 1, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(other.ID()).CurrentPower(g) != 3 {
		t.Fatal("reanimation erased additive bonus")
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), other.ID(), 1, nil); err == nil {
		t.Fatal("unpaid animation succeeded")
	}
	if !g.GetStack().IsEmpty() || g.FindPermanent(other.ID()).CurrentPower(g) != 3 {
		t.Fatal("failed cost changed layer state")
	}
}

func TestRONTemporaryAnimationQueen(t *testing.T) {
	for _, change := range []string{"none", "return", "phase", "phase-in", "source-leaves", "source-return", "source-control", "noncreature"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "ARN", "Sorceress Queen")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			victim := repositoryCard(t, "LEB", "Shivan Dragon")
			victim.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(victim, b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{source.ID()}); err == nil {
				t.Fatal("Queen targeted itself")
			}
			if g.FindPermanent(source.ID()).Tapped || !g.GetStack().IsEmpty() {
				t.Fatal("self target paid cost")
			}
			if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ModifyPT(mage.This, mage.Fixed(3), mage.Fixed(3))), target.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.MutablePermanent(target.ID()).AddCounter(core.P1P1, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			id := target.ID()
			switch change {
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(id))
				b.RemoveFromHand(victim.ID())
				id = g.PutOnBattlefield(victim, b.PlayerID()).ID()
			case "phase":
				g.PhaseOut(id)
			case "phase-in":
				g.PhaseIn(g.PhaseOut(id))
			case "source-leaves":
				g.DestroyPermanent(g.FindPermanent(source.ID()))
			case "source-return":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "noncreature":
				g.AddContinuousEffect(mage.FuncContinuousEffect(core.LayerType, core.EndOfTurn, func(game *mage.Game, _ uuid.UUID) error {
					game.GetEffects().RevokeAttr(id, core.AttrIsCreature)
					return nil
				}))
				g.ApplyContinuousEffects()
			}
			g.ResolveStack()
			g.PhaseIn([]uuid.UUID{id})
			p := g.FindPermanent(id)
			power, toughness := 4, 6
			if change == "phase" || change == "noncreature" {
				power, toughness = 9, 9
			}
			if change == "return" {
				power, toughness = 5, 5
			}
			if p.CurrentPower(g) != power || p.CurrentToughness(g) != toughness || !p.HasKeyword(core.Flying) {
				t.Fatalf("Queen P/T = %d/%d, want %d/%d", p.CurrentPower(g), p.CurrentToughness(g), power, toughness)
			}
			clone.ResolveStack()
			if clone.FindPermanent(target.ID()).CurrentPower(clone) != 4 {
				t.Fatal("clone lost Queen target")
			}
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			want := 6
			if change == "return" {
				want = 5
			}
			if g.FindPermanent(id).CurrentPower(g) != want || clone.FindPermanent(target.ID()).CurrentPower(clone) != 4 {
				t.Fatal("base layer cleanup was not isolated")
			}
		})
	}
}

func TestRONTemporaryAnimationCopyAndInventory(t *testing.T) {
	g, a, b := ronGame()
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 4, 4), b.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 5, 5), a.PlayerID())
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))], ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent, [SetPT(PT(0, 2))], UntilEndOfTurn)])))`)
	spell := factory()
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{first.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(first.ID())
	copied := g.CopySpellOnStack(spell.ID(), b.PlayerID(), true)
	if copied == nil || len(copied.Targets) != 1 || copied.Targets[0] != second.ID() {
		t.Fatal("copy did not retarget")
	}
	g.PhaseIn([]uuid.UUID{first.ID()})
	g.ResolveStack()
	if g.FindPermanent(first.ID()).CurrentPower(g) != 0 || g.FindPermanent(second.ID()).CurrentToughness(g) != 2 {
		t.Fatal("copy and original lost separate typed bindings")
	}
	src := `Card(Name: "Inventory", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Activated(PayMana([]), ActionList([CreatePermanentLayerEffectUntil(ThisPermanent, [SetPT(PT(2, 2)), AddCreatureType(AssemblyWorker), AddCardtype(Creature)], UntilEndOfTurn)]))])`
	nodes, err := Parse(strings.NewReader(src), "inventory.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]*InventorySignature{}
	walkInventory(nodes[0], "Inventory", "Card", "card", "collection", true, entries)
	for name, role := range map[string]string{"PT": "fixed_pt", "Creature": "card_type", "AssemblyWorker": "creature_subtype"} {
		found := false
		for _, entry := range entries {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s role %s", name, role)
		}
	}
}

func TestRONTemporaryAnimationDeclarations(t *testing.T) {
	for _, mod := range []string{`SetPT(PT(0, 2))`, `AddCardtype(Creature), AddCreatureType(AssemblyWorker), SetPT(PT(2, 2))`} {
		factory := permanentScopeFactory(t, "Instant", fmt.Sprintf(`SpellActions(Targeted([TargetPermanent(AnyPermanent)], ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent, [%s], UntilEndOfTurn)])))`, mod))
		first, second := factory(), factory()
		action := first.Abilities()[0].(*mage.ActionDefinition)
		if action.Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
			t.Fatal("target shared across factories")
		}
		if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
			t.Fatal(err)
		}
		invalid := mage.NewInstant("Invalid", "", mage.WithAbility(mage.Spell(mage.Sequence(action.Effects()...))))
		if mage.ValidateCardDefinition(invalid.Name(), invalid) == nil {
			t.Fatal("unbound temporary subject accepted")
		}
	}
	for _, mod := range []string{`SetPT(PT(Integer(1), 2))`, `SetPT(ManualPT(1, 2))`, `SetPT(PT(1))`, `AddCardtype(Instant)`, `AddCardtype(Creature())`, `AddCreatureType(Forest)`, `SetCreatureType(Forest)`, `SetCreatureType(AssemblyWorker())`, `AddCreatureType(AssemblyWorker())`, `AddCreatureType("")`} {
		src := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Activated(PayMana([]), ActionList([CreatePermanentLayerEffectUntil(ThisPermanent, [%s], UntilEndOfTurn)]))])`, mod)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted invalid modification %s", mod)
		}
	}
	for _, typ := range []string{"Instant", "Artifact"} {
		for _, filter := range []string{`Other(HostPermanent)`, `Other(Ref_TargetPermanent)`, `Other(ThisPermanent())`, `Or([AnyPermanent, Other(ThisPermanent)])`} {
			src := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [%s], Subtypes: []), Rules: [%s(Targeted([TargetPermanent(%s)], ActionList([TapPermanent(Ref_TargetPermanent)])))])`, typ, "SpellActions", filter)
			if typ == "Artifact" {
				src = strings.Replace(src, "SpellActions(", "Activated(PayMana([]), ", 1)
			}
			nodes, err := Parse(strings.NewReader(src), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatal("invalid source filter accepted")
			}
		}
	}
}
