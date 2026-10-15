package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const ronDiscardLibraryRule = `ReplaceWouldDiscard(AnEffectWouldCausePlayerToDiscardACard(You),[MayAction(DiscardItToTopOfLibraryInstead)])`

type ronDiscardLibraryPlayer struct {
	*mage.BasePlayer
	accept    bool
	calls     int
	reasons   []string
	decisions []bool
}

func (p *ronDiscardLibraryPlayer) ChooseMayAbility(reason string) bool {
	p.calls++
	p.reasons = append(p.reasons, reason)
	if p.calls <= len(p.decisions) {
		return p.decisions[p.calls-1]
	}
	return p.accept
}

func TestRONDiscardReplacementRecords(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			a := &ronDiscardLibraryPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			card := repositoryCard(t, set, "Library of Leng")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast succeeded")
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.MaximumHandSize(a.PlayerID()) != -1 || g.MaximumHandSize(b.PlayerID()) != 7 {
				t.Fatal("hand size scope")
			}
			first := mage.NewLand("Effect discard")
			first.SetOwner(a.PlayerID())
			a.AddToHand(first)
			if _, ok := g.PlayerDiscardByEffect(a, first.ID(), card.ID()); !ok || len(a.Library()) != 1 || a.Library()[0].ID() != first.ID() || a.calls != 1 {
				t.Fatal("effect discard did not use replacement")
			}
			second := mage.NewLand("Cost discard")
			second.SetOwner(a.PlayerID())
			a.AddToHand(second)
			if _, ok := g.PlayerDiscardByCost(a, second.ID(), card.ID()); !ok || len(a.Graveyard()) != 1 || a.calls != 1 {
				t.Fatal("replacement applied to cost")
			}
			a.accept = false
			third := mage.NewLand("Declined")
			third.SetOwner(a.PlayerID())
			a.AddToHand(third)
			if _, ok := g.PlayerDiscardByEffect(a, third.ID(), card.ID()); !ok || len(a.Graveyard()) != 2 || a.calls != 2 {
				t.Fatal("decline failed")
			}
		})
	}
}

func TestRONDiscardReplacementInvalid(t *testing.T) {
	for _, rule := range []string{
		`ReplaceWouldDiscard()`,
		`ReplaceWouldDiscard(AnEffectWouldCausePlayerToDiscardACard(You))`,
		`ReplaceWouldDiscard(AnEffectWouldCausePlayerToDiscardACard(You),[])`,
		`ReplaceWouldDiscard(AnEffectWouldCausePlayerToDiscardACard(Opponent),[MayAction(DiscardItToTopOfLibraryInstead)])`,
		`ReplaceWouldDiscard(AnEffectWouldCausePlayerToDiscardACard(You()),[MayAction(DiscardItToTopOfLibraryInstead)])`,
		`ReplaceWouldDiscard(AnEffectWouldCausePlayerToDiscardACard(You,You),[MayAction(DiscardItToTopOfLibraryInstead)])`,
		`ReplaceWouldDiscard(PlayerWouldDiscard(You),[MayAction(DiscardItToTopOfLibraryInstead)])`,
		`ReplaceWouldDiscard(AnEffectWouldCausePlayerToDiscardACard(You),[DiscardItToTopOfLibraryInstead])`,
		`ReplaceWouldDiscard(AnEffectWouldCausePlayerToDiscardACard(You),[MayAction(DiscardItToTopOfLibraryInstead())])`,
		`ReplaceWouldDiscard(AnEffectWouldCausePlayerToDiscardACard(You),[MayAction(DiscardItToTopOfLibraryInstead,DiscardItToTopOfLibraryInstead)])`,
		`ReplaceWouldDiscard(AnEffectWouldCausePlayerToDiscardACard(You),[MayAction(DiscardItToTopOfLibraryInstead),MayAction(DiscardItToTopOfLibraryInstead)])`,
		`FromGraveyard(` + ronDiscardLibraryRule + `)`,
		`Activated(PayMana([]),ActionList([` + ronDiscardLibraryRule + `]))`,
		`AsPermanentEnters([` + ronDiscardLibraryRule + `])`,
	} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`)); err == nil {
			t.Fatal("accepted", rule)
		}
	}
	if _, err := Compile(nameScopeNode(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[`+ronDiscardLibraryRule+`])`)); err == nil {
		t.Fatal("accepted spell static")
	}
}

