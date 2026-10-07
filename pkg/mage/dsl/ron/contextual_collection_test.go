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

func TestRONContextualCollectionObelisk(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "ATQ", "Obelisk of Undoing")
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	own := mage.NewArtifact("Own", "")
	own.SetOwner(a.PlayerID())
	target := g.PutOnBattlefield(own, a.PlayerID())
	foreign := mage.NewArtifact("Foreign", "")
	foreign.SetOwner(b.PlayerID())
	q := g.PutOnBattlefield(foreign, a.PlayerID())
	a.ManaPool().Add(core.Colorless, 6)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err == nil {
		t.Fatal("foreign target accepted")
	}
	if a.ManaPool().TotalMana() != 6 || g.FindPermanent(p.ID()).Tapped {
		t.Fatal("illegal target spent resources")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(target.ID()) == nil || a.ManaPool().TotalMana() != 0 || !g.FindPermanent(p.ID()).Tapped {
		t.Fatal("payment or timing")
	}
	clone := g.Clone()
	g.ResolveStack()
	if g.FindPermanent(target.ID()) != nil || len(a.Hand()) != 1 || clone.FindPermanent(target.ID()) == nil {
		t.Fatal("return or clone")
	}
	clone.ResolveStack()
	if clone.FindPermanent(target.ID()) != nil {
		t.Fatal("clone lost target")
	}
}

func TestRONContextualCollectionRemoveEnchantments(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Remove Enchantments")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	own := mage.NewEnchantment("Own", "")
	own.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(own, a.PlayerID())
	foreign := mage.NewEnchantment("Foreign", "")
	foreign.SetOwner(b.PlayerID())
	q := g.PutOnBattlefield(foreign, a.PlayerID())
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 1, 4), a.PlayerID())
	aura := mage.NewEnchantment("Owned aura", "", mage.WithSubTypes("Aura"))
	aura.SetOwner(a.PlayerID())
	r := g.PutOnBattlefield(aura, b.PlayerID())
	g.Attach(r.ID(), host.ID())
	other := mage.NewEnchantment("Other aura", "", mage.WithSubTypes("Aura"))
	other.SetOwner(b.PlayerID())
	s := g.PutOnBattlefield(other, b.PlayerID())
	g.Attach(s.ID(), host.ID())
	spared := g.PutOnBattlefield(mage.NewEnchantment("Spared", ""), b.PlayerID())
	a.ManaPool().Add(core.White, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(p.ID()) != nil || g.FindPermanent(r.ID()) != nil || len(a.Hand()) != 2 {
		t.Fatal("owned enchantments not returned")
	}
	if g.FindPermanent(q.ID()) != nil || g.FindPermanent(s.ID()) != nil || len(b.Graveyard()) != 2 {
		t.Fatal("remaining enchantments not destroyed")
	}
	if g.FindPermanent(spared.ID()) == nil || g.FindPermanent(host.ID()) == nil {
		t.Fatal("unrelated permanents changed")
	}
}

func TestRONContextualCollectionObeliskLegality(t *testing.T) {
	for _, change := range []string{"control", "phase", "phase-cycle", "return", "shroud", "depart-source", "self", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "ATQ", "Obelisk of Undoing")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			c := mage.NewArtifact("Target", "")
			c.SetOwner(a.PlayerID())
			q := g.PutOnBattlefield(c, a.PlayerID())
			id := q.ID()
			if change == "self" {
				id = p.ID()
			}
			a.ManaPool().Add(core.Colorless, 5)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{id}); err == nil {
				t.Fatal("unpaid activation accepted")
			}
			if a.ManaPool().TotalMana() != 5 || g.FindPermanent(p.ID()).Tapped || g.StackSize() != 0 {
				t.Fatal("unpaid activation changed state")
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{id}); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: q.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phase":
				g.PhaseOut(q.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(q.ID()))
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(q.ID()))
				a.RemoveFromHand(c.ID())
				g.PutOnBattlefield(c, a.PlayerID())
			case "shroud":
				g.MutablePermanent(q.ID()).GrantBaseAttr(core.Shroud)
			case "depart-source":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				g.ResolveTopOfStack()
				if g.FindPermanent(q.ID()) == nil {
					t.Fatal("copy used original controller")
				}
			}
			g.ResolveStack()
			if change == "self" {
				if g.FindPermanent(p.ID()) != nil {
					t.Fatal("self return failed")
				}
				return
			}
			absent := g.FindPermanentIncludingPhased(q.ID()) == nil
			want := change == "phase-cycle" || change == "depart-source" || change == "copy"
			if absent != want {
				t.Fatalf("target absent %v, want %v", absent, want)
			}
		})
	}
}

