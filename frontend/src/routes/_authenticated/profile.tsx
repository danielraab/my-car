import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { LoaderCircle, RefreshCw, Save, UserRound } from 'lucide-react'
import { type FormEvent, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  ApiError,
  distanceUnits,
  getMe,
  type Profile,
  type ProfileUpdate,
  type Session,
  updateMe,
  volumeUnits,
} from '#/api/client'
import { sessionQueryKey } from '#/auth/session'
import { PasskeySection } from '#/components/passkey-section'

const profileQueryKey = ['me'] as const

export const Route = createFileRoute('/_authenticated/profile')({
  component: ProfilePage,
})

type FormValues = Required<
  Pick<
    ProfileUpdate,
    'firstName' | 'lastName' | 'currency' | 'volumeUnit' | 'distanceUnit'
  >
>
type FormField = keyof FormValues
const nameFields = ['firstName', 'lastName'] as const
const formFields: FormField[] = [
  ...nameFields,
  'currency',
  'volumeUnit',
  'distanceUnit',
]
const knownReasons = new Set(['too_long', 'invalid_type', 'invalid_enum'])

function ProfilePage() {
  const { t } = useTranslation()
  const profile = useQuery({ queryKey: profileQueryKey, queryFn: getMe })

  return (
    <section className="form-page" aria-labelledby="profile-title">
      <header className="form-page-header">
        <div className="status-icon">
          <UserRound aria-hidden="true" size={24} />
        </div>
        <div>
          <p className="eyebrow">{t('profile.eyebrow')}</p>
          <h1 id="profile-title">{t('profile.title')}</h1>
          <p>{t('profile.description')}</p>
        </div>
      </header>

      {profile.isPending ? (
        <output className="form-status">
          <LoaderCircle className="spin" aria-hidden="true" size={25} />
          <p>{t('profile.loading')}</p>
        </output>
      ) : profile.isError ? (
        <div className="form-status" role="alert">
          <RefreshCw aria-hidden="true" size={25} />
          <h2>{t('profile.loadErrorTitle')}</h2>
          <p>{t('profile.loadError')}</p>
          <button
            className="button button-primary"
            type="button"
            onClick={() => profile.refetch()}
          >
            <RefreshCw aria-hidden="true" size={17} />
            {t('profile.retry')}
          </button>
        </div>
      ) : (
        <ProfileForm profile={profile.data} />
      )}

      <PasskeySection />
    </section>
  )
}

