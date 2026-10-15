import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { Session } from '#/api/client'
import { i18n } from '#/i18n'
import { renderApp } from '#/test/render-app'

const profile = {
  id: '617c3d87-21b4-4cb9-96f3-e03510892296',
  email: 'driver@example.com',
  firstName: 'Ada',
  lastName: 'Lovelace',
  currency: 'EUR',
  volumeUnit: 'liter',
  distanceUnit: 'kilometer',
}

function response(status: number, body?: unknown) {
  return new Response(body === undefined ? null : JSON.stringify(body), {
    status,
    headers:
      body === undefined ? undefined : { 'Content-Type': 'application/json' },
  })
}

type Handler = (init?: RequestInit) => Response | Promise<Response>

// Routes fetch calls by "METHOD path"; the session is always valid and the
// passkey section (covered in -passkeys.test.tsx) finds no passkeys.
function backend(routes: Record<string, Handler | Handler[]>) {
  const calls: { key: string; init?: RequestInit }[] = []
  const fetchMock = vi.fn(async (path: string, init?: RequestInit) => {
    const key = `${init?.method ?? 'GET'} ${path}`
    calls.push({ key, init })
    if (key === 'GET /api/v1/session') return response(200, { profile })
    if (key === 'GET /api/v1/passkeys') return response(200, { items: [] })
    const route = routes[key]
    const handler = Array.isArray(route) ? route.shift() : route
    if (!handler) throw new Error(`unexpected request ${key}`)
    return handler(init)
  })
  vi.stubGlobal('fetch', fetchMock)
  return calls
}

beforeEach(async () => {
  await i18n.changeLanguage('en')
  document.documentElement.lang = 'en'
  vi.unstubAllGlobals()
})