func TestRONContextualCollectionRemoveLiveMembership(t *testing.T) {
	for _, change := range []string{"attack", "unblocked-return", "control", "reattach", "phase", "phase-cycle", "indestructible", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Remove Enchantments")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 4), b.PlayerID())
			ownHost := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 4), a.PlayerID())
			create := func(name string, owner uuid.UUID) *mage.Permanent {
				c := mage.NewEnchantment(name, "", mage.WithSubTypes("Aura"))
				c.SetOwner(owner)
				p := g.PutOnBattlefield(c, b.PlayerID())
				g.Attach(p.ID(), host.ID())
				return p
			}
			owned := create("Owned", a.PlayerID())
			foreign := create("Foreign", b.PlayerID())
			a.ManaPool().Add(core.White, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "attack", "indestructible", "unblocked-return":
				host.RevokeBaseAttr(core.AttrSummonSick)
				g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{host.ID()})
				if change == "indestructible" {
					g.MutablePermanent(foreign.ID()).GrantBaseAttr(core.Indestructible)
				}
				if change == "unblocked-return" {
					g.GetCombat().RemoveFromCombat(host.ID())
				}
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: ownHost.ID(), TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "reattach":
				g.Attach(owned.ID(), ownHost.ID())
				g.Attach(foreign.ID(), ownHost.ID())
			case "phase":
				g.PhaseOut(host.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			returned := change == "attack" || change == "control" || change == "reattach" || change == "indestructible"
			if (len(a.Hand()) == 1) != returned {
				t.Fatal("wrong owned Aura selection")
			}
			foreignAbsent := change == "attack" || change == "control" || change == "reattach" || change == "copy"
			if (g.FindPermanentIncludingPhased(foreign.ID()) == nil) != foreignAbsent {
				t.Fatal("wrong foreign Aura selection")
			}
			if clone.FindPermanentIncludingPhased(owned.ID()) == nil {
				t.Fatal("clone changed with original")
			}
			clone.ResolveStack()
			if (len(clone.GetPlayer(a.PlayerID()).Hand()) == 1) != returned {
				t.Fatal("clone lost contextual selection")
			}
		})
	}
}

