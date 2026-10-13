import { useQuery } from '@tanstack/react-query'
import { createFileRoute, Link } from '@tanstack/react-router'
import {
  Bike,
  Car,
  CarFront,
  ChartNoAxesCombined,
  ChevronLeft,
  ChevronRight,
  Fuel,
  LoaderCircle,
  ReceiptText,
  RefreshCw,
  Truck,
  Wrench,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { expenseKinds } from '#/api/client'
import { isDeregistered } from '#/cars/deregistration'
import { formatAmount } from '#/cars/format'
import { allCarsQueryOptions } from '#/cars/queries'
import { CarSelect } from '#/components/car-select'
import { StackedBarChart } from '#/components/stacked-bar-chart'
import { monthlyExpenses } from '#/dashboard/monthly-expenses'
import { expenseStatisticsQueryOptions } from '#/dashboard/queries'
import {
  type DashboardSearch,
  validateDashboardSearch,
} from '#/dashboard/search'
import { yearToInstants } from '#/lib/date-range'

export const Route = createFileRoute('/_authenticated/home')({
  validateSearch: validateDashboardSearch,
  component: DashboardPage,
})

// Car types are free text; the legacy app knew these three.
function carIcon(type: string) {
  const normalized = type.trim().toLowerCase()
  if (normalized === 'car') return CarFront
  if (normalized === 'truck') return Truck
  if (normalized === 'bike' || normalized === 'motorcycle') return Bike
  return Car
}

function DashboardPage() {
  const { t, i18n } = useTranslation()
  const locale = i18n.resolvedLanguage ?? 'en'
  const search = Route.useSearch()
  const navigate = Route.useNavigate()
  // The current year is not written into the URL, so a link without a year
  // always means the current one.
  const year = search.year ?? new Date().getFullYear()
  const carId = search.carId
  const shortcutSearch = carId ? { carId } : {}
  // Changes replace the history entry instead of adding one per click.
  const update = (change: DashboardSearch) =>
    navigate({
      replace: true,
      search: (previous) => ({ ...previous, ...change }),
    })
  const cars = useQuery(allCarsQueryOptions)
  // Only the cards leave deregistered cars out; "all cars" statistics keep
  // including them.
  const cardCars = cars.data?.items.filter((car) => !isDeregistered(car)) ?? []
  const statistics = useQuery(
    expenseStatisticsQueryOptions({
      ...(carId ? { carId } : {}),
      ...yearToInstants(year),
    }),
  )
  const monthLabel = new Intl.DateTimeFormat(locale, { month: 'short' })
  const narrowMonthLabel = new Intl.DateTimeFormat(locale, { month: 'narrow' })
  const firsts = Array.from(
    { length: 12 },
    (_, month) => new Date(year, month, 1),
  )
  const categories = firsts.map((first) => monthLabel.format(first))
  const narrowCategories = firsts.map((first) => narrowMonthLabel.format(first))
  const months = statistics.data
    ? monthlyExpenses(statistics.data.items, year)
    : []

  return (
    <section className="list-page" aria-labelledby="dashboard-title">
      <header className="form-page-header">
        <div className="status-icon">
          <ChartNoAxesCombined aria-hidden="true" size={24} />
        </div>
        <div>
          <p className="eyebrow">{t('dashboard.eyebrow')}</p>
          <h1 id="dashboard-title">{t('dashboard.title')}</h1>
          <p>{t('dashboard.description')}</p>
        </div>
      </header>

      {cardCars.length > 0 ? (
        <section className="dashboard-cars" aria-label={t('dashboard.cars')}>
          <ul>
            {cardCars.map((car) => {
              const Icon = carIcon(car.type)
              return (
                <li key={car.id}>
                  <Link to="/cars/$carId" params={{ carId: car.id }}>
                    <Icon aria-hidden="true" size={22} />
                    {car.make} {car.name}
                  </Link>
                </li>
              )
            })}
          </ul>
        </section>
      ) : null}

      <ul className="dashboard-shortcuts" aria-label={t('dashboard.shortcuts')}>
        <li>
          <Link
            className="button button-secondary"
            to="/refuels/create"
            search={shortcutSearch}
          >
            <Fuel aria-hidden="true" size={18} />
            {t('dashboard.addRefuel')}
          </Link>
        </li>
        <li>
          <Link
            className="button button-secondary"
            to="/repairs/create"
            search={shortcutSearch}
          >
            <Wrench aria-hidden="true" size={18} />
            {t('dashboard.addRepair')}
          </Link>
        </li>
        <li>
          <Link
            className="button button-secondary"
            to="/tickets/create"
            search={shortcutSearch}
          >
            <ReceiptText aria-hidden="true" size={18} />
            {t('dashboard.addTicket')}
          </Link>
        </li>
      </ul>

      <div className="list-toolbar">
        <CarSelect
          id="dashboard-filter"
          label={t('dashboard.filter')}
          allLabel={t('dashboard.allCars')}
          value={carId}
          onChange={(value) => update({ carId: value })}
        />
        <fieldset className="year-stepper">
          <legend className="sr-only">{t('dashboard.year')}</legend>
          <button
            className="icon-button"
            type="button"
            aria-label={t('dashboard.previousYear', { year: year - 1 })}
            title={t('dashboard.previousYear', { year: year - 1 })}
            onClick={() => update({ year: year - 1 })}
          >
            <ChevronLeft aria-hidden="true" size={20} />
          </button>
          <output aria-live="polite">{year}</output>
          <button
            className="icon-button"
            type="button"
            aria-label={t('dashboard.nextYear', { year: year + 1 })}
            title={t('dashboard.nextYear', { year: year + 1 })}
            onClick={() => update({ year: year + 1 })}
          >
            <ChevronRight aria-hidden="true" size={20} />
          </button>
        </fieldset>
      </div>

      <section className="chart-card" aria-labelledby="dashboard-chart-title">
        <h2 id="dashboard-chart-title">{t('dashboard.chart', { year })}</h2>
        <p className="chart-description">{t('dashboard.chartDescription')}</p>
        {statistics.isPending ? (
          <output className="chart-status">
            <LoaderCircle className="spin" aria-hidden="true" size={20} />
            {t('dashboard.loading')}
          </output>
        ) : statistics.isError ? (
          <div className="chart-status" role="alert">
            <p>{t('dashboard.loadError')}</p>
            <button
              className="button button-secondary"
              type="button"
              onClick={() => statistics.refetch()}
            >
              <RefreshCw aria-hidden="true" size={17} />
              {t('dashboard.retry')}
            </button>
          </div>
        ) : (
          <StackedBarChart
            categories={categories}
            narrowCategories={narrowCategories}
            series={expenseKinds.map((kind) => ({
              key: kind,
              className: `chart-${kind}`,
              label: t(`dashboard.series.${kind}`),
              values: months.map((month) => month[kind]),
            }))}
            formatValue={(value) => formatAmount(value, locale)}
            label={t('dashboard.chart', { year })}
            categoryLabel={t('dashboard.month')}
            totalLabel={t('dashboard.total')}
            empty={
              <p className="chart-empty">{t('dashboard.empty', { year })}</p>
            }
          />
        )}
      </section>
    </section>
  )
}
