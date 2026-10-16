import { describe, expect, it } from 'vitest'

import { currencySymbol, formatDecimal, formatMoney } from './format'

// Intl separates a symbol from the number with a no-break space.
const nbsp = (text: string) => text.replace(/ /g, ' ')

describe('formatMoney', () => {
  it('places the euro symbol the way each language does', () => {
    expect(nbsp(formatMoney('1234.5', 'de', 'EUR'))).toBe('1.234,50 €')
    expect(formatMoney('1234.5', 'en', 'EUR')).toBe('€1,234.50')
  })

  it('keeps stored decimals for a currency without a minor unit', () => {
    expect(formatMoney('120.50', 'en', 'JPY')).toBe('¥120.50')
  })

  it('shows per-volume prices with three fraction digits', () => {
    expect(nbsp(formatMoney('1.659', 'de', 'EUR', 3))).toBe('1,659 €')
  })

  it('shows a value that is not a number unchanged', () => {
    expect(formatMoney('abc', 'en', 'EUR')).toBe('abc')
  })
})

describe('currencySymbol', () => {
  it('names the currency the way the language does', () => {
    expect(currencySymbol('en', 'USD')).toBe('$')
    expect(currencySymbol('de', 'EUR')).toBe('€')
  })
})

describe('formatDecimal', () => {
  it('limits the fraction digits without padding', () => {
    expect(formatDecimal('42.5', 'en', 2)).toBe('42.5')
    expect(formatDecimal('6.666', 'de', 2)).toBe('6,67')
    expect(formatDecimal('n/a', 'en', 2)).toBe('n/a')
  })
})
