import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { Car, Repair } from '#/api/client'
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

const service: Repair = {
  id: '4d7c2b1e-8a3f-4c6d-9e0f-1a2b3c4d5e6f',
  attachmentCount: 0,
  carId: golf.id,
  date: '2026-03-14T08:15:00Z',
  station: 'Garage Huber',
  odometerReading: 123456,
  type: 'service',
  amount: '120.50',
  description: 'Oil change',
}

const brakes: Repair = {
  ...service,
  id: '5e8d3c2f-9b4a-4d7e-8f1a-2b3c4d5e6f70',
  carId: zoe.id,
  date: '2026-05-02T14:00:00Z',
  station: 'Autohaus Maier',
  odometerReading: null,
  type: 'wearing_part',
  amount: '79.50',
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

// The query of the most recent repairs list request.
function listQuery(calls: ReturnType<typeof backend>) {
  const call = calls
    .filter((candidate) => candidate.url.split('?')[0] === '/api/v1/repairs')
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
const noStations = { 'GET /api/v1/repairs/stations': () => response(200, []) }
const repairPath = `/api/v1/repairs/${service.id}`

beforeEach(async () => {
  await i18n.changeLanguage('en')
  document.documentElement.lang = 'en'
  vi.unstubAllGlobals()
})

describe('repairs list', () => {
  it('shows the documented columns, a running total and links', async () => {
    backend({
      ...allCars,
      'GET /api/v1/repairs': () =>
        response(200, { items: [service, brakes], nextCursor: null }),
    })
    renderApp('/repairs')

    const table = await screen.findByRole('table', { name: 'Your repairs' })
    const headers = within(table)
      .getAllByRole('columnheader')
      .map((header) => header.textContent)
    expect(headers).toEqual([
      'Date and time',
      'Car',
      'Garage',
      'Odometer',
      'Type',
      'Amount',
      'Attachments',
    ])
    const rows = within(table).getAllByRole('row')
    expect(rows).toHaveLength(4)
    expect(
      within(rows[1]).getByRole('link', { name: shown(service.date) }),
    ).toHaveAttribute('href', `/repairs/${service.id}/edit`)
    expect(
      await within(rows[1]).findByRole('link', { name: 'VW Golf' }),
    ).toHaveAttribute('href', `/cars/${golf.id}`)
    expect(rows[1]).toHaveTextContent('Garage Huber')
    expect(rows[1]).toHaveTextContent('123,456 km')
    expect(rows[1]).toHaveTextContent('Service')
    expect(rows[1]).toHaveTextContent('120.50')
    expect(rows[2]).toHaveTextContent('Renault Zoe')
    expect(rows[2]).toHaveTextContent('Not recorded')
    expect(rows[2]).toHaveTextContent('Wearing part')
    expect(rows[3]).toHaveTextContent('Total of the repairs shown')
    expect(rows[3]).toHaveTextContent('200.00')
    expect(screen.getByRole('link', { name: 'Add repair' })).toHaveAttribute(
      'href',
      '/repairs/create',
    )
    expect(
      screen.queryByRole('button', { name: 'Load more repairs' }),
    ).not.toBeInTheDocument()
  })

  it('appends a further page and includes it in the total', async () => {
    const calls = backend({
      ...allCars,
      'GET /api/v1/repairs': () =>
        response(200, { items: [service], nextCursor: 'c1' }),
      'GET /api/v1/repairs?cursor=c1': () =>
        response(200, { items: [brakes], nextCursor: null }),
    })
    const user = userEvent.setup()
    renderApp('/repairs')

    const total = () => screen.getAllByRole('row').at(-1)
    await screen.findByText('Garage Huber')
    expect(total()).toHaveTextContent('120.50')
    await user.click(screen.getByRole('button', { name: 'Load more repairs' }))

    expect(await screen.findByText('Autohaus Maier')).toBeInTheDocument()
    expect(screen.getByText('Garage Huber')).toBeInTheDocument()
    expect(total()).toHaveTextContent('200.00')
    expect(
      screen.queryByRole('button', { name: 'Load more repairs' }),
    ).not.toBeInTheDocument()
    expect(calls.map((call) => call.key)).toContain(
      'GET /api/v1/repairs?cursor=c1',
    )
  })

  it('shows a retryable error when the repairs cannot be loaded', async () => {
    backend({
      ...allCars,
      'GET /api/v1/repairs': [
        () => response(503, { code: 'unavailable', message: 'down' }),
        () => response(200, { items: [service], nextCursor: null }),
      ],
    })
    const user = userEvent.setup()
    renderApp('/repairs')

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Repairs unavailable')
    expect(screen.queryByRole('table')).not.toBeInTheDocument()

    await user.click(within(alert).getByRole('button', { name: 'Try again' }))
    expect(await screen.findByText('Garage Huber')).toBeInTheDocument()
  })

  it('follows a switch to German', async () => {
    backend({
      ...allCars,
      'GET /api/v1/repairs': () =>
        response(200, { items: [service], nextCursor: null }),
    })
    const user = userEvent.setup()
    renderApp('/repairs')

    await screen.findByText('Garage Huber')
    await user.selectOptions(
      screen.getAllByRole('combobox', { name: 'Language' })[0],
      'de',
    )

    expect(
      await screen.findByRole('heading', { name: 'Reparaturen' }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('columnheader', { name: 'Werkstatt' }),
    ).toBeInTheDocument()
    const row = screen.getAllByRole('row')[1]
    expect(row).toHaveTextContent(shown(service.date, 'de'))
    expect(row).toHaveTextContent('123.456 km')
    expect(row).toHaveTextContent('Service')
    expect(row).toHaveTextContent('120,50')
    expect(
      screen.getByText('Summe der angezeigten Reparaturen'),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('link', { name: 'Reparatur hinzufügen' }),
    ).toBeInTheDocument()
  })
  it('requests the last six months across all cars by default', async () => {
    const calls = backend({
      ...allCars,
      'GET /api/v1/repairs': () =>
        response(200, { items: [service], nextCursor: null }),
    })
    renderApp('/repairs')

    await screen.findByText('Garage Huber')
    expect(screen.getByLabelText('Filter by car')).toHaveValue('')
    expect(screen.getByLabelText('From')).toHaveValue(defaultFrom())
    expect(screen.getByLabelText('To')).toHaveValue('')
    expect(listQuery(calls)).toEqual(dateRangeToInstants(defaultFrom()))
  })

  it('filters by car and range and keeps both in the URL', async () => {
    const calls = backend({
      ...allCars,
      'GET /api/v1/repairs': () =>
        response(200, { items: [service, brakes], nextCursor: null }),
      [`GET /api/v1/repairs?carId=${golf.id}`]: () =>
        response(200, { items: [service], nextCursor: null }),
    })
    const user = userEvent.setup()
    const { router } = renderApp('/repairs')

    await screen.findByText('Autohaus Maier')
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
      expect(screen.queryByText('Autohaus Maier')).not.toBeInTheDocument(),
    )
    expect(screen.getByRole('link', { name: 'Add repair' })).toHaveAttribute(
      'href',
      `/repairs/create?carId=${golf.id}`,
    )
  })

  it('restores the car and range from the URL', async () => {
    const calls = backend({
      ...allCars,
      [`GET /api/v1/repairs?carId=${golf.id}`]: () =>
        response(200, { items: [service], nextCursor: null }),
    })
    renderApp(`/repairs?carId=${golf.id}&from=2025-01-01`)

    await screen.findByText('Garage Huber')
    await waitFor(() =>
      expect(screen.getByLabelText('Filter by car')).toHaveValue(golf.id),
    )
    expect(screen.getByLabelText('From')).toHaveValue('2025-01-01')
    expect(listQuery(calls)).toEqual({
      carId: golf.id,
      ...dateRangeToInstants('2025-01-01'),
    })
  })

  it('says when the range holds no repairs, in both languages', async () => {
    backend({
      ...allCars,
      'GET /api/v1/repairs': () =>
        response(200, { items: [], nextCursor: null }),
    })
    const user = userEvent.setup()
    renderApp('/repairs')

    expect(
      await screen.findByText('No repairs in this date range.'),
    ).toBeInTheDocument()
    await user.selectOptions(
      screen.getAllByRole('combobox', { name: 'Language' })[0],
      'de',
    )
    expect(
      await screen.findByText('Keine Reparaturen in diesem Zeitraum.'),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('combobox', { name: 'Nach Auto filtern' }),
    ).toBeInTheDocument()
    expect(screen.getByLabelText('Von')).toBeInTheDocument()
  })
})

describe('repair creation', () => {
  async function fillRequired(user: ReturnType<typeof userEvent.setup>) {
    fireEvent.change(await screen.findByLabelText('Date and time'), {
      target: { value: '2026-09-27T12:30' },
    })
    await user.type(
      screen.getByRole('combobox', { name: 'Garage' }),
      'Garage Huber',
    )
    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Type' }),
      'service',
    )
    await user.type(screen.getByRole('textbox', { name: 'Amount' }), '120,50')
  }

  it('creates a repair and returns to the list', async () => {
    const calls = backend({
      ...allCars,
      'GET /api/v1/repairs/stations': () => response(200, ['Garage Huber']),
      'POST /api/v1/repairs': () => response(201, service),
      'GET /api/v1/repairs': () =>
        response(200, { items: [service], nextCursor: null }),
    })
    const user = userEvent.setup()
    const { router } = renderApp('/repairs/create')

    const car = await screen.findByRole('combobox', { name: 'Car' })
    expect(car).toHaveValue('')
    await user.selectOptions(car, golf.id)
    await fillRequired(user)
    await user.type(
      screen.getByRole('textbox', { name: /Odometer reading/ }),
      '123456',
    )
    await user.type(
      screen.getByRole('textbox', { name: /Description/ }),
      ' Oil change ',
    )
    await user.click(screen.getByRole('button', { name: 'Add repair' }))

    expect(await screen.findByText('Garage Huber')).toBeInTheDocument()
    await waitFor(() => expect(router.state.location.pathname).toBe('/repairs'))
    expect(bodyOf(calls, 'POST /api/v1/repairs')).toEqual({
      carId: golf.id,
      date: new Date('2026-09-27T12:30').toISOString(),
      station: 'Garage Huber',
      odometerReading: 123456,
      type: 'service',
      amount: '120.50',
      description: 'Oil change',
    })
  })

  it("preselects the car named in the page's carId", async () => {
    const calls = backend({
      ...allCars,
      ...noStations,
      'POST /api/v1/repairs': () => response(201, brakes),
      'GET /api/v1/repairs': () =>
        response(200, { items: [brakes], nextCursor: null }),
    })
    const user = userEvent.setup()
    renderApp(`/repairs/create?carId=${zoe.id}`)

    expect(await screen.findByRole('combobox', { name: 'Car' })).toHaveValue(
      zoe.id,
    )
    await fillRequired(user)
    await user.click(screen.getByRole('button', { name: 'Add repair' }))

    await screen.findByText('Autohaus Maier')
    expect(bodyOf(calls, 'POST /api/v1/repairs')).toMatchObject({
      carId: zoe.id,
      odometerReading: null,
      description: '',
    })
  })

  it('shows a backend rejection on the affected field and keeps the input', async () => {
    backend({
      ...allCars,
      ...noStations,
      'POST /api/v1/repairs': () =>
        response(400, {
          code: 'validation_failed',
          message: 'request validation failed',
          fields: { amount: 'negative' },
        }),
    })
    const user = userEvent.setup()
    const { router } = renderApp(`/repairs/create?carId=${golf.id}`)

    await fillRequired(user)
    await user.click(screen.getByRole('button', { name: 'Add repair' }))

    const amount = screen.getByRole('textbox', { name: 'Amount' })
    expect(
      await screen.findByText('Enter a value of zero or more.'),
    ).toHaveAttribute('id', 'repair-amount-error')
    expect(amount).toHaveAttribute('aria-invalid', 'true')
    expect(amount).toHaveAttribute(
      'aria-describedby',
      'repair-amount-hint repair-amount-error',
    )
    expect(screen.getByRole('combobox', { name: 'Garage' })).toHaveValue(
      'Garage Huber',
    )
    expect(router.state.location.pathname).toBe('/repairs/create')
  })

  it('rejects a non-numeric odometer reading without sending it', async () => {
    const calls = backend({ ...allCars, ...noStations })
    const user = userEvent.setup()
    renderApp(`/repairs/create?carId=${golf.id}`)

    await fillRequired(user)
    const odometer = screen.getByRole('textbox', { name: /Odometer reading/ })
    await user.type(odometer, '12.5k')
    await user.click(screen.getByRole('button', { name: 'Add repair' }))

    expect(odometer).toHaveAttribute('aria-invalid', 'true')
    expect(
      screen.getByText('Check the format of this value.'),
    ).toBeInTheDocument()
    expect(calls.some((call) => call.key.startsWith('POST'))).toBe(false)
  })

  it('asks for a car first when the caller has none', async () => {
    backend({
      ...noStations,
      'GET /api/v1/cars?limit=100': () =>
        response(200, { items: [], nextCursor: null }),
    })
    renderApp('/repairs/create')

    expect(
      await screen.findByRole('heading', { name: 'No car yet' }),
    ).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Add car' })).toHaveAttribute(
      'href',
      '/cars/create',
    )
    expect(
      screen.queryByRole('button', { name: 'Add repair' }),
    ).not.toBeInTheDocument()
  })
})

