import { useInfiniteQuery } from '@tanstack/react-query'
import { createFileRoute, Link } from '@tanstack/react-router'
import { CarFront, LoaderCircle, Plus, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { AttachmentIndicator } from '#/attachments/attachment-indicator'
import { CarStateBadge } from '#/cars/car-state-badge'
import { isCarMuted } from '#/cars/deregistration'
import { formatDate } from '#/cars/format'
import { carsListQueryOptions } from '#/cars/queries'
import { useUnitFormat } from '#/lib/display-units'

export const Route = createFileRoute('/_authenticated/cars/')({
  component: CarsPage,
})

function CarsPage() {
  const { t, i18n } = useTranslation()
  const locale = i18n.resolvedLanguage ?? 'en'
  const units = useUnitFormat()
  const cars = useInfiniteQuery(carsListQueryOptions)
  const items = cars.data?.pages.flatMap((page) => page.items) ?? []

  return (
    <section className="list-page" aria-labelledby="cars-title">
      <header className="form-page-header">
        <div className="status-icon">
          <CarFront aria-hidden="true" size={24} />
        </div>
        <div>
          <p className="eyebrow">{t('cars.eyebrow')}</p>
          <h1 id="cars-title">{t('cars.title')}</h1>
          <p>{t('cars.description')}</p>
        </div>
        <Link className="button button-primary page-action" to="/cars/create">
          <Plus aria-hidden="true" size={18} />
          {t('cars.add')}
        </Link>
      </header>

      {cars.isPending ? (
        <output className="form-status">
          <LoaderCircle className="spin" aria-hidden="true" size={25} />
          <p>{t('cars.loading')}</p>
        </output>
      ) : cars.data === undefined ? (
        <div className="form-status" role="alert">
          <RefreshCw aria-hidden="true" size={25} />
          <h2>{t('cars.loadErrorTitle')}</h2>
          <p>{t('cars.loadError')}</p>
          <button
            className="button button-primary"
            type="button"
            onClick={() => cars.refetch()}
          >
            <RefreshCw aria-hidden="true" size={17} />
            {t('cars.retry')}
          </button>
        </div>
      ) : items.length === 0 ? (
        <div className="form-status">
          <p>{t('cars.empty')}</p>
        </div>
      ) : (
        <>
          <div className="table-wrap">
            <table className="data-table">
              <caption className="sr-only">{t('cars.tableLabel')}</caption>
              <thead>
                <tr>
                  <th scope="col">{t('cars.fields.type')}</th>
                  <th scope="col">{t('cars.fields.make')}</th>
                  <th scope="col">{t('cars.fields.name')}</th>
                  <th scope="col">{t('cars.fields.fuel')}</th>
                  <th scope="col">{t('cars.fields.firstRegistration')}</th>
                  <th scope="col">{t('cars.fields.licensePlate')}</th>
                  <th scope="col" className="numeric">
                    {t('cars.fields.purchasePrice')}
                  </th>
                  <th scope="col">
                    <span className="sr-only">{t('attachments.title')}</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {items.map((car) => (
                  <tr
                    key={car.id}
                    className={isCarMuted(car) ? 'row-muted' : undefined}
                  >
                    <td>{car.type}</td>
                    <td>{car.make}</td>
                    <td>
                      <Link
                        className="table-link"
                        to="/cars/$carId"
                        params={{ carId: car.id }}
                      >
                        {car.name}
                      </Link>
                      <CarStateBadge car={car} />
                    </td>
                    <td>{t(`cars.fuels.${car.fuel}`)}</td>
                    <td>
                      {car.firstRegistration === null
                        ? t('cars.notRecorded')
                        : formatDate(car.firstRegistration, locale)}
                    </td>
                    <td>{car.licensePlate ?? t('cars.notRecorded')}</td>
                    <td className="numeric">
                      {car.purchasePrice === null
                        ? t('cars.notRecorded')
                        : units.money(car.purchasePrice)}
                    </td>
                    <td className="attachment-cell">
                      <AttachmentIndicator
                        owner={{ kind: 'cars', id: car.id }}
                        count={car.attachmentCount}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {cars.isFetchNextPageError ? (
            <p className="field-error" role="alert">
              {t('cars.loadMoreError')}
            </p>
          ) : null}
          {cars.hasNextPage ? (
            <div className="list-footer">
              <button
                className="button button-secondary"
                disabled={cars.isFetchingNextPage}
                type="button"
                onClick={() => cars.fetchNextPage()}
              >
                {cars.isFetchingNextPage ? (
                  <LoaderCircle className="spin" aria-hidden="true" size={17} />
                ) : null}
                {cars.isFetchingNextPage
                  ? t('cars.loadingMore')
                  : t('cars.loadMore')}
              </button>
            </div>
          ) : null}
        </>
      )}
    </section>
  )
}
