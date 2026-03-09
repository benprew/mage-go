// game.js — WASM bridge and game UI logic for Mage

// Action types (match interactive.ActionType)
const ActionPass = 0;
const ActionPlayLand = 1;
const ActionCastSpell = 2;
const ActionActivateAbility = 3;
const ActionSelectAttackers = 4;
const ActionSelectBlockers = 5;
const ActionUndo = 6;

// Prompt types (match interactive.PromptType)
const PromptNone = 0;
const PromptMainPhaseAction = 1;
const PromptPriority = 2;
const PromptDeclareAttackers = 3;
const PromptDeclareBlockers = 4;
const PromptChooseTargets = 5;

// Choice types (match interactive.ChoiceType)
const ChoicePermanent = 0;
const ChoiceCardsFromHand = 1;
const ChoiceManaColor = 2;
const ChoiceCardFromLibrary = 3;
const ChoiceMay = 4;
const ChoiceMode = 5;
const ChoiceNumber = 6;

// Current game state
let currentState = null;
let currentOptions = [];
let currentPrompt = PromptNone;
let selectedAttackers = new Set();
let blockerAssignments = [];
let lastLogLength = 0;

// Auto-pass: when the only option is "Pass", auto-send it after a brief delay
let autoPassEnabled = true;
let autoPassTimer = null;
const AUTO_PASS_DELAY = 150; // ms — just enough to update the UI

// Preset decks
const PRESETS = {
    "WG Aggro": [
        ...Array(9).fill("Plains"),
        ...Array(7).fill("Forest"),
        ...Array(4).fill("Savannah Lions"),
        ...Array(3).fill("White Knight"),
        ...Array(4).fill("Llanowar Elves"),
        ...Array(3).fill("Grizzly Bears"),
        ...Array(2).fill("Serra Angel"),
        ...Array(4).fill("Healing Salve"),
        ...Array(2).fill("Giant Growth"),
        ...Array(2).fill("Swords to Plowshares"),
    ],
    "RB Burn": [
        ...Array(8).fill("Mountain"),
        ...Array(8).fill("Swamp"),
        ...Array(4).fill("Lightning Bolt"),
        ...Array(4).fill("Terror"),
        ...Array(4).fill("Black Knight"),
        ...Array(4).fill("Ironclaw Orcs"),
        ...Array(4).fill("Hill Giant"),
        ...Array(4).fill("Hypnotic Specter"),
    ],
    "Mono Red": [
        ...Array(16).fill("Mountain"),
        ...Array(4).fill("Lightning Bolt"),
        ...Array(4).fill("Ironclaw Orcs"),
        ...Array(4).fill("Hill Giant"),
        ...Array(4).fill("Gray Ogre"),
        ...Array(4).fill("Mons's Goblin Raiders"),
        ...Array(4).fill("Shatter"),
    ],
    "Mono White": [
        ...Array(16).fill("Plains"),
        ...Array(4).fill("Savannah Lions"),
        ...Array(4).fill("White Knight"),
        ...Array(2).fill("Serra Angel"),
        ...Array(4).fill("Swords to Plowshares"),
        ...Array(4).fill("Healing Salve"),
        ...Array(2).fill("Holy Strength"),
        ...Array(4).fill("Mesa Pegasus"),
    ],
};

// ---- WASM Init ----

async function initWasm() {
    const go = new Go();
    const result = await WebAssembly.instantiateStreaming(fetch("mage.wasm"), go.importObject);
    go.run(result.instance);

    document.getElementById("loading").style.display = "none";
    document.getElementById("deck-select").style.display = "block";
    setupPresets();
}

function setupPresets() {
    const container = document.getElementById("presets");
    for (const [name, cards] of Object.entries(PRESETS)) {
        const btn = document.createElement("button");
        btn.className = "preset-btn";
        btn.textContent = name + " (" + cards.length + ")";
        btn.onclick = () => {
            const counts = {};
            cards.forEach(c => { counts[c] = (counts[c] || 0) + 1; });
            const lines = Object.entries(counts).map(([name, count]) =>
                count > 1 ? count + " " + name : name
            );
            document.getElementById("deck-textarea").value = lines.join("\n");
        };
        container.appendChild(btn);
    }

    // Populate AI deck dropdown with the same presets.
    const aiDeckSelect = document.getElementById("ai-deck");
    for (const name of Object.keys(PRESETS)) {
        const opt = document.createElement("option");
        opt.value = name;
        opt.textContent = name;
        // Default AI deck to RB Burn if available.
        if (name === "RB Burn") opt.selected = true;
        aiDeckSelect.appendChild(opt);
    }
}