function ProfileForm({ profile }: { profile: Profile }) {
  const { t, i18n } = useTranslation()
  const locale = i18n.resolvedLanguage ?? 'en'
  const queryClient = useQueryClient()
  const [values, setValues] = useState(() => formValues(profile))
  const save = useMutation({
    mutationFn: updateMe,
    onSuccess: (saved) => {
      queryClient.setQueryData(profileQueryKey, saved)
      queryClient.setQueryData<Session | null>(sessionQueryKey, (session) =>
        session ? { ...session, profile: saved } : session,
      )
      setValues(formValues(saved))
    },
  })
  const currencies = useMemo(
    () => currencyOptions(locale, values.currency),
    [locale, values.currency],
  )

  const fieldReasons =
    save.error instanceof ApiError && save.error.status === 400
      ? (save.error.fields ?? {})
      : {}
  const fieldErrors = formFields.filter((field) => fieldReasons[field])
  const formError = save.isError && fieldErrors.length === 0

  function fieldMessage(field: FormField): string {
    const reason = fieldReasons[field]
    return t(
      `profile.fieldErrors.${knownReasons.has(reason) ? reason : 'invalid'}`,
    )
  }

  function change<F extends FormField>(field: F, value: FormValues[F]) {
    setValues((current) => ({ ...current, [field]: value }))
    if (!save.isIdle) save.reset()
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    save.mutate(values)
  }

  function errorProps(field: FormField) {
    const invalid = fieldErrors.includes(field)
    return {
      id: field,
      name: field,
      'aria-describedby': invalid ? `${field}-error` : undefined,
      'aria-invalid': invalid,
    }
  }

  function fieldError(field: FormField) {
    return fieldErrors.includes(field) ? (
      <p className="field-error" id={`${field}-error`} role="alert">
        {fieldMessage(field)}
      </p>
    ) : null
  }

  return (
    <div className="form-card">
      <dl className="read-only-field">
        <dt>{t('profile.emailLabel')}</dt>
        <dd>{profile.email}</dd>
        <dd className="field-hint">{t('profile.emailHint')}</dd>
      </dl>

      <form noValidate onSubmit={submit}>
        {nameFields.map((field) => (
          <div className="form-field" key={field}>
            <label htmlFor={field}>{t(`profile.${field}Label`)}</label>
            <div className="input-wrap">
              <input
                {...errorProps(field)}
                autoComplete={
                  field === 'firstName' ? 'given-name' : 'family-name'
                }
                maxLength={100}
                type="text"
                value={values[field]}
                onChange={(event) => change(field, event.target.value)}
              />
            </div>
            {fieldError(field)}
          </div>
        ))}

        <fieldset className="profile-units" aria-describedby="units-hint">
          <legend>{t('profile.unitsTitle')}</legend>
          <p className="field-hint" id="units-hint">
            {t('profile.unitsHint')}
          </p>
          <div className="form-field">
            <label htmlFor="currency">{t('profile.currencyLabel')}</label>
            <div className="input-wrap">
              <select
                {...errorProps('currency')}
                value={values.currency}
                onChange={(event) => change('currency', event.target.value)}
              >
                {currencies.map((option) => (
                  <option key={option.code} value={option.code}>
                    {option.label}
                  </option>
                ))}
              </select>
            </div>
            {fieldError('currency')}
          </div>
          <div className="form-field">
            <label htmlFor="volumeUnit">{t('profile.volumeUnitLabel')}</label>
            <div className="input-wrap">
              <select
                {...errorProps('volumeUnit')}
                value={values.volumeUnit}
                onChange={(event) => {
                  const unit = volumeUnits.find(
                    (option) => option === event.target.value,
                  )
                  if (unit) change('volumeUnit', unit)
                }}
              >
                {volumeUnits.map((unit) => (
                  <option key={unit} value={unit}>
                    {t(`profile.volumeUnits.${unit}`)}
                  </option>
                ))}
              </select>
            </div>
            {fieldError('volumeUnit')}
          </div>
          <div className="form-field">
            <label htmlFor="distanceUnit">
              {t('profile.distanceUnitLabel')}
            </label>
            <div className="input-wrap">
              <select
                {...errorProps('distanceUnit')}
                value={values.distanceUnit}
                onChange={(event) => {
                  const unit = distanceUnits.find(
                    (option) => option === event.target.value,
                  )
                  if (unit) change('distanceUnit', unit)
                }}
              >
                {distanceUnits.map((unit) => (
                  <option key={unit} value={unit}>
                    {t(`profile.distanceUnits.${unit}`)}
                  </option>
                ))}
              </select>
            </div>
            {fieldError('distanceUnit')}
          </div>
        </fieldset>

        {formError ? (
          <p className="field-error" role="alert">
            {t('profile.saveError')}
          </p>
        ) : null}
        {save.isSuccess ? (
          <output className="form-success">{t('profile.saved')}</output>
        ) : null}

        <button
          className="button button-primary"
          disabled={save.isPending}
          type="submit"
        >
          <Save aria-hidden="true" size={18} />
          {save.isPending ? t('profile.saving') : t('profile.save')}
        </button>
      </form>
    </div>
  )
}

function formValues(profile: Profile): FormValues {
  return {
    firstName: profile.firstName,
    lastName: profile.lastName,
    currency: profile.currency,
    volumeUnit: profile.volumeUnit,
    distanceUnit: profile.distanceUnit,
  }
}

// Every ISO 4217 currency the browser knows, named in the active language
// and sorted by that name. The current currency is always offered, even if
// the browser does not list it.
function currencyOptions(locale: string, current: string) {
  const names = new Intl.DisplayNames(locale, { type: 'currency' })
  const codes = new Set(Intl.supportedValuesOf('currency'))
  codes.add(current)
  return [...codes]
    .map((code) => {
      const name = names.of(code) ?? code
      return { code, name, label: name === code ? code : `${name} (${code})` }
    })
    .sort((a, b) => a.name.localeCompare(b.name, locale))
}
