import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { Car, Ticket } from '#/api/client'
import { i18n } from '#/i18n'
import { dateRangeToInstants, defaultFrom } from '#/lib/date-range'
import { renderApp } from '#/test/render-app'

const profile = {
  id: '617c3d87-21b4-4cb9-96f3-e03510892296',
  email: 'driver@example.com',
  firstName: 'Ada',
  lastName: 'Lovelace',
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

const zoe: Car = {
  ...golf,
  id: '3f1e8a52-7c2b-4d0e-9a61-5b4c3d2e1f00',
  make: 'Renault',
  name: 'Zoe',
}

const parking: Ticket = {
  id: '4d7c2b1e-8a3f-4c6d-9e0f-1a2b3c4d5e6f',
  attachmentCount: 0,
  carId: golf.id,
  date: '2026-03-14T08:15:00Z',
  type: 'parking',
  location: 'Wien Mariahilf',
  amount: '36.00',
  description: 'Short-term parking zone',
}

const speeding: Ticket = {
  ...parking,
  id: '5e8d3c2f-9b4a-4d7e-8f1a-2b3c4d5e6f70',
  carId: zoe.id,
  date: '2026-05-02T14:00:00Z',
  type: 'velocity',
  location: 'A1 Linz',
  amount: '70.50',
  description: '',
}

function response(status: number, body?: unknown) {
  return new Response(body === undefined ? null : JSON.stringify(body), {
    status,
    headers:
      body === undefined ? undefined : { 'Content-Type': 'application/json' },
  })
}

type Handler = (init?: RequestInit) => Response | Promise<Response>

// Routes fetch calls by "METHOD path"; the session is always valid.
// Routes match without the list's date range, which every list request
// carries; `url` keeps the full request for the range tests.
function withoutRange(path: string) {
  const [pathname, search] = path.split('?')
  const query = new URLSearchParams(search)
  query.delete('from')
  query.delete('to')
  const rest = query.toString()
  return rest ? `${pathname}?${rest}` : pathname
}

function backend(routes: Record<string, Handler | Handler[]>) {
  const calls: { key: string; url: string; init?: RequestInit }[] = []
  const fetchMock = vi.fn(async (path: string, init?: RequestInit) => {
    const key = `${init?.method ?? 'GET'} ${withoutRange(path)}`
    calls.push({ key, url: path, init })
    if (key === 'GET /api/v1/session') return response(200, { profile })
    const route = routes[key]
    const handler = Array.isArray(route) ? route.shift() : route
    if (!handler) throw new Error(`unexpected request ${key}`)
    return handler(init)
  })
  vi.stubGlobal('fetch', fetchMock)
  return calls
}

// The query of the most recent tickets list request.
function listQuery(calls: ReturnType<typeof backend>) {
  const call = calls
    .filter((candidate) => candidate.url.split('?')[0] === '/api/v1/tickets')
    .at(-1)
  return call === undefined
    ? undefined
    : Object.fromEntries(new URLSearchParams(call.url.split('?')[1]))
}

function bodyOf(calls: ReturnType<typeof backend>, key: string) {
  const call = calls.find((candidate) => candidate.key === key)
  return JSON.parse(String(call?.init?.body))
}

// Dates are shown in the viewer's zone, whatever zone the tests run in.
function shown(iso: string, locale = 'en') {
  return new Intl.DateTimeFormat(locale, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(iso))
}

const allCars = {
  'GET /api/v1/cars?limit=100': () =>
    response(200, { items: [golf, zoe], nextCursor: null }),
}
const noLocations = { 'GET /api/v1/tickets/locations': () => response(200, []) }
const ticketPath = `/api/v1/tickets/${parking.id}`

beforeEach(async () => {
  await i18n.changeLanguage('en')
  document.documentElement.lang = 'en'
  vi.unstubAllGlobals()
})

describe('tickets list', () => {
  it('shows the documented columns, a running total and links', async () => {
    backend({
      ...allCars,
      'GET /api/v1/tickets': () =>
        response(200, { items: [parking, speeding], nextCursor: null }),
    })
    renderApp('/tickets')

    const table = await screen.findByRole('table', { name: 'Your tickets' })
    const headers = within(table)
      .getAllByRole('columnheader')
      .map((header) => header.textContent)
    expect(headers).toEqual([
      'Date and time',
      'Car',
      'Type',
      'Location',
      'Amount',
      'Attachments',
    ])
    const rows = within(table).getAllByRole('row')
    expect(rows).toHaveLength(4)
    expect(
      within(rows[1]).getByRole('link', { name: shown(parking.date) }),
    ).toHaveAttribute('href', `/tickets/${parking.id}/edit`)
    expect(
      await within(rows[1]).findByRole('link', { name: 'VW Golf' }),
    ).toHaveAttribute('href', `/cars/${golf.id}`)
    expect(rows[1]).toHaveTextContent('Parking')
    expect(rows[1]).toHaveTextContent('Wien Mariahilf')
    expect(rows[1]).toHaveTextContent('36.00')
    expect(rows[2]).toHaveTextContent('Renault Zoe')
    expect(rows[2]).toHaveTextContent('Speeding')
    expect(rows[3]).toHaveTextContent('Total of the tickets shown')
    expect(rows[3]).toHaveTextContent('106.50')
    expect(screen.getByRole('link', { name: 'Add ticket' })).toHaveAttribute(
      'href',
      '/tickets/create',
    )
    expect(
      screen.queryByRole('button', { name: 'Load more tickets' }),
    ).not.toBeInTheDocument()
  })

  it('appends a further page and includes it in the total', async () => {
    const calls = backend({
      ...allCars,
      'GET /api/v1/tickets': () =>
        response(200, { items: [parking], nextCursor: 'c1' }),
      'GET /api/v1/tickets?cursor=c1': () =>
        response(200, { items: [speeding], nextCursor: null }),
    })
    const user = userEvent.setup()
    renderApp('/tickets')

    const total = () => screen.getAllByRole('row').at(-1)
    await screen.findByText('Wien Mariahilf')
    expect(total()).toHaveTextContent('36.00')
    await user.click(screen.getByRole('button', { name: 'Load more tickets' }))

    expect(await screen.findByText('A1 Linz')).toBeInTheDocument()
    expect(screen.getByText('Wien Mariahilf')).toBeInTheDocument()
    expect(total()).toHaveTextContent('106.50')
    expect(
      screen.queryByRole('button', { name: 'Load more tickets' }),
    ).not.toBeInTheDocument()
    expect(calls.map((call) => call.key)).toContain(
      'GET /api/v1/tickets?cursor=c1',
    )
  })

  it('shows a retryable error when the tickets cannot be loaded', async () => {
    backend({
      ...allCars,
      'GET /api/v1/tickets': [
        () => response(503, { code: 'unavailable', message: 'down' }),
        () => response(200, { items: [parking], nextCursor: null }),
      ],
    })
    const user = userEvent.setup()
    renderApp('/tickets')

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Tickets unavailable')
    expect(screen.queryByRole('table')).not.toBeInTheDocument()

    await user.click(within(alert).getByRole('button', { name: 'Try again' }))
    expect(await screen.findByText('Wien Mariahilf')).toBeInTheDocument()
  })

  it('follows a switch to German', async () => {
    backend({
      ...allCars,
      'GET /api/v1/tickets': () =>
        response(200, { items: [speeding], nextCursor: null }),
    })
    const user = userEvent.setup()
    renderApp('/tickets')

    await screen.findByText('A1 Linz')
    await user.selectOptions(
      screen.getAllByRole('combobox', { name: 'Language' })[0],
      'de',
    )

    expect(
      await screen.findByRole('heading', { name: 'Strafzettel' }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('columnheader', { name: 'Ort' }),
    ).toBeInTheDocument()
    const row = screen.getAllByRole('row')[1]
    expect(row).toHaveTextContent(shown(speeding.date, 'de'))
    expect(row).toHaveTextContent('Geschwindigkeit')
    expect(row).toHaveTextContent('70,50')
    expect(
      screen.getByText('Summe der angezeigten Strafzettel'),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('link', { name: 'Strafzettel hinzufügen' }),
    ).toBeInTheDocument()
  })
  it('requests the last six months across all cars by default', async () => {
    const calls = backend({
      ...allCars,
      'GET /api/v1/tickets': () =>
        response(200, { items: [parking], nextCursor: null }),
    })
    renderApp('/tickets')

    await screen.findByText('Wien Mariahilf')
    expect(screen.getByLabelText('Filter by car')).toHaveValue('')
    expect(screen.getByLabelText('From')).toHaveValue(defaultFrom())
    expect(screen.getByLabelText('To')).toHaveValue('')
    expect(listQuery(calls)).toEqual(dateRangeToInstants(defaultFrom()))
  })

  it('filters by car and range and keeps both in the URL', async () => {
    const calls = backend({
      ...allCars,
      'GET /api/v1/tickets': () =>
        response(200, { items: [parking, speeding], nextCursor: null }),
      [`GET /api/v1/tickets?carId=${golf.id}`]: () =>
        response(200, { items: [parking], nextCursor: null }),
    })
    const user = userEvent.setup()
    const { router } = renderApp('/tickets')

    await screen.findByText('A1 Linz')
    await user.selectOptions(
      await screen.findByRole('combobox', { name: 'Filter by car' }),
      golf.id,
    )
    // A date picker reports the whole date at once.
    fireEvent.change(screen.getByLabelText('From'), {
      target: { value: '2026-01-01' },
    })
    fireEvent.change(screen.getByLabelText('To'), {
      target: { value: '2026-01-31' },
    })

    await waitFor(() =>
      expect(router.state.location.search).toEqual({
        carId: golf.id,
        from: '2026-01-01',
        to: '2026-01-31',
      }),
    )
    await waitFor(() =>
      expect(listQuery(calls)).toEqual({
        carId: golf.id,
        ...dateRangeToInstants('2026-01-01', '2026-01-31'),
      }),
    )
    await waitFor(() =>
      expect(screen.queryByText('A1 Linz')).not.toBeInTheDocument(),
    )
    expect(screen.getByRole('link', { name: 'Add ticket' })).toHaveAttribute(
      'href',
      `/tickets/create?carId=${golf.id}`,
    )
  })

  it('restores the car and range from the URL', async () => {
    const calls = backend({
      ...allCars,
      [`GET /api/v1/tickets?carId=${golf.id}`]: () =>
        response(200, { items: [parking], nextCursor: null }),
    })
    renderApp(`/tickets?carId=${golf.id}&from=2025-01-01`)

    await screen.findByText('Wien Mariahilf')
    await waitFor(() =>
      expect(screen.getByLabelText('Filter by car')).toHaveValue(golf.id),
    )
    expect(screen.getByLabelText('From')).toHaveValue('2025-01-01')
    expect(listQuery(calls)).toEqual({
      carId: golf.id,
      ...dateRangeToInstants('2025-01-01'),
    })
  })

  it('says when the range holds no tickets, in both languages', async () => {
    backend({
      ...allCars,
      'GET /api/v1/tickets': () =>
        response(200, { items: [], nextCursor: null }),
    })
    const user = userEvent.setup()
    renderApp('/tickets')

    expect(
      await screen.findByText('No tickets in this date range.'),
    ).toBeInTheDocument()
    await user.selectOptions(
      screen.getAllByRole('combobox', { name: 'Language' })[0],
      'de',
    )
    expect(
      await screen.findByText('Keine Strafzettel in diesem Zeitraum.'),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('combobox', { name: 'Nach Auto filtern' }),
    ).toBeInTheDocument()
    expect(screen.getByLabelText('Von')).toBeInTheDocument()
  })
})

describe('ticket creation', () => {
  async function fillRequired(user: ReturnType<typeof userEvent.setup>) {
    fireEvent.change(await screen.findByLabelText('Date and time'), {
      target: { value: '2026-09-27T12:30' },
    })
    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Type' }),
      'velocity',
    )
    await user.type(
      screen.getByRole('combobox', { name: 'Location' }),
      'A1 Linz',
    )
    await user.type(screen.getByRole('textbox', { name: 'Amount' }), '70,50')
  }

  it('creates a ticket and returns to the list', async () => {
    const calls = backend({
      ...allCars,
      ...noLocations,
      'POST /api/v1/tickets': () => response(201, speeding),
      'GET /api/v1/tickets': () =>
        response(200, { items: [speeding], nextCursor: null }),
    })
    const user = userEvent.setup()
    const { router } = renderApp('/tickets/create')

    const car = await screen.findByRole('combobox', { name: 'Car' })
    expect(car).toHaveValue('')
    await user.selectOptions(car, zoe.id)
    await fillRequired(user)
    await user.type(
      screen.getByRole('textbox', { name: /Description/ }),
      ' Radar ',
    )
    await user.click(screen.getByRole('button', { name: 'Add ticket' }))

    expect(await screen.findByText('A1 Linz')).toBeInTheDocument()
    await waitFor(() => expect(router.state.location.pathname).toBe('/tickets'))
    expect(bodyOf(calls, 'POST /api/v1/tickets')).toEqual({
      carId: zoe.id,
      date: new Date('2026-09-27T12:30').toISOString(),
      type: 'velocity',
      location: 'A1 Linz',
      amount: '70.50',
      description: 'Radar',
    })
  })

  it("preselects the car named in the page's carId", async () => {
    const calls = backend({
      ...allCars,
      ...noLocations,
      'POST /api/v1/tickets': () => response(201, parking),
      'GET /api/v1/tickets': () =>
        response(200, { items: [parking], nextCursor: null }),
    })
    const user = userEvent.setup()
    renderApp(`/tickets/create?carId=${golf.id}`)

    expect(await screen.findByRole('combobox', { name: 'Car' })).toHaveValue(
      golf.id,
    )
    await fillRequired(user)
    await user.click(screen.getByRole('button', { name: 'Add ticket' }))

    await screen.findByText('Wien Mariahilf')
    expect(bodyOf(calls, 'POST /api/v1/tickets')).toMatchObject({
      carId: golf.id,
      description: '',
    })
  })

  it("offers the caller's earlier locations as suggestions", async () => {
    backend({
      ...allCars,
      'GET /api/v1/tickets/locations': () =>
        response(200, ['A1 Linz', 'Wien Mariahilf']),
    })
    renderApp(`/tickets/create?carId=${golf.id}`)

    const location = await screen.findByRole('combobox', { name: 'Location' })
    const list = document.getElementById(location.getAttribute('list') ?? '')
    await waitFor(() =>
      expect(
        Array.from(list?.querySelectorAll('option') ?? []).map(
          (option) => option.value,
        ),
      ).toEqual(['A1 Linz', 'Wien Mariahilf']),
    )
  })

  it('shows a backend rejection on the affected field and keeps the input', async () => {
    backend({
      ...allCars,
      ...noLocations,
      'POST /api/v1/tickets': () =>
        response(400, {
          code: 'validation_failed',
          message: 'request validation failed',
          fields: { amount: 'negative' },
        }),
    })
    const user = userEvent.setup()
    const { router } = renderApp(`/tickets/create?carId=${golf.id}`)

    await fillRequired(user)
    await user.click(screen.getByRole('button', { name: 'Add ticket' }))

    const amount = screen.getByRole('textbox', { name: 'Amount' })
    expect(
      await screen.findByText('Enter a value of zero or more.'),
    ).toHaveAttribute('id', 'ticket-amount-error')
    expect(amount).toHaveAttribute('aria-invalid', 'true')
    expect(amount).toHaveAttribute(
      'aria-describedby',
      'ticket-amount-hint ticket-amount-error',
    )
    expect(screen.getByRole('combobox', { name: 'Location' })).toHaveValue(
      'A1 Linz',
    )
    expect(router.state.location.pathname).toBe('/tickets/create')
  })

  it('asks for a car first when the caller has none', async () => {
    backend({
      ...noLocations,
      'GET /api/v1/cars?limit=100': () =>
        response(200, { items: [], nextCursor: null }),
    })
    renderApp('/tickets/create')

    expect(
      await screen.findByRole('heading', { name: 'No car yet' }),
    ).toBeInTheDocument()
    expect(
      screen.getByText('A ticket belongs to a car. Add a car first.'),
    ).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Add car' })).toHaveAttribute(
      'href',
      '/cars/create',
    )
    expect(
      screen.queryByRole('button', { name: 'Add ticket' }),
    ).not.toBeInTheDocument()
  })
})

