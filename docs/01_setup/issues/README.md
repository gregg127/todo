# Issues — todo-cli v1

Vertical slices derived from [the PRD](../prd.md). Each one is a thin end-to-end path that
is demoable on its own; the file is the ticket, since no issue tracker is configured for
this repo.

| # | Slice | Blocked by | User stories |
|---|---|---|---|
| [01](./01-walking-skeleton.md) | Walking skeleton: binary, install, empty board, `--help` | — | 1–4, 6, 7, 9, 10, 67 |
| [02](./02-markdown-store-read.md) | Markdown store: read an existing board | 01 | 5, 11, 12, 60, 61, 66 |
| [03](./03-cursor-movement.md) | Cursor movement | 02 | 13, 19–22 |
| [04](./04-viewport-and-presentation.md) | Viewport and presentation | 03 | 14–18, 23 |
| [05](./05-status-changes-and-persistence.md) | Status changes and atomic persistence | 04 | 8, 24–28, 58, 59 |
| [06](./06-reordering.md) | Reordering within a section | 05 | 29–32 |
| [07](./07-add-edit-delete.md) | Adding, editing and deleting tasks | 05 | 33–43 |
| [08](./08-undo.md) | Undo | 06, 07 | 44–50 |
| [09](./09-filtering.md) | Filtering | 07 | 51–57 |
| [10](./10-external-reload.md) | External reload after a hand-edit | 08, 09 | 62–65 |

```
01 → 02 → 03 → 04 → 05 ─┬→ 06 ─┬→ 08 ─┬→ 10
                        └→ 07 ─┤      │
                               └→ 09 ─┘
```

06 and 07 can be picked up in parallel once 05 lands; so can 08 and 09 after that.
