import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { ReceiptText } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { createTicket } from '#/api/client'
import { isDeregistered } from '#/cars/deregistration'
import { allCarsQueryOptions } from '#/cars/queries'
import { CarsStatus } from '#/repairs/cars-status'
import { ticketLocationsQueryOptions, ticketsQueryKey } from '#/tickets/queries'
import { TicketForm } from '#/tickets/ticket-form'

export const Route = createFileRoute('/_authenticated/tickets/create')({
  validateSearch: (search: Record<string, unknown>): { carId?: string } =>
    typeof search.carId === 'string' ? { carId: search.carId } : {},
  component: CreateTicketPage,
})

function CreateTicketPage() {
  const { t } = useTranslation()
  const { carId } = Route.useSearch()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const cars = useQuery(allCarsQueryOptions)
  const locations = useQuery(ticketLocationsQueryOptions)
  const create = useMutation({
    mutationFn: createTicket,
    onSuccess: () => {
      queryClient.removeQueries({ queryKey: ticketsQueryKey })
      void navigate({ to: '/tickets' })
    },
  })
  // Deregistered cars are not offered, so a carId naming one preselects
  // nothing.
  const carList = (cars.data?.items ?? []).filter((car) => !isDeregistered(car))
  const initialCarId =
    carList.find((car) => car.id === carId)?.id ??
    (carList.length === 1 ? carList[0].id : '')

  return (
    <section className="form-page" aria-labelledby="create-ticket-title">
      <header className="form-page-header">
        <div className="status-icon">
          <ReceiptText aria-hidden="true" size={24} />
        </div>
        <div>
          <p className="eyebrow">{t('tickets.create.eyebrow')}</p>
          <h1 id="create-ticket-title">{t('tickets.create.title')}</h1>
          <p>{t('tickets.create.description')}</p>
        </div>
      </header>

      {carList.length > 0 ? (
        <TicketForm
          cars={carList}
          initialCarId={initialCarId}
          locations={locations.data ?? []}
          submitLabel={t('tickets.create.submit')}
          submittingLabel={t('tickets.create.submitting')}
          pending={create.isPending}
          error={create.error}
          onSubmit={(input) => create.mutate(input)}
          onEdit={() => {
            if (create.isError) create.reset()
          }}
          secondaryAction={
            <Link className="button button-secondary" to="/tickets">
              {t('tickets.create.cancel')}
            </Link>
          }
        />
      ) : (
        <CarsStatus cars={cars} namespace="tickets" />
      )}
    </section>
  )
}
