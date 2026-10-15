import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { Car, Refuel, Repair, Ticket } from '#/api/client'
import { i18n } from '#/i18n'
import { dateRangeToInstants, defaultFrom } from '#/lib/date-range'
import { renderApp } from '#/test/render-app'

// The viewer's PDF rendering is covered in the attachment tests.
vi.mock('#/attachments/pdf', () => ({
  getDocument: () => ({
    promise: new Promise(() => {}),
    destroy: async () => {},
  }),
}))

const profile = {
  id: '617c3d87-21b4-4cb9-96f3-e03510892296',
  email: 'driver@example.com',
  firstName: 'Ada',
  lastName: 'Lovelace',
  currency: 'EUR',
  volumeUnit: 'liter',
  distanceUnit: 'kilometer',
}

const golf: Car = {
  id: '9b0a5f0e-5d8f-4a55-9d59-0b8f1f3c2a10',
  attachmentCount: 0,
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
  attachmentCount: 0,
  carId: golf.id,
  date: '2026-09-01T10:00:00Z',
  station: 'Shell Wien',
  odometerReading: 1500,
  fuel: 'normal',
  liters: '40',
  amount: '60',
  perLiter: '1.5',
  distance: 500,
  consumption: '8',
}

const earlier: Refuel = {
  ...refuel,
  id: '33333333-3333-3333-3333-333333333332',
  date: '2026-08-01T10:00:00Z',
  odometerReading: 1000,
  distance: null,
  consumption: null,
}

const repair: Repair = {
  id: '4d7c2b1e-8a3f-4c6d-9e0f-1a2b3c4d5e6f',
  attachmentCount: 0,
  carId: golf.id,
  date: '2026-08-14T08:15:00Z',
  station: 'Garage Huber',
  odometerReading: 1200,
  type: 'service',
  amount: '120.50',
  description: '',
}

const ticket: Ticket = {
  id: '5e8d3c2f-9b4a-4d7e-8f1a-2b3c4d5e6f70',
  attachmentCount: 0,
  carId: golf.id,
  date: '2026-07-02T14:00:00Z',
  type: 'parking',
  location: 'Wien Mitte',
  amount: '36.00',
  description: '',
}

function response(status: number, body?: unknown) {
  return new Response(body === undefined ? null : JSON.stringify(body), {
    status,
    headers:
      body === undefined ? undefined : { 'Content-Type': 'application/json' },
  })
}

type Routes = Partial<
  Record<
    'car' | 'refuels' | 'chart' | 'repairs' | 'tickets' | 'attachments',
    () => Response
  >
>

// Answers by path without its query, so tests can inspect the query in the
// recorded calls; the session is always valid.
function backend(routes: Routes = {}) {
  const calls: string[] = []
  const page = (items: unknown[]) => () =>
    response(200, { items, nextCursor: null })
  const handlers: Record<string, () => Response> = {
    [`/api/v1/cars/${golf.id}`]: routes.car ?? (() => response(200, golf)),
    '/api/v1/refuels': routes.refuels ?? page([refuel]),
    '/api/v1/refuels/chart':
      routes.chart ?? (() => response(200, { items: [earlier, refuel] })),
    '/api/v1/repairs': routes.repairs ?? page([repair]),
    '/api/v1/tickets': routes.tickets ?? page([ticket]),
    [`/api/v1/cars/${golf.id}/attachments`]: routes.attachments ?? page([]),
  }
  vi.stubGlobal(
    'fetch',
    vi.fn(async (path: string, init?: RequestInit) => {
      if ((init?.method ?? 'GET') !== 'GET') {
        throw new Error(`unexpected ${init?.method} ${path}`)
      }
      calls.push(path)
      if (path === '/api/v1/session') return response(200, { profile })
      const handler = handlers[path.split('?')[0]]
      if (!handler) throw new Error(`unexpected request ${path}`)
      return handler()
    }),
  )
  return calls
}