func TestRONContextualCollectionDeclarations(t *testing.T) {
	for _, filter := range []string{
		`OwnedByAPlayer`, `OwnedByAPlayer()`, `OwnedByAPlayer(SinglePlayer(You),AnyPlayer)`, `OwnedByAPlayer(You)`, `OwnedByAPlayer(SinglePlayer(You()))`, `OwnedByAPlayer(SinglePlayer(Ref_TargetPlayer))`,
		`ControlledByAPlayer(Opponent())`, `ControlledByAPlayer(SinglePlayer(Trigger_ThatPlayer))`,
		`AttachedToAPermanent()`, `AttachedToAPermanent(OwnedByAPlayer(SinglePlayer(Ref_TargetPermanent)))`, `And([])`, `Or([])`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([PutEachPermanentIntoItsOwnersHand(` + filter + `)]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", filter)
		}
	}
	for _, filter := range []string{`OwnedByAPlayer(SinglePlayer(You))`, `Or([OwnedByAPlayer(SinglePlayer(You)),IsCardtype(Artifact)])`, `And([OwnedByAPlayer(SinglePlayer(You)),OwnedByAPlayer(SinglePlayer(You))])`} {
		card := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([PutEachPermanentIntoItsOwnersHand(`+filter+`)]))`)()
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	for _, filter := range []string{`OwnedByAPlayer(You)`, `OwnedByAPlayer(SinglePlayer(Ref_TargetPlayer))`, `OwnedByAPlayer(Opponent)`, `Or([OwnedByAPlayer(SinglePlayer(You)),IsCardtype(Artifact)])`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),Targeted([TargetPermanent(` + filter + `)],ActionList([TapPermanent(Ref_TargetPermanent)])))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted target %s", filter)
		}
	}
}

func TestRONContextualCollectionBatches(t *testing.T) {
	for _, destroy := range []bool{false, true} {
		g, a, _ := ronGame()
		p := g.PutOnBattlefield(mage.NewEnchantment("First", ""), a.PlayerID())
		q := g.PutOnBattlefield(mage.NewEnchantment("Second", ""), a.PlayerID())
		watcher := g.PutOnBattlefield(mage.NewLand("Observer", mage.WithAbility(mage.NewStateTriggered(false, mage.Sacrifice(mage.This)).SetCondition(func(_ *core.GameEvent, game mage.GameReader, _, _ uuid.UUID) bool {
			return game.FindPermanent(p.ID()) == nil && game.FindPermanent(q.ID()) != nil
		}))), a.PlayerID())
		action := "PutEachPermanentIntoItsOwnersHand"
		if destroy {
			action = "DestroyEachPermanent"
		}
		card := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([`+action+`(Or([And([IsCardtype(Enchantment),ControlledByAPlayer(SinglePlayer(You))]),And([IsCardtype(Enchantment),OwnedByAPlayer(SinglePlayer(You))])]))]))`)()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if g.FindPermanent(watcher.ID()) == nil {
			t.Fatal("batch exposed intermediate state")
		}
		if g.FindPermanent(p.ID()) != nil || g.FindPermanent(q.ID()) != nil {
			t.Fatal("batch did not move both members")
		}
	}
}

func TestRONContextualCollectionStrictLoad(t *testing.T) {
	files := fstest.MapFS{}
	for _, tc := range []struct{ set, name string }{{"ATQ", "Obelisk of Undoing"}, {"LEG", "Remove Enchantments"}} {
		n := hostUntapRecord(t, tc.set, tc.name)
		factory, err := Compile(n)
		if err != nil {
			t.Fatal(err)
		}
		first, second := factory(), factory()
		if first.Abilities()[0] == second.Abilities()[0] {
			t.Fatal("factories share ability")
		}
		if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile("../../../../cards/mtgish/" + tc.set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "`+tc.name+`"`) {
				files[tc.set+".ron"] = &fstest.MapFile{Data: []byte(line)}
			}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("complete fixture rejected", err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[UnknownRule])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("rejected load changed registry")
	}
}

func TestRONContextualCollectionCastingAndInventory(t *testing.T) {
	g, a, _ := ronGame()
	g.SetStep(core.PrecombatMain)
	card := repositoryCard(t, "ATQ", "Obelisk of Undoing")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 {
		t.Fatal("unpaid permanent cast")
	}
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(card.ID()) == nil || a.ManaPool().TotalMana() != 0 {
		t.Fatal("permanent entry or mana cost")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, []uuid.UUID{card.ID()}); err == nil {
		t.Fatal("six-mana activation cost missing")
	}
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[
 SpellActions(ActionList([
  PutEachPermanentIntoItsOwnersHand(Or([
   OwnedByAPlayer(SinglePlayer(You)),
   AttachedToAPermanent(ControlledByAPlayer(Opponent))
  ]))
 ]))
])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 {
		t.Fatal("inventory rejected supported nested forms", report.Issues)
	}
	expected := map[string]string{"OwnedByAPlayer": "permanent_filter", "ControlledByAPlayer": "permanent_filter", "AttachedToAPermanent": "permanent_filter", "You": "player_reference", "Opponent": "player_filter"}
	for _, signature := range report.Signatures {
		if role, ok := expected[signature.Name]; ok {
			if signature.Role != role {
				t.Fatalf("%s role %s", signature.Name, signature.Role)
			}
			delete(expected, signature.Name)
		}
	}
	if len(expected) != 0 {
		t.Fatal("missing nested roles", expected)
	}
}