describe('repair editing', () => {
  it('saves a change and shows it on the list', async () => {
    const saved = {
      ...service,
      station: 'Garage Berger',
      odometerReading: null,
    }
    const calls = backend({
      ...allCars,
      ...noStations,
      [`GET ${repairPath}`]: () => response(200, service),
      [`PATCH ${repairPath}`]: () => response(200, saved),
      'GET /api/v1/repairs': () =>
        response(200, { items: [saved], nextCursor: null }),
    })
    const user = userEvent.setup()
    const { router } = renderApp(`/repairs/${service.id}/edit`)

    const station = await screen.findByRole('combobox', { name: 'Garage' })
    expect(station).toHaveValue('Garage Huber')
    const car = screen.getByRole('combobox', { name: 'Car' })
    expect(car).toHaveValue(golf.id)
    expect(car).toBeDisabled()
    expect(screen.getByRole('textbox', { name: /Description/ })).toHaveValue(
      'Oil change',
    )
    await user.clear(station)
    await user.type(station, 'Garage Berger')
    await user.clear(screen.getByRole('textbox', { name: /Odometer reading/ }))
    await user.click(screen.getByRole('button', { name: 'Save changes' }))

    expect(await screen.findByText('Garage Berger')).toBeInTheDocument()
    await waitFor(() => expect(router.state.location.pathname).toBe('/repairs'))
    const body = bodyOf(calls, `PATCH ${repairPath}`)
    expect(body).toEqual({
      date: new Date(service.date).toISOString(),
      station: 'Garage Berger',
      odometerReading: null,
      type: 'service',
      amount: '120.50',
      description: 'Oil change',
    })
    expect(body).not.toHaveProperty('carId')
  })

  it('keeps the repair when a delete is not confirmed', async () => {
    const calls = backend({
      ...allCars,
      ...noStations,
      [`GET ${repairPath}`]: () => response(200, service),
    })
    const user = userEvent.setup()
    renderApp(`/repairs/${service.id}/edit`)

    await user.click(
      await screen.findByRole('button', { name: 'Delete repair' }),
    )
    expect(
      screen.getByRole('group', { name: /Delete this repair/ }),
    ).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Keep repair' }))

    expect(
      screen.getByRole('button', { name: 'Delete repair' }),
    ).toBeInTheDocument()
    expect(calls.some((call) => call.key.startsWith('DELETE'))).toBe(false)
  })

  it('deletes the repair after confirmation and returns to the list', async () => {
    const calls = backend({
      ...allCars,
      ...noStations,
      [`GET ${repairPath}`]: () => response(200, service),
      [`DELETE ${repairPath}`]: () => response(204),
      'GET /api/v1/repairs': () =>
        response(200, { items: [], nextCursor: null }),
    })
    const user = userEvent.setup()
    const { router } = renderApp(`/repairs/${service.id}/edit`)

    await user.click(
      await screen.findByRole('button', { name: 'Delete repair' }),
    )
    await user.click(screen.getByRole('button', { name: 'Yes, delete repair' }))

    expect(
      await screen.findByText('No repairs in this date range.'),
    ).toBeInTheDocument()
    await waitFor(() => expect(router.state.location.pathname).toBe('/repairs'))
    expect(
      calls.filter((call) => call.key === `DELETE ${repairPath}`),
    ).toHaveLength(1)
  })

  it('shows a not-found state for a missing or unowned repair', async () => {
    backend({
      ...allCars,
      [`GET ${repairPath}`]: () =>
        response(404, { code: 'not_found', message: 'resource not found' }),
    })
    renderApp(`/repairs/${service.id}/edit`)

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Repair not found')
    expect(
      within(alert).getByRole('link', { name: 'Back to repairs' }),
    ).toHaveAttribute('href', '/repairs')
    expect(
      screen.queryByRole('button', { name: 'Delete repair' }),
    ).not.toBeInTheDocument()
  })
})

