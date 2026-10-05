package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestBasicLandDependencyOverridesTimestamps(t *testing.T) {
	for _, order := range []string{"selection-first", "change-first"} {
		for _, starts := range []string{"Forest", "Mountain"} {
			t.Run(order+"/"+starts, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				land := g.PutOnBattlefield(NewLand("Land", WithSubTypes(starts)), b.PlayerID())
				var selective, changer *Permanent
				addSelective := func() {
					selective = g.PutOnBattlefield(NewEnchantment("Select Mountains", "", WithStaticAbility(BecomesBasicLandsEffect(HasSubType("Mountain"), "Plains"))), a.PlayerID())
				}
				addChange := func() {
					changer = g.PutOnBattlefield(NewEnchantment("Set Forest", "", WithStaticAbility(BecomesBasicLandsEffect(HasSubType("Forest"), "Mountain"))), a.PlayerID())
				}
				if order == "selection-first" {
					addSelective()
					addChange()
				} else {
					addChange()
					addSelective()
				}
				for range 3 {
					g.ApplyContinuousEffects()
					p := g.FindPermanent(land.ID())
					if !p.HasSubType("Plains") || countManaAbilityColor(p, core.White) != 1 || countManaAbilityColor(p, core.Green) != 0 {
						t.Fatal("dependent selection did not wait for type change")
					}
				}
				clone := g.Clone()
				g.RemoveFromBattlefield(g.FindPermanent(changer.ID()))
				want := starts
				if starts == "Mountain" {
					want = "Plains"
				}
				if !g.FindPermanent(land.ID()).HasSubType(want) || !clone.FindPermanent(land.ID()).HasSubType("Plains") {
					t.Fatal("source lifetime or clone isolation")
				}
				g.RemoveFromBattlefield(g.FindPermanent(selective.ID()))
				if !g.FindPermanent(land.ID()).HasSubType(starts) {
					t.Fatal("printed subtype was not restored")
				}
			})
		}
	}
}

func TestBasicLandDependencyAddsMembersAndKeepsProbePure(t *testing.T) {
	g, a, source, land, printed := basicLandTypeTestGame(t)
	other := g.PutOnBattlefield(NewLand("Forest", WithSubTypes("Forest")), a.PlayerID())
	selective := BecomesBasicLandsEffect(HasSubType("Mountain"), "Plains")
	selective.SetSourceID(source.ID())
	g.AddContinuousEffect(selective)
	change := BecomesBasicLandsEffect(IsLand, "Mountain")
	change.SetSourceID(source.ID())
	detector, ok := selective.(DependencyDetector)
	if !ok {
		t.Fatal("missing basic land dependency detector")
	}
	before := g.Clone()
	for range 3 {
		if !detector.DependsOn(change, g) {
			t.Fatal("new member did not produce dependency")
		}
		if !g.FindPermanent(other.ID()).HasSubType("Forest") || !before.FindPermanent(other.ID()).HasSubType("Forest") {
			t.Fatal("dependency probe changed live game")
		}
	}
	g.AddContinuousEffect(change)
	for _, id := range []uuid.UUID{land.ID(), other.ID()} {
		p := g.FindPermanent(id)
		if !p.HasSubType("Plains") || countManaAbilityColor(p, core.White) != 1 || countManaAbilityColor(p, core.Red) != 0 {
			t.Fatal("new member missed dependent type change")
		}
	}
	if hasRuntimeAbility(g.FindPermanent(land.ID()), printed) {
		t.Fatal("rules text survived basic type change")
	}
}

func TestBasicLandDependencyLoopUsesTimestamps(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		g, a, _, land, _ := basicLandTypeTestGame(t)
		g.PutOnBattlefield(NewLand("Forest", WithSubTypes("Forest")), a.PlayerID())
		first, second := BecomesBasicLandsEffect(HasSubType("Mountain"), "Forest"), BecomesBasicLandsEffect(HasSubType("Forest"), "Mountain")
		if reverse {
			first, second = second, first
		}
		g.PutOnBattlefield(NewEnchantment("First", "", WithStaticAbility(first)), a.PlayerID())
		g.PutOnBattlefield(NewEnchantment("Second", "", WithStaticAbility(second)), a.PlayerID())
		want := "Mountain"
		if reverse {
			want = "Forest"
		}
		for range 3 {
			g.ApplyContinuousEffects()
			if !g.FindPermanent(land.ID()).HasSubType(want) {
				t.Fatal("dependency loop did not use timestamps")
			}
		}
	}
}

