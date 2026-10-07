package ron

import (
	"maps"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func controllerEnchantClause(t *testing.T, set, name string) mage.Card {
	t.Helper()
	n := hostUntapRecord(t, set, name)
	n.Fields = maps.Clone(n.Fields)
	rules := n.Fields["Rules"]
	for _, rule := range rules.Args {
		if rule.Name == "EnchantPermanent" {
			rules.Args = []Node{rule}
			n.Fields["Rules"] = rules
			factory, err := Compile(n)
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			return card
		}
	}
	t.Fatal("missing enchant clause")
	return nil
}

func TestRONEnchantControllerClauses(t *testing.T) {
	for _, tc := range []struct {
		set, name, kind string
		opponent        bool
	}{
		{"LEG", "Cocoon", "creature", false},
		{"LEG", "Relic Bind", "artifact", true},
		{"FEM", "Tourach's Gate", "land", false},
	} {
		for _, change := range []string{"none", "host-control", "aura-control", "return", "phase", "clone", "shroud", "copy", "copy-opponent"} {
			t.Run(tc.name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				controller, other := a.PlayerID(), b.PlayerID()
				if tc.opponent {
					controller, other = other, controller
				}
				makeHost := func() mage.Card {
					switch tc.kind {
					case "land":
						return mage.NewLand("Host")
					case "artifact":
						return mage.NewArtifact("Host", "")
					default:
						return mage.NewCreature("Host", "", 2, 2)
					}
				}
				hostCard := makeHost()
				host := g.PutOnBattlefield(hostCard, controller)
				illegal := g.PutOnBattlefield(makeHost(), other)
				card := controllerEnchantClause(t, tc.set, tc.name)
				if card.CastTargets()[0] == controllerEnchantClause(t, tc.set, tc.name).CastTargets()[0] {
					t.Fatal("factory instances share casting targets")
				}
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				a.ManaPool().Add(core.Green, 1)
				a.ManaPool().Add(core.Blue, 1)
				a.ManaPool().Add(core.Black, 2)
				a.ManaPool().Add(core.Colorless, 2)
				mana := a.ManaPool().TotalMana()
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{illegal.ID()}, 0); err == nil || a.ManaPool().TotalMana() != mana || len(a.Hand()) != 1 || g.StackSize() != 0 {
					t.Fatal("illegal controller target changed payment")
				}
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				clone := g.Clone()
				if change == "copy" || change == "copy-opponent" {
					copyController := a.PlayerID()
					if change == "copy-opponent" {
						copyController = b.PlayerID()
					}
					if g.CopySpellOnStack(card.ID(), copyController, false) == nil {
						t.Fatal("Aura copy failed")
					}
				}
				if change == "host-control" {
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: illegal.ID(), TargetID: host.ID(), ControllerID: other, Duration: core.Indefinite})
				}
				if change == "return" {
					g.RemoveFromBattlefield(host)
					g.PutOnBattlefield(hostCard, controller)
				}
				if change == "phase" {
					g.PhaseOut(host.ID())
				}
				if change == "shroud" {
					g.MutablePermanent(host.ID()).GrantBaseAttr(core.Shroud)
				}
				g.ResolveStack()
				failed := change == "host-control" || change == "return" || change == "phase" || change == "shroud"
				if (g.FindPermanent(card.ID()) == nil) != failed {
					t.Fatal("wrong resolution legality")
				}
				clone.ResolveStack()
				if clone.FindPermanent(card.ID()) == nil {
					t.Fatal("pending clone lost legal target")
				}
				if failed {
					return
				}
				if change == "copy" || change == "copy-opponent" {
					count := 0
					for _, p := range g.FilterBattlefield(mage.AnyPermanent) {
						if p.Name() == tc.name {
							count++
							if p.AttachedTo != host.ID() {
								t.Fatal("copy lost enchant restriction")
							}
						}
					}
					want := 2
					if change == "copy-opponent" {
						want = 1
					}
					if count != want {
						t.Fatal("Aura copy did not enter separately")
					}
				}
				if change == "aura-control" {
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: host.ID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					g.CheckStateBasedActions()
					if g.FindPermanent(card.ID()) != nil || len(a.Graveyard()) != 1 {
						t.Fatal("current Aura controller ignored")
					}
				} else {
					g.MutablePermanent(host.ID()).GrantBaseAttr(core.Shroud)
					g.PhaseIn(g.PhaseOut(host.ID()))
					g.CheckStateBasedActions()
					if g.FindPermanent(card.ID()) == nil {
						t.Fatal("shroud or phase cycle ended legal attachment")
					}
				}
			})
		}
	}
}