function parseDeckList(text) {
    const cards = [];
    for (let line of text.split("\n")) {
        line = line.trim();
        if (!line || line.startsWith("#") || line.startsWith("//")) continue;
        const match = line.match(/^(\d+)\s+(.+)$/);
        if (match) {
            const count = parseInt(match[1]);
            const name = match[2].trim();
            for (let i = 0; i < count; i++) cards.push(name);
        } else {
            cards.push(line);
        }
    }
    return cards;
}

function startGame() {
    const text = document.getElementById("deck-textarea").value;
    const cards = parseDeckList(text);
    if (cards.length < 7) {
        document.getElementById("deck-error").textContent = "Deck needs at least 7 cards.";
        return;
    }

    // AI configuration.
    const aiPersonality = document.getElementById("ai-personality").value;
    const aiMode = document.getElementById("ai-mode").value;
    const aiDeckName = document.getElementById("ai-deck").value;
    const aiDeckCards = PRESETS[aiDeckName] || PRESETS["RB Burn"];
    const aiDeckJSON = JSON.stringify(aiDeckCards);

    const deckJSON = JSON.stringify(cards);
    const err = mageStartGame(deckJSON, onGameMsg, onChoiceReq, aiDeckJSON, aiPersonality, aiMode);
    if (err) {
        document.getElementById("deck-error").textContent = String(err);
        return;
    }

    document.getElementById("deck-select").style.display = "none";
    document.getElementById("game-ui").style.display = "flex";
}

// ---- Message Handlers ----

function onGameMsg(jsonStr) {
    const msg = JSON.parse(jsonStr);
    currentState = msg.state;
    currentOptions = msg.options || [];
    currentPrompt = msg.prompt;

    renderGame(msg);

    // Auto-pass: if only option is Pass (priority prompt, not main phase), auto-send
    if (autoPassEnabled && !msg.gameOver && shouldAutoPass(msg)) {
        if (autoPassTimer) clearTimeout(autoPassTimer);
        autoPassTimer = setTimeout(() => {
            mageSendAction(JSON.stringify({ type: ActionPass }));
        }, AUTO_PASS_DELAY);
    }
}

function shouldAutoPass(msg) {
    // Only auto-pass priority prompts (not main phase — player might want to act)
    if (msg.prompt !== PromptPriority) return false;
    // Only if the sole option is Pass
    if (!msg.options || msg.options.length !== 1) return false;
    return msg.options[0].type === ActionPass;
}

function onChoiceReq(jsonStr) {
    const req = JSON.parse(jsonStr);
    showChoiceModal(req);
}

// ---- Rendering ----

function renderGame(msg) {
    if (!msg.state) return;
    const s = msg.state;

    // Top bar: turn, phase
    document.getElementById("turn-info").textContent = "Turn " + s.Turn;
    document.getElementById("phase-info").textContent = s.Step;

    // Status banner — the main "what's happening" indicator
    renderStatusBanner(msg);

    // Life totals — big and prominent
    document.getElementById("you-name").textContent = s.You.Name;
    document.getElementById("opp-name").textContent = s.Opponent.Name;
    const youLife = document.getElementById("you-life");
    const oppLife = document.getElementById("opp-life");
    youLife.textContent = s.You.Life;
    oppLife.textContent = s.Opponent.Life;
    youLife.className = "life-number" + (s.You.Life <= 5 ? " danger" : "");
    oppLife.className = "life-number" + (s.Opponent.Life <= 5 ? " danger" : "");

    // Stat pills
    document.getElementById("lib-count").textContent = "Lib: " + s.You.LibraryCount;
    document.getElementById("gy-count").textContent = "GY: " + s.You.GraveyardCount;
    document.getElementById("opp-hand-count").textContent = "Opp Hand: " + s.Opponent.HandCount;

    // Mana pool
    renderManaPool(s.You.ManaPool);

    // Battlefields
    renderPermanents("opp-perms", s.Opponent.Battlefield, false);
    renderPermanents("my-perms", s.You.Battlefield, true);

    // Hand
    renderHand(s.You.Hand);

    // Stack
    renderStack(s.StackItems);

    // Log — append only new entries for performance
    renderLog(msg.log);

    // Actions
    renderActions(msg);
}

