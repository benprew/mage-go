package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTargetDependency(t *testing.T) {
	for _, change := range []string{"normal", "type", "return", "phase", "phase-cycle", "shroud", "second-shroud", "source-departure", "copy", "clone"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			first := DeclarePermanentTarget("first", AnyPermanent, TargetYouControl())
			second := DeclarePermanentTarget("second", AnyPermanent, TargetOpponentControls(), TargetSharesCardTypeWith(first, core.TypeArtifact, core.TypeCreature, core.TypeLand))
			card := NewArtifact("Source", "", WithAbility(Activate(Costs(ManaPayment("{1}")), Sequence(TapSubject(first), TapSubject(second), GainLifeFor(You, Fixed(1))), WithTargets(first, second))))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			refCard := NewCreature("Reference", "", 1, 5)
			refCard.SetOwner(a.PlayerID())
			ref := g.PutOnBattlefield(refCard, a.PlayerID())
			member := g.PutOnBattlefield(NewCreature("Member", "", 1, 5), b.PlayerID())
			other := g.PutOnBattlefield(NewLand("Other"), b.PlayerID())
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{ref.ID(), other.ID()}); err == nil || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
				t.Fatal("invalid dependency spent resources")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{ref.ID(), member.ID()}); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "type":
				g.MutablePermanent(ref.ID()).Card.(*BaseCard).types = []core.CardType{core.TypeArtifact}
				g.MutablePermanent(ref.ID()).RevokeBaseAttr(core.AttrIsCreature)
			case "return":
				g.BouncePermanentToHand(ref)
				a.RemoveFromHand(refCard.ID())
				g.PutOnBattlefield(refCard, a.PlayerID())
			case "phase":
				g.PhaseOut(ref.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(ref.ID()))
			case "shroud":
				g.MutablePermanent(ref.ID()).GrantBaseAttr(core.Shroud)
			case "second-shroud":
				g.MutablePermanent(member.ID()).GrantBaseAttr(core.Shroud)
			case "source-departure":
				g.BouncePermanentToHand(source)
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			if change == "clone" {
				clone.PhaseOut(member.ID())
			}
			g.ResolveStack()
			if clone.FindPermanentIncludingPhased(member.ID()).Tapped {
				t.Fatal("clone shared effects")
			}
			clone.ResolveStack()
			for _, game := range []*Game{g, clone} {
				want := change != "type" && change != "return" && change != "phase" && change != "second-shroud" && (change != "clone" || game != clone)
				if game.FindPermanentIncludingPhased(member.ID()).Tapped != want {
					t.Fatal("wrong dependent legality")
				}
			}
		})
	}
}

func TestTargetDependencyDeclarations(t *testing.T) {
	for _, scenario := range []string{"nil", "unbound", "self", "forward", "duplicate", "invalid", "plural"} {
		t.Run(scenario, func(t *testing.T) {
			first := DeclarePermanentTarget("first", AnyPermanent)
			ref := first
			types := []core.CardType{core.TypeCreature}
			if scenario == "nil" {
				ref = nil
			}
			if scenario == "unbound" {
				ref = DeclarePermanentTarget("other", AnyPermanent)
			}
			if scenario == "duplicate" {
				types = append(types, core.TypeCreature)
			}
			if scenario == "invalid" {
				types = []core.CardType{core.CardType(255)}
			}
			second := DeclarePermanentTarget("second", AnyPermanent, TargetSharesCardTypeWith(ref, types...))
			var targets []TargetBinding
			switch scenario {
			case "self":
				first.spec.(*PermanentTarget).typeDependencies = []targetTypeDependency{{first, nil}}
				targets = []TargetBinding{first}
			case "forward":
				targets = []TargetBinding{second, first}
			case "plural":
				targets = []TargetBinding{first, DeclarePermanentTargets("second", 1, 2, AnyPermanent, TargetSharesCardTypeWith(first))}
			default:
				targets = []TargetBinding{first, second}
			}
			card := NewInstant("Invalid", "", WithAbility(Spell(TapSubject(first), WithTargets(targets...))))
			if err := ValidateCardDefinition(card.Name(), card); err == nil {
				t.Fatal("invalid dependency accepted")
			}
		})
	}
}

