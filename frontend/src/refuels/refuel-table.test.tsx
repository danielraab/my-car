import { screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { Car, Refuel } from '#/api/client'
import '#/i18n'
import { renderInRouter } from '#/test/render-in-router'

import { RefuelTable } from './refuel-table'

const golf: Car = {
  id: '22222222-2222-2222-2222-222222222222',
  type: 'Hatchback',
  make: 'VW',
  name: 'Golf',
  fuel: 'diesel',
  firstRegistration: '2019-03-01',
  licensePlate: 'W-123AB',
  fin: null,
  isActive: true,
  purchaseDate: null,
  purchasePrice: null,
  deregistrationDate: null,
  information: null,
  createdAt: '2026-09-27T10:00:00Z',
  updatedAt: '2026-09-27T10:00:00Z',
}

const refuel: Refuel = {
  id: '33333333-3333-3333-3333-333333333333',
  carId: '22222222-2222-2222-2222-222222222222',
  date: '2026-09-28T10:00:00Z',
  station: 'Fuel',
  odometerReading: 1500,
  fuel: 'normal',
  liters: '40',
  amount: '60',
  perLiter: '1.5',
  distance: null,
  consumption: null,
}

describe('RefuelTable', () => {
  it('shows the car column and total when given', async () => {
    await renderInRouter(
      <RefuelTable
        items={[refuel]}
        derived={new Map([[refuel.id, { ...refuel, consumption: '8' }]])}
        cars={new Map([[refuel.carId, golf]])}
        total={60}
      />,
    )
    expect(
      await screen.findByRole('columnheader', { name: 'Car' }),
    ).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'VW Golf' })).toBeInTheDocument()
    expect(screen.getByText('Total of the refuels shown')).toHaveAttribute(
      'colspan',
      '7',
    )
    expect(screen.getByText('8 l/100 km')).toBeInTheDocument()
  })

  it('hides the car column and total when left out', async () => {
    await renderInRouter(<RefuelTable items={[refuel]} derived={new Map()} />)
    expect(await screen.findByRole('table')).toBeInTheDocument()
    expect(
      screen.queryByRole('columnheader', { name: 'Car' }),
    ).not.toBeInTheDocument()
    expect(screen.getAllByRole('columnheader')).toHaveLength(8)
    expect(
      screen.queryByText('Total of the refuels shown'),
    ).not.toBeInTheDocument()
  })

  it.each([
    [{ isActive: false }, 'Inactive'],
    [{ deregistrationDate: '2020-01-01' }, 'Deregistered'],
  ])('mutes and badges the row of a car with %o', async (change, badge) => {
    await renderInRouter(
      <RefuelTable
        items={[refuel]}
        derived={new Map()}
        cars={new Map([[refuel.carId, { ...golf, ...change }]])}
      />,
    )
    const row = (await screen.findAllByRole('row'))[1]
    expect(row).toHaveClass('row-muted')
    expect(row).toHaveTextContent(badge)
    expect(screen.getByRole('link', { name: 'VW Golf' })).toBeInTheDocument()
  })

  it('does not mute the row of an active car', async () => {
    await renderInRouter(
      <RefuelTable
        items={[refuel]}
        derived={new Map()}
        cars={new Map([[refuel.carId, golf]])}
      />,
    )
    const row = (await screen.findAllByRole('row'))[1]
    expect(row).not.toHaveClass('row-muted')
    expect(row).not.toHaveTextContent(/Inactive|Deregistered/)
  })
})
