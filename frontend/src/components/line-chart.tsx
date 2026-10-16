import { type CSSProperties, type ReactNode, useId } from 'react'
import { useTranslation } from 'react-i18next'

import {
  niceTicks,
  type TimeUnit,
  timeTicks,
  type ValueTicks,
} from '#/lib/chart-ticks'

import { ChartPopover } from './chart-popover'
import { useChartInteraction } from './use-chart-interaction'

export type LinePoint = { id: string; time: number; value: number }
export type LineSeries = {
  key: string
  // Sets the series colour through its --series custom property.
  className: string
  label: string
  points: LinePoint[]
  // The value axis the series is plotted against; left by default.
  axis?: 'left' | 'right'
}

// Date-axis labels nearer an end than this (in percent) are aligned to it
// instead of centred, so they never hang past the plot.
const edge = 12

// Time-series lines over a grid of round value ticks and calendar
// boundaries. Series share the left value axis unless some are assigned to
// a right axis, which then gets ticks of its own; gridlines follow the left
// axis. Points arrive in time order; those without a
// finite time or value are left out, and without any points the chart
// renders empty instead. The plot is one tab stop: hovering, tapping or
// scrubbing picks the nearest point on screen, and the arrow keys step
// through the points by date; the active point's exact numbers show in a
// popover and are announced.
export function LineChart({
  series,
  formatValue,
  formatTick,
  label,
  empty,
  showLegend = false,
  axisLabels,
}: {
  series: LineSeries[]
  // Formats a point's value for the popover; series is the point's line.
  formatValue: (value: number, series: LineSeries) => string
  // Formats a value-axis tick; step is the distance between ticks.
  formatTick: (value: number, step: number) => string
  label: string
  empty: ReactNode
  showLegend?: boolean
  // Unit captions above each value axis, shown when both axes are drawn.
  axisLabels?: { left: string; right: string }
}) {
  const { t, i18n } = useTranslation()
  const locale = i18n.resolvedLanguage ?? 'en'
  const hintId = useId()
  const lines = series
    .map((line) => ({
      ...line,
      points: line.points.filter(
        (point) => Number.isFinite(point.time) && Number.isFinite(point.value),
      ),
    }))
    .filter((line) => line.points.length > 0)
  // A right axis only exists beside a left one; on its own, a right-axis
  // series is plotted against the left axis.
  const twoAxes =
    lines.some((line) => line.axis === 'right') &&
    lines.some((line) => line.axis !== 'right')
  const onRight = (line: LineSeries) => twoAxes && line.axis === 'right'
  // Every point in stepping order: by date, then by series.
  const ordered = lines
    .flatMap((line, order) =>
      line.points.map((point) => ({ ...point, line, order })),
    )
    .sort((a, b) => a.time - b.time || a.order - b.order)

  const times = ordered.map((point) => point.time)
  const minTime = Math.min(...times)
  const maxTime = Math.max(...times)
  const ticksOf = (right: boolean) => {
    const values = ordered
      .filter((point) => onRight(point.line) === right)
      .map((point) => point.value)
    return values.length > 0
      ? niceTicks(Math.min(...values), Math.max(...values))
      : niceTicks(0, 1)
  }
  const ticks = ticksOf(false)
  const rightTicks = twoAxes ? ticksOf(true) : null
  const x = (time: number) =>
    maxTime === minTime ? 50 : ((time - minTime) / (maxTime - minTime)) * 100
  const scale = (axis: ValueTicks) => (value: number) =>
    100 - ((value - axis.start) / (axis.end - axis.start)) * 100
  const y = scale(ticks)
  const yRight = rightTicks === null ? y : scale(rightTicks)
  const yOf = (line: LineSeries) => (onRight(line) ? yRight : y)

  const { active, areaProps } = useChartInteraction(
    ordered.length,
    (px, py, width, height) => {
      let nearest: number | null = null
      let best = Number.POSITIVE_INFINITY
      ordered.forEach((point, i) => {
        const dx = (x(point.time) / 100) * width - px
        const dy = (yOf(point.line)(point.value) / 100) * height - py
        const distance = dx * dx + dy * dy
        if (distance < best) {
          best = distance
          nearest = i
        }
      })
      return nearest
    },
  )

  if (ordered.length === 0) return empty

  const date = new Intl.DateTimeFormat(locale, { dateStyle: 'medium' })
  const time = timeTicks(minTime, maxTime)
  const formatTime = timeFormatter(time.unit, locale, (quarter, year) =>
    t('charts.quarter', { quarter, year }),
  )
  const current = active === null ? null : ordered[active]
  const announcement =
    current === null
      ? ''
      : [
          date.format(current.time),
          showLegend ? current.line.label : null,
          formatValue(current.value, current.line),
        ]
          .filter(Boolean)
          .join(' · ')

  const valueAxis = (axis: ValueTicks, position: (value: number) => number) => (
    <div
      className={
        axis === rightTicks ? 'chart-y-axis chart-y-axis-right' : 'chart-y-axis'
      }
      aria-hidden="true"
    >
      <div className="chart-y-sizer">
        {axis.values.map((value) => (
          <span key={value}>{formatTick(value, axis.step)}</span>
        ))}
      </div>
      {axis.values.map((value) => (
        <span
          key={value}
          className="chart-tick"
          style={{ '--at': `${position(value)}%` } as CSSProperties}
        >
          {formatTick(value, axis.step)}
        </span>
      ))}
    </div>
  )
  const currentY = current === null ? 0 : yOf(current.line)(current.value)

  return (
    <figure className="line-chart">
      {twoAxes && axisLabels ? (
        <div className="chart-axis-labels" aria-hidden="true">
          <span>{axisLabels.left}</span>
          <span>{axisLabels.right}</span>
        </div>
      ) : null}
      <div
        className={twoAxes ? 'chart-plot chart-plot-two-axes' : 'chart-plot'}
      >
        {valueAxis(ticks, y)}
        <div
          {...areaProps}
          className="chart-area"
          role="slider"
          tabIndex={0}
          aria-label={label}
          aria-describedby={hintId}
          aria-valuemin={1}
          aria-valuemax={ordered.length}
          aria-valuenow={(active ?? 0) + 1}
          aria-valuetext={announcement || undefined}
        >
          <svg
            aria-hidden="true"
            viewBox="0 0 100 100"
            preserveAspectRatio="none"
          >
            <g className="chart-grid">
              {ticks.values.map((value) => (
                <line
                  key={`y${value}`}
                  vectorEffect="non-scaling-stroke"
                  x1={0}
                  x2={100}
                  y1={y(value)}
                  y2={y(value)}
                />
              ))}
              {time.ticks.map((tick) => (
                <line
                  key={`x${tick.time}`}
                  vectorEffect="non-scaling-stroke"
                  x1={x(tick.time)}
                  x2={x(tick.time)}
                  y1={0}
                  y2={100}
                />
              ))}
            </g>
            {lines.map((line) => {
              const yLine = yOf(line)
              return (
                <g key={line.key} className={`chart-series ${line.className}`}>
                  {line.points.length > 1 ? (
                    <polyline
                      fill="none"
                      vectorEffect="non-scaling-stroke"
                      points={line.points
                        .map(
                          (point) => `${x(point.time)},${yLine(point.value)}`,
                        )
                        .join(' ')}
                    />
                  ) : null}
                  {line.points.map((point) => (
                    // A zero-length line with a round cap draws an undistorted dot.
                    <line
                      key={point.id}
                      className={
                        point.id === current?.id && line === current.line
                          ? 'chart-dot chart-dot-active'
                          : 'chart-dot'
                      }
                      vectorEffect="non-scaling-stroke"
                      x1={x(point.time)}
                      x2={x(point.time)}
                      y1={yLine(point.value)}
                      y2={yLine(point.value)}
                    />
                  ))}
                </g>
              )
            })}
          </svg>
          {current === null ? null : (
            <ChartPopover
              x={x(current.time)}
              y={currentY}
              placement={currentY >= 50 ? 'above' : 'below'}
            >
              <strong>{date.format(current.time)}</strong>
              {showLegend ? (
                <dl>
                  <dt className={current.line.className}>
                    {current.line.label}
                  </dt>
                  <dd>{formatValue(current.value, current.line)}</dd>
                </dl>
              ) : (
                formatValue(current.value, current.line)
              )}
            </ChartPopover>
          )}
        </div>
        {rightTicks === null ? null : valueAxis(rightTicks, yRight)}
        <div className="chart-x-axis" aria-hidden="true">
          {time.ticks
            .filter((tick) => tick.wide)
            .map((tick) => {
              const at = x(tick.time)
              const classes = ['chart-tick']
              if (at < edge) classes.push('chart-tick-start')
              if (at > 100 - edge) classes.push('chart-tick-end')
              if (!tick.narrow) classes.push('chart-tick-narrow-hidden')
              return (
                <span
                  key={tick.time}
                  className={classes.join(' ')}
                  style={{ '--at': `${at}%` } as CSSProperties}
                >
                  {formatTime(tick.time)}
                </span>
              )
            })}
        </div>
      </div>
      <p id={hintId} className="sr-only">
        {t('charts.keyboardHint')}
      </p>
      {showLegend ? (
        <figcaption>
          <ul className="chart-legend">
            {lines.map((line) => (
              <li key={line.key} className={line.className}>
                {line.label}
              </li>
            ))}
          </ul>
        </figcaption>
      ) : null}
    </figure>
  )
}

// Labels a calendar boundary for its unit: day and month for weeks, the
// month (with its year in January) for months, "Q2 2026" for quarters.
function timeFormatter(
  unit: TimeUnit,
  locale: string,
  quarter: (quarter: number, year: number) => string,
): (time: number) => string {
  const week = new Intl.DateTimeFormat(locale, {
    day: 'numeric',
    month: 'short',
  })
  const month = new Intl.DateTimeFormat(locale, { month: 'short' })
  const january = new Intl.DateTimeFormat(locale, {
    month: 'short',
    year: 'numeric',
  })
  const year = new Intl.DateTimeFormat(locale, { year: 'numeric' })
  return (time) => {
    const date = new Date(time)
    switch (unit) {
      case 'week':
        return week.format(date)
      case 'month':
        return (date.getMonth() === 0 ? january : month).format(date)
      case 'quarter':
        return quarter(Math.floor(date.getMonth() / 3) + 1, date.getFullYear())
      case 'year':
        return year.format(date)
    }
  }
}
