package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTargetOnlySubtype(t *testing.T) {
	g, _, _ := newSpellCopyGame()
	wall := DeclarePermanentTarget("wall", AnyPermanent, TargetHasSubtype("Wall"))
	broad := DeclarePermanentTargets("optional", 0, 1, AnyPermanent)
	for _, tc := range []struct {
		name    string
		targets []Target
		want    bool
	}{
		{"wall", []Target{wall}, true},
		{"two walls", []Target{wall, wall}, true},
		{"empty", nil, false},
		{"unknown", []Target{wall, broad}, false},
		{"zero broad", []Target{wall, &countBoundTarget{inner: broad.spec, count: 0}}, false},
		{"zero wall", []Target{wall, &countBoundTarget{inner: wall.spec, count: 0}}, true},
		{"player", []Target{wall, DeclarePlayerTarget("player")}, false},
		{"spell", []Target{wall, DeclareSpellTarget("spell")}, false},
		{"contradiction", []Target{DeclarePermanentTarget("impossible", AnyPermanent, TargetWithDomain(AllPermanentDomains(SubtypeDomain("Wall"), NonSubtypeDomain("Wall"))))}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewInstant("Spell", "")
			ability := Activate(Costs(GenericCost(0)), GainLifeFor(You, Fixed(1)))
			for _, source := range []Card{SpellTargetSourceWithDeclarations(card, tc.targets), g.AbilityTargetSource(card, ability, tc.targets)} {
				if got := TargetSourceCanTargetOnlySubtype(source, "Wall"); got != tc.want {
					t.Fatalf("proof = %v, want %v", got, tc.want)
				}
				if CanTargetOnlySubtypeFilter("Wall").Match(source) != tc.want {
					t.Fatal("filter differs from proof")
				}
				if TargetSourceCanTargetOnlySubtype(source, "Elf") {
					t.Fatal("wrong subtype")
				}
			}
		})
	}
	card := NewInstant("Wall spell", "", Spell(TapSubject(wall), WithTargets(wall)))
	if !TargetSourceCanTargetOnlySubtype(SpellTargetSource(card), "Wall") {
		t.Fatal("ordinary spell declarations lost")
	}
	for _, source := range []Card{nil, card, g.AbilityTargetSource(card, nil, nil), SpellTargetSource(NewInstant("Modal", "", WithAbility(SpellModes(ModeOf("wall", TapSubject(wall), WithTargets(wall)), ModeOf("life", GainLifeFor(You, Fixed(1)))))))} {
		if TargetSourceCanTargetOnlySubtype(source, "Wall") {
			t.Fatal("unknown source supplied proof")
		}
	}
	protected := NewCreature("Protected", "", 0, 5, WithAbility(Static(CantBeTargetedBySpells(This, CanTargetOnlySubtypeFilter("Wall")))))
	if err := ValidateCardDefinition(protected.Name(), protected); err != nil {
		t.Fatal(err)
	}
	for _, subtype := range []string{"", " Wall", "Wall "} {
		invalid := NewCreature("Invalid", "", 0, 5, WithAbility(Static(CantBeTargetedByAbilities(This, CanTargetOnlySubtypeFilter(subtype)))))
		if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil {
			t.Fatal("accepted invalid subtype")
		}
	}
	input := []Target{wall}
	view := SpellTargetSourceWithDeclarations(card, input)
	input[0] = broad
	if !TargetSourceCanTargetOnlySubtype(view, "Wall") {
		t.Fatal("input changed view")
	}
}

