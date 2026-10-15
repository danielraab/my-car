import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { Fuel } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { createRefuel, type RefuelInput } from '#/api/client'
import {
  AttachmentQueue,
  useAttachmentQueue,
} from '#/attachments/attachment-queue'
import { isDeregistered } from '#/cars/deregistration'
import { allCarsQueryOptions } from '#/cars/queries'
import { refuelStationsQueryOptions, refuelsQueryKey } from '#/refuels/queries'
import { RefuelForm } from '#/refuels/refuel-form'
import { CarsStatus } from '#/repairs/cars-status'

export const Route = createFileRoute('/_authenticated/refuels/create')({
  validateSearch: (search: Record<string, unknown>): { carId?: string } =>
    typeof search.carId === 'string' ? { carId: search.carId } : {},
  component: CreateRefuelPage,
})

function CreateRefuelPage() {
  const { t } = useTranslation()
  const { carId } = Route.useSearch()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const cars = useQuery(allCarsQueryOptions)
  const stations = useQuery(refuelStationsQueryOptions)
  const queue = useAttachmentQueue()
  const create = useMutation({
    // Queued files are uploaded once the refuel exists; ones that fail are
    // reported on its edit screen, since the refuel itself is saved.
    mutationFn: async (input: RefuelInput) => {
      const refuel = await createRefuel(input)
      return {
        refuel,
        failed: await queue.upload({ kind: 'refuels', id: refuel.id }),
      }
    },
    onSuccess: ({ refuel, failed }) => {
      queryClient.removeQueries({ queryKey: refuelsQueryKey })
      if (failed.length > 0) {
        void navigate({
          to: '/refuels/$refuelId/edit',
          params: { refuelId: refuel.id },
          state: { failedUploads: failed },
        })
        return
      }
      void navigate({ to: '/refuels' })
    },
  })
  // Deregistered cars are not offered, so a carId naming one preselects
  // nothing.
  const carList = (cars.data?.items ?? []).filter((car) => !isDeregistered(car))
  const initialCarId =
    carList.find((car) => car.id === carId)?.id ??
    (carList.length === 1 ? carList[0].id : '')

  return (
    <section className="form-page" aria-labelledby="create-refuel-title">
      <header className="form-page-header">
        <div className="status-icon">
          <Fuel aria-hidden="true" size={24} />
        </div>
        <div>
          <p className="eyebrow">{t('refuels.create.eyebrow')}</p>
          <h1 id="create-refuel-title">{t('refuels.create.title')}</h1>
          <p>{t('refuels.create.description')}</p>
        </div>
      </header>

      {carList.length > 0 ? (
        <RefuelForm
          cars={carList}
          initialCarId={initialCarId}
          stations={stations.data ?? []}
          submitLabel={t('refuels.create.submit')}
          submittingLabel={t('refuels.create.submitting')}
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
            <Link className="button button-secondary" to="/refuels">
              {t('refuels.create.cancel')}
            </Link>
          }
        />
      ) : (
        <CarsStatus cars={cars} namespace="refuels" />
      )}
    </section>
  )
}
