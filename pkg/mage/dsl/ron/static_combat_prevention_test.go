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

func TestRONStaticCombatPreventionCards(t *testing.T) {
	for _, name := range []string{"Demonic Torment", "Gaseous Form", "Enchanted Being"} {
		for _, enchanted := range []bool{false, true} {
			t.Run(name+map[bool]string{false: "/plain", true: "/enchanted"}[enchanted], func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "LEG", name)
				if err := mage.ValidateCardDefinition(name, card); err != nil {
					t.Fatal(err)
				}
				host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 10), a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				if name == "Enchanted Being" {
					host = source
				} else {
					g.Attach(source.ID(), host.ID())
				}
				attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 1, 10), b.PlayerID())
				if enchanted {
					attachRONTestAura(t, g, attacker, b.PlayerID())
				}
				host = g.FindPermanent(host.ID())
				attacker = g.FindPermanent(attacker.ID())
				g.GetCombat().AddAttacker(attacker.ID(), a.PlayerID())
				g.GetCombat().AddBlocker(host.ID(), attacker.ID())
				g.ExecuteCombatDamage()
				wantIncoming := attacker.CurrentPower(g)
				wantOutgoing := host.CurrentPower(g)
				if name == "Gaseous Form" || name == "Enchanted Being" && enchanted {
					wantIncoming = 0
				}
				if name != "Enchanted Being" {
					wantOutgoing = 0
				}
				if g.FindPermanent(host.ID()).Damage != wantIncoming || g.FindPermanent(attacker.ID()).Damage != wantOutgoing {
					t.Fatal("wrong combat packet prevented")
				}
				before := g.FindPermanent(host.ID()).Damage
				g.DealDamageToPermanent(g.FindPermanent(host.ID()), 1, attacker.ID())
				if g.FindPermanent(host.ID()).Damage != before+1 {
					t.Fatal("noncombat damage prevented")
				}
				host = g.FindPermanent(host.ID())
				host.RevokeBaseAttr(core.AttrSummonSick)
				if host.CanDeclareAsAttacker(g) == (name == "Demonic Torment") {
					t.Fatal("attack restriction")
				}
			})
		}
	}
}

func TestRONStaticCombatPreventionCasting(t *testing.T) {
	for _, name := range []string{"Demonic Torment", "Gaseous Form", "Enchanted Being"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 10), b.PlayerID())
			targets := []uuid.UUID{host.ID()}
			if name == "Enchanted Being" {
				targets = nil
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err == nil {
				t.Fatal("unpaid cast accepted")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed casting changed state")
			}
			a.ManaPool().Add(core.Colorless, 2)
			switch name {
			case "Demonic Torment":
				a.ManaPool().Add(core.Black, 1)
			case "Gaseous Form":
				a.ManaPool().Add(core.Blue, 1)
			default:
				a.ManaPool().Add(core.White, 2)
			}
			if name != "Enchanted Being" {
				land := g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{land.ID()}, 0); err == nil {
					t.Fatal("illegal aura target accepted")
				}
				if a.ManaPool().TotalMana() != 3 || len(a.Hand()) != 1 || g.StackSize() != 0 {
					t.Fatal("illegal target spent costs")
				}
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()) != nil || g.StackSize() != 1 {
				t.Fatal("permanent entered before resolution")
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				p := game.FindPermanent(card.ID())
				if p == nil {
					t.Fatal("RON permanent did not resolve")
				}
				if name != "Enchanted Being" && p.AttachedTo != host.ID() {
					t.Fatal("aura resolved onto wrong host")
				}
			}
		})
	}
}

