# frontend/charts Specification

## Purpose
Defines behaviour shared by every chart in the frontend: a value popover
that reveals exact numbers through pointer, touch and keyboard input, and
gridlines with labelled round-number ticks, in German and English.

## Requirements

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

### Requirement: Bar charts reveal a whole column in a popover
Every stacked bar chart SHALL show a popover for one active category at a
time. When the pointer is over the plot, the active category SHALL be the
one whose column slot contains the pointer, regardless of the pointer's
height or the size of its segments. The popover SHALL show the category's
label, each series' formatted value, and the formatted total. The popover
SHALL keep the same vertical position for every category, overlaying the
chart, so that moving between categories moves it horizontally only.

#### Scenario: Hovering a month
- **WHEN** the user hovers anywhere in April's column on the dashboard chart, including above its bar
- **THEN** a popover shows April with its refuel, repair and ticket amounts and their total

#### Scenario: Thin segment is still reachable
- **WHEN** a month's ticket amount is too small to be visible as a segment
- **THEN** hovering the month's column still shows the ticket amount in the popover

#### Scenario: Moving between months
- **WHEN** the user moves the pointer from March's column to April's
- **THEN** the popover moves horizontally to April without changing its vertical position

### Requirement: Chart popovers respond to mouse and touch input
For a mouse or pen, moving over the plot SHALL set the active value, and
leaving the plot SHALL hide the popover. For touch, touching the plot SHALL
set the active value, and dragging horizontally across the plot SHALL keep
updating it (scrubbing) without scrolling the page. Vertical dragging
SHALL still scroll the page. After touch ends, the popover SHALL remain
until the user touches outside the chart or activates another value.

#### Scenario: Mouse leaves the plot
- **WHEN** the popover is shown from a mouse hover and the mouse leaves the plot
- **THEN** the popover is hidden

#### Scenario: Tap on a phone
- **WHEN** the user taps a point's position on the consumption chart on a touch device
- **THEN** the popover shows that point and stays shown after the finger lifts

#### Scenario: Scrubbing
- **WHEN** the user drags a finger horizontally across the fuel-price chart
- **THEN** the popover follows the nearest point along the way and the page does not scroll horizontally

#### Scenario: Tap outside
- **WHEN** a popover is shown after a tap and the user taps outside the chart
- **THEN** the popover is hidden

### Requirement: Chart plots are keyboard accessible
Each chart's plot SHALL be a single focusable element in the tab order with
the chart's localized label and a localized hint on how to step through
values. When the plot receives focus, the first value SHALL become active
and its popover SHALL be shown: the earliest point for a line chart, the
first category for a bar chart. The left and right arrow keys SHALL move
to the previous and next value; for a line chart with several series,
values SHALL be ordered by date across all series. Home and End SHALL move
to the first and last value. Escape SHALL hide the popover without moving
focus, and the popover SHALL hide when the plot loses focus. The plot
SHALL show a visible focus indicator, and the active value's popover text
SHALL be announced to assistive technology when it changes.

#### Scenario: Tabbing into a chart
- **WHEN** the user tabs onto the dashboard chart
- **THEN** the plot shows a focus indicator and the popover shows January's values

#### Scenario: Stepping with arrow keys
- **WHEN** the consumption chart's plot has focus with its first point active and the user presses the right arrow key
- **THEN** the second point by date becomes active and its values are announced

#### Scenario: Jumping to the end
- **WHEN** the plot has focus and the user presses End
- **THEN** the last value becomes active

#### Scenario: Escape
- **WHEN** the popover is shown on a focused plot and the user presses Escape
- **THEN** the popover is hidden and focus stays on the plot

### Requirement: Chart popovers stay within the page width
A chart popover SHALL never extend beyond the horizontal bounds of its
plot, so that it is never cut off horizontally and never makes the page
wider or horizontally scrollable, at any viewport width the application
supports. It SHALL be no wider than the plot. It MAY extend above or below
the plot.

#### Scenario: Point at the right edge on a phone
- **WHEN** on a 360 px wide viewport the user taps the latest point, at the plot's right edge
- **THEN** the popover is fully visible, its right edge aligned within the plot, and the page has no horizontal scroll

#### Scenario: Point at the left edge
- **WHEN** the active point is the first point, at the plot's left edge
- **THEN** the popover's left edge stays within the plot

### Requirement: Line charts show a labelled grid
Every line chart SHALL draw horizontal gridlines at round value ticks,
steps of 1, 2 or 5 times a power of ten, with about four to six ticks, and
SHALL extend its value axis to the outermost ticks that enclose all
plotted values. Each horizontal gridline SHALL be labelled on the value
axis, with the precision the step needs. The chart SHALL draw vertical
gridlines at calendar boundaries within its date range, using weeks,
months, quarters or years so that the number of lines stays readable for
the range, and SHALL label them on the date axis in the active language.
Narrow screens MAY show fewer date labels than gridlines. Gridlines SHALL
be drawn in a light grey at reduced opacity behind the data. When all
values are equal, the axis SHALL still span at least one step around the
value.

#### Scenario: Consumption grid
- **WHEN** the consumption values in range lie between 6.12 and 7.93 l/100 km
- **THEN** the value axis runs from 6 to 8 with labelled gridlines at 6, 6.5, 7, 7.5 and 8

#### Scenario: Monthly vertical lines
- **WHEN** the chart's date range spans about six months
- **THEN** a vertical gridline marks each month start within the range, labelled with the localized month name

#### Scenario: Single value
- **WHEN** the chart has one point with value 7
- **THEN** the value axis spans ticks below and above 7 and the point is drawn between them

### Requirement: Bar charts show horizontal gridlines
Every stacked bar chart SHALL draw horizontal gridlines at round value
ticks starting from 0, steps of 1, 2 or 5 times a power of ten, and SHALL
extend its value axis to the first tick at or above the largest category
total. Each gridline SHALL be labelled on the value axis, formatted like
the chart's values. Bar charts SHALL NOT draw vertical gridlines.
Gridlines SHALL be drawn in a light grey at reduced opacity behind the
bars.

#### Scenario: Dashboard grid
- **WHEN** the largest monthly total of the shown year is 530 €
- **THEN** the value axis runs from 0 to 600 with labelled gridlines at 0, 200, 400 and 600, and no vertical gridlines

### Requirement: Chart interaction text is localized
The keyboard hint, the popover's dates, category labels, series names,
values and totals, and all tick labels SHALL be available in German and
English and formatted for the active language.

#### Scenario: German popover
- **WHEN** the active language is German and the user hovers a fuel-price point
- **THEN** the popover shows a German date, the German fuel name and a price with a decimal comma
