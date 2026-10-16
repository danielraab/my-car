import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import type { Car, Repair } from '#/api/client'
import { AttachmentIndicator } from '#/attachments/attachment-indicator'
import { CarStateBadge, carRowClass } from '#/cars/car-state-badge'
import { carLabel } from '#/cars/format'
import { useUnitFormat } from '#/lib/display-units'
import { formatDateTime } from '#/repairs/format'

// The repairs table shared by /repairs and a car's Expenses tab. Leaving out
// cars hides the car column; a total adds a footer row summing the
// amounts.
export function RepairTable({
  items,
  cars,
  total,
}: {
  items: Repair[]
  cars?: Map<string, Car>
  total?: number
}) {
  const { t, i18n } = useTranslation()
  const locale = i18n.resolvedLanguage ?? 'en'
  const units = useUnitFormat()
  const showCar = cars !== undefined

  return (
    <div className="table-wrap">
      <table className="data-table">
        <caption className="sr-only">{t('repairs.tableLabel')}</caption>
        <thead>
          <tr>
            <th scope="col">{t('repairs.fields.date')}</th>
            {showCar ? <th scope="col">{t('repairs.fields.car')}</th> : null}
            <th scope="col">{t('repairs.fields.station')}</th>
            <th scope="col" className="numeric">
              {units.odometerColumn}
            </th>
            <th scope="col">{t('repairs.fields.type')}</th>
            <th scope="col" className="numeric">
              {t('repairs.fields.amount')}
            </th>
            <th scope="col">
              <span className="sr-only">{t('attachments.title')}</span>
            </th>
          </tr>
        </thead>
        <tbody>
          {items.map((repair) => (
            <tr
              key={repair.id}
              className={carRowClass(cars?.get(repair.carId))}
            >
              <td>
                <Link
                  className="table-link"
                  to="/repairs/$repairId/edit"
                  params={{ repairId: repair.id }}
                >
                  {formatDateTime(repair.date, locale)}
                </Link>
              </td>
              {showCar ? (
                <td>
                  <Link to="/cars/$carId" params={{ carId: repair.carId }}>
                    {carLabel(cars.get(repair.carId)) ??
                      t('repairs.unknownCar')}
                  </Link>
                  <CarStateBadge car={cars.get(repair.carId)} />
                </td>
              ) : null}
              <td className="wrap">{repair.station}</td>
              <td className="numeric">
                {repair.odometerReading === null
                  ? t('repairs.notRecorded')
                  : units.distance(repair.odometerReading)}
              </td>
              <td>{t(`repairs.types.${repair.type}`)}</td>
              <td className="numeric">{units.money(repair.amount)}</td>
              <td className="attachment-cell">
                <AttachmentIndicator
                  owner={{ kind: 'repairs', id: repair.id }}
                  count={repair.attachmentCount}
                />
              </td>
            </tr>
          ))}
        </tbody>
        {total === undefined ? null : (
          <tfoot>
            <tr>
              <td colSpan={showCar ? 5 : 4}>{t('repairs.total')}</td>
              <td className="numeric">{units.money(total)}</td>
              <td />
            </tr>
          </tfoot>
        )}
      </table>
    </div>
  )
}