func TestTargetOnlySubtypeSelectedModes(t *testing.T) {
	for _, choice := range []int{0, 1} {
		g, a, b := newSpellCopyGame()
		caster := &targetOnlyModePlayer{BasePlayer: a.BasePlayer, mode: choice}
		g.players[0] = caster
		protected := g.PutOnBattlefield(NewCreature("Protected", "", 0, 5, WithSubTypes("Wall"), WithAbility(Static(CantBeTargetedBySpells(This, CanTargetOnlySubtypeFilter("Wall"))))), b.PlayerID())
		wall := DeclarePermanentTarget("wall", AnyPermanent, TargetHasSubtype("Wall"))
		broad := DeclarePermanentTarget("broad", AnyPermanent)
		card := NewInstant("Modal", "{1}", WithAbility(SpellModes(ModeOf("wall", TapSubject(wall), WithTargets(wall)), ModeOf("broad", TapSubject(broad), WithTargets(broad)))))
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		a.ManaPool().Add(core.Colorless, 1)
		err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{protected.ID()}, 0)
		if choice == 0 {
			if err == nil || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
				t.Fatal("wall mode paid for illegal target")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if TargetSourceCanTargetOnlySubtype(g.resolutionTargetSource(g.StackPeek()), "Wall") {
			t.Fatal("unselected mode supplied proof")
		}
		clone := g.Clone()
		cp := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true)
		if cp == nil || TargetSourceCanTargetOnlySubtype(g.resolutionTargetSource(cp), "Wall") {
			t.Fatal("copy changed mode proof")
		}
		g.ResolveStack()
		clone.ResolveStack()
		if !g.FindPermanent(protected.ID()).Tapped || !clone.FindPermanent(protected.ID()).Tapped {
			t.Fatal("mode failed at resolution")
		}
	}
}

type targetOnlyModePlayer struct {
	*BasePlayer
	mode int
}

func (p *targetOnlyModePlayer) ChooseMode([]string, string) int { return p.mode }

func TestTargetOnlySubtypeMixedZeroGroups(t *testing.T) {
	for _, wallOnly := range []bool{false, true} {
		g, a, b := newSpellCopyGame()
		protected := g.PutOnBattlefield(NewCreature("Protected", "", 0, 5, WithSubTypes("Wall"), WithAbility(Static(CantBeTargetedBySpells(This, CanTargetOnlySubtypeFilter("Wall"))))), b.PlayerID())
		first := DeclarePermanentTargets("first", 1, 1, AnyPermanent, TargetHasSubtype("Wall"))
		zero := DeclarePermanentTargets("zero", 0, 0, AnyPermanent)
		if wallOnly {
			zero = DeclarePermanentTargets("zero", 0, 0, AnyPermanent, TargetHasSubtype("Wall"))
		}
		member := BindPermanent("member")
		card := NewInstant("Mixed zero", "{1}", Spell(ForEach(first, member, TapSubject(From(member))), WithTargetGroups(first, zero)))
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		a.ManaPool().Add(core.Colorless, 1)
		err := g.CastSpellByNameWithTargetGroups(a.PlayerID(), card.Name(), [][]uuid.UUID{{protected.ID()}, {}}, 0)
		if wallOnly {
			if err == nil || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
				t.Fatal("zero wall group lost proof or rollback")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if len(g.StackPeek().TargetSpecs) != 1 || TargetSourceCanTargetOnlySubtype(g.resolutionTargetSource(g.StackPeek()), "Wall") {
			t.Fatal("zero broad group omitted")
		}
		clone := g.Clone()
		copied := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true)
		if copied == nil || TargetSourceCanTargetOnlySubtype(g.resolutionTargetSource(copied), "Wall") || TargetSourceCanTargetOnlySubtype(clone.resolutionTargetSource(clone.StackPeek()), "Wall") {
			t.Fatal("copy/clone omitted zero broad group")
		}
		g.ResolveStack()
		clone.ResolveStack()
		if !g.FindPermanent(protected.ID()).Tapped || !clone.FindPermanent(protected.ID()).Tapped {
			t.Fatal("zero broad group lost at resolution")
		}
	}
}

func TestTargetOnlySubtypeSourceIsolation(t *testing.T) {
	g, _, _ := newSpellCopyGame()
	wall := DeclarePermanentTarget("wall", AnyPermanent, TargetHasSubtype("Wall"))
	card := NewInstant("Wall spell", "", Spell(TapSubject(wall), WithTargets(wall)))
	unrelated := NewInstant("Unrelated", "")
	obj := &StackObject{SourceID: card.ID(), Card: card, TargetSource: unrelated}
	source := g.resolutionTargetSource(obj)
	if source != unrelated || TargetSourceCanTargetOnlySubtype(source, "Wall") {
		t.Fatal("unrelated source inherited spell declaration")
	}
}
