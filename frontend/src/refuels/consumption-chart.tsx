import { useTranslation } from 'react-i18next'

import type { CarFuel, Refuel } from '#/api/client'
import { LineChart } from '#/components/line-chart'
import { formatValueTick } from '#/lib/chart-ticks'
import { useUnitFormat } from '#/lib/display-units'

// A single line of one car's consumption, in its quantity unit (kWh for an
// electric car, the account volume unit otherwise) per 100 of the account
// distance unit, over time. Refuels without a consumption (a car's first, or
// one without odometer readings) are left out.
export function ConsumptionChart({
  items,
  label,
  carFuel,
}: {
  items: Refuel[]
  label: string
  carFuel?: CarFuel
}) {
  const { t, i18n } = useTranslation()
  const locale = i18n.resolvedLanguage ?? 'en'
  const units = useUnitFormat()

  return (
    <LineChart
      label={label}
      formatTick={(value, step) => formatValueTick(value, step, locale)}
      formatValue={(value) => units.quantity(carFuel).consumption(value)}
      empty={
        <p className="chart-empty">
          {t('refuels.consumptionChartEmpty', {
            odometer: units.odometerColumn,
          })}
        </p>
      }
      series={[
        {
          key: 'consumption',
          className: 'chart-consumption',
          label,
          points: items.flatMap((item) =>
            item.consumption === null
              ? []
              : [
                  {
                    id: item.id,
                    time: Date.parse(item.date),
                    value: Number(item.consumption),
                  },
                ],
          ),
        },
      ]}
    />
  )
}
