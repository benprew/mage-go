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

func TestRONBlockedDamagePreventionWall(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Wall of Vapor")
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("unpaid casting did not roll back")
	}
	a.ManaPool().Add(core.Colorless, 3)
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()) != nil {
		t.Fatal("entered before resolution")
	}
	g.ResolveStack()
	wall := g.FindPermanent(card.ID())
	if wall.CurrentPower(g) != 0 || wall.CurrentToughness(g) != 1 || !wall.HasKeyword(core.Defender) {
		t.Fatal("wall metadata")
	}
	wall.RevokeBaseAttr(core.AttrSummonSick)
	if wall.CanDeclareAsAttacker(g) {
		t.Fatal("defender can attack")
	}
	attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 3, 10), b.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 10), b.PlayerID())
	attacker.RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{attacker.ID()})
	g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: wall.ID(), AttackerID: attacker.ID()}})
	clone := g.Clone()
	g.ExecuteCombatDamage()
	if g.FindPermanent(wall.ID()) == nil || g.FindPermanent(wall.ID()).Damage != 0 {
		t.Fatal("combat damage not prevented")
	}
	g.DealDamageToPermanent(g.FindPermanent(wall.ID()), 2, attacker.ID())
	if g.FindPermanent(wall.ID()).Damage != 0 {
		t.Fatal("noncombat damage from blocked creature not prevented")
	}
	g.DealDamageToPermanent(g.FindPermanent(wall.ID()), 1, other.ID())
	if g.FindPermanent(wall.ID()).Damage != 1 {
		t.Fatal("unrelated damage prevented")
	}
	clone.ExecuteCombatDamage()
	if clone.FindPermanent(wall.ID()).Damage != 0 {
		t.Fatal("clone lost prevention")
	}
	clone.GetCombat().RemoveFromCombat(attacker.ID())
	clone.DealDamageToPermanent(clone.FindPermanent(wall.ID()), 1, attacker.ID())
	if clone.FindPermanent(wall.ID()).Damage != 1 {
		t.Fatal("ended pair retained prevention")
	}
}

func TestRONBlockedDamagePreventionInvalid(t *testing.T) {
	for _, filter := range []string{`IsBlockedByDefender`, `IsBlockedByDefender()`, `IsBlockedByDefender(ThisPermanent())`, `IsBlockedByDefender(HostPermanent)`, `IsBlockedByDefender(Ref_TargetPermanent)`, `IsBlockedByDefender(You)`, `IsBlockingAttacker(ThisPermanent)`, `IsBlockedByDefender(ThisPermanent,ThisPermanent)`} {
		rule := `PreventDamage(DamageWouldBeDealtByAPermanentToRecipient(And([IsCardtype(Creature),` + filter + `]),Permanent(ThisPermanent)),[PreventThatDamage])`
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:0,Toughness:1),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", filter)
		}
	}
}

func TestRONBlockedDamagePreventionShadows(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "LEG.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range nodes {
		if node.Fields["Name"].String != "Wall of Shadows" {
			continue
		}
		if _, err := Compile(node); err != nil {
			t.Fatal(err)
		}
		return
	}
	t.Fatal("missing Wall of Shadows")
}

func TestRONBlockedDamagePreventionStrictInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "LEG.ron")
	if err != nil {
		t.Fatal(err)
	}
	var vapor, shadows string
	for _, node := range nodes {
		line := strings.Split(string(data), "\n")[node.Position.Line-1]
		switch node.Fields["Name"].String {
		case "Wall of Vapor":
			vapor = line
		case "Wall of Shadows":
			shadows = line
		}
	}
	files := fstest.MapFS{"vapor.ron": {Data: []byte(vapor)}, "reprint.ron": {Data: []byte(vapor)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("reprint not deduplicated")
	}
	first, err := registry.CreateCard("Wall of Vapor")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Wall of Vapor")
	if err != nil {
		t.Fatal(err)
	}
	if first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("shared static declaration")
	}
	g, a, b := ronGame()
	left := g.PutOnBattlefield(first, a.PlayerID())
	right := g.PutOnBattlefield(second, a.PlayerID())
	attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 10), b.PlayerID())
	attacker.RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{attacker.ID()})
	g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: left.ID(), AttackerID: attacker.ID()}})
	saved := g.CaptureDamageSource(attacker.ID())
	g.DealSimultaneousDamage(mage.PermanentDamageAssignment(left, 1, saved), mage.PermanentDamageAssignment(right, 1, saved))
	if g.FindPermanent(left.ID()).Damage != 0 || g.FindPermanent(right.ID()).Damage != 1 {
		t.Fatal("instances shared protection")
	}
	files["shadows.ron"] = &fstest.MapFile{Data: []byte(shadows)}
	if err := LoadInto(registry, files, "*.ron"); err != nil || !registry.CardRegistered("Wall of Shadows") {
		t.Fatal("complete Shadows record failed", err)
	}
	invalid := strings.Replace(shadows, "Wall of Shadows", "Invalid Shadows", 1)
	invalid = strings.Replace(invalid, "CanTargetOnly(IsCreatureType(Wall))", "CanTargetOnly(IsCreatureType(Wall()))", 1)
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(invalid)}
	if partial, err := LoadFS(files, "*.ron"); err == nil || partial != nil {
		t.Fatal("published incomplete collection")
	}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("failed load replaced registry")
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 3 || len(report.Issues) != 1 || report.Issues[0].Card != "Invalid Shadows" {
		t.Fatal("wrong remaining gap")
	}
	found := false
	for _, entry := range report.Signatures {
		if entry.Name == "IsBlockedByDefender" {
			found = true
			if entry.Role != "permanent_filter" {
				t.Fatal("wrong filter role")
			}
		}
	}
	if !found {
		t.Fatal("missing nested role")
	}
}

func TestRONBlockedDamagePreventionShapeAndContext(t *testing.T) {
	base := `PreventDamage(DamageWouldBeDealtByAPermanentToRecipient(And([IsCardtype(Creature),IsBlockedByDefender(ThisPermanent)]),Permanent(ThisPermanent)),[PreventThatDamage])`
	for _, rule := range []string{
		strings.Replace(base, "IsCardtype(Creature)", "IsCardtype(Artifact)", 1),
		strings.Replace(base, "Permanent(ThisPermanent)", "Permanent(HostPermanent)", 1),
		strings.Replace(base, "Permanent(ThisPermanent)", "Player(You)", 1),
		strings.Replace(base, "[PreventThatDamage]", "[PreventThatDamage()]", 1),
		strings.Replace(base, "[PreventThatDamage]", "[]", 1),
		`Activated(PayMana([]),ActionList([` + base + `]))`,
		strings.Replace(base, "And([IsCardtype(Creature),IsBlockedByDefender(ThisPermanent)])", "IsBlockedByDefender(ThisPermanent)", 1),
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:0,Toughness:1),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
	reversed := strings.Replace(base, "IsCardtype(Creature),IsBlockedByDefender(ThisPermanent)", "IsBlockedByDefender(ThisPermanent),IsCardtype(Creature)", 1)
	nodes, err := Parse(strings.NewReader(`Card(Name:"Valid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:0,Toughness:1),Rules:[`+reversed+`])`), "valid.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Valid", factory()); err != nil {
		t.Fatal(err)
	}
}

func TestRONBlockedDamagePreventionOrdering(t *testing.T) {
	for mode := range 2 {
		a := &ronModePlayer{BasePlayer: mage.NewBasePlayer("A"), mode: mode}
		b := mage.NewBasePlayer("B")
		g := mage.NewGame(a, b)
		wall := g.PutOnBattlefield(repositoryCard(t, "LEG", "Wall of Vapor"), a.PlayerID())
		attacker := g.PutOnBattlefield(repositoryCard(t, "LEB", "Prodigal Sorcerer"), b.PlayerID())
		attacker.RevokeBaseAttr(core.AttrSummonSick)
		g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{attacker.ID()})
		g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: wall.ID(), AttackerID: attacker.ID()}})
		g.UntapPermanent(g.FindPermanent(attacker.ID()))
		if err := g.ActivateAbilityByIndex(b.PlayerID(), attacker.ID(), 0, []uuid.UUID{wall.ID()}); err != nil {
			t.Fatal(err)
		}
		if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
			t.Fatal("copy failed")
		}
		g.SetCreatureDamageRedirect(wall.ID(), a.PlayerID())
		g.ResolveStack()
		want, prompts := 20, 2
		if mode == 0 {
			want, prompts = 19, 1
		}
		if a.Life() != want || a.prompts != prompts || g.FindPermanent(wall.ID()).Damage != 0 {
			t.Fatalf("ordering: life=%d prompts=%d", a.Life(), a.prompts)
		}
	}
}