describe('repairs of inactive and deregistered cars', () => {
  const inactive: Car = { ...zoe, isActive: false }
  const sold: Car = {
    ...golf,
    id: 'c7d2e0a4-1b3f-4e5a-8c6d-7e8f9a0b1c2d',
    make: 'Ford',
    name: 'Ka',
    deregistrationDate: '2020-01-01',
  }
  const soldEntry = {
    ...service,
    id: '6f9e4d3a-0c5b-4e8f-9a2b-3c4d5e6f7a81',
    carId: sold.id,
    station: 'Ford entry',
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
      'GET /api/v1/repairs': () =>
        response(200, {
          items: [service, brakes, soldEntry],
          nextCursor: null,
        }),
    })
    renderApp('/repairs')

    const table = await screen.findByRole('table', { name: 'Your repairs' })
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
      [`GET /api/v1/repairs?carId=${sold.id}`]: () =>
        response(200, { items: [soldEntry], nextCursor: null }),
    })
    renderApp(`/repairs?carId=${sold.id}`)

    await screen.findByText('Ford entry')
    await waitFor(() =>
      expect(screen.getByLabelText('Filter by car')).toHaveValue(sold.id),
    )
  })

  it('offers only cars that are not deregistered on creation', async () => {
    backend({ ...cars, ...noStations })
    renderApp('/repairs/create')

    await screen.findByRole('combobox', { name: 'Car' })
    expect(carOptions('Car')).not.toContain('Ford Ka')
    expect(carOptions('Car')).toContain('VW Golf')
    expect(carOptions('Car')).toContain('Renault Zoe')
  })

  it('does not preselect a deregistered car named in carId', async () => {
    backend({ ...cars, ...noStations })
    renderApp(`/repairs/create?carId=${sold.id}`)

    expect(await screen.findByRole('combobox', { name: 'Car' })).toHaveValue('')
  })
})
