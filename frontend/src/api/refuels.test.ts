import { afterEach, describe, expect, it, vi } from 'vitest'
import { getRefuelChart, getRefuels } from './client'

const ok = (body: unknown) =>
  new Response(JSON.stringify(body), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  })
const item = {
  id: '33333333-3333-3333-3333-333333333333',
  attachmentCount: 0,
  carId: '22222222-2222-2222-2222-222222222222',
  date: '2026-09-28T10:00:00Z',
  station: 'Fuel',
  odometerReading: null,
  fuel: 'normal',
  liters: '40',
  amount: '60',
  perLiter: '1.5',
  distance: null,
  consumption: null,
}
afterEach(() => vi.unstubAllGlobals())
describe('refuel client guards', () => {
  it('accepts a documented page', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(ok({ items: [item], nextCursor: null })),
    )
    await expect(getRefuels()).resolves.toEqual({
      items: [item],
      nextCursor: null,
    })
  })
  it('rejects malformed derived values', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValue(
          ok({ items: [{ ...item, perLiter: 1.5 }], nextCursor: null }),
        ),
    )
    await expect(getRefuels()).rejects.toMatchObject({
      status: 502,
      code: 'invalid_response',
    })
  })
})
describe('refuel client filters', () => {
  it('sends only the supplied car and range parameters', async () => {
    const fetchMock = vi
      .fn()
      .mockImplementation(async () => ok({ items: [], nextCursor: null }))
    vi.stubGlobal('fetch', fetchMock)
    await getRefuels({ carId: item.carId, from: '2026-01-01T00:00:00.000Z' })
    expect(fetchMock.mock.calls[0][0]).toBe(
      `/api/v1/refuels?carId=${item.carId}&from=2026-01-01T00%3A00%3A00.000Z`,
    )
    await getRefuelChart()
    expect(fetchMock.mock.calls[1][0]).toBe('/api/v1/refuels/chart')
    await getRefuelChart({
      carId: item.carId,
      from: '2026-01-01T00:00:00.000Z',
      to: '2026-02-01T00:00:00.000Z',
    })
    expect(fetchMock.mock.calls[2][0]).toBe(
      `/api/v1/refuels/chart?carId=${item.carId}&from=2026-01-01T00%3A00%3A00.000Z&to=2026-02-01T00%3A00%3A00.000Z`,
    )
  })
})