function query(calls: string[], path: string) {
  const call = calls.find((candidate) => candidate.split('?')[0] === path)
  return call === undefined
    ? undefined
    : Object.fromEntries(new URLSearchParams(call.split('?')[1]))
}

const expenseCalls = (calls: string[]) =>
  calls.filter((call) => /\/api\/v1\/(refuels|repairs|tickets)/.test(call))

beforeEach(async () => {
  await i18n.changeLanguage('en')
  document.documentElement.lang = 'en'
  vi.unstubAllGlobals()
})

describe('car attachments', () => {
  it("lists the car's own attachments on the Details tab", async () => {
    backend({
      attachments: () =>
        response(200, {
          items: [
            {
              id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
              filename: 'registration.pdf',
              contentType: 'application/pdf',
              size: 2048,
              createdAt: '2026-01-01T00:00:00Z',
            },
          ],
          nextCursor: null,
        }),
    })
    renderApp(`/cars/${golf.id}`)

    const section = await screen.findByRole('region', { name: 'Attachments' })
    const open = await within(section).findByRole('button', {
      name: 'registration.pdf',
    })
    expect(within(section).getByText('PDF document')).toBeInTheDocument()

    await userEvent.setup().click(open)
    expect(
      await screen.findByRole('dialog', { name: 'registration.pdf' }),
    ).toBeInTheDocument()
  })

  it('says when the car has no attachments, in both languages', async () => {
    backend()
    renderApp(`/cars/${golf.id}`)
    expect(await screen.findByText('No attachments.')).toBeInTheDocument()
    await i18n.changeLanguage('de')
    expect(await screen.findByText('Keine Anhänge.')).toBeInTheDocument()
  })
})

describe('car detail tabs', () => {
  it('opens on Details without requesting expenses', async () => {
    const calls = backend()
    renderApp(`/cars/${golf.id}`)

    const tab = await screen.findByRole('tab', { name: 'Details' })
    expect(tab).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByText('Hatchback')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Edit car' })).toHaveAttribute(
      'href',
      `/cars/${golf.id}/edit`,
    )
    expect(screen.queryByLabelText('From')).not.toBeInTheDocument()
    expect(expenseCalls(calls)).toEqual([])
  })

  it('switches tabs through the URL', async () => {
    backend()
    const user = userEvent.setup()
    const { router } = renderApp(`/cars/${golf.id}`)

    await user.click(await screen.findByRole('tab', { name: 'Expenses' }))
    await waitFor(() =>
      expect(router.state.location.search).toEqual({ tab: 'expenses' }),
    )
    expect(
      await screen.findByRole('heading', { name: 'Repairs' }),
    ).toBeInTheDocument()

    router.history.back()
    await waitFor(() =>
      expect(screen.getByRole('tab', { name: 'Details' })).toHaveAttribute(
        'aria-selected',
        'true',
      ),
    )
  })

  it('opens the tab named in the URL', async () => {
    backend()
    renderApp(`/cars/${golf.id}?tab=consumption`)

    expect(
      await screen.findByRole('tab', { name: 'Consumption' }),
    ).toHaveAttribute('aria-selected', 'true')
    expect(
      await screen.findByRole('slider', { name: 'Consumption' }),
    ).toBeInTheDocument()
  })

  it('falls back to the defaults for an unknown tab or a malformed date', async () => {
    const calls = backend()
    const user = userEvent.setup()
    renderApp(`/cars/${golf.id}?tab=bogus&from=2026-02-30&to=soon`)

    expect(await screen.findByRole('tab', { name: 'Details' })).toHaveAttribute(
      'aria-selected',
      'true',
    )
    expect(expenseCalls(calls)).toEqual([])

    await user.click(screen.getByRole('tab', { name: 'Expenses' }))
    expect(await screen.findByLabelText('From')).toHaveValue(defaultFrom())
    expect(screen.getByLabelText('To')).toHaveValue('')
    await waitFor(() =>
      expect(query(calls, '/api/v1/repairs')).toEqual({
        carId: golf.id,
        ...dateRangeToInstants(defaultFrom()),
      }),
    )
  })

  it('shows no tabs for a car that cannot be found', async () => {
    backend({
      car: () =>
        response(404, { code: 'not_found', message: 'resource not found' }),
    })
    renderApp(`/cars/${golf.id}?tab=expenses`)

    expect(await screen.findByRole('alert')).toHaveTextContent('Car not found')
    expect(screen.queryByRole('tab')).not.toBeInTheDocument()
  })
})

