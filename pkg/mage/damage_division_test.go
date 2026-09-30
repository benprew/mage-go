package mage

import (
	"maps"
	"math"
	"testing"

	"github.com/google/uuid"
)

func TestDamageDivisionNormalization(t *testing.T) {
	a, b, c, other := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	targets := []uuid.UUID{a, b, c}
	fallback := map[uuid.UUID]int{a: 2, b: 1, c: 1}
	for _, tc := range []struct {
		name  string
		input map[uuid.UUID]int
		want  map[uuid.UUID]int
	}{
		{"valid", map[uuid.UUID]int{a: 1, b: 2, c: 1}, map[uuid.UUID]int{a: 1, b: 2, c: 1}},
		{"nil", nil, fallback},
		{"missing", map[uuid.UUID]int{a: 4}, fallback},
		{"zero", map[uuid.UUID]int{a: 3, b: 1, c: 0}, fallback},
		{"negative", map[uuid.UUID]int{a: 4, b: 1, c: -1}, fallback},
		{"unknown", map[uuid.UUID]int{a: 1, b: 1, c: 1, other: 1}, fallback},
		{"excess", map[uuid.UUID]int{a: 2, b: 2, c: 2}, fallback},
		{"short", map[uuid.UUID]int{a: 1, b: 1, c: 1}, fallback},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := maps.Clone(tc.input)
			got := sanitizeDamageDistribution(tc.input, targets, 4)
			if !maps.Equal(got, tc.want) {
				t.Fatalf("distribution = %v, want %v", got, tc.want)
			}
			if !maps.Equal(tc.input, before) {
				t.Fatal("normalization changed the player response")
			}
			got[a] = 99
			if !maps.Equal(tc.input, before) {
				t.Fatal("output shares the player response")
			}
		})
	}
	maximum := map[uuid.UUID]int{a: math.MaxInt - 2, b: 1, c: 1}
	if got := sanitizeDamageDistribution(maximum, targets, math.MaxInt); !maps.Equal(got, maximum) {
		t.Fatalf("maximum legal allocation changed: %v", got)
	}
	overflow := map[uuid.UUID]int{a: math.MaxInt, b: math.MaxInt, c: 5}
	if got := sanitizeDamageDistribution(overflow, targets, 3); !maps.Equal(got, map[uuid.UUID]int{a: 1, b: 1, c: 1}) {
		t.Fatalf("overflow accepted: %v", got)
	}
	for _, tc := range []struct {
		name    string
		targets []uuid.UUID
		total   int
	}{
		{"too many", targets, 2}, {"duplicate", []uuid.UUID{a, a}, 4},
		{"nil target", []uuid.UUID{a, uuid.Nil}, 4}, {"empty", nil, 4},
		{"zero", targets, 0}, {"negative", targets, -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := sanitizeDamageDistribution(nil, tc.targets, tc.total); got != nil {
				t.Fatalf("impossible division = %v", got)
			}
		})
	}
}

func TestDamageDivisionNormalizationExhaustive(t *testing.T) {
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	for count := 1; count <= 3; count++ {
		for total := count; total <= 5; total++ {
			var visit func(int, map[uuid.UUID]int)
			visit = func(index int, input map[uuid.UUID]int) {
				if index < count {
					for amount := -1; amount <= 6; amount++ {
						input[ids[index]] = amount
						visit(index+1, input)
					}
					return
				}
				sum, valid := 0, true
				for _, amount := range input {
					sum += amount
					valid = valid && amount > 0
				}
				got := sanitizeDamageDistribution(input, ids[:count], total)
				if valid && sum == total && !maps.Equal(input, got) {
					t.Fatalf("legal allocation changed: %v -> %v", input, got)
				}
				actual := 0
				for _, id := range ids[:count] {
					if got[id] < 1 {
						t.Fatalf("zero share: %v -> %v", input, got)
					}
					actual += got[id]
				}
				if len(got) != count || actual != total {
					t.Fatalf("wrong targets or total: %v", got)
				}
			}
			visit(0, make(map[uuid.UUID]int))
		}
	}
}

func TestDamageDivisionSpellActivationCopyClone(t *testing.T) {
	for _, activation := range []bool{false, true} {
		t.Run(map[bool]string{false: "spell", true: "activation"}[activation], func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			victim := g.PutOnBattlefield(NewCreature("Division victim", "", 2, 5), b.PlayerID())
			targets := []uuid.UUID{victim.ID(), b.PlayerID()}
			effect := DealDividedDamage(Fixed(4))
			if activation {
				source := g.PutOnBattlefield(NewArtifact("Division source", "", WithActivatedAbility(effect, Tap(), WithTarget(TargetUpToNCreaturesOrPlayers(2)))), a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets); err != nil {
					t.Fatal(err)
				}
			} else {
				spell := NewInstant("Division spell", "", NewMultiTargetSpell([]Target{TargetUpToNCreaturesOrPlayers(2)}, effect))
				spell.SetOwner(a.PlayerID())
				a.AddToHand(spell)
				if err := g.CastSpellByName(a.PlayerID(), spell.Name(), targets); err != nil {
					t.Fatal(err)
				}
			}
			original := g.StackPeek()
			want := map[uuid.UUID]int{victim.ID(): 3, b.PlayerID(): 1}
			if !maps.Equal(original.DamageDistribution, want) {
				t.Fatalf("announced distribution = %v", original.DamageDistribution)
			}
			clone := g.Clone()
			clonedOriginal := clone.StackPeek()
			copied := clone.CopyStackObjectDirect(clonedOriginal, a.PlayerID(), false)
			if !maps.Equal(copied.DamageDistribution, want) {
				t.Fatal("copy lost the division")
			}
			clone.PhaseOut(victim.ID())
			clone.ResolveStackObject(copied)
			if clone.GetPlayer(b.PlayerID()).Life() != 19 {
				t.Fatal("illegal target share was redistributed")
			}
			copied.DamageDistribution[b.PlayerID()] = 7
			if !maps.Equal(original.DamageDistribution, want) || !maps.Equal(clonedOriginal.DamageDistribution, want) {
				t.Fatal("copy or clone shared a division")
			}
			g.ResolveStack()
			if b.Life() != 19 || g.FindPermanent(victim.ID()).Damage != 3 {
				t.Fatal("original division did not resolve independently")
			}
		})
	}
}

func TestDamageDivisionDefaultPlayer(t *testing.T) {
	p := NewBasePlayer("A")
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	want := map[uuid.UUID]int{ids[0]: 2, ids[1]: 1, ids[2]: 1}
	if got := p.ChooseDamageDistribution(ids, 4, "division", nil); !maps.Equal(got, want) {
		t.Fatalf("default division = %v, want %v", got, want)
	}
}
