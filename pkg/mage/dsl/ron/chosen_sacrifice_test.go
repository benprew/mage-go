package ron

import (
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONChosenSacrificeRecords(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			for _, state := range []string{"unpaid", "paid", "wrong-mana", "no-lands", "phased", "control", "left", "copy", "clone", "activation"} {
				t.Run(state, func(t *testing.T) {
					a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: state == "paid" || state == "wrong-mana"}
					b := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("B")}
					g := mage.NewGame(a, b)
					g.SetTurn(1)
					c := repositoryCard(t, set, "Demonic Hordes")
					source := g.PutOnBattlefield(c, a.PlayerID())
					source.RevokeBaseAttr(core.AttrSummonSick)
					first := g.PutOnBattlefield(mage.NewLand("First"), a.PlayerID())
					card := mage.NewLand("Chosen")
					card.SetOwner(b.PlayerID())
					chosen := g.PutOnBattlefield(card, a.PlayerID())
					b.choice = chosen.ID()
					foreign := g.PutOnBattlefield(mage.NewLand("Foreign"), b.PlayerID())
					if state == "activation" {
						if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{foreign.ID()}); err != nil {
							t.Fatal(err)
						}
						g.ResolveStack()
						if g.FindPermanent(foreign.ID()) != nil || !g.FindPermanent(source.ID()).Tapped {
							t.Fatal("land destruction failed")
						}
						return
					}
					if state == "paid" {
						a.ManaPool().Add(core.Black, 3)
					}
					if state == "wrong-mana" {
						a.ManaPool().Add(core.Blue, 3)
					}
					g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
					g.PutTriggersOnStack()
					if g.StackSize() != 0 {
						t.Fatal("opponent upkeep triggered")
					}
					g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
					g.PutTriggersOnStack()
					if g.StackSize() != 1 || a.offers != 0 || g.FindPermanent(source.ID()).Tapped || g.FindPermanent(chosen.ID()) == nil {
						t.Fatal("resolution timing")
					}
					switch state {
					case "no-lands":
						g.PhaseOut(first.ID())
						g.PhaseOut(chosen.ID())
					case "phased":
						g.PhaseOut(chosen.ID())
					case "control":
						g.AddControlEffect(mage.ControlEffectSpec{SourceID: foreign.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					case "left":
						g.BouncePermanentToHand(source)
					case "copy":
						if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
							t.Fatal("copy")
						}
					}
					clone := g.Clone()
					if state == "clone" {
						clone.PhaseOut(chosen.ID())
					}
					g.ResolveStack()
					sacrificedChosen := state != "paid" && state != "no-lands" && state != "phased"
					if (g.FindPermanentIncludingPhased(chosen.ID()) == nil) != sacrificedChosen {
						t.Fatal("opponent choice or captured controller")
					}
					if (g.FindPermanent(foreign.ID()) == nil) != (state == "copy") {
						t.Fatal("copied trigger used the wrong sacrificing player")
					}
					if state == "phased" && g.FindPermanent(first.ID()) != nil {
						t.Fatal("phased candidate selected")
					}
					if state == "paid" && (g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 0) {
						t.Fatal("paid branch")
					}
					if state != "paid" && state != "left" && !g.FindPermanent(source.ID()).Tapped {
						t.Fatal("unpaid tap")
					}
					if state == "wrong-mana" && a.ManaPool().TotalMana() != 3 {
						t.Fatal("failed payment changed mana")
					}
					if clone.FindPermanentIncludingPhased(chosen.ID()) == nil {
						t.Fatal("original changed clone")
					}
					clone.ResolveStack()
					if state == "clone" && (clone.FindPermanentIncludingPhased(chosen.ID()) == nil || clone.FindPermanent(first.ID()) != nil) {
						t.Fatal("clone selection")
					}
				})
			}
		})
	}
}