func TestBasicLandDependencyRechecksRemainingEffects(t *testing.T) {
	g, a, _, land, _ := basicLandTypeTestGame(t)
	forest := g.PutOnBattlefield(NewLand("Other", WithSubTypes("Mountain")), a.PlayerID())
	// The first change alters the dependency between the remaining effects.
	for _, effect := range []ContinuousEffect{
		BecomesBasicLandsEffect(HasSubType("Mountain"), "Forest"),
		BecomesBasicLandsEffect(HasSubType("Mountain"), "Forest"),
		BecomesBasicLandsEffect(HasSubType("Forest"), "Mountain"),
	} {
		g.PutOnBattlefield(NewEnchantment("Change", "", WithStaticAbility(effect)), a.PlayerID())
	}
	for _, id := range []uuid.UUID{land.ID(), forest.ID()} {
		if !g.FindPermanent(id).HasSubType("Forest") {
			t.Fatal("remaining dependency order was not reevaluated")
		}
	}
}

func TestBasicLandAttachedKeepsHostIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewLand("Host", WithSubTypes("Forest"))
	host := g.PutOnBattlefield(card, b.PlayerID())
	aura := g.PutOnBattlefield(NewAura("Type Aura", "", WithCastTarget(TargetPermanent(IsLand)), WithStaticAbility(BecomesBasicLandAttachedEffect("Swamp"))), a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	if !g.FindPermanent(host.ID()).HasSubType("Swamp") {
		t.Fatal("attached type change missing")
	}
	clone := g.Clone()
	g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
	host = g.PutOnBattlefield(card, b.PlayerID())
	if !g.FindPermanent(host.ID()).HasSubType("Forest") {
		t.Fatal("old attachment changed returned host")
	}
	if !clone.FindPermanent(host.ID()).HasSubType("Swamp") {
		t.Fatal("host return changed clone")
	}
	g.Attach(aura.ID(), host.ID())
	ids := g.PhaseOut(host.ID())
	g.ApplyContinuousEffects()
	g.PhaseIn(ids)
	g.ApplyContinuousEffects()
	if !g.FindPermanent(host.ID()).HasSubType("Swamp") {
		t.Fatal("phase cycle lost attachment")
	}
	ids = g.PhaseOut(aura.ID())
	g.ApplyContinuousEffects()
	if !g.FindPermanent(host.ID()).HasSubType("Forest") {
		t.Fatal("direct source phasing retained type change")
	}
	g.PhaseIn(ids)
	g.ApplyContinuousEffects()
	if !g.FindPermanent(host.ID()).HasSubType("Swamp") {
		t.Fatal("source phase-in lost type change")
	}
	g.AddContinuousEffect(RemoveAllAbilities(aura.ID(), core.Indefinite))
	for range 2 {
		g.ApplyContinuousEffects()
		if !g.FindPermanent(host.ID()).HasSubType("Swamp") {
			t.Fatal("layer 6 removal erased layer 4 change")
		}
	}
}

func TestBasicLandDependencyTracksPrintedEffectExistence(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := g.PutOnBattlefield(NewLand("Forest", WithSubTypes("Forest")), b.PlayerID())
	source := g.PutOnBattlefield(NewLand("Static Land", WithStaticAbility(BecomesBasicLandsEffect(HasSubType("Forest"), "Plains"))), a.PlayerID())
	g.PutOnBattlefield(NewEnchantment("Remove Printed Rule", "", WithStaticAbility(BecomesBasicLandsEffect(Named("Static Land"), "Swamp"))), a.PlayerID())
	for range 3 {
		g.ApplyContinuousEffects()
		if !g.FindPermanent(source.ID()).HasSubType("Swamp") || !g.FindPermanent(target.ID()).HasSubType("Forest") {
			t.Fatal("land type change did not suppress printed effect before its application")
		}
	}
}