func TestTargetDependencyQuery(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first := DeclarePermanentTarget("first", Or(IsCreature, IsLand), TargetYouControl())
	kinds := []core.CardType{core.TypeCreature, core.TypeLand}
	second := DeclarePermanentTarget("second", AnyPermanent, TargetOpponentControls(), TargetSharesCardTypeWith(first, kinds...))
	kinds[0] = core.TypeEnchantment
	card := NewArtifact("Source", "", WithAbility(Activate(Costs(), TapSubject(second), WithTargets(first, second))))
	source := g.PutOnBattlefield(card, a.PlayerID())
	noPartner := g.PutOnBattlefield(NewLand("No partner"), a.PlayerID())
	ref := g.PutOnBattlefield(NewCreature("Reference", "", 1, 5), a.PlayerID())
	member := g.PutOnBattlefield(NewCreature("Member", "", 1, 5), b.PlayerID())
	specs := []Target{first.targetDeclaration, second.targetDeclaration}
	g.resolution.SetX(7)
	for range 3 {
		if !slices.Equal(g.PossibleTargetGroup(a.PlayerID(), card, specs, 0, nil, 0), []uuid.UUID{ref.ID()}) {
			t.Fatal("first candidates include an impossible later group")
		}
		got := g.PossibleTargetGroup(a.PlayerID(), card, specs, 1, []uuid.UUID{ref.ID()}, 0)
		if !slices.Equal(got, []uuid.UUID{member.ID()}) {
			t.Fatalf("wrong comparison: %v", got)
		}
		if len(g.PossibleTargetGroup(a.PlayerID(), card, specs, 1, []uuid.UUID{member.ID()}, 0)) != 0 {
			t.Fatal("invalid prefix accepted")
		}
		if !card.Abilities()[0].(ActivatedAbility).CanActivate(a.PlayerID(), g) {
			t.Fatal("query did not find a later compatible first choice")
		}
		if len(a.chooseQueue) != 0 || g.resolution.X() != 7 || g.resolution.targetQueryBindings != nil || g.StackSize() != 0 || member.Tapped {
			t.Fatal("query changed state")
		}
	}
	g.PhaseOut(ref.ID())
	if card.Abilities()[0].(ActivatedAbility).CanActivate(a.PlayerID(), g) {
		t.Fatal("incompatible pair was available")
	}
	g.PhaseIn([]uuid.UUID{ref.ID()})
	g.PhaseOut(noPartner.ID())
	forced := ForcedActivationTargets(a.PlayerID(), card, specs, g)
	if !slices.Equal(forced, []uuid.UUID{ref.ID(), member.ID()}) {
		t.Fatalf("wrong forced pair: %v", forced)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if !member.Tapped {
		t.Fatal("forced dependency failed")
	}
}

func TestTargetDependencyCopyAndPrompt(t *testing.T) {
	for _, activation := range []bool{false, true} {
		g, a, b := newSpellCopyGame()
		first := DeclarePermanentTarget("first", Or(IsCreature, IsLand), TargetYouControl())
		second := DeclarePermanentTarget("second", AnyPermanent, TargetOpponentControls(), TargetSharesCardTypeWith(first))
		body := Sequence(TapSubject(first), TapSubject(second))
		var card *BaseCard
		if activation {
			card = NewArtifact("Source", "", WithAbility(Activate(Costs(), body, WithTargets(first, second))))
		} else {
			card = NewInstant("Spell", "", WithAbility(Spell(body, WithTargets(first, second))))
		}
		card.SetOwner(a.PlayerID())
		ref := g.PutOnBattlefield(NewCreature("Reference", "", 1, 5), a.PlayerID())
		member := g.PutOnBattlefield(NewCreature("Member", "", 1, 5), b.PlayerID())
		newRef := g.PutOnBattlefield(NewLand("New reference"), a.PlayerID())
		newMember := g.PutOnBattlefield(NewLand("New member"), b.PlayerID())
		if activation {
			source := g.PutOnBattlefield(card, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{ref.ID(), member.ID()}); err != nil {
				t.Fatal(err)
			}
		} else {
			a.AddToHand(card)
			a.chooseQueue = [][]uuid.UUID{{ref.ID()}, {member.ID()}}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), g.promptTargetsForListWithX(a.PlayerID(), SpellTargetSource(card), card.CastTargets(), 0), 0); err != nil {
				t.Fatal(err)
			}
		}
		original := g.StackPeek()
		a.chooseQueue = [][]uuid.UUID{{newRef.ID()}, {newMember.ID()}}
		copyObj := g.CopyStackObjectDirect(original, a.PlayerID(), true)
		if copyObj == nil || !slices.Equal(copyObj.Targets, []uuid.UUID{newRef.ID(), newMember.ID()}) || !slices.Equal(original.Targets, []uuid.UUID{ref.ID(), member.ID()}) {
			t.Fatal("copy query lost new first choice")
		}
		g.ResolveStackObject(copyObj)
		if !newRef.Tapped || !newMember.Tapped || ref.Tapped || member.Tapped {
			t.Fatal("copy did not keep its pair")
		}
		g.ResolveStackObject(original)
		if !ref.Tapped || !member.Tapped {
			t.Fatal("original lost its pair")
		}
	}
}

