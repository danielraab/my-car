import { afterEach, describe, expect, it, vi } from 'vitest'

import {
  ApiError,
  createCar,
  createRepair,
  createTicket,
  deleteCar,
  deleteRepair,
  deleteTicket,
  getAuthenticationMethods,
  getCar,
  getCars,
  getExpenseStatistics,
  getMe,
  getRepair,
  getRepairStations,
  getRepairs,
  getSession,
  getTicket,
  getTicketLocations,
  getTickets,
  logout,
  requestMagicLink,
  UnauthorizedError,
  updateCar,
  updateMe,
  updateRepair,
  updateTicket,
} from './client'

const session = {
  profile: {
    id: '617c3d87-21b4-4cb9-96f3-e03510892296',
    email: 'driver@example.com',
    firstName: 'Ada',
    lastName: 'Lovelace',
  },
}

afterEach(() => vi.unstubAllGlobals())

describe('API client', () => {
  it('returns enabled authentication methods', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValue(
          new Response(
            JSON.stringify({ methods: ['magic_link', 'passkey', 'oidc'] }),
            { status: 200 },
          ),
        ),
    )

    await expect(getAuthenticationMethods()).resolves.toEqual({
      methods: ['magic_link', 'passkey', 'oidc'],
    })
  })

  it.each([
    {},
    { methods: [] },
    { methods: ['oidc'] },
    { methods: ['magic_link', 'password'] },
    { methods: ['magic_link', 'magic_link'] },
  ])('rejects malformed authentication methods %#', async (body) => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValue(new Response(JSON.stringify(body), { status: 200 })),
    )

    await expect(getAuthenticationMethods()).rejects.toMatchObject({
      status: 502,
      code: 'invalid_response',
    })
  })

  it('preserves authentication-method API failures', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValue(
          new Response(
            JSON.stringify({ code: 'unavailable', message: 'Try later' }),
            { status: 503 },
          ),
        ),
    )

    await expect(getAuthenticationMethods()).rejects.toMatchObject({
      status: 503,
      code: 'unavailable',
    })
  })

  it('returns a valid current session using same-origin credentials', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(session), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    await expect(getSession()).resolves.toEqual(session)
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/session',
      expect.objectContaining({ credentials: 'same-origin' }),
    )
  })

  it('distinguishes an anonymous session', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response(null, { status: 401 })),
    )
    await expect(getSession()).rejects.toBeInstanceOf(UnauthorizedError)
  })

  it('rejects malformed session JSON', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ profile: { email: 4 } }), {
          status: 200,
        }),
      ),
    )
    await expect(getSession()).rejects.toMatchObject({
      status: 502,
      code: 'invalid_response',
    })
  })

  it('preserves documented API errors and field details', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            code: 'validation_failed',
            message: 'Invalid email',
            fields: { email: 'invalid' },
          }),
          { status: 400 },
        ),
      ),
    )
    await expect(
      requestMagicLink({ email: 'not-an-email' }),
    ).rejects.toMatchObject({
      status: 400,
      code: 'validation_failed',
      fields: { email: 'invalid' },
    })
  })

  it('accepts magic-link and logout empty success responses', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(new Response(null, { status: 202 }))
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(
      requestMagicLink({ email: 'driver@example.com', returnTo: '/cars' }),
    ).resolves.toBeUndefined()
    await expect(logout()).resolves.toBeUndefined()
  })

  it('distinguishes logout with an already-invalid session', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response(null, { status: 401 })),
    )
    await expect(logout()).rejects.toBeInstanceOf(UnauthorizedError)
  })

  it('normalizes transport failures', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('offline')))
    await expect(getSession()).rejects.toEqual(
      expect.objectContaining({ status: 0, code: 'network_error' }),
    )
    await expect(getSession()).rejects.toBeInstanceOf(ApiError)
    await expect(getAuthenticationMethods()).rejects.toMatchObject({
      status: 0,
      code: 'network_error',
    })
  })

  it('reads the current profile', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        new Response(JSON.stringify(session.profile), { status: 200 }),
      )
    vi.stubGlobal('fetch', fetchMock)

    await expect(getMe()).resolves.toEqual(session.profile)
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/me',
      expect.objectContaining({ credentials: 'same-origin' }),
    )
  })

  it('patches only the supplied name fields', async () => {
    const updated = { ...session.profile, firstName: 'Grace' }
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response(JSON.stringify(updated), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(updateMe({ firstName: 'Grace' })).resolves.toEqual(updated)
    const [path, init] = fetchMock.mock.calls[0]
    expect(path).toBe('/api/v1/me')
    expect(init).toMatchObject({ method: 'PATCH', credentials: 'same-origin' })
    expect(JSON.parse(init.body)).toEqual({ firstName: 'Grace' })
  })

  it('exposes profile validation field reasons', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            code: 'validation_failed',
            message: 'request validation failed',
            fields: { lastName: 'too_long' },
          }),
          { status: 400 },
        ),
      ),
    )

    await expect(updateMe({ lastName: 'x' })).rejects.toMatchObject({
      status: 400,
      code: 'validation_failed',
      fields: { lastName: 'too_long' },
    })
  })

  it('treats a profile 401 as unauthorized', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response(null, { status: 401 })),
    )

    await expect(getMe()).rejects.toBeInstanceOf(UnauthorizedError)
  })

  it.each([{}, { ...session.profile, email: null }, session])(
    'rejects malformed profiles %#',
    async (body) => {
      vi.stubGlobal(
        'fetch',
        vi
          .fn()
          .mockResolvedValue(
            new Response(JSON.stringify(body), { status: 200 }),
          ),
      )

      await expect(getMe()).rejects.toMatchObject({
        status: 502,
        code: 'invalid_response',
      })
    },
  )
})

