## Context

`GET /refuels`, `GET /repairs` and `GET /tickets` each page with a keyset
cursor over `(date, id)`. The order is `ORDER BY date, id`, and the next
page is selected with `(date, id) > (cursor.date, cursor.id)`. The cursor
encodes the last item's date and ID. The frontend tables render rows in
the order they are returned and append each further page.

Refuel `distance`/`consumption` in the tables come from the chart query
(`GET /refuels/chart`) and are looked up by refuel ID. They do not depend
on the order the collection returns.

## Goals / Non-Goals

**Goals:**
- Return the three expense collections newest first, keeping cursor
  pagination correct: stable, with no duplicates and no gaps.
- Keep the change limited to the ordering contract. Response shapes stay
  the same.

**Non-Goals:**
- A user-selectable sort direction or sort column.
- Reordering the chart series, the expense statistics, suggestions, cars,
  passkeys or attachments.
- Accepting cursors issued under the old ordering.

## Decisions

- **Order by `date DESC, id DESC`.** Both keys are reversed, so the
  ordering stays a strict total order and the row-value comparison still
  works with a single predicate. Mixing `date DESC, id ASC` would need an
  expanded `date < d OR (date = d AND id > i)` predicate, which gains
  nothing.
- **Cursor predicate becomes `(date, id) < (cursor.date, cursor.id)`.** The
  cursor encoding (last item's date and ID) is unchanged. Only the
  comparison flips, so `Cursor`, `Encode` and `DecodeCursor` stay as they
  are.
- **Old cursors are not detected.** A cursor is just a `(date, id)`
  position. An old cursor replayed against the new ordering returns valid
  rows in the new order rather than an error. Cursors live only in
  in-memory query state of the bundled frontend, so a reload after deploy
  starts from the first page anyway. Versioning the cursor format was
  considered and rejected as unnecessary.
- **No frontend logic change.** The tables already render in server order
  and append pages. Only tests or fixtures that assert ascending order are
  adjusted.
- **No new index.** PostgreSQL scans a btree index in either direction, so
  any index that serves `ORDER BY date, id` serves the descending order
  too.

## Risks / Trade-offs

- [The legacy parity order was date-ascending] → This proposal documents
  the deviation explicitly. The parity concern for derived values
  (`DRV-02`/`DRV-03`) is about chronological predecessors, which the chart
  query still computes in ascending order.
- [A page held open across a deploy continues from an old cursor] → It
  still receives valid, deduplicated rows in the new order. At worst a
  partially loaded table mixes orders until it is refreshed. This is
  accepted.