function renderStatusBanner(msg) {
    const s = msg.state;
    const whoEl = document.getElementById("status-who");
    const textEl = document.getElementById("status-text");
    const phaseEl = document.getElementById("status-phase");

    const isYourTurn = s.ActivePlayer === s.You.Name;
    const phase = s.Step || "";

    // Who badge
    if (isYourTurn) {
        whoEl.textContent = "Your Turn";
        whoEl.className = "status-who your-turn";
    } else {
        whoEl.textContent = "AI's Turn";
        whoEl.className = "status-who ai-turn";
    }

    // Phase display
    phaseEl.textContent = phase;

    // Context-aware status text
    if (msg.gameOver) {
        textEl.textContent = msg.winner ? msg.winner + " wins!" : "Draw!";
        textEl.className = "status-text";
        return;
    }

    const opts = msg.options || [];
    const onlyPass = opts.length === 1 && opts[0].type === ActionPass;
    const hasStack = s.StackItems && s.StackItems.length > 0;
    const topOfStack = hasStack ? s.StackItems[0].Name : "";

    switch (msg.prompt) {
        case PromptNone:
            if (!isYourTurn) {
                textEl.textContent = "AI is thinking...";
                textEl.className = "status-text muted";
            } else {
                textEl.textContent = "Waiting...";
                textEl.className = "status-text muted";
            }
            break;

        case PromptMainPhaseAction:
            textEl.textContent = "Play a land, cast a spell, or pass.";
            textEl.className = "status-text";
            break;

        case PromptPriority:
            if (onlyPass && !hasStack) {
                textEl.textContent = "Passing priority...";
                textEl.className = "status-text muted";
            } else if (onlyPass && hasStack) {
                textEl.textContent = "Resolving: " + topOfStack;
                textEl.className = "status-text muted";
            } else if (hasStack) {
                textEl.textContent = topOfStack + " is on the stack. Respond or pass.";
                textEl.className = "status-text";
            } else {
                textEl.textContent = "You have priority. Cast an instant or pass.";
                textEl.className = "status-text";
            }
            break;

        case PromptDeclareAttackers:
            textEl.textContent = "Choose which creatures attack.";
            textEl.className = "status-text";
            break;

        case PromptDeclareBlockers: {
            const numAttackers = s.Opponent.Battlefield
                ? s.Opponent.Battlefield.filter(p => p.Attacking).length
                : 0;
            textEl.textContent = numAttackers + " creature" + (numAttackers !== 1 ? "s" : "") + " attacking. Assign blockers.";
            textEl.className = "status-text";
            break;
        }

        default:
            textEl.textContent = "";
            textEl.className = "status-text muted";
    }
}

function renderManaPool(mp) {
    if (!mp) return;
    const el = document.getElementById("mana-pool");
    const parts = [];
    if (mp.White) parts.push('<span class="mana-w">W:' + mp.White + '</span>');
    if (mp.Blue) parts.push('<span class="mana-u">U:' + mp.Blue + '</span>');
    if (mp.Black) parts.push('<span class="mana-b">B:' + mp.Black + '</span>');
    if (mp.Red) parts.push('<span class="mana-r">R:' + mp.Red + '</span>');
    if (mp.Green) parts.push('<span class="mana-g">G:' + mp.Green + '</span>');
    if (mp.Colorless) parts.push('<span class="mana-c">C:' + mp.Colorless + '</span>');
    el.innerHTML = parts.length
        ? '<span class="mana-label">Mana </span>' + parts.join(" ")
        : "";
}

