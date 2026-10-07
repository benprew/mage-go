package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTemporaryAbilityPartWater(t *testing.T) {
	for _, change := range []string{"none", "shroud", "return", "phase", "phase-cycle", "all-illegal", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			spell := repositoryCard(t, "LEG", "Part Water")
			if err := mage.ValidateCardDefinition(spell.Name(), spell); err != nil {
				t.Fatal(err)
			}
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			first := mage.NewCreature("First", "", 2, 4)
			first.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(first, a.PlayerID())
			q := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 4), b.PlayerID())
			r := g.PutOnBattlefield(mage.NewCreature("Unselected", "", 2, 4), a.PlayerID())
			ids := []uuid.UUID{p.ID(), q.ID()}
			a.ManaPool().Add(core.Blue, 1)
			a.ManaPool().Add(core.Colorless, 3)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), ids, 2); err == nil || a.ManaPool().TotalMana() != 4 || g.StackSize() != 0 || len(a.Hand()) != 1 {
				t.Fatal("XX payment rollback")
			}
			a.ManaPool().Add(core.Colorless, 1)
			for _, invalid := range [][]uuid.UUID{{p.ID()}, {p.ID(), p.ID()}, {p.ID(), q.ID(), r.ID()}} {
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), invalid, 2); err == nil || a.ManaPool().TotalMana() != 5 || g.StackSize() != 0 {
					t.Fatal("invalid X targets changed payment")
				}
			}
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), ids, 2); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 0 || p.HasKeyword(core.Islandwalk) {
				t.Fatal("payment or early grant")
			}
			switch change {
			case "shroud", "all-illegal":
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
				if change == "all-illegal" {
					g.MutablePermanent(q.ID()).GrantBaseAttr(core.Shroud)
				}
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(first.ID())
				g.PutOnBattlefield(first, a.PlayerID())
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			g.PhaseIn([]uuid.UUID{p.ID()})
			wantP := change == "none" || change == "phase-cycle" || change == "copy"
			wantQ := change != "all-illegal"
			if g.FindPermanent(p.ID()).HasKeyword(core.Islandwalk) != wantP || g.FindPermanent(q.ID()).HasKeyword(core.Islandwalk) != wantQ || g.FindPermanent(r.ID()).HasKeyword(core.Islandwalk) {
				t.Fatal("target legality or membership")
			}
			if clone.FindPermanentIncludingPhased(q.ID()).HasKeyword(core.Islandwalk) {
				t.Fatal("resolution changed pending clone")
			}
			clone.ResolveStack()
			if clone.FindPermanent(q.ID()).HasKeyword(core.Islandwalk) != wantQ {
				t.Fatal("clone lost targets")
			}
			if wantP {
				if mage.HasLandwalkEvasion(g.FindPermanent(p.ID()), b.PlayerID(), g) {
					t.Fatal("islandwalk without Island")
				}
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), b.PlayerID())
				if !mage.HasLandwalkEvasion(g.FindPermanent(p.ID()), b.PlayerID(), g) {
					t.Fatal("islandwalk evasion")
				}
				g.PhaseIn(g.PhaseOut(p.ID()))
				if !g.FindPermanent(p.ID()).HasKeyword(core.Islandwalk) {
					t.Fatal("phasing erased grant")
				}
			}
			g.RunStepWithPriority(core.Cleanup)
			if g.FindPermanent(p.ID()).HasKeyword(core.Islandwalk) || g.FindPermanent(q.ID()).HasKeyword(core.Islandwalk) || clone.FindPermanent(q.ID()).HasKeyword(core.Islandwalk) != wantQ {
				t.Fatal("expiry or clone isolation")
			}
		})
	}
	g, a, _ := ronGame()
	spell := repositoryCard(t, "LEG", "Part Water")
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(a.Graveyard()) != 1 {
		t.Fatal("X zero did not resolve")
	}
}

