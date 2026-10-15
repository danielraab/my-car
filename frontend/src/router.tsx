import { QueryClient } from '@tanstack/react-query'
import type { createMemoryHistory } from '@tanstack/react-router'
import { createRouter as createTanStackRouter } from '@tanstack/react-router'

import { routeTree } from './routeTree.gen'

export function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, refetchOnWindowFocus: false },
      mutations: { retry: false },
    },
  })
}

export function getRouter(
  queryClient = createQueryClient(),
  history?: ReturnType<typeof createMemoryHistory>,
) {
  const router = createTanStackRouter({
    routeTree,
    context: { queryClient },
    history,
    scrollRestoration: true,
    defaultPreload: 'intent',
    defaultPreloadStaleTime: 0,
  })

  return router
}

declare module '@tanstack/react-router' {
  interface Register {
    router: ReturnType<typeof getRouter>
  }
  interface HistoryState {
    // Files of a just-created entry that could not be uploaded, reported on
    // its edit screen.
    failedUploads?: string[]
  }
}
