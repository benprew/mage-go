package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDestructionCountHellfire(t *testing.T) {
	for _, scenario := range []string{"normal", "regeneration", "indestructible", "phased", "empty", "copy"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Hellfire")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			black := g.PutOnBattlefield(mage.NewCreature("Black", "{B}", 2, 10), b.PlayerID())
			survivor := g.PutOnBattlefield(mage.NewCreature("Indestructible", "", 2, 10, mage.WithKeyword(core.Indestructible)), b.PlayerID())
			var victim *mage.Permanent
			if scenario != "empty" {
				victim = g.PutOnBattlefield(mage.NewCreature("White", "{W}", 2, 2), b.PlayerID())
			}
			if scenario == "regeneration" {
				g.AddRegenerationShield(victim.ID())
			}
			if scenario == "indestructible" {
				g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Indestructible)
			}
			if scenario == "phased" {
				g.PhaseOut(victim.ID())
			}
			a.ManaPool().Add(core.Black, 4)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || a.ManaPool().TotalMana() != 4 || len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed payment changed state")
			}
			a.ManaPool().Add(core.Black, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if a.Life() != 20 || g.FindPermanent(black.ID()) == nil {
				t.Fatal("effect ran at announcement")
			}
			if scenario == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			watch, _ := mage.InspectPermanentCollection(mage.MatchingPermanents(mage.IsCreature))
			visits := watch.Select(g)
			wantProof := scenario == "normal" || scenario == "copy"
			for range 2 {
				if proof, known := g.InspectSpellDestruction(g.StackPeek(), visits); !known || proof != wantProof {
					t.Fatalf("complete Hellfire query = %v/%v, want %v/true", proof, known, wantProof)
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			damage := 3
			if scenario == "normal" {
				damage = 4
			}
			if a.Life() != 20-damage {
				t.Fatalf("life %d, want %d", a.Life(), 20-damage)
			}
			wantB := 20
			if scenario == "copy" {
				wantB = 16
			}
			if b.Life() != wantB || g.FindPermanent(black.ID()) == nil || g.FindPermanent(survivor.ID()) == nil {
				t.Fatal("recipient or selection")
			}
			if victim != nil && (g.FindPermanentIncludingPhased(victim.ID()) == nil) != (scenario == "normal" || scenario == "copy") {
				t.Fatal("destruction result")
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("clone changed before resolution")
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != a.Life() || clone.GetPlayer(b.PlayerID()).Life() != b.Life() {
				t.Fatal("clone lost result bindings")
			}
		})
	}
}

func TestRONDestructionCountEruption(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, set, "Volcanic Eruption")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			first := g.PutOnBattlefield(repositoryCard(t, "LEB", "Mountain"), a.PlayerID())
			second := g.PutOnBattlefield(repositoryCard(t, "LEB", "Mountain"), b.PlayerID())
			g.TapPermanent(first)
			survivor := g.PutOnBattlefield(mage.NewCreature("Survivor", "", 2, 10, mage.WithKeyword(core.Shroud)), b.PlayerID())
			a.ManaPool().Add(core.Blue, 4)
			targets := []uuid.UUID{first.ID(), second.ID()}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 2); err == nil || a.ManaPool().TotalMana() != 4 || len(a.Hand()) != 1 {
				t.Fatal("unpaid X changed state")
			}
			a.ManaPool().Add(core.Blue, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{survivor.ID(), second.ID()}, 2); err == nil || a.ManaPool().TotalMana() != 5 {
				t.Fatal("illegal target changed state")
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 2); err != nil {
				t.Fatal(err)
			}
			callbacks := 0
			g.SetOnDamageDealt(func(source, target string, amount int, combat bool) {
				callbacks++
				if source != "Volcanic Eruption" || amount != 2 || combat || a.Life() != 18 || b.Life() != 18 || g.FindPermanent(survivor.ID()).Damage != 2 || g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) != nil {
					t.Error("incomplete damage or destruction batch")
				}
			})
			g.ResolveStack()
			if callbacks != 3 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("damage recipients or payment")
			}
		})
	}
}

