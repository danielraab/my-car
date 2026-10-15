import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Car, Refuel } from '#/api/client'
import { i18n } from '#/i18n'
import { dateRangeToInstants, defaultFrom } from '#/lib/date-range'
import { renderApp } from '#/test/render-app'

const profile = {
  id: '11111111-1111-1111-1111-111111111111',
  email: 'driver@example.com',
  firstName: 'Ada',
  lastName: 'Lovelace',
}
const car: Car = {
  id: '22222222-2222-2222-2222-222222222222',
  attachmentCount: 0,
  type: 'Car',
  make: 'VW',
  name: 'Golf',
  fuel: 'diesel',
  firstRegistration: '2020-01-01',
  licensePlate: 'W-1',
  fin: null,
  isActive: true,
  purchaseDate: null,
  purchasePrice: null,
  deregistrationDate: null,
  information: null,
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
}
const refuel: Refuel = {
  id: '33333333-3333-3333-3333-333333333333',
  attachmentCount: 0,
  carId: car.id,
  date: '2026-01-02T10:00:00Z',
  station: 'Fuel',
  odometerReading: 1000,
  fuel: 'normal',
  liters: '40',
  amount: '60',
  perLiter: '1.5',
  distance: null,
  consumption: null,
}
const receipt = {
  id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
  filename: 'receipt.jpg',
  contentType: 'image/jpeg',
  size: 2048,
  createdAt: '2026-01-02T10:05:00Z',
}
const response = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
// The query of the most recent request to exactly `path`.
function query(calls: string[], path: string) {
  const call = calls
    .filter((candidate) => candidate.split('?')[0] === `GET ${path}`)
    .at(-1)
  return call === undefined
    ? undefined
    : Object.fromEntries(new URLSearchParams(call.split('?')[1]))
}
function backend(refuels: Refuel[] = [refuel], cars: Car[] = [car]) {
  const calls: string[] = []
  vi.stubGlobal(
    'fetch',
    vi.fn(async (path: string, init?: RequestInit) => {
      const call = `${init?.method ?? 'GET'} ${path}`
      calls.push(call)
      if (path === '/api/v1/session') return response({ profile })
      if (path === '/api/v1/cars?limit=100')
        return response({ items: cars, nextCursor: null })
      if (path === '/api/v1/refuels/stations') return response([])
      if (path.startsWith('/api/v1/refuels/chart'))
        return response({ items: refuels })
      if (
        call === 'POST /api/v1/refuels' ||
        call.startsWith('PATCH /api/v1/refuels/')
      )
        return response(refuel, call.startsWith('POST') ? 201 : 200)
      if (call.startsWith('DELETE /api/v1/refuels/'))
        return new Response(null, { status: 204 })
      if (path === `/api/v1/refuels/${refuel.id}`) return response(refuel)
      if (path === '/api/v1/attachments/limits')
        return response({
          maxFileBytes: 10485760,
          maxFilesPerEntry: 10,
          allowedContentTypes: ['image/jpeg', 'application/pdf'],
        })
      if (path === `/api/v1/refuels/${refuel.id}/attachments`) {
        if (init?.method === 'POST') {
          const file = (init.body as FormData).get('file') as File
          return file.name.startsWith('broken')
            ? response({ code: 'internal_error', message: 'boom' }, 500)
            : response({ ...receipt, filename: file.name }, 201)
        }
        return response({ items: [receipt], nextCursor: null })
      }
      if (path.startsWith('/api/v1/refuels'))
        return response({ items: refuels, nextCursor: null })
      throw new Error(path)
    }),
  )
  return calls
}
beforeEach(async () => {
  await i18n.changeLanguage('en')
  vi.unstubAllGlobals()
})
describe('refuels', () => {
  it('charts only once the cars are loaded and retries failed cars', async () => {
    const calls = backend()
    const delegate = vi.mocked(fetch)
    let cars: (value: Response) => void = () => {}
    vi.stubGlobal(
      'fetch',
      vi.fn((path: string, init?: RequestInit) => {
        if (path !== '/api/v1/cars?limit=100') return delegate(path, init)
        calls.push(`GET ${path}`)
        return new Promise<Response>((resolve) => {
          cars = resolve
        })
      }),
    )
    const user = userEvent.setup()
    renderApp('/refuels')
    const chart = await screen.findByRole('region', {
      name: 'Fuel price history',
    })
    await waitFor(() =>
      expect(query(calls, '/api/v1/refuels/chart')).not.toBeUndefined(),
    )
    expect(within(chart).queryByRole('slider')).not.toBeInTheDocument()
    expect(within(chart).getByRole('status')).toBeInTheDocument()

    cars(response({ code: 'internal_error', message: 'boom' }, 500))
    expect(
      await within(chart).findByText(
        'The fuel price history could not be loaded.',
      ),
    ).toBeInTheDocument()

    await user.click(within(chart).getByRole('button', { name: 'Try again' }))
    cars(response({ items: [car], nextCursor: null }))
    expect(
      await within(chart).findByRole('slider', { name: 'Fuel price history' }),
    ).toBeInTheDocument()
    expect(within(chart).getByText('Diesel · Normal')).toBeInTheDocument()
  })

  it('shows the complete chart, table and total and filters both queries', async () => {
    const calls = backend()
    const user = userEvent.setup()
    const rendered = renderApp('/refuels')
    const table = await screen.findByRole('table', { name: 'Your refuels' })
    expect(
      within(table).queryByRole('columnheader', { name: 'Fuel' }),
    ).not.toBeInTheDocument()
    expect(
      screen.getByRole('slider', { name: 'Fuel price history' }),
    ).toBeInTheDocument()
    const { router } = rendered
    await user.selectOptions(screen.getByLabelText('Filter by car'), car.id)
    await waitFor(() =>
      expect(router.state.location.search).toEqual({ carId: car.id }),
    )
    const expected = { carId: car.id, ...dateRangeToInstants(defaultFrom()) }
    for (const path of ['/api/v1/refuels', '/api/v1/refuels/chart']) {
      await waitFor(() => expect(query(calls, path)).toEqual(expected))
    }
  })
  it('requests the last six months by default', async () => {
    const calls = backend()
    renderApp('/refuels')
    await screen.findByRole('table', { name: 'Your refuels' })
    expect(screen.getByLabelText('From')).toHaveValue(defaultFrom())
    expect(screen.getByLabelText('To')).toHaveValue('')
    const expected = dateRangeToInstants(defaultFrom())
    for (const path of ['/api/v1/refuels', '/api/v1/refuels/chart']) {
      await waitFor(() => expect(query(calls, path)).toEqual(expected))
    }
  })
  it('writes an edited range to the URL and requests it', async () => {
    const calls = backend()
    const { router } = renderApp('/refuels')
    await screen.findByRole('table', { name: 'Your refuels' })
    // A date picker reports the whole date at once.
    fireEvent.change(screen.getByLabelText('From'), {
      target: { value: '2026-01-01' },
    })
    fireEvent.change(screen.getByLabelText('To'), {
      target: { value: '2026-01-31' },
    })
    await waitFor(() =>
      expect(router.state.location.search).toEqual({
        from: '2026-01-01',
        to: '2026-01-31',
      }),
    )
    const expected = dateRangeToInstants('2026-01-01', '2026-01-31')
    for (const path of ['/api/v1/refuels', '/api/v1/refuels/chart']) {
      await waitFor(() => expect(query(calls, path)).toEqual(expected))
    }
  })
  it('restores the car and range from the URL', async () => {
    const calls = backend()
    renderApp(`/refuels?carId=${car.id}&from=2025-01-01`)
    await screen.findByRole('table', { name: 'Your refuels' })
    await waitFor(() =>
      expect(screen.getByLabelText('Filter by car')).toHaveValue(car.id),
    )
    expect(screen.getByLabelText('From')).toHaveValue('2025-01-01')
    const expected = { carId: car.id, ...dateRangeToInstants('2025-01-01') }
    for (const path of ['/api/v1/refuels', '/api/v1/refuels/chart']) {
      expect(query(calls, path)).toEqual(expected)
    }
  })
  it('falls back to the default for a malformed range', async () => {
    const calls = backend()
    renderApp('/refuels?from=2026-02-30&to=soon')
    await screen.findByRole('table', { name: 'Your refuels' })
    expect(screen.getByLabelText('From')).toHaveValue(defaultFrom())
    expect(query(calls, '/api/v1/refuels')).toEqual(
      dateRangeToInstants(defaultFrom()),
    )
  })
  it('says when the range holds no refuels, in both languages', async () => {
    backend([])
    const user = userEvent.setup()
    renderApp('/refuels')
    expect(
      await screen.findByText('No refuels in this date range.'),
    ).toBeInTheDocument()
    await user.selectOptions(screen.getByLabelText('Language'), 'de')
    expect(
      await screen.findByText('Keine Tankvorgänge in diesem Zeitraum.'),
    ).toBeInTheDocument()
  })
  it('preselects a car on creation', async () => {
    backend()
    renderApp(`/refuels/create?carId=${car.id}`)
    expect(await screen.findByLabelText('Car')).toHaveValue(car.id)
  })
  it('creates a refuel and returns to the list', async () => {
    const calls = backend()
    const user = userEvent.setup()
    renderApp('/refuels/create')
    await user.selectOptions(await screen.findByLabelText('Car'), car.id)
    await user.type(screen.getByLabelText('Station'), 'Fuel')
    await user.type(screen.getByLabelText('Litres'), '40')
    await user.type(screen.getByLabelText('Amount'), '60')
    await user.click(screen.getByRole('button', { name: 'Add refuel' }))
    expect(
      await screen.findByRole('heading', { name: 'Refuels' }),
    ).toBeInTheDocument()
    expect(calls).toContain('POST /api/v1/refuels')
  })
  it('deletes a refuel only after confirmation', async () => {
    const calls = backend()
    const user = userEvent.setup()
    renderApp(`/refuels/${refuel.id}/edit`)
    await user.click(
      await screen.findByRole('button', { name: 'Delete refuel' }),
    )
    await user.click(screen.getByRole('button', { name: 'Yes, delete refuel' }))
    expect(
      await screen.findByRole('heading', { name: 'Refuels' }),
    ).toBeInTheDocument()
    expect(calls).toContain(`DELETE /api/v1/refuels/${refuel.id}`)
  })
  it('switches refuel labels to German', async () => {
    backend()
    const user = userEvent.setup()
    renderApp('/refuels')
    await screen.findByRole('heading', { name: 'Refuels' })
    await user.selectOptions(screen.getByLabelText('Language'), 'de')
    expect(
      await screen.findByRole('heading', { name: 'Tankvorgänge' }),
    ).toBeInTheDocument()
  })
})