describe('cars API client', () => {
  const car = {
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
    purchasePrice: '18500.50',
    deregistrationDate: null,
    information: null,
    createdAt: '2026-09-27T10:00:00Z',
    updatedAt: '2026-09-27T10:00:00Z',
  }

  function stub(status: number, body?: unknown) {
    const fetchMock = vi.fn(
      async (_path: string, _init?: RequestInit) =>
        new Response(body === undefined ? null : JSON.stringify(body), {
          status,
        }),
    )
    vi.stubGlobal('fetch', fetchMock)
    return fetchMock
  }

  it('reads a page of cars and passes the cursor', async () => {
    const page = { items: [car], nextCursor: 'next' }
    const fetchMock = stub(200, page)

    await expect(getCars()).resolves.toEqual(page)
    expect(fetchMock.mock.calls[0][0]).toBe('/api/v1/cars')

    await getCars({ cursor: 'a/b', limit: 10 })
    expect(fetchMock.mock.calls[1][0]).toBe(
      '/api/v1/cars?cursor=a%2Fb&limit=10',
    )
  })

  it('creates, reads, updates and deletes a car', async () => {
    let fetchMock = stub(201, car)
    await expect(
      createCar({
        type: 'Hatchback',
        make: 'VW',
        name: 'Golf',
        fuel: 'diesel',
        firstRegistration: '2019-03-01',
        licensePlate: 'W-123AB',
      }),
    ).resolves.toEqual(car)
    expect(fetchMock.mock.calls[0][1]).toMatchObject({ method: 'POST' })

    fetchMock = stub(200, car)
    await expect(getCar(car.id)).resolves.toEqual(car)
    expect(fetchMock.mock.calls[0][0]).toBe(`/api/v1/cars/${car.id}`)

    fetchMock = stub(200, { ...car, fin: 'ABC' })
    await expect(updateCar(car.id, { fin: 'ABC' })).resolves.toMatchObject({
      fin: 'ABC',
    })
    const [path, init] = fetchMock.mock.calls[0]
    expect(path).toBe(`/api/v1/cars/${car.id}`)
    expect(init).toMatchObject({ method: 'PATCH' })
    expect(JSON.parse(String(init?.body))).toEqual({ fin: 'ABC' })

    fetchMock = stub(204)
    await expect(deleteCar(car.id)).resolves.toBeUndefined()
    expect(fetchMock.mock.calls[0][1]).toMatchObject({ method: 'DELETE' })
  })

  it('accepts a car without registration details', async () => {
    const bare = { ...car, firstRegistration: null, licensePlate: null }
    stub(200, bare)
    await expect(getCar(car.id)).resolves.toEqual(bare)
  })

  it('exposes car validation field reasons', async () => {
    stub(400, {
      code: 'validation_failed',
      message: 'request validation failed',
      fields: { fuel: 'invalid_enum' },
    })

    await expect(updateCar(car.id, { name: 'x' })).rejects.toMatchObject({
      status: 400,
      fields: { fuel: 'invalid_enum' },
    })
  })

  it('treats a car 401 as unauthorized and keeps a 404', async () => {
    stub(401)
    await expect(getCars()).rejects.toBeInstanceOf(UnauthorizedError)

    stub(404, { code: 'not_found', message: 'resource not found' })
    await expect(getCar(car.id)).rejects.toMatchObject({
      status: 404,
      code: 'not_found',
    })
    stub(404, { code: 'not_found', message: 'resource not found' })
    await expect(deleteCar(car.id)).rejects.toMatchObject({ status: 404 })
  })

  it.each([
    { items: [{ ...car, fuel: 'petrol' }], nextCursor: null },
    { items: [{ ...car, fin: undefined }], nextCursor: null },
    { items: [{ ...car, licensePlate: undefined }], nextCursor: null },
    { items: [car] },
    [car],
  ])('rejects malformed car pages %#', async (body) => {
    stub(200, body)

    await expect(getCars()).rejects.toMatchObject({
      status: 502,
      code: 'invalid_response',
    })
  })
})

