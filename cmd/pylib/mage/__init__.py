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
struct MageNewGame_return MageNewGame(char *cfgJSON);
char *MageState(int64_t id);
char *MageLegal(int64_t id);
char *MageStep(int64_t id, char *actionJSON);
void MageFree(int64_t id);
void MageFreeString(char *s);
char *MageRegisteredCards(void);
"""


class MageError(RuntimeError):
    pass


_ffi: Any = None  # set by load()
_lib: Any = None  # set by load()


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
    global _ffi, _lib
    ffi = FFI()
    ffi.cdef(_CDEF)
    path = lib_path or _default_lib_path()
    _lib = ffi.dlopen(os.path.abspath(path))
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
