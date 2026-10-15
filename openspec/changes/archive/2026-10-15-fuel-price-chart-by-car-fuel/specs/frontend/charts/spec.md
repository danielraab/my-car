## MODIFIED Requirements

### Requirement: Line charts reveal the nearest point in a popover
Every line chart SHALL show a popover for one active point at a time. When
the pointer is over the plot, the active point SHALL be the plotted point
with the smallest on-screen distance to the pointer, measured in rendered
pixels on both axes, across all series. The popover SHALL show the point's
localized date and its formatted value; when the chart shows a legend, it
SHALL also name the point's series. The active point SHALL be visually
highlighted while the popover is shown.

#### Scenario: Hovering near a point
- **WHEN** the user moves the mouse over the consumption chart close to the point of a refuel dated 3 March 2026 with 7.12 l/100 km
- **THEN** a popover shows that date and "7.12 l/100 km", and that point is highlighted

#### Scenario: Nearest point is chosen in screen distance
- **WHEN** the plot is rendered much wider than tall and the pointer is nearer to point A in the chart's own unstretched proportions but nearer on screen to point B
- **THEN** the popover shows point B

#### Scenario: Chart with several series
- **WHEN** the user hovers the fuel-price chart nearest to a normal refuel of a diesel car
- **THEN** the popover names the "Diesel · Normal" series along with the date and per-litre price