describe('ticket editing', () => {
  it('saves a change and shows it on the list', async () => {
    const saved = { ...parking, location: 'Wien Neubau', description: '' }
    const calls = backend({
      ...allCars,
      ...noLocations,
      [`GET ${ticketPath}`]: () => response(200, parking),
      [`PATCH ${ticketPath}`]: () => response(200, saved),
      'GET /api/v1/tickets': () =>
        response(200, { items: [saved], nextCursor: null }),
    })
    const user = userEvent.setup()
    const { router } = renderApp(`/tickets/${parking.id}/edit`)

    const location = await screen.findByRole('combobox', { name: 'Location' })
    expect(location).toHaveValue('Wien Mariahilf')
    const car = screen.getByRole('combobox', { name: 'Car' })
    expect(car).toHaveValue(golf.id)
    expect(car).toBeDisabled()
    expect(screen.getByRole('combobox', { name: 'Type' })).toHaveValue(
      'parking',
    )
    const description = screen.getByRole('textbox', { name: /Description/ })
    expect(description).toHaveValue('Short-term parking zone')
    await user.clear(location)
    await user.type(location, 'Wien Neubau')
    await user.clear(description)
    await user.click(screen.getByRole('button', { name: 'Save changes' }))

    expect(await screen.findByText('Wien Neubau')).toBeInTheDocument()
    await waitFor(() => expect(router.state.location.pathname).toBe('/tickets'))
    const body = bodyOf(calls, `PATCH ${ticketPath}`)
    expect(body).toEqual({
      date: new Date(parking.date).toISOString(),
      type: 'parking',
      location: 'Wien Neubau',
      amount: '36.00',
      description: '',
    })
    expect(body).not.toHaveProperty('carId')
  })

  it('keeps the ticket when a delete is not confirmed', async () => {
    const calls = backend({
      ...allCars,
      ...noLocations,
      [`GET ${ticketPath}`]: () => response(200, parking),
    })
    const user = userEvent.setup()
    renderApp(`/tickets/${parking.id}/edit`)

    await user.click(
      await screen.findByRole('button', { name: 'Delete ticket' }),
    )
    expect(
      screen.getByRole('group', { name: /Delete this ticket/ }),
    ).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Keep ticket' }))

    expect(
      screen.getByRole('button', { name: 'Delete ticket' }),
    ).toBeInTheDocument()
    expect(calls.some((call) => call.key.startsWith('DELETE'))).toBe(false)
  })

  it('deletes the ticket after confirmation and returns to the list', async () => {
    const calls = backend({
      ...allCars,
      ...noLocations,
      [`GET ${ticketPath}`]: () => response(200, parking),
      [`DELETE ${ticketPath}`]: () => response(204),
      'GET /api/v1/tickets': () =>
        response(200, { items: [], nextCursor: null }),
    })
    const user = userEvent.setup()
    const { router } = renderApp(`/tickets/${parking.id}/edit`)

    await user.click(
      await screen.findByRole('button', { name: 'Delete ticket' }),
    )
    await user.click(screen.getByRole('button', { name: 'Yes, delete ticket' }))

    expect(
      await screen.findByText('No tickets in this date range.'),
    ).toBeInTheDocument()
    await waitFor(() => expect(router.state.location.pathname).toBe('/tickets'))
    expect(
      calls.filter((call) => call.key === `DELETE ${ticketPath}`),
    ).toHaveLength(1)
  })

  it('shows a not-found state for a missing or unowned ticket', async () => {
    backend({
      ...allCars,
      [`GET ${ticketPath}`]: () =>
        response(404, { code: 'not_found', message: 'resource not found' }),
    })
    renderApp(`/tickets/${parking.id}/edit`)

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Ticket not found')
    expect(
      within(alert).getByRole('link', { name: 'Back to tickets' }),
    ).toHaveAttribute('href', '/tickets')
    expect(
      screen.queryByRole('button', { name: 'Delete ticket' }),
    ).not.toBeInTheDocument()
  })
})

