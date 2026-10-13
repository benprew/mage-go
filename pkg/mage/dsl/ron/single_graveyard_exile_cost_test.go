package ron

import (
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSingleGraveyardExileCostNightSoil(t *testing.T) {
	for _, change := range []string{"own", "foreign", "depart", "control", "phase", "phase-cycle", "return", "copy", "counter"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "FEM", "Night Soil")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Green, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			holder := b
			if change == "own" {
				holder = a
			}
			for range 2 {
				c := mage.NewCreature("Payment", "", 1, 1)
				c.SetOwner(holder.PlayerID())
				holder.AddToGraveyard(c)
			}
			other := mage.NewLand("Retained")
			other.SetOwner(holder.PlayerID())
			holder.AddToGraveyard(other)
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if len(g.GetExile()) != 2 || len(holder.Graveyard()) != 1 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("payment did not finish at announcement")
			}
			if len(g.AllBattlefield()) != 1 {
				t.Fatal("token entered before resolution")
			}
			want := 1
			switch change {
			case "depart":
				g.DestroyPermanent(g.FindPermanent(source.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phase":
				g.PhaseOut(source.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.PutOnBattlefield(card, b.PlayerID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				want = 2
			case "counter":
				g.GetStack().RemoveByID(g.StackPeek().ID)
				want = 0
			}
			clone := g.Clone()
			g.ResolveStack()
			tokens := 0
			controllers := map[uuid.UUID]int{}
			for _, p := range g.AllBattlefield() {
				if !p.IsToken {
					continue
				}
				tokens++
				controllers[p.ControllerID()]++
				if !p.HasSubType("Saproling") || !slices.Contains(p.Colors(), core.Green) || p.CurrentPower(g) != 1 || p.CurrentToughness(g) != 1 {
					t.Fatal("wrong token")
				}
				if change != "copy" && p.ControllerID() != a.PlayerID() {
					t.Fatal("token lost saved controller")
				}
			}
			if change == "copy" && (controllers[a.PlayerID()] != 1 || controllers[b.PlayerID()] != 1) {
				t.Fatal("copy lost its own controller")
			}
			if tokens != want || len(g.GetExile()) != 2 {
				t.Fatal("wrong result or payment repeated")
			}
			for _, p := range clone.AllBattlefield() {
				if p.IsToken {
					t.Fatal("clone changed before resolution")
				}
			}
			clone.ResolveStack()
			count := 0
			for _, p := range clone.AllBattlefield() {
				if p.IsToken {
					count++
				}
			}
			if count != want || len(clone.GetExile()) != 2 {
				t.Fatal("clone lost pending body")
			}
		})
	}
}

func TestRONSingleGraveyardExileCostRollback(t *testing.T) {
	for _, reason := range []string{"split", "wrong-type", "unpaid", "phased"} {
		t.Run(reason, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "FEM", "Night Soil"), a.PlayerID())
			for i := range 2 {
				var c mage.Card = mage.NewCreature("Payment", "", 1, 1)
				holder := a
				if reason == "split" && i == 1 {
					holder = b
				}
				if reason == "wrong-type" && i == 1 {
					c = mage.NewLand("Wrong")
				}
				c.SetOwner(holder.PlayerID())
				holder.AddToGraveyard(c)
			}
			mana := 1
			if reason == "unpaid" {
				mana = 0
			}
			a.ManaPool().Add(core.Colorless, mana)
			if reason == "phased" {
				g.PhaseOut(source.ID())
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
				t.Fatal("invalid activation accepted")
			}
			if a.ManaPool().TotalMana() != mana || len(g.GetExile()) != 0 || len(a.Graveyard())+len(b.Graveyard()) != 2 || g.StackSize() != 0 {
				t.Fatal("failed activation changed resources")
			}
		})
	}
}

