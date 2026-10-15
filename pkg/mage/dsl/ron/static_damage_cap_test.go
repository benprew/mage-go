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

const ronStaticDamageCap = `ReplaceWouldDealDamage(AnAmountOfDamageWouldBeDealtByASourceToRecipient(GreaterThanOrEqualTo(Integer(3)),Or([IsCardtype(Instant),IsCardtype(Sorcery)]),Player(You)),[DealDamageInstead(Integer(2))])`

func TestRONStaticDamageCapAmulet(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Forethought Amulet")
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast succeeded")
	}
	a.ManaPool().Add(core.Colorless, 5)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(card.ID()) == nil || a.ManaPool().TotalMana() != 0 {
		t.Fatal("paid cast did not enter")
	}
	g.SetActivePlayerIndex(1)
	for _, kind := range []string{"Instant", "Sorcery"} {
		for _, amount := range []string{"0", "1", "2", "3", "7"} {
			factory := permanentScopeFactory(t, kind, `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([SpellDealsDamage(ThisSpell,Integer(`+amount+`),Player(Ref_TargetPlayer))])))`)
			spell := factory()
			spell.SetOwner(b.PlayerID())
			b.AddToHand(spell)
			if err := g.CastSpellByID(b.PlayerID(), spell.ID(), []uuid.UUID{a.PlayerID()}, 0); err != nil {
				t.Fatal(err)
			}
			before := a.Life()
			g.ResolveStack()
			want := map[string]int{"0": 0, "1": 1, "2": 2, "3": 2, "7": 2}[amount]
			if before-a.Life() != want {
				t.Fatalf("%s damage %s: lost %d, want %d", kind, amount, before-a.Life(), want)
			}
		}
	}
}

