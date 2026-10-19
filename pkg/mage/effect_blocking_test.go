package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type effectBlockingPlayer struct {
	Player
	accept       bool
	name         string
	choices      int
	beforeChoice func()
}

func (p *effectBlockingPlayer) ChooseMayAbility(string) bool { p.choices++; return p.accept }
func (p *effectBlockingPlayer) ChoosePermanent(candidates []*Permanent, _ string, _ GameReader) *Permanent {
	var chosen *Permanent
	for _, candidate := range candidates {
		if chosen == nil || candidate.Name() == p.name {
			chosen = candidate
		}
		if candidate.Name() == p.name {
			break
		}
	}
	if p.beforeChoice != nil {
		p.beforeChoice()
	}
	return chosen
}

func TestEffectBlockingRestrictionsAndChoice(t *testing.T) {
	for _, scenario := range []string{"ordinary", "decline", "tapped", "flying", "unblockable", "protection", "menace", "cant-block", "limit", "band"} {
		t.Run(scenario, func(t *testing.T) {
			a := &effectBlockingPlayer{Player: NewBasePlayer("A"), accept: scenario != "decline", name: "Second"}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			first := g.PutOnBattlefield(NewCreature("First", "", 2, 10), a.PlayerID())
			second := g.PutOnBattlefield(NewCreature("Second", "", 3, 10), a.PlayerID())
			blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 1, 10), b.PlayerID())
			switch scenario {
			case "tapped":
				blocker.Tapped = true
			case "flying":
				second.GrantBaseAttr(core.Flying)
			case "unblockable":
				second.GrantBaseAttr(core.UnblockableKW)
			case "protection":
				second.Card.AddAbility(ProtectionFromCardType(core.TypeCreature))
				second.RuntimeAbilities = append(second.RuntimeAbilities, ProtectionFromCardType(core.TypeCreature))
			case "menace":
				second.GrantBaseAttr(core.Menace)
			case "cant-block":
				if err := ApplyEffect(g, Until(core.EndOfTurn, CantBlock(This)), blocker.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "limit":
				g.PutOnBattlefield(NewEnchantment("Limit", "", WithAbility(Static(LimitBlockDeclarations(0)))), a.PlayerID())
			}
			for _, p := range []*Permanent{first, second} {
				g.combat.AddAttackerIncarnation(p.ID(), p.incarnationID, b.PlayerID())
				g.combat.GroupFor(p.ID()).Unblocked = true
			}
			if scenario == "band" {
				g.combat.AddBand([]uuid.UUID{first.ID(), second.ID()})
			}
			g.effects.Apply(g)
			if err := ApplyEffect(g, MayHavePermanentBlockAttacker(You, This, MatchingPermanents(AnyPermanent)), blocker.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			want := scenario != "decline"
			if g.combat.IsBlocking(blocker.ID()) != want || g.combat.GroupFor(second.ID()).Blocked != want {
				t.Fatal("wrong resolution choice")
			}
			if g.combat.GroupFor(first.ID()).Blocked != (want && scenario == "band") {
				t.Fatal("wrong selected attacker or band membership")
			}
			if blocker.Tapped != (scenario == "tapped") {
				t.Fatal("blocking changed tapped status")
			}
			if g.combat.BlockedAlone != uuid.Nil {
				t.Fatal("effect changed declaration-only history")
			}
		})
	}
}

func TestEffectBlockingEventsAndHistory(t *testing.T) {
	a := &effectBlockingPlayer{Player: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 10), a.PlayerID())
	blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 1, 10), b.PlayerID())
	g.combat.AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, b.PlayerID())
	g.combat.GroupFor(attacker.ID()).Unblocked = true
	counts := make(map[core.EventType]int)
	options := make([]CardOption, 0, 7)
	for _, kind := range []core.EventType{core.EvtDeclaredBlocker, core.EvtCreatureBlockedBy, core.EvtCombatPair, core.EvtBlockerGroup, core.EvtCreatureBecomesBlocked, core.EvtBlockersDecl, core.EvtAttacksUnblocked} {
		options = append(options, WithAbility(NewTriggered(kind, false).SetCondition(func(event *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool { counts[event.Type]++; return false })))
	}
	g.PutOnBattlefield(NewEnchantment("Watcher", "", options...), a.PlayerID())
	body := MayHavePermanentBlockAttacker(You, This, MatchingPermanents(IsCreature))
	for range 2 {
		if err := ApplyEffect(g, body, blocker.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	for kind, want := range map[core.EventType]int{core.EvtDeclaredBlocker: 1, core.EvtCreatureBlockedBy: 1, core.EvtCombatPair: 2, core.EvtBlockerGroup: 2, core.EvtCreatureBecomesBlocked: 1, core.EvtBlockersDecl: 0, core.EvtAttacksUnblocked: 0} {
		if counts[kind] != want {
			t.Fatalf("event %v: got %d want %d", kind, counts[kind], want)
		}
	}
	if !g.wasBlockedByThisTurn(capturedPermanent{blocker.ID(), blocker.incarnationID}, attacker) {
		t.Fatal("missing block history")
	}
	if err := ApplyEffect(g, RemovePermanentFromCombatAndUnblock(This), blocker.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !g.combat.GroupFor(attacker.ID()).Unblocked {
		t.Fatal("missing sole-blocker history")
	}
	if err := ApplyEffect(g, body, blocker.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if counts[core.EvtDeclaredBlocker] != 2 || counts[core.EvtCreatureBecomesBlocked] != 2 {
		t.Fatal("reblocking did not create new transitions")
	}
}

func TestEffectBlockingIdentityAndClone(t *testing.T) {
	for _, change := range []string{"none", "blocker-return", "blocker-phase", "attacker-return", "attacker-phase", "control", "clone", "invalid-choice"} {
		t.Run(change, func(t *testing.T) {
			a := &effectBlockingPlayer{Player: NewBasePlayer("A"), accept: true}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 10), a.PlayerID())
			blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 1, 10), b.PlayerID())
			g.combat.AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, b.PlayerID())
			binding := BindPermanent("blocker")
			ctx := &EffectContext{Game: g, SourceID: blocker.ID(), Controller: a.PlayerID()}
			if err := Snapshot(This, binding).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "blocker-return":
				g.RemoveFromBattlefield(blocker)
				g.PutOnBattlefield(blocker.Card, b.PlayerID())
			case "blocker-phase":
				g.PhaseOut(blocker.ID())
			case "attacker-return":
				g.RemoveFromBattlefield(attacker)
				g.PutOnBattlefield(attacker.Card, a.PlayerID())
			case "attacker-phase":
				g.PhaseOut(attacker.ID())
			case "control":
				ApplyEffect(g, Until(core.Indefinite, ChangeController(This, You)), blocker.ID(), a.PlayerID(), nil)
			case "invalid-choice":
				a.beforeChoice = func() { g.PhaseOut(attacker.ID()) }
			}
			original := g
			if change == "clone" {
				g = g.Clone()
				g.SetPlayerAt(0, &effectBlockingPlayer{Player: g.GetPlayer(a.PlayerID()), accept: true})
				ctx.Game = g
			}
			if err := MayHavePermanentBlockAttacker(You, From(binding), MatchingPermanents(AnyPermanent)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			want := change == "none" || change == "clone"
			if g.combat.IsBlocking(blocker.ID()) != want {
				t.Fatal("lost exact identity or defender check")
			}
			if change == "clone" && original.combat.IsBlocking(blocker.ID()) {
				t.Fatal("clone changed original combat")
			}
		})
	}
}

