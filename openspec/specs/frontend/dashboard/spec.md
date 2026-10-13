# frontend/dashboard Specification

## Purpose
Defines the authenticated `/home` dashboard: an overview of the caller's
cars, shortcuts to record new expenses, and a monthly expense chart for one
calendar year, in German and English.

## Requirements

### Requirement: Dashboard shows the caller's cars and create shortcuts
The frontend SHALL render `/home` inside the authenticated application shell. It SHALL show a card for each of the caller's cars that is not deregistered, with the car's make and name and an icon for its type, and each card SHALL link to that car's detail screen. A car is deregistered when its deregistration date is set and is today or earlier in the user's local time zone; its active state SHALL NOT affect the cards. When the caller has no cars to show, no cards SHALL be shown. The screen SHALL offer localized shortcuts to create a refuel, a repair and a ticket; when a car is selected in the dashboard's car selector, each shortcut SHALL preselect that car. The dashboard's car selector SHALL NOT offer a deregistered car, except one that is currently selected through the `carId` search parameter, and the statistics SHALL continue to include deregistered cars when "all cars" is selected.

#### Scenario: User opens the dashboard
- **WHEN** an authenticated user with two cars navigates to `/home`
- **THEN** the screen shows a card for each car linking to its detail screen, and shortcuts to create a refuel, a repair and a ticket

#### Scenario: Deregistered car is not shown
- **WHEN** one of the caller's two cars has a deregistration date of today or earlier
- **THEN** the dashboard shows a card only for the other car and the car selector offers only that car

#### Scenario: Future deregistration date
- **WHEN** a car's deregistration date is after today
- **THEN** the car still has a card and is still offered by the selector

#### Scenario: Inactive car without deregistration date
- **WHEN** a car has `isActive` set to `false` and no deregistration date
- **THEN** the dashboard still shows its card

#### Scenario: Shortcut with a selected car
- **WHEN** the user has selected one of their cars on the dashboard and follows the "add repair" shortcut
- **THEN** the repair creation screen opens with that car preselected

#### Scenario: Deep link to a deregistered car
- **WHEN** the user opens `/home?carId=<id>` for a deregistered car
- **THEN** the chart shows that car's expenses and the selector shows that car as selected

#### Scenario: Totals include deregistered cars
- **WHEN** "all cars" is selected and one car is deregistered
- **THEN** the chart includes that car's expenses

### Requirement: Dashboard charts one year of expenses per month
The frontend SHALL show a stacked bar chart with one bar for each month of one calendar year, January to December, in which each bar stacks that month's refuel, repair and ticket spending. Months SHALL be determined in the viewer's local time zone: the frontend SHALL request the expense-statistics operation from the start of 1 January of the year to the start of 1 January of the following year, both in local time, and SHALL assign each returned row to the local month of its date. Monthly totals SHALL be summed without floating-point rounding error. The chart SHALL have a localized accessible label and a localized legend naming the three series. When the year has no expenses for the selection, the screen SHALL show a localized empty state instead of the chart. While the data loads it SHALL show a localized loading state, and a failure SHALL show a localized error with a retry action.

#### Scenario: Expense just after local midnight
- **WHEN** the viewer's time zone is ahead of UTC and a refuel is dated 00:30 local time on 1 April
- **THEN** its amount counts towards April, not March

#### Scenario: Month without expenses
- **WHEN** the shown year has expenses in March and May but none in April
- **THEN** the chart still has a bar position for April, with zero height

#### Scenario: Year without expenses
- **WHEN** the caller has no expenses in the shown year for the selected car or cars
- **THEN** the screen shows a localized empty state

#### Scenario: Statistics fail to load
- **WHEN** the expense-statistics operation fails
- **THEN** the screen shows a localized error with a retry action

### Requirement: Dashboard year and car are navigable and kept in the URL
The dashboard SHALL show the current local year by default. It SHALL offer localized "previous year" and "next year" actions that move the shown year by one; both SHALL always be enabled, including when the shown year is the current year. It SHALL offer the same caller-car selector as the expense overviews, with an "all cars" choice, and SHALL apply the selection to the chart. The shown year and the selected car SHALL be kept in the URL as `?year=` and `?carId=`, so that reloading or opening a link restores them; the default year SHALL NOT be written to the URL. A missing or malformed `year` SHALL show the current year, and a malformed `carId` SHALL fall back to all cars. Changing the year or car SHALL replace the current history entry rather than add one.

#### Scenario: User moves to the previous year
- **WHEN** the dashboard shows 2026 and the user activates "previous year"
- **THEN** the URL carries `year=2025` and the chart shows 2025's expenses

#### Scenario: User moves past the current year
- **WHEN** the dashboard shows the current year and the user activates "next year"
- **THEN** the following year is shown, with the empty state if it has no expenses

#### Scenario: User filters by car
- **WHEN** the user selects one of their cars
- **THEN** the URL carries that `carId` and the chart shows only that car's expenses

#### Scenario: User reloads a filtered dashboard
- **WHEN** the user opens `/home?year=2024&carId=<id>` for one of their cars
- **THEN** the dashboard shows 2024 for that car

#### Scenario: Malformed year
- **WHEN** the user opens `/home?year=abc`
- **THEN** the dashboard shows the current year

### Requirement: Dashboard is fully localized
Every dashboard string, including headings, shortcut labels, the car selector, the year navigation, the chart's label, legend and series names, month names, amounts, and the loading, empty and error states, SHALL be available in German and English, with months and amounts formatted for the active language.

#### Scenario: User switches language
- **WHEN** the user changes from English to German on the dashboard
- **THEN** all its labels, month names and messages are German
