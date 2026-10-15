## Context

The app shows no currency anywhere. Amounts go through
`formatAmount(value, locale)` in `frontend/src/cars/format.ts` (tables, totals,
purchase price, dashboard chart), except the per-litre price, which the
refuel table and fuel-price chart format with their own `Intl.NumberFormat`.
Units are hard-coded in `frontend/src/i18n/resources.ts`: `odometerValue:
'{{value}} km'`, `litersValue: '{{value}} l'`, `consumptionValue: '{{value}}
l/100 km'`, "Odometer reading (km)", "Litres", "Per litre", and chart
descriptions such as "Litres per 100 km over time…".

On the backend, `accounts` has `first_name`/`last_name`, and `auth.Account`
is scanned from several queries: the session lookup and both login upserts in
`internal/auth/store.go`, `internal/auth/passkey_store.go`, and
`internal/profile/store.go`. The OpenAPI `Session` schema embeds `Profile`.
The frontend's `_authenticated` route ensures the session query
(`['session']`) is loaded before any authenticated screen renders, and the
profile save already writes the saved profile into that cache.

Consumption is computed on the backend (`internal/refuels/handler.go`,
`liters / distance × 100`) and is unit-agnostic: with gallons and miles the
same number means gal/100 mi.

## Goals / Non-Goals

**Goals:**
- One per-account currency, volume unit and distance unit, persisted and
  exposed through the profile.
- Every amount, volume, distance, per-volume price and consumption label
  follows those settings on every screen, in German and English, without a
  reload after a change.
- Stored values are shown exactly as stored; nothing is converted.

**Non-Goals:**
- Currency or unit conversion, exchange rates, per-car or per-entry currency.
- Inverted consumption (mpg, km/l), US vs imperial gallons, kWh for electric
  cars.
- Renaming API members (`liters`, `perLiter`, `odometerReading`).

## Decisions

### Store settings as three text columns on `accounts`
`currency TEXT NOT NULL DEFAULT 'EUR' CHECK (currency ~ '^[A-Z]{3}$')`,
`volume_unit TEXT NOT NULL DEFAULT 'liter' CHECK (volume_unit IN
('liter','gallon'))`, `distance_unit TEXT NOT NULL DEFAULT 'kilometer'
CHECK (distance_unit IN ('kilometer','mile'))` in migration `000009`.
Defaults fill existing rows and make the legacy import and both login
upserts work unchanged.
- *Alternative:* Postgres enums like the category columns. Rejected for the
  currency (about 180 values, and the list changes); text plus CHECK is used
  for the units too so all three behave the same and adding a unit later is a
  one-line constraint change.
- *Alternative:* a separate `account_settings` table. Rejected: three scalar
  values with defaults, always read together with the profile.

### Currency is validated in Go, not by the database
The CHECK only enforces the shape; `internal/profile` rejects anything that
is not exactly three uppercase letters or not known to
`golang.org/x/text/currency` (`currency.ParseISO`), reported as
`invalid_enum`. `golang.org/x/text` is already an indirect dependency, so it
becomes direct without adding a new module.
- *Alternative:* an OpenAPI `enum` of all codes. Rejected: a long, churning
  list in the contract that the frontend would duplicate anyway. The contract
  documents `currency` as `pattern: ^[A-Z]{3}$` with a description naming
  ISO 4217; units are OpenAPI enums.

### Settings travel with `auth.Account` / `Profile`
`auth.Account` gains `Currency`, `VolumeUnit` and `DistanceUnit`; every query
that scans an account selects the three columns, so `GET /me`, `PATCH /me`
and `GET /session` return them with no extra round trip.
`UpdateAccountNames` becomes a general profile update (`COALESCE` per
supplied field). `parseUpdate` accepts the three new members and keeps the
current rules for unknown members, `email`, and all-or-nothing application.

