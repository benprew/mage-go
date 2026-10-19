package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronOneSpellManaPlayer struct{ *mage.BasePlayer }

func (*ronOneSpellManaPlayer) ChooseMayAbility(string) bool { return true }

const oneSpellManaAction = `CreatePlayerEffectUntil(You,[MayActionOnce(CastASpellAndMaySpendManaAsThoughAnyTypeToPayManaCost(AnySpell))],UntilEndOfTurn)`

func TestRONOneSpellManaGameplay(t *testing.T) {
	a := &ronOneSpellManaPlayer{mage.NewBasePlayer("A")}
	g := mage.NewGame(a, mage.NewBasePlayer("B"))
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	c := repositoryCard(t, "LEG", "North Star")
	a.AddToHand(c)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil {
		t.Fatal("unpaid casting")
	}
	a.ManaPool().Add(core.Colorless, 4)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), c.ID(), 0, nil); err == nil {
		t.Fatal("unpaid activation")
	}
	a.ManaPool().Add(core.Colorless, 4)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), c.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(c.ID()).Tapped {
		t.Fatal("activation did not tap")
	}
	g.DestroyPermanent(g.FindPermanent(c.ID()))
	g.ResolveStack()
	nodes, err := Parse(strings.NewReader(`Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),ManaCost:[ManaCostW],Rules:[SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,GainLife(Integer(3)))])))])`), "probe.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	spell := factory()
	a.AddToHand(spell)
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{uuid.New()}, 0); err == nil {
		t.Fatal("illegal target accepted")
	}
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{a.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 23 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("permission did not pay or spell did not resolve")
	}
	second := factory()
	a.AddToHand(second)
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), second.ID(), []uuid.UUID{a.PlayerID()}, 0); err == nil {
		t.Fatal("permission used twice")
	}
}

func TestRONOneSpellManaDeclarations(t *testing.T) {
	for _, bad := range []string{
		strings.Replace(oneSpellManaAction, "You", "Ref_TargetPlayer", 1),
		strings.Replace(oneSpellManaAction, "AnySpell", "AnySpell()", 1),
		strings.Replace(oneSpellManaAction, "AnySpell", "IsCardtype(Creature)", 1),
		strings.Replace(oneSpellManaAction, "UntilEndOfTurn", "UntilEndOfCombat", 1),
		strings.Replace(oneSpellManaAction, "UntilEndOfTurn", "UntilEndOfTurn()", 1),
		strings.Replace(oneSpellManaAction, "MayActionOnce(", "MayActionOnce(Unknown,", 1),
		strings.Replace(oneSpellManaAction, "AnyTypeToPayManaCost", "AnyColorToCast", 1),
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + bad + `]))])`
		if _, err := LoadFS(fstest.MapFS{"invalid.ron": {Data: []byte(src)}}, "*.ron"); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
}

func TestRONOneSpellManaLoadAndInventory(t *testing.T) {
	node := repositoryRecord(t, "LEG", "North Star", "star.ron")
	build, err := Compile(node)
	if err != nil {
		t.Fatal(err)
	}
	first, second := build(), build()
	if first.ID() == second.ID() {
		t.Fatal("factory reused card identity")
	}
	if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
		t.Fatal(err)
	}
	source := `Card(Name:"Permission",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + oneSpellManaAction + `]))])`
	fs := fstest.MapFS{"valid.ron": {Data: []byte(source)}}
	registry, err := LoadFS(fs, "*.ron")
	if err != nil || registry == nil {
		t.Fatalf("strict loading: %v", err)
	}
	report, err := InspectFS(fs, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatalf("inventory: %v", err)
	}
	roles := map[string]bool{}
	for _, signature := range report.Signatures {
		if signature.Name == "MayActionOnce" {
			roles[signature.Role] = true
		}
		if signature.Name == "CastASpellAndMaySpendManaAsThoughAnyTypeToPayManaCost" {
			roles[signature.Role] = true
		}
		if signature.Name == "AnySpell" {
			roles[signature.Role] = true
		}
	}
	for _, role := range []string{"temporary_player_operation", "one_spell_mana_action", "spell_filter"} {
		if !roles[role] {
			t.Fatalf("missing role %s", role)
		}
	}
	fs["invalid.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(source, "AnySpell", "Unknown", 1))}
	if result, err := LoadFS(fs, "*.ron"); err == nil || result != nil {
		t.Fatal("invalid collection published a registry")
	}
}

func TestRONOneSpellManaContext(t *testing.T) {
	for _, rule := range []string{
		`AsPermanentEnters([` + oneSpellManaAction + `])`,
		`PlayerEffect(You,[MayActionOnce(CastASpellAndMaySpendManaAsThoughAnyTypeToPayManaCost(AnySpell))])`,
		`SpellActions(ActionList([MayActionOnce(CastASpellAndMaySpendManaAsThoughAnyTypeToPayManaCost(AnySpell))]))`,
	} {
		source := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + rule + `])`
		if _, err := LoadFS(fstest.MapFS{"invalid.ron": {Data: []byte(source)}}, "*.ron"); err == nil {
			t.Fatalf("accepted escaped context %s", rule)
		}
	}
}
