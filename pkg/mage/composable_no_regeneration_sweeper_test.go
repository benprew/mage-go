package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCollectionNoRegenerationPreservesCharacteristics(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Victim", "", 2, 2)
	c.SetOwner(a.PlayerID())
	g.PutOnBattlefield(c, a.PlayerID())
	g.AddRegenerationShield(c.ID())
	if err := DestroyEachWithoutRegeneration(MatchingPermanents(IsCreature)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(c.ID()) != nil {
		t.Fatal("regeneration prevented destruction")
	}
	lki := g.LookupObject(c.ID())
	if lki == nil || lki.ViewHasKeyword(core.CantRegenerate) {
		t.Fatal("destruction changed the victim's last-known characteristics")
	}
}

func TestDestroyEachWithoutRegenerationCollections(t *testing.T) {
	for _, artifactsOnly := range []bool{false, true} {
		name := "creatures"
		filter := IsCreature
		if artifactsOnly {
			name = "artifacts"
			filter = IsArtifact
		}
		t.Run(name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			effect := DestroyEachWithoutRegeneration(MatchingPermanents(filter))
			if err := effect.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			var destroyed, survivors []*Permanent
			for _, owner := range []*BasePlayer{a, b} {
				for _, scenario := range []string{"ordinary", "indestructible", "regenerate", "phased", "excluded"} {
					card := NewCreature(scenario, "", 2, 2)
					if artifactsOnly {
						card = NewArtifact(scenario, "")
					}
					if scenario == "excluded" {
						card = NewEnchantment("excluded", "")
					}
					card.SetOwner(owner.PlayerID())
					p := g.PutOnBattlefield(card, owner.PlayerID())
					switch scenario {
					case "indestructible":
						p.GrantBaseAttr(core.Indestructible)
					case "regenerate":
						g.AddRegenerationShield(p.ID())
					case "phased":
						p.PhasedOut = true
					}
					if scenario == "ordinary" || scenario == "regenerate" {
						destroyed = append(destroyed, p)
					} else {
						survivors = append(survivors, p)
					}
				}
			}
			clone := g.Clone()
			if err := effect.Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			for _, p := range destroyed {
				if clone.FindPermanent(p.ID()) != nil || g.FindPermanent(p.ID()) == nil {
					t.Fatalf("wrong destruction or clone isolation for %s", p.Name())
				}
			}
			for _, p := range survivors {
				got := clone.FindPermanentIncludingPhased(p.ID())
				if got == nil {
					t.Fatalf("destroyed %s", p.Name())
				}
			}
		})
	}
}

func TestNoRegenerationSimultaneousDeathObservation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ab := WhenEvent(CreatureDies(), GainLifeFor(You, CreaturesDiedThisTurn()), WithInterveningIf(deathCountConditionForTest(2)))
	first := NewCreature("Observer", "", 1, 1, WithAbility(ab))
	second := NewCreature("Victim", "", 1, 1)
	protected := NewCreature("Survivor", "", 1, 1, WithKeyword(core.Indestructible))
	state := NewEnchantment("State observer", "", WithAbility(NewStateTriggered(false, Sacrifice(This)).SetCondition(
		func(_ *core.GameEvent, game GameReader, _, _ uuid.UUID) bool { return game.CreatureDeaths() == 1 })))
	g.PutOnBattlefield(state, a.PlayerID())
	for _, card := range []Card{first, second, protected} {
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
	}
	clone := g.Clone()
	clone.AddRegenerationShield(first.ID())
	clone.AddRegenerationShield(second.ID())
	if err := DestroyEachWithoutRegeneration(MatchingPermanents(IsCreature)).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	clone.ResolveStack()
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 24 {
		t.Fatalf("clone life = %d, want 24 from two triggers that each see two deaths", got)
	}
	if clone.CreatureDeaths() != 2 || g.CreatureDeaths() != 0 || a.Life() != 20 || g.FindPermanent(first.ID()) == nil {
		t.Fatal("simultaneous deaths changed the original game or counted prevented destruction")
	}
	if clone.FindPermanent(state.ID()) == nil {
		t.Fatal("state trigger observed a partial simultaneous group")
	}
}
