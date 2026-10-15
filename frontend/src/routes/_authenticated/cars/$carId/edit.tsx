import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { Pencil } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { type Car, type CarUpdate, deleteCar, updateCar } from '#/api/client'
import { AttachmentManager } from '#/attachments/attachment-manager'
import { CarForm } from '#/cars/car-form'
import { CarStatus } from '#/cars/car-status'
import { carQueryKey, carQueryOptions, carsListQueryKey } from '#/cars/queries'
import { TwoStepDeleteButton } from '#/components/two-step-delete-button'
import { repairsQueryKey } from '#/repairs/queries'

export const Route = createFileRoute('/_authenticated/cars/$carId/edit')({
  component: EditCarPage,
})

function EditCarPage() {
  const { carId } = Route.useParams()
  const car = useQuery(carQueryOptions(carId))

  return car.data ? (
    <EditCar car={car.data} />
  ) : (
    <section className="form-page">
      <CarStatus error={car.error} retry={() => car.refetch()} />
    </section>
  )
}

function EditCar({ car }: { car: Car }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const save = useMutation({
    mutationFn: (input: CarUpdate) => updateCar(car.id, input),
    onSuccess: (saved) => {
      queryClient.setQueryData(carQueryKey(saved.id), saved)
      queryClient.removeQueries({ queryKey: carsListQueryKey })
      void navigate({ to: '/cars/$carId', params: { carId: saved.id } })
    },
  })
  const remove = useMutation({
    mutationFn: () => deleteCar(car.id),
    onSuccess: () => {
      queryClient.removeQueries({ queryKey: carQueryKey(car.id) })
      queryClient.removeQueries({ queryKey: carsListQueryKey })
      // Deleting a car also deletes its repairs.
      queryClient.removeQueries({ queryKey: repairsQueryKey })
      void navigate({ to: '/cars' })
    },
  })

  return (
    <section className="form-page" aria-labelledby="edit-car-title">
      <header className="form-page-header">
        <div className="status-icon">
          <Pencil aria-hidden="true" size={24} />
        </div>
        <div>
          <p className="eyebrow">{t('cars.edit.eyebrow')}</p>
          <h1 id="edit-car-title">
            {t('cars.edit.title', { name: car.name })}
          </h1>
          <p>{t('cars.edit.description')}</p>
        </div>
      </header>

      <CarForm
        car={car}
        submitLabel={t('cars.edit.submit')}
        submittingLabel={t('cars.edit.submitting')}
        pending={save.isPending}
        error={save.error}
        onSubmit={(input) => save.mutate(input)}
        onEdit={() => {
          if (save.isError) save.reset()
        }}
        secondaryAction={
          <Link
            className="button button-secondary"
            to="/cars/$carId"
            params={{ carId: car.id }}
          >
            {t('cars.edit.cancel')}
          </Link>
        }
      />

      <AttachmentManager owner={{ kind: 'cars', id: car.id }} />

      <div className="form-card danger-zone">
        <TwoStepDeleteButton
          label={t('cars.delete.action')}
          prompt={t('cars.delete.prompt')}
          confirmLabel={t('cars.delete.confirm')}
          cancelLabel={t('cars.delete.cancel')}
          pendingLabel={t('cars.delete.deleting')}
          pending={remove.isPending}
          onConfirm={() => remove.mutate()}
        />
        {remove.isError ? (
          <p className="field-error" role="alert">
            {t('cars.delete.error')}
          </p>
        ) : null}
      </div>
    </section>
  )
}