describe('repairs API client', () => {
  const repair = {
    id: '4d7c2b1e-8a3f-4c6d-9e0f-1a2b3c4d5e6f',
    attachmentCount: 0,
    carId: '9b0a5f0e-5d8f-4a55-9d59-0b8f1f3c2a10',
    date: '2026-09-27T10:30:00Z',
    station: 'Garage',
    odometerReading: null,
    type: 'service',
    amount: '120.50',
    description: '',
  }

  function stub(status: number, body?: unknown) {
    const fetchMock = vi.fn(
      async (_path: string, _init?: RequestInit) =>
        new Response(body === undefined ? null : JSON.stringify(body), {
          status,
        }),
    )
    vi.stubGlobal('fetch', fetchMock)
    return fetchMock
  }

  it('reads a page of repairs and passes the cursor and car filter', async () => {
    const page = { items: [repair], nextCursor: 'next' }
    const fetchMock = stub(200, page)

    await expect(getRepairs()).resolves.toEqual(page)
    expect(fetchMock.mock.calls[0][0]).toBe('/api/v1/repairs')

    await getRepairs({ cursor: 'a/b', limit: 10, carId: repair.carId })
    expect(fetchMock.mock.calls[1][0]).toBe(
      `/api/v1/repairs?cursor=a%2Fb&limit=10&carId=${repair.carId}`,
    )

    await getRepairs({
      carId: repair.carId,
      from: '2026-01-01T00:00:00.000Z',
      to: '2026-02-01T00:00:00.000Z',
    })
    expect(fetchMock.mock.calls[2][0]).toBe(
      `/api/v1/repairs?carId=${repair.carId}&from=2026-01-01T00%3A00%3A00.000Z&to=2026-02-01T00%3A00%3A00.000Z`,
    )
  })

  it('creates, reads, updates and deletes a repair', async () => {
    let fetchMock = stub(201, repair)
    await expect(
      createRepair({
        carId: repair.carId,
        date: repair.date,
        station: 'Garage',
        type: 'service',
        amount: '120.50',
      }),
    ).resolves.toEqual(repair)
    expect(fetchMock.mock.calls[0][1]?.method).toBe('POST')

    fetchMock = stub(200, repair)
    await expect(getRepair(repair.id)).resolves.toEqual(repair)
    expect(fetchMock.mock.calls[0][0]).toBe(`/api/v1/repairs/${repair.id}`)

    fetchMock = stub(200, { ...repair, odometerReading: 1200 })
    await expect(
      updateRepair(repair.id, { odometerReading: 1200 }),
    ).resolves.toMatchObject({ odometerReading: 1200 })
    const [path, init] = fetchMock.mock.calls[0]
    expect(path).toBe(`/api/v1/repairs/${repair.id}`)
    expect(init?.method).toBe('PATCH')
    expect(init?.body).toBe('{"odometerReading":1200}')

    fetchMock = stub(204)
    await expect(deleteRepair(repair.id)).resolves.toBeUndefined()
    expect(fetchMock.mock.calls[0][1]?.method).toBe('DELETE')
  })

  it('reads the station suggestions', async () => {
    const fetchMock = stub(200, ['Alpha', 'Zeta'])
    await expect(getRepairStations()).resolves.toEqual(['Alpha', 'Zeta'])
    expect(fetchMock.mock.calls[0][0]).toBe('/api/v1/repairs/stations')
  })

  it('exposes repair validation field reasons', async () => {
    stub(400, {
      code: 'validation_failed',
      message: 'request validation failed',
      fields: { amount: 'negative' },
    })
    await expect(
      updateRepair(repair.id, { amount: '-1' }),
    ).rejects.toMatchObject({
      status: 400,
      fields: { amount: 'negative' },
    })
  })

  it('treats a repair 401 as unauthorized and keeps a 404', async () => {
    stub(401, { code: 'unauthorized', message: 'authentication required' })
    await expect(getRepairs()).rejects.toBeInstanceOf(UnauthorizedError)

    stub(404, { code: 'not_found', message: 'resource not found' })
    await expect(getRepair(repair.id)).rejects.toMatchObject({
      status: 404,
      code: 'not_found',
    })
    await expect(deleteRepair(repair.id)).rejects.toMatchObject({
      status: 404,
    })
  })

  it.each([
    { items: [{ ...repair, type: 'tuning' }], nextCursor: null },
    { items: [{ ...repair, odometerReading: 12.5 }], nextCursor: null },
    { items: [{ ...repair, description: null }], nextCursor: null },
    { items: [repair] },
    [repair],
  ])('rejects malformed repair pages %#', async (body) => {
    stub(200, body)
    await expect(getRepairs()).rejects.toMatchObject({
      status: 502,
      code: 'invalid_response',
    })
  })

  it('rejects malformed station suggestions', async () => {
    stub(200, ['Alpha', 3])
    await expect(getRepairStations()).rejects.toMatchObject({
      code: 'invalid_response',
    })
  })
})

