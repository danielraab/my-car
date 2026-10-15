import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import {
  ArrowLeft,
  LoaderCircle,
  Pencil,
  RefreshCw,
  SearchX,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import {
  ApiError,
  type Car,
  deleteTicket,
  type Ticket,
  type TicketInput,
  updateTicket,
} from '#/api/client'
import { AttachmentManager } from '#/attachments/attachment-manager'
import { allCarsQueryOptions } from '#/cars/queries'
import { TwoStepDeleteButton } from '#/components/two-step-delete-button'
import { CarsStatus } from '#/repairs/cars-status'
import {
  ticketLocationsQueryOptions,
  ticketQueryOptions,
  ticketsQueryKey,
} from '#/tickets/queries'
import { TicketForm } from '#/tickets/ticket-form'

export const Route = createFileRoute('/_authenticated/tickets/$ticketId/edit')({
  component: EditTicketPage,
})

function EditTicketPage() {
  const { ticketId } = Route.useParams()
  const ticket = useQuery(ticketQueryOptions(ticketId))
  const cars = useQuery(allCarsQueryOptions)

  if (ticket.data === undefined) {
    return (
      <section className="form-page">
        <TicketStatus error={ticket.error} retry={() => ticket.refetch()} />
      </section>
    )
  }
  if (cars.data === undefined) {
    return (
      <section className="form-page">
        <CarsStatus cars={cars} namespace="tickets" />
      </section>
    )
  }
  return <EditTicket ticket={ticket.data} cars={cars.data.items} />
}

// The pending or failed state of the ticket query; a 404 covers both a
// missing ticket and one on another account's car.
function TicketStatus({
  error,
  retry,
}: {
  error: Error | null
  retry: () => void
}) {
  const { t } = useTranslation()

  if (error === null) {
    return (
      <output className="form-status">
        <LoaderCircle className="spin" aria-hidden="true" size={25} />
        <p>{t('tickets.edit.loading')}</p>
      </output>
    )
  }

  if (error instanceof ApiError && error.status === 404) {
    return (
      <div className="form-status" role="alert">
        <SearchX aria-hidden="true" size={25} />
        <h2>{t('tickets.edit.notFoundTitle')}</h2>
        <p>{t('tickets.edit.notFound')}</p>
        <Link className="button button-secondary" to="/tickets">
          <ArrowLeft aria-hidden="true" size={17} />
          {t('tickets.edit.back')}
        </Link>
      </div>
    )
  }

  return (
    <div className="form-status" role="alert">
      <RefreshCw aria-hidden="true" size={25} />
      <h2>{t('tickets.edit.loadErrorTitle')}</h2>
      <p>{t('tickets.edit.loadError')}</p>
      <button className="button button-primary" type="button" onClick={retry}>
        <RefreshCw aria-hidden="true" size={17} />
        {t('tickets.retry')}
      </button>
    </div>
  )
}

function EditTicket({ ticket, cars }: { ticket: Ticket; cars: Car[] }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const locations = useQuery(ticketLocationsQueryOptions)
  const done = () => {
    queryClient.removeQueries({ queryKey: ticketsQueryKey })
    void navigate({ to: '/tickets' })
  }
  const save = useMutation({
    // The car is fixed once a ticket exists; TicketUpdate has no carId.
    mutationFn: ({ carId: _car, ...update }: TicketInput) =>
      updateTicket(ticket.id, update),
    onSuccess: done,
  })
  const remove = useMutation({
    mutationFn: () => deleteTicket(ticket.id),
    onSuccess: done,
  })

  return (
    <section className="form-page" aria-labelledby="edit-ticket-title">
      <header className="form-page-header">
        <div className="status-icon">
          <Pencil aria-hidden="true" size={24} />
        </div>
        <div>
          <p className="eyebrow">{t('tickets.edit.eyebrow')}</p>
          <h1 id="edit-ticket-title">{t('tickets.edit.title')}</h1>
          <p>{t('tickets.edit.description')}</p>
        </div>
      </header>

      <TicketForm
        ticket={ticket}
        cars={cars}
        locations={locations.data ?? []}
        submitLabel={t('tickets.edit.submit')}
        submittingLabel={t('tickets.edit.submitting')}
        pending={save.isPending}
        error={save.error}
        onSubmit={(input) => save.mutate(input)}
        onEdit={() => {
          if (save.isError) save.reset()
        }}
        secondaryAction={
          <Link className="button button-secondary" to="/tickets">
            {t('tickets.edit.cancel')}
          </Link>
        }
      />

      <AttachmentManager owner={{ kind: 'tickets', id: ticket.id }} />

      <div className="form-card danger-zone">
        <TwoStepDeleteButton
          label={t('tickets.delete.action')}
          prompt={t('tickets.delete.prompt')}
          confirmLabel={t('tickets.delete.confirm')}
          cancelLabel={t('tickets.delete.cancel')}
          pendingLabel={t('tickets.delete.deleting')}
          pending={remove.isPending}
          onConfirm={() => remove.mutate()}
        />
        {remove.isError ? (
          <p className="field-error" role="alert">
            {t('tickets.delete.error')}
          </p>
        ) : null}
      </div>
    </section>
  )
}
