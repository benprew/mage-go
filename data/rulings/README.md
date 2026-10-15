# Card rulings

Read these files with the complete `cards/mtgish/*.ron` record before you
implement a card. For an engine feature, read the rulings for every card used
to define or test the feature, then consult the relevant Comprehensive Rules.
Rulings are reference evidence, not executable card definitions.

Look up the exact card name in `index.json`. Its value is a filename relative
to this directory. Each `<oracle-id>.json` file contains `name`, `oracle_id`,
`rulings_uri`, `retrieved_at` (UTC), and `rulings`. Each ruling preserves its
Oracle ID, source, publication date, and text. `wotc` identifies Wizards of the
Coast; `scryfall` identifies a Scryfall note. Reprints share one Oracle file.
An empty `rulings` array records a successful check with no published rulings.
A missing entry or file is not evidence that a card has no rulings.

The normal Go test suite checks every RON card against this store. A missing
entry, missing file, invalid JSON, or absent rulings array fails the check and
reports the card's download command. An explicit empty array passes. You can
run the check without network access:

```bash
go run ./cmd/fetchset -check -ron cards/mtgish
go run ./cmd/fetchset -check -card "Time Vault"
```

Refresh all names in the RON collection, including cards the compiler rejects:

```bash
go run ./cmd/fetchset -ron cards/mtgish
```

This command parses the RON files without compiling them. It resolves unique
names through Scryfall's collection API in batches of at most 75, then downloads
the bulk rulings once and keeps only the selected cards. It does not modify RON
definitions or per-set card data. The retrieval time records when the snapshot
was downloaded, not when a ruling was published.

Fetch a missing card or refresh one entry:

```bash
go run ./cmd/fetchset -card "Time Vault"
```

Download a set and refresh its entries in the shared store:

```bash
go run ./cmd/fetchset -o data/DRK.json DRK
```

Use `-rulings-dir <directory>` to select another store. Updates preserve entries
for other cards. Downloads complete before store writes begin. Each file is
replaced atomically, and the index is written last. The complete store update
is not a transaction; do not run two writers against the same store at once.
Commit the index and Oracle files together so future work can read them offline.