func TestRONChosenSacrificeDeclarations(t *testing.T) {
	for _, action := range []string{"SacrificeAPermanent", "SacrificeAPermanent()", "SacrificeAPermanent(And([]))", "SacrificeAPermanent(ThisPermanent)", "SacrificeAPermanentOfAPlayersChoice(IsCardtype(Land))", "SacrificeAPermanentOfAPlayersChoice(IsCardtype(Land),Opponent())", "SacrificeAPermanentOfAPlayersChoice(IsCardtype(Land),Trigger_ThatPlayer)", "SacrificeAPermanentOfAPlayersChoice(IsCardtype(Land),AnyPlayer)"} {
		t.Run(action, func(t *testing.T) {
			n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+action+`]))])`)
			if _, err := Compile(n); err == nil {
				t.Fatal("invalid accepted")
			}
		})
	}
	n := staticBaseRecord(t, "ARN", "Serendib Djinn")
	if _, err := Compile(n); err != nil {
		t.Fatal(err)
	}
}

func TestRONChosenSacrificeSelfChoice(t *testing.T) {
	n := staticBaseRecord(t, "ARN", "Serendib Djinn")
	rule := n.Fields["Rules"].Args[1]
	rule.Args[1].Args[0].Args = rule.Args[1].Args[0].Args[:1]
	factory := staticBaseSubrule(t, n, rule)
	g, a, b := ronGame()
	g.PutOnBattlefield(factory(), a.PlayerID())
	victim := g.PutOnBattlefield(mage.NewLand("Own land"), a.PlayerID())
	foreign := g.PutOnBattlefield(mage.NewLand("Foreign"), b.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.FindPermanent(victim.ID()) != nil || g.FindPermanent(foreign.ID()) == nil {
		t.Fatal("self choice scope")
	}
}

func TestRONChosenSacrificePlayerScope(t *testing.T) {
	for _, action := range []string{
		`PlayerAction(ControllerOfPermanent(Ref_TargetPermanent),SacrificeAPermanent(IsCardtype(Land)))`,
		`PlayerActions(ControllerOfPermanent(Ref_TargetPermanent),[SacrificeAPermanentOfAPlayersChoice(IsCardtype(Land),You)])`,
	} {
		t.Run(action, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([`+action+`])))`)
			g, a, b := ronGame()
			source := factory()
			source.SetOwner(a.PlayerID())
			a.AddToHand(source)
			target := g.PutOnBattlefield(mage.NewCreature("Target", "", 1, 1), b.PlayerID())
			victim := g.PutOnBattlefield(mage.NewLand("Victim"), b.PlayerID())
			own := g.PutOnBattlefield(mage.NewLand("Own"), a.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), source.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(victim.ID()) != nil || g.FindPermanent(own.ID()) == nil {
				t.Fatal("scoped sacrifice used effect controller")
			}
		})
	}
}

func TestRONChosenSacrificeSpell(t *testing.T) {
	g, a, b := ronGame()
	factory := permanentScopeFactory(t, "Sorcery", `SpellActions(ActionList([SacrificeAPermanent(IsCardtype(Creature)),GainLife(Integer(2))]))`)
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 2, 2, mage.WithKeyword(core.Shroud), mage.WithKeyword(core.Indestructible)), a.PlayerID())
	foreign := g.PutOnBattlefield(mage.NewCreature("Foreign", "", 2, 2), b.PlayerID())
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(victim.ID()) == nil {
		t.Fatal("sacrificed at announcement")
	}
	g.ResolveStack()
	if g.FindPermanent(victim.ID()) != nil || g.FindPermanent(foreign.ID()) == nil || a.Life() != 22 {
		t.Fatal("resolving sacrifice or continuation")
	}
}

func TestRONChosenSacrificeInventory(t *testing.T) {
	text := `Card(Name:"Good",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([SacrificeAPermanentOfAPlayersChoice(IsCardtype(Land),Opponent)]))])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(text)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory", err)
	}
	found := false
	for _, sig := range report.Signatures {
		if sig.Name == "Opponent" && sig.Role == "player_reference" {
			found = true
		}
	}
	if !found {
		t.Fatal("chooser inventory role")
	}
	for _, rule := range []string{`Activated(SacrificeAPermanentOfAPlayersChoice(IsCardtype(Land),Opponent),ActionList([]))`, `AsPermanentEnters([SacrificeAPermanent(IsCardtype(Land))])`, `EachPermanentLayerEffect(AnyPermanent,[SacrificeAPermanent(IsCardtype(Land))])`} {
		text := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + rule + `])`
		if registry, err := LoadFS(fstest.MapFS{"a.ron": {Data: []byte(text)}}, "*.ron"); err == nil || registry != nil {
			t.Fatal("wrong role published registry")
		}
	}
}
