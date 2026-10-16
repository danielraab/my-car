## ADDED Requirements

### Requirement: Line charts can plot series against a second value axis
A line chart SHALL support assigning each series to the left or the right
value axis. When at least one series is assigned to the right axis, the
chart SHALL draw a right value axis with its own round ticks, computed from
the right-axis series only by the rules of "Line charts show a labelled
grid", and the left axis SHALL be computed from the left-axis series only.
Horizontal gridlines SHALL follow the left axis only. Each axis MAY carry a
unit label. When no series is assigned to the right axis, the chart SHALL
look and behave as a single-axis chart. Nearest-point selection SHALL use
each point's on-screen position, whichever axis it is plotted against.

#### Scenario: Two value ranges
- **WHEN** a chart plots one series between 1.50 and 1.80 against the left
  axis and one between 0.25 and 0.75 against the right axis
- **THEN** the left axis is labelled with ticks around 1.5 to 1.8, the right
  axis with ticks around 0.2 to 0.8, and both series use the chart's full
  height

#### Scenario: No right-axis series
- **WHEN** every series of a chart is assigned to the left axis
- **THEN** no right axis or right-axis labels are drawn
