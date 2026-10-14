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

func TestRONSpellTargetRestrictionCards(t *testing.T) {
	for _, name := range []string{"Bartel Runeaxe", "Tetsuo Umezawa"} {
		t.Run(name, func(t *testing.T) {
			a := &ronRestrictionPlayer{BasePlayer: mage.NewBasePlayer("A")}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			card := repositoryCard(t, "LEG", name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 {
				t.Fatal("unpaid cast changed state")
			}
			for _, c := range []core.Color{core.Blue, core.Black, core.Red, core.Green} {
				a.ManaPool().Add(c, 10)
			}
			a.ManaPool().Add(core.Colorless, 10)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			aura := repositoryCard(t, "LEB", "Holy Strength")
			aura.SetOwner(a.PlayerID())
			a.AddToHand(aura)
			a.ManaPool().Add(core.White, 1)
			if err := g.CastSpellByID(a.PlayerID(), aura.ID(), []uuid.UUID{source.ID()}, 0); err == nil || len(a.Hand()) != 1 || a.ManaPool().Count(core.White) != 1 {
				t.Fatal("Aura prohibition or payment rollback")
			}
			spell := repositoryCard(t, "LEB", "Giant Growth")
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{source.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(source.ID()).CurrentPower(g) != card.Power()+3 {
				t.Fatal("non-Aura spell was restricted")
			}
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			if name == "Bartel Runeaxe" {
				if !g.FindPermanent(source.ID()).HasKeyword(core.Vigilance) {
					t.Fatal("missing vigilance")
				}
				a.attackers = []uuid.UUID{source.ID()}
				g.RunStepWithPriority(core.DeclareAttackers)
				if !g.IsAttackingInCombat(source.ID()) {
					t.Fatal("vigilance creature did not attack")
				}
				if g.FindPermanent(source.ID()).Tapped {
					t.Fatal("vigilance attack tapped")
				}
			} else {
				target := g.PutOnBattlefield(mage.NewCreature("Victim", "", 2, 4), b.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{target.ID()}); err == nil || g.FindPermanent(source.ID()).Tapped {
					t.Fatal("untapped nonblocking target accepted")
				}
				g.TapPermanent(target)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{target.ID()}); err != nil {
					t.Fatal(err)
				}
				clone := g.Clone()
				g.ResolveStack()
				clone.ResolveStack()
				if g.FindPermanent(target.ID()) != nil || clone.FindPermanent(target.ID()) != nil {
					t.Fatal("destruction failed")
				}
			}
		})
	}
}

func TestRONSpellTargetRestrictionInvalid(t *testing.T) {
	for _, rule := range []string{
		`PermanentRuleEffect(Ref_TargetPermanent,[CantBeTheTargetOfSpells(AnySpell)])`,
		`PermanentRuleEffect(ThisPermanent(),[CantBeTheTargetOfSpells(AnySpell)])`,
		`PermanentRuleEffect(HostPermanent,[CantBeTheTargetOfSpells(AnySpell)])`,
		`PermanentRuleEffect(ThisPermanent,[CantBeTheTargetOfSpells()])`,
		`PermanentRuleEffect(ThisPermanent,[CantBeTheTargetOfSpells(AnySpell())])`,
		`PermanentRuleEffect(ThisPermanent,[CantBeTheTargetOfSpells(And([]))])`,
		`PermanentRuleEffect(ThisPermanent,[CantBeTheTargetOfSpells(IsEnchantmentType(Saga))])`,
		`PermanentRuleEffect(ThisPermanent,[CantBeTheTargetOfSpells(CanTargetOnly(IsCreatureType()))])`,
		`PermanentRuleEffect(ThisPermanent,[CantBeTheTargetOfSpells(AnySpell),CantBeTheTargetOfAbilities(AnyAbility)])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[`+rule+`],CardPT:(Power:2,Toughness:3))`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

type ronRestrictionPlayer struct {
	*mage.BasePlayer
	attackers []uuid.UUID
}

func (p *ronRestrictionPlayer) DeclareAttackers(*mage.Game) []uuid.UUID { return p.attackers }

func TestRONSpellTargetRestrictionHost(t *testing.T) {
	for _, change := range []string{"none", "move", "source-remove", "source-phase", "source-return", "host-return", "host-phase", "host-abilities", "control"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			hostCard := mage.NewCreature("Host", "", 2, 5)
			hostCard.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 5), b.PlayerID())
			nodes, err := Parse(strings.NewReader(`Card(Name:"Rule",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),PermanentRuleEffect(HostPermanent,[CantBeTheTargetOfSpells(AnySpell)])])`), "host.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(source.ID(), host.ID())
			clone := g.Clone()
			switch change {
			case "move":
				g.Attach(source.ID(), other.ID())
			case "source-remove":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			case "source-phase":
				g.PhaseOut(source.ID())
			case "source-return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.PutOnBattlefield(card, a.PlayerID())
			case "host-return":
				g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
				g.PutOnBattlefield(hostCard, b.PlayerID())
			case "host-phase":
				g.PhaseOut(host.ID())
			case "host-abilities":
				g.AddContinuousEffect(mage.RemoveAllAbilities(host.ID(), core.Indefinite))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			spell := repositoryCard(t, "LEB", "Giant Growth")
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(core.Green, 1)
			allowed := change == "move" || change == "source-remove" || change == "source-phase" || change == "source-return" || change == "host-return"
			err = g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{host.ID()}, 0)
			if (err == nil) != allowed {
				t.Fatalf("allowed=%v err=%v", allowed, err)
			}
			cloneHost := clone.FindPermanent(host.ID())
			if cloneHost.CanBeTargetedBy(mage.SpellTargetSource(spell), a.PlayerID(), clone) {
				t.Fatal("clone rule changed")
			}
			if change == "move" && g.FindPermanent(other.ID()).CanBeTargetedBy(mage.SpellTargetSource(spell), a.PlayerID(), g) {
				t.Fatal("rule did not follow attachment")
			}
			if allowed {
				g.ResolveStack()
				if g.FindPermanent(host.ID()).CurrentPower(g) != 5 {
					t.Fatal("spell failed")
				}
			}
		})
	}
}