describe('refuels of inactive and deregistered cars', () => {
  const inactive: Car = {
    ...car,
    id: '3f1e8a52-7c2b-4d0e-9a61-5b4c3d2e1f00',
    make: 'Renault',
    name: 'Zoe',
    isActive: false,
  }
  const sold: Car = {
    ...car,
    id: 'c7d2e0a4-1b3f-4e5a-8c6d-7e8f9a0b1c2d',
    make: 'Ford',
    name: 'Ka',
    deregistrationDate: '2020-01-01',
  }
  const cars = [car, inactive, sold]
  const entries: Refuel[] = [
    refuel,
    {
      ...refuel,
      id: '44444444-4444-4444-4444-444444444444',
      carId: inactive.id,
    },
    {
      ...refuel,
      id: '55555555-5555-5555-5555-555555555555',
      carId: sold.id,
      station: 'Ford station',
    },
  ]
  const carOptions = (label: string) =>
    within(screen.getByLabelText(label))
      .getAllByRole('option')
      .map((option) => option.textContent)

  it('lists every refuel, muting and badging those of such cars', async () => {
    backend(entries, cars)
    renderApp('/refuels')

    const table = await screen.findByRole('table', { name: 'Your refuels' })
    const rows = within(table).getAllByRole('row').slice(1, 4)
    expect(rows.map((row) => row.classList.contains('row-muted'))).toEqual([
      false,
      true,
      true,
    ])
    expect(rows[1]).toHaveTextContent('Inactive')
    expect(rows[2]).toHaveTextContent('Ford station')
    expect(rows[2]).toHaveTextContent('Deregistered')
    await waitFor(() =>
      expect(carOptions('Filter by car')).toEqual([
        'All cars',
        'VW Golf',
        'Renault Zoe',
      ]),
    )
  })

  it('keeps a deregistered car selected through the URL', async () => {
    backend(entries, cars)
    renderApp(`/refuels?carId=${sold.id}`)

    await screen.findByRole('table', { name: 'Your refuels' })
    await waitFor(() =>
      expect(screen.getByLabelText('Filter by car')).toHaveValue(sold.id),
    )
  })

  it('offers only cars that are not deregistered on creation', async () => {
    backend(entries, cars)
    renderApp('/refuels/create')

    await screen.findByLabelText('Car')
    expect(carOptions('Car')).not.toContain('Ford Ka')
    expect(carOptions('Car')).toContain('VW Golf')
    expect(carOptions('Car')).toContain('Renault Zoe')
  })

  it('does not preselect a deregistered car named in carId', async () => {
    backend(entries, cars)
    renderApp(`/refuels/create?carId=${sold.id}`)

    expect(await screen.findByLabelText('Car')).toHaveValue('')
  })
})

