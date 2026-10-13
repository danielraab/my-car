import type {
  PasskeyCreationOptionsJSON,
  PasskeyRequestOptionsJSON,
  PublicKeyCredentialJSON,
} from '#/auth/passkeys'
import type { components } from './schema.gen'

export type ApiErrorBody = components['schemas']['Error']
export type Session = components['schemas']['Session']
export type Profile = components['schemas']['Profile']
export type ProfileUpdate = components['schemas']['ProfileUpdate']
export type MagicLinkRequest = components['schemas']['MagicLinkRequest']
export type AuthenticationMethods =
  components['schemas']['AuthenticationMethods']
export type AuthenticationMethod = AuthenticationMethods['methods'][number]
export const authenticationMethods: readonly AuthenticationMethod[] = [
  'magic_link',
  'passkey',
  'oidc',
]
export type PasskeySummary = components['schemas']['PasskeySummary']
export type Car = components['schemas']['Car']
export type CarFuel = Car['fuel']
type GeneratedCarInput = components['schemas']['CarInput']
// The generator marks defaulted members as required; isActive may be omitted.
export type CarInput = Omit<GeneratedCarInput, 'isActive'> &
  Partial<Pick<GeneratedCarInput, 'isActive'>>
export type CarUpdate = components['schemas']['CarUpdate']
export type CarPage = components['schemas']['CarPage']

export type Repair = components['schemas']['Repair']
export type RepairType = Repair['type']
export type RepairInput = components['schemas']['RepairInput']
export type RepairUpdate = components['schemas']['RepairUpdate']
export type RepairPage = components['schemas']['RepairPage']
export type Ticket = components['schemas']['Ticket']
export type TicketType = Ticket['type']
export type TicketInput = components['schemas']['TicketInput']
export type TicketUpdate = components['schemas']['TicketUpdate']
export type TicketPage = components['schemas']['TicketPage']
export type Refuel = components['schemas']['Refuel']
export type RefuelInput = components['schemas']['RefuelInput']
export type RefuelUpdate = components['schemas']['RefuelUpdate']
export type RefuelPage = components['schemas']['RefuelPage']
export type RefuelChart = components['schemas']['RefuelChart']
export type RefuelFuel = Refuel['fuel']
export type ExpenseStatistics = components['schemas']['ExpenseStatistics']
export type ExpenseRow = ExpenseStatistics['items'][number]
export type ExpenseKind = ExpenseRow['kind']
export const expenseKinds: readonly ExpenseKind[] = [
  'refuel',
  'repair',
  'ticket',
]
export const refuelFuels: readonly RefuelFuel[] = ['normal', 'special', 'other']

export const repairTypes: readonly RepairType[] = [
  'check',
  'service',
  'wearing_part',
  'crash_repair',
]

export const ticketTypes: readonly TicketType[] = [
  'parking',
  'velocity',
  'other',
]

export const carFuels: readonly CarFuel[] = [
  'gasoline',
  'diesel',
  'electric',
  'other',
]

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    message: string,
    readonly fields?: Record<string, string>,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

