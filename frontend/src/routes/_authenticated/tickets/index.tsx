import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { createFileRoute, Link } from '@tanstack/react-router'
import { LoaderCircle, Plus, ReceiptText, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { allCarsQueryOptions } from '#/cars/queries'
import { CarSelect } from '#/components/car-select'
import { DateRangeControl } from '#/components/date-range-control'
import { dateRangeToInstants, defaultFrom } from '#/lib/date-range'
import { type ListSearch, validateListSearch } from '#/lib/list-search'
import { ticketsListQueryOptions } from '#/tickets/queries'
import { TicketTable } from '#/tickets/ticket-table'

export const Route = createFileRoute('/_authenticated/tickets/')({
  validateSearch: validateListSearch,
  component: TicketsPage,
})

function TicketsPage() {
  const { t } = useTranslation()
  const search = Route.useSearch()
  const navigate = Route.useNavigate()
  const carId = search.carId ?? ''
  // The default is not written into the URL, so a link without a range
  // always means the last six months.
  const from = search.from ?? defaultFrom()
  // Filter changes replace the history entry instead of adding one per
  // selection or keystroke.
  const update = (change: ListSearch) =>
    navigate({
      replace: true,
      search: (previous) => ({ ...previous, ...change }),
    })
  const tickets = useInfiniteQuery(
    ticketsListQueryOptions({
      ...(carId ? { carId } : {}),
      ...dateRangeToInstants(from, search.to),
    }),
  )
  const cars = useQuery(allCarsQueryOptions)
  const items = tickets.data?.pages.flatMap((page) => page.items) ?? []
  const total = items.reduce((sum, ticket) => sum + Number(ticket.amount), 0)
  const carsById = new Map(cars.data?.items.map((car) => [car.id, car]))

  return (
    <section className="list-page" aria-labelledby="tickets-title">
      <header className="form-page-header">
        <div className="status-icon">
          <ReceiptText aria-hidden="true" size={24} />
        </div>
        <div>
          <p className="eyebrow">{t('tickets.eyebrow')}</p>
          <h1 id="tickets-title">{t('tickets.title')}</h1>
          <p>{t('tickets.description')}</p>
        </div>
        <Link
          className="button button-primary page-action"
          to="/tickets/create"
          search={carId ? { carId } : {}}
        >
          <Plus aria-hidden="true" size={18} />
          {t('tickets.add')}
        </Link>
      </header>

      <div className="list-toolbar">
        <CarSelect
          id="ticket-filter"
          label={t('tickets.filter')}
          allLabel={t('tickets.allCars')}
          value={search.carId}
          onChange={(carId) => update({ carId })}
        />
        <DateRangeControl from={from} to={search.to} onChange={update} />
      </div>

      {tickets.isPending ? (
        <output className="form-status">
          <LoaderCircle className="spin" aria-hidden="true" size={25} />
          <p>{t('tickets.loading')}</p>
        </output>
      ) : tickets.data === undefined ? (
        <div className="form-status" role="alert">
          <RefreshCw aria-hidden="true" size={25} />
          <h2>{t('tickets.loadErrorTitle')}</h2>
          <p>{t('tickets.loadError')}</p>
          <button
            className="button button-primary"
            type="button"
            onClick={() => tickets.refetch()}
          >
            <RefreshCw aria-hidden="true" size={17} />
            {t('tickets.retry')}
          </button>
        </div>
      ) : items.length === 0 ? (
        <div className="form-status">
          <p>{t('tickets.empty')}</p>
        </div>
      ) : (
        <>
          <TicketTable items={items} cars={carsById} total={total} />

          {tickets.isFetchNextPageError ? (
            <p className="field-error" role="alert">
              {t('tickets.loadMoreError')}
            </p>
          ) : null}
          {tickets.hasNextPage ? (
            <div className="list-footer">
              <button
                className="button button-secondary"
                disabled={tickets.isFetchingNextPage}
                type="button"
                onClick={() => tickets.fetchNextPage()}
              >
                {tickets.isFetchingNextPage ? (
                  <LoaderCircle className="spin" aria-hidden="true" size={17} />
                ) : null}
                {tickets.isFetchingNextPage
                  ? t('tickets.loadingMore')
                  : t('tickets.loadMore')}
              </button>
            </div>
          ) : null}
        </>
      )}
    </section>
  )
}