### Frontend reads settings from the session cache via a context
The `_authenticated` layout already subscribes to the `['session']` query; it
provides `profile.currency/volumeUnit/distanceUnit` through a
`DisplayUnitsContext` (`lib/display-units.tsx`), and a `useUnitFormat()` hook
turns them into formatters and labels for the active language. Without a
provider (public routes, isolated component tests) the context falls back to
the defaults. Because the profile save already updates the session cache,
every consumer re-renders after a save.
- *Alternative:* each component reads the session query itself. Rejected
  during implementation: chart and table components are tested without a
  query client, and the layout already holds the session.

### Formatting goes through a small set of helpers
`format.ts` replaces `formatAmount` with `formatMoney(value, locale,
currency, digits = 2)` using `Intl.NumberFormat(locale, { style:
'currency', currency, minimumFractionDigits: digits, maximumFractionDigits:
digits })`, with `digits = 3` for per-volume prices. Fixing the digits
overrides the currency's default (JPY's 0), so stored decimals are never
rounded away. Non-numeric input is still returned unchanged. Every existing
`formatAmount` caller, the refuel table's per-litre cell, and the fuel-price
chart's value formatter switch to it. Chart Y-axis ticks (`formatValueTick`)
stay plain numbers; the unit is named in the chart description and popover.

A form-input suffix (currency symbol from `Intl.NumberFormat(...).
formatToParts`, or the unit abbreviation) is a small shared component, wired
into the input's `aria-describedby`.

### Unit labels are i18n keys, interpolated
Add `units.volume.{liter,gallon}` (`l`/`gal`), `units.distance.{kilometer,
mile}` (`km`/`mi`), and long forms for headers and descriptions
(`Litres`/`Gallons`, `Liter`/`Gallonen`, `per litre`/`per gallon`, …) in both
locales. Existing strings take the unit as a parameter, e.g. `odometerValue:
'{{value}} {{distance}}'`, `consumptionValue: '{{value}} {{volume}}/100
{{distance}}'`, `odometerReading: 'Odometer reading ({{distance}})'`.
Helpers wrap the common cases so call sites don't assemble units by hand.
- *Alternative:* `Intl.NumberFormat` `style: 'unit'`. Rejected: there is no
  sanctioned "per 100 km" unit, and German/English wording in headers is
  needed anyway.

### Currency choices from the browser
The profile select lists `Intl.supportedValuesOf('currency')`, labelled with
`Intl.DisplayNames(locale, { type: 'currency' })` plus the code, sorted by
localized name, and always includes the current value. The no-conversion
hint sits below the three selects.

## Risks / Trade-offs

- [The browser's currency list and `x/text`'s list differ slightly (historic
  or fund codes)] → An unaccepted choice gets a localized field error on save;
  the currency in use stays shown because the current value is always in the
  list.
- [Relabelling looks like conversion to users] → The profile hint states
  values are not converted; the spec forbids conversion everywhere.
- ["gal/100 mi" is an unusual unit for gallon users] → Accepted for now; an
  inverted display would be its own proposal.
- [Missing a hard-coded "km"/"l" string] → Tasks include grepping
  `resources.ts` and the components for `km`, `l/100`, `litre`, `Liter`, and
  a test rendering the refuel table and consumption chart with gallons and
  miles.
- [Every `auth.Account` query must add the columns; one missed scan would
  return empty settings] → Column count mismatches fail at scan time; store
  tests cover session, login and passkey paths.

## Migration Plan

1. Migration `000009_account_display_units` adds the columns with defaults;
   the down migration drops them. It is additive and safe on a live database.
2. Backend and frontend ship together (the backend serves the frontend
   build), so the required `Profile` members never reach an old client.
3. Rollback: deploy the previous build and run the down migration; no other
   data is affected.

## Open Questions

None. Decisions from exploration: default EUR, any ISO 4217 code, symbol in
every cell, settings per account, generic gallon, consumption relabelled not
inverted, electric cars out of scope.
