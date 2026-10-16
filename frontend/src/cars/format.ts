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

// Formats an amount in the account currency. The fraction digits are fixed
// rather than the currency's own (JPY has none), so a stored value is never
// rounded: the currency only labels the number, it is never converted.
export function formatMoney(
  value: string | number,
  locale: string,
  currency: string,
  fractionDigits = 2,
): string {
  const amount = Number(value)
  if (!Number.isFinite(amount)) return String(value)
  return new Intl.NumberFormat(locale, {
    style: 'currency',
    currency,
    minimumFractionDigits: fractionDigits,
    maximumFractionDigits: fractionDigits,
  }).format(amount)
}

// The symbol the active language uses for currency, such as "€" or "CHF".
export function currencySymbol(locale: string, currency: string): string {
  return (
    new Intl.NumberFormat(locale, { style: 'currency', currency })
      .formatToParts(0)
      .find((part) => part.type === 'currency')?.value ?? currency
  )
}

// Formats a decimal string with at most fractionDigits digits; anything that
// is not a number is shown unchanged.
export function formatDecimal(
  value: string | number,
  locale: string,
  fractionDigits: number,
): string {
  const number = Number(value)
  if (!Number.isFinite(number)) return String(value)
  return new Intl.NumberFormat(locale, {
    maximumFractionDigits: fractionDigits,
  }).format(number)
}

// How a car is named where it is referenced, such as a table's car column.
export function carLabel(
  car: { make: string; name: string } | undefined,
): string | undefined {
  return car === undefined ? undefined : `${car.make} ${car.name}`
}