func TestBasicLandDependencyLifetimeAndMana(t *testing.T) {
	g, a, source, land, printed := basicLandTypeTestGame(t)
	grant := FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
		ability := NewManaAbility(core.Blue)
		ability.SetSource(land.ID())
		ability.SetController(a.PlayerID())
		p := g.MutablePermanent(land.ID())
		if p != nil {
			p.RuntimeAbilities = append(p.RuntimeAbilities, WrapGrantedAbility(ability))
		}
		return nil
	})
	grant.SetSourceID(source.ID())
	g.AddContinuousEffect(grant)
	change := BecomesBasicLandsEffect(HasSubType("Mountain"), "Plains")
	change.SetSourceID(source.ID())
	g.AddContinuousEffect(change)
	for range 3 {
		g.ApplyContinuousEffects()
		p := g.FindPermanent(land.ID())
		if !p.HasSubType("Plains") || hasRuntimeAbility(p, printed) || countManaAbilityColor(p, core.White) != 1 || countManaAbilityColor(p, core.Blue) != 1 {
			t.Fatal("type change or granted ability retention")
		}
	}
	ids := g.PhaseOut(source.ID())
	g.ApplyContinuousEffects()
	if !g.FindPermanent(land.ID()).HasSubType("Mountain") || !hasRuntimeAbility(g.FindPermanent(land.ID()), printed) {
		t.Fatal("phased source kept static type effect")
	}
	g.PhaseIn(ids)
	g.ApplyContinuousEffects()
	ids = g.PhaseOut(land.ID())
	g.ApplyContinuousEffects()
	g.PhaseIn(ids)
	g.ApplyContinuousEffects()
	if !g.FindPermanent(land.ID()).HasSubType("Plains") {
		t.Fatal("target phase-in missed current membership")
	}
	g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: land.ID(), ControllerID: g.AllPlayers()[1].PlayerID(), Duration: core.Indefinite})
	if !g.FindPermanent(land.ID()).HasSubType("Plains") {
		t.Fatal("global selection changed with target controller")
	}
}

func TestBasicLandSetDeclarationIsolation(t *testing.T) {
	types := []string{"Plains", "Island"}
	effect := BecomesBasicLandsEffect(IsLand, types...)
	types[0] = "Swamp"
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := g.PutOnBattlefield(NewLand("Animated Gate", WithCardType(core.TypeCreature), WithSubTypes("Gate", "Elemental"), WithSuperTypes(core.SuperLegendary)), b.PlayerID())
	source := g.PutOnBattlefield(NewEnchantment("Source", "", WithStaticAbility(effect)), a.PlayerID())
	p := g.FindPermanent(land.ID())
	if !p.HasSubType("Plains") || !p.HasSubType("Island") || p.HasSubType("Swamp") || p.HasSubType("Gate") || !p.HasSubType("Elemental") || !p.HasType(core.TypeCreature) || !IsLegendary.Match(p, g) {
		t.Fatal("declaration mutation or subtype-family loss")
	}
	if countManaAbilityColor(p, core.White) != 1 || countManaAbilityColor(p, core.Blue) != 1 {
		t.Fatal("multiple basic types did not supply mana")
	}
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(source.ID()))
	if !g.FindPermanent(land.ID()).HasSubType("Plains") || !clone.FindPermanent(land.ID()).HasSubType("Gate") {
		t.Fatal("clone changed immutable declaration")
	}
	for _, subtypes := range [][]string{nil, {"Desert"}, {"Plains", "Invalid"}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatal("invalid basic type declaration accepted")
				}
			}()
			BecomesBasicLandsEffect(IsLand, subtypes...)
		}()
	}
}

func TestBasicLandDependencyWaitsForLoop(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	mountain := g.PutOnBattlefield(NewLand("Mountain", WithSubTypes("Mountain")), b.PlayerID())
	forest := g.PutOnBattlefield(NewLand("Forest", WithSubTypes("Forest")), b.PlayerID())
	for _, effect := range []ContinuousEffect{
		BecomesBasicLandsEffect(HasSubType("Mountain"), "Mountain", "Island"),
		BecomesBasicLandsEffect(HasSubType("Mountain"), "Forest"),
		BecomesBasicLandsEffect(HasSubType("Forest"), "Mountain"),
	} {
		g.PutOnBattlefield(NewEnchantment("Change", "", WithStaticAbility(effect)), a.PlayerID())
	}
	for _, id := range []uuid.UUID{mountain.ID(), forest.ID()} {
		if !g.FindPermanent(id).HasSubType("Island") {
			t.Fatal("dependent effect did not wait for the loop")
		}
	}
}
