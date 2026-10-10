package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextSpellCastLayerRealCards(t *testing.T) {
	for _, tc := range []struct {
		name  string
		color core.Color
	}{
		{"Crystal Rod", core.Blue}, {"Iron Star", core.Red},
		{"Ivory Cup", core.White}, {"Throne of Bone", core.Black},
		{"Wooden Sphere", core.Green},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			card := repositoryCard(t, "LEA", tc.name)
			p := g.PutOnBattlefield(card, a.PlayerID())
			id := p.RuntimeAbilities[0].AbilityID()
			first := textTriggerSpell(tc.color)
			castRONSpell(t, g, b, first)
			change, _ := mage.ReplaceColorWord(tc.color, core.Colorless)
			if err := g.AddPrintedSpellCastTextEffect(p, id, core.EndOfTurn, change); err == nil {
				t.Fatal("invalid color change accepted")
			}
			next := core.Red
			if tc.color == next {
				next = core.Blue
			}
			change, _ = mage.ReplaceColorWord(tc.color, next)
			if err := g.AddPrintedSpellCastTextEffect(p, id, core.EndOfTurn, change); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(p.ID()).RuntimeAbilities[0].AbilityID() != id {
				t.Fatal("active identity changed")
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 2 || a.offers != 0 {
				t.Fatal("pending trigger lost text or made an early choice")
			}
			a.ManaPool().Add(core.Colorless, 1)
			g.ResolveTopOfStack()
			if a.Life() != 21 || a.offers != 1 {
				t.Fatal("pending payment lost its original declaration")
			}
			g.ResolveStack()
			castRONSpell(t, g, b, textTriggerSpell(tc.color))
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("old filter stayed active")
			}
			g.ResolveStack()
			castRONSpell(t, g, b, textTriggerSpell(next))
			g.PutTriggersOnStack()
			if g.StackSize() != 2 {
				t.Fatal("new filter did not trigger")
			}
			a.accept = false
			a.ManaPool().Add(core.Colorless, 1)
			g.ResolveTopOfStack()
			if a.Life() != 21 || a.ManaPool().TotalMana() != 1 {
				t.Fatal("declined revised payment spent resources")
			}
		})
	}
}

