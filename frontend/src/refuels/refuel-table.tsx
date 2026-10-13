import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import type { Car, Refuel } from '#/api/client'
import { CarStateBadge, carRowClass } from '#/cars/car-state-badge'
import { carLabel, formatAmount } from '#/cars/format'
import { formatDateTime } from '#/repairs/format'

function formatNumber(value: string, locale: string, fractionDigits: number) {
  const number = Number(value)
  if (!Number.isFinite(number)) return value
  return new Intl.NumberFormat(locale, {
    maximumFractionDigits: fractionDigits,
  }).format(number)
}

// The refuels table shared by /refuels and a car's Expenses tab. Leaving out
// cars hides the car column. Consumption comes from derived, the chart
// series, because a table page may not hold each row's predecessor. A total
// adds a footer row summing the amounts.
export function RefuelTable({
  items,
  derived,
  cars,
  total,
}: {
  items: Refuel[]
  derived: Map<string, Refuel>
  cars?: Map<string, Car>
  total?: number
}) {
  const { t, i18n } = useTranslation()
  const locale = i18n.resolvedLanguage ?? 'en'
  const showCar = cars !== undefined

  return (
    <div className="table-wrap">
      <table className="data-table">
        <caption className="sr-only">{t('refuels.tableLabel')}</caption>
        <thead>
          <tr>
            <th scope="col">{t('refuels.fields.date')}</th>
            {showCar ? <th scope="col">{t('refuels.fields.car')}</th> : null}
            <th scope="col">{t('refuels.fields.station')}</th>
            <th scope="col" className="numeric">
              {t('refuels.odometerColumn')}
            </th>
            <th scope="col">{t('refuels.fields.fuel')}</th>
            <th scope="col" className="numeric">
              {t('refuels.fields.liters')}
            </th>
            <th scope="col" className="numeric">
              {t('refuels.fields.perLiter')}
            </th>
            <th scope="col" className="numeric">
              {t('refuels.fields.amount')}
            </th>
            <th scope="col" className="numeric">
              {t('refuels.fields.consumption')}
            </th>
          </tr>
        </thead>
        <tbody>
          {items.map((refuel) => {
            const consumption = derived.get(refuel.id)?.consumption
            return (
              <tr
                key={refuel.id}
                className={carRowClass(cars?.get(refuel.carId))}
              >
                <td>
                  <Link
                    className="table-link"
                    to="/refuels/$refuelId/edit"
                    params={{ refuelId: refuel.id }}
                  >
                    {formatDateTime(refuel.date, locale)}
                  </Link>
                </td>
                {showCar ? (
                  <td>
                    <Link to="/cars/$carId" params={{ carId: refuel.carId }}>
                      {carLabel(cars.get(refuel.carId)) ??
                        t('refuels.unknownCar')}
                    </Link>
                    <CarStateBadge car={cars.get(refuel.carId)} />
                  </td>
                ) : null}
                <td className="wrap">{refuel.station}</td>
                <td className="numeric">
                  {refuel.odometerReading === null
                    ? t('refuels.notRecorded')
                    : t('refuels.odometerValue', {
                        value: new Intl.NumberFormat(locale).format(
                          refuel.odometerReading,
                        ),
                      })}
                </td>
                <td>{t(`refuels.fuels.${refuel.fuel}`)}</td>
                <td className="numeric">
                  {t('refuels.litersValue', {
                    value: formatNumber(refuel.liters, locale, 2),
                  })}
                </td>
                <td className="numeric">
                  {formatNumber(refuel.perLiter, locale, 3)}
                </td>
                <td className="numeric">
                  {formatAmount(refuel.amount, locale)}
                </td>
                <td className="numeric">
                  {consumption == null
                    ? '—'
                    : t('refuels.consumptionValue', {
                        value: formatNumber(consumption, locale, 2),
                      })}
                </td>
              </tr>
            )
          })}
        </tbody>
        {total === undefined ? null : (
          <tfoot>
            <tr>
              <td colSpan={showCar ? 7 : 6}>{t('refuels.total')}</td>
              <td className="numeric">{formatAmount(String(total), locale)}</td>
              <td />
            </tr>
          </tfoot>
        )}
      </table>
    </div>
  )
}