func TestRONDestructionCountEruptionLegality(t *testing.T) {
	for _, change := range []string{"regeneration", "indestructible", "phase", "return", "shroud", "all-illegal", "copy", "zero"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEB", "Volcanic Eruption")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			first := g.PutOnBattlefield(repositoryCard(t, "LEB", "Mountain"), b.PlayerID())
			second := g.PutOnBattlefield(repositoryCard(t, "LEB", "Mountain"), b.PlayerID())
			a.ManaPool().Add(core.Blue, 5)
			targets := []uuid.UUID{first.ID(), second.ID()}
			x := 2
			if change == "zero" {
				targets = nil
				x = 0
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, x); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "regeneration":
				g.AddRegenerationShield(second.ID())
			case "indestructible":
				g.MutablePermanent(second.ID()).GrantBaseAttr(core.Indestructible)
			case "phase":
				g.PhaseOut(second.ID())
			case "return":
				g.RemoveFromBattlefield(second)
				g.PutOnBattlefield(second.Card, b.PlayerID())
			case "shroud":
				g.MutablePermanent(second.ID()).GrantBaseAttr(core.Shroud)
			case "all-illegal":
				g.PhaseOut(first.ID())
				g.PhaseOut(second.ID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			amount := 1
			if change == "copy" {
				amount = 2
			}
			if change == "zero" || change == "all-illegal" {
				amount = 0
			}
			if a.Life() != 20-amount || b.Life() != 20-amount {
				t.Fatalf("damage %d/%d, want %d", 20-a.Life(), 20-b.Life(), amount)
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("clone isolation")
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != a.Life() {
				t.Fatal("clone resolution")
			}
		})
	}
}

func TestRONDestructionCountInvalid(t *testing.T) {
	death := `TheNumberOfPermanentsThatDiedThisWay(IsCardtype(Creature))`
	move := `TheNumberOfPermanentsPutIntoAPlayersGraveyardThisWay(IsLandType(Mountain),AnyPlayer)`
	for _, body := range []string{
		`GainLife(` + death + `)`,
		`DestroyEachPermanent(IsCardtype(Creature)),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([GainLife(` + death + `)]))`,
		`DestroyEachPermanent(IsCardtype(Creature)),GainLife(Plus(` + death + `,` + move + `))`,
		`GainLife(` + move + `),DestroyEachPermanent(IsLandType(Mountain))`,
		`DestroyEachPermanent(IsCardtype(Creature)),GainLife(TheNumberOfPermanentsThatDiedThisWay(IsCardtype(Artifact)))`,
		`DestroyEachPermanent(IsLandType(Mountain)),GainLife(TheNumberOfPermanentsPutIntoAPlayersGraveyardThisWay(IsLandType(Forest),AnyPlayer))`,
		`DestroyEachPermanent(IsLandType(Mountain)),GainLife(TheNumberOfPermanentsPutIntoAPlayersGraveyardThisWay(IsLandType(Mountain),You))`,
		`DestroyEachPermanent(IsCardtype(Creature)),GainLife(TheNumberOfPermanentsThatDiedThisWay())`,
		`DestroyEachPermanent(IsCardtype(Creature)),GainLife(Plus(` + death + `))`,
		`DestroyEachPermanent(IsCardtype(Creature)),GainLife(` + death + `),GainLife(` + death + `)`,
	} {
		t.Run(body, func(t *testing.T) {
			src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([%s]))])`, body)
			nodes, err := Parse(strings.NewReader(src), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Compile(nodes[0]); err == nil {
				t.Fatal("invalid result declaration accepted")
			}
		})
	}
}

func TestRONDestructionCountExileAndToken(t *testing.T) {
	for _, name := range []string{"Hellfire", "Volcanic Eruption"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			set := "LEG"
			if name == "Volcanic Eruption" {
				set = "LEB"
			}
			card := repositoryCard(t, set, name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			types := []core.CardType{core.TypeCreature}
			subtypes := []string(nil)
			if name == "Volcanic Eruption" {
				types = []core.CardType{core.TypeLand}
				subtypes = []string{"Mountain"}
			}
			token := g.PutOnBattlefield(mage.NewToken("Token", 2, 2, types, subtypes), b.PlayerID())
			var exiled *mage.Permanent
			if name == "Hellfire" {
				exiled = g.PutOnBattlefield(mage.NewCreature("Exiled", "{W}", 2, 2), b.PlayerID())
			} else {
				exiled = g.PutOnBattlefield(repositoryCard(t, "LEB", "Mountain"), b.PlayerID())
			}
			g.AddExileIfWouldGoToGraveyardThisTurn(exiled.Card.ID(), uuid.Nil)
			targets := []uuid.UUID(nil)
			x := 0
			amount := 4
			color := core.Black
			if name == "Volcanic Eruption" {
				targets = []uuid.UUID{token.ID(), exiled.ID()}
				x = 2
				amount = 1
				color = core.Blue
			}
			a.ManaPool().Add(color, 5)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, x); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if a.Life() != 20-amount || g.FindPermanent(token.ID()) != nil || g.FindPermanent(exiled.ID()) != nil || len(g.GetExile()) != 1 {
				t.Fatal("token or replaced destruction count")
			}
			if name == "Volcanic Eruption" && b.Life() != 19 {
				t.Fatal("player damage")
			}
		})
	}
}

func TestRONDestructionCountIndependentInstructions(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([DestroyEachPermanent(IsCardtype(Creature)),GainLife(TheNumberOfPermanentsThatDiedThisWay(IsCardtype(Creature))),DestroyEachPermanent(IsCardtype(Land)),GainLife(TheNumberOfPermanentsPutIntoAPlayersGraveyardThisWay(IsCardtype(Land),AnyPlayer))]))`)
	first, second := factory(), factory()
	if first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("shared ability instance")
	}
	g, a, b := ronGame()
	s1 := g.PutOnBattlefield(first, a.PlayerID())
	s2 := g.PutOnBattlefield(second, b.PlayerID())
	g.PutOnBattlefield(mage.NewCreature("Creature", "", 2, 2), a.PlayerID())
	g.PutOnBattlefield(mage.NewLand("Land"), a.PlayerID())
	g.PutOnBattlefield(mage.NewLand("Land2"), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), s1.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), s2.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveTopOfStack()
	if b.Life() != 23 || a.Life() != 20 {
		t.Fatal("separate instructions did not retain their counts")
	}
	g.ResolveTopOfStack()
	if a.Life() != 20 || b.Life() != 23 {
		t.Fatal("empty instruction retained another instance result")
	}
}

