package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCoinRoundSimultaneousAndScope(t *testing.T) {
	g, a, b := newSpellCopyGame()
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	member := BindPlayer("outer")
	g.resolution.playerBindings = map[*PlayerBinding]uuid.UUID{member: b.PlayerID()}
	g.SetCoinFlipResults([]bool{false, false, true, false, true, true, false})
	calls := 0
	g.SetOnDamageDealt(func(_, _ string, amount int, combat bool) {
		calls++
		wantB := 19
		if calls == 3 {
			wantB = 18
		}
		if a.Life() != 19 || b.Life() != wantB || amount != 1 || combat || g.resolution.playerBindings[member] != b.PlayerID() {
			t.Error("partial round batch or changed outer binding")
		}
	})
	effect := RepeatHeadsOrTailsDamage(This, Fixed(1), You, FromPlayer(member), You)
	card := NewInstant("Procedure", "", Spell(Sequence(SnapshotPlayer(You, member), effect)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if err := effect.Apply(&EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if calls != 3 || g.FlipCoin(a.PlayerID()) || g.resolution.playerBindings[member] != b.PlayerID() {
		t.Fatal("duplicates, repeat condition or frame")
	}
}

func TestCoinRoundSourceIdentity(t *testing.T) {
	for _, change := range []string{"depart", "return", "phase", "cycle", "control"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			card := NewCreature("Source", "", 1, 1, WithKeyword(core.Lifelink), WithAbility(Activate(Costs(), RepeatHeadsOrTailsDamage(This, Fixed(1), You, ActivePlayer))))
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.SetActivePlayerIndex(1)
			g.SetCoinFlipResults([]bool{false, false, true, true, false})
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			switch change {
			case "depart", "return":
				g.BouncePermanentToHand(source)
				if change == "return" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phase":
				g.PhaseOut(source.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.ResolveStack()
			clone.ResolveStack()
			wantA, wantB := 21, 19
			if change == "control" {
				wantA, wantB = 19, 21
			}
			if a.Life() != wantA || b.Life() != wantB || g.FlipCoin(a.PlayerID()) {
				t.Fatalf("source %s life %d/%d", change, a.Life(), b.Life())
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 21 || clone.GetPlayer(b.PlayerID()).Life() != 19 || clone.FlipCoin(a.PlayerID()) {
				t.Fatal("clone lost source information or randomness")
			}
		})
	}
}

func TestCoinRoundReplacementOrder(t *testing.T) {
	for choice := range 2 {
		a := NewBasePlayer("A")
		b := &sourceFilterChoicePlayer{BasePlayer: NewBasePlayer("B"), choice: choice}
		g := NewGame(a, b)
		source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
		victim := g.PutOnBattlefield(NewCreature("Redirect", "", 1, 10), b.PlayerID())
		if err := ApplyEffect(g, PreventDamageThisTurn(You, PermanentFilter{}), source.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.SetActivePlayerIndex(1)
		g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: victim.ID()})
		g.SetCoinFlipResults([]bool{true, false, true, true, false})
		if err := RepeatHeadsOrTailsDamage(This, Fixed(1), You, ActivePlayer).Apply(&EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		want := 0
		if choice == 1 {
			want = 1
		}
		if b.Life() != 20 || g.FindPermanent(victim.ID()).Damage != want || b.prompts != 1 || g.FlipCoin(a.PlayerID()) {
			t.Fatal("replacement order or repeat condition")
		}
	}
}

func TestCoinRoundDefinition(t *testing.T) {
	for _, effect := range []ResolvingEffect{
		RepeatHeadsOrTailsDamage(nil, Fixed(1), You),
		RepeatHeadsOrTailsDamage(This, nil, You),
		RepeatHeadsOrTailsDamage(This, Fixed(1)),
		RepeatHeadsOrTailsDamage(This, Fixed(1), nil),
		RepeatHeadsOrTailsDamage(This, Fixed(1), FromPlayer(BindPlayer("unbound"))),
	} {
		if err := ValidateCardDefinition("Invalid", NewInstant("Invalid", "", Spell(effect))); err == nil {
			t.Fatal("invalid coin procedure")
		}
	}
}

func TestCoinRoundLethalAndNoLimit(t *testing.T) {
	g, a, b := newSpellCopyGame()
	card := NewInstant("Procedure", "", Spell(RepeatHeadsOrTailsDamage(This, Fixed(1), You, ActivePlayer)))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.SetLife(1)
	b.SetLife(1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.SetActivePlayerIndex(1)
	coins := make([]bool, 0, 2005)
	for range 1001 {
		coins = append(coins, false, false)
	}
	coins = append(coins, true, true, false)
	g.SetCoinFlipResults(coins)
	g.ResolveStack()
	if a.Life() != -1000 || b.Life() != -1000 || g.StackSize() != 0 || g.FlipCoin(a.PlayerID()) {
		t.Fatal("round cap or a lethal packet stopped resolution")
	}
}

func TestCoinRoundEmptyAndNonpositive(t *testing.T) {
	for _, amount := range []int{-2, 0, 1} {
		g, a, b := newSpellCopyGame()
		source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
		missing := FromPlayer(BindPlayer("absent"))
		g.SetCoinFlipResults([]bool{false, true, false})
		ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID()}
		if err := RepeatHeadsOrTailsDamage(This, Fixed(amount), missing).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if g.FlipCoin(a.PlayerID()) {
			t.Fatal("absent participant consumed randomness")
		}
		g.SetCoinFlipResults([]bool{false, true, false})
		if err := RepeatHeadsOrTailsDamage(This, Fixed(amount), You).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if a.Life() != 20-max(0, amount) || b.Life() != 20 || g.FlipCoin(a.PlayerID()) {
			t.Fatal("nonpositive damage stopped the coin procedure")
		}
	}
}
