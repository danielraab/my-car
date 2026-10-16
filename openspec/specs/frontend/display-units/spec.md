# frontend/display-units Specification

## Purpose

Defines how the frontend labels amounts, volumes, distances, per-volume prices
and consumption with the caller's account currency and units, without ever
converting the values.

## Requirements

### Requirement: Display settings come from the caller's profile
The frontend SHALL take the currency, volume unit and distance unit used on
every authenticated screen from the caller's profile as embedded in the
session. A setting change saved on the profile screen SHALL apply to every
screen without a page reload. The settings SHALL only change how values are
labelled: the frontend SHALL NOT convert, rescale or recalculate any stored or
server-computed value because of them.

#### Scenario: Settings change applies everywhere
- **WHEN** the user changes the currency from EUR to CHF on the profile screen
  and then opens `/refuels`
- **THEN** every amount on `/refuels` is labelled in CHF without a page reload

#### Scenario: Values are relabelled, not converted
- **WHEN** a refuel stored with amount `65.90`, `42.5` litres and odometer
  reading `120000` is shown after the user switches to USD, gallons and miles
- **THEN** it shows an amount of 65.90 in USD, a volume of 42.5 gal and an
  odometer reading of 120,000 mi

### Requirement: Amounts are shown with the account currency
Every monetary amount the frontend displays SHALL be formatted as a currency
amount in the account currency for the active language, so that the language
decides the symbol and its placement. This SHALL apply to every table row and
total of refuels, repairs and tickets, the car purchase price on the cars list
and car detail screen, the dashboard expense chart's values, and the refuel
per-volume price wherever it is shown. Amounts SHALL be shown with exactly two
fraction digits and per-volume prices with exactly three, whatever the
currency's customary minor unit. A value that is not a number SHALL be shown
unchanged, as before. Chart value-axis tick labels MAY omit the currency
symbol.

#### Scenario: Euro amount in German
- **WHEN** the account currency is EUR, the language is German and a ticket
  amount is `1234.5`
- **THEN** the amount is shown as "1.234,50 €"

#### Scenario: Euro amount in English
- **WHEN** the account currency is EUR, the language is English and a ticket
  amount is `1234.5`
- **THEN** the amount is shown as "€1,234.50"

#### Scenario: Currency without minor unit keeps stored decimals
- **WHEN** the account currency is JPY and a repair amount is `120.50`
- **THEN** the amount is shown with the fraction digits ".50" rather than
  being rounded to a whole number

#### Scenario: Per-volume price
- **WHEN** the account currency is EUR, the language is German and a refuel's
  per-volume price is `1.659`
- **THEN** the per-volume price is shown as "1,659 €"

### Requirement: Volumes, distances and consumption use the account units
The frontend SHALL label a refuel's quantity by the fuel of the refuel's car:
in kilowatt-hours ("kWh") when the car's fuel is `electric`, and otherwise in
the account volume unit (`liter` as "l", `gallon` as "gal"). Odometer
readings and distances SHALL use the account distance unit (`kilometer` as
"km", `mile` as "mi") for every car. A refuel's unit price SHALL be labelled
per its quantity unit, and its consumption as quantity unit per 100 distance
units (for example "l/100 km", "gal/100 mi" or "kWh/100 km"). Where values of
cars with different quantity units can appear together, such as the refuel
tables, column headers SHALL name the quantity, unit price and consumption
without a unit and every value cell SHALL show its own unit. Form field
labels, value cells, chart titles, chart descriptions, popovers and
empty-state texts that name a quantity, distance, unit price or consumption
SHALL use these units in both German and English and SHALL NOT hard-code
litres or kilometres. No value SHALL be converted between units.

#### Scenario: Default units
- **WHEN** the account uses `liter` and `kilometer` and a gasoline car's
  refuel is shown
- **THEN** its odometer reading shows "km", its quantity "l", its unit price
  is per litre and its consumption shows "l/100 km", as before

#### Scenario: Gallons and miles
- **WHEN** the account uses `gallon` and `mile` and a diesel car's refuel has
  a consumption of `7.12`
- **THEN** its consumption is shown as "7.12 gal/100 mi", its odometer reading
  in "mi", its quantity in "gal", and its unit price per gallon

#### Scenario: Charge of an electric car
- **WHEN** the account uses `gallon` and `mile` and an electric car's refuel
  has quantity `40`, unit price `0.350` and consumption `28.10`
- **THEN** its quantity is shown as "40 kWh", its unit price per kWh, and its
  consumption as "28.10 kWh/100 mi"

#### Scenario: Mixed table
- **WHEN** the refuels table shows a gasoline car's refuel and an electric
  car's charge
- **THEN** the quantity, unit price and consumption headers name no unit, the
  gasoline row's cells show litres and the electric row's cells show kWh

#### Scenario: Consumption chart texts follow the units
- **WHEN** the account uses `gallon` and `mile` and the user opens a gasoline
  car's Consumption tab
- **THEN** the chart's description and popover values name gallons per 100
  miles in the active language

### Requirement: Form inputs show their unit
Inputs for an amount or purchase price SHALL show the account currency as a
visible suffix, inputs for a refuel quantity SHALL show the quantity unit of
the selected car as defined by "Volumes, distances and consumption use the
account units", and inputs for an odometer reading SHALL show the account
distance unit. The quantity suffix SHALL follow a change of the selected car
without a page reload. The suffix SHALL be part of the input's accessible
description or label.

#### Scenario: Refuel form in miles and gallons
- **WHEN** the account uses USD, `gallon` and `mile` and the user opens
  `/refuels/create` with a gasoline car selected
- **THEN** the amount input shows the USD symbol, the quantity input shows
  "gal" and the odometer input shows "mi"

#### Scenario: Refuel form for an electric car
- **WHEN** the user selects an electric car on `/refuels/create`
- **THEN** the quantity input shows "kWh"