describe('car detail date range', () => {
  it('requests the last six months by default', async () => {
    const calls = backend()
    renderApp(`/cars/${golf.id}?tab=expenses`)

    await screen.findByRole('heading', { name: 'Tickets' })
    const expected = {
      carId: golf.id,
      ...dateRangeToInstants(defaultFrom()),
    }
    expect(screen.getByLabelText('From')).toHaveValue(defaultFrom())
    expect(screen.getByLabelText('To')).toHaveValue('')
    for (const path of [
      '/api/v1/refuels',
      '/api/v1/refuels/chart',
      '/api/v1/repairs',
      '/api/v1/tickets',
    ]) {
      await waitFor(() => expect(query(calls, path)).toEqual(expected))
    }
  })

  it('writes an edited range to the URL and requests it', async () => {
    const calls = backend()
    const { router } = renderApp(`/cars/${golf.id}?tab=expenses`)

    await screen.findByRole('heading', { name: 'Tickets' })
    // A date picker reports the whole date at once.
    fireEvent.change(screen.getByLabelText('From'), {
      target: { value: '2026-01-01' },
    })
    fireEvent.change(screen.getByLabelText('To'), {
      target: { value: '2026-01-31' },
    })

    await waitFor(() =>
      expect(router.state.location.search).toEqual({
        tab: 'expenses',
        from: '2026-01-01',
        to: '2026-01-31',
      }),
    )
    const expected = {
      carId: golf.id,
      ...dateRangeToInstants('2026-01-01', '2026-01-31'),
    }
    await waitFor(() =>
      expect(
        calls.some(
          (call) =>
            call.startsWith('/api/v1/tickets?') &&
            JSON.stringify(
              Object.fromEntries(new URLSearchParams(call.split('?')[1])),
            ) === JSON.stringify(expected),
        ),
      ).toBe(true),
    )
  })

  it('keeps the range when switching to Consumption', async () => {
    const calls = backend()
    const user = userEvent.setup()
    const { router } = renderApp(
      `/cars/${golf.id}?tab=expenses&from=2026-01-01&to=2026-06-30`,
    )

    await user.click(await screen.findByRole('tab', { name: 'Consumption' }))
    await waitFor(() =>
      expect(router.state.location.search).toEqual({
        tab: 'consumption',
        from: '2026-01-01',
        to: '2026-06-30',
      }),
    )
    expect(screen.getByLabelText('From')).toHaveValue('2026-01-01')
    expect(query(calls, '/api/v1/refuels/chart')).toEqual({
      carId: golf.id,
      ...dateRangeToInstants('2026-01-01', '2026-06-30'),
    })
  })
})

