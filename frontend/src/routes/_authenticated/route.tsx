import { useQuery } from '@tanstack/react-query'
import { createFileRoute, redirect, useRouter } from '@tanstack/react-router'
import { useMemo } from 'react'
import { validReturnTo } from '#/auth/return-to'
import { sessionQueryKey, sessionQueryOptions } from '#/auth/session'
import { AppShell } from '#/components/app-shell'
import { SessionUnavailable } from '#/components/session-unavailable'
import { DisplayUnitsContext, displayUnitsOf } from '#/lib/display-units'

export const Route = createFileRoute('/_authenticated')({
  beforeLoad: async ({ context, location }) => {
    const session =
      await context.queryClient.ensureQueryData(sessionQueryOptions)
    if (!session) {
      throw redirect({
        to: '/auth/login',
        search: { returnTo: validReturnTo(location.href) },
      })
    }
    return { session }
  },
  component: AuthenticatedLayout,
  errorComponent: SessionError,
})

function AuthenticatedLayout() {
  const { session: resolved } = Route.useRouteContext()
  // Follow the cached session so profile edits reach the shell without a
  // reload; the route context value covers the moment before it is cached.
  const { data } = useQuery(sessionQueryOptions)
  const session = data ?? resolved
  const { currency, volumeUnit, distanceUnit } = session.profile
  const units = useMemo(
    () => displayUnitsOf({ currency, volumeUnit, distanceUnit }),
    [currency, volumeUnit, distanceUnit],
  )
  return (
    <DisplayUnitsContext.Provider value={units}>
      <AppShell session={session} />
    </DisplayUnitsContext.Provider>
  )
}

function SessionError({ reset }: { reset: () => void }) {
  const router = useRouter()
  function retry() {
    router.options.context.queryClient.removeQueries({
      queryKey: sessionQueryKey,
    })
    reset()
    void router.invalidate()
  }
  return (
    <main className="centered-page">
      <SessionUnavailable retry={retry} />
    </main>
  )
}