describe('tickets of inactive and deregistered cars', () => {
  const inactive: Car = { ...zoe, isActive: false }
  const sold: Car = {
    ...golf,
    id: 'c7d2e0a4-1b3f-4e5a-8c6d-7e8f9a0b1c2d',
    make: 'Ford',
    name: 'Ka',
    deregistrationDate: '2020-01-01',
  }
  const soldEntry = {
    ...parking,
    id: '6f9e4d3a-0c5b-4e8f-9a2b-3c4d5e6f7a81',
    carId: sold.id,
    location: 'Ford entry',
  }
  const cars = {
    'GET /api/v1/cars?limit=100': () =>
      response(200, { items: [golf, inactive, sold], nextCursor: null }),
  }
  const carOptions = (name: string) =>
    within(screen.getByRole('combobox', { name }))
      .getAllByRole('option')
      .map((option) => option.textContent)

  it('lists every entry, muting and badging those of such cars', async () => {
    backend({
      ...cars,
      'GET /api/v1/tickets': () =>
        response(200, {
          items: [parking, speeding, soldEntry],
          nextCursor: null,
        }),
    })
    renderApp('/tickets')

    const table = await screen.findByRole('table', { name: 'Your tickets' })
    const [, activeRow, inactiveRow, soldRow] =
      within(table).getAllByRole('row')
    expect(activeRow).not.toHaveClass('row-muted')
    expect(inactiveRow).toHaveClass('row-muted')
    expect(inactiveRow).toHaveTextContent('Inactive')
    expect(soldRow).toHaveClass('row-muted')
    expect(soldRow).toHaveTextContent('Ford entry')
    expect(soldRow).toHaveTextContent('Deregistered')
    await waitFor(() =>
      expect(carOptions('Filter by car')).toEqual([
        'All cars',
        'VW Golf',
        'Renault Zoe',
      ]),
    )
  })

  it('keeps a deregistered car selected through the URL', async () => {
    backend({
      ...cars,
      [`GET /api/v1/tickets?carId=${sold.id}`]: () =>
        response(200, { items: [soldEntry], nextCursor: null }),
    })
    renderApp(`/tickets?carId=${sold.id}`)

    await screen.findByText('Ford entry')
    await waitFor(() =>
      expect(screen.getByLabelText('Filter by car')).toHaveValue(sold.id),
    )
  })

  it('offers only cars that are not deregistered on creation', async () => {
    backend({ ...cars, ...noLocations })
    renderApp('/tickets/create')

    await screen.findByRole('combobox', { name: 'Car' })
    expect(carOptions('Car')).not.toContain('Ford Ka')
    expect(carOptions('Car')).toContain('VW Golf')
    expect(carOptions('Car')).toContain('Renault Zoe')
  })

  it('does not preselect a deregistered car named in carId', async () => {
    backend({ ...cars, ...noLocations })
    renderApp(`/tickets/create?carId=${sold.id}`)

    expect(await screen.findByRole('combobox', { name: 'Car' })).toHaveValue('')
  })
})
