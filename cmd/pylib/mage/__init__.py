"""Python bindings for the mage-go engine via cffi.

Build the shared library first:

    go build -buildmode=c-shared -o libmage.dylib ./cmd/pylib   # macOS
    go build -buildmode=c-shared -o libmage.so    ./cmd/pylib   # linux

Then:

    import mage
    deck = {"name": "A", "cards": [{"name": "Mountain", "count": 20},
                                    {"name": "Lightning Bolt", "count": 40}]}
    game = mage.new_game(deck, deck, seed=42, shuffle=True)
    while not game.is_over:
        game.step(pick_action(game.pending))
    print("winner:", game.winner)

Override the library location with the MAGE_LIB env var or mage.load("/path").
"""

from __future__ import annotations

import os
import platform
from typing import Any

import orjson
from cffi import FFI


_CDEF = """
struct MageNewGame_return {
    int64_t r0;
    char *r1;
};
typedef struct {
    int64_t n;
    const int64_t *handles;
    const int64_t *perspective_player_idx;
} MageBatchRequest;
typedef struct {
    int64_t *ready;
    int64_t *game_over;
    int64_t *pending_player_idx;
    int64_t *winner_player_idx;
} MageBatchPollOutputs;
typedef struct {
    int64_t n;
    int64_t max_options;
    int64_t max_targets_per_option;
    const int64_t *handles;
    const int64_t *decision_start;
    const int64_t *decision_count;
    const int64_t *selected_choice_cols;
    const int64_t *may_selected;
} MageStepChoiceRequest;
typedef struct {
    int64_t max_options;
    int64_t max_targets_per_option;
    int64_t max_cached_choices;
    int64_t zone_slot_count;
    int64_t game_info_dim;
    int64_t option_scalar_dim;
    int64_t target_scalar_dim;
    int64_t decision_capacity;
    int64_t emit_render_plan;
    int64_t render_plan_capacity;
    int64_t dedup_card_bodies;
} MageEncodeConfig;
typedef struct {
    int64_t *trace_kind_id;
    int64_t *slot_card_rows;
    float *slot_occupied;
    float *slot_tapped;
    float *game_info;
    int64_t *pending_kind_id;
    int64_t *num_present_options;
    int64_t *option_kind_ids;
    float *option_scalars;
    float *option_mask;
    int64_t *option_ref_slot_idx;
    int64_t *option_ref_card_row;
    float *target_mask;
    int64_t *target_type_ids;
    float *target_scalars;
    float *target_overflow;
    int64_t *target_ref_slot_idx;
    uint8_t *target_ref_is_player;
    uint8_t *target_ref_is_self;
    uint8_t *may_mask;
    int64_t *decision_start;
    int64_t *decision_count;
    int64_t *decision_option_idx;
    int64_t *decision_target_idx;
    uint8_t *decision_mask;
    uint8_t *uses_none_head;
    int32_t *render_plan;
    int64_t *render_plan_lengths;
    int64_t *render_plan_overflow;
} MageEncodeOutputs;
typedef struct {
    int64_t decision_rows_written;
    int64_t error_code;
    char *error_message;
} MageEncodeResult;
struct MageNewGame_return MageNewGame(char *cfgJSON);
char *MageState(int64_t id);
char *MageLegal(int64_t id);
char *MageStep(int64_t id, char *actionJSON);
char *MageSetCardNameRows(char *cardNameRowsJSON);
MageEncodeResult MageBatchPoll(
    MageBatchRequest *req,
    MageBatchPollOutputs *out
);
MageEncodeResult MageBatchStepByChoice(
    MageStepChoiceRequest *req
);
MageEncodeResult MageEncodeBatch(
    MageBatchRequest *req,
    MageEncodeConfig *cfg,
    MageEncodeOutputs *out
);
int64_t MagePendingPlayer(int64_t id);
int64_t MageIsOver(int64_t id);
char *MageWinner(int64_t id);
void MageFree(int64_t id);
void MageFreeString(char *s);
char *MageRegisteredCards(void);
char *MageRegisteredManaCosts(void);

typedef struct {
    int32_t fragment_count;
    const int32_t *structural_tokens;
    const int32_t *structural_offsets;
    int32_t turn_min;
    int32_t turn_max;
    int32_t step_count;
    const int32_t *turn_step_tokens;
    const int32_t *turn_step_offsets;
    int32_t life_min;
    int32_t life_max;
    int32_t owner_count;
    const int32_t *life_owner_tokens;
    const int32_t *life_owner_offsets;
    int32_t ability_min;
    int32_t ability_max;
    const int32_t *ability_tokens;
    const int32_t *ability_offsets;
    int32_t count_min;
    int32_t count_max;
    const int32_t *count_tokens;
    const int32_t *count_offsets;
    int32_t zone_count;
    const int32_t *zone_open_tokens;
    const int32_t *zone_open_offsets;
    const int32_t *zone_close_tokens;
    const int32_t *zone_close_offsets;
    int32_t action_verb_count;
    const int32_t *action_verb_tokens;
    const int32_t *action_verb_offsets;
    int32_t mana_color_count;
    const int32_t *mana_glyph_tokens;
    const int32_t *mana_glyph_offsets;
    int32_t card_ref_count;
    const int32_t *card_ref_ids;
    int32_t pad_id;
    int32_t option_id;
    int32_t target_open_id;
    int32_t target_close_id;
    int32_t tapped_id;
    int32_t untapped_id;
    int32_t card_closer_len;
    const int32_t *card_closer;
    int32_t status_tapped_len;
    const int32_t *status_tapped;
    int32_t status_untapped_len;
    const int32_t *status_untapped;
    int32_t card_row_count;
    const int32_t *card_body_tokens;
    const int64_t *card_body_offsets;
    const int32_t *card_name_tokens;
    const int64_t *card_name_offsets;
    int32_t dict_open_id;
    int32_t dict_close_id;
    int32_t card_open_id;
    const int32_t *dict_entry_ids;
    int32_t self_id;
    int32_t opp_id;
    int32_t stack_open_id;
    int32_t stack_close_id;
    int32_t command_open_id;
    int32_t command_close_id;
    int32_t choose_target_id;
    int32_t choose_block_id;
    int32_t choose_damage_order_id;
    int32_t choose_mode_id;
    int32_t choose_may_id;
    int32_t choose_x_digit_id;
    int32_t choose_mana_source_id;
    int32_t choose_play_id;
    int32_t use_ability_id;
    int32_t chosen_id;
    int32_t yes_id;
    int32_t no_id;
    int32_t none_id;
    int32_t x_end_id;
    int32_t mulligan_id;
    int32_t keep_id;
    int32_t num_count;
    const int32_t *num_ids;
} MageTokenTables;

int32_t MageRegisterTokenTables(MageTokenTables *tables);
char *MageTokenTableSummary(void);
char *MageTokenTableLookup(int32_t kind, int32_t k0, int32_t k1);
char *MageEncodeTimingSummary(int32_t reset);

typedef struct {
    int32_t max_tokens;
    int32_t max_options;
    int32_t max_targets;
    int32_t max_card_refs;
} MageTokenAssemblerConfig;

typedef struct {
    int32_t max_blanks;
    int32_t max_legal_per_blank;
} MageBlankAssemblerConfig;

typedef struct {
    int32_t *token_ids;
    int32_t *cu_seqlens;
    int32_t *seq_lengths;
    int32_t *state_positions;
    int32_t *card_ref_positions;
    int32_t *token_overflow;
} MagePackedTokenAssemblerOutputs;

typedef struct {
    int32_t k_max;
    int32_t v_max;
    int32_t *blank_positions;
    int32_t *blank_kind;
    int32_t *blank_group;
    int32_t *blank_group_kind;
    int32_t *blank_option_index;
    int32_t *blank_legal_ids;
    uint8_t *blank_legal_mask;
    int32_t *blank_overflow;
    int32_t *blank_count;
    int32_t *blank_legal_count;
} MagePackedBlankOutputs;

MageEncodeResult MageEncodeTokensPacked(
    MageBatchRequest *req,
    MageEncodeConfig *cfg,
    MageEncodeOutputs *out,
    MageTokenAssemblerConfig *tok_cfg,
    MagePackedTokenAssemblerOutputs *packed_out,
    MageBlankAssemblerConfig *blank_cfg,
    MagePackedBlankOutputs *blank_out
);
"""