describe('tickets API client', () => {
  const ticket = {
    id: '5e8d3c2f-9b4a-4d7e-8f1a-2b3c4d5e6f70',
    attachmentCount: 0,
    carId: '9b0a5f0e-5d8f-4a55-9d59-0b8f1f3c2a10',
    date: '2026-09-27T10:30:00Z',
    type: 'parking',
    location: 'Vienna',
    amount: '36.00',
    description: '',
  }

  function stub(status: number, body?: unknown) {
    const fetchMock = vi.fn(
      async (_path: string, _init?: RequestInit) =>
        new Response(body === undefined ? null : JSON.stringify(body), {
          status,
        }),
    )
    vi.stubGlobal('fetch', fetchMock)
    return fetchMock
  }

  it('reads a page of tickets and passes the cursor and car filter', async () => {
    const page = { items: [ticket], nextCursor: 'next' }
    const fetchMock = stub(200, page)

    await expect(getTickets()).resolves.toEqual(page)
    expect(fetchMock.mock.calls[0][0]).toBe('/api/v1/tickets')

    await getTickets({ cursor: 'a/b', limit: 10, carId: ticket.carId })
    expect(fetchMock.mock.calls[1][0]).toBe(
      `/api/v1/tickets?cursor=a%2Fb&limit=10&carId=${ticket.carId}`,
    )

    await getTickets({
      carId: ticket.carId,
      from: '2026-01-01T00:00:00.000Z',
      to: '2026-02-01T00:00:00.000Z',
    })
    expect(fetchMock.mock.calls[2][0]).toBe(
      `/api/v1/tickets?carId=${ticket.carId}&from=2026-01-01T00%3A00%3A00.000Z&to=2026-02-01T00%3A00%3A00.000Z`,
    )
  })

  it('creates, reads, updates and deletes a ticket', async () => {
    let fetchMock = stub(201, ticket)
    await expect(
      createTicket({
        carId: ticket.carId,
        date: ticket.date,
        type: 'parking',
        location: 'Vienna',
        amount: '36.00',
      }),
    ).resolves.toEqual(ticket)
    expect(fetchMock.mock.calls[0][0]).toBe('/api/v1/tickets')
    expect(fetchMock.mock.calls[0][1]?.method).toBe('POST')

    fetchMock = stub(200, ticket)
    await expect(getTicket(ticket.id)).resolves.toEqual(ticket)
    expect(fetchMock.mock.calls[0][0]).toBe(`/api/v1/tickets/${ticket.id}`)

    fetchMock = stub(200, { ...ticket, description: 'Radar' })
    await expect(
      updateTicket(ticket.id, { description: 'Radar' }),
    ).resolves.toMatchObject({ description: 'Radar' })
    const [path, init] = fetchMock.mock.calls[0]
    expect(path).toBe(`/api/v1/tickets/${ticket.id}`)
    expect(init?.method).toBe('PATCH')
    expect(init?.body).toBe('{"description":"Radar"}')

    fetchMock = stub(204)
    await expect(deleteTicket(ticket.id)).resolves.toBeUndefined()
    expect(fetchMock.mock.calls[0][1]?.method).toBe('DELETE')
  })

  it('reads the location suggestions', async () => {
    const fetchMock = stub(200, ['Graz', 'Vienna'])
    await expect(getTicketLocations()).resolves.toEqual(['Graz', 'Vienna'])
    expect(fetchMock.mock.calls[0][0]).toBe('/api/v1/tickets/locations')
  })

  it('exposes ticket validation field reasons', async () => {
    stub(400, {
      code: 'validation_failed',
      message: 'request validation failed',
      fields: { location: 'empty' },
    })
    await expect(
      updateTicket(ticket.id, { location: ' ' }),
    ).rejects.toMatchObject({
      status: 400,
      fields: { location: 'empty' },
    })
  })

  it('treats a ticket 401 as unauthorized and keeps a 404', async () => {
    stub(401, { code: 'unauthorized', message: 'authentication required' })
    await expect(getTickets()).rejects.toBeInstanceOf(UnauthorizedError)

    stub(404, { code: 'not_found', message: 'resource not found' })
    await expect(getTicket(ticket.id)).rejects.toMatchObject({
      status: 404,
      code: 'not_found',
    })
    await expect(deleteTicket(ticket.id)).rejects.toMatchObject({
      status: 404,
    })
  })

  it.each([
    { items: [{ ...ticket, type: 'towing' }], nextCursor: null },
    { items: [{ ...ticket, location: 3 }], nextCursor: null },
    { items: [{ ...ticket, description: null }], nextCursor: null },
    { items: [ticket] },
    [ticket],
  ])('rejects malformed ticket pages %#', async (body) => {
    stub(200, body)
    await expect(getTickets()).rejects.toMatchObject({
      status: 502,
      code: 'invalid_response',
    })
  })
})