describe('refuel attachments', () => {
  async function fillRefuel(user: ReturnType<typeof userEvent.setup>) {
    await user.selectOptions(await screen.findByLabelText('Car'), car.id)
    await user.type(screen.getByLabelText('Station'), 'Fuel')
    await user.type(screen.getByLabelText('Litres'), '40')
    await user.type(screen.getByLabelText('Amount'), '60')
  }
  const pdf = (name: string) =>
    new File(['%PDF'], name, { type: 'application/pdf' })

  it('uploads queued files after creating the refuel', async () => {
    const calls = backend()
    const user = userEvent.setup()
    renderApp('/refuels/create')
    await fillRefuel(user)
    await user.upload(await screen.findByLabelText('Add files'), [
      pdf('receipt.pdf'),
    ])
    expect(await screen.findByText('receipt.pdf')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Add refuel' }))

    expect(
      await screen.findByRole('heading', { name: 'Refuels' }),
    ).toBeInTheDocument()
    const posts = calls.filter((call) => call.startsWith('POST'))
    expect(posts).toEqual([
      'POST /api/v1/refuels',
      `POST /api/v1/refuels/${refuel.id}/attachments`,
    ])
  })

  it('opens the edit screen naming files that failed to upload', async () => {
    backend()
    const user = userEvent.setup()
    renderApp('/refuels/create')
    await fillRefuel(user)
    await user.upload(await screen.findByLabelText('Add files'), [
      pdf('broken.pdf'),
    ])
    await user.click(screen.getByRole('button', { name: 'Add refuel' }))

    expect(
      await screen.findByText(
        'The entry was saved, but these files could not be uploaded: broken.pdf. You can add them again here.',
      ),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('heading', { name: 'Edit refuel' }),
    ).toBeInTheDocument()
    expect(
      within(
        await screen.findByRole('list', { name: 'Attached files' }),
      ).getByRole('button', { name: 'receipt.jpg' }),
    ).toBeInTheDocument()
  })

  it('shows a clickable paperclip on rows with attachments', async () => {
    backend([{ ...refuel, attachmentCount: 1 }])
    const user = userEvent.setup()
    renderApp('/refuels')
    const table = await screen.findByRole('table', { name: 'Your refuels' })
    await user.click(
      within(table).getByRole('button', { name: '1 attachment' }),
    )
    await user.click(await screen.findByRole('button', { name: 'receipt.jpg' }))

    const viewer = await screen.findByRole('dialog', { name: 'receipt.jpg' })
    expect(within(viewer).getByAltText('receipt.jpg')).toHaveAttribute(
      'src',
      `/api/v1/attachments/${receipt.id}/content`,
    )
    expect(within(viewer).getByText('1 / 1')).toBeInTheDocument()
    expect(
      screen.queryByRole('list', { name: 'Attached files' }),
    ).not.toBeInTheDocument()
  })

  it('shows no paperclip on rows without attachments', async () => {
    backend()
    renderApp('/refuels')
    const table = await screen.findByRole('table', { name: 'Your refuels' })
    expect(
      within(table).queryByRole('button', { name: /attachment/ }),
    ).not.toBeInTheDocument()
  })

  it('explains the downscaling in German', async () => {
    backend()
    await i18n.changeLanguage('de')
    renderApp('/refuels/create')
    expect(
      await screen.findByText(/werden vor dem Hochladen verkleinert/),
    ).toBeInTheDocument()
    expect(screen.getByText('Anhänge')).toBeInTheDocument()
  })
})
