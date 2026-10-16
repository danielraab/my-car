import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it } from 'vitest'

import type { Car, Refuel } from '#/api/client'
import { i18n } from '#/i18n'

import { FuelPriceChart } from './fuel-price-chart'

const diesel = { fuel: 'diesel' } as Car
const gasoline = { fuel: 'gasoline' } as Car
const cars = new Map([
  ['22222222-2222-2222-2222-222222222222', diesel],
  ['22222222-2222-2222-2222-222222222223', gasoline],
  ['22222222-2222-2222-2222-222222222224', diesel],
  ['22222222-2222-2222-2222-222222222225', { fuel: 'electric' } as Car],
])

const base: Refuel = {
  id: '33333333-3333-3333-3333-333333333331',
  attachmentCount: 0,
  carId: '22222222-2222-2222-2222-222222222222',
  date: '2026-01-10T10:00:00Z',
  station: 'Fuel',
  odometerReading: null,
  fuel: 'normal',
  quantity: '40',
  amount: '60',
  unitPrice: '1.5',
  distance: null,
  consumption: null,
}
const items: Refuel[] = [
  base,
  {
    ...base,
    id: '33333333-3333-3333-3333-333333333332',
    date: '2026-02-10T10:00:00Z',
    unitPrice: '1.62',
  },
  {
    ...base,
    id: '33333333-3333-3333-3333-333333333333',
    date: '2026-03-10T10:00:00Z',
    fuel: 'special',
    unitPrice: '1.749',
  },
]

