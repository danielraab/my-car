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
  deleteRepair,
  type Repair,
  type RepairInput,
  updateRepair,
} from '#/api/client'
import { AttachmentManager } from '#/attachments/attachment-manager'
import { allCarsQueryOptions } from '#/cars/queries'
import { TwoStepDeleteButton } from '#/components/two-step-delete-button'
import { CarsStatus } from '#/repairs/cars-status'
import {
  repairQueryOptions,
  repairStationsQueryOptions,
  repairsQueryKey,
} from '#/repairs/queries'
import { RepairForm } from '#/repairs/repair-form'

export const Route = createFileRoute('/_authenticated/repairs/$repairId/edit')({
  component: EditRepairPage,
})

function EditRepairPage() {
  const { repairId } = Route.useParams()
  const repair = useQuery(repairQueryOptions(repairId))
  const cars = useQuery(allCarsQueryOptions)

  if (repair.data === undefined) {
    return (
      <section className="form-page">
        <RepairStatus error={repair.error} retry={() => repair.refetch()} />
      </section>
    )
  }
  if (cars.data === undefined) {
    return (
      <section className="form-page">
        <CarsStatus cars={cars} />
      </section>
    )
  }
  return <EditRepair repair={repair.data} cars={cars.data.items} />
}

// The pending or failed state of the repair query; a 404 covers both a
// missing repair and one on another account's car.
function RepairStatus({
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
        <p>{t('repairs.edit.loading')}</p>
      </output>
    )
  }

  if (error instanceof ApiError && error.status === 404) {
    return (
      <div className="form-status" role="alert">
        <SearchX aria-hidden="true" size={25} />
        <h2>{t('repairs.edit.notFoundTitle')}</h2>
        <p>{t('repairs.edit.notFound')}</p>
        <Link className="button button-secondary" to="/repairs">
          <ArrowLeft aria-hidden="true" size={17} />
          {t('repairs.edit.back')}
        </Link>
      </div>
    )
  }

  return (
    <div className="form-status" role="alert">
      <RefreshCw aria-hidden="true" size={25} />
      <h2>{t('repairs.edit.loadErrorTitle')}</h2>
      <p>{t('repairs.edit.loadError')}</p>
      <button className="button button-primary" type="button" onClick={retry}>
        <RefreshCw aria-hidden="true" size={17} />
        {t('repairs.retry')}
      </button>
    </div>
  )
}

function EditRepair({ repair, cars }: { repair: Repair; cars: Car[] }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const stations = useQuery(repairStationsQueryOptions)
  const done = () => {
    queryClient.removeQueries({ queryKey: repairsQueryKey })
    void navigate({ to: '/repairs' })
  }
  const save = useMutation({
    // The car is fixed once a repair exists; RepairUpdate has no carId.
    mutationFn: ({ carId: _car, ...update }: RepairInput) =>
      updateRepair(repair.id, update),
    onSuccess: done,
  })
  const remove = useMutation({
    mutationFn: () => deleteRepair(repair.id),
    onSuccess: done,
  })

  return (
    <section className="form-page" aria-labelledby="edit-repair-title">
      <header className="form-page-header">
        <div className="status-icon">
          <Pencil aria-hidden="true" size={24} />
        </div>
        <div>
          <p className="eyebrow">{t('repairs.edit.eyebrow')}</p>
          <h1 id="edit-repair-title">{t('repairs.edit.title')}</h1>
          <p>{t('repairs.edit.description')}</p>
        </div>
      </header>

      <RepairForm
        repair={repair}
        cars={cars}
        stations={stations.data ?? []}
        submitLabel={t('repairs.edit.submit')}
        submittingLabel={t('repairs.edit.submitting')}
        pending={save.isPending}
        error={save.error}
        onSubmit={(input) => save.mutate(input)}
        onEdit={() => {
          if (save.isError) save.reset()
        }}
        secondaryAction={
          <Link className="button button-secondary" to="/repairs">
            {t('repairs.edit.cancel')}
          </Link>
        }
      />

      <AttachmentManager owner={{ kind: 'repairs', id: repair.id }} />

      <div className="form-card danger-zone">
        <TwoStepDeleteButton
          label={t('repairs.delete.action')}
          prompt={t('repairs.delete.prompt')}
          confirmLabel={t('repairs.delete.confirm')}
          cancelLabel={t('repairs.delete.cancel')}
          pendingLabel={t('repairs.delete.deleting')}
          pending={remove.isPending}
          onConfirm={() => remove.mutate()}
        />
        {remove.isError ? (
          <p className="field-error" role="alert">
            {t('repairs.delete.error')}
          </p>
        ) : null}
      </div>
    </section>
  )
}
