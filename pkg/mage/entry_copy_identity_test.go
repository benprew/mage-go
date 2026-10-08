package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func entryCopyPhysical(t *testing.T) Card {
	t.Helper()
	plan, err := NewCardDefinitionPlan(func() Card { return NewCreature("Physical", "{3}", 0, 1) })
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	return card
}

func TestEntryCopyIdentityInvalid(t *testing.T) {
	g := NewGame(NewBasePlayer("A"), NewBasePlayer("B"))
	source := g.PutOnBattlefield(copiableTestCard(t), g.players[0].PlayerID())
	values, err := g.CaptureCopiableValues(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, original := range []Card{nil, (*BaseCard)(nil), NewCreature("Legacy", "", 1, 1)} {
		if _, err := BuildEntryCopy(original, values); err == nil {
			t.Fatal("unknown physical provenance accepted")
		}
	}
	for _, invalid := range []*CopiableValues{nil, {}} {
		if _, err := BuildEntryCopy(entryCopyPhysical(t), invalid); err == nil {
			t.Fatal("unknown copy provenance accepted")
		}
	}
	for _, change := range []string{"identity", "metadata", "plan", "spell"} {
		original := entryCopyPhysical(t).(*BaseCard)
		switch change {
		case "identity":
			original.SetID(uuid.Nil)
		case "metadata":
			original.SetBasePT(9, 9)
		case "plan":
			WithPrintedCardDefinitionPlan(source.Card.(*BaseCard).definitionPlan)(original)
		case "spell":
			plan, err := NewCardDefinitionPlan(func() Card { return NewInstant("Spell", "", NewSpellAbility()) })
			if err != nil {
				t.Fatal(err)
			}
			built, err := plan.Build()
			if err != nil {
				t.Fatal(err)
			}
			original = built.(*BaseCard)
		}
		if _, err := BuildEntryCopy(original, values); err == nil {
			t.Fatalf("invalid %s accepted", change)
		}
	}
}

func TestEntryCopyIdentityIndependentAndCopyOfCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(copiableTestCard(t), a.PlayerID())
	values, err := g.CaptureCopiableValues(source)
	if err != nil {
		t.Fatal(err)
	}
	original := entryCopyPhysical(t)
	original.SetOwner(b.PlayerID())
	first, err := BuildEntryCopy(original, values)
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildEntryCopy(original, values)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID() != original.ID() || second.ID() != original.ID() || first.Abilities()[0] == second.Abilities()[0] || first.Abilities()[0].(ActivatedAbility).Targets()[0] == second.Abilities()[0].(ActivatedAbility).Targets()[0] {
		t.Fatal("entry copy did not preserve identity with separate rules")
	}
	p := g.PutOnBattlefield(first, b.PlayerID())
	saved, err := g.CaptureCopiableValues(p)
	if err != nil {
		t.Fatal(err)
	}
	other := entryCopyPhysical(t)
	other.SetOwner(a.PlayerID())
	third, err := BuildEntryCopy(other, saved)
	if err != nil {
		t.Fatal(err)
	}
	thirdPerm := g.PutOnBattlefield(third, a.PlayerID())
	if thirdPerm.Name() != source.Name() || thirdPerm.Card.ID() == first.ID() {
		t.Fatal("copy of a copy inherited physical identity")
	}
	g.ExilePermanent(thirdPerm)
	if g.FindExiledCard(other.ID()).Card.Name() != "Physical" {
		t.Fatal("copy of a copy lost its own physical definition")
	}
	ordinary := first.Copy()
	newID := uuid.New()
	ordinary.SetID(newID)
	ordinary.SetOwner(a.PlayerID())
	q := g.PutOnBattlefield(ordinary, a.PlayerID())
	g.BouncePermanentToHand(q)
	if a.Hand()[0].ID() != newID || a.Hand()[0].Owner() != a.PlayerID() || a.Hand()[0].Name() != "Physical" || first.Owner() != b.PlayerID() {
		t.Fatal("ordinary copy shared physical identity state")
	}
}

func TestEntryCopyIdentityFaceDownAndTokens(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(copiableTestCard(t), a.PlayerID())
	source.FaceDown = true
	values, err := g.CaptureCopiableValues(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []bool{false, true} {
		physical := entryCopyPhysical(t).(*BaseCard)
		physical.isToken = token
		physical.SetOwner(b.PlayerID())
		card, err := BuildEntryCopy(physical, values)
		if err != nil {
			t.Fatal(err)
		}
		p := g.PutOnBattlefield(card, b.PlayerID())
		if p.FaceDown || p.Name() != "" || p.CurrentPower(g) != 2 || p.Card.(*BaseCard).HasManaCost() || p.IsToken != token || p.HasAttr(core.Flying) {
			t.Fatal("face-down values transferred status or lost token identity")
		}
		g.BouncePermanentToHand(p)
	}
	if len(b.Hand()) != 1 || b.Hand()[0].Name() != "Physical" {
		t.Fatal("token retained a destination card or face-down copy lost physical rules")
	}
}

func TestEntryCopyIdentityOtherDeparturePaths(t *testing.T) {
	for _, path := range []string{"ante", "gathered-bounce", "return-cost", "failed-entry"} {
		t.Run(path, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			physical := entryCopyPhysical(t)
			physical.SetOwner(b.PlayerID())
			var values *CopiableValues
			if path == "failed-entry" {
				plan, err := NewCardDefinitionPlan(func() Card {
					return NewArtifact("Rejected entry", "", WithAbility(&optionalEntryProbe{
						BaseAbility: BaseAbility{id: uuid.New(), abilityType: core.AbilityStatic},
						run:         func(*Game, *Permanent) bool { return false },
					}))
				})
				if err != nil {
					t.Fatal(err)
				}
				values = plan.CopiableValues()
			} else {
				source := g.PutOnBattlefield(copiableTestCard(t), a.PlayerID())
				var err error
				values, err = g.CaptureCopiableValues(source)
				if err != nil {
					t.Fatal(err)
				}
			}
			card, err := BuildEntryCopy(physical, values)
			if err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(card, b.PlayerID())
			var destination []Card
			switch path {
			case "ante":
				g.anteEnabled = true
				if err := g.MoveToAnte(b.PlayerID(), p.ID()); err != nil {
					t.Fatal(err)
				}
				destination = b.Ante()
			case "gathered-bounce":
				ctx := &EffectContext{Game: g, Vars: make(map[string]any)}
				ctx.SetUUID("victim", p.ID())
				if err := BounceGathered("victim").Apply(ctx); err != nil {
					t.Fatal(err)
				}
				destination = b.Hand()
			case "return-cost":
				if err := ReturnToHandCost(nil).Pay(uuid.New(), b.PlayerID(), g); err != nil {
					t.Fatal(err)
				}
				destination = b.Hand()
			case "failed-entry":
				if p != nil {
					t.Fatal("failed entry installed a permanent")
				}
				destination = b.Graveyard()
			}
			if len(destination) != 1 || destination[0].Name() != "Physical" || destination[0].ID() != physical.ID() {
				t.Fatal("departure retained effective copy rules")
			}
		})
	}
}
