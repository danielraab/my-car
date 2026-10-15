import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

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
const laptop = {
  id: '3f2b8f0e-1a7c-4c55-9d43-0f6f3c7b9a11',
  name: 'Laptop',
  createdAt: '2026-03-01T10:00:00Z',
  lastUsedAt: '2026-04-02T10:00:00Z',
  authenticatorName: 'Google Password Manager',
  backedUp: true,
}
const securityKey = {
  id: '9a6f1d2c-3b4e-4f50-8a9b-0c1d2e3f4a5b',
  name: 'Security key',
  createdAt: '2026-03-05T10:00:00Z',
  lastUsedAt: null,
  authenticatorName: null,
  backedUp: false,
}
const options = {
  publicKey: {
    challenge: 'AQID',
    rp: { id: 'cars.example', name: 'my-car' },
    user: { id: 'BAU', name: 'driver@example.com', displayName: 'Ada' },
  },
}

function response(status: number, body?: unknown) {
  return new Response(body === undefined ? null : JSON.stringify(body), {
    status,
    headers:
      body === undefined ? undefined : { 'Content-Type': 'application/json' },
  })
}

type Handler = (init?: RequestInit) => Response | Promise<Response>

// Routes fetch calls by "METHOD path". The session is valid unless a
// "GET /api/v1/session" route says otherwise.
function backend(routes: Record<string, Handler | Handler[]>) {
  const calls: { key: string; init?: RequestInit }[] = []
  vi.stubGlobal(
    'fetch',
    vi.fn(async (path: string, init?: RequestInit) => {
      const key = `${init?.method ?? 'GET'} ${path}`
      calls.push({ key, init })
      const route = routes[key]
      const handler = Array.isArray(route) ? route.shift() : route
      if (handler) return handler(init)
      if (key === 'GET /api/v1/session') return response(200, { profile })
      throw new Error(`unexpected request ${key}`)
    }),
  )
  return calls
}

const bytes = (...values: number[]) => new Uint8Array(values).buffer

function stubWebAuthn(credentials: {
  create?: () => Promise<unknown>
  get?: () => Promise<unknown>
}) {
  vi.stubGlobal('PublicKeyCredential', function PublicKeyCredential() {})
  Object.defineProperty(navigator, 'credentials', {
    configurable: true,
    value: {
      create: vi.fn(credentials.create ?? (() => Promise.resolve(null))),
      get: vi.fn(credentials.get ?? (() => Promise.resolve(null))),
    },
  })
}

const createdCredential = {
  id: 'AQI',
  rawId: bytes(1, 2),
  type: 'public-key',
  getClientExtensionResults: () => ({}),
  response: {
    clientDataJSON: bytes(3),
    attestationObject: bytes(4),
    getTransports: () => ['internal'],
  },
}
const assertedCredential = {
  id: 'AQI',
  rawId: bytes(1, 2),
  type: 'public-key',
  getClientExtensionResults: () => ({}),
  response: {
    clientDataJSON: bytes(3),
    authenticatorData: bytes(4),
    signature: bytes(5),
    userHandle: bytes(6),
  },
}

beforeEach(async () => {
  await i18n.changeLanguage('en')
  document.documentElement.lang = 'en'
  vi.unstubAllGlobals()
})

afterEach(() => {
  Reflect.deleteProperty(navigator, 'credentials')
})

