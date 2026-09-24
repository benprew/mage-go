package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestManaValueOfPermanent(t *testing.T) {
	for _, tc := range []struct {
		cost string
		want int
	}{{"{3}{W}", 4}, {"{X}{2}", 2}, {"", 0}, {"{W/U}{B/R}", 2}} {
		t.Run(tc.cost, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewArtifact("Subject", tc.cost)
			g.PutOnBattlefield(card, a.PlayerID())
			saved, number := BindPermanent("subject"), BindNumber("mana value")
			ctx := &EffectContext{Game: g, SourceID: card.ID(), Controller: a.PlayerID()}
			if err := Sequence(Snapshot(This, saved), SnapshotNumber(ManaValueOf(From(saved)), number)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			for _, change := range []string{"live", "phase", "phase in", "return"} {
				switch change {
				case "phase":
					g.PhaseOut(card.ID())
				case "phase in":
					g.PhaseIn([]uuid.UUID{card.ID()})
				case "return":
					g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
					g.PutOnBattlefield(card, a.PlayerID())
				}
				want := tc.want
				if change == "phase" || change == "return" {
					want = 0
				}
				if got := ManaValueOf(From(saved)).resolveValue(ctx); got != want {
					t.Fatalf("%s: mana value = %d, want %d", change, got, want)
				}
				if got := FromNumber(number).resolveValue(ctx); got != tc.want {
					t.Fatalf("snapshot = %d, want %d", got, tc.want)
				}
			}
		})
	}
}

func TestManaValueOfValidation(t *testing.T) {
	for _, subject := range []PermanentRef{nil, From(BindPermanent("missing")), DeclarePermanentTarget("undeclared", IsArtifact)} {
		card := NewInstant("Invalid", "", WithAbility(Spell(GainLifeFor(You, ManaValueOf(subject)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid subject accepted")
		}
	}
}

func TestManaValueOfTargetClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first, second := NewArtifact("First", "{1}"), NewArtifact("Second", "{4}")
	first.SetOwner(a.PlayerID())
	second.SetOwner(b.PlayerID())
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	x, y := DeclarePermanentTarget("first", IsArtifact), DeclarePermanentTarget("second", IsArtifact)
	n := BindNumber("value")
	body := Sequence(SnapshotNumber(ManaValueOf(y), n), Destroy(y), GainLifeFor(You, FromNumber(n)))
	ability := Spell(body, WithTargets(x, y))
	card := NewInstant("Read second", "", WithAbility(ability))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.pushStack(&StackObject{ID: card.ID(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID(), second.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 24 || clone.FindPermanent(second.ID()) != nil || clone.FindPermanent(first.ID()) == nil {
		t.Fatal("wrong target or captured amount")
	}
	if a.Life() != 20 || g.FindPermanent(second.ID()) == nil {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	if a.Life() != 24 {
		t.Fatal("original lost binding state")
	}
}