func TestRONSpellTargetRestrictionFilters(t *testing.T) {
	for _, filter := range []string{`AnySpell`, `IsEnchantmentType(Aura)`, `IsCardtype(Enchantment)`, `IsColor(White)`, `And([IsColor(White),Or([IsEnchantmentType(Aura),IsCardtype(Instant)])])`} {
		source := `Card(Name:"Rule",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[PermanentRuleEffect(ThisPermanent,[CantBeTheTargetOfSpells(` + filter + `)])],CardPT:(Power:2,Toughness:5))`
		nodes, err := Parse(strings.NewReader(source), "filters.ron")
		if err != nil {
			t.Fatal(err)
		}
		factory, err := Compile(nodes[0])
		if err != nil {
			t.Fatal(err)
		}
		g, a, b := ronGame()
		target := g.PutOnBattlefield(factory(), b.PlayerID())
		aura := repositoryCard(t, "LEB", "Holy Strength")
		if target.CanBeTargetedBy(mage.SpellTargetSource(aura), a.PlayerID(), g) {
			t.Fatalf("filter failed: %s", filter)
		}
	}
}

func TestRONSpellTargetRestrictionLoadAndInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "LEG.ron")
	if err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{}
	for _, node := range nodes {
		name := node.Fields["Name"].String
		if name == "Bartel Runeaxe" || name == "Tetsuo Umezawa" {
			files[name+".ron"] = &fstest.MapFile{Data: []byte(strings.Split(string(data), "\n")[node.Position.Line-1])}
		}
	}
	if len(files) != 2 {
		t.Fatal("missing complete records")
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || registry == nil {
		t.Fatalf("strict load: %v", err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Rules:[Unsupported])`)}
	if result, err := LoadFS(files, "*.ron"); err == nil || result != nil {
		t.Fatal("published invalid collection")
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == "IsEnchantmentType" && signature.Role == "spell_filter" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing nested spell filter role")
	}
}
