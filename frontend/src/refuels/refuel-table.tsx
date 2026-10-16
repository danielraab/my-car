import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import type { Car, CarFuel, Refuel } from '#/api/client'
import { AttachmentIndicator } from '#/attachments/attachment-indicator'
import { CarStateBadge, carRowClass } from '#/cars/car-state-badge'
import { carLabel } from '#/cars/format'
import { useUnitFormat } from '#/lib/display-units'
import { formatInstantDate } from '#/repairs/format'

// The refuels table shared by /refuels and a car's Expenses tab. Leaving out
// cars hides the car column; carFuel then gives the one car's fuel. Each
// row's quantity, unit price and consumption carry the unit of its car (kWh
// for an electric car), so the headers name none. Consumption comes from
// derived, the chart series, because a table page may not hold each row's
// predecessor. A total adds a footer row summing the amounts.
export function RefuelTable({
  items,
  derived,
  cars,
  carFuel,
  total,
}: {
  items: Refuel[]
  derived: Map<string, Refuel>
  cars?: Map<string, Car>
  carFuel?: CarFuel
  total?: number
}) {
  const { t, i18n } = useTranslation()
  const locale = i18n.resolvedLanguage ?? 'en'
  const units = useUnitFormat()
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
              {units.odometerColumn}
            </th>
            <th scope="col" className="numeric">
              {t('refuels.fields.quantity')}
            </th>
            <th scope="col" className="numeric">
              {t('refuels.fields.unitPrice')}
            </th>
            <th scope="col" className="numeric">
              {t('refuels.fields.amount')}
            </th>
            <th scope="col" className="numeric">
              {t('refuels.fields.consumption')}
            </th>
            <th scope="col">
              <span className="sr-only">{t('attachments.title')}</span>
            </th>
          </tr>
        </thead>
        <tbody>
          {items.map((refuel) => {
            const consumption = derived.get(refuel.id)?.consumption
            const quantity = units.quantity(
              cars?.get(refuel.carId)?.fuel ?? carFuel,
            )
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
                    {formatInstantDate(refuel.date, locale)}
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
                    : units.distance(refuel.odometerReading)}
                </td>
                <td className="numeric">{quantity.value(refuel.quantity)}</td>
                <td className="numeric">
                  {quantity.unitPrice(refuel.unitPrice)}
                </td>
                <td className="numeric">{units.money(refuel.amount)}</td>
                <td className="numeric">
                  {consumption == null
                    ? '—'
                    : quantity.consumption(consumption)}
                </td>
                <td className="attachment-cell">
                  <AttachmentIndicator
                    owner={{ kind: 'refuels', id: refuel.id }}
                    count={refuel.attachmentCount}
                  />
                </td>
              </tr>
            )
          })}
        </tbody>
        {total === undefined ? null : (
          <tfoot>
            <tr>
              <td colSpan={showCar ? 6 : 5}>{t('refuels.total')}</td>
              <td className="numeric">{units.money(total)}</td>
              <td />
              <td />
            </tr>
          </tfoot>
        )}
      </table>
    </div>
  )
}
