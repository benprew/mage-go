package mage

import "testing"

func TestLoseGameFor(t *testing.T) {
	for _, life := range []int{20, 0, -3} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		g.PutOnBattlefield(NewArtifact("Exemption", "", WithStaticAbility(PreventLifeTotalLossStatic(You))), a.PlayerID())
		a.SetLife(life)
		clone := g.Clone()
		effect := LoseGameFor(You)
		if err := effect.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		if a.IsAlive() || !b.IsAlive() || a.Life() != life || !g.IsGameOver() || g.Winner() != b.Name() {
			t.Fatal("explicit loss or life changed")
		}
		if !clone.GetPlayer(a.PlayerID()).IsAlive() {
			t.Fatal("loss changed clone")
		}
		a.SetLife(30)
		if a.IsAlive() {
			t.Fatal("life gain reversed loss")
		}
		if err := effect.Apply(&EffectContext{Game: clone, Controller: b.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		if clone.GetPlayer(b.PlayerID()).IsAlive() || !clone.GetPlayer(a.PlayerID()).IsAlive() {
			t.Fatal("controller or clone state shared")
		}
	}
}

func TestLoseGameForDefinition(t *testing.T) {
	for _, subject := range []PlayerRef{nil, FromPlayer(BindPlayer("unbound"))} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), LoseGameFor(subject))))
		if err := ValidateCardDefinition("Invalid", card); err == nil {
			t.Fatal("invalid player accepted")
		}
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	if err := LoseGameFor(FromPlayer(BindPlayer("absent"))).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if !a.IsAlive() || !b.IsAlive() {
		t.Fatal("absent subject caused loss")
	}
}
