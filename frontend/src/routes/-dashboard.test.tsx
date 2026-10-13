import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { Car, ExpenseRow } from '#/api/client'
import { i18n } from '#/i18n'
import { toCalendarDate, yearToInstants } from '#/lib/date-range'
import { renderApp } from '#/test/render-app'

const profile = {
  id: '617c3d87-21b4-4cb9-96f3-e03510892296',
  email: 'driver@example.com',
  firstName: 'Ada',
  lastName: 'Lovelace',
}

const golf: Car = {
  id: '9b0a5f0e-5d8f-4a55-9d59-0b8f1f3c2a10',
  type: 'Car',
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

const honda: Car = {
  ...golf,
  id: '3f1e8a52-7c2b-4d0e-9a61-5b4c3d2e1f00',
  type: 'Bike',
  make: 'Honda',
  name: 'CB500',
}

// Deregistered since yesterday; still registered until far in the future.
const yesterday = new Date()
yesterday.setDate(yesterday.getDate() - 1)
const sold: Car = {
  ...golf,
  id: 'c7d2e0a4-1b3f-4e5a-8c6d-7e8f9a0b1c2d',
  make: 'Ford',
  name: 'Ka',
  deregistrationDate: toCalendarDate(yesterday),
}
const leaving: Car = { ...honda, deregistrationDate: '2999-12-31' }

const currentYear = new Date().getFullYear()

// Mid-month dates, so the month is the same in every time zone.
function rows(year: number): ExpenseRow[] {
  return [
    { date: `${year}-03-10T12:00:00Z`, kind: 'refuel', amount: '0.1' },
    { date: `${year}-03-15T12:00:00Z`, kind: 'refuel', amount: '0.2' },
    { date: `${year}-03-20T12:00:00Z`, kind: 'ticket', amount: '36.00' },
    { date: `${year}-07-12T12:00:00Z`, kind: 'repair', amount: '120.50' },
  ]
}

function response(status: number, body?: unknown) {
  return new Response(body === undefined ? null : JSON.stringify(body), {
    status,
    headers:
      body === undefined ? undefined : { 'Content-Type': 'application/json' },
  })
}

type Handler = (query: URLSearchParams) => Response

// Routes fetch calls by "METHOD path" without the query; the session is
// always valid. Every statistics request is recorded with its query.
function backend(statistics: Handler | Handler[], cars: Car[] = [golf, honda]) {
  const requests: Record<string, string>[] = []
  const fetchMock = vi.fn(async (path: string, init?: RequestInit) => {
    const [pathname, search] = path.split('?')
    const query = new URLSearchParams(search)
    const key = `${init?.method ?? 'GET'} ${pathname}`
    if (key === 'GET /api/v1/session') return response(200, { profile })
    if (key === 'GET /api/v1/cars') {
      return response(200, { items: cars, nextCursor: null })
    }
    if (key === 'GET /api/v1/stats/expenses') {
      requests.push(Object.fromEntries(query))
      const handler = Array.isArray(statistics)
        ? statistics.shift()
        : statistics
      if (!handler) throw new Error('unexpected statistics request')
      return handler(query)
    }
    throw new Error(`unexpected request ${key}`)
  })
  vi.stubGlobal('fetch', fetchMock)
  return requests
}

// Answers with the rows of the requested local year.
const byYear: Handler = (query) => {
  const year = new Date(query.get('to') ?? '').getFullYear() - 1
  return response(200, { items: rows(year) })
}

function tableRows() {
  return within(screen.getByRole('table')).getAllByRole('row')
}

beforeEach(async () => {
  await i18n.changeLanguage('en')
  document.documentElement.lang = 'en'
  vi.unstubAllGlobals()
})

describe('dashboard', () => {
  it('charts the current year per month across all cars by default', async () => {
    const requests = backend(byYear)
    const { router } = renderApp('/home')

    expect(
      await screen.findByRole('slider', { name: `Expenses in ${currentYear}` }),
    ).toBeInTheDocument()
    expect(requests.at(-1)).toEqual(yearToInstants(currentYear))
    expect(router.state.location.search).toEqual({})
    expect(screen.getByLabelText('Filter by car')).toHaveValue('')

    const all = tableRows()
    expect(all).toHaveLength(13)
    expect(all[0]).toHaveTextContent('MonthRefuelsRepairsTicketsTotal')
    const month = (index: number) =>
      new Intl.DateTimeFormat('en', { month: 'short' }).format(
        new Date(currentYear, index, 1),
      )
    expect(all[1]).toHaveTextContent(`${month(0)}0.000.000.000.00`)
    expect(all[3]).toHaveTextContent(`${month(2)}0.300.0036.0036.30`)
    expect(all[7]).toHaveTextContent(`${month(6)}0.00120.500.00120.50`)
    const legend = document.querySelector('.chart-legend')
    expect(legend).toHaveTextContent('Refuels')
    expect(legend).toHaveTextContent('Repairs')
    expect(legend).toHaveTextContent('Tickets')
  })

  it('shows a card per car and shortcuts to record expenses', async () => {
    backend(byYear)
    renderApp('/home')

    const cars = await screen.findByRole('region', { name: 'Your cars' })
    expect(within(cars).getByRole('link', { name: 'VW Golf' })).toHaveAttribute(
      'href',
      `/cars/${golf.id}`,
    )
    expect(
      within(cars).getByRole('link', { name: 'Honda CB500' }),
    ).toHaveAttribute('href', `/cars/${honda.id}`)
    const shortcuts = screen.getByRole('list', { name: 'Record an expense' })
    expect(
      within(shortcuts).getByRole('link', { name: 'Add refuel' }),
    ).toHaveAttribute('href', '/refuels/create')
    expect(
      within(shortcuts).getByRole('link', { name: 'Add repair' }),
    ).toHaveAttribute('href', '/repairs/create')
    expect(
      within(shortcuts).getByRole('link', { name: 'Add ticket' }),
    ).toHaveAttribute('href', '/tickets/create')
  })

  it('leaves deregistered cars out of the cards and the car filter', async () => {
    const inactive = { ...golf, isActive: false }
    const requests = backend(byYear, [inactive, leaving, sold])
    renderApp('/home')

    const cars = await screen.findByRole('region', { name: 'Your cars' })
    expect(
      within(cars)
        .getAllByRole('link')
        .map((link) => link.textContent),
    ).toEqual(['VW Golf', 'Honda CB500'])
    const options = () =>
      within(screen.getByRole('combobox', { name: 'Filter by car' }))
        .getAllByRole('option')
        .map((option) => option.textContent)
    await waitFor(() =>
      expect(options()).toEqual(['All cars', 'VW Golf', 'Honda CB500']),
    )
    // "All cars" statistics are not narrowed to the cars shown.
    expect(requests.at(-1)).toEqual(yearToInstants(currentYear))
  })

  it('shows no cards when every car is deregistered', async () => {
    backend(byYear, [sold])
    renderApp('/home')

    await screen.findByRole('slider', { name: `Expenses in ${currentYear}` })
    expect(
      screen.queryByRole('region', { name: 'Your cars' }),
    ).not.toBeInTheDocument()
  })

  it('keeps a deregistered car selected through the URL', async () => {
    const requests = backend(byYear, [golf, sold])
    renderApp(`/home?carId=${sold.id}`)

    await screen.findByRole('slider', { name: `Expenses in ${currentYear}` })
    await waitFor(() =>
      expect(screen.getByLabelText('Filter by car')).toHaveValue(sold.id),
    )
    expect(requests.at(-1)).toEqual({
      carId: sold.id,
      ...yearToInstants(currentYear),
    })
    const cars = screen.getByRole('region', { name: 'Your cars' })
    expect(
      within(cars).queryByRole('link', { name: 'Ford Ka' }),
    ).not.toBeInTheDocument()
  })

  it('moves between years and past the current one', async () => {
    const requests = backend(byYear)
    const user = userEvent.setup()
    const { router } = renderApp('/home')

    await screen.findByRole('slider', { name: `Expenses in ${currentYear}` })
    await user.click(
      screen.getByRole('button', {
        name: `Previous year (${currentYear - 1})`,
      }),
    )
    await waitFor(() =>
      expect(router.state.location.search).toEqual({ year: currentYear - 1 }),
    )
    expect(
      await screen.findByRole('slider', {
        name: `Expenses in ${currentYear - 1}`,
      }),
    ).toBeInTheDocument()
    expect(requests.at(-1)).toEqual(yearToInstants(currentYear - 1))

    await user.click(
      screen.getByRole('button', { name: `Next year (${currentYear})` }),
    )
    const next = screen.getByRole('button', {
      name: `Next year (${currentYear + 1})`,
    })
    await waitFor(() => expect(next).toBeEnabled())
    await user.click(next)
    await waitFor(() =>
      expect(router.state.location.search).toEqual({ year: currentYear + 1 }),
    )
    await waitFor(() =>
      expect(requests.at(-1)).toEqual(yearToInstants(currentYear + 1)),
    )
  })

  it('filters by car and passes it to the shortcuts', async () => {
    const requests = backend(byYear)
    const user = userEvent.setup()
    const { router } = renderApp('/home')

    await screen.findByRole('slider', { name: `Expenses in ${currentYear}` })
    await user.selectOptions(
      await screen.findByRole('combobox', { name: 'Filter by car' }),
      honda.id,
    )

    await waitFor(() =>
      expect(router.state.location.search).toEqual({ carId: honda.id }),
    )
    await waitFor(() =>
      expect(requests.at(-1)).toEqual({
        carId: honda.id,
        ...yearToInstants(currentYear),
      }),
    )
    expect(screen.getByRole('link', { name: 'Add repair' })).toHaveAttribute(
      'href',
      `/repairs/create?carId=${honda.id}`,
    )
  })

  it('restores the year and car from the URL', async () => {
    const requests = backend(byYear)
    renderApp(`/home?year=2024&carId=${golf.id}`)

    expect(
      await screen.findByRole('slider', { name: 'Expenses in 2024' }),
    ).toBeInTheDocument()
    await waitFor(() =>
      expect(screen.getByLabelText('Filter by car')).toHaveValue(golf.id),
    )
    expect(requests.at(-1)).toEqual({ carId: golf.id, ...yearToInstants(2024) })
  })

  it('shows the current year for a malformed year', async () => {
    const requests = backend(byYear)
    renderApp('/home?year=abc')

    expect(
      await screen.findByRole('slider', { name: `Expenses in ${currentYear}` }),
    ).toBeInTheDocument()
    expect(requests.at(-1)).toEqual(yearToInstants(currentYear))
  })

  it('shows the empty state for a year without expenses', async () => {
    backend(() => response(200, { items: [] }))
    renderApp('/home?year=2019')

    expect(await screen.findByText('No expenses in 2019.')).toBeInTheDocument()
    expect(screen.queryByRole('slider')).not.toBeInTheDocument()
  })

  it('shows a retryable error when the statistics cannot be loaded', async () => {
    backend([
      () => response(503, { code: 'unavailable', message: 'down' }),
      byYear,
    ])
    const user = userEvent.setup()
    renderApp('/home')

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('We could not load your expenses.')
    await user.click(within(alert).getByRole('button', { name: 'Try again' }))
    expect(
      await screen.findByRole('slider', { name: `Expenses in ${currentYear}` }),
    ).toBeInTheDocument()
  })

  it('follows a switch to German', async () => {
    backend(byYear)
    const user = userEvent.setup()
    renderApp('/home?year=2025')

    await screen.findByRole('slider', { name: 'Expenses in 2025' })
    await user.selectOptions(
      screen.getAllByRole('combobox', { name: 'Language' })[0],
      'de',
    )

    expect(
      await screen.findByRole('heading', { name: 'Übersicht', level: 1 }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('slider', { name: 'Ausgaben 2025' }),
    ).toBeInTheDocument()
    expect(screen.getByLabelText('Nach Auto filtern')).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Vorheriges Jahr (2024)' }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('link', { name: 'Reparatur hinzufügen' }),
    ).toBeInTheDocument()
    const all = tableRows()
    expect(all[0]).toHaveTextContent(
      'MonatTankvorgängeReparaturenStrafzettelSumme',
    )
    // Short month names depend on the ICU data, so they are computed.
    const month = (index: number) =>
      new Intl.DateTimeFormat('de', { month: 'short' }).format(
        new Date(2025, index, 1),
      )
    expect(all[3]).toHaveTextContent(`${month(2)}0,300,0036,0036,30`)
    expect(all[10]).toHaveTextContent(month(9))
  })
})
