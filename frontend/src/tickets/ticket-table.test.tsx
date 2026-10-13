import { screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { Car, Ticket } from '#/api/client'
import '#/i18n'
import { renderInRouter } from '#/test/render-in-router'

import { TicketTable } from './ticket-table'

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

const item: Ticket = {
  id: '4d7c2b1e-8a3f-4c6d-9e0f-1a2b3c4d5e6f',
  carId: '22222222-2222-2222-2222-222222222222',
  date: '2026-03-14T08:15:00Z',
  type: 'parking',
  location: 'Wien',
  amount: '36.00',
  description: '',
}

describe('TicketTable', () => {
  it('shows the car column and total when given', async () => {
    await renderInRouter(
      <TicketTable
        items={[item]}
        cars={new Map([[item.carId, golf]])}
        total={Number(item.amount)}
      />,
    )
    expect(
      await screen.findByRole('columnheader', { name: 'Car' }),
    ).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'VW Golf' })).toBeInTheDocument()
    expect(screen.getByText('Total of the tickets shown')).toHaveAttribute(
      'colspan',
      '4',
    )
  })

  it('hides the car column and total when left out', async () => {
    await renderInRouter(<TicketTable items={[item]} />)
    expect(await screen.findByRole('table')).toBeInTheDocument()
    expect(
      screen.queryByRole('columnheader', { name: 'Car' }),
    ).not.toBeInTheDocument()
    expect(screen.getAllByRole('columnheader')).toHaveLength(4)
    expect(
      screen.queryByText('Total of the tickets shown'),
    ).not.toBeInTheDocument()
  })

  it.each([
    [{ isActive: false }, 'Inactive'],
    [{ deregistrationDate: '2020-01-01' }, 'Deregistered'],
  ])('mutes and badges the row of a car with %o', async (change, badge) => {
    await renderInRouter(
      <TicketTable
        items={[item]}
        cars={new Map([[item.carId, { ...golf, ...change }]])}
      />,
    )
    const row = (await screen.findAllByRole('row'))[1]
    expect(row).toHaveClass('row-muted')
    expect(row).toHaveTextContent(badge)
    expect(screen.getByRole('link', { name: 'VW Golf' })).toBeInTheDocument()
  })

  it('does not mute the row of an active car', async () => {
    await renderInRouter(
      <TicketTable items={[item]} cars={new Map([[item.carId, golf]])} />,
    )
    const row = (await screen.findAllByRole('row'))[1]
    expect(row).not.toHaveClass('row-muted')
    expect(row).not.toHaveTextContent(/Inactive|Deregistered/)
  })
})