function renderPermanents(containerId, perms, isYours) {
    const container = document.getElementById(containerId);
    container.innerHTML = "";
    if (!perms) return;

    // Group by type: lands first, then creatures, then other
    const lands = [];
    const creatures = [];
    const other = [];
    for (const p of perms) {
        if (p.IsLand) lands.push(p);
        else if (p.IsCreature) creatures.push(p);
        else other.push(p);
    }

    const renderGroup = (group) => {
        for (const p of group) {
            const card = document.createElement("div");
            card.className = "card";
            if (p.Tapped) card.classList.add("tapped");
            if (p.Attacking) card.classList.add("attacking");
            if (p.SummonSick && p.IsCreature) card.classList.add("summon-sick");

            let html = '<span class="card-name">' + escHtml(p.Name) + '</span>';
            if (p.ManaCost) html += ' <span class="card-cost">' + escHtml(p.ManaCost) + '</span>';
            if (p.IsCreature) html += '<br><span class="card-pt">' + p.Power + '/' + p.Toughness + '</span>';
            if (p.Keywords && p.Keywords.length) {
                html += '<br><span class="card-keywords">' + escHtml(p.Keywords.join(", ")) + '</span>';
            }
            if (p.Counters) {
                const cstrs = Object.entries(p.Counters).filter(([,v]) => v > 0).map(([k,v]) => k + ":" + v);
                if (cstrs.length) html += '<br><span class="card-counters">' + escHtml(cstrs.join(", ")) + '</span>';
            }
            // Show type line for non-obvious permanents
            if (!p.IsCreature && !p.IsLand) {
                const typeLine = [p.Types, p.SubTypes].filter(Boolean).join(" — ");
                if (typeLine) html += '<br><span class="card-types">' + escHtml(typeLine) + '</span>';
            }
            card.innerHTML = html;
            card.dataset.id = p.ID;

            // Hover tooltip
            card.addEventListener("mouseenter", (e) => showTooltip(e, p));
            card.addEventListener("mouseleave", hideTooltip);
            card.addEventListener("mousemove", moveTooltip);

            container.appendChild(card);
        }
    };

    renderGroup(creatures);
    renderGroup(other);
    renderGroup(lands);
}

function renderHand(hand) {
    const container = document.getElementById("hand-cards");
    container.innerHTML = "";
    document.getElementById("hand-count").textContent = hand ? hand.length : 0;
    if (!hand) return;
    for (const c of hand) {
        const card = document.createElement("div");
        card.className = "card hand-card";
        let html = '<span class="card-name">' + escHtml(c.Name) + '</span>';
        if (c.ManaCost) html += ' <span class="card-cost">' + escHtml(c.ManaCost) + '</span>';
        if (c.Types && c.Types.includes("Creature")) {
            html += '<br><span class="card-pt">' + c.Power + '/' + c.Toughness + '</span>';
        }
        card.innerHTML = html;
        card.dataset.id = c.ID;

        // Hover tooltip for hand cards
        card.addEventListener("mouseenter", (e) => showTooltip(e, c));
        card.addEventListener("mouseleave", hideTooltip);
        card.addEventListener("mousemove", moveTooltip);

        container.appendChild(card);
    }
}

function renderStack(items) {
    const zone = document.getElementById("stack-zone");
    const container = document.getElementById("stack-items");
    if (!items || items.length === 0) {
        zone.style.display = "none";
        return;
    }
    zone.style.display = "block";
    container.innerHTML = "";
    for (const item of items) {
        const el = document.createElement("div");
        el.style.cssText = "padding: 3px 8px; color: #d4c5a0; font-size: 12px; font-family: Georgia, serif;";
        let text = item.Name;
        if (item.Controller) text += " (" + item.Controller + ")";
        if (item.Targets && item.Targets.length) text += " \u2192 " + item.Targets.join(", ");
        if (item.IsAbility) {
            el.style.color = "#a0c0d8";
            text = "\u2022 " + text;
        }
        el.textContent = text;
        container.appendChild(el);
    }
}

