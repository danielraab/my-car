import { describe, expect, it } from 'vitest'

import { isCarMuted, isDeregistered } from './deregistration'

// Local noon keeps the calendar day stable in any test time zone.
const today = new Date(2026, 9, 13, 12)

describe('isDeregistered', () => {
  it.each([
    ['2026-10-13', true],
    ['2026-10-12', true],
    ['2019-03-01', true],
    ['2026-10-14', false],
    [null, false],
  ])('deregistration date %s gives %s', (deregistrationDate, expected) => {
    expect(isDeregistered({ deregistrationDate }, today)).toBe(expected)
  })

  it('uses the local calendar day right after midnight', () => {
    const justAfterMidnight = new Date(2026, 9, 14, 0, 0, 1)
    expect(
      isDeregistered({ deregistrationDate: '2026-10-14' }, justAfterMidnight),
    ).toBe(true)
  })
})

describe('isCarMuted', () => {
  it.each([
    [true, null, false],
    [false, null, true],
    [true, '2026-10-13', true],
    [true, '2026-10-14', false],
    [false, '2026-10-14', true],
  ])(
    'isActive %s with deregistration date %s gives %s',
    (isActive, deregistrationDate, expected) => {
      expect(isCarMuted({ isActive, deregistrationDate }, today)).toBe(expected)
    },
  )
})