func TestEffectBlockingDefinitionAndWaiting(t *testing.T) {
	for _, effect := range []ResolvingEffect{
		MayHavePermanentBlockAttacker(nil, This, MatchingPermanents(IsCreature)),
		MayHavePermanentBlockAttacker(You, nil, MatchingPermanents(IsCreature)),
		MayHavePermanentBlockAttacker(You, This, nil),
		MayHavePermanentBlockAttacker(You, From(BindPermanent("missing")), MatchingPermanents(IsCreature)),
	} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), effect)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("accepted invalid declaration")
		}
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 1, 10), a.PlayerID())
	if err := ApplyEffect(g, AtNext(EachEndStep, MayHavePermanentBlockAttacker(You, This, MatchingPermanents(IsCreature))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	saved := g.CaptureDamageSource(p.ID())
	g.RemoveFromBattlefield(p)
	g.zones.ClearLKI()
	got := g.WaitingDelayedPermanentSources()
	if len(got) != 1 || !got[0].SameObject(saved) {
		t.Fatal("waiting query lost explicit source")
	}
}

func TestEffectBlockingBandEvents(t *testing.T) {
	a := &effectBlockingPlayer{Player: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 10), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 10), a.PlayerID())
	blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 1, 10), b.PlayerID())
	for _, p := range []*Permanent{first, second} {
		g.combat.AddAttackerIncarnation(p.ID(), p.incarnationID, b.PlayerID())
	}
	g.combat.AddBand([]uuid.UUID{first.ID(), second.ID()})
	blocks, pairs := 0, 0
	watcher := NewTriggered(core.EvtDeclaredBlocker, false).SetCondition(func(event *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		pairs++
		if event.Flag {
			blocks++
		}
		for _, attacker := range []*Permanent{first, second} {
			if !g.combat.GroupFor(attacker.ID()).Blocked || !g.wasBlockedByThisTurn(capturedPermanent{blocker.ID(), blocker.incarnationID}, attacker) {
				t.Fatal("event observed partial band membership or history")
			}
		}
		return false
	})
	g.PutOnBattlefield(NewEnchantment("Watcher", "", WithAbility(watcher)), a.PlayerID())
	if err := ApplyEffect(g, MayHavePermanentBlockAttacker(You, This, MatchingPermanents(IsCreature)), blocker.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if blocks != 1 || pairs != 2 {
		t.Fatal("wrong block and pair events for a band")
	}
}

func TestEffectBlockingBatch(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 10), a.PlayerID())
		first := g.PutOnBattlefield(NewCreature("First", "", 1, 10), b.PlayerID())
		second := g.PutOnBattlefield(NewCreature("Second", "", 1, 10), b.PlayerID())
		g.combat.AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, b.PlayerID())
		assignments := []effectBlockAssignment{
			{capturedPermanent{first.ID(), first.incarnationID}, capturedPermanent{attacker.ID(), attacker.incarnationID}},
			{capturedPermanent{second.ID(), second.incarnationID}, capturedPermanent{attacker.ID(), attacker.incarnationID}},
		}
		if invalid {
			g.RemoveFromBattlefield(second)
			g.PutOnBattlefield(second.Card, b.PlayerID())
		}
		events := 0
		watcher := NewTriggered(core.EvtDeclaredBlocker, false).SetCondition(func(_ *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
			events++
			if len(g.combat.GroupFor(attacker.ID()).BlockerIDs) != 2 {
				t.Fatal("event observed partial batch")
			}
			return false
		})
		g.PutOnBattlefield(NewEnchantment("Watcher", "", WithAbility(watcher)), a.PlayerID())
		if g.addEffectBlockers(assignments) == invalid {
			t.Fatal("wrong batch result")
		}
		want := 2
		if invalid {
			want = 0
		}
		if len(g.combat.GroupFor(attacker.ID()).BlockerIDs) != want || events != want {
			t.Fatal("partial membership or wrong events")
		}
	}
}