function renderLog(log) {
    const area = document.getElementById("log-area");
    if (!log) return;

    // Full re-render (log is capped at 100 entries server-side anyway)
    area.innerHTML = "";
    for (const entry of log) {
        const div = document.createElement("div");
        div.className = "log-entry";
        // Colorize certain log entries
        if (entry.includes("attacks") || entry.includes("blocks") || entry.includes("damage")) {
            div.classList.add("log-combat");
        } else if (entry.includes("casts") || entry.includes("plays")) {
            div.classList.add("log-cast");
        }
        div.textContent = entry;
        area.appendChild(div);
    }
    area.scrollTop = area.scrollHeight;
    lastLogLength = log.length;
}

function renderActions(msg) {
    const bar = document.getElementById("action-bar");
    bar.innerHTML = "";

    if (msg.gameOver) {
        const overlay = document.getElementById("game-over-overlay");
        overlay.classList.add("visible");
        document.getElementById("game-over-text").textContent =
            msg.winner ? msg.winner + " Wins" : "Draw";
        document.getElementById("game-over-detail").textContent =
            msg.winner === currentState.You.Name
                ? "The battle is won."
                : "You have fallen.";
        return;
    }

    if (currentPrompt === PromptNone || shouldAutoPass(msg)) {
        // No actions to show — status banner explains what's happening
        return;
    }

    // Compact label — the banner already explains the situation
    const label = document.createElement("div");
    label.className = "prompt-label";
    label.textContent = "Actions";
    bar.appendChild(label);

    if (currentPrompt === PromptDeclareAttackers) {
        renderAttackerSelection(bar);
        return;
    }

    if (currentPrompt === PromptDeclareBlockers) {
        renderBlockerSelection(bar);
        return;
    }

    // Standard action list
    for (const opt of currentOptions) {
        const btn = document.createElement("button");
        btn.className = "action-btn";
        if (opt.type === ActionPass) btn.classList.add("pass-btn");
        btn.textContent = opt.label;
        btn.onclick = () => handleAction(opt);
        bar.appendChild(btn);
    }
}

function renderAttackerSelection(bar) {
    selectedAttackers = new Set();
    if (!currentState || !currentState.You) return;

    for (const opt of currentOptions) {
        if (opt.type === ActionPass) continue;
        const btn = document.createElement("button");
        btn.className = "action-btn";
        btn.textContent = opt.label;
        btn.dataset.permId = opt.permanentID;
        btn.onclick = () => {
            const id = opt.permanentID;
            if (selectedAttackers.has(id)) {
                selectedAttackers.delete(id);
                btn.classList.remove("attacker-selected");
            } else {
                selectedAttackers.add(id);
                btn.classList.add("attacker-selected");
            }
        };
        bar.appendChild(btn);
    }

    // Confirm + Don't Attack buttons
    const confirmBtn = document.createElement("button");
    confirmBtn.className = "action-btn pass-btn";
    confirmBtn.textContent = "Confirm Attackers";
    confirmBtn.onclick = () => {
        mageSendAction(JSON.stringify({
            type: ActionSelectAttackers,
            attackers: Array.from(selectedAttackers)
        }));
    };
    bar.appendChild(confirmBtn);

    const skipBtn = document.createElement("button");
    skipBtn.className = "action-btn";
    skipBtn.textContent = "Don't Attack";
    skipBtn.style.color = "#6b6b7b";
    skipBtn.onclick = () => {
        mageSendAction(JSON.stringify({
            type: ActionSelectAttackers,
            attackers: []
        }));
    };
    bar.appendChild(skipBtn);
}