func TestRONTextSpellCastLayerPendingTargetsAndClone(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPlayerCastsASpell(AnyPlayer,IsColor(Red)),Targeted([TargetPermanent(IsColor(Red))],ActionList([DestroyPermanent(Ref_TargetPermanent)])))`)
	g, a, b := ronModalGame(0)
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	id := p.RuntimeAbilities[0].AbilityID()
	red := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	blue := g.PutOnBattlefield(textColorCard(core.Blue), b.PlayerID())
	castRONSpell(t, g, b, textTriggerSpell(core.Red))
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.AddPrintedSpellCastTextEffect(p, id, core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	a.targets = []uuid.UUID{red.ID()}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("pending targets used revised text")
	}
	castRONSpell(t, g, b, textTriggerSpell(core.Green))
	a.targets = []uuid.UUID{green.ID()}
	g.PutTriggersOnStack()
	if g.StackSize() != 4 {
		t.Fatal("fresh trigger lost targets")
	}
	clone := g.Clone()
	change, _ = mage.ReplaceColorWord(core.Green, core.Blue)
	if err := g.AddPrintedSpellCastTextEffect(p, id, core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	castRONSpell(t, g, b, textTriggerSpell(core.Blue))
	a.targets = []uuid.UUID{blue.ID()}
	g.PutTriggersOnStack()
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.ResolveStack()
	if g.FindPermanent(red.ID()) != nil || g.FindPermanent(green.ID()) != nil || g.FindPermanent(blue.ID()) != nil {
		t.Fatal("queued declarations changed")
	}
	if clone.FindPermanent(red.ID()) == nil || clone.FindPermanent(green.ID()) == nil {
		t.Fatal("clone shared resolution")
	}
	clone.ResolveStack()
	if clone.FindPermanent(red.ID()) != nil || clone.FindPermanent(green.ID()) != nil || clone.FindPermanent(blue.ID()) == nil {
		t.Fatal("clone lost earlier declarations")
	}
	castRONSpell(t, clone, clone.GetPlayer(b.PlayerID()).(*mage.SearchPlayer).BasePlayer, textTriggerSpell(core.Blue))
	clone.PutTriggersOnStack()
	if clone.StackSize() != 1 {
		t.Fatal("clone inherited a later text change")
	}
}

func TestRONTextSpellCastLayerIdentityAndLifetime(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Sol'kanar the Swamp King")
	p := g.PutOnBattlefield(card, a.PlayerID())
	var id uuid.UUID
	for _, ability := range p.RuntimeAbilities {
		if _, ok := ability.(*mage.GenericTriggered); ok {
			id = ability.AbilityID()
			break
		}
	}
	if id == uuid.Nil {
		t.Fatal("missing printed spell-cast trigger")
	}
	change, _ := mage.ReplaceColorWord(core.Black, core.Red)
	if err := g.AddPrintedSpellCastTextEffect(p, id, core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(p.ID()).HasKeyword(core.Swampwalk) {
		t.Fatal("sibling landwalk changed")
	}
	castRONSpell(t, g, b, textTriggerSpell(core.Red))
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.ApplyContinuousEffects()
	phased := g.PhaseOut(p.ID())
	if err := g.AddPrintedSpellCastTextEffect(p, id, core.EndOfTurn, change); err == nil {
		t.Fatal("phased source revised")
	}
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatal("pending controller changed")
	}
	g.PhaseIn(phased)
	castRONSpell(t, g, a, textTriggerSpell(core.Red))
	g.ResolveStack()
	if b.Life() != 21 {
		t.Fatal("phase cycle lost revision or current controller")
	}
	stale := g.FindPermanent(p.ID())
	g.BouncePermanentToHand(stale)
	a.RemoveFromHand(card.ID())
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.AddPrintedSpellCastTextEffect(stale, id, core.EndOfTurn, change); err == nil {
		t.Fatal("returned source accepted old visit")
	}
	castRONSpell(t, g, b, textTriggerSpell(core.Red))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("returned source inherited revision")
	}
	g.ResolveStack()
	castRONSpell(t, g, b, textTriggerSpell(core.Black))
	g.ResolveStack()
	if a.Life() != 22 || !returned.HasKeyword(core.Swampwalk) {
		t.Fatal("printed origin not restored")
	}
}

func TestRONTextSpellCastLayerEventBindings(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPlayerCastsASpell(AnyPlayer,IsColor(Red)),ActionList([PlayerMayCost(Trigger_ThatPlayer,PayMana([ManaCostGeneric(1)])),Unless(CostWasPaid,[CounterSpell(Trigger_ThatSpell)])]))`)
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: false}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	id := p.RuntimeAbilities[0].AbilityID()
	first := castRONSpell(t, g, a.BasePlayer, textTriggerSpell(core.Red))
	g.PutTriggersOnStack()
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.AddPrintedSpellCastTextEffect(p, id, core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	second := castRONSpell(t, g, b.BasePlayer, textTriggerSpell(core.Green))
	g.PutTriggersOnStack()
	if g.StackSize() != 4 || a.offers != 0 || b.offers != 0 {
		t.Fatal("event detection or choice timing changed")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	g.ResolveTopOfStack()
	g.ResolveTopOfStack()
	if g.GetStack().FindByID(first.ID) == nil || g.GetStack().FindByID(second.ID) != nil || a.offers != 0 || b.offers != 2 {
		t.Fatalf("captured events: first=%v second=%v offers=%d/%d", g.GetStack().FindByID(first.ID) != nil, g.GetStack().FindByID(second.ID) != nil, a.offers, b.offers)
	}
	g.ResolveTopOfStack()
	if a.offers != 1 || g.GetStack().FindByID(first.ID) == nil {
		t.Fatal("pending earlier binding was overwritten")
	}
}

func TestRONTextSpellCastLayerOtherRealCards(t *testing.T) {
	for _, tc := range []struct{ set, name, spell, spellSet string }{
		{"ATQ", "Citanul Druid", "Sol Ring", "LEA"},
		{"ATQ", "Urza's Chalice", "Sol Ring", "LEA"},
		{"LEG", "Presence of the Master", "Crusade", "LEA"},
		{"LEG", "Nether Void", "Sol Ring", "LEA"},
		{"LEG", "In the Eye of Chaos", "Dark Ritual", "LEA"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: false}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetActivePlayerIndex(1)
			g.SetTurn(1)
			p := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
			id := p.RuntimeAbilities[0].AbilityID()
			change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
			if err := g.AddPrintedSpellCastTextEffect(p, id, core.EndOfTurn, change); err != nil {
				t.Fatal(err)
			}
			victim := castRONSpell(t, g, b.BasePlayer, repositoryCard(t, tc.spellSet, tc.spell))
			g.PutTriggersOnStack()
			if g.StackSize() != 2 {
				t.Fatal("revised trigger did not detect the cast")
			}
			a.ManaPool().Add(core.Colorless, 1)
			g.ResolveTopOfStack()
			switch tc.name {
			case "Citanul Druid":
				if g.FindPermanent(p.ID()).Counters[core.P1P1] != 1 {
					t.Fatal("revised counter body failed")
				}
			case "Urza's Chalice":
				if a.Life() != 21 || a.offers != 1 || b.offers != 0 {
					t.Fatal("revised payment body failed")
				}
			default:
				if g.GetStack().FindByID(victim.ID) != nil {
					t.Fatal("revised event binding failed to counter")
				}
				if tc.name != "Presence of the Master" && (b.offers != 1 || a.offers != 0) {
					t.Fatal("wrong payment player")
				}
			}
		})
	}
}

