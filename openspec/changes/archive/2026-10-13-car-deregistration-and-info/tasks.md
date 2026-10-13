## 1. Database

- [x] 1.1 Add migration `000007_car_deregistration_information` (up/down) adding nullable `deregistration_date DATE` and `information TEXT` with a not-blank check and a `deregistration_date >= first_registration` check
- [x] 1.2 Extend the migration tests in `db_test.go` for the new columns and both checks

## 2. Backend API

- [x] 2.1 Add `deregistrationDate` and `information` to `CarInput`, `CarUpdate`, and `Car` in `backend/openapi.yaml` (nullable, `maxLength: 2000` on information) and document `before_first_registration`
- [x] 2.2 Add `ReasonBeforeFirstRegistration` to `internal/apierror`
- [x] 2.3 Extend `cars.Car`, `Input`, `Patch`, `columns`, scan, insert, and update in `internal/cars/store.go`
- [x] 2.4 Parse and validate both members in `internal/cars/handler.go`, including the cross-field date check against the stored car on PATCH, and blank/`too_long` information
- [x] 2.5 Add store and handler tests for create, clear, active-only update, blank/oversized information, and date-order rejection (create and PATCH)
- [x] 2.6 Confirm seed and legacy import still work (new columns stay null) and `openapi_sync_test.go` passes

## 3. Frontend foundation

- [x] 3.1 Regenerate `frontend/src/api/schema.gen.ts` and update `api/client.ts` type guards and `CarInput`
- [x] 3.2 Add the shared `isDeregistered(car, today)` and `isCarMuted(car, today)` helpers in `frontend/src/cars/` with unit tests (today, past, future, null)
- [x] 3.3 Add `de` and `en` strings for the switch, date, information, badges, and the date-order validation

## 4. Cars screens

- [x] 4.1 Add the active switch, deregistration date, and information textarea to `cars/car-form.tsx` with client date-order validation and server field-error mapping
- [x] 4.2 Show active state, deregistration date, and information (line breaks preserved) on the car detail Details tab
- [x] 4.3 Mute inactive or deregistered rows with a text badge in the cars list
- [x] 4.4 Update `-cars.test.tsx` and `-car-detail.test.tsx` for the new fields, switch, muting, and clear-date behavior

## 5. Selections, dashboard, and linked lists

- [x] 5.1 Make `components/car-select.tsx` omit deregistered cars except the currently selected one
- [x] 5.2 Filter deregistered cars out of the dashboard cards in `routes/_authenticated/home.tsx` and keep the all-cars statistics unchanged
- [x] 5.3 Filter deregistered cars out of the car field in the refuel, repair, and ticket create forms and ignore an unavailable `carId` preselect
- [x] 5.4 Mute and badge rows of inactive or deregistered cars in the refuel, repair, and ticket lists
- [x] 5.5 Add or update tests for the dashboard, the three create forms, the three lists, and the deep-link cases

## 6. Docs and verification

- [x] 6.1 Record the active switch, deregistration date, information, and the hiding behavior as intentional additions in `docs/parity-checklist.md`
- [x] 6.2 Run backend tests, frontend typecheck, lint, and tests, and manually check the flows in both locales
