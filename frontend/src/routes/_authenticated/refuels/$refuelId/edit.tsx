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
  deleteRefuel,
  type Refuel,
  type RefuelInput,
  updateRefuel,
} from '#/api/client'
import { AttachmentManager } from '#/attachments/attachment-manager'
import { allCarsQueryOptions } from '#/cars/queries'
import { TwoStepDeleteButton } from '#/components/two-step-delete-button'
import {
  refuelQueryOptions,
  refuelStationsQueryOptions,
  refuelsQueryKey,
} from '#/refuels/queries'
import { RefuelForm } from '#/refuels/refuel-form'
import { CarsStatus } from '#/repairs/cars-status'

export const Route = createFileRoute('/_authenticated/refuels/$refuelId/edit')({
  component: EditRefuelPage,
})

function EditRefuelPage() {
  const { refuelId } = Route.useParams()
  const refuel = useQuery(refuelQueryOptions(refuelId))
  const cars = useQuery(allCarsQueryOptions)

  if (refuel.data === undefined) {
    return (
      <section className="form-page">
        <RefuelStatus error={refuel.error} retry={() => refuel.refetch()} />
      </section>
    )
  }
  if (cars.data === undefined) {
    return (
      <section className="form-page">
        <CarsStatus cars={cars} namespace="refuels" />
      </section>
    )
  }
  return <EditRefuel refuel={refuel.data} cars={cars.data.items} />
}

// The pending or failed state of the refuel query; a 404 covers both a
// missing refuel and one on another account's car.
function RefuelStatus({
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
        <p>{t('refuels.edit.loading')}</p>
      </output>
    )
  }

  if (error instanceof ApiError && error.status === 404) {
    return (
      <div className="form-status" role="alert">
        <SearchX aria-hidden="true" size={25} />
        <h2>{t('refuels.edit.notFoundTitle')}</h2>
        <p>{t('refuels.edit.notFound')}</p>
        <Link className="button button-secondary" to="/refuels">
          <ArrowLeft aria-hidden="true" size={17} />
          {t('refuels.edit.back')}
        </Link>
      </div>
    )
  }

  return (
    <div className="form-status" role="alert">
      <RefreshCw aria-hidden="true" size={25} />
      <h2>{t('refuels.edit.loadErrorTitle')}</h2>
      <p>{t('refuels.edit.loadError')}</p>
      <button className="button button-primary" type="button" onClick={retry}>
        <RefreshCw aria-hidden="true" size={17} />
        {t('refuels.retry')}
      </button>
    </div>
  )
}

function EditRefuel({ refuel, cars }: { refuel: Refuel; cars: Car[] }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const stations = useQuery(refuelStationsQueryOptions)
  const done = () => {
    queryClient.removeQueries({ queryKey: refuelsQueryKey })
    void navigate({ to: '/refuels' })
  }
  const save = useMutation({
    // The car is fixed once a refuel exists; RefuelUpdate has no carId.
    mutationFn: ({ carId: _car, ...update }: RefuelInput) =>
      updateRefuel(refuel.id, update),
    onSuccess: done,
  })
  const remove = useMutation({
    mutationFn: () => deleteRefuel(refuel.id),
    onSuccess: done,
  })

  return (
    <section className="form-page" aria-labelledby="edit-refuel-title">
      <header className="form-page-header">
        <div className="status-icon">
          <Pencil aria-hidden="true" size={24} />
        </div>
        <div>
          <p className="eyebrow">{t('refuels.edit.eyebrow')}</p>
          <h1 id="edit-refuel-title">{t('refuels.edit.title')}</h1>
          <p>{t('refuels.edit.description')}</p>
        </div>
      </header>

      <RefuelForm
        refuel={refuel}
        cars={cars}
        stations={stations.data ?? []}
        submitLabel={t('refuels.edit.submit')}
        submittingLabel={t('refuels.edit.submitting')}
        pending={save.isPending}
        error={save.error}
        onSubmit={(input) => save.mutate(input)}
        onEdit={() => {
          if (save.isError) save.reset()
        }}
        secondaryAction={
          <Link className="button button-secondary" to="/refuels">
            {t('refuels.edit.cancel')}
          </Link>
        }
      />

      <AttachmentManager owner={{ kind: 'refuels', id: refuel.id }} />

      <div className="form-card danger-zone">
        <TwoStepDeleteButton
          label={t('refuels.delete.action')}
          prompt={t('refuels.delete.prompt')}
          confirmLabel={t('refuels.delete.confirm')}
          cancelLabel={t('refuels.delete.cancel')}
          pendingLabel={t('refuels.delete.deleting')}
          pending={remove.isPending}
          onConfirm={() => remove.mutate()}
        />
        {remove.isError ? (
          <p className="field-error" role="alert">
            {t('refuels.delete.error')}
          </p>
        ) : null}
      </div>
    </section>
  )
}