func TestRONTemporaryAbilityGoblinWizard(t *testing.T) {
	for _, change := range []string{"none", "source-left", "target-return", "target-phase", "phase-cycle", "target-shroud", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "DRK", "Goblin Wizard")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Red, 2)
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			targetCard := repositoryCard(t, "LEB", "Goblin Balloon Brigade")
			targetCard.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			wrong := g.PutOnBattlefield(mage.NewCreature("Not a Goblin", "", 2, 3), b.PlayerID())
			a.ManaPool().Add(core.Red, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{wrong.ID()}); err == nil || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
				t.Fatal("invalid target changed state")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			white := mage.NewCreature("White", "{W}", 1, 1)
			g.PutOnBattlefield(white, a.PlayerID())
			if target.HasProtectionFromInGame(white, g) || a.ManaPool().TotalMana() != 0 {
				t.Fatal("early protection or payment")
			}
			switch change {
			case "source-left":
				g.DestroyPermanent(g.FindPermanent(source.ID()))
			case "target-return":
				g.BouncePermanentToHand(g.FindPermanent(target.ID()))
				b.RemoveFromHand(targetCard.ID())
				g.PutOnBattlefield(targetCard, b.PlayerID())
			case "target-phase":
				g.PhaseOut(target.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "target-shroud":
				g.MutablePermanent(target.ID()).GrantBaseAttr(core.Shroud)
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			g.PhaseIn([]uuid.UUID{target.ID()})
			want := change == "none" || change == "source-left" || change == "phase-cycle" || change == "copy"
			if g.FindPermanent(target.ID()).HasProtectionFromInGame(white, g) != want || clone.FindPermanentIncludingPhased(target.ID()).HasProtectionFromInGame(white, clone) {
				t.Fatal("identity, legality or clone isolation")
			}
			clone.ResolveStack()
			clone.PhaseIn([]uuid.UUID{target.ID()})
			if clone.FindPermanent(target.ID()).HasProtectionFromInGame(white, clone) != want {
				t.Fatal("clone grant")
			}
			g.RunStepWithPriority(core.Cleanup)
			if g.FindPermanent(target.ID()).HasProtectionFromInGame(white, g) || clone.FindPermanent(target.ID()).HasProtectionFromInGame(white, clone) != want {
				t.Fatal("expiry or clone isolation")
			}
		})
	}
}

func TestRONTemporaryAbilityGoblinWizardHandEntry(t *testing.T) {
	for _, accept := range []bool{false, true} {
		a := &ronHandEntryPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: accept}
		b := &ronHandEntryPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
		g := mage.NewGame(a, b)
		g.SetStep(core.PrecombatMain)
		g.SetTurn(1)
		card := repositoryCard(t, "DRK", "Goblin Wizard")
		card.SetOwner(a.PlayerID())
		source := g.PutOnBattlefield(card, a.PlayerID())
		goblin := repositoryCard(t, "LEB", "Goblin Balloon Brigade")
		goblin.SetOwner(a.PlayerID())
		a.AddToHand(goblin)
		other := repositoryCard(t, "LEB", "Llanowar Elves")
		other.SetOwner(a.PlayerID())
		a.AddToHand(other)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil || source.Tapped {
			t.Fatal("summoning sickness rollback")
		}
		g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		if a.choices != 0 || g.FindPermanent(goblin.ID()) != nil || !g.FindPermanent(source.ID()).Tapped {
			t.Fatal("hand choice timing or tap cost")
		}
		g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
		g.ResolveStack()
		if (g.FindPermanent(goblin.ID()) != nil) != accept || g.FindPermanent(other.ID()) != nil || a.choices != map[bool]int{false: 0, true: 1}[accept] {
			t.Fatal("optional filtered entry after source departure")
		}
	}
}

