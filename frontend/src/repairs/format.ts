export function formatDateTime(value: string, locale: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat(locale, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date)
}

// An instant shown as its calendar day in the viewer's zone.
export function formatInstantDate(value: string, locale: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat(locale, { dateStyle: 'medium' }).format(date)
}

function pad(value: number): string {
  return String(value).padStart(2, '0')
}

// A datetime-local input holds a wall-clock time in the viewer's zone.
export function toLocalInput(date: Date): string {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

// An unparsable value is passed through so the backend reports it.
export function fromLocalInput(value: string): string {
  const date = new Date(value)
  return value === '' || Number.isNaN(date.getTime())
    ? value
    : date.toISOString()
}
