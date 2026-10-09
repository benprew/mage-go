package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSourcePreventionOperationsGameplay(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		gain      bool
	}{
		{"DRK", "Dark Sphere", false}, {"LEA", "Reverse Damage", true}, {"LEB", "Reverse Damage", true},
		{"2ED", "Reverse Damage", true}, {"3ED", "Reverse Damage", true}, {"CED", "Reverse Damage", true}, {"CEI", "Reverse Damage", true},
	} {
		t.Run(tc.set+"/"+tc.name, func(t *testing.T) {
			a := &generalRONSourcePlayer{BasePlayer: mage.NewBasePlayer("A")}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			source := g.PutOnBattlefield(mage.NewCreature("Source", "{R}", 5, 10, mage.WithKeyword(core.Shroud), mage.WithKeyword(core.Lifelink)), b.PlayerID())
			a.wanted = source.ID()
			card := repositoryCard(t, tc.set, tc.name)
			card.SetOwner(a.PlayerID())
			if tc.gain {
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || a.calls != 0 || g.StackSize() != 0 || len(a.Hand()) != 1 {
					t.Fatal("failed payment changed state")
				}
				a.ManaPool().Add(core.White, 3)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
			} else {
				p := g.PutOnBattlefield(card, a.PlayerID())
				g.TapPermanent(p)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil || g.FindPermanent(p.ID()) == nil || a.calls != 0 {
					t.Fatal("failed cost changed state")
				}
				g.UntapPermanent(p)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if g.FindPermanent(p.ID()) != nil || len(a.Graveyard()) != 1 {
					t.Fatal("sacrifice cost was not paid")
				}
			}
			if a.calls != 0 {
				t.Fatal("source choice ran before resolution")
			}
			pending := g.Clone()
			pending.SetPlayerAt(0, &sourceOperationsClonePlayer{Player: pending.GetPlayer(a.PlayerID()), wanted: source.ID()})
			g.ResolveStack()
			if a.calls != 1 || !a.chosen.Valid() {
				t.Fatal("resolution did not choose an untargeted source")
			}
			g.DealDamageToPlayer(a, 0, source.ID())
			g.DealDamageToPlayer(b, 1, source.ID())
			g.DealSimultaneousDamage(mage.PlayerDamageAssignment(a, 2, a.chosen), mage.PlayerDamageAssignment(a, 3, a.chosen))
			want := 17
			wantLifelink := 23
			if tc.gain {
				want = 25
				wantLifelink = 20
			}
			if a.Life() != want || b.Life() != wantLifelink {
				t.Fatalf("first event life=%d/%d want=%d/%d", a.Life(), b.Life(), want, wantLifelink)
			}
			g.DealDamageToPlayer(a, 4, source.ID())
			if a.Life() != want-4 {
				t.Fatal("next source event used consumed prevention")
			}
			pending.ResolveStack()
			pending.DealDamageToPlayer(pending.GetPlayer(a.PlayerID()), 5, source.ID())
			if pending.GetPlayer(a.PlayerID()).Life() != want {
				t.Fatal("pending clone lost independent shield")
			}
		})
	}
}

func TestRONSourcePreventionOperationsInvalid(t *testing.T) {
	const event = `NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Player(You))`
	for _, ops := range []string{
		`[]`, `[PreventThatDamage()]`, `[GainLife(TheAmountOfDamagePreventedThisWay),PreventThatDamage]`,
		`[PreventThatDamage,PreventThatDamage]`, `[PreventSomeOfThatDamage]`,
		`[PreventSomeOfThatDamage(HalfRoundedDown(WouldDealDamage_ThatMuchDamage()))]`,
		`[PreventSomeOfThatDamage(HalfRoundedDown(Integer(2)))]`,
		`[PreventThatDamage,GainLife(Integer(3))]`, `[PreventThatDamage,GainLife(TheAmountOfDamagePreventedThisWay())]`,
		`[PreventThatDamage,GainLife(TheAmountOfDamagePreventedThisWay),GainLife(TheAmountOfDamagePreventedThisWay)]`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([ChooseADamageSource(AnyDamageSource),CreateFuturePreventDamage(` + event + `,` + ops + `)]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", ops)
		}
	}
	for _, action := range []string{`GainLife(TheAmountOfDamagePreventedThisWay)`, `GainLife(WouldDealDamage_ThatMuchDamage)`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+action+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("replacement value escaped its scope")
		}
	}
}

