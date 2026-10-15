import { useTranslation } from 'react-i18next'

import { type Car, carFuels, type Refuel, refuelFuels } from '#/api/client'
import { LineChart } from '#/components/line-chart'
import { formatValueTick } from '#/lib/chart-ticks'

// A line of the per-litre price over time for each pair of car fuel and
// refuel fuel present in items, which arrive in date order. Refuels whose
// car is not in cars are left out.
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
  const price = new Intl.NumberFormat(locale, {
    minimumFractionDigits: 3,
    maximumFractionDigits: 3,
  })

  return (
    <LineChart
      label={label}
      formatTick={(value, step) => formatValueTick(value, step, locale)}
      showLegend
      formatValue={(value) => price.format(value)}
      empty={<p className="chart-empty">{t('refuels.chartEmpty')}</p>}
      series={carFuels.flatMap((carFuel) =>
        refuelFuels.map((fuel) => ({
          key: `${carFuel}-${fuel}`,
          className: `chart-fuel-${carFuel}-${fuel}`,
          label: `${t(`cars.fuels.${carFuel}`)} · ${t(`refuels.fuels.${fuel}`)}`,
          points: items
            .filter(
              (item) =>
                item.fuel === fuel && cars.get(item.carId)?.fuel === carFuel,
            )
            .map((item) => ({
              id: item.id,
              time: Date.parse(item.date),
              value: Number(item.perLiter),
            })),
        })),
      )}
    />
  )
}
