import { useTranslation } from 'react-i18next'

import {
  type Car,
  type CarFuel,
  carFuels,
  type Refuel,
  refuelFuelsFor,
} from '#/api/client'
import { LineChart } from '#/components/line-chart'
import { formatValueTick } from '#/lib/chart-ticks'
import { useUnitFormat } from '#/lib/display-units'

// A line of the unit price over time for each pair of car fuel and refuel
// subtype present in items, which arrive in date order. Refuels whose car is
// not in cars are left out. Charges of electric cars are priced per kWh, so
// their lines use a right value axis beside fuel lines priced per volume.
export function FuelPriceChart({
  items,
  cars,
  label,
}: {
  items: Refuel[]
  cars: Map<string, Car>
  label: string
}) {
  const { t, i18n } = useTranslation()
  const locale = i18n.resolvedLanguage ?? 'en'
  const units = useUnitFormat()
  const carFuelOf = new Map<string, CarFuel>()
  const axisLabel = (carFuel: CarFuel) =>
    t('units.perUnit', {
      price: units.currencySymbol,
      unit: units.quantity(carFuel).symbol,
    })

  return (
    <LineChart
      label={label}
      formatTick={(value, step) => formatValueTick(value, step, locale)}
      showLegend
      formatValue={(value, line) =>
        units.quantity(carFuelOf.get(line.key)).unitPrice(value)
      }
      axisLabels={{ left: axisLabel('diesel'), right: axisLabel('electric') }}
      empty={<p className="chart-empty">{t('refuels.chartEmpty')}</p>}
      series={carFuels.flatMap((carFuel) =>
        refuelFuelsFor(carFuel).map((fuel) => {
          const key = `${carFuel}-${fuel}`
          carFuelOf.set(key, carFuel)
          return {
            key,
            className: `chart-fuel-${carFuel}-${fuel}`,
            label: `${t(`cars.fuels.${carFuel}`)} · ${t(`refuels.fuels.${fuel}`)}`,
            axis:
              carFuel === 'electric' ? ('right' as const) : ('left' as const),
            points: items
              .filter(
                (item) =>
                  item.fuel === fuel && cars.get(item.carId)?.fuel === carFuel,
              )
              .map((item) => ({
                id: item.id,
                time: Date.parse(item.date),
                value: Number(item.unitPrice),
              })),
          }
        }),
      )}
    />
  )
}
