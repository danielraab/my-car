import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it } from 'vitest'

import type { Refuel } from '#/api/client'
import { i18n } from '#/i18n'
import { DisplayUnitsContext } from '#/lib/display-units'

import { ConsumptionChart } from './consumption-chart'

const first: Refuel = {
  id: '33333333-3333-3333-3333-333333333331',
  attachmentCount: 0,
  carId: '22222222-2222-2222-2222-222222222222',
  date: '2026-01-10T10:00:00Z',
  station: 'Fuel',
  odometerReading: 1000,
  fuel: 'normal',
  liters: '40',
  amount: '60',
  perLiter: '1.5',
  distance: null,
  consumption: null,
}
const items: Refuel[] = [
  first,
  {
    ...first,
    id: '33333333-3333-3333-3333-333333333332',
    date: '2026-02-10T10:00:00Z',
    odometerReading: 1500,
    distance: 500,
    consumption: '8',
  },
  {
    ...first,
    id: '33333333-3333-3333-3333-333333333333',
    date: '2026-03-10T10:00:00Z',
    odometerReading: 2100,
    distance: 600,
    consumption: '6.67',
  },
]

afterEach(async () => {
  await i18n.changeLanguage('en')
})

describe('ConsumptionChart', () => {
  it('plots only refuels with a consumption', () => {
    const { container } = render(
      <ConsumptionChart items={items} label="Consumption" />,
    )
    expect(
      screen.getByRole('slider', { name: 'Consumption' }),
    ).toBeInTheDocument()
    expect(container.querySelectorAll('.chart-dot')).toHaveLength(2)
    expect(container.querySelectorAll('polyline')).toHaveLength(1)
    expect(screen.queryByRole('list')).not.toBeInTheDocument()
  })

  it('labels round value ticks and weekly gridlines', () => {
    const { container } = render(
      <ConsumptionChart items={items} label="Consumption" />,
    )
    const ticks = (axis: string) =>
      [...container.querySelectorAll(`.${axis} > .chart-tick`)].map(
        (tick) => tick.textContent,
      )
    expect(ticks('chart-y-axis')).toEqual(['6.5', '7.0', '7.5', '8.0'])
    // Mondays between 10 February and 10 March.
    expect(ticks('chart-x-axis')).toEqual([
      'Feb 16',
      'Feb 23',
      'Mar 2',
      'Mar 9',
    ])
  })

  it('names the account units without converting the values', async () => {
    const user = userEvent.setup()
    render(
      <DisplayUnitsContext.Provider
        value={{ currency: 'USD', volumeUnit: 'gallon', distanceUnit: 'mile' }}
      >
        <ConsumptionChart items={items} label="Consumption" />
      </DisplayUnitsContext.Provider>,
    )
    await user.tab()
    expect(document.querySelector('.chart-popover')).toHaveTextContent(
      'Feb 10, 20268 gal/100 mi',
    )
  })

  it('shows each refuel in a popover from the keyboard', async () => {
    const user = userEvent.setup()
    render(<ConsumptionChart items={items} label="Consumption" />)
    await user.tab()
    const popover = () => document.querySelector('.chart-popover')
    expect(popover()).toHaveTextContent('Feb 10, 20268 l/100 km')
    await user.keyboard('{ArrowRight}')
    expect(popover()).toHaveTextContent('Mar 10, 20266.67 l/100 km')
    expect(screen.getByRole('slider')).toHaveAttribute(
      'aria-valuetext',
      'Mar 10, 2026 · 6.67 l/100 km',
    )
  })

  it('shows the empty state when no refuel has a consumption', () => {
    render(<ConsumptionChart items={[first]} label="Consumption" />)
    expect(screen.queryByRole('slider')).not.toBeInTheDocument()
    expect(
      screen.getByText(
        'Not enough refuels with odometer readings in this range.',
      ),
    ).toBeInTheDocument()
  })

  it('formats values and dates in German', async () => {
    await i18n.changeLanguage('de')
    const user = userEvent.setup()
    const { container } = render(
      <ConsumptionChart items={items} label="Verbrauch" />,
    )
    expect(
      container.querySelector('.chart-y-axis > .chart-tick'),
    ).toHaveTextContent('6,5')
    await user.tab()
    await user.keyboard('{End}')
    expect(document.querySelector('.chart-popover')).toHaveTextContent(
      '10.03.20266,67 l/100 km',
    )
    expect(screen.getByRole('slider')).toHaveAccessibleDescription(
      'Mit den Pfeiltasten links und rechts gehst du die Werte durch.',
    )
  })
})