func TestRONTemporaryAbilityInvalid(t *testing.T) {
	for _, ability := range []string{`Protection`, `Protection()`, `Protection(FromColor(Colors([])))`, `Protection(FromColor(Colors([Colorless])))`, `Protection(FromColor(Colors([White()])))`, `Protection(FromColor(Colors([White])),You)`, `Protection(FromCardtype(Creature))`} {
		rule := `Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbility([` + ability + `])],UntilEndOfTurn)]))`
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", ability)
		}
	}
	for _, modification := range []string{`AddAbility`, `AddAbility()`, `AddAbility([])`, `AddAbility([Flying()])`, `AddAbility([Rampage(2)])`, `AddAbility([Flying]),SetColor(SimpleColorList([Red]))`} {
		rule := `SpellActions(Targeted([NumberTargetPermanents(Integer(1),IsCardtype(Creature))],ActionList([CreateEachPermanentLayerEffectUntil(Ref_TargetPermanents,[` + modification + `],UntilEndOfTurn)])))`
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[`+rule+`])`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", modification)
		}
	}
}

func TestRONTemporaryAbilityProtectionRules(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "DRK", "Goblin Wizard")
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	goblin := repositoryCard(t, "LEB", "Goblin Balloon Brigade")
	goblin.SetOwner(b.PlayerID())
	target := g.PutOnBattlefield(goblin, b.PlayerID())
	aura := repositoryCard(t, "LEB", "Holy Strength")
	aura.SetOwner(a.PlayerID())
	a.AddToHand(aura)
	a.ManaPool().Add(core.White, 1)
	if err := g.CastSpellByID(a.PlayerID(), aura.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(aura.ID()) == nil {
		t.Fatal("Aura did not attach")
	}
	a.ManaPool().Add(core.Red, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{target.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.CheckStateBasedActions()
	if g.FindPermanent(aura.ID()) != nil || len(a.Graveyard()) != 1 {
		t.Fatal("protection must remove a white Aura")
	}
	white := g.PutOnBattlefield(mage.NewCreature("White", "{W}", 2, 2), a.PlayerID())
	if target = g.FindPermanent(target.ID()); target.CanBeTargetedBy(white.Card, a.PlayerID(), g) || mage.CanBlock(white, target, g) {
		t.Fatal("protection target or blocking restriction")
	}
	g.DealDamageToPermanent(target, 3, white.ID())
	if g.FindPermanent(target.ID()).Damage != 0 {
		t.Fatal("white damage not prevented")
	}
	g.ChangeObjectColor(white.ID(), core.Blue)
	if !g.FindPermanent(target.ID()).CanBeTargetedBy(white.Card, a.PlayerID(), g) {
		t.Fatal("protection ignored current source color")
	}
	g.DealDamageToPermanent(g.FindPermanent(target.ID()), 1, white.ID())
	if g.FindPermanent(target.ID()).Damage != 1 {
		t.Fatal("other damage prevented")
	}
}

func TestRONTemporaryAbilityFactoriesAndInventory(t *testing.T) {
	files := fstest.MapFS{}
	for _, tc := range []struct{ set, name string }{{"DRK", "Goblin Wizard"}, {"LEG", "Part Water"}} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + tc.set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), tc.set+".ron")
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range nodes {
			if n.Fields["Name"].String != tc.name {
				continue
			}
			files[tc.set+".ron"] = &fstest.MapFile{Data: []byte(strings.Split(string(data), "\n")[n.Position.Line-1])}
			factory, err := Compile(n)
			if err != nil {
				t.Fatal(err)
			}
			first, second := factory(), factory()
			index := 0
			if tc.name == "Goblin Wizard" {
				index = 1
			}
			if first.Abilities()[index].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[index].(*mage.ActionDefinition).Targets()[0] {
				t.Fatal("shared targets")
			}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || len(registry.RegisteredCardNames()) != 2 {
		t.Fatalf("strict real-card load: %v", err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 2 || len(report.Issues) != 0 {
		t.Fatalf("inventory: %v", err)
	}
	for name, role := range map[string]string{"Protection": "keyword", "FromColor": "protection_filter", "Colors": "color_list", "Landwalk": "keyword"} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s role %s", name, role)
		}
	}
}

func TestRONTemporaryAbilityColorsAndChoices(t *testing.T) {
	for _, color := range []string{"White", "Blue", "Black", "Red", "Green"} {
		for _, choice := range []bool{false, true} {
			g, a, b := ronGame()
			grant := `CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbility([Protection(FromColor(Colors([` + color + `]))),Flying])],UntilEndOfTurn)`
			if choice {
				grant = `ChooseAnAbility([Protection(FromColor(Colors([` + color + `]))),Flying]),CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbilityVariable(TheChosenAbility)],UntilEndOfTurn)`
			}
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([`+grant+`]))`)
			card := factory()
			card.SetOwner(a.PlayerID())
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			symbol := map[string]string{"White": "W", "Blue": "U", "Black": "B", "Red": "R", "Green": "G"}[color]
			colored := g.PutOnBattlefield(mage.NewCreature("Color", "{"+symbol+"}", 2, 2), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if !g.FindPermanent(source.ID()).HasProtectionFromInGame(colored.Card, g) || g.FindPermanent(source.ID()).HasKeyword(core.Flying) == choice {
				t.Fatal("color or resolution choice")
			}
		}
	}
}
