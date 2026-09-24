package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestComposableMillContextAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	cards := []Card{NewArtifact("top", ""), NewArtifact("middle", ""), NewArtifact("bottom", "")}
	a.SetLibrary(cards)
	creature := NewCreature("Amount", "{2}", 2, 2)
	g.PutOnBattlefield(creature, b.PlayerID())
	body := Mill(You, ValueAtResolution(TargetPermanentManaValue()))
	clone := g.Clone()
	if err := ApplyEffect(clone, body, uuid.Nil, a.PlayerID(), []uuid.UUID{creature.ID()}); err != nil {
		t.Fatal(err)
	}
	ca := clone.GetPlayer(a.PlayerID())
	if len(ca.Library()) != 1 || ca.Library()[0].ID() != cards[2].ID() || len(ca.Graveyard()) != 2 {
		t.Fatal("mill lost the outer value context or library order")
	}
	if len(a.Library()) != 3 || len(a.Graveyard()) != 0 || len(b.Graveyard()) != 0 {
		t.Fatal("mill changed another player or original game")
	}
}

func TestComposableMillAmountsAndModifiers(t *testing.T) {
	for _, amount := range []int{-1, 0, 1, 4} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		a.SetLibrary([]Card{NewArtifact("top", ""), NewArtifact("middle", ""), NewArtifact("bottom", "")})
		source := NewArtifact("Modifier", "")
		g.PutOnBattlefield(source, b.PlayerID())
		g.AddMillReplacement(source.ID(), DoubleMillForOpponents(source.ID(), b.PlayerID()))
		if err := ApplyEffect(g, Mill(You, Fixed(amount)), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		want := min(3, max(0, amount*2))
		if len(a.Graveyard()) != want || len(a.Library()) != 3-want {
			t.Fatalf("amount %d: graveyard=%d library=%d", amount, len(a.Graveyard()), len(a.Library()))
		}
	}
}

func TestComposableMillModifierBeforeClone(t *testing.T) {
	for _, amount := range []int{-1, 0, 1, 4} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		cards := []Card{NewArtifact("top", ""), NewArtifact("middle", ""), NewArtifact("bottom", "")}
		a.SetLibrary(cards)
		source := NewArtifact("Modifier", "")
		g.PutOnBattlefield(source, b.PlayerID())
		g.AddMillReplacement(source.ID(), DoubleMillForOpponents(source.ID(), b.PlayerID()))
		clone := g.Clone()
		if err := ApplyEffect(clone, Mill(You, Fixed(amount)), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		player := clone.GetPlayer(a.PlayerID())
		want := min(3, max(0, amount*2))
		if len(player.Graveyard()) != want || len(player.Library()) != 3-want {
			t.Fatalf("amount %d: graveyard=%d library=%d", amount, len(player.Graveyard()), len(player.Library()))
		}
		for i, card := range player.Graveyard() {
			if card.ID() != cards[i].ID() {
				t.Fatal("clone milled the wrong card")
			}
		}
		if len(a.Library()) != 3 || len(a.Graveyard()) != 0 || len(clone.GetPlayer(b.PlayerID()).Graveyard()) != 0 {
			t.Fatal("mill changed the original game or the other player")
		}
		if player.DrewFromEmpty() {
			t.Fatal("mill counted as a draw")
		}
	}
}

func TestComposableMillAbsenceAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.SetLibrary([]Card{NewArtifact("Card", "")})
	if err := ApplyEffect(g, Mill(ControllerOf(This), Fixed(1)), uuid.New(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Library()) != 1 {
		t.Fatal("absent player fell back to controller")
	}
	for _, body := range []ResolvingEffect{Mill(nil, Fixed(1)), Mill(You, nil), Mill(You, PowerOf(DeclarePermanentTarget("undeclared", IsCreature)))} {
		card := NewSorcery("Invalid", "", WithAbility(Spell(body)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid mill definition accepted")
		}
	}
	info := InspectDefinition(Mill(You, Fixed(2)))
	if len(info.Children) != 2 || info.Properties.Outcome != OutcomeDetriment {
		t.Fatalf("metadata = %+v", info)
	}
}

func TestMillTargetPlayerUsesLibraryTop(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	top, middle, bottom := NewArtifact("top", ""), NewArtifact("middle", ""), NewArtifact("bottom", "")
	a.SetLibrary([]Card{top, middle, bottom})
	if err := ApplyEffect(g, MillTargetPlayer(Fixed(2)), uuid.Nil, b.PlayerID(), []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if len(a.Library()) != 1 || a.Library()[0].ID() != bottom.ID() || len(a.Graveyard()) != 2 {
		t.Fatal("mill did not remove the top two cards")
	}
	if a.Graveyard()[0].ID() != top.ID() || a.Graveyard()[1].ID() != middle.ID() {
		t.Fatal("wrong milled cards")
	}
	if a.DrewFromEmpty() {
		t.Fatal("mill counted as a draw")
	}
}

func TestComposableMillSavedValueInLoop(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewArtifact("Source", "")
	g.PutOnBattlefield(source, a.PlayerID())
	for range 2 {
		g.PutOnBattlefield(NewCreature("Amount", "", 1, 1), a.PlayerID())
	}
	a.SetLibrary([]Card{NewArtifact("top", ""), NewArtifact("middle", ""), NewArtifact("bottom", "")})
	saved := BindPermanent("member")
	body := ForEach(ControlledPermanents(IsCreature), saved, Mill(You, PowerOf(From(saved))))
	card := NewSorcery("Loop", "", WithAbility(Spell(body)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Graveyard()) != 2 || len(a.Library()) != 1 || a.Library()[0].Name() != "bottom" {
		t.Fatal("mill lost the saved value in the loop frame")
	}
}
