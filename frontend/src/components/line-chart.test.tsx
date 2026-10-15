import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import '#/i18n'

import { LineChart, type LineSeries } from './line-chart'

const day = 24 * 60 * 60 * 1000
const start = new Date(2026, 0, 1).getTime()

// Points at x 0, 50, 65 and 100 of the plot (by time) and y 100, 0, 100, 50
// of the 0..100 value axis, drawn top to bottom.
const series: LineSeries[] = [
  {
    key: 'a',
    className: 'chart-fuel-diesel-normal',
    label: 'Normal',
    points: [
      { id: 'p0', time: start, value: 0 },
      { id: 'b', time: start + 50 * day, value: 0 },
      { id: 'c', time: start + 100 * day, value: 50 },
    ],
  },
  {
    key: 'b',
    className: 'chart-fuel-diesel-special',
    label: 'Premium',
    points: [{ id: 'a', time: start + 65 * day, value: 100 }],
  },
]

function renderChart(showLegend = true) {
  render(
    <LineChart
      series={series}
      formatValue={(value) => `v${value}`}
      formatTick={(value) => `t${value}`}
      label="Prices"
      empty={<p>Nothing</p>}
      showLegend={showLegend}
    />,
  )
  const plot = screen.getByRole('slider', { name: 'Prices' })
  // A plot ten times wider than tall.
  vi.spyOn(plot, 'getBoundingClientRect').mockReturnValue({
    x: 0,
    y: 0,
    left: 0,
    top: 0,
    right: 1000,
    bottom: 100,
    width: 1000,
    height: 100,
    toJSON: () => ({}),
  })
  return plot
}

const popover = () => document.querySelector('.chart-popover')

describe('LineChart', () => {
  it('draws labelled round value ticks and calendar gridlines', () => {
    renderChart()
    const labels = [...document.querySelectorAll('.chart-y-axis > .chart-tick')]
    expect(labels.map((label) => label.textContent)).toEqual([
      't0',
      't20',
      't40',
      't60',
      't80',
      't100',
    ])
    // Six value lines and three month starts (February to April).
    expect(document.querySelectorAll('.chart-grid line')).toHaveLength(9)
    const months = [...document.querySelectorAll('.chart-x-axis > .chart-tick')]
    expect(months.map((label) => label.textContent)).toEqual([
      'Feb',
      'Mar',
      'Apr',
    ])
  })

  it('picks the nearest point in screen pixels', async () => {
    const user = userEvent.setup()
    const plot = renderChart()
    // At x 50 and the top: 15 units from "a" but 100 from "b" in the chart's
    // own proportions, while on screen "a" is 150 px away and "b" 100 px.
    await user.pointer({ target: plot, coords: { clientX: 500, clientY: 0 } })
    expect(popover()).toHaveTextContent('Feb 20, 2026Normalv0')
    expect(document.querySelectorAll('.chart-dot-active')).toHaveLength(1)
    expect(plot).toHaveAttribute('aria-valuetext', 'Feb 20, 2026 · Normal · v0')

    await user.pointer({ target: plot, coords: { clientX: 660, clientY: 0 } })
    expect(popover()).toHaveTextContent('Mar 7, 2026Premiumv100')
  })

  it('names the series only when the chart has a legend', async () => {
    const user = userEvent.setup()
    const plot = renderChart(false)
    await user.pointer({ target: plot, coords: { clientX: 0, clientY: 100 } })
    expect(popover()).toHaveTextContent(/^Jan 1, 2026v0$/)
  })

  it('hides the popover when the mouse leaves', async () => {
    const user = userEvent.setup()
    const plot = renderChart()
    await user.pointer({ target: plot, coords: { clientX: 0, clientY: 100 } })
    expect(popover()).not.toBeNull()
    await user.pointer({ target: document.body })
    expect(popover()).toBeNull()
  })

  it('keeps a tapped value until a tap outside the chart', async () => {
    const user = userEvent.setup()
    const plot = renderChart()
    await user.pointer([
      { keys: '[TouchA>]', target: plot, coords: { clientX: 0, clientY: 100 } },
      { pointerName: 'TouchA', coords: { clientX: 1000, clientY: 50 } },
      { keys: '[/TouchA]' },
    ])
    expect(popover()).toHaveTextContent('Apr 11, 2026Normalv50')
    // Popover at the right edge, clamped by CSS from its anchor.
    expect((popover() as HTMLElement).style.getPropertyValue('--x')).toBe(
      '100%',
    )
    await user.pointer({ keys: '[TouchA]', target: document.body })
    expect(popover()).toBeNull()
  })

  it('steps through the points by date from the keyboard', async () => {
    const user = userEvent.setup()
    const plot = renderChart()
    await user.tab()
    expect(plot).toHaveFocus()
    expect(popover()).toHaveTextContent('Jan 1, 2026')

    await user.keyboard('{ArrowRight}')
    expect(popover()).toHaveTextContent('Feb 20, 2026')
    await user.keyboard('{ArrowRight}')
    expect(popover()).toHaveTextContent('Mar 7, 2026Premium')
    await user.keyboard('{End}')
    expect(popover()).toHaveTextContent('Apr 11, 2026')
    await user.keyboard('{ArrowRight}')
    expect(popover()).toHaveTextContent('Apr 11, 2026')
    await user.keyboard('{Home}')
    expect(popover()).toHaveTextContent('Jan 1, 2026')

    await user.keyboard('{Escape}')
    expect(popover()).toBeNull()
    expect(plot).toHaveFocus()

    await user.keyboard('{ArrowRight}')
    await user.tab()
    expect(popover()).toBeNull()
  })

  it('describes the keyboard use', () => {
    const plot = renderChart()
    expect(plot).toHaveAccessibleDescription(
      'Use the left and right arrow keys to step through the values.',
    )
  })
})
