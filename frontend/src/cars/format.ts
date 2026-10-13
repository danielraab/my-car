// Full-dates carry no time zone, so they are formatted in UTC to keep the
// calendar day from shifting in the viewer's zone.
export function formatDate(value: string, locale: string): string {
  const date = new Date(`${value}T00:00:00Z`)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat(locale, {
    dateStyle: 'medium',
    timeZone: 'UTC',
  }).format(date)
}

export function formatAmount(value: string, locale: string): string {
  const amount = Number(value)
  if (!Number.isFinite(amount)) return value
  return new Intl.NumberFormat(locale, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(amount)
}

// How a car is named where it is referenced, such as a table's car column.
export function carLabel(
  car: { make: string; name: string } | undefined,
): string | undefined {
  return car === undefined ? undefined : `${car.make} ${car.name}`
}
