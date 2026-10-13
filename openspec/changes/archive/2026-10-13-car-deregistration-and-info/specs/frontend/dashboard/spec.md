# Spec Delta

## MODIFIED Requirements

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