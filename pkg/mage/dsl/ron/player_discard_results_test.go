package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronOptionalDiscardPlayer struct {
	*mage.BasePlayer
	count    int
	requests int
}

func (p *ronOptionalDiscardPlayer) ChooseNumber(minimum, maximum int, _ string) int {
	p.requests++
	return max(minimum, min(maximum, p.count))
}
func TestRONPlayerDiscardResultsMindBomb(t *testing.T) {
	for _, counts := range [][2]int{{0, 0}, {1, 2}, {3, 3}, {9, 1}} {
		t.Run(fmt.Sprint(counts), func(t *testing.T) {
			a := &ronOptionalDiscardPlayer{BasePlayer: mage.NewBasePlayer("A"), count: counts[0]}
			b := &ronOptionalDiscardPlayer{BasePlayer: mage.NewBasePlayer("B"), count: counts[1]}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			for range 3 {
				a.AddToHand(mage.NewInstant("A card", ""))
				b.AddToHand(mage.NewInstant("B card", ""))
			}
			card := repositoryCard(t, "DRK", "Mind Bomb")
			castRONSpell(t, g, a.BasePlayer, card)
			if a.requests != 0 || b.requests != 0 || a.Life() != 20 || b.Life() != 20 {
				t.Fatal("resolution action ran during casting")
			}
			g.ResolveStack()
			wantA, wantB := min(3, counts[0]), min(3, counts[1])
			if a.Life() != 17+wantA || b.Life() != 17+wantB || len(a.Hand()) != 3-wantA || len(b.Hand()) != 3-wantB || a.requests != 1 || b.requests != 1 {
				t.Fatalf("life %d/%d hands %d/%d requests %d/%d", a.Life(), b.Life(), len(a.Hand()), len(b.Hand()), a.requests, b.requests)
			}
		})
	}
}
func TestRONPlayerDiscardResultsInvalid(t *testing.T) {
	producer := `EachPlayerAction(AnyPlayer,DiscardUptoNumberCards(Integer(3)))`
	read := `PlayerAction(You,GainLife(TheNumberOfCardsDiscardedByPlayerThisWay(You)))`
	for _, body := range []string{
		read, read + `,` + producer,
		`EachPlayerAction(Opponent,DiscardUptoNumberCards(Integer(3)))`,
		`EachPlayerAction(AnyPlayer,DiscardUptoNumberCards(Integer(-1)))`,
		`EachPlayerAction(AnyPlayer,DiscardUptoNumberCards)`,
		producer + `,PlayerAction(You,GainLife(TheNumberOfCardsDiscardedByPlayerThisWay(ActionPlayer)))`,
		`FlipACoin_OnLose([` + producer + `]),` + read,
		producer + `,PlayerAction(You,GainLife(TheNumberOfCardsDiscardedByPlayerThisWay(DealsDamage_ThatPlayer)))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([` + body + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid-discard-results.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONPlayerDiscardResultsCopiesAndClones(t *testing.T) {
	a := &ronOptionalDiscardPlayer{BasePlayer: mage.NewBasePlayer("A"), count: 1}
	b := &ronOptionalDiscardPlayer{BasePlayer: mage.NewBasePlayer("B"), count: 2}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	for range 3 {
		a.AddToHand(mage.NewInstant("A card", ""))
		b.AddToHand(mage.NewInstant("B card", ""))
	}
	card := repositoryCard(t, "DRK", "Mind Bomb")
	castRONSpell(t, g, a.BasePlayer, card)
	clone := g.Clone()
	if g.CopySpellOnStack(card.ID(), b.PlayerID(), false) == nil {
		t.Fatal("copy absent")
	}
	g.ResolveStack()
	if a.Life() != 16 || b.Life() != 17 || len(a.Hand()) != 1 || len(b.Hand()) != 0 || a.requests != 2 || b.requests != 2 {
		t.Fatalf("copies life %d/%d hands %d/%d", a.Life(), b.Life(), len(a.Hand()), len(b.Hand()))
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 17 || clone.GetPlayer(b.PlayerID()).Life() != 17 || len(clone.GetPlayer(a.PlayerID()).Hand()) != 3 || len(clone.GetPlayer(b.PlayerID()).Hand()) != 3 {
		t.Fatal("simulation choices or clone isolation")
	}
}
func TestRONPlayerDiscardResultsTimingRollbackAndPrevention(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "DRK", "Mind Bomb")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("cast without mana")
	}
	if len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("failed cast changed hand")
	}
	a.ManaPool().Add(core.Blue, 1)
	g.SetStep(core.DeclareAttackers)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("sorcery cast during combat")
	}
	if len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 1 {
		t.Fatal("timing check spent resources")
	}
	g.SetStep(core.PrecombatMain)
	g.AddPreventionShield(b.PlayerID(), 2)
	g.SetOnDamageDealt(func(_, _ string, _ int, combat bool) {
		if combat || a.Life() != 17 || b.Life() != 19 {
			t.Error("partial damage batch or life loss replaced damage")
		}
	})
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 17 || b.Life() != 19 || len(a.Graveyard()) != 1 {
		t.Fatal("zero hands, prevention or spell departure")
	}
}
func TestRONPlayerDiscardResultsIndependentProducers(t *testing.T) {
	producer := `EachPlayerAction(AnyPlayer,DiscardUptoNumberCards(Integer(1)))`
	read := `PlayerAction(You,GainLife(TheNumberOfCardsDiscardedByPlayerThisWay(You)))`
	factory := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([`+producer+`,`+read+`,EachPlayerAction(AnyPlayer,DiscardUptoNumberCards(Integer(0))),`+read+`]))`)
	a := &ronOptionalDiscardPlayer{BasePlayer: mage.NewBasePlayer("A"), count: 1}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	a.AddToHand(mage.NewInstant("One", ""))
	a.AddToHand(mage.NewInstant("Two", ""))
	castRONSpell(t, g, a.BasePlayer, factory())
	g.ResolveStack()
	if a.Life() != 21 || len(a.Hand()) != 1 || a.requests != 1 {
		t.Fatal("independent producer used previous count")
	}
}

func TestRONPlayerDiscardResultsInventory(t *testing.T) {
	src := `Card(Name:"Results",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([EachPlayerAction(AnyPlayer,DiscardUptoNumberCards(Integer(3))),PlayerAction(You,GainLife(TheNumberOfCardsDiscardedByPlayerThisWay(You)))]))])`
	report, err := InspectFS(fstest.MapFS{"results.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 {
		t.Fatal(report.Issues)
	}
	found := false
	for _, entry := range report.Signatures {
		if entry.Name == "You" && strings.Contains(entry.Context, "TheNumberOfCardsDiscardedByPlayerThisWay") {
			found = true
			if entry.Role != "player_reference" {
				t.Fatalf("wrong result player role %s", entry.Role)
			}
		}
	}
	if !found {
		t.Fatal("result player not inventoried")
	}
}

func TestRONPlayerDiscardResultsActivationSources(t *testing.T) {
	for _, change := range []string{"phased", "departed", "returned"} {
		t.Run(change, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([EachPlayerAction(AnyPlayer,DiscardUptoNumberCards(Integer(1))),PlayerAction(You,GainLife(TheNumberOfCardsDiscardedByPlayerThisWay(You)))]))`)
			a := &ronOptionalDiscardPlayer{BasePlayer: mage.NewBasePlayer("A"), count: 1}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			a.AddToHand(mage.NewInstant("Card", ""))
			card := factory()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if a.requests != 0 || len(a.Hand()) != 1 {
				t.Fatal("activation made resolving choices")
			}
			switch change {
			case "phased":
				g.PhaseOut(source.ID())
			case "departed", "returned":
				g.DestroyPermanent(source)
				if change == "returned" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			}
			g.ResolveStack()
			if a.Life() != 21 || b.Life() != 20 || len(a.Hand()) != 0 {
				t.Fatal("source change replaced saved controller or binding")
			}
		})
	}
}