export class UnauthorizedError extends ApiError {
  constructor(message = 'Authentication required') {
    super(401, 'unauthorized', message)
    this.name = 'UnauthorizedError'
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function isProfile(value: unknown): value is Profile {
  if (!isRecord(value)) return false
  const { id, email, firstName, lastName } = value
  return [id, email, firstName, lastName].every(
    (field) => typeof field === 'string',
  )
}

function isNullableString(value: unknown): value is string | null {
  return value === null || typeof value === 'string'
}

function isCar(value: unknown): value is Car {
  if (!isRecord(value)) return false
  const required = [
    value.id,
    value.type,
    value.make,
    value.name,
    value.createdAt,
    value.updatedAt,
  ]
  return (
    required.every((field) => typeof field === 'string') &&
    carFuels.some((fuel) => fuel === value.fuel) &&
    typeof value.isActive === 'boolean' &&
    [
      value.firstRegistration,
      value.licensePlate,
      value.fin,
      value.purchaseDate,
      value.purchasePrice,
      value.deregistrationDate,
      value.information,
    ].every(isNullableString)
  )
}

function isCarPage(value: unknown): value is CarPage {
  return (
    isRecord(value) &&
    Array.isArray(value.items) &&
    value.items.every(isCar) &&
    isNullableString(value.nextCursor)
  )
}

function isRepair(value: unknown): value is Repair {
  if (!isRecord(value)) return false
  const { odometerReading } = value
  return (
    [
      value.id,
      value.carId,
      value.date,
      value.station,
      value.amount,
      value.description,
    ].every((field) => typeof field === 'string') &&
    repairTypes.some((type) => type === value.type) &&
    (odometerReading === null ||
      (typeof odometerReading === 'number' &&
        Number.isInteger(odometerReading)))
  )
}

function isRepairPage(value: unknown): value is RepairPage {
  return (
    isRecord(value) &&
    Array.isArray(value.items) &&
    value.items.every(isRepair) &&
    isNullableString(value.nextCursor)
  )
}

function isTicket(value: unknown): value is Ticket {
  if (!isRecord(value)) return false
  return (
    [
      value.id,
      value.carId,
      value.date,
      value.location,
      value.amount,
      value.description,
    ].every((field) => typeof field === 'string') &&
    ticketTypes.some((type) => type === value.type)
  )
}

function isTicketPage(value: unknown): value is TicketPage {
  return (
    isRecord(value) &&
    Array.isArray(value.items) &&
    value.items.every(isTicket) &&
    isNullableString(value.nextCursor)
  )
}

function isRefuel(value: unknown): value is Refuel {
  if (!isRecord(value)) return false
  const nullableNumber = (v: unknown) =>
    v === null || (typeof v === 'number' && Number.isInteger(v))
  return (
    [
      value.id,
      value.carId,
      value.date,
      value.station,
      value.liters,
      value.amount,
      value.perLiter,
    ].every((v) => typeof v === 'string') &&
    refuelFuels.some((fuel) => fuel === value.fuel) &&
    nullableNumber(value.odometerReading) &&
    nullableNumber(value.distance) &&
    isNullableString(value.consumption)
  )
}

function isRefuelPage(value: unknown): value is RefuelPage {
  return (
    isRecord(value) &&
    Array.isArray(value.items) &&
    value.items.every(isRefuel) &&
    isNullableString(value.nextCursor)
  )
}

function isRefuelChart(value: unknown): value is RefuelChart {
  return (
    isRecord(value) && Array.isArray(value.items) && value.items.every(isRefuel)
  )
}

function isExpenseStatistics(value: unknown): value is ExpenseStatistics {
  return (
    isRecord(value) &&
    Array.isArray(value.items) &&
    value.items.every(
      (item) =>
        isRecord(item) &&
        typeof item.date === 'string' &&
        typeof item.amount === 'string' &&
        expenseKinds.some((kind) => kind === item.kind),
    )
  )
}

function isStringList(value: unknown): value is string[] {
  return Array.isArray(value) && value.every((item) => typeof item === 'string')
}

function isSession(value: unknown): value is Session {
  return isRecord(value) && isProfile(value.profile)
}

function isAuthenticationMethods(
  value: unknown,
): value is AuthenticationMethods {
  if (!isRecord(value) || !Array.isArray(value.methods)) return false
  const methods = value.methods
  return (
    methods.length > 0 &&
    methods.includes('magic_link') &&
    new Set(methods).size === methods.length &&
    methods.every((method) => authenticationMethods.includes(method))
  )
}

function isPasskeySummary(value: unknown): value is PasskeySummary {
  return (
    isRecord(value) &&
    [value.id, value.name, value.createdAt].every(
      (field) => typeof field === 'string',
    ) &&
    isNullableString(value.lastUsedAt) &&
    isNullableString(value.authenticatorName) &&
    typeof value.backedUp === 'boolean'
  )
}

function isPasskeyList(value: unknown): value is { items: PasskeySummary[] } {
  return (
    isRecord(value) &&
    Array.isArray(value.items) &&
    value.items.every(isPasskeySummary)
  )
}

function isWebAuthnOptions(
  value: unknown,
): value is { publicKey: Record<string, unknown> } {
  return (
    isRecord(value) &&
    isRecord(value.publicKey) &&
    typeof value.publicKey.challenge === 'string'
  )
}

async function jsonBody(response: Response): Promise<unknown> {
  try {
    return await response.json()
  } catch {
    throw new ApiError(
      502,
      'invalid_response',
      'The server response is invalid',
    )
  }
}

export async function getAuthenticationMethods(): Promise<AuthenticationMethods> {
  const response = await fetchApi('/api/v1/auth/methods')
  if (!response.ok) throw await errorFrom(response)
  const body = await jsonBody(response)
  if (!isAuthenticationMethods(body)) {
    throw new ApiError(
      502,
      'invalid_response',
      'The server response is invalid',
    )
  }
  return body
}

async function errorFrom(response: Response): Promise<ApiError> {
  let body: unknown
  try {
    body = await response.json()
  } catch {
    body = undefined
  }

  if (isRecord(body)) {
    const code = typeof body.code === 'string' ? body.code : 'request_failed'
    const message =
      typeof body.message === 'string' ? body.message : response.statusText
    const fields = isRecord(body.fields)
      ? Object.fromEntries(
          Object.entries(body.fields).filter(
            (entry): entry is [string, string] => typeof entry[1] === 'string',
          ),
        )
      : undefined
    return new ApiError(response.status, code, message, fields)
  }

  return new ApiError(
    response.status,
    'request_failed',
    response.statusText || 'Request failed',
  )
}

async function fetchApi(path: string, init?: RequestInit): Promise<Response> {
  try {
    return await fetch(path, {
      ...init,
      credentials: 'same-origin',
      headers: {
        Accept: 'application/json',
        ...init?.headers,
      },
    })
  } catch {
    throw new ApiError(0, 'network_error', 'The server could not be reached')
  }
}

export async function getSession(): Promise<Session> {
  const response = await fetchApi('/api/v1/session')
  if (response.status === 401) throw new UnauthorizedError()
  if (!response.ok) throw await errorFrom(response)

  const body = await jsonBody(response)
  if (!isSession(body)) {
    throw new ApiError(
      502,
      'invalid_response',
      'The server response is invalid',
    )
  }
  return body
}

export async function getMe(): Promise<Profile> {
  const response = await fetchApi('/api/v1/me')
  return profileFrom(response)
}

export async function updateMe(input: ProfileUpdate): Promise<Profile> {
  const response = await fetchApi('/api/v1/me', {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  return profileFrom(response)
}

function profileFrom(response: Response): Promise<Profile> {
  return decoded(response, isProfile)
}

async function decoded<T>(
  response: Response,
  guard: (value: unknown) => value is T,
): Promise<T> {
  if (response.status === 401) throw new UnauthorizedError()
  if (!response.ok) throw await errorFrom(response)

  const body = await jsonBody(response)
  if (!guard(body)) {
    throw new ApiError(
      502,
      'invalid_response',
      'The server response is invalid',
    )
  }
  return body
}

function carPath(carId: string): string {
  return `/api/v1/cars/${encodeURIComponent(carId)}`
}

export async function getCars(
  options: { cursor?: string | null; limit?: number } = {},
): Promise<CarPage> {
  const query = new URLSearchParams()
  if (options.cursor) query.set('cursor', options.cursor)
  if (options.limit !== undefined) query.set('limit', String(options.limit))
  const search = query.toString()
  return decoded(
    await fetchApi(`/api/v1/cars${search ? `?${search}` : ''}`),
    isCarPage,
  )
}

export async function getCar(carId: string): Promise<Car> {
  return decoded(await fetchApi(carPath(carId)), isCar)
}

export async function createCar(input: CarInput): Promise<Car> {
  const response = await fetchApi('/api/v1/cars', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  return decoded(response, isCar)
}

export async function updateCar(carId: string, input: CarUpdate): Promise<Car> {
  const response = await fetchApi(carPath(carId), {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  return decoded(response, isCar)
}

export async function deleteCar(carId: string): Promise<void> {
  const response = await fetchApi(carPath(carId), { method: 'DELETE' })
  if (response.status === 401) throw new UnauthorizedError()
  if (!response.ok) throw await errorFrom(response)
  if (response.status !== 204) {
    throw new ApiError(
      502,
      'invalid_response',
      'The server response is invalid',
    )
  }
}

// Narrows an expense collection or chart to one of the caller's cars and a
// date range; `from` is inclusive and `to` exclusive (ISO date-times).
export type ExpenseFilter = { carId?: string; from?: string; to?: string }

function setFilter(query: URLSearchParams, filter: ExpenseFilter) {
  if (filter.carId) query.set('carId', filter.carId)
  if (filter.from) query.set('from', filter.from)
  if (filter.to) query.set('to', filter.to)
}

function repairPath(repairId: string): string {
  return `/api/v1/repairs/${encodeURIComponent(repairId)}`
}

export async function getRepairs(
  options: { cursor?: string | null; limit?: number } & ExpenseFilter = {},
): Promise<RepairPage> {
  const query = new URLSearchParams()
  if (options.cursor) query.set('cursor', options.cursor)
  if (options.limit !== undefined) query.set('limit', String(options.limit))
  setFilter(query, options)
  const search = query.toString()
  return decoded(
    await fetchApi(`/api/v1/repairs${search ? `?${search}` : ''}`),
    isRepairPage,
  )
}

export async function getRepair(repairId: string): Promise<Repair> {
  return decoded(await fetchApi(repairPath(repairId)), isRepair)
}

export async function getRepairStations(): Promise<string[]> {
  return decoded(await fetchApi('/api/v1/repairs/stations'), isStringList)
}

export async function createRepair(input: RepairInput): Promise<Repair> {
  const response = await fetchApi('/api/v1/repairs', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  return decoded(response, isRepair)
}

export async function updateRepair(
  repairId: string,
  input: RepairUpdate,
): Promise<Repair> {
  const response = await fetchApi(repairPath(repairId), {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  return decoded(response, isRepair)
}

export async function deleteRepair(repairId: string): Promise<void> {
  const response = await fetchApi(repairPath(repairId), { method: 'DELETE' })
  if (response.status === 401) throw new UnauthorizedError()
  if (!response.ok) throw await errorFrom(response)
  if (response.status !== 204) {
    throw new ApiError(
      502,
      'invalid_response',
      'The server response is invalid',
    )
  }
}

function ticketPath(ticketId: string): string {
  return `/api/v1/tickets/${encodeURIComponent(ticketId)}`
}

export async function getTickets(
  options: { cursor?: string | null; limit?: number } & ExpenseFilter = {},
): Promise<TicketPage> {
  const query = new URLSearchParams()
  if (options.cursor) query.set('cursor', options.cursor)
  if (options.limit !== undefined) query.set('limit', String(options.limit))
  setFilter(query, options)
  const search = query.toString()
  return decoded(
    await fetchApi(`/api/v1/tickets${search ? `?${search}` : ''}`),
    isTicketPage,
  )
}

export async function getTicket(ticketId: string): Promise<Ticket> {
  return decoded(await fetchApi(ticketPath(ticketId)), isTicket)
}

export async function getTicketLocations(): Promise<string[]> {
  return decoded(await fetchApi('/api/v1/tickets/locations'), isStringList)
}

export async function createTicket(input: TicketInput): Promise<Ticket> {
  const response = await fetchApi('/api/v1/tickets', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  return decoded(response, isTicket)
}

export async function updateTicket(
  ticketId: string,
  input: TicketUpdate,
): Promise<Ticket> {
  const response = await fetchApi(ticketPath(ticketId), {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  return decoded(response, isTicket)
}

export async function deleteTicket(ticketId: string): Promise<void> {
  const response = await fetchApi(ticketPath(ticketId), { method: 'DELETE' })
  if (response.status === 401) throw new UnauthorizedError()
  if (!response.ok) throw await errorFrom(response)
  if (response.status !== 204) {
    throw new ApiError(
      502,
      'invalid_response',
      'The server response is invalid',
    )
  }
}

function refuelPath(id: string) {
  return `/api/v1/refuels/${encodeURIComponent(id)}`
}
export async function getRefuels(
  options: { cursor?: string | null; limit?: number } & ExpenseFilter = {},
): Promise<RefuelPage> {
  const q = new URLSearchParams()
  if (options.cursor) q.set('cursor', options.cursor)
  if (options.limit) q.set('limit', String(options.limit))
  setFilter(q, options)
  return decoded(
    await fetchApi(`/api/v1/refuels${q.size ? `?${q}` : ''}`),
    isRefuelPage,
  )
}
export async function getRefuel(id: string): Promise<Refuel> {
  return decoded(await fetchApi(refuelPath(id)), isRefuel)
}
export async function getRefuelStations(): Promise<string[]> {
  return decoded(await fetchApi('/api/v1/refuels/stations'), isStringList)
}
export async function getRefuelChart(
  filter: ExpenseFilter = {},
): Promise<RefuelChart> {
  const q = new URLSearchParams()
  setFilter(q, filter)
  return decoded(
    await fetchApi(`/api/v1/refuels/chart${q.size ? `?${q}` : ''}`),
    isRefuelChart,
  )
}
// Every expense of the caller's in [from, to), optionally for one car, as
// rows the client groups itself: only it knows the viewer's time zone.
export async function getExpenseStatistics(
  filter: ExpenseFilter & { from: string; to: string },
): Promise<ExpenseStatistics> {
  const q = new URLSearchParams()
  setFilter(q, filter)
  return decoded(
    await fetchApi(`/api/v1/stats/expenses?${q}`),
    isExpenseStatistics,
  )
}
export async function createRefuel(input: RefuelInput): Promise<Refuel> {
  return decoded(
    await fetchApi('/api/v1/refuels', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    }),
    isRefuel,
  )
}
export async function updateRefuel(
  id: string,
  input: RefuelUpdate,
): Promise<Refuel> {
  return decoded(
    await fetchApi(refuelPath(id), {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    }),
    isRefuel,
  )
}
export async function deleteRefuel(id: string): Promise<void> {
  const response = await fetchApi(refuelPath(id), { method: 'DELETE' })
  if (response.status === 401) throw new UnauthorizedError()
  if (!response.ok) throw await errorFrom(response)
  if (response.status !== 204)
    throw new ApiError(
      502,
      'invalid_response',
      'The server response is invalid',
    )
}

export async function requestMagicLink(input: MagicLinkRequest): Promise<void> {
  const response = await fetchApi('/api/v1/auth/magic-links', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  if (!response.ok) throw await errorFrom(response)
  if (response.status !== 202) {
    throw new ApiError(
      502,
      'invalid_response',
      'The server response is invalid',
    )
  }
}

export async function logout(): Promise<void> {
  const response = await fetchApi('/api/v1/session', { method: 'DELETE' })
  if (response.status === 401) throw new UnauthorizedError()
  if (!response.ok) throw await errorFrom(response)
  if (response.status !== 204) {
    throw new ApiError(
      502,
      'invalid_response',
      'The server response is invalid',
    )
  }
}

function passkeyPath(passkeyId: string): string {
  return `/api/v1/passkeys/${encodeURIComponent(passkeyId)}`
}

function jsonRequest(method: string, body?: unknown): RequestInit {
  return body === undefined
    ? { method }
    : {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      }
}

async function expectNoContent(response: Response): Promise<void> {
  if (!response.ok) throw await errorFrom(response)
  if (response.status !== 204) {
    throw new ApiError(
      502,
      'invalid_response',
      'The server response is invalid',
    )
  }
}

export async function getPasskeys(): Promise<PasskeySummary[]> {
  const response = await fetchApi('/api/v1/passkeys')
  return (await decoded(response, isPasskeyList)).items
}

/** Starts registration; fails with code `reauthentication_required` (403)
 * when the session is older than the fresh-login window. */
export async function startPasskeyRegistration(): Promise<PasskeyCreationOptionsJSON> {
  const response = await fetchApi(
    '/api/v1/passkeys/registration-options',
    jsonRequest('POST'),
  )
  return decoded(response, isWebAuthnOptions)
}

export async function registerPasskey(input: {
  name: string
  credential: PublicKeyCredentialJSON
}): Promise<PasskeySummary> {
  const response = await fetchApi(
    '/api/v1/passkeys',
    jsonRequest('POST', input),
  )
  return decoded(response, isPasskeySummary)
}

export async function renamePasskey(
  passkeyId: string,
  name: string,
): Promise<PasskeySummary> {
  const response = await fetchApi(
    passkeyPath(passkeyId),
    jsonRequest('PATCH', { name }),
  )
  return decoded(response, isPasskeySummary)
}

export async function deletePasskey(passkeyId: string): Promise<void> {
  const response = await fetchApi(passkeyPath(passkeyId), {
    method: 'DELETE',
  })
  if (response.status === 401) throw new UnauthorizedError()
  await expectNoContent(response)
}

export async function startPasskeyLogin(): Promise<PasskeyRequestOptionsJSON> {
  const response = await fetchApi(
    '/api/v1/auth/passkey/options',
    jsonRequest('POST'),
  )
  if (!response.ok) throw await errorFrom(response)
  const body = await jsonBody(response)
  if (!isWebAuthnOptions(body)) {
    throw new ApiError(
      502,
      'invalid_response',
      'The server response is invalid',
    )
  }
  return body
}

/** Completes a passkey login; the backend sets the session cookie. A rejected
 * assertion fails with an ApiError of status 401. */
export async function completePasskeyLogin(
  credential: PublicKeyCredentialJSON,
): Promise<void> {
  const response = await fetchApi(
    '/api/v1/auth/passkey',
    jsonRequest('POST', credential),
  )
  await expectNoContent(response)
}