function renderBlockerSelection(bar) {
    blockerAssignments = [];

    const attackers = currentState.Opponent.Battlefield.filter(p => p.Attacking);
    if (attackers.length === 0) {
        mageSendAction(JSON.stringify({ type: ActionSelectBlockers, blockers: [] }));
        return;
    }

    const blockerOpts = currentOptions.filter(o => o.type !== ActionPass);
    if (blockerOpts.length === 0) {
        const btn = document.createElement("button");
        btn.className = "action-btn pass-btn";
        btn.textContent = "No Blockers Available";
        btn.onclick = () => {
            mageSendAction(JSON.stringify({ type: ActionSelectBlockers, blockers: [] }));
        };
        bar.appendChild(btn);
        return;
    }

    const statusDiv = document.createElement("div");
    statusDiv.style.cssText = "color: #6b6b7b; font-size: 11px; margin: 4px 0;";
    statusDiv.id = "blocker-status";
    bar.appendChild(statusDiv);

    for (const opt of blockerOpts) {
        const blockerBtn = document.createElement("button");
        blockerBtn.className = "action-btn";
        blockerBtn.textContent = "Assign " + opt.label + " \u2192 ...";
        blockerBtn.onclick = () => {
            showBlockerTargetModal(opt, attackers);
        };
        bar.appendChild(blockerBtn);
    }

    const doneBtn = document.createElement("button");
    doneBtn.className = "action-btn pass-btn";
    doneBtn.textContent = "Done Assigning Blockers";
    doneBtn.onclick = () => {
        mageSendAction(JSON.stringify({
            type: ActionSelectBlockers,
            blockers: blockerAssignments
        }));
    };
    bar.appendChild(doneBtn);

    const skipBtn = document.createElement("button");
    skipBtn.className = "action-btn";
    skipBtn.textContent = "Don't Block";
    skipBtn.style.color = "#6b6b7b";
    skipBtn.onclick = () => {
        mageSendAction(JSON.stringify({ type: ActionSelectBlockers, blockers: [] }));
    };
    bar.appendChild(skipBtn);

    updateBlockerStatus();
}

function showBlockerTargetModal(blockerOpt, attackers) {
    const container = document.getElementById("modal-container");
    const overlay = document.createElement("div");
    overlay.className = "modal-overlay";
    const modal = document.createElement("div");
    modal.className = "modal";
    modal.innerHTML = '<h3>Block with ' + escHtml(blockerOpt.label) + '</h3>';

    for (const atk of attackers) {
        const btn = document.createElement("button");
        btn.className = "modal-option";
        let text = atk.Name;
        if (atk.IsCreature) text += " " + atk.Power + "/" + atk.Toughness;
        btn.textContent = text;
        btn.onclick = () => {
            blockerAssignments.push({
                blockerID: blockerOpt.permanentID,
                attackerID: atk.ID
            });
            container.innerHTML = "";
            updateBlockerStatus();
        };
        modal.appendChild(btn);
    }

    const cancelBtn = document.createElement("button");
    cancelBtn.className = "modal-option";
    cancelBtn.textContent = "Cancel";
    cancelBtn.style.borderColor = "#8b2020";
    cancelBtn.onclick = () => { container.innerHTML = ""; };
    modal.appendChild(cancelBtn);

    overlay.appendChild(modal);
    overlay.onclick = (e) => { if (e.target === overlay) container.innerHTML = ""; };
    container.innerHTML = "";
    container.appendChild(overlay);
}

function updateBlockerStatus() {
    const el = document.getElementById("blocker-status");
    if (!el) return;
    if (blockerAssignments.length === 0) {
        el.textContent = "No blockers assigned.";
    } else {
        el.textContent = blockerAssignments.length + " blocker(s) assigned.";
    }
}

// ---- Action Handling ----

function handleAction(opt) {
    switch (opt.type) {
        case ActionPass:
            mageSendAction(JSON.stringify({ type: ActionPass }));
            break;
        case ActionPlayLand:
            mageSendAction(JSON.stringify({
                type: ActionPlayLand,
                cardID: opt.cardID,
                cardName: opt.cardName
            }));
            break;
        case ActionCastSpell:
            if (opt.needsTarget) {
                showTargetModal(opt);
            } else {
                mageSendAction(JSON.stringify({
                    type: ActionCastSpell,
                    cardID: opt.cardID,
                    cardName: opt.cardName
                }));
            }
            break;
        case ActionActivateAbility:
            if (opt.needsTarget) {
                showTargetModal(opt);
            } else {
                mageSendAction(JSON.stringify({
                    type: ActionActivateAbility,
                    permanentID: opt.permanentID,
                    abilityIndex: opt.abilityIndex,
                    cardName: opt.cardName
                }));
            }
            break;
    }
}