func TestRONTextSpellCastLayerInvalid(t *testing.T) {
	for _, rule := range []string{
		`TriggerA(WhenAPlayerCastsASpell(AnyPlayer(),IsColor(Red)),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPlayerCastsASpell(AnyPlayer,IsColor(Red)),ActionList([CounterSpell(Trigger_ThatPlayer)]))`,
		`TriggerA(WhenAPlayerCastsASpell(AnyPlayer,IsColor(Red)),Targeted([TargetPermanent(IsColor(Red))],ActionList([DestroyPermanent(Ref_TargetPermanent_2)])))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid revised declaration accepted")
		}
	}
	for _, name := range []string{"Magical Hack", "Sleight of Mind"} {
		for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
			if _, err := Compile(hostUntapRecord(t, set, name)); err == nil {
				t.Fatal("incomplete text consumer accepted", set, name)
			}
		}
	}
}

func TestRONTextSpellCastLayerIndependentDeclarations(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPlayerCastsASpell(AnyPlayer,IsColor(Red)),ActionList([GainLife(Integer(1))])),TriggerA(WhenAPlayerCastsASpell(AnyPlayer,IsColor(Red)),ActionList([GainLife(Integer(4))]))`)
	g, a, b := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	other := g.PutOnBattlefield(factory(), b.PlayerID())
	first, second := p.RuntimeAbilities[0].AbilityID(), p.RuntimeAbilities[1].AbilityID()
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.AddPrintedSpellCastTextEffect(p, other.RuntimeAbilities[0].AbilityID(), core.EndOfTurn, change); err == nil {
		t.Fatal("another instance supplied the declaration")
	}
	if err := g.AddPrintedSpellCastTextEffect(p, first, core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	change, _ = mage.ReplaceColorWord(core.Red, core.Blue)
	if err := g.AddPrintedSpellCastTextEffect(p, second, core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	castRONSpell(t, g, a, textTriggerSpell(core.Green))
	castRONSpell(t, g, a, textTriggerSpell(core.Blue))
	g.PutTriggersOnStack()
	if g.StackSize() != 4 {
		t.Fatal("independent declarations shared revision state")
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	g.ResolveStack()
	if a.Life() != 25 || b.Life() != 20 {
		t.Fatal("departure lost pending bodies or shared another instance")
	}
	if err := g.AddPrintedSpellCastTextEffect(other, other.RuntimeAbilities[0].AbilityID(), core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
}
