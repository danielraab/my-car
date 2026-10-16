import { Tab, TabGroup, TabList, TabPanel, TabPanels } from '@headlessui/react'
import { useQuery } from '@tanstack/react-query'
import { createFileRoute, Link } from '@tanstack/react-router'
import { ArrowLeft, CarFront, Pencil } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import type { Car } from '#/api/client'
import { AttachmentOverview } from '#/attachments/attachment-indicator'
import { CarConsumption } from '#/cars/car-consumption'
import { CarExpenses } from '#/cars/car-expenses'
import { CarStatus } from '#/cars/car-status'
import { formatDate } from '#/cars/format'
import { carQueryOptions } from '#/cars/queries'
import { DateRangeControl } from '#/components/date-range-control'
import {
  dateRangeToInstants,
  defaultFrom,
  isCalendarDate,
} from '#/lib/date-range'
import { useUnitFormat } from '#/lib/display-units'

const tabs = ['details', 'expenses', 'consumption'] as const
type CarTab = (typeof tabs)[number]

type CarSearch = { tab?: CarTab; from?: string; to?: string }

// Unknown tabs and malformed dates fall back to the defaults rather than
// breaking the screen. Every key is set, even to undefined, because the
// route's search is merged over the unvalidated search of its parents.
function validateSearch(search: Record<string, unknown>): CarSearch {
  return {
    tab: tabs.find((candidate) => candidate === search.tab),
    from: isCalendarDate(search.from) ? search.from : undefined,
    to: isCalendarDate(search.to) ? search.to : undefined,
  }
}

export const Route = createFileRoute('/_authenticated/cars/$carId/')({
  validateSearch,
  component: CarDetailPage,
})

function CarDetailPage() {
  const { carId } = Route.useParams()
  const car = useQuery(carQueryOptions(carId))

  return car.data ? (
    <CarView car={car.data} />
  ) : (
    <section className="form-page">
      <CarStatus error={car.error} retry={() => car.refetch()} />
    </section>
  )
}

function CarView({ car }: { car: Car }) {
  const { t } = useTranslation()
  const search = Route.useSearch()
  const navigate = Route.useNavigate()
  const tab = search.tab ?? 'details'
  // The default is not written into the URL, so a link without a range
  // always means the last six months.
  const from = search.from ?? defaultFrom()
  const filter = { carId: car.id, ...dateRangeToInstants(from, search.to) }

  return (
    <section className="list-page car-page" aria-labelledby="car-title">
      <header className="form-page-header">
        <div className="status-icon">
          <CarFront aria-hidden="true" size={24} />
        </div>
        <div>
          <p className="eyebrow">{t('cars.detail.eyebrow')}</p>
          <h1 id="car-title">{car.name}</h1>
          <p>
            {car.licensePlate === null
              ? car.make
              : `${car.make} · ${car.licensePlate}`}
          </p>
        </div>
        <Link
          className="button button-primary page-action"
          to="/cars/$carId/edit"
          params={{ carId: car.id }}
        >
          <Pencil aria-hidden="true" size={17} />
          {t('cars.detail.edit')}
        </Link>
      </header>

      <TabGroup
        selectedIndex={tabs.indexOf(tab)}
        onChange={(index) =>
          navigate({
            search: (previous) => ({ ...previous, tab: tabs[index] }),
          })
        }
      >
        <div className="car-tabs-bar">
          <TabList className="car-tabs" aria-label={t('cars.detail.tabsLabel')}>
            {tabs.map((name) => (
              <Tab key={name} className="car-tab">
                {t(`cars.detail.tabs.${name}`)}
              </Tab>
            ))}
          </TabList>
          {tab === 'details' ? null : (
            <DateRangeControl
              from={from}
              to={search.to}
              onChange={(range) =>
                // Typing a date replaces the history entry instead of adding
                // one per keystroke. A cleared "from" goes back to the
                // default; a cleared "to" leaves the range open.
                navigate({
                  replace: true,
                  search: (previous) => ({ ...previous, ...range }),
                })
              }
            />
          )}
        </div>
        <TabPanels>
          <TabPanel>
            <CarDetails car={car} />
          </TabPanel>
          <TabPanel>
            <CarExpenses filter={filter} carFuel={car.fuel} />
          </TabPanel>
          <TabPanel>
            <CarConsumption filter={filter} carFuel={car.fuel} />
          </TabPanel>
        </TabPanels>
      </TabGroup>
    </section>
  )
}

function CarDetails({ car }: { car: Car }) {
  const { t, i18n } = useTranslation()
  const locale = i18n.resolvedLanguage ?? 'en'
  const units = useUnitFormat()
  const notRecorded = t('cars.notRecorded')
  const details: [string, string][] = [
    ['type', car.type],
    ['make', car.make],
    ['name', car.name],
    ['fuel', t(`cars.fuels.${car.fuel}`)],
    [
      'firstRegistration',
      car.firstRegistration === null
        ? notRecorded
        : formatDate(car.firstRegistration, locale),
    ],
    ['licensePlate', car.licensePlate ?? notRecorded],
    ['fin', car.fin ?? notRecorded],
    [
      'purchaseDate',
      car.purchaseDate === null
        ? notRecorded
        : formatDate(car.purchaseDate, locale),
    ],
    [
      'purchasePrice',
      car.purchasePrice === null ? notRecorded : units.money(car.purchasePrice),
    ],
    [
      'isActive',
      t(car.isActive ? 'cars.states.active' : 'cars.states.inactive'),
    ],
    [
      'deregistrationDate',
      car.deregistrationDate === null
        ? notRecorded
        : formatDate(car.deregistrationDate, locale),
    ],
  ]

  return (
    <div className="form-card">
      <dl className="detail-list">
        {details.map(([field, value]) => (
          <div key={field}>
            <dt>{t(`cars.fields.${field}`)}</dt>
            <dd>{value}</dd>
          </div>
        ))}
        <div className="detail-wide">
          <dt>{t('cars.fields.information')}</dt>
          <dd className="preserve-lines">{car.information ?? notRecorded}</dd>
        </div>
      </dl>

      <AttachmentOverview owner={{ kind: 'cars', id: car.id }} />

      <div className="form-actions">
        <Link className="button button-secondary" to="/cars">
          <ArrowLeft aria-hidden="true" size={17} />
          {t('cars.detail.back')}
        </Link>
      </div>
    </div>
  )
}