func TestRONSingleGraveyardExileCostInvalid(t *testing.T) {
	valid := `Exile([NumberCardsInASingleGraveyard(Integer(2),IsCardtype(Creature),AnyPlayer)],[])`
	for _, cost := range []string{
		strings.Replace(valid, "Integer(2)", "ValueX", 1),
		strings.Replace(valid, "Integer(2)", "Integer(-1)", 1),
		strings.Replace(valid, "AnyPlayer", "AnyPlayer()", 1),
		strings.Replace(valid, "AnyPlayer", "SinglePlayer(You)", 1),
		strings.Replace(valid, "IsCardtype(Creature)", "And([])", 1),
		strings.Replace(valid, "IsCardtype(Creature)", "ControlledByPlayer(ActionPlayer)", 1),
		strings.Replace(valid, "],[])", "],[ExilesFaceDown])", 1),
		`And([` + valid + `,` + valid + `])`,
	} {
		text := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[Activated(` + cost + `,ActionList([GainLife(Integer(1))]))])`
		nodes, err := Parse(strings.NewReader(text), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", cost)
		}
	}
	for _, body := range []string{valid, `MayCost(` + valid + `),If(CostWasPaid,[GainLife(Integer(1))])`, `GainLife(ManaValueOfExiled(TheCardExiledThisWay))`} {
		rule := `SpellActions(ActionList([` + body + `]))`
		if strings.HasPrefix(body, "GainLife") {
			rule = `Activated(` + valid + `,ActionList([` + body + `]))`
		}
		kind := "Sorcery"
		if strings.HasPrefix(body, "GainLife") {
			kind = "Enchantment"
		}
		text := `Card(Name:"Invalid context",Typeline:(Supertypes:[],Cardtypes:[` + kind + `],Subtypes:[]),Rules:[` + rule + `])`
		nodes, err := Parse(strings.NewReader(text), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted context %s", body)
		}
	}
}

func TestRONSingleGraveyardExileCostIndependentActivations(t *testing.T) {
	g, a, b := ronGame()
	first := g.PutOnBattlefield(repositoryCard(t, "FEM", "Night Soil"), a.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "FEM", "Night Soil"), b.PlayerID())
	for _, player := range []*mage.BasePlayer{a, b} {
		for range 4 {
			card := mage.NewCreature("Payment", "", 1, 1)
			card.SetOwner(player.PlayerID())
			player.AddToGraveyard(card)
		}
		player.ManaPool().Add(core.Colorless, 2)
	}
	for range 2 {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		if err := g.ActivateAbilityByIndex(b.PlayerID(), second.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	if g.StackSize() != 4 || len(g.GetExile()) != 8 || len(a.Graveyard())+len(b.Graveyard()) != 0 {
		t.Fatal("activation payments shared choices")
	}
	g.ResolveStack()
	count := map[uuid.UUID]int{}
	for _, p := range g.AllBattlefield() {
		if p.IsToken {
			count[p.ControllerID()]++
		}
	}
	if count[a.PlayerID()] != 2 || count[b.PlayerID()] != 2 {
		t.Fatal("pending bodies shared controllers")
	}
}

func TestRONSingleGraveyardExileCostInventory(t *testing.T) {
	report, err := InspectFS(os.DirFS("../../../../cards/mtgish"), "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, issue := range report.Issues {
		if issue.Card == "Night Soil" {
			t.Fatal(issue)
		}
	}
	want := map[string]string{"NumberCardsInASingleGraveyard": "exile_payment_object", "Integer": "announcement_value", "IsCardtype": "card_filter", "AnyPlayer": "player_collection"}
	seen := map[string]bool{}
	for _, signature := range report.Signatures {
		if !slices.Contains(signature.Cards, "Night Soil") {
			continue
		}
		if role, ok := want[signature.Name]; ok {
			if signature.Role != role {
				t.Fatalf("%s role = %s, want %s", signature.Name, signature.Role, role)
			}
			seen[signature.Name] = true
		}
	}
	if len(seen) != len(want) {
		t.Fatal("missing cost roles", seen)
	}
	registry := mage.NewRegistry()
	registry.Register("Sentinel", func() mage.Card { return mage.NewLand("Sentinel") })
	data, err := os.ReadFile("../../../../cards/mtgish/FEM.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var complete string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Night Soil"`) {
			complete = line
		}
	}
	if complete == "" {
		t.Fatal("record missing")
	}
	collection := fstest.MapFS{"cards.ron": &fstest.MapFile{Data: []byte(complete + "\n" + `Card(Name:"Unsupported",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[UnknownRule])`)}}
	if err := LoadInto(registry, collection, "*.ron"); err == nil {
		t.Fatal("partial collection published")
	}
	if names := registry.RegisteredCardNames(); len(names) != 1 || names[0] != "Sentinel" {
		t.Fatal("rejected collection changed registry")
	}
}