describe('FuelPriceChart', () => {
  it('draws one line per car fuel and fuel with axis labels and a legend', () => {
    const { container } = render(
      <FuelPriceChart items={items} cars={cars} label="Fuel price" />,
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
    // Only diesel normal has two points, so only it draws a line.
    expect(container.querySelectorAll('polyline')).toHaveLength(1)
    expect(container.querySelectorAll('.chart-dot')).toHaveLength(3)
    const legend = screen.getAllByRole('listitem').map((li) => li.textContent)
    expect(legend).toEqual(['Diesel · Normal', 'Diesel · Premium'])
  })

  it('names the car fuel and fuel in the popover', async () => {
    const user = userEvent.setup()
    render(<FuelPriceChart items={items} cars={cars} label="Fuel price" />)
    await user.tab()
    await user.keyboard('{End}')
    expect(document.querySelector('.chart-popover')).toHaveTextContent(
      'Mar 10, 2026Diesel · Premium€1.749',
    )
  })

  it('shows the empty message without data', () => {
    render(<FuelPriceChart items={[]} cars={cars} label="Fuel price" />)
    expect(screen.queryByRole('slider')).not.toBeInTheDocument()
    expect(screen.getByText(/no refuel/i)).toBeInTheDocument()
  })

  it('splits the same fuel by car fuel and joins cars with the same fuel', () => {
    const { container } = render(
      <FuelPriceChart
        items={[
          base,
          {
            ...base,
            id: '33333333-3333-3333-3333-333333333334',
            carId: '22222222-2222-2222-2222-222222222224',
            date: '2026-02-10T10:00:00Z',
          },
          {
            ...base,
            id: '33333333-3333-3333-3333-333333333335',
            carId: '22222222-2222-2222-2222-222222222223',
            date: '2026-03-10T10:00:00Z',
          },
        ]}
        cars={cars}
        label="Fuel price"
      />,
    )
    const legend = screen.getAllByRole('listitem').map((li) => li.textContent)
    expect(legend).toEqual(['Gasoline · Normal', 'Diesel · Normal'])
    // Both diesel cars' refuels form one line.
    expect(container.querySelectorAll('polyline')).toHaveLength(1)
  })

  it("keeps a pair's colour class whatever else is shown", () => {
    const { container, rerender } = render(
      <FuelPriceChart items={items} cars={cars} label="Fuel price" />,
    )
    const premium = () =>
      container.querySelector('.chart-fuel-diesel-special') !== null
    expect(premium()).toBe(true)
    rerender(
      <FuelPriceChart items={items.slice(2)} cars={cars} label="Fuel price" />,
    )
    expect(premium()).toBe(true)
  })

  it('leaves out refuels whose car is unknown', () => {
    render(
      <FuelPriceChart
        items={[{ ...base, carId: '99999999-9999-9999-9999-999999999999' }]}
        cars={cars}
        label="Fuel price"
      />,
    )
    expect(screen.queryByRole('slider')).not.toBeInTheDocument()
  })

  describe('with charges of an electric car', () => {
    afterEach(async () => {
      await i18n.changeLanguage('en')
    })

    const electricCar = '22222222-2222-2222-2222-222222222225'
    const charges: Refuel[] = [
      {
        ...base,
        id: '33333333-3333-3333-3333-333333333341',
        carId: electricCar,
        date: '2026-01-12T10:00:00Z',
        fuel: 'home',
        unitPrice: '0.28',
      },
      {
        ...base,
        id: '33333333-3333-3333-3333-333333333342',
        carId: electricCar,
        date: '2026-02-12T10:00:00Z',
        fuel: 'dc',
        unitPrice: '0.69',
      },
    ]
    const axisTicks = (container: HTMLElement, selector: string) =>
      [...container.querySelectorAll(`${selector} > .chart-tick`)].map((tick) =>
        Number(tick.textContent),
      )

    it('plots charges per kWh on a right axis beside fuel prices', () => {
      const { container } = render(
        <FuelPriceChart
          items={[...items.slice(0, 2), ...charges]}
          cars={cars}
          label="Fuel price"
        />,
      )
      const legend = screen.getAllByRole('listitem').map((li) => li.textContent)
      expect(legend).toEqual([
        'Diesel · Normal',
        'Electric · Home',
        'Electric · DC fast',
      ])
      const left = axisTicks(
        container,
        '.chart-y-axis:not(.chart-y-axis-right)',
      )
      const right = axisTicks(container, '.chart-y-axis-right')
      expect(Math.min(...left)).toBeGreaterThanOrEqual(1.5)
      expect(Math.max(...right)).toBeLessThan(1)
      expect(screen.getByText('€/l')).toBeInTheDocument()
      expect(screen.getByText('€/kWh')).toBeInTheDocument()
      expect(container.querySelector('.chart-fuel-electric-dc')).not.toBeNull()
    })

    it('names the unit of each price in the popover', async () => {
      const user = userEvent.setup()
      render(
        <FuelPriceChart
          items={[...items.slice(0, 2), ...charges]}
          cars={cars}
          label="Fuel price"
        />,
      )
      await user.tab()
      await user.keyboard('{End}')
      expect(document.querySelector('.chart-popover')).toHaveTextContent(
        'Feb 12, 2026Electric · DC fast€0.690/kWh',
      )
      await user.keyboard('{Home}')
      expect(document.querySelector('.chart-popover')).toHaveTextContent(
        'Diesel · Normal€1.500/l',
      )
    })

    it('uses the left axis alone for an electric car only', () => {
      const { container } = render(
        <FuelPriceChart items={charges} cars={cars} label="Fuel price" />,
      )
      expect(container.querySelector('.chart-y-axis-right')).toBeNull()
      expect(screen.queryByText('€/kWh')).toBeNull()
      expect(Math.max(...axisTicks(container, '.chart-y-axis'))).toBeLessThan(1)
    })

    it('labels charging types in German', async () => {
      await i18n.changeLanguage('de')
      render(<FuelPriceChart items={charges} cars={cars} label="Preise" />)
      const legend = screen.getAllByRole('listitem').map((li) => li.textContent)
      expect(legend).toEqual(['Elektro · Zuhause', 'Elektro · DC-Schnellladen'])
    })
  })
})