func TestRONStaticCombatPreventionLifecycle(t *testing.T) {
	for _, name := range []string{"Demonic Torment", "Gaseous Form", "Enchanted Being"} {
		for _, change := range []string{"none", "phase", "phase-cycle", "leave", "return", "reattach", "remove-source", "remove-host", "control", "clone"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "LEG", name)
				host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 10), a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				if name == "Enchanted Being" {
					host = source
				} else {
					g.Attach(source.ID(), host.ID())
				}
				attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 1, 10), b.PlayerID())
				attachRONTestAura(t, g, attacker, b.PlayerID())
				active := true
				switch change {
				case "phase":
					g.PhaseOut(source.ID())
					active = false
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(source.ID()))
				case "leave":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					active = false
				case "return":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					source = g.PutOnBattlefield(card, a.PlayerID())
					if name == "Enchanted Being" {
						host = source
					} else {
						active = false
					}
				case "reattach":
					if name != "Enchanted Being" {
						other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 10), a.PlayerID())
						g.Attach(source.ID(), other.ID())
						active = false
					}
				case "remove-source":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), source.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					active = false
				case "remove-host":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), host.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					if name == "Enchanted Being" {
						active = false
					}
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: attacker.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "clone":
					clone := g.Clone()
					clone.RemoveFromBattlefield(clone.FindPermanent(source.ID()))
					if g.FindPermanent(source.ID()) == nil {
						t.Fatal("clone changed live source")
					}
				}
				if g.FindPermanent(host.ID()) == nil {
					if name != "Enchanted Being" || (change != "phase" && change != "leave") {
						t.Fatal("unexpected absent subject")
					}
					return
				}
				host = g.FindPermanent(host.ID())
				attacker = g.FindPermanent(attacker.ID())
				g.GetCombat().AddAttacker(attacker.ID(), a.PlayerID())
				g.GetCombat().AddBlocker(host.ID(), attacker.ID())
				g.ExecuteCombatDamage()
				incoming := attacker.CurrentPower(g)
				outgoing := host.CurrentPower(g)
				if active && name != "Demonic Torment" {
					incoming = 0
				}
				if active && name != "Enchanted Being" {
					outgoing = 0
				}
				if g.FindPermanent(host.ID()) == nil {
					if incoming < host.CurrentToughness(g) {
						t.Fatal("unexpected death")
					}
					return
				}
				if g.FindPermanent(host.ID()).Damage != incoming || g.FindPermanent(attacker.ID()).Damage != outgoing {
					t.Fatalf("wrong lifecycle prevention: incoming=%d outgoing=%d", g.FindPermanent(host.ID()).Damage, g.FindPermanent(attacker.ID()).Damage)
				}
				if name == "Demonic Torment" {
					p := g.FindPermanent(host.ID())
					p.RevokeBaseAttr(core.AttrSummonSick)
					if p.CanDeclareAsAttacker(g) == active {
						t.Fatal("attack rule lifecycle")
					}
				}
			})
		}
	}
}

func TestRONStaticCombatPreventionInvalid(t *testing.T) {
	for _, rule := range []string{
		`PreventDamage(CombatDamageWouldBeDealtByCreature(HostPermanent),[PreventThatDamage])`,
		`PreventDamage(CombatDamageWouldBeDealtByCreature(ThisPermanent()),[PreventThatDamage])`,
		`PreventDamage(CombatDamageWouldBeDealtByCreature(Ref_TargetPermanent),[PreventThatDamage])`,
		`PreventDamage(CombatDamageWouldBeDealtByCreature(ThisPermanent),[PreventThatDamage()])`,
		`PreventDamage(CombatDamageWouldBeDealtByCreature(ThisPermanent),[])`,
		`PreventDamage(CombatDamageWouldBeDealtByCreature(ThisPermanent,ThisPermanent),[PreventThatDamage])`,
		`PreventDamage(CombatDamageWouldBeDealtToRecipient(Player(You)),[PreventThatDamage])`,
		`PreventDamage(CombatDamageWouldBeDealtByACreatureToRecipient(IsCardtype(),Permanent(ThisPermanent)),[PreventThatDamage])`,
		`PreventDamage(Or([CombatDamageWouldBeDealtToRecipient(Permanent(ThisPermanent)),CombatDamageWouldBeDealtByCreature(HostPermanent)]),[PreventThatDamage])`,
		`PreventDamage(CombatDamageWouldBeDealtByACreatureToRecipient(IsBlocking,Permanent(ThisPermanent)),[PreventThatDamage])`,
		`PreventDamage(CombatDamageWouldBeDealtByACreatureToRecipient(And([]),Permanent(ThisPermanent)),[PreventThatDamage])`,
		`PreventDamage(CombatDamageWouldBeDealtByACreatureToRecipient(IsEnchanted(),Permanent(ThisPermanent)),[PreventThatDamage])`,
		`PermanentRuleEffect(ThisPermanent,[CantAttack()])`,
		`PermanentRuleEffect(Ref_TargetPermanent,[CantAttack])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONStaticCombatPreventionOrdering(t *testing.T) {
	for mode := range 2 {
		a := mage.NewBasePlayer("A")
		b := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B"), mode: mode}
		g := mage.NewGame(a, b)
		victim := g.PutOnBattlefield(repositoryCard(t, "LEG", "Enchanted Being"), b.PlayerID())
		attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 1, 10), a.PlayerID())
		attachRONTestAura(t, g, attacker, a.PlayerID())
		g.SetCreatureDamageRedirect(victim.ID(), b.PlayerID())
		g.GetCombat().AddAttacker(attacker.ID(), b.PlayerID())
		g.GetCombat().AddBlocker(victim.ID(), attacker.ID())
		g.ExecuteCombatDamage()
		want := 20
		if mode == 0 {
			want = 18
		}
		if b.Life() != want || b.prompts != 1 || g.FindPermanent(victim.ID()).Damage != 0 {
			t.Fatalf("ordering life=%d prompts=%d", b.Life(), b.prompts)
		}
	}
}

func TestRONStaticCombatPreventionStrictInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	selected := map[string]bool{"Demonic Torment": true, "Gaseous Form": true, "Enchanted Being": true}
	files := fstest.MapFS{}
	for _, node := range nodes {
		if selected[node.Fields["Name"].String] {
			line := strings.Split(string(data), "\n")[node.Position.Line-1]
			files[node.Fields["Name"].String+".ron"] = &fstest.MapFile{Data: []byte(line)}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 3 {
		t.Fatal("strict complete records")
	}
	first, err := registry.CreateCard("Gaseous Form")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Gaseous Form")
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	left := g.PutOnBattlefield(mage.NewCreature("Left", "", 2, 10), a.PlayerID())
	right := g.PutOnBattlefield(mage.NewCreature("Right", "", 2, 10), a.PlayerID())
	firstAura := g.PutOnBattlefield(first, a.PlayerID())
	secondAura := g.PutOnBattlefield(second, a.PlayerID())
	g.Attach(firstAura.ID(), left.ID())
	g.Attach(secondAura.ID(), right.ID())
	clone := g.Clone()
	g.RemoveFromBattlefield(g.FindPermanent(firstAura.ID()))
	g.GetCombat().AddAttacker(left.ID(), b.PlayerID())
	g.GetCombat().AddAttacker(right.ID(), b.PlayerID())
	g.ExecuteCombatDamage()
	if b.Life() != 18 {
		t.Fatal("factories shared source or recipient")
	}
	clone.GetCombat().AddAttacker(left.ID(), b.PlayerID())
	clone.GetCombat().AddAttacker(right.ID(), b.PlayerID())
	clone.ExecuteCombatDamage()
	if clone.GetPlayer(b.PlayerID()).Life() != 20 {
		t.Fatal("clone lost active rules")
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Invalid",Rules:[Unknown])`)}
	if registry, err = LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("partial registry published")
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range report.Signatures {
		if entry.Name == "IsEnchanted" {
			found = true
			if entry.Role != "permanent_filter" {
				t.Fatal("nested filter role")
			}
		}
	}
	if !found {
		t.Fatal("nested filter missing")
	}
}

