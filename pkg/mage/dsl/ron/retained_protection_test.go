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

func TestRONRetainedProtectionGuardian(t *testing.T) {
	g, a, b := ronGame()
	host := g.PutOnBattlefield(mage.NewArtifact("Subject", ""), a.PlayerID())
	old := g.PutOnBattlefield(mage.NewAura("Old Aura", "", mage.WithCastTarget(mage.TargetArtifact())), b.PlayerID())
	g.Attach(old.ID(), host.ID())
	card := repositoryCard(t, "ARN", "Guardian Beast")
	castRONSpell(t, g, a, card)
	g.ResolveStack()
	if !g.FindPermanent(host.ID()).HasKeyword(core.Indestructible) {
		t.Fatal("artifact did not gain indestructible")
	}
	g.CheckStateBasedActions()
	if g.FindPermanent(old.ID()) == nil || g.FindPermanent(old.ID()).AttachedTo != host.ID() {
		t.Fatal("existing Aura was removed")
	}
	newAura := g.PutOnBattlefield(mage.NewAura("New Aura", "", mage.WithCastTarget(mage.TargetArtifact())), b.PlayerID())
	g.Attach(newAura.ID(), host.ID())
	if g.FindPermanent(newAura.ID()).IsAttached() {
		t.Fatal("new Aura attached")
	}
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: newAura.ID(), TargetID: host.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	if g.FindPermanent(host.ID()).ControllerID() != a.PlayerID() || !g.FindPermanent(host.ID()).HasKeyword(core.Indestructible) {
		t.Fatal("blocked control gain changed control or lost the later keyword layer")
	}
	g.MutablePermanent(card.ID()).Tapped = true
	g.ApplyContinuousEffects()
	if g.FindPermanent(host.ID()).ControllerID() != b.PlayerID() || g.FindPermanent(host.ID()).HasKeyword(core.Indestructible) {
		t.Fatal("tapped source retained protection")
	}
	g.MutablePermanent(card.ID()).Tapped = false
	g.ApplyContinuousEffects()
	if g.FindPermanent(host.ID()).ControllerID() != b.PlayerID() {
		t.Fatal("untapping undid established control")
	}
}

func TestRONRetainedProtectionDeclarations(t *testing.T) {
	valid := `Card(Name:"Rule",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[If(PermanentPassesFilter(ThisPermanent,IsUntapped),[EachPermanentRuleEffect(IsCardtype(Artifact),[CantBeEnchantedAndDoesntRemove(IsEnchantmentType(Aura)),CantBeGainedControlOf])])])`
	if _, err := Compile(nameScopeNode(t, valid)); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{
		strings.Replace(valid, "IsEnchantmentType(Aura)", "IsEnchantmentType(Curse)", 1),
		strings.Replace(valid, "IsEnchantmentType(Aura)", "IsEnchantmentType(Aura, Aura)", 1),
		strings.Replace(valid, "CantBeGainedControlOf", "CantBeGainedControlOf()", 1),
		strings.Replace(valid, "CantBeGainedControlOf", "CantAttack", 1),
		strings.Replace(valid, "IsCardtype(Artifact)", "Ref_TargetPermanent", 1),
		strings.Replace(valid, "CantBeGainedControlOf", "CantBeGainedControlOf,CantBeGainedControlOf", 1),
	} {
		if _, err := Compile(nameScopeNode(t, bad)); err == nil {
			t.Fatalf("accepted invalid declaration: %s", bad)
		}
	}
}

func TestRONRetainedProtectionCasting(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "ARN", "Guardian Beast")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 {
		t.Fatal("unpaid spell changed zones")
	}
	a.ManaPool().Add(core.Black, 1)
	a.ManaPool().Add(core.Colorless, 3)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	host := g.PutOnBattlefield(mage.NewArtifact("Subject", ""), a.PlayerID())
	aura := mage.NewAura("Enchant Artifact", "", mage.WithCastTarget(mage.TargetArtifact()))
	aura.SetOwner(b.PlayerID())
	b.AddToHand(aura)
	if err := g.CastSpellByID(b.PlayerID(), aura.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
		t.Fatal("Aura spell targeted a protected artifact")
	}
	if len(b.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("illegal Aura cast changed zones")
	}
}

func TestRONRetainedProtectionScopeAndRemoval(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "ARN", "Guardian Beast")
	source := g.PutOnBattlefield(card, a.PlayerID())
	artifact := mage.NewArtifact("Artifact", "")
	artifact.SetOwner(a.PlayerID())
	host := g.PutOnBattlefield(artifact, a.PlayerID())
	creature := g.PutOnBattlefield(mage.NewCreature("Artifact Creature", "", 2, 3, mage.WithCardType(core.TypeArtifact)), a.PlayerID())
	opponent := g.PutOnBattlefield(mage.NewArtifact("Opponent Artifact", ""), b.PlayerID())
	if !g.FindPermanent(host.ID()).HasKeyword(core.Indestructible) || creature.HasKeyword(core.Indestructible) || opponent.HasKeyword(core.Indestructible) {
		t.Fatal("incorrect live membership")
	}
	g.DestroyPermanent(g.FindPermanent(host.ID()))
	if g.FindPermanent(host.ID()) == nil {
		t.Fatal("destroyed an indestructible artifact")
	}
	g.DoSacrifice(g.FindPermanent(host.ID()))
	if g.FindPermanent(host.ID()) != nil {
		t.Fatal("prevented sacrifice")
	}
	later := mage.NewArtifact("Later Artifact", "")
	later.SetOwner(a.PlayerID())
	newHost := g.PutOnBattlefield(later, a.PlayerID())
	if !newHost.HasKeyword(core.Indestructible) {
		t.Fatal("later artifact was excluded")
	}
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(newHost.ID()).HasKeyword(core.Indestructible) {
		t.Fatal("removed source ability still applied")
	}
	g.DestroyPermanent(g.FindPermanent(newHost.ID()))
	if g.FindPermanent(newHost.ID()) != nil {
		t.Fatal("destruction remained prohibited")
	}
}

func TestRONRetainedProtectionLoadAndInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/ARN.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var source string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Guardian Beast"`) {
			source = line
			break
		}
	}
	if source == "" {
		t.Fatal("missing record")
	}
	files := fstest.MapFS{"valid.ron": {Data: []byte(source)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Guardian Beast")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Guardian Beast")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID() == second.ID() {
		t.Fatal("factory reused an instance")
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(source, "IsEnchantmentType(Aura)", "IsEnchantmentType(Curse)", 1))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || !registry.CardRegistered("Guardian Beast") {
		t.Fatal("non-atomic load")
	}
	delete(files, "invalid.ron")
	inventory, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"CantBeEnchantedAndDoesntRemove": "permanent_rule_operation", "CantBeGainedControlOf": "permanent_rule_operation", "IsEnchantmentType": "aura_restriction_filter", "Aura": "subtype"} {
		found := false
		for _, sig := range inventory.Signatures {
			if sig.Name == name && sig.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing role %s for %s", role, name)
		}
	}
}