describe('car detail expenses', () => {
  it("lists the car's expenses without a car column and links to add more", async () => {
    backend()
    renderApp(`/cars/${golf.id}?tab=expenses`)

    const refuels = (
      await screen.findByRole('heading', { name: 'Refuels' })
    ).closest('section') as HTMLElement
    expect(await within(refuels).findByText('Shell Wien')).toBeInTheDocument()
    expect(within(refuels).getByText('8 l/100 km')).toBeInTheDocument()
    expect(
      within(refuels).queryByRole('columnheader', { name: 'Car' }),
    ).not.toBeInTheDocument()
    expect(
      within(refuels).getByRole('link', { name: 'Add refuel' }),
    ).toHaveAttribute('href', `/refuels/create?carId=${golf.id}`)

    const repairs = screen
      .getByRole('heading', { name: 'Repairs' })
      .closest('section') as HTMLElement
    expect(await within(repairs).findByText('Garage Huber')).toBeInTheDocument()
    expect(
      within(repairs).getByRole('link', { name: 'Add repair' }),
    ).toHaveAttribute('href', `/repairs/create?carId=${golf.id}`)

    const tickets = screen
      .getByRole('heading', { name: 'Tickets' })
      .closest('section') as HTMLElement
    expect(await within(tickets).findByText('Wien Mitte')).toBeInTheDocument()
    expect(
      within(tickets).getByRole('link', { name: 'Add ticket' }),
    ).toHaveAttribute('href', `/tickets/create?carId=${golf.id}`)
    expect(screen.queryByText(/Total of the/)).not.toBeInTheDocument()
  })

  it('keeps the other sections when one fails', async () => {
    backend({
      repairs: () =>
        response(500, { code: 'internal', message: 'unexpected error' }),
    })
    renderApp(`/cars/${golf.id}?tab=expenses`)

    const repairs = (
      await screen.findByRole('heading', { name: 'Repairs' })
    ).closest('section') as HTMLElement
    expect(await within(repairs).findByRole('alert')).toHaveTextContent(
      'The repairs could not be loaded.',
    )
    expect(
      within(repairs).getByRole('button', { name: 'Try again' }),
    ).toBeInTheDocument()
    expect(await screen.findByText('Shell Wien')).toBeInTheDocument()
    expect(screen.getByText('Wien Mitte')).toBeInTheDocument()
  })

  it('shows an empty message for a section without records', async () => {
    backend({
      tickets: () => response(200, { items: [], nextCursor: null }),
    })
    renderApp(`/cars/${golf.id}?tab=expenses`)

    expect(
      await screen.findByText('No tickets in this date range.'),
    ).toBeInTheDocument()
  })

  it('appends a further page to its own section only', async () => {
    const second: Repair = {
      ...repair,
      id: '4d7c2b1e-8a3f-4c6d-9e0f-1a2b3c4d5e70',
      station: 'Autohaus Maier',
    }
    const pages = [
      response(200, { items: [repair], nextCursor: 'next' }),
      response(200, { items: [second], nextCursor: null }),
    ]
    const calls = backend({ repairs: () => pages.shift() as Response })
    const user = userEvent.setup()
    renderApp(`/cars/${golf.id}?tab=expenses`)

    await user.click(
      await screen.findByRole('button', { name: 'Load more repairs' }),
    )
    expect(await screen.findByText('Autohaus Maier')).toBeInTheDocument()
    expect(screen.getByText('Garage Huber')).toBeInTheDocument()
    expect(
      query(
        calls.filter((c) => c.includes('cursor=')),
        '/api/v1/repairs',
      ),
    ).toMatchObject({ cursor: 'next', carId: golf.id })
    expect(
      calls.filter((call) => call.startsWith('/api/v1/tickets')),
    ).toHaveLength(1)
  })
})

describe('car detail consumption', () => {
  it('charts the car and shows a retry on failure', async () => {
    let fail = true
    backend({
      chart: () =>
        fail
          ? response(500, { code: 'internal', message: 'unexpected error' })
          : response(200, { items: [earlier, refuel] }),
    })
    const user = userEvent.setup()
    renderApp(`/cars/${golf.id}?tab=consumption`)

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'The consumption history could not be loaded.',
    )
    fail = false
    await user.click(screen.getByRole('button', { name: 'Try again' }))
    expect(
      await screen.findByRole('slider', { name: 'Consumption' }),
    ).toBeInTheDocument()
  })

  it('shows the empty state in German', async () => {
    backend({ chart: () => response(200, { items: [earlier] }) })
    await i18n.changeLanguage('de')
    renderApp(`/cars/${golf.id}?tab=consumption`)

    expect(
      await screen.findByRole('tab', { name: 'Verbrauch' }),
    ).toHaveAttribute('aria-selected', 'true')
    expect(
      await screen.findByText(
        'Nicht genug Tankvorgänge mit Kilometerstand in diesem Zeitraum.',
      ),
    ).toBeInTheDocument()
    expect(screen.getByLabelText('Von')).toBeInTheDocument()
  })
})
