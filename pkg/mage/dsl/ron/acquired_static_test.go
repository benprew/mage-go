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

func TestRONAcquiredStaticEarthbind(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			g, a, b := ronGame()
			host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 5, mage.WithKeyword(core.Flying)), b.PlayerID())
			card := repositoryCard(t, set, "Earthbind")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
				t.Fatal("unpaid cast succeeded")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed cast changed game")
			}
			a.ManaPool().Add(core.Red, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(host.ID()).Damage != 2 || g.FindPermanent(host.ID()).HasKeyword(core.Flying) {
				t.Fatal("full RON did not damage and acquire host removal")
			}
		})
	}
}

func TestRONAcquiredStaticTiming(t *testing.T) {
	for _, change := range []string{"normal", "nonflying", "lost-before-resolution", "gained-after-entry", "source-left", "source-return", "host-return", "phase", "control", "reattach", "copy", "prevented", "removed", "later-flying"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			options := []mage.CardOption{}
			if change != "nonflying" && change != "gained-after-entry" {
				options = append(options, mage.WithKeyword(core.Flying))
			}
			hostCard := mage.NewCreature("Host", "", 2, 6, options...)
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			card := repositoryCard(t, "LEA", "Earthbind")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Red, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveTopOfStack()
			donor := g.FindPermanent(card.ID())
			if donor == nil {
				t.Fatal("Aura failed to enter")
			}
			switch change {
			case "lost-before-resolution":
				if err := mage.ApplyEffect(g, mage.Until(core.Indefinite, mage.RemoveKeyword(mage.This, core.Flying)), host.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "gained-after-entry":
				if err := mage.ApplyEffect(g, mage.Until(core.Indefinite, mage.GrantKeywordModification(mage.This, core.Flying)), host.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "source-left", "source-return":
				g.BouncePermanentToHand(donor)
				if change == "source-return" {
					a.RemoveFromHand(card.ID())
					returned := g.PutOnBattlefield(card, a.PlayerID())
					g.Attach(returned.ID(), host.ID())
				}
			case "host-return":
				g.RemoveFromBattlefield(host)
				g.PutOnBattlefield(hostCard, b.PlayerID())
			case "phase":
				g.PhaseOut(donor.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: donor.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "reattach":
				other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 6, mage.WithKeyword(core.Flying)), b.PlayerID())
				g.Attach(donor.ID(), other.ID())
				host = other
			case "prevented":
				g.AddPreventionShield(host.ID(), 2)
			}
			g.PutTriggersOnStack()
			if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			damage := 2
			if change == "nonflying" || change == "gained-after-entry" || change == "lost-before-resolution" || change == "host-return" || change == "prevented" {
				damage = 0
			}
			if change == "copy" {
				damage = 2
			}
			flying := change == "gained-after-entry" || change == "source-left" || change == "source-return" || change == "host-return" || change == "phase"
			if g.FindPermanent(host.ID()).Damage != damage || g.FindPermanent(host.ID()).HasKeyword(core.Flying) != flying {
				t.Fatalf("damage=%d flying=%v want %d/%v", g.FindPermanent(host.ID()).Damage, g.FindPermanent(host.ID()).HasKeyword(core.Flying), damage, flying)
			}
			if clone.FindPermanent(host.ID()).Damage != damage || clone.FindPermanent(host.ID()).HasKeyword(core.Flying) != flying {
				t.Fatal("clone lost pending rule facts")
			}
			if change == "removed" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(donor.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
				if !g.FindPermanent(host.ID()).HasKeyword(core.Flying) || clone.FindPermanent(host.ID()).HasKeyword(core.Flying) {
					t.Fatal("removal or clone isolation failed")
				}
			}
			if change == "later-flying" {
				if err := mage.ApplyEffect(g, mage.Until(core.Indefinite, mage.GrantKeywordModification(mage.This, core.Flying)), host.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if !g.FindPermanent(host.ID()).HasKeyword(core.Flying) {
					t.Fatal("later grant lost timestamp order")
				}
				other := g.PutOnBattlefield(mage.NewCreature("Other host", "", 2, 6), b.PlayerID())
				g.Attach(donor.ID(), other.ID())
				g.Attach(donor.ID(), host.ID())
				if g.FindPermanent(host.ID()).HasKeyword(core.Flying) {
					t.Fatal("reattachment did not update static timestamp")
				}
			}
		})
	}
}

func TestRONAcquiredStaticInvalid(t *testing.T) {
	valid := `CreatePermanentLayerEffect(ThisPermanent,[AddAbility([PermanentLayerEffect(HostPermanent,[LosesAbility(Flying)])])])`
	for _, bad := range []string{
		`CreatePermanentLayerEffect(HostPermanent,[AddAbility([PermanentLayerEffect(HostPermanent,[LosesAbility(Flying)])])])`,
		`CreatePermanentLayerEffect(ThisPermanent,[AddAbility([PermanentLayerEffect(ThisPermanent,[LosesAbility(Flying)])])])`,
		`CreatePermanentLayerEffect(ThisPermanent,[AddAbility([PermanentLayerEffect(HostPermanent,[])])])`,
		`CreatePermanentLayerEffect(ThisPermanent,[AddAbility([PermanentLayerEffect(HostPermanent,[LosesAbility(Flying())])])])`,
		`CreatePermanentLayerEffect(ThisPermanent,[AddAbility([PermanentLayerEffect(Ref_TargetPermanent,[LosesAbility(Flying)])])])`,
		`CreatePermanentLayerEffect(ThisPermanent,[AddAbility([PermanentLayerEffect(HostPermanent,[AdjustPT(1,1)])])])`,
		`CreatePermanentLayerEffect(ThisPermanent,[AddAbility([PermanentLayerEffect(HostPermanent,[LosesAbility(Flying)]),Flying])])`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),Activated(PayMana([]),ActionList([` + bad + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
	src := `Card(Name:"No host",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([` + valid + `]))])`
	nodes, err := Parse(strings.NewReader(src), "invalid.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(nodes[0]); err == nil {
		t.Fatal("host grant without Aura accepted")
	}
}

func TestRONAcquiredStaticNativeLoadAndCopy(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEA.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Earthbind"`) {
			record = line
		}
	}
	if record == "" {
		t.Fatal("missing Earthbind")
	}
	registry := mage.NewRegistry()
	files := fstest.MapFS{"earthbind.ron": {Data: []byte(record)}}
	if err := LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 6, mage.WithKeyword(core.Flying)), b.PlayerID())
	card, err := registry.CreateCard("Earthbind")
	if err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("spell copy failed")
	}
	clone := g.Clone()
	g.ResolveStack()
	clone.ResolveStack()
	for _, game := range []*mage.Game{g, clone} {
		p := game.FindPermanent(host.ID())
		if p.Damage != 2 || p.HasKeyword(core.Flying) || len(p.Attachments) != 2 {
			t.Fatal("physical and copied Aura entry lost condition, attachment or grant")
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(record, "LosesAbility(Flying)", "LosesAbility(Flying())", 1))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection changed registry")
	}
	report, err := InspectFS(fstest.MapFS{"earthbind.ron": {Data: []byte(record)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatalf("inventory failed: %v", err)
	}
	for _, occurrence := range report.Signatures {
		if occurrence.Name == "HostPermanent" && occurrence.Role != "permanent_reference" {
			t.Fatal("acquired host lost typed inventory role")
		}
		if occurrence.Name == "LosesAbility" && occurrence.Role != "static_modification" {
			t.Fatal("acquired removal lost static inventory role")
		}
	}
}