func TestRONDestructionCountReplacementOrder(t *testing.T) {
	for _, index := range []int{0, 1} {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			a := mage.NewBasePlayer("A")
			b := &counterReplacementChooser{BasePlayer: mage.NewBasePlayer("B"), index: index}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			victim := g.PutOnBattlefield(mage.NewCreature("Victim", "{W}", 2, 2), b.PlayerID())
			g.AddRegenerationShield(victim.ID())
			g.AddExileIfWouldGoToGraveyardThisTurn(victim.Card.ID(), uuid.Nil)
			card := repositoryCard(t, "LEG", "Hellfire")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Black, 5)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if a.Life() != 17 || b.choices == 0 {
				t.Fatal("replacement order or actual deaths")
			}
			if (g.FindPermanent(victim.ID()) == nil) != (index == 1) {
				t.Fatal("affected player choice was ignored")
			}
		})
	}
}

func TestRONDestructionCountEruptionRetargetedCopy(t *testing.T) {
	a := &ronModePlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B")}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	originalTarget := g.PutOnBattlefield(repositoryCard(t, "LEB", "Mountain"), b.PlayerID())
	copyTarget := g.PutOnBattlefield(repositoryCard(t, "LEB", "Mountain"), b.PlayerID())
	card := repositoryCard(t, "LEB", "Volcanic Eruption")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Blue, 4)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{originalTarget.ID()}, 1); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	b.targets = []uuid.UUID{copyTarget.ID()}
	if g.CopyStackObjectDirect(original, b.PlayerID(), true) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveTopOfStack()
	if a.Life() != 19 || b.Life() != 19 || g.FindPermanent(originalTarget.ID()) == nil || g.FindPermanent(copyTarget.ID()) != nil || original.Targets[0] != originalTarget.ID() {
		t.Fatal("copy changed original declaration")
	}
	g.ResolveStack()
	if a.Life() != 18 || b.Life() != 18 || g.FindPermanent(originalTarget.ID()) != nil {
		t.Fatal("original lost result or target")
	}
}

func TestRONDestructionCountNoRegenerationAndFilterGuarantee(t *testing.T) {
	factory := permanentScopeFactory(t, "Sorcery", `SpellActions(ActionList([DestroyEachPermanentNoRegen(And([IsCardtype(Land),IsLandType(Mountain)])),GainLife(Plus(TheNumberOfPermanentsPutIntoAPlayersGraveyardThisWay(IsLandType(Mountain),AnyPlayer),Integer(2)))]))`)
	g, a, b := ronGame()
	mountain := g.PutOnBattlefield(repositoryCard(t, "LEB", "Mountain"), b.PlayerID())
	g.AddRegenerationShield(mountain.ID())
	g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 23 || g.FindPermanent(mountain.ID()) != nil {
		t.Fatal("conjunction guarantee or no-regeneration result")
	}
}

func TestRONDestructionCountInventory(t *testing.T) {
	report, err := InspectFS(os.DirFS("../../../../cards/mtgish"), "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	foundDeath, foundMove := false, false
	for _, signature := range report.Signatures {
		if signature.Name == "TheNumberOfPermanentsThatDiedThisWay" && signature.Role == "value" && signature.CompiledOccurrences == 1 {
			foundDeath = true
		}
		if signature.Name == "TheNumberOfPermanentsPutIntoAPlayersGraveyardThisWay" && signature.Role == "value" && signature.CompiledOccurrences > 0 {
			foundMove = true
		}
	}
	if !foundDeath || !foundMove {
		t.Fatal("compiled destruction result roles missing")
	}
}