describe('expense statistics API client', () => {
  const carId = '9b0a5f0e-5d8f-4a55-9d59-0b8f1f3c2a10'
  const range = {
    from: '2025-12-31T23:00:00.000Z',
    to: '2026-12-31T23:00:00.000Z',
  }
  const statistics = {
    items: [
      { date: '2026-03-31T22:30:00Z', kind: 'refuel', amount: '61.234' },
      { date: '2026-05-01T08:00:00Z', kind: 'ticket', amount: '30' },
    ],
  }

  function stub(status: number, body?: unknown) {
    const fetchMock = vi.fn(
      async (_path: string, _init?: RequestInit) =>
        new Response(body === undefined ? null : JSON.stringify(body), {
          status,
        }),
    )
    vi.stubGlobal('fetch', fetchMock)
    return fetchMock
  }

  it('passes the range and the optional car filter', async () => {
    const fetchMock = stub(200, statistics)

    await expect(getExpenseStatistics(range)).resolves.toEqual(statistics)
    expect(fetchMock.mock.calls[0][0]).toBe(
      '/api/v1/stats/expenses?from=2025-12-31T23%3A00%3A00.000Z&to=2026-12-31T23%3A00%3A00.000Z',
    )

    await getExpenseStatistics({ ...range, carId })
    expect(fetchMock.mock.calls[1][0]).toBe(
      `/api/v1/stats/expenses?carId=${carId}&from=2025-12-31T23%3A00%3A00.000Z&to=2026-12-31T23%3A00%3A00.000Z`,
    )
  })

  it.each([
    { items: [{ ...statistics.items[0], kind: 'toll' }] },
    { items: [{ ...statistics.items[0], amount: 61.234 }] },
    { items: [{ kind: 'refuel', amount: '1' }] },
    statistics.items,
  ])('rejects malformed statistics %#', async (body) => {
    stub(200, body)
    await expect(getExpenseStatistics(range)).rejects.toMatchObject({
      status: 502,
      code: 'invalid_response',
    })
  })
})
