## Why

The refuel, repair and ticket tables list the oldest entry first. Users
mostly care about recent entries, so the record they just added sits at
the bottom of the last page, behind a "load more" click.

## What Changes

- **BREAKING (API ordering)**: the paginated refuel, repair and ticket
  collections (`GET /refuels`, `GET /repairs`, `GET /tickets`) return
  entries ordered by date descending, then ID descending. Cursors continue
  from the last item in that order. Existing cursors from the old ordering
  are not migrated. They are short-lived page tokens, and the only client
  is the bundled frontend.
- The `/refuels`, `/repairs` and `/tickets` tables and the car Expenses tab
  show the newest entry at the top. "Load more" appends older entries.
- Unchanged: the refuel chart series and the expense-statistics query stay
  date-ascending, because charts and per-car consumption need chronological
  order. Station and location suggestions stay lexical. Cars, passkeys and
  attachments keep their current ordering.
- Server-computed `distance` and `consumption` do not change. They are
  measured against each car's chronological predecessor, whatever order
  the page is in.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `api-contract/expenses`: the refuel, repair and ticket collections are
  ordered by date then ID, newest first. Repairs are now named explicitly
  alongside refuels and tickets.
- `frontend/cars`: the Expenses tab sections list records newest first.
- `frontend/refuels`, `frontend/repairs`, `frontend/tickets`: the list
  tables show the newest entry first, and further pages append older
  entries.

## Impact

- Backend: the `List` queries and keyset cursor predicates in
  `backend/internal/{refuels,repairs,tickets}/store.go`, plus their store
  tests.
- `backend/openapi.yaml`: the descriptions of the three collection
  responses. No schema shapes change, so the generated frontend types are
  unaffected.
- Frontend: no logic change is expected, because tables render rows in the
  order the server returns them. Tests or fixtures that assert row order
  may need updating.
- Parity: the legacy app listed expenses date-ascending. This proposal is
  the intentional, documented deviation from that.
