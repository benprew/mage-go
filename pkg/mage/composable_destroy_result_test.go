package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDestroyEachCreatureDeathsResult(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	count := BindNumber("creatures died this way")
	makeCreature := func(name string) *Permanent {
		card := NewCreature(name, "{1}", 2, 2)
		card.SetOwner(b.PlayerID())
		return g.PutOnBattlefield(card, b.PlayerID())
	}
	first := makeCreature("First")
	second := makeCreature("Second")
	regenerated := makeCreature("Regenerated")
	g.AddRegenerationShield(regenerated.ID())
	indestructible := makeCreature("Indestructible")
	indestructible.GrantBaseAttr(core.Indestructible)
	phased := makeCreature("Phased")
	phased.PhasedOut = true
	artifact := NewArtifact("Artifact", "")
	artifact.SetOwner(b.PlayerID())
	g.PutOnBattlefield(artifact, b.PlayerID())
	body := Sequence(
		DestroyEach(MatchingPermanents(Or(IsCreature, IsArtifact)), CreaturesDiedThisWayInto(count)),
		GainLifeFor(You, FromNumber(count)),
	)
	clone := g.Clone()
	if err := body.Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if clone.resolution.numberBindings[count] != 2 || clone.GetPlayer(a.PlayerID()).Life() != 22 {
		t.Fatalf("clone result = %d, life = %d", clone.resolution.numberBindings[count], clone.GetPlayer(a.PlayerID()).Life())
	}
	if g.FindPermanent(first.ID()) == nil || g.FindPermanent(second.ID()) == nil || a.Life() != 20 {
		t.Fatal("clone changed the original game")
	}
	if err := body.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.resolution.numberBindings[count] != 2 || a.Life() != 22 || g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) != nil {
		t.Fatal("result did not count the two destroyed creatures")
	}
	for _, p := range []*Permanent{regenerated, indestructible, phased} {
		if g.FindPermanentIncludingPhased(p.ID()) == nil {
			t.Fatalf("%s should remain", p.Name())
		}
	}
	if err := DestroyEach(MatchingPermanents(IsLand), CreaturesDiedThisWayInto(count)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.resolution.numberBindings[count] != 0 || a.Life() != 22 {
		t.Fatal("second instruction retained a stale result")
	}
}

func TestDestroyEachCreatureDeathsResultDefinition(t *testing.T) {
	count := BindNumber("creatures died")
	set := MatchingPermanents(IsCreature)
	valid := NewSorcery("Valid", "", WithAbility(Spell(Sequence(
		DestroyEach(set, CreaturesDiedThisWayInto(count)),
		GainLifeFor(You, FromNumber(count)),
	))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		body ResolvingEffect
		want string
	}{
		{"nil", DestroyEach(set, CreaturesDiedThisWayInto(nil)), "binding"},
		{"empty", DestroyEach(set, CreaturesDiedThisWayInto(BindNumber(""))), "name"},
		{"duplicate", DestroyEach(set, CreaturesDiedThisWayInto(count), CreaturesDiedThisWayInto(count)), "once"},
		{"missing", GainLifeFor(You, FromNumber(count)), "producer"},
		{"late", Sequence(GainLifeFor(You, FromNumber(count)), DestroyEach(set, CreaturesDiedThisWayInto(count))), "producer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewSorcery("Invalid", "", WithAbility(Spell(tc.body)))
			if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(strings.ToLower(err.Error()), tc.want) {
				t.Fatalf("error %v does not contain %q", err, tc.want)
			}
		})
	}
}

func TestDestroyEachCreatureDeathsResultWithoutRegeneration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	count := BindNumber("deaths")
	card := NewCreature("Shielded", "", 2, 2)
	card.SetOwner(b.PlayerID())
	shielded := g.PutOnBattlefield(card, b.PlayerID())
	g.AddRegenerationShield(shielded.ID())
	if err := DestroyEachWithoutRegeneration(MatchingPermanents(IsCreature), CreaturesDiedThisWayInto(count)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if got := g.resolution.numberBindings[count]; got != 1 || g.FindPermanent(shielded.ID()) != nil {
		t.Fatalf("result = %d; shielded permanent still present = %t", got, g.FindPermanent(shielded.ID()) != nil)
	}
}

func TestDestroyEachCreatureDeathsResultSpellDamage(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	count := BindNumber("nonblack creatures died")
	spell := NewSorcery("Death Count", "", WithAbility(Spell(Sequence(
		DestroyEach(MatchingPermanents(And(IsCreature, Not(HasColorFilter(core.Black)))), CreaturesDiedThisWayInto(count)),
		DealDamageFrom(This, You, FromNumber(count)),
	))))
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"White", "Black"} {
		cost := "{W}"
		if name == "Black" {
			cost = "{B}"
		}
		card := NewCreature(name, cost, 2, 2)
		card.SetOwner(b.PlayerID())
		g.PutOnBattlefield(card, b.PlayerID())
	}
	a.AddToHand(spell)
	g.SetStep(core.PrecombatMain)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 20 {
		t.Fatalf("life totals = %d/%d; want 19/20", a.Life(), b.Life())
	}
	if len(g.FilterBattlefield(HasColorFilter(core.Black))) != 1 {
		t.Fatal("black creature should remain")
	}
}