describe('passkey sign-in', () => {
  const methods = (list: string[]) => () => response(200, { methods: list })

  it('is offered only when enabled and supported by the browser', async () => {
    backend({
      'GET /api/v1/session': () => response(401),
      'GET /api/v1/auth/methods': methods(['magic_link', 'passkey']),
    })
    const { unmount } = renderApp('/auth/login')
    expect(
      await screen.findByRole('button', { name: 'Email me a sign-in link' }),
    ).toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Sign in with a passkey' }),
    ).not.toBeInTheDocument()
    unmount()

    stubWebAuthn({})
    backend({
      'GET /api/v1/session': () => response(401),
      'GET /api/v1/auth/methods': methods(['magic_link']),
    })
    const second = renderApp('/auth/login')
    expect(
      await screen.findByRole('button', { name: 'Email me a sign-in link' }),
    ).toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Sign in with a passkey' }),
    ).not.toBeInTheDocument()
    second.unmount()

    backend({
      'GET /api/v1/session': () => response(401),
      'GET /api/v1/auth/methods': methods(['magic_link', 'passkey']),
    })
    renderApp('/auth/login')
    expect(
      await screen.findByRole('button', { name: 'Sign in with a passkey' }),
    ).toBeInTheDocument()
    expect(screen.getByText('or')).toBeInTheDocument()
  })

  it('signs in without an email and continues to the return path', async () => {
    stubWebAuthn({ get: () => Promise.resolve(assertedCredential) })
    // The login page itself never resolves the session, so the first session
    // request is the protected route's, after the passkey login succeeded.
    const calls = backend({
      'GET /api/v1/auth/methods': methods(['magic_link', 'passkey']),
      'POST /api/v1/auth/passkey/options': () => response(200, options),
      'POST /api/v1/auth/passkey': () => response(204),
      'GET /api/v1/me': () => response(200, profile),
      'GET /api/v1/passkeys': () => response(200, { items: [laptop] }),
    })
    const user = userEvent.setup()
    renderApp('/auth/login?returnTo=%2Fprofile')

    await user.click(
      await screen.findByRole('button', { name: 'Sign in with a passkey' }),
    )
    expect(
      await screen.findByRole('heading', { name: 'Profile', level: 1 }),
    ).toBeInTheDocument()
    const keys = calls.map((call) => call.key)
    expect(keys.indexOf('GET /api/v1/session')).toBeGreaterThan(
      keys.indexOf('POST /api/v1/auth/passkey'),
    )
    const login = calls.find((call) => call.key === 'POST /api/v1/auth/passkey')
    const body = JSON.parse(String(login?.init?.body))
    expect(body.response.userHandle).toBe('Bg')
    expect(body).not.toHaveProperty('email')
  })

  it('reports a rejected passkey and stays on the login page', async () => {
    stubWebAuthn({ get: () => Promise.resolve(assertedCredential) })
    backend({
      'GET /api/v1/session': () => response(401),
      'GET /api/v1/auth/methods': methods(['magic_link', 'passkey']),
      'POST /api/v1/auth/passkey/options': () => response(200, options),
      'POST /api/v1/auth/passkey': () =>
        response(401, { code: 'unauthorized', message: 'no' }),
    })
    const user = userEvent.setup()
    renderApp('/auth/login')

    await user.click(
      await screen.findByRole('button', { name: 'Sign in with a passkey' }),
    )
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Your passkey could not be verified',
    )
    expect(
      screen.getByRole('heading', { name: 'Sign in to My cars' }),
    ).toBeInTheDocument()
  })

  it('stays silent when the browser prompt is cancelled', async () => {
    stubWebAuthn({
      get: () => Promise.reject(new DOMException('cancel', 'NotAllowedError')),
    })
    backend({
      'GET /api/v1/session': () => response(401),
      'GET /api/v1/auth/methods': methods(['magic_link', 'passkey']),
      'POST /api/v1/auth/passkey/options': () => response(200, options),
    })
    const user = userEvent.setup()
    renderApp('/auth/login')

    const button = await screen.findByRole('button', {
      name: 'Sign in with a passkey',
    })
    await user.click(button)
    await waitFor(() => expect(button).toBeEnabled())
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('is localized in German', async () => {
    await i18n.changeLanguage('de')
    stubWebAuthn({})
    backend({
      'GET /api/v1/session': () => response(401),
      'GET /api/v1/auth/methods': methods(['magic_link', 'passkey']),
    })
    renderApp('/auth/login')
    expect(
      await screen.findByRole('button', { name: 'Mit Passkey anmelden' }),
    ).toBeInTheDocument()
  })
})