describe('profile screen', () => {
  it('shows a read-only email and prefilled names without token details', async () => {
    backend({ 'GET /api/v1/me': () => response(200, profile) })
    renderApp('/profile')

    expect(
      await screen.findByRole('textbox', { name: 'First name' }),
    ).toHaveValue('Ada')
    expect(screen.getByRole('textbox', { name: 'Last name' })).toHaveValue(
      'Lovelace',
    )
    expect(
      screen.queryByRole('textbox', { name: /Email/ }),
    ).not.toBeInTheDocument()
    const main = screen
      .getByRole('heading', { name: 'Profile' })
      .closest('section') as HTMLElement
    expect(within(main).getByText('driver@example.com')).toBeInTheDocument()
    expect(within(main).getByText(/cannot be changed here/)).toBeInTheDocument()
    expect(main).not.toHaveTextContent(/token|expires|iat|exp\b/i)
  })

  it('shows a retryable error when the profile cannot be loaded', async () => {
    backend({
      'GET /api/v1/me': [
        () => response(503, { code: 'unavailable', message: 'down' }),
        () => response(200, profile),
      ],
    })
    const user = userEvent.setup()
    renderApp('/profile')

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Profile unavailable')
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument()

    await user.click(within(alert).getByRole('button', { name: 'Try again' }))
    expect(
      await screen.findByRole('textbox', { name: 'First name' }),
    ).toHaveValue('Ada')
  })

  it('saves names and settings, not the email, and updates the session profile', async () => {
    const saved = { ...profile, firstName: 'Grace' }
    const calls = backend({
      'GET /api/v1/me': () => response(200, profile),
      'PATCH /api/v1/me': () => response(200, saved),
    })
    const user = userEvent.setup()
    const { queryClient } = renderApp('/profile')

    const firstName = await screen.findByRole('textbox', { name: 'First name' })
    await user.clear(firstName)
    await user.type(firstName, 'Grace')
    await user.click(screen.getByRole('button', { name: 'Save changes' }))

    expect(await screen.findByRole('status')).toHaveTextContent(
      'Your profile has been saved.',
    )
    const patch = calls.find((call) => call.key === 'PATCH /api/v1/me')
    expect(JSON.parse(String(patch?.init?.body))).toEqual({
      firstName: 'Grace',
      lastName: 'Lovelace',
      currency: 'EUR',
      volumeUnit: 'liter',
      distanceUnit: 'kilometer',
    })
    expect(firstName).toHaveValue('Grace')
    expect(
      queryClient.getQueryData<Session>(['session'])?.profile.firstName,
    ).toBe('Grace')
  })

  it('shows a backend name rejection on the affected field', async () => {
    backend({
      'GET /api/v1/me': () => response(200, profile),
      'PATCH /api/v1/me': () =>
        response(400, {
          code: 'validation_failed',
          message: 'request validation failed',
          fields: { lastName: 'too_long' },
        }),
    })
    const user = userEvent.setup()
    renderApp('/profile')

    await user.click(
      await screen.findByRole('button', { name: 'Save changes' }),
    )

    const lastName = screen.getByRole('textbox', { name: 'Last name' })
    expect(
      await screen.findByText('Use at most 100 characters.'),
    ).toHaveAttribute('id', 'lastName-error')
    expect(lastName).toHaveAttribute('aria-invalid', 'true')
    expect(lastName).toHaveAttribute('aria-describedby', 'lastName-error')
    expect(screen.getByRole('textbox', { name: 'First name' })).toHaveAttribute(
      'aria-invalid',
      'false',
    )
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })

  it('keeps unsaved input when saving fails unexpectedly', async () => {
    backend({
      'GET /api/v1/me': () => response(200, profile),
      'PATCH /api/v1/me': () => {
        throw new TypeError('network down')
      },
    })
    const user = userEvent.setup()
    renderApp('/profile')

    const lastName = await screen.findByRole('textbox', { name: 'Last name' })
    await user.clear(lastName)
    await user.type(lastName, 'Hopper')
    await user.click(screen.getByRole('button', { name: 'Save changes' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Your changes could not be saved.',
    )
    expect(lastName).toHaveValue('Hopper')
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })

  it('prefills the currency and units with a no-conversion hint', async () => {
    backend({ 'GET /api/v1/me': () => response(200, profile) })
    renderApp('/profile')

    expect(
      await screen.findByRole('combobox', { name: 'Currency' }),
    ).toHaveValue('EUR')
    expect(
      screen.getByRole('option', { name: 'Euro (EUR)' }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('option', { name: 'US Dollar (USD)' }),
    ).toBeInTheDocument()
    expect(screen.getByRole('combobox', { name: 'Volume' })).toHaveValue(
      'liter',
    )
    expect(screen.getByRole('combobox', { name: 'Distance' })).toHaveValue(
      'kilometer',
    )
    expect(
      screen.getByRole('group', { name: 'Currency and units' }),
    ).toHaveAccessibleDescription(/are not converted/)
  })

  it('saves new settings and relabels other screens without a reload', async () => {
    const saved = {
      ...profile,
      currency: 'USD',
      volumeUnit: 'gallon',
      distanceUnit: 'mile',
    }
    const calls = backend({
      'GET /api/v1/me': () => response(200, profile),
      'PATCH /api/v1/me': () => response(200, saved),
      'GET /api/v1/cars': () =>
        response(200, {
          items: [
            {
              id: '9b0a5f0e-5d8f-4a55-9d59-0b8f1f3c2a10',
              attachmentCount: 0,
              type: 'Hatchback',
              make: 'VW',
              name: 'Golf',
              fuel: 'diesel',
              firstRegistration: null,
              licensePlate: null,
              fin: null,
              isActive: true,
              purchaseDate: null,
              purchasePrice: '18500.50',
              deregistrationDate: null,
              information: null,
              createdAt: '2026-09-27T10:00:00Z',
              updatedAt: '2026-09-27T10:00:00Z',
            },
          ],
          nextCursor: null,
        }),
    })
    const user = userEvent.setup()
    const { queryClient, router } = renderApp('/profile')

    await user.selectOptions(
      await screen.findByRole('combobox', { name: 'Currency' }),
      'USD',
    )
    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Volume' }),
      'gallon',
    )
    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Distance' }),
      'mile',
    )
    await user.click(screen.getByRole('button', { name: 'Save changes' }))

    expect(await screen.findByRole('status')).toHaveTextContent(
      'Your profile has been saved.',
    )
    const patch = calls.find((call) => call.key === 'PATCH /api/v1/me')
    expect(JSON.parse(String(patch?.init?.body))).toMatchObject({
      currency: 'USD',
      volumeUnit: 'gallon',
      distanceUnit: 'mile',
    })
    expect(
      queryClient.getQueryData<Session>(['session'])?.profile.currency,
    ).toBe('USD')

    await router.navigate({ to: '/cars' })
    const table = await screen.findByRole(
      'table',
      { name: 'Your cars' },
      { timeout: 5000 },
    )
    expect(table).toHaveTextContent('$18,500.50')
    // The currency select lists every ISO 4217 code, which makes the user
    // events in this test slower than the default timeout allows.
  }, 20_000)

  it('shows a backend currency rejection on the currency choice', async () => {
    backend({
      'GET /api/v1/me': () => response(200, profile),
      'PATCH /api/v1/me': () =>
        response(400, {
          code: 'validation_failed',
          message: 'request validation failed',
          fields: { currency: 'invalid_enum' },
        }),
    })
    const user = userEvent.setup()
    renderApp('/profile')

    await user.click(
      await screen.findByRole('button', { name: 'Save changes' }),
    )

    expect(
      await screen.findByText('Choose one of the offered values.'),
    ).toHaveAttribute('id', 'currency-error')
    expect(screen.getByRole('combobox', { name: 'Currency' })).toHaveAttribute(
      'aria-invalid',
      'true',
    )
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })

  it('follows a switch to German', async () => {
    backend({ 'GET /api/v1/me': () => response(200, profile) })
    const user = userEvent.setup()
    renderApp('/profile')

    await screen.findByRole('textbox', { name: 'First name' })
    await user.selectOptions(
      screen.getAllByRole('combobox', { name: 'Language' })[0],
      'de',
    )

    expect(
      await screen.findByRole('textbox', { name: 'Vorname' }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('textbox', { name: 'Nachname' }),
    ).toBeInTheDocument()
    expect(screen.getByText('E-Mail-Adresse')).toBeInTheDocument()
    expect(
      screen.getByText(/daher kann sie hier nicht geändert werden/),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Änderungen speichern' }),
    ).toBeInTheDocument()
    expect(screen.getByRole('combobox', { name: 'Währung' })).toHaveValue('EUR')
    expect(
      screen.getByRole('option', { name: 'Gallonen (gal)' }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('option', { name: 'Meilen (mi)' }),
    ).toBeInTheDocument()
    expect(screen.getByText(/werden nicht umgerechnet/)).toBeInTheDocument()
  })
})
