import { describe, expect, it, vi } from 'vitest'

import { ApiError, type Session, UnauthorizedError } from '#/api/client'

vi.mock('#/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('#/api/client')>()
  return { ...actual, getSession: vi.fn() }
})

import { getSession } from '#/api/client'

import { resolveSession } from './session'

const getSessionMock = vi.mocked(getSession)

describe('resolveSession', () => {
  it('returns the authenticated session', async () => {
    const session: Session = {
      profile: {
        id: 'id',
        email: 'driver@example.com',
        firstName: 'Ada',
        lastName: 'Lovelace',
        currency: 'EUR',
        volumeUnit: 'liter',
        distanceUnit: 'kilometer',
      },
    }
    getSessionMock.mockResolvedValueOnce(session)
    await expect(resolveSession()).resolves.toEqual(session)
  })

  it('maps only unauthorized responses to anonymous', async () => {
    getSessionMock.mockRejectedValueOnce(new UnauthorizedError())
    await expect(resolveSession()).resolves.toBeNull()
  })

  it('preserves retryable failures', async () => {
    const error = new ApiError(503, 'unavailable', 'Unavailable')
    getSessionMock.mockRejectedValueOnce(error)
    await expect(resolveSession()).rejects.toBe(error)
  })
})