func TestRONSourcePreventionOperationsInventory(t *testing.T) {
	src := mustRONSet(t, "DRK")
	report, err := InspectFS(fstest.MapFS{"sphere.ron": &fstest.MapFile{Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == "WouldDealDamage_ThatMuchDamage" {
			found = true
			if signature.Role != "proposed_damage_value" {
				t.Fatal("wrong proposed amount role", signature.Role)
			}
		}
	}
	if !found {
		t.Fatal("missing proposed amount")
	}
	src = mustRONSet(t, "LEB")
	report, err = InspectFS(fstest.MapFS{"reverse.ron": &fstest.MapFile{Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found = false
	for _, signature := range report.Signatures {
		if signature.Name == "TheAmountOfDamagePreventedThisWay" {
			found = true
			if signature.Role != "prevention_result" {
				t.Fatal("wrong result role", signature.Role)
			}
		}
	}
	if !found {
		t.Fatal("missing actual prevention result")
	}
}

func installRONSourcePrevention(t *testing.T, g *mage.Game, a *generalRONSourcePlayer, name string) {
	t.Helper()
	set := "LEB"
	if name == "Dark Sphere" {
		set = "DRK"
	}
	card := repositoryCard(t, set, name)
	card.SetOwner(a.PlayerID())
	if name == "Dark Sphere" {
		p := g.PutOnBattlefield(card, a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	} else {
		a.AddToHand(card)
		a.ManaPool().Add(core.White, 3)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRONSourcePreventionOperationsLifecycle(t *testing.T) {
	for _, name := range []string{"Dark Sphere", "Reverse Damage"} {
		for _, state := range []string{"depart", "return", "phase", "phase-cycle", "cleanup", "spell"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				a := &generalRONSourcePlayer{BasePlayer: mage.NewBasePlayer("A")}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				g.SetTurn(1)
				card := repositoryCard(t, "LEB", "Hill Giant")
				card.SetOwner(b.PlayerID())
				a.wanted = card.ID()
				var p *mage.Permanent
				if state == "spell" {
					g.SetActivePlayerIndex(1)
					castRONSpell(t, g, b, card)
				} else {
					p = g.PutOnBattlefield(card, b.PlayerID())
				}
				installRONSourcePrevention(t, g, a, name)
				g.ResolveTopOfStack()
				saved := a.chosen
				if state == "spell" {
					g.ResolveTopOfStack()
					p = g.FindPermanent(card.ID())
					saved = g.CaptureDamageSource(p.ID())
				}
				switch state {
				case "depart", "return":
					g.BouncePermanentToHand(p)
					if state == "return" {
						b.RemoveFromHand(card.ID())
						p = g.PutOnBattlefield(card, b.PlayerID())
						g.DealDamageToPlayer(a, 1, p.ID())
						if a.Life() != 19 {
							t.Fatal("returned source inherited shield")
						}
					}
				case "phase":
					g.PhaseOut(p.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "cleanup":
					g.GetEffects().ClearReplacementsEndOfTurn()
				}
				clone := g.Clone()
				for _, game := range []*mage.Game{g, clone} {
					recipient := game.GetPlayer(a.PlayerID())
					before := recipient.Life()
					game.DealDamageToPlayerFromSource(recipient, 5, saved)
					want := before - 3
					if name == "Reverse Damage" {
						want = before + 5
					}
					if state == "cleanup" {
						want = before - 5
					}
					if recipient.Life() != want {
						t.Fatalf("life=%d want=%d", recipient.Life(), want)
					}
					game.DealDamageToPlayerFromSource(recipient, 2, saved)
					if recipient.Life() != want-2 {
						t.Fatal("shield did not expire after prevention")
					}
				}
			})
		}
	}
}

func TestRONSourcePreventionOperationsCopyBindings(t *testing.T) {
	a := &generalRONSourcePlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 5, 10), b.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 5, 10), b.PlayerID())
	installRONSourcePrevention(t, g, a, "Reverse Damage")
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	a.wanted = first.ID()
	g.ResolveTopOfStack()
	a.wanted = second.ID()
	g.ResolveTopOfStack()
	clone := g.Clone()
	g.DealDamageToPlayer(a, 2, first.ID())
	g.DealDamageToPlayer(a, 3, second.ID())
	if a.Life() != 25 {
		t.Fatal("copied spell shared choice or prevention result")
	}
	g.DealDamageToPlayer(a, 1, first.ID())
	g.DealDamageToPlayer(a, 1, second.ID())
	if a.Life() != 23 {
		t.Fatal("copied shields shared consumption")
	}
	recipient := clone.GetPlayer(a.PlayerID())
	clone.DealDamageToPlayer(recipient, 4, second.ID())
	if recipient.Life() != 24 {
		t.Fatal("clone shared installed result state")
	}
}

type sourceOperationsClonePlayer struct {
	mage.Player
	wanted uuid.UUID
}

func (p *sourceOperationsClonePlayer) ChooseDamageSource(sources []mage.DamageSource, _ *mage.Game) int {
	for i, source := range sources {
		if source.ID() == p.wanted {
			return i
		}
	}
	return 0
}

func TestRONSourcePreventionOperationsSupportedPrograms(t *testing.T) {
	for _, tc := range []struct {
		operations string
		want       int
		gain       int
	}{
		{`PreventSomeOfThatDamage(WouldDealDamage_ThatMuchDamage)`, 20, 0},
		{`PreventSomeOfThatDamage(HalfRoundedDown(WouldDealDamage_ThatMuchDamage)),GainLife(TheAmountOfDamagePreventedThisWay)`, 19, 2},
	} {
		src := `Card(Name:"Program",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([ChooseADamageSource(AnyDamageSource),CreateFuturePreventDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Player(You)),[` + tc.operations + `])]))])`
		nodes, err := Parse(strings.NewReader(src), "program.ron")
		if err != nil {
			t.Fatal(err)
		}
		factory, err := Compile(nodes[0])
		if err != nil {
			t.Fatal(err)
		}
		a := &generalRONSourcePlayer{BasePlayer: mage.NewBasePlayer("A")}
		b := mage.NewBasePlayer("B")
		g := mage.NewGame(a, b)
		g.SetStep(core.PrecombatMain)
		p := g.PutOnBattlefield(mage.NewCreature("Source", "", 5, 10), b.PlayerID())
		a.wanted = p.ID()
		card := factory()
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		source := g.CaptureDamageSource(p.ID())
		g.DealSimultaneousDamage(mage.PlayerDamageAssignment(a, 2, source), mage.PlayerDamageAssignment(a, 3, source))
		if a.Life() != tc.want || g.PlayerLifeGainedThisTurn(a.PlayerID()) != tc.gain {
			t.Fatal("program used wrong proposed amount or actual result")
		}
	}
}

func TestRONSourcePreventionOperationsCombat(t *testing.T) {
	for _, name := range []string{"Dark Sphere", "Reverse Damage"} {
		a := &generalRONSourcePlayer{BasePlayer: mage.NewBasePlayer("A")}
		b := mage.NewBasePlayer("B")
		g := mage.NewGame(a, b)
		g.SetStep(core.PrecombatMain)
		g.SetTurn(1)
		p := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 4, 10, mage.WithKeyword(core.DoubleStrike)), b.PlayerID())
		p.RevokeBaseAttr(core.AttrSummonSick)
		a.wanted = p.ID()
		g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{p.ID()})
		g.ExecuteBlockers(nil)
		installRONSourcePrevention(t, g, a, name)
		g.ResolveStack()
		g.ExecuteCombatDamage()
		want := 14
		if name == "Reverse Damage" {
			want = 20
		}
		if a.Life() != want {
			t.Fatal("combat reused shield across strike events", name, a.Life(), want)
		}
	}
}

func TestRONSourcePreventionOperationsExplicitPlayers(t *testing.T) {
	src := `Card(Name:"Player program",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([ChooseAPermanent(IsCardtype(Creature)),ChooseADamageSource(AnyDamageSource),CreateFuturePreventDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Player(ControllerOfPermanent(TheChosenPermanent))),[PreventThatDamage,GainLife(TheAmountOfDamagePreventedThisWay)])]))])`
	nodes, err := Parse(strings.NewReader(src), "players.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	a := &generalRONSourcePlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	p := g.PutOnBattlefield(mage.NewCreature("Source", "", 5, 10), b.PlayerID())
	a.wanted = p.ID()
	card := factory()
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.DealDamageToPlayer(b, 5, p.ID())
	if a.Life() != 25 || b.Life() != 20 {
		t.Fatal("implicit gain must use the saved action controller", a.Life(), b.Life())
	}
}
