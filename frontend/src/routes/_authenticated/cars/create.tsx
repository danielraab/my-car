import { useMutation, useQueryClient } from '@tanstack/react-query'
import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { CarFront } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { type CarInput, createCar } from '#/api/client'
import {
  AttachmentQueue,
  useAttachmentQueue,
} from '#/attachments/attachment-queue'
import { CarForm } from '#/cars/car-form'
import { carQueryKey, carsListQueryKey } from '#/cars/queries'

export const Route = createFileRoute('/_authenticated/cars/create')({
  component: CreateCarPage,
})

function CreateCarPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const queue = useAttachmentQueue()
  const create = useMutation({
    // Queued files are uploaded once the car exists; ones that fail are
    // reported on its edit screen, since the car itself is saved.
    mutationFn: async (input: CarInput) => {
      const car = await createCar(input)
      return { car, failed: await queue.upload({ kind: 'cars', id: car.id }) }
    },
    onSuccess: ({ car, failed }) => {
      queryClient.removeQueries({ queryKey: carsListQueryKey })
      if (queue.files.length === 0) {
        queryClient.setQueryData(carQueryKey(car.id), car)
      } else {
        // The returned car predates the uploads, so its count is stale.
        queryClient.removeQueries({ queryKey: carQueryKey(car.id) })
      }
      if (failed.length > 0) {
        void navigate({
          to: '/cars/$carId/edit',
          params: { carId: car.id },
          state: { failedUploads: failed },
        })
        return
      }
      void navigate({ to: '/cars/$carId', params: { carId: car.id } })
    },
  })

  return (
    <section className="form-page" aria-labelledby="create-car-title">
      <header className="form-page-header">
        <div className="status-icon">
          <CarFront aria-hidden="true" size={24} />
        </div>
        <div>
          <p className="eyebrow">{t('cars.create.eyebrow')}</p>
          <h1 id="create-car-title">{t('cars.create.title')}</h1>
          <p>{t('cars.create.description')}</p>
        </div>
      </header>

      <CarForm
        submitLabel={t('cars.create.submit')}
        submittingLabel={t('cars.create.submitting')}
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
          <Link className="button button-secondary" to="/cars">
            {t('cars.edit.cancel')}
          </Link>
        }
      />
    </section>
  )
}
