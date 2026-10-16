## 1. Persistence

- [x] 1.1 Add migration `000009_account_display_units` (up/down) adding `currency`, `volume_unit`, `distance_unit` to `accounts` with defaults `EUR`/`liter`/`kilometer` and the CHECK constraints from design.md
- [x] 1.2 Add a migration test: existing accounts get the defaults, invalid unit and lowercase currency values are rejected, car/expense rows untouched

## 2. API contract

- [x] 2.1 In `openapi/openapi.yaml` and `backend/openapi.yaml`, add required `currency` (`pattern: ^[A-Z]{3}$`, ISO 4217 description), `volumeUnit` (enum `liter`, `gallon`) and `distanceUnit` (enum `kilometer`, `mile`) to `Profile`, and the same as optional members to `ProfileUpdate`
- [x] 2.2 Update the descriptions of refuel `liters`, `perLiter`, `consumption` and every `odometerReading`/`distance` to say values are in the account's configured units and never converted
- [x] 2.3 Run `pnpm generate:api`, `pnpm lint:openapi`, and the backend OpenAPI sync test

## 3. Backend

- [x] 3.1 Add `Currency`, `VolumeUnit`, `DistanceUnit` to `auth.Account` and select them in every account scan (session lookup and both login upserts in `internal/auth/store.go`, `internal/auth/passkey_store.go`, `internal/profile/store.go`)
- [x] 3.2 Include the three members in the profile JSON written by `internal/profile/handler.go` and in the session response
- [x] 3.3 Generalise `UpdateAccountNames` into a profile update that applies each supplied field (names and settings) in one statement
- [x] 3.4 Extend `parseUpdate` to accept `currency`, `volumeUnit`, `distanceUnit`: non-string → `invalid_type`; currency not exactly three uppercase letters or unknown to `golang.org/x/text/currency` → `invalid_enum`; unit outside its set → `invalid_enum`; keep all-or-nothing behaviour and the `email`/unknown-member rules. Make `golang.org/x/text` a direct dependency
- [x] 3.5 Handler and store tests for each scenario in `specs/api-contract/profile/spec.md` (defaults, valid update, `XYZ`/`usd`/`EURO`, unknown units, mixed valid+invalid body unchanged, expense values unchanged), plus session and login paths returning the settings

## 4. Frontend foundations

- [x] 4.1 Add a `useDisplayUnits()` hook reading currency and units from the `['session']` query, defaulting to `EUR`/`liter`/`kilometer`
- [x] 4.2 Add a currency formatter in `cars/format.ts` (`style: 'currency'`, fixed 2 or 3 fraction digits, non-numbers unchanged) and replace `formatAmount` usages with it
- [x] 4.3 Add de/en unit strings (`l`/`gal`, `km`/`mi`, long forms for headers and descriptions) and helpers for volume, distance, per-volume and consumption labels
- [x] 4.4 Add a shared input-suffix component (currency symbol or unit) linked via `aria-describedby`
- [x] 4.5 Unit tests for the formatter (EUR de/en, JPY keeps decimals, per-volume 3 digits, non-number passthrough) and label helpers

## 5. Frontend screens

- [x] 5.1 Refuels: table cells and headers (odometer, volume, per-volume price, amount, consumption, total), create/edit form suffixes and labels, fuel-price chart values and description, consumption chart description and popover
- [x] 5.2 Repairs and tickets: amount cells and totals, odometer cells and labels, form suffixes
- [x] 5.3 Cars: purchase price on list and detail, car form purchase-price suffix, the detail screen's refuel/repair tabs, Consumption tab texts
- [x] 5.4 Dashboard: stacked bar chart values in the account currency
- [x] 5.5 Replace every remaining hard-coded unit in `i18n/resources.ts` (grep for `km`, `l/100`, `litre`, `Liter`, `per litre`, `pro Liter`) with interpolated units in both locales
- [x] 5.6 Route tests: refuels, repairs, tickets, cars, car detail and dashboard render with USD/gallon/mile; existing tests still pass with defaults

## 6. Profile screen

- [x] 6.1 Add currency (from `Intl.supportedValuesOf('currency')`, localized via `Intl.DisplayNames`, current value always present), volume unit and distance unit selects, prefilled from the profile
- [x] 6.2 Add the localized no-conversion hint; submit changed settings with the names; show `400` field errors on the matching select
- [x] 6.3 Ensure the saved profile updates both `['me']` and `['session']` caches so other screens relabel without reload
- [x] 6.4 Profile tests: prefill, saving USD/mile sends those members, currency field error, German labels, another screen relabels after save

## 7. Wrap-up

- [x] 7.1 Note the feature in `docs/parity-checklist.md` as an addition beyond parity, if the checklist tracks those
- [x] 7.2 Run backend tests, `pnpm typecheck`, `pnpm test`, `pnpm check`, `pnpm check:api`