class MageError(RuntimeError):
    pass


_ffi: Any = None  # set by load()
_lib: Any = None  # set by load()
_lib_path_used: str | None = None


def _default_lib_path() -> str:
    override = os.environ.get("MAGE_LIB")
    if override:
        return override
    here = os.path.dirname(os.path.abspath(__file__))
    suffix = {"Darwin": ".dylib", "Linux": ".so", "Windows": ".dll"}.get(
        platform.system(), ".so"
    )
    candidate = os.path.join(here, "libmage" + suffix)
    if os.path.exists(candidate):
        return candidate
    for name in os.listdir(here):
        if name.startswith("libmage.") and name.endswith((".so", ".dylib", ".dll")):
            return os.path.join(here, name)
    raise FileNotFoundError(
        f"libmage not found next to {__file__}; "
        f"build with `go build -buildmode=c-shared -o cmd/pylib/libmage{suffix} ./cmd/pylib` "
        f"or set MAGE_LIB=/path/to/libmage{suffix}"
    )


def load(lib_path: str | None = None) -> None:
    """Explicitly (re)load the shared library. Called lazily by first API use."""
    global _ffi, _lib, _lib_path_used
    ffi = FFI()
    ffi.cdef(_CDEF)
    path = lib_path or _default_lib_path()
    _lib_path_used = os.path.abspath(path)
    _lib = ffi.dlopen(_lib_path_used)
    _ffi = ffi


