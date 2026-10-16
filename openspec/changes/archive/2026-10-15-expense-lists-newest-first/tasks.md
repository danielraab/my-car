## 1. Backend

- [x] 1.1 In `backend/internal/refuels/store.go` `List`, order by `date DESC, id DESC`, flip the cursor predicate to `<`, and update the doc comment
- [x] 1.2 Apply the same change to `backend/internal/repairs/store.go` `List`
- [x] 1.3 Apply the same change to `backend/internal/tickets/store.go` `List`
- [x] 1.4 Update the refuel, repair and ticket store tests to expect newest-first pages and cursor continuation, including a same-date tie broken by descending ID
- [x] 1.5 Check that the handler tests do not assume ascending order, and adjust any that do

## 2. API contract

- [x] 2.1 In `backend/openapi.yaml`, change the refuel, repair and ticket collection response descriptions to "ordered newest first: by date descending, then ID descending"
- [x] 2.2 Run the OpenAPI sync test and regenerate the frontend schema types if the generator picks up descriptions

## 3. Frontend

- [x] 3.1 Confirm that the `/refuels`, `/repairs` and `/tickets` tables and the car Expenses tab render rows in server order, with no client-side re-sorting
- [x] 3.2 Update any frontend tests or fixtures that assert ascending row order

## 4. Verification

- [x] 4.1 Run the backend tests (`go test ./...`) and the frontend tests, lint and typecheck
