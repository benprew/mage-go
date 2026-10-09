package ron

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func combatTextChange(t *testing.T, from, to core.Color) mage.TextChange {
	t.Helper()
	change, err := mage.ReplaceColorWord(from, to)
	if err != nil {
		t.Fatal(err)
	}
	return change
}

func fireTextPair(g *mage.Game, source, other *mage.Permanent, kind core.EventType) {
	g.FireEvent(core.GameEvent{Type: kind, SourceID: source.ID(), TargetID: other.ID(), PlayerID: source.ControllerID()})
}

func TestRONTextCombatPairRecords(t *testing.T) {
	for _, kind := range []core.EventType{core.EvtDeclaredBlocker, core.EvtCreatureBlockedBy} {
		t.Run(kind.String(), func(t *testing.T) {
			g, a, b := ronGame()
			p := g.PutOnBattlefield(repositoryCard(t, "LEG", "Abomination"), a.PlayerID())
			green := g.PutOnBattlefield(mage.NewCreature("Green", "{G}", 0, 8), b.PlayerID())
			red := g.PutOnBattlefield(mage.NewCreature("Red", "{R}", 0, 8), b.PlayerID())
			white := g.PutOnBattlefield(mage.NewCreature("White", "{W}", 0, 8), b.PlayerID())
			blue := g.PutOnBattlefield(mage.NewCreature("Blue", "{U}", 0, 8), b.PlayerID())
			idx := 0
			if kind == core.EvtCreatureBlockedBy {
				idx = 1
			}
			id := p.RuntimeAbilities[idx].AbilityID()
			fireTextPair(g, p, green, kind)
			if err := g.RevisePrintedCombatPairTrigger(p, id, combatTextChange(t, core.Green, core.Red)); err != nil {
				t.Fatal(err)
			}
			fireTextPair(g, p, green, kind)
			fireTextPair(g, p, red, kind)
			fireTextPair(g, p, white, kind)
			g.PutTriggersOnStack()
			if g.StackSize() != 3 {
				t.Fatalf("pending event text changed: %d", g.StackSize())
			}
			clone := g.Clone()
			if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			if err := g.RevisePrintedCombatPairTrigger(p, id, combatTextChange(t, core.Red, core.Blue)); err != nil {
				t.Fatal(err)
			}
			fireTextPair(g, p, red, kind)
			fireTextPair(g, p, blue, kind)
			g.PutTriggersOnStack()
			if g.StackSize() != 5 {
				t.Fatalf("ordered text changes lost: %d", g.StackSize())
			}
			g.ResolveStack()
			if g.FindPermanent(green.ID()) == nil || g.FindPermanent(red.ID()) == nil {
				t.Fatal("delayed destruction ran early")
			}
			g.BouncePermanentToHand(blue)
			b.RemoveFromHand(blue.Card.ID())
			returned := g.PutOnBattlefield(blue.Card, b.PlayerID())
			g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
			g.RunStepWithPriority(core.EndCombat)
			if g.FindPermanent(green.ID()) != nil || g.FindPermanent(red.ID()) != nil || g.FindPermanent(white.ID()) != nil || g.FindPermanent(returned.ID()) == nil {
				t.Fatal("delayed exact event subjects changed")
			}
			if clone.FindPermanent(green.ID()) == nil {
				t.Fatal("clone shared destruction")
			}
			cp := clone.FindPermanent(p.ID())
			fireTextPair(clone, cp, clone.FindPermanent(red.ID()), kind)
			clone.PutTriggersOnStack()
			if clone.StackSize() != 4 {
				t.Fatal("clone inherited later revision")
			}
			clone.ResolveStack()
			clone.RunStepWithPriority(core.EndCombat)
			if clone.FindPermanent(green.ID()) != nil || clone.FindPermanent(red.ID()) != nil || clone.FindPermanent(white.ID()) != nil || clone.FindPermanent(blue.ID()) == nil {
				t.Fatal("clone lost saved bodies")
			}
		})
	}
}