func TestTargetDependencyTriggers(t *testing.T) {
	for _, modal := range []bool{false, true} {
		g, a, b := newSpellCopyGame()
		first := DeclarePermanentTarget("first", IsLand, TargetYouControl())
		second := DeclarePermanentTarget("second", AnyPermanent, TargetOpponentControls(), TargetSharesCardTypeWith(first))
		body := TapSubject(second)
		var trigger TriggeredAbility
		if modal {
			trigger = WhenEventModes(Enters(This), ModeOf("pair", body, WithTargets(first, second)), ModeOf("life", GainLifeFor(You, Fixed(1))))
		} else {
			trigger = WhenEvent(Enters(This), body, WithTargets(first, second))
		}
		ref := g.PutOnBattlefield(NewLand("Reference"), a.PlayerID())
		member := g.PutOnBattlefield(NewLand("Member"), b.PlayerID())
		a.chooseQueue = [][]uuid.UUID{{ref.ID()}, {member.ID()}}
		source := NewCreature("Source", "", 1, 5, WithAbility(trigger))
		source.SetOwner(a.PlayerID())
		if err := ValidateCardDefinition(source.Name(), source); err != nil {
			t.Fatal(err)
		}
		g.PutOnBattlefield(source, a.PlayerID())
		g.PutTriggersOnStack()
		g.ResolveStack()
		if !member.Tapped {
			t.Fatal("trigger query lost earlier target")
		}
	}
}

func TestTargetDependencyCopyReturnedChoice(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first := DeclarePermanentTarget("first", AnyPermanent, TargetYouControl())
	second := DeclarePermanentTarget("second", AnyPermanent, TargetOpponentControls(), TargetSharesCardTypeWith(first))
	spell := NewInstant("Spell", "", WithAbility(Spell(Sequence(TapSubject(first), TapSubject(second)), WithTargets(first, second))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	refCard := NewCreature("Reference", "", 1, 5)
	refCard.SetOwner(a.PlayerID())
	ref := g.PutOnBattlefield(refCard, a.PlayerID())
	member := g.PutOnBattlefield(NewCreature("Member", "", 1, 5), b.PlayerID())
	newMember := g.PutOnBattlefield(NewLand("New member"), b.PlayerID())
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{ref.ID(), member.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	g.BouncePermanentToHand(ref)
	a.RemoveFromHand(refCard.ID())
	refCard.types = []core.CardType{core.TypeLand}
	returned := g.PutOnBattlefield(refCard, a.PlayerID())
	a.chooseQueue = [][]uuid.UUID{{returned.ID()}, {newMember.ID()}}
	copyObj := g.CopyStackObjectDirect(original, a.PlayerID(), true)
	if copyObj == nil || !slices.Equal(copyObj.Targets, []uuid.UUID{returned.ID(), newMember.ID()}) {
		t.Fatal("new first visit did not supply copied second candidates")
	}
	g.ResolveStackObject(copyObj)
	if !returned.Tapped || !newMember.Tapped || member.Tapped {
		t.Fatal("copy did not capture the newly chosen visits")
	}
	g.ResolveStackObject(original)
	if member.Tapped {
		t.Fatal("original followed the returned first visit")
	}
}