func TestRONEnchantControllerReporting(t *testing.T) {
	for _, tc := range []struct{ set, name, blocker string }{
		{"LEG", "Cocoon", "unsupported indefinite type modification (AddAbility)"},
		{"LEG", "Relic Bind", "unsupported action list (Modal_ChooseOne)"},
		{"FEM", "Tourach's Gate", "unsupported cost reference (HostPermanent)"},
	} {
		if _, err := Compile(hostUntapRecord(t, tc.set, tc.name)); err == nil || !strings.Contains(err.Error(), tc.blocker) {
			t.Fatalf("%s: full-record blocker changed: %v", tc.name, err)
		}
	}
	valid := `Card(Name:"Supported clause",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(And([ControlledByAPlayer(Opponent),IsCardtype(Artifact)]))])`
	files := fstest.MapFS{"a.ron": {Data: []byte(valid)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("wrong clause registry")
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(valid, "Opponent", "Trigger_ThatPlayer"))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid enchant replaced registry")
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 || len(report.Issues) != 1 {
		t.Fatalf("inventory: %v", err)
	}
	wanted := map[string]string{"ControlledByAPlayer": "permanent_filter", "Opponent": "player_filter", "EnchantPermanent": "rule"}
	for _, signature := range report.Signatures {
		if role, ok := wanted[signature.Name]; ok && signature.Role == role {
			delete(wanted, signature.Name)
		}
	}
	if len(wanted) != 0 {
		t.Fatalf("missing inventory roles: %v", wanted)
	}
	for _, filter := range []string{`IsCardtype(Creature)`, `ControlledByAPlayer(Opponent)`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[EnchantPermanent(`+filter+`)])`), "invalid-context.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("non-Aura enchant accepted")
		}
	}
}

func TestRONEnchantControllerNestedFilters(t *testing.T) {
	for _, filter := range []string{
		`And([ControlledByAPlayer(SinglePlayer(You)),And([IsCardtype(Creature),IsColor(Green)])])`,
		`And([And([IsColor(Green),IsCardtype(Creature)]),ControlledByAPlayer(SinglePlayer(You))])`,
		`And([Or([IsColor(Green),IsColor(Blue)]),IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You))])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Nested",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(`+filter+`)])`), "nested-enchant.ron")
		if err != nil {
			t.Fatal(err)
		}
		factory, err := Compile(nodes[0])
		if err != nil {
			t.Fatal(err)
		}
		g, a, b := ronGame()
		good := g.PutOnBattlefield(mage.NewCreature("Green", "{G}", 2, 2), a.PlayerID())
		g.PutOnBattlefield(mage.NewCreature("Opponent", "{G}", 2, 2), b.PlayerID())
		g.PutOnBattlefield(mage.NewCreature("Red", "{R}", 2, 2), a.PlayerID())
		g.PutOnBattlefield(mage.NewArtifact("Artifact", "{G}"), a.PlayerID())
		possible := factory().CastTargets()[0].Possible(a.PlayerID(), factory(), g)
		if len(possible) != 1 || possible[0] != good.ID() {
			t.Fatal("nested conjunction lost a restriction")
		}
	}
}

func TestRONEnchantControllerInvalid(t *testing.T) {
	for _, filter := range []string{
		`ControlledByAPlayer`, `ControlledByAPlayer()`, `ControlledByAPlayer(You)`,
		`ControlledByAPlayer(SinglePlayer(Trigger_ThatPlayer))`, `ControlledByAPlayer(SinglePlayer(You()))`,
		`ControlledByAPlayer(SinglePlayer(You),Opponent)`, `ControlledByAPlayer(SinglePlayer(Opponent))`,
		`And([ControlledByAPlayer(Opponent),ControlledByAPlayer(SinglePlayer(You))])`,
		`And([ControlledByAPlayer(SinglePlayer(You)),ControlledByAPlayer(SinglePlayer(You))])`,
		`Or([ControlledByAPlayer(Opponent),IsCardtype(Creature)])`, `And([])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(`+filter+`)])`), "invalid-enchant.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", filter)
		}
	}
}
