import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'

import type { Refuel } from '#/api/client'
import '#/i18n'

import { FuelPriceChart } from './fuel-price-chart'

const base: Refuel = {
  id: '33333333-3333-3333-3333-333333333331',
  attachmentCount: 0,
  carId: '22222222-2222-2222-2222-222222222222',
  date: '2026-01-10T10:00:00Z',
  station: 'Fuel',
  odometerReading: null,
  fuel: 'normal',
  liters: '40',
  amount: '60',
  perLiter: '1.5',
  distance: null,
  consumption: null,
}
const items: Refuel[] = [
  base,
  {
    ...base,
    id: '33333333-3333-3333-3333-333333333332',
    date: '2026-02-10T10:00:00Z',
    perLiter: '1.62',
  },
  {
    ...base,
    id: '33333333-3333-3333-3333-333333333333',
    date: '2026-03-10T10:00:00Z',
    fuel: 'special',
    perLiter: '1.749',
  },
]

describe('FuelPriceChart', () => {
  it('draws one line per fuel with axis labels and a legend', () => {
    const { container } = render(
      <FuelPriceChart items={items} label="Fuel price" />,
    )
    expect(
      screen.getByRole('slider', { name: 'Fuel price' }),
    ).toBeInTheDocument()
    const ticks = (axis: string) =>
      [...container.querySelectorAll(`.${axis} > .chart-tick`)].map(
        (tick) => tick.textContent,
      )
    expect(ticks('chart-y-axis')).toEqual([
      '1.50',
      '1.55',
      '1.60',
      '1.65',
      '1.70',
      '1.75',
    ])
    // Mondays from 12 January to 9 March.
    const weeks = ticks('chart-x-axis')
    expect(weeks).toHaveLength(9)
    expect([weeks[0], weeks.at(-1)]).toEqual(['Jan 12', 'Mar 9'])
    // Only the normal fuel has two points, so only it draws a line.
    expect(container.querySelectorAll('polyline')).toHaveLength(1)
    expect(container.querySelectorAll('.chart-dot')).toHaveLength(3)
    const legend = screen.getAllByRole('listitem').map((li) => li.textContent)
    expect(legend).toEqual(['Normal', 'Premium'])
  })

  it('names the fuel in the popover', async () => {
    const user = userEvent.setup()
    render(<FuelPriceChart items={items} label="Fuel price" />)
    await user.tab()
    await user.keyboard('{End}')
    expect(document.querySelector('.chart-popover')).toHaveTextContent(
      'Mar 10, 2026Premium1.749',
    )
  })

  it('shows the empty message without data', () => {
    render(<FuelPriceChart items={[]} label="Fuel price" />)
    expect(screen.queryByRole('slider')).not.toBeInTheDocument()
    expect(screen.getByText(/no refuel/i)).toBeInTheDocument()
  })
})