func TestRONTextCombatPairBodyAndLifetime(t *testing.T) {
	for _, kind := range []core.EventType{core.EvtDeclaredBlocker, core.EvtCreatureBlockedBy} {
		t.Run(kind.String(), func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Aisling Leprechaun")
			p := g.PutOnBattlefield(card, a.PlayerID())
			sibling := g.PutOnBattlefield(repositoryCard(t, "LEG", "Aisling Leprechaun"), a.PlayerID())
			old := g.PutOnBattlefield(mage.NewCreature("Old", "", 0, 8), b.PlayerID())
			fresh := g.PutOnBattlefield(mage.NewCreature("Fresh", "", 0, 8), b.PlayerID())
			idx := 0
			if kind == core.EvtCreatureBlockedBy {
				idx = 1
			}
			id := p.RuntimeAbilities[idx].AbilityID()
			change := combatTextChange(t, core.Green, core.Red)
			fireTextPair(g, p, old, kind)
			if err := g.RevisePrintedCombatPairTrigger(p, id, change); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(p.ID()).RuntimeAbilities[idx].AbilityID() != id {
				t.Fatal("active ID changed")
			}
			fireTextPair(g, p, fresh, kind)
			g.PutTriggersOnStack()
			if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			g.ResolveStack()
			if !slices.Contains(g.FindPermanent(old.ID()).Colors(), core.Green) || !slices.Contains(g.FindPermanent(fresh.ID()).Colors(), core.Red) {
				t.Fatal("pending or fresh body text lost")
			}
			if err := g.RevisePrintedCombatPairTrigger(p, sibling.RuntimeAbilities[idx].AbilityID(), change); err == nil {
				t.Fatal("foreign declaration accepted")
			}
			phased := g.PhaseOut(p.ID())
			if err := g.RevisePrintedCombatPairTrigger(p, id, change); err == nil {
				t.Fatal("phased source accepted")
			}
			g.PhaseIn(phased)
			remover := g.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), a.PlayerID())
			removal := mage.RemoveAllAbilities(p.ID(), core.WhileOnBattlefield)
			removal.SetSourceID(remover.ID())
			g.AddContinuousEffect(removal)
			g.ApplyContinuousEffects()
			if err := g.RevisePrintedCombatPairTrigger(p, id, change); err == nil {
				t.Fatal("removed declaration accepted")
			}
			fireTextPair(g, g.FindPermanent(p.ID()), old, kind)
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("removed ability triggered")
			}
			clone := g.Clone()
			clone.RemoveFromBattlefield(clone.FindPermanent(remover.ID()))
			clone.ApplyContinuousEffects()
			fireTextPair(clone, clone.FindPermanent(p.ID()), clone.FindPermanent(old.ID()), kind)
			clone.ResolveStack()
			if !slices.Contains(clone.FindPermanent(old.ID()).Colors(), core.Red) || !slices.Contains(g.FindPermanent(old.ID()).Colors(), core.Green) {
				t.Fatal("clone restoration shared state or lost revision")
			}
			g.RemoveFromBattlefield(remover)
			g.ApplyContinuousEffects()
			fireTextPair(g, g.FindPermanent(p.ID()), old, kind)
			g.ResolveStack()
			if !slices.Contains(g.FindPermanent(old.ID()).Colors(), core.Red) {
				t.Fatal("restoration lost revision")
			}
			stale := g.FindPermanent(p.ID())
			g.BouncePermanentToHand(stale)
			a.RemoveFromHand(card.ID())
			returned := g.PutOnBattlefield(card, a.PlayerID())
			if err := g.RevisePrintedCombatPairTrigger(stale, id, change); err == nil {
				t.Fatal("old source visit accepted")
			}
			fireTextPair(g, returned, fresh, kind)
			g.ResolveStack()
			if !slices.Contains(g.FindPermanent(fresh.ID()).Colors(), core.Green) {
				t.Fatal("returned source inherited text")
			}
		})
	}
}

func TestRONTextCombatPairAuraAndRecords(t *testing.T) {
	type record struct{ set, name string }
	cards := make([]record, 0, 15)
	cards = append(cards, record{"LEG", "Infernal Medusa"}, record{"ATQ", "Battering Ram"}, record{"DRK", "Venom"})
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		cards = append(cards, record{set, "Cockatrice"}, record{set, "Thicket Basilisk"})
	}
	for _, tc := range cards {
		t.Run(tc.set+"/"+tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			var p, subject *mage.Permanent
			if tc.name == "Venom" {
				subject = g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 8), a.PlayerID())
				p = g.PutOnBattlefield(card, a.PlayerID())
				g.Attach(p.ID(), subject.ID())
			} else {
				p = g.PutOnBattlefield(card, a.PlayerID())
				subject = p
			}
			for _, ability := range p.RuntimeAbilities {
				if _, ok := ability.(*mage.GenericTriggered); !ok {
					continue
				}
				if tc.name == "Battering Ram" && ability == p.RuntimeAbilities[0] {
					continue
				}
				if err := g.RevisePrintedCombatPairTrigger(p, ability.AbilityID(), combatTextChange(t, core.Green, core.Red)); err != nil {
					t.Fatal(err)
				}
			}
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 0, 8), b.PlayerID())
			if tc.name == "Battering Ram" {
				other = g.PutOnBattlefield(mage.NewCreature("Wall", "", 0, 8, mage.WithSubTypes("Wall")), b.PlayerID())
			}
			kind := core.EvtCreatureBlockedBy
			if tc.name == "Venom" {
				kind = core.EvtCombatPair
			}
			fireTextPair(g, subject, other, kind)
			g.ResolveStack()
			g.RunStepWithPriority(core.EndCombat)
			if g.FindPermanent(other.ID()) != nil {
				t.Fatal("revised declaration lost delayed binding")
			}
		})
	}
}