describe('profile passkeys', () => {
  const profileRoutes = {
    'GET /api/v1/me': () => response(200, profile),
  }

  async function passkeySection() {
    const heading = await screen.findByRole('heading', {
      name: 'Passkeys',
      level: 2,
    })
    return heading.closest('section') as HTMLElement
  }

  it('lists passkeys with their details and the fresh-login hint', async () => {
    stubWebAuthn({})
    backend({
      ...profileRoutes,
      'GET /api/v1/passkeys': () =>
        response(200, { items: [laptop, securityKey] }),
    })
    renderApp('/profile')

    const section = await passkeySection()
    const list = await within(section).findByRole('list', {
      name: 'Your passkeys',
    })
    const [first, second] = within(list).getAllByRole('listitem')
    expect(first).toHaveTextContent('Laptop')
    expect(first).toHaveTextContent(
      'Google Password Manager · Synced across devices',
    )
    expect(first).toHaveTextContent('Added Mar 1, 2026')
    expect(first).toHaveTextContent('Last used Apr 2, 2026')
    expect(second).toHaveTextContent('Stored on this device only')
    expect(second).toHaveTextContent('Not used for sign-in yet')
    expect(section).toHaveTextContent(
      'adding a passkey requires a sign-in within the last 5 minutes',
    )
    expect(
      within(section).getByRole('button', { name: 'Add passkey' }),
    ).toBeInTheDocument()
  })

  it('adds a passkey with a default name', async () => {
    stubWebAuthn({ create: () => Promise.resolve(createdCredential) })
    const created = { ...securityKey, name: 'Passkey 1' }
    const calls = backend({
      ...profileRoutes,
      'GET /api/v1/passkeys': () => response(200, { items: [] }),
      'POST /api/v1/passkeys/registration-options': () =>
        response(200, options),
      'POST /api/v1/passkeys': () => response(201, created),
    })
    const user = userEvent.setup()
    renderApp('/profile')

    const section = await passkeySection()
    expect(
      await within(section).findByText('You have not added a passkey yet.'),
    ).toBeInTheDocument()
    await user.click(
      within(section).getByRole('button', { name: 'Add passkey' }),
    )

    expect(
      await within(section).findByText('Your passkey has been added.'),
    ).toBeInTheDocument()
    expect(within(section).getByRole('listitem')).toHaveTextContent('Passkey 1')
    const register = calls.find((call) => call.key === 'POST /api/v1/passkeys')
    expect(JSON.parse(String(register?.init?.body))).toMatchObject({
      name: 'Passkey 1',
      credential: { rawId: 'AQI', response: { attestationObject: 'BA' } },
    })
  })

  it('offers a fresh login when the session is too old', async () => {
    stubWebAuthn({ create: () => Promise.resolve(createdCredential) })
    backend({
      ...profileRoutes,
      'GET /api/v1/passkeys': () => response(200, { items: [] }),
      'POST /api/v1/passkeys/registration-options': () =>
        response(403, {
          code: 'reauthentication_required',
          message: 'login again',
        }),
    })
    const user = userEvent.setup()
    renderApp('/profile')

    const section = await passkeySection()
    await user.click(
      await within(section).findByRole('button', { name: 'Add passkey' }),
    )
    const alert = await within(section).findByRole('alert')
    expect(alert).toHaveTextContent('more than 5 minutes ago')
    expect(
      within(alert).getByRole('link', { name: 'Sign in again' }),
    ).toHaveAttribute('href', '/auth/login?returnTo=%2Fprofile')
    expect(navigator.credentials.create).not.toHaveBeenCalled()
    expect(section).not.toHaveTextContent('Your passkey has been added.')
  })

  it('shows a neutral notice when adding is cancelled', async () => {
    stubWebAuthn({
      create: () =>
        Promise.reject(new DOMException('cancel', 'NotAllowedError')),
    })
    backend({
      ...profileRoutes,
      'GET /api/v1/passkeys': () => response(200, { items: [laptop] }),
      'POST /api/v1/passkeys/registration-options': () =>
        response(200, options),
    })
    const user = userEvent.setup()
    renderApp('/profile')

    const section = await passkeySection()
    await user.click(
      await within(section).findByRole('button', { name: 'Add passkey' }),
    )
    expect(
      await within(section).findByText('Adding the passkey was cancelled.'),
    ).toBeInTheDocument()
    expect(within(section).queryByRole('alert')).not.toBeInTheDocument()
    expect(within(section).getAllByRole('listitem')).toHaveLength(1)
  })

  it('hides adding without browser support but keeps management', async () => {
    backend({
      ...profileRoutes,
      'GET /api/v1/passkeys': () => response(200, { items: [laptop] }),
      'PATCH /api/v1/passkeys/3f2b8f0e-1a7c-4c55-9d43-0f6f3c7b9a11': () =>
        response(200, { ...laptop, name: 'Work laptop' }),
    })
    const user = userEvent.setup()
    renderApp('/profile')

    const section = await passkeySection()
    await within(section).findByRole('listitem')
    expect(
      within(section).queryByRole('button', { name: 'Add passkey' }),
    ).not.toBeInTheDocument()
    expect(section).toHaveTextContent('This browser does not support passkeys.')

    await user.click(
      within(section).getByRole('button', { name: 'Rename Laptop' }),
    )
    const input = within(section).getByRole('textbox', { name: 'Passkey name' })
    await user.clear(input)
    await user.click(within(section).getByRole('button', { name: 'Save name' }))
    expect(await within(section).findByRole('alert')).toHaveTextContent(
      'Enter a name.',
    )
    await user.type(input, 'Work laptop')
    await user.click(within(section).getByRole('button', { name: 'Save name' }))
    expect(
      await within(section).findByRole('heading', { name: 'Work laptop' }),
    ).toBeInTheDocument()
  })

  it('removes a passkey after confirmation', async () => {
    stubWebAuthn({})
    backend({
      ...profileRoutes,
      'GET /api/v1/passkeys': () =>
        response(200, { items: [laptop, securityKey] }),
      'DELETE /api/v1/passkeys/3f2b8f0e-1a7c-4c55-9d43-0f6f3c7b9a11': () =>
        response(204),
    })
    const user = userEvent.setup()
    renderApp('/profile')

    const section = await passkeySection()
    const [first] = await within(section).findAllByRole('listitem')
    await user.click(within(first).getByRole('button', { name: 'Remove' }))
    expect(first).toHaveTextContent('Remove “Laptop”?')
    await user.click(
      within(first).getByRole('button', { name: 'Remove passkey' }),
    )

    expect(
      await within(section).findByText('The passkey has been removed.'),
    ).toBeInTheDocument()
    expect(within(section).getAllByRole('listitem')).toHaveLength(1)
    expect(section).not.toHaveTextContent('Laptop')
  })

  it('returns to login when removing the passkey of the current session', async () => {
    stubWebAuthn({})
    backend({
      ...profileRoutes,
      'GET /api/v1/session': [
        () => response(200, { profile }),
        () => response(401),
      ],
      'GET /api/v1/auth/methods': () =>
        response(200, { methods: ['magic_link', 'passkey'] }),
      'GET /api/v1/passkeys': () => response(200, { items: [laptop] }),
      'DELETE /api/v1/passkeys/3f2b8f0e-1a7c-4c55-9d43-0f6f3c7b9a11': () =>
        response(204),
    })
    const user = userEvent.setup()
    renderApp('/profile')

    const section = await passkeySection()
    const item = await within(section).findByRole('listitem')
    await user.click(within(item).getByRole('button', { name: 'Remove' }))
    await user.click(
      within(item).getByRole('button', { name: 'Remove passkey' }),
    )
    expect(
      await screen.findByRole('heading', { name: 'Sign in to My cars' }),
    ).toBeInTheDocument()
  })

  it('shows a retryable error when passkeys cannot be loaded', async () => {
    stubWebAuthn({})
    backend({
      ...profileRoutes,
      'GET /api/v1/passkeys': [
        () => response(503, { code: 'unavailable', message: 'down' }),
        () => response(200, { items: [laptop] }),
      ],
    })
    const user = userEvent.setup()
    renderApp('/profile')

    const section = await passkeySection()
    const alert = await within(section).findByRole('alert')
    expect(alert).toHaveTextContent('We could not load your passkeys.')
    await user.click(within(alert).getByRole('button', { name: 'Try again' }))
    expect(await within(section).findByRole('listitem')).toHaveTextContent(
      'Laptop',
    )
  })

  it('is localized in German', async () => {
    await i18n.changeLanguage('de')
    stubWebAuthn({})
    backend({
      ...profileRoutes,
      'GET /api/v1/passkeys': () => response(200, { items: [securityKey] }),
    })
    renderApp('/profile')

    const heading = await screen.findByRole('heading', {
      name: 'Passkeys',
      level: 2,
    })
    const section = heading.closest('section') as HTMLElement
    expect(
      await within(section).findByText('Nur auf diesem Gerät gespeichert'),
    ).toBeInTheDocument()
    expect(
      within(section).getByRole('button', { name: 'Passkey hinzufügen' }),
    ).toBeInTheDocument()
    expect(section).toHaveTextContent('innerhalb der letzten 5 Minuten')
  })
})