function showTargetModal(opt) {
    const targets = [];
    if (currentState) {
        if (currentState.You.Battlefield) {
            for (const p of currentState.You.Battlefield) {
                let label = "Your " + p.Name;
                if (p.IsCreature) label += " " + p.Power + "/" + p.Toughness;
                if (p.Tapped) label += " (T)";
                targets.push({ id: p.ID, label });
            }
        }
        if (currentState.Opponent.Battlefield) {
            for (const p of currentState.Opponent.Battlefield) {
                let label = "Opp " + p.Name;
                if (p.IsCreature) label += " " + p.Power + "/" + p.Toughness;
                if (p.Tapped) label += " (T)";
                targets.push({ id: p.ID, label });
            }
        }
        targets.push({ id: currentState.You.ID, label: currentState.You.Name + " (Life: " + currentState.You.Life + ")" });
        targets.push({ id: currentState.Opponent.ID, label: currentState.Opponent.Name + " (Life: " + currentState.Opponent.Life + ")" });
    }

    const container = document.getElementById("modal-container");
    const overlay = document.createElement("div");
    overlay.className = "modal-overlay";
    const modal = document.createElement("div");
    modal.className = "modal";
    modal.innerHTML = '<h3>Target for ' + escHtml(opt.label) + '</h3>';

    for (const t of targets) {
        const btn = document.createElement("button");
        btn.className = "modal-option";
        btn.textContent = t.label;
        btn.onclick = () => {
            container.innerHTML = "";
            mageSendAction(JSON.stringify({
                type: opt.type,
                cardID: opt.cardID,
                cardName: opt.cardName,
                permanentID: opt.permanentID,
                abilityIndex: opt.abilityIndex,
                targets: [t.id]
            }));
        };
        modal.appendChild(btn);
    }

    const cancelBtn = document.createElement("button");
    cancelBtn.className = "modal-option";
    cancelBtn.textContent = "Cancel";
    cancelBtn.style.borderColor = "#8b2020";
    cancelBtn.onclick = () => { container.innerHTML = ""; };
    modal.appendChild(cancelBtn);

    overlay.appendChild(modal);
    overlay.onclick = (e) => { if (e.target === overlay) container.innerHTML = ""; };
    container.innerHTML = "";
    container.appendChild(overlay);
}

// ---- Choice Modals ----

function showChoiceModal(req) {
    const container = document.getElementById("modal-container");
    const overlay = document.createElement("div");
    overlay.className = "modal-overlay";
    const modal = document.createElement("div");
    modal.className = "modal";

    const isMulti = req.type === ChoiceCardsFromHand;
    const isMay = req.type === ChoiceMay;

    let title = req.reason || "Choose";
    if (isMulti && req.amount) title += " (select " + req.amount + ")";
    modal.innerHTML = '<h3>' + escHtml(title) + '</h3>';

    if (isMulti) {
        const selected = new Set();
        for (let i = 0; i < req.options.length; i++) {
            const opt = req.options[i];
            const btn = document.createElement("button");
            btn.className = "modal-option";
            btn.textContent = opt.label;
            btn.onclick = () => {
                if (selected.has(i)) {
                    selected.delete(i);
                    btn.classList.remove("selected");
                } else {
                    selected.add(i);
                    btn.classList.add("selected");
                }
            };
            modal.appendChild(btn);
        }

        const submitBtn = document.createElement("button");
        submitBtn.className = "modal-submit";
        submitBtn.textContent = "Confirm";
        submitBtn.onclick = () => {
            const ids = [];
            for (const idx of selected) {
                if (req.options[idx].id && req.options[idx].id !== "00000000-0000-0000-0000-000000000000") {
                    ids.push(req.options[idx].id);
                }
            }
            container.innerHTML = "";
            mageSendChoice(JSON.stringify({ selectedIDs: ids }));
        };
        modal.appendChild(submitBtn);
    } else if (isMay) {
        for (let i = 0; i < req.options.length; i++) {
            const opt = req.options[i];
            const btn = document.createElement("button");
            btn.className = "modal-option";
            btn.textContent = opt.label;
            btn.onclick = () => {
                container.innerHTML = "";
                mageSendChoice(JSON.stringify({ accepted: i === 0 }));
            };
            modal.appendChild(btn);
        }
    } else if (req.type === ChoiceManaColor) {
        for (let i = 0; i < req.options.length; i++) {
            const opt = req.options[i];
            const btn = document.createElement("button");
            btn.className = "modal-option";
            btn.textContent = opt.label;
            btn.onclick = () => {
                container.innerHTML = "";
                mageSendChoice(JSON.stringify({ selectedColor: opt.color }));
            };
            modal.appendChild(btn);
        }
    } else if (req.type === ChoiceMode || req.type === ChoiceNumber) {
        for (let i = 0; i < req.options.length; i++) {
            const opt = req.options[i];
            const btn = document.createElement("button");
            btn.className = "modal-option";
            btn.textContent = opt.label;
            btn.onclick = () => {
                container.innerHTML = "";
                mageSendChoice(JSON.stringify({ selectedIndex: i }));
            };
            modal.appendChild(btn);
        }
    } else {
        // Single select (permanent, card from library)
        for (let i = 0; i < req.options.length; i++) {
            const opt = req.options[i];
            const btn = document.createElement("button");
            btn.className = "modal-option";
            btn.textContent = opt.label;
            btn.onclick = () => {
                container.innerHTML = "";
                const ids = [];
                if (opt.id && opt.id !== "00000000-0000-0000-0000-000000000000") {
                    ids.push(opt.id);
                }
                mageSendChoice(JSON.stringify({ selectedIDs: ids }));
            };
            modal.appendChild(btn);
        }
    }

    overlay.appendChild(modal);
    container.innerHTML = "";
    container.appendChild(overlay);
}

