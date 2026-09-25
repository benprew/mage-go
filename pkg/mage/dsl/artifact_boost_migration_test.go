package dsl_test

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestArtifactBoostTypedDefinitions(t *testing.T) {
	for _, name := range []string{"Elven Lyre", "Coral Helm"} {
		t.Run(name, func(t *testing.T) {
			card, err := mage.CreateCard(name)
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition(name, card); err != nil {
				t.Fatal(err)
			}
			ability := mage.UnwrapAbility(card.Abilities()[0]).(mage.ActivatedAbility)
			if ability.SorcerySpeed() || len(ability.Targets()) != 1 {
				t.Fatal("expected one target and instant timing")
			}
			if _, ok := ability.Targets()[0].(mage.TargetBinding); !ok {
				t.Fatal("missing typed target")
			}
			metadata := mage.InspectDefinition(ability.Effects()[0])
			if metadata.Role != "until" {
				t.Fatalf("effect role = %q, want until", metadata.Role)
			}
			props := metadata.Properties
			if props.Mass || props.Outcome != mage.OutcomeBenefit || props.PowerBoost != 2 || props.ToughnessBoost != 2 {
				t.Fatalf("boost metadata = %+v", props)
			}
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			want := make([]uuid.UUID, 0, 2)
			for _, player := range []*mage.BasePlayer{a, b} {
				p := g.PutOnBattlefield(mage.NewCreature("Creature", "", 2, 2), player.PlayerID())
				want = append(want, p.ID())
				g.PutOnBattlefield(mage.NewArtifact("Artifact", ""), player.PlayerID())
			}
			got := ability.Targets()[0].Possible(a.PlayerID(), card, g)
			if len(got) != len(want) {
				t.Fatalf("targets = %v, want %v", got, want)
			}
			for _, id := range want {
				if !slices.Contains(got, id) {
					t.Fatalf("missing target %v", id)
				}
			}
		})
	}
}
