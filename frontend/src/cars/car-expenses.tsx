import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import type { CarFuel, ExpenseFilter } from '#/api/client'
import {
  refuelChartQueryOptions,
  refuelsListQueryOptions,
} from '#/refuels/queries'
import { RefuelTable } from '#/refuels/refuel-table'
import { repairsListQueryOptions } from '#/repairs/queries'
import { RepairTable } from '#/repairs/repair-table'
import { ticketsListQueryOptions } from '#/tickets/queries'
import { TicketTable } from '#/tickets/ticket-table'

import { ExpenseSection } from './expense-section'

// The car's refuels, repairs and tickets within the filter's range.
export function CarExpenses({
  filter,
  carFuel,
}: {
  filter: ExpenseFilter & { carId: string }
  carFuel: CarFuel
}) {
  const { t } = useTranslation()
  const refuels = useInfiniteQuery(refuelsListQueryOptions(filter))
  // Consumption needs each refuel's predecessor, which only the complete
  // chart series is guaranteed to hold.
  const chart = useQuery(refuelChartQueryOptions(filter))
  const repairs = useInfiniteQuery(repairsListQueryOptions(filter))
  const tickets = useInfiniteQuery(ticketsListQueryOptions(filter))
  const derived = new Map(
    chart.data?.items.map((refuel) => [refuel.id, refuel]),
  )
  const search = { carId: filter.carId }

  return (
    <div className="car-expenses">
      <ExpenseSection
        kind="refuels"
        query={refuels}
        add={
          <Link
            className="button button-secondary"
            to="/refuels/create"
            search={search}
          >
            <Plus aria-hidden="true" size={17} />
            {t('refuels.add')}
          </Link>
        }
      >
        {(items) => (
          <RefuelTable items={items} derived={derived} carFuel={carFuel} />
        )}
      </ExpenseSection>
      <ExpenseSection
        kind="repairs"
        query={repairs}
        add={
          <Link
            className="button button-secondary"
            to="/repairs/create"
            search={search}
          >
            <Plus aria-hidden="true" size={17} />
            {t('repairs.add')}
          </Link>
        }
      >
        {(items) => <RepairTable items={items} />}
      </ExpenseSection>
      <ExpenseSection
        kind="tickets"
        query={tickets}
        add={
          <Link
            className="button button-secondary"
            to="/tickets/create"
            search={search}
          >
            <Plus aria-hidden="true" size={17} />
            {t('tickets.add')}
          </Link>
        }
      >
        {(items) => <TicketTable items={items} />}
      </ExpenseSection>
    </div>
  )
}
