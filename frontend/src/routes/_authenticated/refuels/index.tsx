import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { createFileRoute, Link } from '@tanstack/react-router'
import { Fuel, LoaderCircle, Plus, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { allCarsQueryOptions } from '#/cars/queries'
import { CarSelect } from '#/components/car-select'
import { DateRangeControl } from '#/components/date-range-control'
import { dateRangeToInstants, defaultFrom } from '#/lib/date-range'
import { type ListSearch, validateListSearch } from '#/lib/list-search'
import { FuelPriceChart } from '#/refuels/fuel-price-chart'
import {
  refuelChartQueryOptions,
  refuelsListQueryOptions,
} from '#/refuels/queries'
import { RefuelTable } from '#/refuels/refuel-table'

export const Route = createFileRoute('/_authenticated/refuels/')({
  validateSearch: validateListSearch,
  component: RefuelsPage,
})

function RefuelsPage() {
  const { t } = useTranslation()
  const search = Route.useSearch()
  const navigate = Route.useNavigate()
  const carId = search.carId ?? ''
  // The default is not written into the URL, so a link without a range
  // always means the last six months.
  const from = search.from ?? defaultFrom()
  const filter = {
    ...(carId ? { carId } : {}),
    ...dateRangeToInstants(from, search.to),
  }
  // Filter changes replace the history entry instead of adding one per
  // selection or keystroke.
  const update = (change: ListSearch) =>
    navigate({
      replace: true,
      search: (previous) => ({ ...previous, ...change }),
    })
  const refuels = useInfiniteQuery(refuelsListQueryOptions(filter))
  const chart = useQuery(refuelChartQueryOptions(filter))
  const cars = useQuery(allCarsQueryOptions)
  const items = refuels.data?.pages.flatMap((page) => page.items) ?? []
  const total = items.reduce((sum, refuel) => sum + Number(refuel.amount), 0)
  const carsById = new Map(cars.data?.items.map((car) => [car.id, car]))
  // Consumption needs each refuel's predecessor, which only the complete
  // chart series is guaranteed to hold.
  const derived = new Map(
    chart.data?.items.map((refuel) => [refuel.id, refuel]),
  )

  return (
    <section className="list-page" aria-labelledby="refuels-title">
      <header className="form-page-header">
        <div className="status-icon">
          <Fuel aria-hidden="true" size={24} />
        </div>
        <div>
          <p className="eyebrow">{t('refuels.eyebrow')}</p>
          <h1 id="refuels-title">{t('refuels.title')}</h1>
          <p>{t('refuels.description')}</p>
        </div>
        <Link
          className="button button-primary page-action"
          to="/refuels/create"
          search={carId ? { carId } : {}}
        >
          <Plus aria-hidden="true" size={18} />
          {t('refuels.add')}
        </Link>
      </header>

      <div className="list-toolbar">
        <CarSelect
          id="refuel-filter"
          label={t('refuels.filter')}
          allLabel={t('refuels.allCars')}
          value={search.carId}
          onChange={(carId) => update({ carId })}
        />
        <DateRangeControl from={from} to={search.to} onChange={update} />
      </div>

      <section className="chart-card" aria-labelledby="refuels-chart-title">
        <h2 id="refuels-chart-title">{t('refuels.chart')}</h2>
        <p className="chart-description">{t('refuels.chartDescription')}</p>
        {chart.isPending ? (
          <output className="chart-status">
            <LoaderCircle className="spin" aria-hidden="true" size={20} />
            {t('refuels.loading')}
          </output>
        ) : chart.isError ? (
          <div className="chart-status" role="alert">
            <p>{t('refuels.chartLoadError')}</p>
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
          <FuelPriceChart items={chart.data.items} label={t('refuels.chart')} />
        )}
      </section>

      {refuels.isPending ? (
        <output className="form-status">
          <LoaderCircle className="spin" aria-hidden="true" size={25} />
          <p>{t('refuels.loading')}</p>
        </output>
      ) : refuels.data === undefined ? (
        <div className="form-status" role="alert">
          <RefreshCw aria-hidden="true" size={25} />
          <h2>{t('refuels.loadErrorTitle')}</h2>
          <p>{t('refuels.loadError')}</p>
          <button
            className="button button-primary"
            type="button"
            onClick={() => refuels.refetch()}
          >
            <RefreshCw aria-hidden="true" size={17} />
            {t('refuels.retry')}
          </button>
        </div>
      ) : items.length === 0 ? (
        <div className="form-status">
          <p>{t('refuels.empty')}</p>
        </div>
      ) : (
        <>
          <RefuelTable
            items={items}
            derived={derived}
            cars={carsById}
            total={total}
          />

          {refuels.isFetchNextPageError ? (
            <p className="field-error" role="alert">
              {t('refuels.loadMoreError')}
            </p>
          ) : null}
          {refuels.hasNextPage ? (
            <div className="list-footer">
              <button
                className="button button-secondary"
                disabled={refuels.isFetchingNextPage}
                type="button"
                onClick={() => refuels.fetchNextPage()}
              >
                {refuels.isFetchingNextPage ? (
                  <LoaderCircle className="spin" aria-hidden="true" size={17} />
                ) : null}
                {refuels.isFetchingNextPage
                  ? t('refuels.loadingMore')
                  : t('refuels.loadMore')}
              </button>
            </div>
          ) : null}
        </>
      )}
    </section>
  )
}