func TestRONStaticDamageCapLifetime(t *testing.T) {
	for _, change := range []string{"live", "depart", "return", "phase", "phase-cycle", "remove-ability", "remove-ability-expiry", "cleanup", "control", "clone", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Forethought Amulet")
			amulet := g.PutOnBattlefield(card, a.PlayerID())
			spell := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([SpellDealsDamage(ThisSpell,Integer(6),Player(Ref_TargetPlayer))])))`)()
			spell.SetOwner(b.PlayerID())
			b.AddToHand(spell)
			if err := g.CastSpellByID(b.PlayerID(), spell.ID(), []uuid.UUID{a.PlayerID()}, 0); err != nil {
				t.Fatal(err)
			}
			want := 18
			switch change {
			case "depart", "return":
				g.RemoveFromBattlefield(g.FindPermanent(amulet.ID()))
				want = 14
				if change == "return" {
					g.PutOnBattlefield(card, a.PlayerID())
					want = 18
				}
			case "phase", "phase-cycle":
				g.PhaseOut(amulet.ID())
				want = 14
				if change == "phase-cycle" {
					g.PhaseIn([]uuid.UUID{amulet.ID()})
					want = 18
				}
			case "remove-ability", "remove-ability-expiry":
				g.AddContinuousEffect(mage.RemoveAllAbilities(amulet.ID(), core.EndOfTurn))
				g.ApplyContinuousEffects()
				want = 14
				if change == "remove-ability-expiry" {
					g.GetEffects().RemoveEndOfTurn()
					g.ApplyContinuousEffects()
					want = 18
				}
			case "cleanup":
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				want = 16
			case "control":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ChangeController(mage.This, mage.You)), amulet.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				want = 14
			case "clone":
				clone := g.Clone()
				clone.RemoveFromBattlefield(clone.FindPermanent(amulet.ID()))
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).Life() != 14 || a.Life() != 20 || g.StackSize() != 1 {
					t.Fatal("clone shared source, damage or stack")
				}
			}
			g.ResolveStack()
			if a.Life() != want {
				t.Fatalf("life %d, want %d", a.Life(), want)
			}
			if change == "control" {
				source := g.PutOnBattlefield(mage.NewInstant("Source", "", nil), a.PlayerID())
				g.DealDamageToPlayer(b, 6, source.ID())
				if b.Life() != 18 {
					t.Fatal("rule did not follow current controller")
				}
			}
		})
	}
}

func TestRONStaticDamageCapUpkeep(t *testing.T) {
	for _, accept := range []bool{false, true} {
		a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: accept}
		b := mage.NewBasePlayer("B")
		g := mage.NewGame(a, b)
		amulet := g.PutOnBattlefield(repositoryCard(t, "LEG", "Forethought Amulet"), a.PlayerID())
		a.ManaPool().Add(core.Colorless, 3)
		queueRONUpkeep(g, b)
		if g.StackSize() != 0 {
			t.Fatal("opponent upkeep triggered")
		}
		queueRONUpkeep(g, a)
		if g.StackSize() != 1 || a.offers != 0 {
			t.Fatal("upkeep did not wait for resolution")
		}
		g.ResolveStack()
		if (g.FindPermanent(amulet.ID()) != nil) != accept {
			t.Fatal("upkeep payment or sacrifice")
		}
		wantMana := 3
		if accept {
			wantMana = 0
		}
		if a.ManaPool().TotalMana() != wantMana {
			t.Fatal("incorrect upkeep payment")
		}
	}
}

func TestRONStaticDamageCapSingleTypeZero(t *testing.T) {
	g, a, b := ronGame()
	rule := strings.ReplaceAll(ronStaticDamageCap, "Or([IsCardtype(Instant),IsCardtype(Sorcery)])", "IsCardtype(Artifact)")
	rule = strings.ReplaceAll(rule, "DealDamageInstead(Integer(2))", "DealDamageInstead(Integer(0))")
	factory := permanentScopeFactory(t, "Artifact", rule)
	g.PutOnBattlefield(factory(), a.PlayerID())
	source := g.PutOnBattlefield(mage.NewArtifact("Source", ""), b.PlayerID())
	g.DealDamageToPlayer(a, 3, source.ID())
	if a.Life() != 20 {
		t.Fatal("single source type or zero damage replacement")
	}
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 3), b.PlayerID())
	g.DealDamageToPlayer(a, 3, other.ID())
	if a.Life() != 17 {
		t.Fatal("nonmatching source was reduced")
	}
}

func TestRONStaticDamageCapInvalid(t *testing.T) {
	for _, rule := range []string{
		strings.ReplaceAll(ronStaticDamageCap, "Integer(3)", "Integer(0)"),
		strings.ReplaceAll(ronStaticDamageCap, "Integer(2)", "Integer(-1)"),
		strings.ReplaceAll(ronStaticDamageCap, "Integer(2)", "Integer(3)"),
		strings.ReplaceAll(ronStaticDamageCap, "Integer(2)", "ValueX"),
		strings.ReplaceAll(ronStaticDamageCap, "GreaterThanOrEqualTo", "EqualTo"),
		strings.ReplaceAll(ronStaticDamageCap, "Player(You)", "Permanent(ThisPermanent)"),
		strings.ReplaceAll(ronStaticDamageCap, "Player(You)", "Player(Ref_TargetPlayer)"),
		strings.ReplaceAll(ronStaticDamageCap, "Player(You)", "Player(You())"),
		strings.ReplaceAll(ronStaticDamageCap, "Player(You)", "Player(Opponent)"),
		strings.ReplaceAll(ronStaticDamageCap, "IsCardtype(Instant)", "IsCardtype(Instant())"),
		strings.ReplaceAll(ronStaticDamageCap, "IsCardtype(Instant)", "IsCardtype(Unknown)"),
		strings.ReplaceAll(ronStaticDamageCap, "IsCardtype(Instant)", "IsAttacking"),
		strings.ReplaceAll(ronStaticDamageCap, "IsCardtype(Sorcery)", "IsCardtype(Instant)"),
		strings.ReplaceAll(ronStaticDamageCap, "Or([IsCardtype(Instant),IsCardtype(Sorcery)])", "Or([])"),
		strings.ReplaceAll(ronStaticDamageCap, "[DealDamageInstead(Integer(2))]", "[]"),
		strings.ReplaceAll(ronStaticDamageCap, "[DealDamageInstead(Integer(2))]", "[PreventThatDamage]"),
		strings.ReplaceAll(ronStaticDamageCap, "[DealDamageInstead(Integer(2))]", "[DealDamageInstead(Integer(2)),DealDamageInstead(Integer(1))]"),
		`Activated(PayMana([]),ActionList([` + ronStaticDamageCap + `]))`,
		`SpellActions(ActionList([` + ronStaticDamageCap + `]))`,
		`If(PermanentPassesFilter(ThisPermanent,IsUntapped),[` + ronStaticDamageCap + `])`,
		`AsPermanentEnters([` + ronStaticDamageCap + `])`,
		`ReplaceWouldDraw(APlayerWouldDrawACard(You),[` + ronStaticDamageCap + `])`,
		`ReplaceWouldDealDamage()`,
		`ReplaceWouldDealDamage(AnAmountOfDamageWouldBeDealtByASourceToRecipient(),[])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", rule)
		}
	}
}

func TestRONStaticDamageCapCollection(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Forethought Amulet"`) {
			record = line
		}
	}
	if record == "" {
		t.Fatal("missing complete record")
	}
	registry := mage.NewRegistry()
	files := fstest.MapFS{"amulet.ron": {Data: []byte(record)}}
	if err := LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"AnAmountOfDamageWouldBeDealtByASourceToRecipient": "damage_amount_event", "GreaterThanOrEqualTo": "damage_amount_comparison", "DealDamageInstead": "damage_amount_operation", "Or": "damage_source_filter", "Instant": "card_type", "You": "player_reference"} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing inventory role %s/%s", name, role)
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(strings.ReplaceAll(record, "Forethought Amulet", "Invalid"), "DealDamageInstead(Integer(2))", "DealDamageInstead(ValueX)"))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 || !registry.CardRegistered("Forethought Amulet") {
		t.Fatal("invalid collection changed the registry")
	}
}
