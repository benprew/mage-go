package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPlayerDamageAmountsBatchAndScope(t *testing.T) {
	g, a, b := newSpellCopyGame()
	member := BindPlayer("recipient")
	outer := map[*PlayerBinding]uuid.UUID{member: b.PlayerID()}
	g.resolution.playerBindings = outer
	a.AddToHand(NewInstant("A card", ""))
	b.AddToHand(NewInstant("B card", ""))
	b.AddToHand(NewInstant("B second", ""))
	source := g.PutOnBattlefield(NewCreature("Damage", "", 1, 1), a.PlayerID())
	callbacks := 0
	g.SetOnDamageDealt(func(_, _ string, amount int, combat bool) {
		callbacks++
		if combat || a.Life() != 19 || b.Life() != 18 {
			t.Error("partial damage batch")
		}
		if g.resolution.playerBindings[member] != b.PlayerID() {
			t.Error("recipient escaped its amount scope")
		}
	})
	effect := DealDamageEachPlayer(This, EachPlayer, member, HandSizeOf(FromPlayer(member)))
	card := NewInstant("Valid", "", Spell(effect))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if err := effect.Apply(&EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if callbacks != 2 || g.resolution.playerBindings[member] != b.PlayerID() {
		t.Fatal("batch or scope")
	}
}

func TestPlayerDamageAmountsValidation(t *testing.T) {
	member := BindPlayer("recipient")
	good := DealDamageEachPlayer(This, EachPlayer, member, HandSizeOf(FromPlayer(member)))
	for _, tc := range []struct {
		name    string
		effect  ResolvingEffect
		invalid bool
	}{
		{"valid", good, false},
		{"nil source", DealDamageEachPlayer(nil, EachPlayer, member, Fixed(1)), true},
		{"nil players", DealDamageEachPlayer(This, nil, member, Fixed(1)), true},
		{"nil member", DealDamageEachPlayer(This, EachPlayer, nil, Fixed(1)), true},
		{"nil amount", DealDamageEachPlayer(This, EachPlayer, member, nil), true},
		{"empty name", DealDamageEachPlayer(This, EachPlayer, BindPlayer(""), Fixed(1)), true},
		{"wrong binding", DealDamageEachPlayer(This, EachPlayer, member, HandSizeOf(FromPlayer(BindPlayer("missing")))), true},
		{"escaped", Sequence(good, GainLifeFor(FromPlayer(member), Fixed(1))), true},
		{"read in set", DealDamageEachPlayer(This, OpponentsOf(FromPlayer(member)), member, Fixed(1)), true},
	} {
		card := NewInstant(tc.name, "", Spell(tc.effect))
		if err := ValidateCardDefinition(card.Name(), card); (err != nil) != tc.invalid {
			t.Fatalf("%s: %v", tc.name, err)
		}
	}
}

func TestPlayerDamageAmountsSnapshotAndPrevention(t *testing.T) {
	g, a, b := newSpellCopyGame()
	member := BindPlayer("recipient")
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	amount := SubtractValues(Fixed(21), LifeTotalOf(FromPlayer(member)))
	effect := DealDamageEachPlayer(This, EachPlayer, member, amount)
	g.AddPreventionShield(a.PlayerID(), 1)
	events := 0
	g.SetOnDamageDealt(func(_, _ string, _ int, _ bool) {
		events++
		if a.Life() != 20 || b.Life() != 19 {
			t.Error("prevention batch")
		}
	})
	if err := effect.Apply(&EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if events != 1 || len(g.resolution.playerBindings) != 0 {
		t.Fatal("prevented damage or scope")
	}
	for _, value := range []int{0, -3} {
		if err := DealDamageEachPlayer(This, EachPlayer, member, Fixed(value)).Apply(&EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
	}
	if events != 1 {
		t.Fatal("nonpositive damage")
	}
}

type playerDamageTotalLife struct{}

func (playerDamageTotalLife) Text() string { return "combined life" }
func (playerDamageTotalLife) resolveValue(ctx *EffectContext) int {
	total := 0
	for _, id := range EachPlayer.resolvePlayers(ctx) {
		total += ctx.Game.GetPlayer(id).Life()
	}
	return 41 - total
}
func (playerDamageTotalLife) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value"}
}

func TestPlayerDamageAmountsBeforeFirstDamage(t *testing.T) {
	g, a, b := newSpellCopyGame()
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	if err := DealDamageEachPlayer(This, EachPlayer, BindPlayer("member"), playerDamageTotalLife{}).Apply(&EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 19 || b.Life() != 19 {
		t.Fatal("later amount read an earlier damage result")
	}
}

func TestPlayerDamageAmountsSourceIdentity(t *testing.T) {
	for _, change := range []string{"depart", "return", "control", "phase", "cycle"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			member := BindPlayer("recipient")
			sourceCard := NewCreature("Source", "", 1, 1, WithKeyword(core.Lifelink), WithAbility(Activate(Costs(), DealDamageEachPlayer(This, OpponentsOf(You), member, Fixed(2)))))
			sourceCard.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(sourceCard, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			switch change {
			case "depart", "return":
				g.BouncePermanentToHand(source)
				if change == "return" {
					a.RemoveFromHand(sourceCard.ID())
					g.PutOnBattlefield(sourceCard, b.PlayerID())
				}
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phase":
				g.PhaseOut(source.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			}
			g.ResolveStack()
			clone.ResolveStack()
			wantA, wantB := 22, 18
			if change == "control" {
				wantA, wantB = 20, 20
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("source %s life %d/%d", change, a.Life(), b.Life())
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 18 {
				t.Fatal("clone source state")
			}
		})
	}
}

func TestPlayerDamageAmountsReplacementOrder(t *testing.T) {
	for choice := range 2 {
		a := NewBasePlayer("A")
		b := &sourceFilterChoicePlayer{BasePlayer: NewBasePlayer("B"), choice: choice}
		g := NewGame(a, b)
		source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
		victim := g.PutOnBattlefield(NewCreature("Redirect", "", 1, 10), b.PlayerID())
		if err := ApplyEffect(g, PreventDamageThisTurn(You, PermanentFilter{}), source.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: victim.ID()})
		effect := DealDamageEachPlayer(This, OpponentsOf(You), BindPlayer("recipient"), Fixed(2))
		if err := effect.Apply(&EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		want := 0
		if choice == 1 {
			want = 2
		}
		if b.Life() != 20 || g.FindPermanent(victim.ID()).Damage != want || b.prompts != 1 {
			t.Fatalf("choice %d: life %d damage %d prompts %d", choice, b.Life(), g.FindPermanent(victim.ID()).Damage, b.prompts)
		}
	}
}

type playerDamageTestSet struct{ ids []uuid.UUID }

func (playerDamageTestSet) playerSet()                                  {}
func (s playerDamageTestSet) resolvePlayers(*EffectContext) []uuid.UUID { return s.ids }
func (playerDamageTestSet) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-set"}
}

func TestPlayerDamageAmountsEmptyAndDuplicateSet(t *testing.T) {
	for _, empty := range []bool{false, true} {
		g, a, b := newSpellCopyGame()
		source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
		ids := []uuid.UUID{b.PlayerID(), b.PlayerID(), uuid.New()}
		if empty {
			ids = nil
		}
		member := BindPlayer("recipient")
		if err := DealDamageEachPlayer(This, playerDamageTestSet{ids}, member, Fixed(2)).Apply(&EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		want := 18
		if empty {
			want = 20
		}
		if b.Life() != want || a.Life() != 20 || len(g.resolution.playerBindings) != 0 {
			t.Fatal("set membership or binding scope")
		}
	}
}
