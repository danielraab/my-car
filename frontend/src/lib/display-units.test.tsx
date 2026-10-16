import { renderHook } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, describe, expect, it } from 'vitest'

import { defaultRefuelFuel, refuelFuelsFor } from '#/api/client'
import { i18n } from '#/i18n'

import {
  type DisplayUnits,
  DisplayUnitsContext,
  useUnitFormat,
} from './display-units'

function unitsFor(value: DisplayUnits) {
  const wrapper = ({ children }: { children: ReactNode }) => (
    <DisplayUnitsContext.Provider value={value}>
      {children}
    </DisplayUnitsContext.Provider>
  )
  return renderHook(() => useUnitFormat(), { wrapper }).result.current
}

describe('quantity units', () => {
  afterEach(async () => {
    await i18n.changeLanguage('en')
  })

  it('labels a non-electric car in the account volume unit', () => {
    const units = unitsFor({
      currency: 'EUR',
      volumeUnit: 'gallon',
      distanceUnit: 'mile',
    })
    const quantity = units.quantity('diesel')
    expect(quantity.symbol).toBe('gal')
    expect(quantity.name).toBe('Gallons')
    expect(quantity.value('42.5')).toBe('42.5 gal')
    expect(quantity.unitPrice('1.659')).toBe('€1.659/gal')
    expect(quantity.consumption('7.12')).toBe('7.12 gal/100 mi')
  })

  it('labels an electric car in kWh whatever the volume unit', () => {
    const units = unitsFor({
      currency: 'EUR',
      volumeUnit: 'gallon',
      distanceUnit: 'mile',
    })
    const quantity = units.quantity('electric')
    expect(quantity.symbol).toBe('kWh')
    expect(quantity.name).toBe('Kilowatt-hours')
    expect(quantity.value('40')).toBe('40 kWh')
    expect(quantity.unitPrice('0.35')).toBe('€0.350/kWh')
    expect(quantity.consumption('28.1')).toBe('28.1 kWh/100 mi')
  })

  it('uses German names', async () => {
    await i18n.changeLanguage('de')
    const units = unitsFor({
      currency: 'EUR',
      volumeUnit: 'liter',
      distanceUnit: 'kilometer',
    })
    expect(units.quantity('electric').name).toBe('Kilowattstunden')
    expect(units.quantity('electric').unitPrice('0.35')).toBe(
      '0,350\u00a0€/kWh',
    )
    expect(units.quantity('gasoline').consumption('6.5')).toBe('6,5 l/100 km')
  })
})

describe('refuel subtypes per car fuel', () => {
  it('offers charging subtypes for an electric car', () => {
    expect(refuelFuelsFor('electric')).toEqual(['home', 'ac', 'dc', 'other'])
    expect(defaultRefuelFuel('electric')).toBe('ac')
  })

  it('offers fuel grades for every other car', () => {
    for (const fuel of ['diesel', 'gasoline', 'other'] as const) {
      expect(refuelFuelsFor(fuel)).toEqual(['normal', 'special', 'other'])
      expect(defaultRefuelFuel(fuel)).toBe('normal')
    }
  })
})
