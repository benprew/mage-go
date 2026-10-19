package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const castHistoryBody = `ChooseAPlayer(CastASpellThisTurn(IsCardtype(Sorcery))),ChooseASpellCastThisTurn(And([IsCardtype(Sorcery),CastByPlayer(TheChosenPlayer)])),SpellDealsDamage(ThisSpell,HalfRoundedDown(DamageDealtBySpellThisTurn(TheSpellCastChosenThisWay)),Player(TheChosenPlayer))`

func TestRONCastSpellHistoryGameplay(t *testing.T) {
	g, a, b := ronGame()
	history := permanentScopeFactory(t, "Sorcery", `SpellActions(ActionList([SpellDealsDamage(ThisSpell,Integer(5),Player(You))]))`)()
	history.SetOwner(a.PlayerID())
	a.AddToHand(history)
	if err := g.CastSpellByID(a.PlayerID(), history.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	card := repositoryCard(t, "LEG", "Backdraft")
	card.SetOwner(b.PlayerID())
	b.AddToHand(card)
	if err := g.CastSpellByID(b.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast accepted")
	}
	b.ManaPool().Add(core.Red, 2)
	if err := g.CastSpellByID(b.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.ResolveStack()
	clone.ResolveStack()
	if a.Life() != 13 || b.Life() != 20 || clone.GetPlayer(a.PlayerID()).Life() != 13 {
		t.Fatalf("life %d/%d", a.Life(), b.Life())
	}
}

func TestRONCastSpellHistoryDeclarations(t *testing.T) {
	valid := `Card(Name:"History",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),ManaCost:[],Rules:[SpellActions(ActionList([BODY]))])`
	for _, body := range []string{
		strings.Replace(castHistoryBody, "ChooseASpellCastThisTurn(And([IsCardtype(Sorcery),CastByPlayer(TheChosenPlayer)])),", "", 1),
		strings.Replace(castHistoryBody, "TheSpellCastChosenThisWay", "TheSpellCastChosenThisWay()", 1),
		strings.Replace(castHistoryBody, "CastByPlayer(TheChosenPlayer)", "CastByPlayer(ActionPlayer)", 1),
		strings.Replace(castHistoryBody, "ChooseASpellCastThisTurn(And([IsCardtype(Sorcery),CastByPlayer(TheChosenPlayer)]))", "ChooseASpellCastThisTurn(IsCardtype(Sorcery))", 1),
		strings.Replace(castHistoryBody, "ChooseASpellCastThisTurn", "ChooseASpellThatResolvedThisTurn", 1),
		`ChooseAnAction([[ChooseASpellCastThisTurn(And([IsCardtype(Sorcery),CastByPlayer(You)]))],[GainLife(Integer(0))]]),SpellDealsDamage(ThisSpell,DamageDealtBySpellThisTurn(TheSpellCastChosenThisWay),Player(You))`,
		strings.Replace(castHistoryBody, "DamageDealtBySpellThisTurn(TheSpellCastChosenThisWay)", "DamageDealtBySpellThisTurn()", 1),
		strings.Replace(castHistoryBody, "IsCardtype(Sorcery),CastByPlayer(TheChosenPlayer)", "CastByPlayer(You),CastByPlayer(You)", 1),
	} {
		nodes, err := Parse(strings.NewReader(strings.Replace(valid, "BODY", body, 1)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	permanentScopeFactory(t, "Instant", `SpellActions(ActionList([`+castHistoryBody+`]))`)
	permanentScopeFactory(t, "Instant", `SpellActions(ActionList([ChooseAnAction([[`+castHistoryBody+`],[GainLife(Integer(0))]])]))`)
	nodes, err := Parse(strings.NewReader(`ChooseASpellCastThisTurn(And([IsCardtype(Sorcery),CastByPlayer(You)]))`), "context.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []scope{{}, {resolving: true, enteringSource: true}, {resolving: true, drawReplacementBody: true}} {
		if _, err := compileCastSpellChoice(nodes[0], nil, ctx); err == nil {
			t.Fatal("invalid history context accepted")
		}
	}
}

type ronCastHistoryChooser struct {
	mage.Player
	index, calls int
	options      []mage.DamageSource
}

func (p *ronCastHistoryChooser) ChooseDamageSource(sources []mage.DamageSource, _ *mage.Game) int {
	p.calls++
	p.options = sources
	return p.index
}

func TestRONCastSpellHistoryCopyChoices(t *testing.T) {
	g, a, base := ronGame()
	b := &ronCastHistoryChooser{Player: base}
	g.SetPlayerAt(1, b)
	for _, amount := range []string{"3", "5"} {
		card := permanentScopeFactory(t, "Sorcery", `SpellActions(ActionList([SpellDealsDamage(ThisSpell,Integer(`+amount+`),Player(You))]))`)()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	card := repositoryCard(t, "LEG", "Backdraft")
	card.SetOwner(b.PlayerID())
	b.AddToHand(card)
	b.ManaPool().Add(core.Red, 2)
	if err := g.CastSpellByID(b.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if b.calls != 0 {
		t.Fatal("choice at announcement")
	}
	original := g.StackPeek()
	if g.CopyStackObjectDirect(original, b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	clonePlayer := &ronCastHistoryChooser{Player: clone.GetPlayer(b.PlayerID()), index: 1}
	clone.SetPlayerAt(1, clonePlayer)
	g.ResolveTopOfStack()
	b.index = 1
	g.ResolveTopOfStack()
	clone.ResolveStack()
	if b.calls != 2 || clonePlayer.calls != 2 || len(b.options) != 2 || a.Life() != 9 || clone.GetPlayer(a.PlayerID()).Life() != 8 {
		t.Fatalf("independent choices: calls %d/%d life %d/%d", b.calls, clonePlayer.calls, a.Life(), clone.GetPlayer(a.PlayerID()).Life())
	}
}

func TestRONCastSpellHistoryEmptyAndCountered(t *testing.T) {
	for _, status := range []string{"empty", "pending", "countered", "countered-damage"} {
		t.Run(status, func(t *testing.T) {
			g, a, b := ronGame()
			if status != "empty" {
				card := permanentScopeFactory(t, "Sorcery", `SpellActions(ActionList([GainLife(Integer(0))]))`)()
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				obj := g.StackPeek()
				if status == "countered-damage" {
					g.DealDamageToPlayerFromSource(b, 7, g.CaptureSpellDamageSource(obj.ID))
				}
				if status != "pending" {
					g.CounterStackObject(obj.ID)
				}
			}
			card := repositoryCard(t, "LEG", "Backdraft")
			card.SetOwner(b.PlayerID())
			b.AddToHand(card)
			b.ManaPool().Add(core.Red, 2)
			if err := g.CastSpellByID(b.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveTopOfStack()
			want := 20
			if status == "countered-damage" {
				want = 17
			}
			if a.Life() != want {
				t.Fatalf("life = %d, want %d", a.Life(), want)
			}
		})
	}
}

func TestRONCastSpellHistoryLoadAndInventory(t *testing.T) {
	node := repositoryRecord(t, "LEG", "Backdraft", "backdraft.ron")
	factory, err := Compile(node)
	if err != nil {
		t.Fatal(err)
	}
	first, second := factory(), factory()
	if first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("shared factory declaration")
	}
	if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
		t.Fatal(err)
	}
	file, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(file), "\n") {
		if strings.Contains(line, `Name: "Backdraft"`) {
			record = line
		}
	}
	files := fstest.MapFS{"history.ron": {Data: []byte(record)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || registry == nil {
		t.Fatalf("strict load: %v", err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]string{"ChooseASpellCastThisTurn": "action", "And": "cast_spell_filter", "CastByPlayer": "cast_spell_filter", "TheSpellCastChosenThisWay": "cast_spell_reference", "DamageDealtBySpellThisTurn": "value"}
	for name, role := range roles {
		found := false
		for _, entry := range report.Signatures {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s role %s", name, role)
		}
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(record, "TheSpellCastChosenThisWay", "TheSpellCastChosenThisWay()", 1))}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("partial registry published")
	}
}