// ---- Tooltip ----

function showTooltip(e, cardData) {
    const tt = document.getElementById("card-tooltip");
    tt.querySelector(".tt-name").textContent = cardData.Name || "";
    const typeLine = [cardData.Types, cardData.SubTypes].filter(Boolean).join(" \u2014 ");
    tt.querySelector(".tt-type").textContent = (cardData.ManaCost ? cardData.ManaCost + "  " : "") + typeLine;
    tt.querySelector(".tt-rules").textContent = cardData.RulesText || "";

    const ptEl = tt.querySelector(".tt-pt");
    if (cardData.IsCreature || (cardData.Types && cardData.Types.includes("Creature"))) {
        ptEl.textContent = cardData.Power + "/" + cardData.Toughness;
        ptEl.style.display = "block";
    } else {
        ptEl.style.display = "none";
    }

    // Add status info
    const statusParts = [];
    if (cardData.Tapped) statusParts.push("TAPPED");
    if (cardData.Attacking) statusParts.push("ATTACKING");
    if (cardData.SummonSick) statusParts.push("Summoning Sickness");
    if (cardData.Keywords && cardData.Keywords.length) statusParts.push(cardData.Keywords.join(", "));
    if (cardData.Counters) {
        const cstrs = Object.entries(cardData.Counters).filter(([,v]) => v > 0).map(([k,v]) => k + ": " + v);
        if (cstrs.length) statusParts.push(cstrs.join(", "));
    }
    if (statusParts.length) {
        tt.querySelector(".tt-rules").textContent =
            (cardData.RulesText || "") + (cardData.RulesText ? "\n" : "") + statusParts.join(" \u2022 ");
    }

    tt.classList.add("visible");
    moveTooltip(e);
}

function moveTooltip(e) {
    const tt = document.getElementById("card-tooltip");
    let x = e.clientX + 12;
    let y = e.clientY + 12;
    // Keep on screen
    if (x + 250 > window.innerWidth) x = e.clientX - 250;
    if (y + 200 > window.innerHeight) y = e.clientY - 200;
    tt.style.left = x + "px";
    tt.style.top = y + "px";
}

function hideTooltip() {
    document.getElementById("card-tooltip").classList.remove("visible");
}

// ---- Utilities ----

function escHtml(s) {
    if (!s) return "";
    return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

// ---- Boot ----
initWasm().catch(err => {
    document.getElementById("loading").innerHTML =
        'FAILED<div class="subtitle">' + escHtml(String(err)) + '</div>';
    console.error(err);
});
