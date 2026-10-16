import { createContext, useContext, useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import type { CarFuel, Profile } from '#/api/client'
import { currencySymbol, formatDecimal, formatMoney } from '#/cars/format'

// The account's display settings. They only label values: nothing stored or
// computed by the backend is converted when they change.
export type DisplayUnits = Pick<
  Profile,
  'currency' | 'volumeUnit' | 'distanceUnit'
>

export const defaultDisplayUnits: DisplayUnits = {
  currency: 'EUR',
  volumeUnit: 'liter',
  distanceUnit: 'kilometer',
}

// Provided by the authenticated layout from the cached session, so a saved
// profile change reaches every screen without a reload. Without a provider
// (public routes, isolated component tests) the defaults apply.
export const DisplayUnitsContext =
  createContext<DisplayUnits>(defaultDisplayUnits)

export function displayUnitsOf(profile: DisplayUnits): DisplayUnits {
  return {
    currency: profile.currency,
    volumeUnit: profile.volumeUnit,
    distanceUnit: profile.distanceUnit,
  }
}

// Formatters and labels for amounts, volumes, distances and consumption in
// the account's currency and units and the active language.
export function useUnitFormat() {
  const units = useContext(DisplayUnitsContext)
  const { t, i18n } = useTranslation()
  const locale = i18n.resolvedLanguage ?? 'en'

  return useMemo(() => {
    const volume = t(`units.volume.${units.volumeUnit}`)
    const distance = t(`units.distance.${units.distanceUnit}`)
    const withUnit = (value: string, unit: string) =>
      t('units.value', { value, unit })
    return {
      ...units,
      locale,
      currencySymbol: currencySymbol(locale, units.currency),
      distanceSymbol: distance,
      // "litre" / "gallon", inside running text.
      volumeSingular: t(`units.volumeSingular.${units.volumeUnit}`),
      // "Odometer" / "Kilometerstand" (German names the unit), for columns.
      odometerColumn: t(`units.odometerColumn.${units.distanceUnit}`),
      // "Odometer reading" / "Kilometerstand", for form fields.
      odometerReadingName: t(`units.odometerReading.${units.distanceUnit}`),
      money: (value: string | number) =>
        formatMoney(value, locale, units.currency),
      distance: (value: number) =>
        withUnit(new Intl.NumberFormat(locale).format(value), distance),
      // Units of a refuel's quantity, which follow its car's fuel: kWh for
      // an electric car, the account volume unit otherwise. Every value is
      // labelled with its unit, so rows of different cars can share a
      // column.
      quantity: (carFuel: CarFuel | undefined) => {
        const electric = carFuel === 'electric'
        const symbol = electric ? t('units.energy') : volume
        const consumptionSymbol = t('units.consumption', {
          volume: symbol,
          distance,
        })
        return {
          symbol,
          consumptionSymbol,
          // "Litres" / "Gallons" / "Kilowatt-hours", for field labels.
          name: electric
            ? t('units.energyName')
            : t(`units.volumeName.${units.volumeUnit}`),
          value: (value: string | number) =>
            withUnit(formatDecimal(value, locale, 2), symbol),
          unitPrice: (value: string | number) =>
            t('units.perUnit', {
              price: formatMoney(value, locale, units.currency, 3),
              unit: symbol,
            }),
          consumption: (value: string | number) =>
            withUnit(formatDecimal(value, locale, 2), consumptionSymbol),
        }
      },
    }
  }, [units, locale, t])
}