func TestRONTextCombatPairBoundaries(t *testing.T) {
	for _, rule := range []string{
		`TriggerA(WhenAPlayerCastsASpell(AnyPlayer,IsColor(Red)),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenACreatureBlocks(SinglePermanent(ThisPermanent)),ActionList([GainLife(Integer(1))]))`,
	} {
		g, a, _ := ronGame()
		p := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", rule)(), a.PlayerID())
		if err := g.RevisePrintedCombatPairTrigger(p, p.RuntimeAbilities[0].AbilityID(), combatTextChange(t, core.Red, core.Green)); err == nil {
			t.Fatal("unrelated event accepted")
		}
	}
	for _, event := range []string{
		`WhenACreatureBlocksACreature(SinglePermanent(You),IsCardtype(Creature))`,
		`WhenACreatureBlocksACreature(SinglePermanent(ThisPermanent),IsColor(Green))`,
		`WhenACreatureBlocksACreature(SinglePermanent(ThisPermanent),And([IsCardtype(Creature),IsColor(Green),IsColor(Red)]))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[TriggerA(` + event + `,ActionList([GainLife(Integer(1))]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid event accepted")
		}
	}
}

func TestRONTextCombatPairPendingTargets(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name:"Targets",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:8),Rules:[TriggerA(WhenACreatureBlocksACreature(SinglePermanent(ThisPermanent),IsCardtype(Creature)),Targeted([TargetPermanent(IsColor(Green))],ActionList([DestroyPermanent(Ref_TargetPermanent)])))])`), "targets.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronModalGame(0)
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	opponent := g.PutOnBattlefield(mage.NewCreature("Opponent", "", 0, 8), b.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	red := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	id := p.RuntimeAbilities[0].AbilityID()
	fireTextPair(g, p, opponent, core.EvtDeclaredBlocker)
	if err := g.RevisePrintedCombatPairTrigger(p, id, combatTextChange(t, core.Green, core.Red)); err != nil {
		t.Fatal(err)
	}
	a.targets = []uuid.UUID{green.ID()}
	g.PutTriggersOnStack()
	fireTextPair(g, p, opponent, core.EvtDeclaredBlocker)
	a.targets = []uuid.UUID{red.ID()}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("pending target declarations used fresh text")
	}
	g.ResolveStack()
	if g.FindPermanent(green.ID()) != nil || g.FindPermanent(red.ID()) != nil {
		t.Fatal("target groups lost")
	}
}

func TestRONTextCombatPairPaidCastAndNativeBlocks(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Aisling Leprechaun")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast accepted")
	}
	if g.StackSize() != 0 || g.FindPermanent(card.ID()) != nil || len(a.Hand()) != 1 {
		t.Fatal("failed cast changed resources")
	}
	a.ManaPool().Add(core.Green, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()) != nil {
		t.Fatal("creature entered before resolution")
	}
	g.ResolveTopOfStack()
	p := g.FindPermanent(card.ID())
	if err := g.RevisePrintedCombatPairTrigger(p, p.RuntimeAbilities[1].AbilityID(), combatTextChange(t, core.Green, core.Red)); err != nil {
		t.Fatal(err)
	}
	p.RevokeBaseAttr(core.AttrSummonSick)
	q := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 0, 8), b.PlayerID())
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
	g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: q.ID(), AttackerID: p.ID()}})
	if !slices.Contains(g.FindPermanent(q.ID()).Colors(), core.Red) {
		t.Fatal("native declaration missed revised body")
	}
}

func TestRONTextCombatPairControlAndLKI(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name:"Control",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:8),Rules:[TriggerA(WhenACreatureBlocksACreature(SinglePermanent(ThisPermanent),And([IsCardtype(Creature),IsColor(Green)])),ActionList([PlayerAction(ControllerOfPermanent(ThisPermanent),GainLife(Integer(1)))]))])`), "control.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	red := g.PutOnBattlefield(textColorCard(core.Red), a.PlayerID())
	fireTextPair(g, p, green, core.EvtDeclaredBlocker)
	if err := g.RevisePrintedCombatPairTrigger(p, p.RuntimeAbilities[0].AbilityID(), combatTextChange(t, core.Green, core.Red)); err != nil {
		t.Fatal(err)
	}
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.ApplyContinuousEffects()
	fireTextPair(g, g.FindPermanent(p.ID()), red, core.EvtDeclaredBlocker)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("controller change lost detection")
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 22 {
		t.Fatal("source controller LKI lost")
	}
}
