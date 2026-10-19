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

func TestRONTargetDefinedXMirror(t *testing.T) {
	for _, state := range []string{"normal", "short", "other-target", "changed-target", "gone", "phase", "leave", "return", "control", "copy", "invalid-choice"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			chooser := &ronChainPlayer{ronPaymentPlayer: &ronPaymentPlayer{BasePlayer: a}, target: b.PlayerID()}
			if state == "invalid-choice" {
				chooser.target = uuid.New()
			}
			g.SetPlayerAt(0, chooser)
			c := repositoryCard(t, "DRK", "Reflecting Mirror")
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			a.ManaPool().Add(core.Colorless, 4)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			p := g.FindPermanent(c.ID())
			bolt := repositoryCard(t, "LEB", "Lightning Bolt")
			bolt.SetOwner(b.PlayerID())
			b.AddToHand(bolt)
			b.ManaPool().Add(core.Red, 1)
			target := a.PlayerID()
			if state == "other-target" {
				target = b.PlayerID()
			}
			if err := g.CastSpellByID(b.PlayerID(), bolt.ID(), []uuid.UUID{target}, 0); err != nil {
				t.Fatal(err)
			}
			spell := g.StackPeek()
			amount := 2
			if state == "short" {
				amount = 1
			}
			a.ManaPool().Add(core.Colorless, amount)
			g.SetXValue(71)
			err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{spell.ID})
			if state == "short" {
				if err == nil || g.FindPermanent(p.ID()).Tapped || a.ManaPool().TotalMana() != 1 || g.StackSize() != 1 || g.XValue() != 71 {
					t.Fatal("failed payment changed state", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if g.StackPeek().XValue != 2 || a.ManaPool().TotalMana() != 0 || !g.FindPermanent(p.ID()).Tapped || g.XValue() != 71 {
				t.Fatal("target-derived announcement")
			}
			switch state {
			case "changed-target":
				spell.Targets = []uuid.UUID{b.PlayerID()}
			case "gone":
				g.GetStack().RemoveByID(spell.ID)
			case "phase":
				g.PhaseOut(p.ID())
			case "leave", "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				if state == "return" {
					a.RemoveFromHand(c.ID())
					g.PutOnBattlefield(c, b.PlayerID())
				}
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if cp := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false); cp == nil || cp.XValue != 2 {
					t.Fatal("copied X")
				}
				g.ResolveTopOfStack()
			}
			clone := g.Clone()
			g.ResolveTopOfStack()
			if state != "gone" && clone.GetStack().FindByID(spell.ID).Targets[0] != target && state != "changed-target" && state != "copy" {
				t.Fatal("clone shared targets")
			}
			clone.ResolveTopOfStack()
			for _, game := range []*mage.Game{g, clone} {
				game.ResolveStack()
				hitA := state == "invalid-choice" && game == g
				if state == "gone" {
					if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 20 {
						t.Fatal("absent spell")
					}
					continue
				}
				if (game.GetPlayer(a.PlayerID()).Life() == 17) != hitA || (game.GetPlayer(b.PlayerID()).Life() == 17) == hitA {
					t.Fatal("wrong recipient", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
				}
			}
		})
	}
}

func TestRONTargetDefinedXDeclarations(t *testing.T) {
	record := staticBaseRecord(t, "DRK", "Reflecting Mirror")
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	if err = mage.ValidateCardDefinition("Reflecting Mirror", factory()); err != nil {
		t.Fatal(err)
	}
	for _, rules := range []string{
		`ActivatedWithModifiers(PayManaX([ManaCostX],ValueX),ActionList([GainLife(Integer(1))]),XIs(Twice(ManaValueOfSpell(Ref_TargetSpell))))`,
		`ActivatedWithModifiers(PayManaX([ManaCostX],ValueX),Targeted([TargetSpell(AnySpell)],ActionList([GainLife(Integer(1))])),XIs(Twice(ManaValueOfSpell(Ref_TargetPermanent))))`,
		`ActivatedWithModifiers(PayManaX([ManaCostX],ValueX),Targeted([TargetSpell(AnySpell)],ActionList([GainLife(Integer(1))])),XIs(Twice(ManaValueOfSpell(Ref_TargetSpell()))))`,
		`Activated(PayMana([]),ActionList([ChangeTheTargetsOfSpellToAPlayer(Ref_TargetSpell,AnyPlayer)]))`,
		`Activated(PayMana([]),Targeted([TargetSpell(AnySpell)],ActionList([ChangeTheTargetsOfSpellToAPlayer(Ref_TargetSpell,Opponent)])))`,
		`Activated(PayMana([]),Targeted([TargetSpell(AnySpell)],ActionList([If(SpellPassesFilter(Ref_TargetSpell,TargetsPlayer(You())),[GainLife(Integer(1))])])))`,
	} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + rules + `])`
		if reg, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(src)}}, "*.ron"); err == nil || reg != nil {
			t.Fatal("invalid declaration", rules)
		}
	}
	source := `Card(Name:"Good",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[ActivatedWithModifiers(PayManaX([ManaCostX],ValueX),Targeted([TargetSpell(TargetsOnlyASingleTarget)],ActionList([If(SpellPassesFilter(Ref_TargetSpell,TargetsPlayer(You)),[ChangeTheTargetsOfSpellToAPlayer(Ref_TargetSpell,AnyPlayer)])])),XIs(Twice(ManaValueOfSpell(Ref_TargetSpell))))])`
	nodes, err := Parse(strings.NewReader(source), "valid.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Compile(nodes[0]); err != nil {
		t.Fatal(err)
	}
}

func TestRONTargetDefinedXLoadAndInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Reflecting Mirror"`) {
			record = line
		}
	}
	if record == "" {
		t.Fatal("missing record")
	}
	files := fstest.MapFS{"mirror.ron": {Data: []byte(record)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Reflecting Mirror")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Reflecting Mirror")
	if err != nil {
		t.Fatal(err)
	}
	one := mage.UnwrapAbility(first.Abilities()[0]).(mage.ActivatedAbility)
	two := mage.UnwrapAbility(second.Abilities()[0]).(mage.ActivatedAbility)
	if first.ID() == second.ID() || one.Targets()[0] == two.Targets()[0] {
		t.Fatal("shared factory identity")
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(record, "TargetSpell(TargetsOnlyASingleTarget)", "TargetSpell(And([TargetsOnlyASingleTarget,ManaValueIs(EqualTo(ValueX))]))", 1))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || !registry.CardRegistered("Reflecting Mirror") {
		t.Fatal("cyclic X accepted or registry changed")
	}
	delete(files, "bad.ron")
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory", err)
	}
	for name, role := range map[string]string{"Twice": "activation_x_value", "ManaValueOfSpell": "activation_x_value", "Ref_TargetSpell": "spell_reference", "SpellPassesFilter": "condition", "TargetsPlayer": "resolution_spell_filter", "You": "player_reference", "AnyPlayer": "player_filter"} {
		found := false
		for _, sig := range report.Signatures {
			if sig.Name == name && sig.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatal("missing role", name, role)
		}
	}
}