func TestRONDiscardReplacementLifecycle(t *testing.T) {
	for _, state := range []string{"live", "phase", "phase-cycle", "remove-abilities", "depart", "return", "control"} {
		t.Run(state, func(t *testing.T) {
			a := &ronDiscardLibraryPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			b := &ronDiscardLibraryPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
			g := mage.NewGame(a, b)
			card := repositoryCard(t, "LEA", "Library of Leng")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			clone := g.Clone()
			active := true
			recipient := a
			switch state {
			case "phase":
				g.PhaseOut(source.ID())
				active = false
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "remove-abilities":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				active = false
			case "depart", "return":
				g.BouncePermanentToHand(source)
				if state == "return" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
					recipient = b
				} else {
					active = false
				}
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				recipient = b
			}
			if len(g.WaitingDiscardPermanentSources()) != map[bool]int{true: 1, false: 0}[active] || a.calls+b.calls != 0 {
				t.Fatal("query changed state or wrong lifetime")
			}
			discarded := mage.NewLand("Card")
			discarded.SetOwner(recipient.PlayerID())
			recipient.AddToHand(discarded)
			if _, ok := g.PlayerDiscardByEffect(recipient, discarded.ID(), card.ID()); !ok {
				t.Fatal("discard failed")
			}
			if (len(recipient.Library()) == 1) != active || recipient.calls != map[bool]int{true: 1, false: 0}[active] {
				t.Fatal("wrong recipient or source lifetime")
			}
			if clone.MaximumHandSize(a.PlayerID()) != -1 || clone.MaximumHandSize(b.PlayerID()) != 7 || len(clone.WaitingDiscardPermanentSources()) != 1 || len(clone.GetPlayer(recipient.PlayerID()).Library()) != 0 {
				t.Fatal("clone changed")
			}
		})
	}
}

func TestRONDiscardReplacementCollection(t *testing.T) {
	source := `Card(Name:"Library fixture",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + ronDiscardLibraryRule + `])`
	files := fstest.MapFS{"valid.ron": {Data: []byte(source)}}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]string{"AnEffectWouldCausePlayerToDiscardACard": "discard_replacement_event", "You": "player_reference", "MayAction": "discard_replacement_operation", "DiscardItToTopOfLibraryInstead": "discard_replacement_destination"}
	for _, sig := range report.Signatures {
		if role, ok := roles[sig.Name]; ok && sig.Role == role {
			delete(roles, sig.Name)
		}
	}
	if len(roles) != 0 || report.Compiled != 1 {
		t.Fatal("missing typed roles", roles)
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(strings.ReplaceAll(source, "Library fixture", "Invalid"), "DiscardItToTopOfLibraryInstead)", "DiscardItToTopOfLibraryInstead())"))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 || !registry.CardRegistered("Library fixture") {
		t.Fatal("invalid collection changed registry")
	}
}

func TestRONDiscardReplacementRandomBatch(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		t.Run(map[bool]string{false: "all", true: "mixed"}[mixed], func(t *testing.T) {
			a := &ronDiscardLibraryPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			if mixed {
				a.decisions = []bool{true, false, true}
			}
			g := mage.NewGame(a, mage.NewBasePlayer("B"))
			g.SetStep(core.PrecombatMain)
			g.PutOnBattlefield(repositoryCard(t, "LEA", "Library of Leng"), a.PlayerID())
			spell := permanentScopeFactory(t, "Sorcery", `SpellActions(ActionList([DiscardNumberCardsAtRandom(Integer(3))]))`)()
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			names := []string{"First", "Second", "Third"}
			for _, name := range names {
				c := mage.NewLand(name)
				c.SetOwner(a.PlayerID())
				a.AddToHand(c)
			}
			g.SetRandomResults([]int{0, 0, 0, 7})
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			want := 3
			if mixed {
				want = 2
			}
			if len(a.Hand()) != 0 || len(a.Library()) != want || a.Library()[0].Name() != "First" || a.Library()[want-1].Name() != "Third" || a.calls != 3 || g.RandIntn(10) != 7 || len(g.PublicReveals()) != 0 {
				t.Fatal("random batch lost per-card replacement, privacy, or library order")
			}
			for i, reason := range a.reasons {
				if !strings.Contains(reason, names[i]) {
					t.Fatal("private prompt lacks selected random card")
				}
			}
			if mixed && (len(a.Graveyard()) != 2 || a.Graveyard()[0].Name() != "Second") {
				t.Fatal("declined card did not go to graveyard")
			}
		})
	}
}
