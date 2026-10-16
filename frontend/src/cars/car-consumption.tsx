import { useQuery } from '@tanstack/react-query'
import { LoaderCircle, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import type { CarFuel, ExpenseFilter } from '#/api/client'
import { useUnitFormat } from '#/lib/display-units'
import { ConsumptionChart } from '#/refuels/consumption-chart'
import { refuelChartQueryOptions } from '#/refuels/queries'

// The car's consumption over the filter's range, from the complete chart
// series rather than table pages.
export function CarConsumption({
  filter,
  carFuel,
}: {
  filter: ExpenseFilter & { carId: string }
  carFuel: CarFuel
}) {
  const { t } = useTranslation()
  const units = useUnitFormat()
  const chart = useQuery(refuelChartQueryOptions(filter))

  return (
    <section className="chart-card" aria-labelledby="car-consumption-title">
      <h2 id="car-consumption-title">{t('cars.detail.consumption.title')}</h2>
      <p className="chart-description">
        {t('cars.detail.consumption.description', {
          unit: units.quantity(carFuel).consumptionSymbol,
          odometer: units.odometerColumn,
        })}
      </p>
      {chart.isPending ? (
        <output className="chart-status">
          <LoaderCircle className="spin" aria-hidden="true" size={20} />
          {t('refuels.loading')}
        </output>
      ) : chart.isError ? (
        <div className="chart-status" role="alert">
          <p>{t('cars.detail.consumption.loadError')}</p>
          <button
            className="button button-secondary"
            type="button"
            onClick={() => chart.refetch()}
          >
            <RefreshCw aria-hidden="true" size={17} />
            {t('refuels.retry')}
          </button>
        </div>
      ) : (
        <ConsumptionChart
          items={chart.data.items}
          label={t('cars.detail.consumption.title')}
          carFuel={carFuel}
        />
      )}
    </section>
  )
}