func TestRONStaticCombatPreventionIllegalAuraResolution(t *testing.T) {
	for _, name := range []string{"Demonic Torment", "Gaseous Form"} {
		for _, change := range []string{"return", "phase", "shroud"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "LEG", name)
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				hostCard := mage.NewCreature("Host", "", 2, 10)
				host := g.PutOnBattlefield(hostCard, b.PlayerID())
				a.ManaPool().Add(core.Colorless, 2)
				color := core.Black
				if name == "Gaseous Form" {
					color = core.Blue
				}
				a.ManaPool().Add(color, 1)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				switch change {
				case "return":
					g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
					g.PutOnBattlefield(hostCard, b.PlayerID())
				case "phase":
					g.PhaseOut(host.ID())
				case "shroud":
					g.MutablePermanent(host.ID()).GrantBaseAttr(core.Shroud)
				}
				g.ResolveStack()
				if g.FindPermanent(card.ID()) != nil || len(a.Graveyard()) != 1 {
					t.Fatal("illegal aura resolved")
				}
			})
		}
	}
}

func TestRONStaticCombatPreventionSubjectShapes(t *testing.T) {
	for _, event := range []string{
		`CombatDamageWouldBeDealtByCreature(ThisPermanent)`,
		`CombatDamageWouldBeDealtToRecipient(Permanent(ThisPermanent))`,
		`Or([CombatDamageWouldBeDealtByCreature(ThisPermanent),CombatDamageWouldBeDealtToRecipient(Permanent(ThisPermanent))])`,
	} {
		t.Run(event, func(t *testing.T) {
			nodes, err := Parse(strings.NewReader(`Card(Name:"Subject",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:10),Rules:[PreventDamage(`+event+`,[PreventThatDamage])])`), "subject.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			subject := g.PutOnBattlefield(factory(), a.PlayerID())
			attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 1, 10), b.PlayerID())
			g.GetCombat().AddAttacker(attacker.ID(), a.PlayerID())
			g.GetCombat().AddBlocker(subject.ID(), attacker.ID())
			g.ExecuteCombatDamage()
			incoming, outgoing := 0, 0
			if strings.HasPrefix(event, "CombatDamageWouldBeDealtByCreature") {
				incoming = 1
			}
			if strings.HasPrefix(event, "CombatDamageWouldBeDealtToRecipient") {
				outgoing = 2
			}
			if g.FindPermanent(subject.ID()).Damage != incoming || g.FindPermanent(attacker.ID()).Damage != outgoing {
				t.Fatal("subject direction or reversed union")
			}
		})
	}
}
