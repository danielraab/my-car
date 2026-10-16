import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { i18n } from '#/i18n'
import { renderApp } from '#/test/render-app'

const session = {
  profile: {
    id: '617c3d87-21b4-4cb9-96f3-e03510892296',
    email: 'driver@example.com',
    firstName: 'Ada',
    lastName: 'Lovelace',
    currency: 'EUR',
    volumeUnit: 'liter',
    distanceUnit: 'kilometer',
  },
}

function response(status: number, body?: unknown) {
  return new Response(body === undefined ? null : JSON.stringify(body), {
    status,
    headers:
      body === undefined ? undefined : { 'Content-Type': 'application/json' },
  })
}

beforeEach(async () => {
  await i18n.changeLanguage('en')
  document.documentElement.lang = 'en'
  vi.unstubAllGlobals()
})

describe('public routes', () => {
  it('sends anonymous landing cards through login with their destination', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(401)))
    renderApp('/')

    const refuels = await screen.findByRole(
      'link',
      { name: /Refuels/ },
      { timeout: 3_000 },
    )
    expect(refuels).toHaveAttribute('href', '/auth/login?returnTo=%2Frefuels')
    expect(
      screen.getAllByRole('link', { name: /Sign in/ }).length,
    ).toBeGreaterThan(0)
    expect(
      screen.getByRole('link', { name: 'Source on GitHub' }),
    ).toHaveAttribute('href', 'https://github.com/danielraab/my-cars/')
  })

  it('sends authenticated landing cards directly to protected areas', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(200, session)))
    renderApp('/')

    expect(
      await screen.findByRole('link', { name: /Tickets/ }),
    ).toHaveAttribute('href', '/tickets')
  })

  it('submits a validated magic-link request with account-neutral feedback', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(response(200, { methods: ['magic_link'] }))
      .mockResolvedValueOnce(response(202))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    renderApp('/auth/login?returnTo=%2Frefuels')

    await user.type(
      await screen.findByRole('textbox', { name: 'Email address' }),
      'driver@example.com',
    )
    await user.click(
      screen.getByRole('button', { name: 'Email me a sign-in link' }),
    )

    expect(await screen.findByRole('status')).toHaveTextContent(
      'Check your inbox',
    )
    const [, init] = fetchMock.mock.calls[1]
    expect(JSON.parse(String(init.body))).toEqual({
      email: 'driver@example.com',
      returnTo: '/refuels',
    })
  })

  it('shows localized validation and exposes only the backend OIDC URL', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValue(response(200, { methods: ['magic_link', 'oidc'] })),
    )
    const user = userEvent.setup()
    renderApp('/auth/login?returnTo=https%3A%2F%2Fevil.example')

    await user.type(
      await screen.findByRole('textbox', { name: 'Email address' }),
      'invalid',
    )
    await user.click(
      screen.getByRole('button', { name: 'Email me a sign-in link' }),
    )

    expect(screen.getByRole('alert')).toHaveTextContent('Enter a valid email')
    expect(
      screen.getByRole('link', { name: 'Continue with identity provider' }),
    ).toHaveAttribute('href', '/api/v1/auth/oidc/start?returnTo=%2Fhome')
  })

  it('hides OIDC when only magic links are enabled', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(response(200, { methods: ['magic_link'] })),
    )
    renderApp('/auth/login')

    expect(
      await screen.findByRole('textbox', { name: 'Email address' }),
    ).toBeInTheDocument()
    expect(
      screen.queryByRole('link', { name: 'Continue with identity provider' }),
    ).not.toBeInTheDocument()
    expect(
      screen.getByText('Receive a secure sign-in link by email.'),
    ).toBeInTheDocument()
  })

  it('shows loading and a localized retry when methods are unavailable', async () => {
    const fetchMock = vi
      .fn()
      .mockImplementationOnce(() => new Promise(() => {}))
      .mockResolvedValueOnce(response(503, { code: 'unavailable' }))
      .mockResolvedValueOnce(response(200, { methods: ['magic_link'] }))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    const { unmount } = renderApp('/auth/login')

    expect(await screen.findByRole('status')).toHaveTextContent(
      'Loading sign-in options',
    )
    unmount()

    await i18n.changeLanguage('de')
    renderApp('/auth/login')
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Anmeldeoptionen nicht verfügbar',
    )
    await user.click(screen.getByRole('button', { name: 'Erneut versuchen' }))
    expect(
      await screen.findByRole('textbox', { name: 'E-Mail-Adresse' }),
    ).toBeInTheDocument()
  })
})

