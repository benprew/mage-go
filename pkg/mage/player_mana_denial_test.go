package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestLoseUnspentManaFor(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.ManaPool().Add(Blue, 2)
	b.ManaPool().Add(Red, 3)
	b.ManaPool().AddRestricted(Colorless, 2, ArtifactSpellsOnly{})
	target := DeclarePlayerTarget("player")
	card := NewInstant("Empty", "", Spell(LoseUnspentManaFor(target), WithTargets(target)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	clone.ResolveStack()
	if b.ManaPool().TotalMana() != 5 || clone.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != 0 {
		t.Fatal("clone changed live mana")
	}
	g.ResolveStack()
	if a.ManaPool().TotalMana() != 2 || b.ManaPool().TotalMana() != 0 || b.Life() != 20 || len(b.ManaPool().ProducedThisTurn) != 5 {
		t.Fatal("mana loss changed another pool, life or production history")
	}
	for _, ref := range []PlayerRef{nil, FromPlayer(BindPlayer("missing")), DeclarePlayerTarget("missing")} {
		invalid := NewInstant("Invalid", "", Spell(LoseUnspentManaFor(ref)))
		if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil {
			t.Fatal("accepted missing player declaration")
		}
	}
}

func TestHasManaAbilityFilter(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	cards := []Card{
		NewLand("Fixed", WithManaAbility(Blue)),
		NewLand("Conditional", WithAbility(NewConditionalManaAbility(StateConditionFunc("false", nil, func(StateReader, StateContext) bool { return false }), []ManaProduction{{Color: Blue, Amount: 1}}, nil))),
		NewLand("Triggered", WithAbility(WhenEvent(LandTappedForMana(), AddManaAmount(Green, Fixed(1))))),
		NewLand("Not mana", WithAbility(WhenEvent(Enters(This), AddManaAmount(Green, Fixed(1))))),
		NewLand("Targeted", WithAbility(Activate(Costs(), AddManaAmount(Green, Fixed(1)), WithTargets(DeclarePlayerTarget("target"))))),
		NewLand("Empty"),
	}
	for i, c := range cards {
		p := g.PutOnBattlefield(c, a.PlayerID())
		if got := HasManaAbility.Match(p, g); got != (i < 3) {
			t.Fatalf("%s mana=%v", c.Name(), got)
		}
		if i == 0 {
			if err := ApplyEffect(g, Until(EndOfTurn, RemoveAbilities(This)), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.ApplyContinuousEffects()
			if HasManaAbility.Match(g.FindPermanent(p.ID()), g) {
				t.Fatal("removed ability still matched")
			}
		}
	}
}

func TestSpellControllerWithLKIDeparture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclareSpellTarget("spell")
	victim := NewInstant("Victim", "", Spell(GainLifeFor(You, Fixed(1))))
	victim.SetOwner(a.PlayerID())
	a.AddToHand(victim)
	if err := g.CastSpellByID(a.PlayerID(), victim.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.StackPeek().Controller = b.PlayerID()
	victimStackID := g.StackPeek().ID
	effect := Sequence(Counter(target), GainLifeFor(ControllerOfSpellWithLKI(target), Fixed(3)))
	tax := NewInstant("Tax", "", Spell(effect, WithTargets(target)))
	tax.SetOwner(a.PlayerID())
	a.AddToHand(tax)
	if err := g.CastSpellByID(a.PlayerID(), tax.ID(), []uuid.UUID{victimStackID}, 0); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	clone.ResolveStack()
	g.ResolveStack()
	for _, game := range []*Game{g, clone} {
		if game.GetPlayer(b.PlayerID()).Life() != 23 || game.GetPlayer(a.PlayerID()).Life() != 20 {
			t.Fatal("counter lost departure controller")
		}
	}
	old := g.stack.spellControllers[victimStackID]
	g.PutOnBattlefield(NewLand("Unused"), a.PlayerID())
	if old != b.PlayerID() {
		t.Fatal("departure did not retain changed controller")
	}
}

func TestSpellControllerWithLKIRemovalPaths(t *testing.T) {
	for _, remove := range []string{"pop", "source", "id", "ante"} {
		t.Run(remove, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewInstant("Spell", "")
			card.SetOwner(a.PlayerID())
			obj := newStackObject(a.PlayerID(), card.ID(), card, nil, nil, 0, false)
			g.stack.Push(obj)
			saved := BindSpell("saved")
			g.resolution.spellBindings = map[*SpellBinding]uuid.UUID{saved: obj.ID}
			ref := ControllerOfSpellWithLKI(FromSpell(saved))
			ctx := &EffectContext{Game: g}
			obj.Controller = b.PlayerID()
			switch remove {
			case "pop":
				g.stack.Pop()
			case "source":
				g.stack.RemoveBySourceID(card.ID())
			case "id":
				g.stack.RemoveByID(obj.ID)
			case "ante":
				if _, _, _, err := g.removeCardForAnte(card.ID(), ZoneStack); err != nil {
					t.Fatal(err)
				}
			}
			id, ok := ref.resolvePlayer(ctx)
			if !ok || id != b.PlayerID() {
				t.Fatal("lost departure controller")
			}
			if _, ok := ControllerOfSpell(FromSpell(saved)).resolvePlayer(ctx); ok {
				t.Fatal("live-only query changed")
			}
			g.stack.Push(newStackObject(a.PlayerID(), card.ID(), card, nil, nil, 0, false))
			id, ok = ref.resolvePlayer(ctx)
			if !ok || id != b.PlayerID() {
				t.Fatal("recast replaced departure controller")
			}
			clone := g.Clone()
			clone.stack.spellControllers[obj.ID] = a.PlayerID()
			id, _ = ref.resolvePlayer(ctx)
			if id != b.PlayerID() {
				t.Fatal("clone shared departure map")
			}
		})
	}
}
