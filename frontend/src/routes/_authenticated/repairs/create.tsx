import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { Wrench } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { createRepair, type RepairInput } from '#/api/client'
import {
  AttachmentQueue,
  useAttachmentQueue,
} from '#/attachments/attachment-queue'
import { isDeregistered } from '#/cars/deregistration'
import { allCarsQueryOptions } from '#/cars/queries'
import { CarsStatus } from '#/repairs/cars-status'
import { repairStationsQueryOptions, repairsQueryKey } from '#/repairs/queries'
import { RepairForm } from '#/repairs/repair-form'

export const Route = createFileRoute('/_authenticated/repairs/create')({
  validateSearch: (search: Record<string, unknown>): { carId?: string } =>
    typeof search.carId === 'string' ? { carId: search.carId } : {},
  component: CreateRepairPage,
})

function CreateRepairPage() {
  const { t } = useTranslation()
  const { carId } = Route.useSearch()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const cars = useQuery(allCarsQueryOptions)
  const stations = useQuery(repairStationsQueryOptions)
  const queue = useAttachmentQueue()
  const create = useMutation({
    // Queued files are uploaded once the repair exists; ones that fail are
    // reported on its edit screen, since the repair itself is saved.
    mutationFn: async (input: RepairInput) => {
      const repair = await createRepair(input)
      return {
        repair,
        failed: await queue.upload({ kind: 'repairs', id: repair.id }),
      }
    },
    onSuccess: ({ repair, failed }) => {
      queryClient.removeQueries({ queryKey: repairsQueryKey })
      if (failed.length > 0) {
        void navigate({
          to: '/repairs/$repairId/edit',
          params: { repairId: repair.id },
          state: { failedUploads: failed },
        })
        return
      }
      void navigate({ to: '/repairs' })
    },
  })
  // Deregistered cars are not offered, so a carId naming one preselects
  // nothing.
  const carList = (cars.data?.items ?? []).filter((car) => !isDeregistered(car))
  const initialCarId =
    carList.find((car) => car.id === carId)?.id ??
    (carList.length === 1 ? carList[0].id : '')

  return (
    <section className="form-page" aria-labelledby="create-repair-title">
      <header className="form-page-header">
        <div className="status-icon">
          <Wrench aria-hidden="true" size={24} />
        </div>
        <div>
          <p className="eyebrow">{t('repairs.create.eyebrow')}</p>
          <h1 id="create-repair-title">{t('repairs.create.title')}</h1>
          <p>{t('repairs.create.description')}</p>
        </div>
      </header>

      {carList.length > 0 ? (
        <RepairForm
          cars={carList}
          initialCarId={initialCarId}
          stations={stations.data ?? []}
          submitLabel={t('repairs.create.submit')}
          submittingLabel={t('repairs.create.submitting')}
          pending={create.isPending}
          error={create.error}
          onSubmit={(input) => create.mutate(input)}
          attachments={
            <AttachmentQueue queue={queue} disabled={create.isPending} />
          }
          onEdit={() => {
            if (create.isError) create.reset()
          }}
          secondaryAction={
            <Link className="button button-secondary" to="/repairs">
              {t('repairs.create.cancel')}
            </Link>
          }
        />
      ) : (
        <CarsStatus cars={cars} />
      )}
    </section>
  )
}