describe('protected routes', () => {
  it('redirects anonymous users without rendering protected content', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(401)))
    const { router } = renderApp('/cars')

    expect(
      await screen.findByRole('heading', { name: 'Sign in to My cars' }),
    ).toBeInTheDocument()
    expect(
      screen.queryByRole('heading', { name: 'Cars' }),
    ).not.toBeInTheDocument()
    expect(router.state.location.search).toEqual({ returnTo: '/cars' })
  })

  it('renders the dashboard inside the shell', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(200, session)))
    renderApp('/home')

    expect(
      await screen.findByRole('heading', { name: 'Dashboard', level: 1 }),
    ).toBeInTheDocument()
    expect(
      screen.getByText('Signed in as driver@example.com'),
    ).toBeInTheDocument()
    expect(screen.getByRole('navigation')).toBeInTheDocument()
    expect(
      screen.getByRole('link', { name: 'Source on GitHub' }),
    ).toHaveAttribute('href', 'https://github.com/danielraab/my-cars/')
  })

  it('presents server failure as retryable and recovers', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})
    const consoleWarn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        response(503, { code: 'unavailable', message: 'Unavailable' }),
      )
      .mockResolvedValueOnce(response(200, session))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    renderApp('/home')

    expect(
      await screen.findByRole('heading', {
        name: 'We could not check your session',
      }),
    ).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Try again' }))
    expect(
      await screen.findByRole('heading', { name: 'Dashboard', level: 1 }),
    ).toBeInTheDocument()
    consoleError.mockRestore()
    consoleWarn.mockRestore()
  })

  it('opens and closes keyboard-accessible mobile navigation', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(200, session)))
    const user = userEvent.setup()
    renderApp('/home')
    await screen.findByRole('heading', { name: 'Dashboard', level: 1 })

    await user.click(screen.getByRole('button', { name: 'Open navigation' }))
    const dialog = screen.getByRole('dialog')
    expect(within(dialog).getByRole('navigation')).toBeInTheDocument()
    expect(
      within(dialog).getByRole('link', { name: 'Cars' }),
    ).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Close navigation' }))
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
    )
  })
})

describe('logout', () => {
  it.each([204, 401])(
    'clears local session state after status %s',
    async (status) => {
      const fetchMock = vi.fn(async (_input: string, init?: RequestInit) =>
        init?.method === 'DELETE' ? response(status) : response(200, session),
      )
      vi.stubGlobal('fetch', fetchMock)
      const user = userEvent.setup()
      renderApp('/home')

      await user.click(await screen.findByRole('button', { name: 'Sign out' }))
      expect(
        await screen.findByRole('heading', {
          name: 'Keep every journey under control.',
        }),
      ).toBeInTheDocument()
    },
  )

  it('retains protected content when logout fails unexpectedly', async () => {
    const fetchMock = vi.fn(async (_input: string, init?: RequestInit) =>
      init?.method === 'DELETE'
        ? response(500, { code: 'failed', message: 'Failed' })
        : response(200, session),
    )
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    renderApp('/home')

    await user.click(await screen.findByRole('button', { name: 'Sign out' }))
    // The stubbed backend answers every request with the session, so the
    // dashboard's own statistics request fails and raises an alert too.
    await waitFor(() =>
      expect(
        screen
          .getAllByRole('alert')
          .some((alert) => alert.textContent?.includes('Sign-out failed')),
      ).toBe(true),
    )
    expect(
      screen.getByRole('heading', { name: 'Dashboard', level: 1 }),
    ).toBeInTheDocument()
  })
})