def _ensure_loaded() -> None:
    if _lib is None:
        load()


def _take_raw(cstr) -> Any:
    if cstr == _ffi.NULL:
        raise MageError("null response from Go")
    try:
        raw = _ffi.string(cstr)
    finally:
        _lib.MageFreeString(cstr)
    return orjson.loads(raw)


def _take(cstr) -> dict[str, Any]:
    resp = _take_raw(cstr)
    if isinstance(resp, dict) and not resp.get("ok", True):
        raise MageError(resp.get("error", "unknown error"))
    return resp


def registered_cards() -> list[str]:
    _ensure_loaded()
    return _take_raw(_lib.MageRegisteredCards())


def registered_mana_costs() -> list[str]:
    _ensure_loaded()
    return _take_raw(_lib.MageRegisteredManaCosts())


def resolved_library_path() -> str:
    _ensure_loaded()
    assert _lib_path_used is not None
    return _lib_path_used


def new_game(
    deck_a: dict,
    deck_b: dict,
    name_a: str = "",
    name_b: str = "",
    seed: int = 0,
    shuffle: bool = True,
    hand_size: int = 7,
) -> "Game":
    _ensure_loaded()
    cfg = orjson.dumps({
        "player_a": deck_a,
        "player_b": deck_b,
        "name_a": name_a,
        "name_b": name_b,
        "seed": seed,
        "shuffle": shuffle,
        "hand_size": hand_size,
    })
    ret = _lib.MageNewGame(_ffi.new("char[]", cfg))
    resp = _take(ret.r1)
    if ret.r0 < 0:
        raise MageError(resp.get("error", "new_game failed"))
    return Game(int(ret.r0), resp)


class Game:
    """One live game handle. Not thread-safe; create one per worker."""

    def __init__(self, handle: int, initial: dict):
        self._id = handle
        self._last = initial

    @property
    def handle(self) -> int:
        return self._id

    @property
    def state(self) -> dict:
        return self._last.get("state") or {}

    @property
    def pending(self) -> dict | None:
        return self._last.get("pending")

    @property
    def is_over(self) -> bool:
        return bool(self._last.get("game_over"))

    @property
    def winner(self) -> str:
        return self._last.get("winner", "")

    def refresh_state(self) -> dict:
        self._last = _take(_lib.MageState(self._id))
        return self._last

    def legal(self) -> dict | None:
        ret = _lib.MageLegal(self._id)
        if ret == _ffi.NULL:
            return None
        try:
            raw = _ffi.string(ret)
        finally:
            _lib.MageFreeString(ret)
        if not raw or raw == b"null":
            return None
        return orjson.loads(raw)

    def step(self, action: dict) -> dict:
        payload = orjson.dumps(action)
        self._last = _take(_lib.MageStep(self._id, _ffi.new("char[]", payload)))
        return self._last

    def close(self):
        if self._id >= 0 and _lib is not None:
            _lib.MageFree(self._id)
            self._id = -1

    def __enter__(self):
        return self

    def __exit__(self, *exc):
        self.close()

    def __del__(self):
        try:
            self.close()
        except Exception:
            pass
