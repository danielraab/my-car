import { useQuery } from '@tanstack/react-query'
import { LoaderCircle, Paperclip } from 'lucide-react'
import { type ChangeEvent, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'

import type { AttachmentLimits } from '#/api/client'

import { MAX_IMAGE_EDGE } from './downscale'
import { formatBytes, prepareFiles, type Refusal } from './files'
import { attachmentLimitsQueryOptions } from './queries'

// Lets the user pick images and PDFs (on phones, also take a photo), scales
// large images down, and hands over the files that fit the limits; the
// others are named with the reason. The hint always says that images are
// scaled down and what the limits are.
export function FilePicker({
  existingCount,
  disabled = false,
  onFiles,
}: {
  existingCount: number
  disabled?: boolean
  onFiles: (files: File[]) => void
}) {
  const { t, i18n } = useTranslation()
  const locale = i18n.resolvedLanguage ?? 'en'
  const limits = useQuery(attachmentLimitsQueryOptions)
  const [refused, setRefused] = useState<Refusal[]>([])
  const [preparing, setPreparing] = useState(false)
  const inputId = useId()
  const hintId = useId()

  if (limits.data === undefined) {
    return limits.isError ? (
      <p className="field-error" role="alert">
        {t('attachments.limitsError')}
      </p>
    ) : (
      <p className="field-hint">{t('attachments.limitsLoading')}</p>
    )
  }
  const loaded = limits.data

  async function pick(event: ChangeEvent<HTMLInputElement>) {
    const picked = Array.from(event.target.files ?? [])
    // Reset so picking the same file again fires change again.
    event.target.value = ''
    if (picked.length === 0) return
    setPreparing(true)
    const result = await prepareFiles(picked, loaded, existingCount)
    setPreparing(false)
    setRefused(result.refused)
    if (result.accepted.length > 0) onFiles(result.accepted)
  }

  const full = existingCount >= loaded.maxFilesPerEntry
  const unavailable = disabled || preparing || full

  return (
    <div className="attachment-picker">
      <label
        className={`button button-secondary${unavailable ? ' is-disabled' : ''}`}
        htmlFor={inputId}
      >
        {preparing ? (
          <LoaderCircle className="spin" aria-hidden="true" size={17} />
        ) : (
          <Paperclip aria-hidden="true" size={17} />
        )}
        {t('attachments.add')}
      </label>
      <input
        id={inputId}
        className="sr-only"
        type="file"
        multiple
        accept={loaded.allowedContentTypes.join(',')}
        disabled={unavailable}
        aria-describedby={hintId}
        onChange={(event) => void pick(event)}
      />
      <p className="field-hint" id={hintId}>
        {t('attachments.hint', {
          edge: MAX_IMAGE_EDGE,
          size: formatBytes(loaded.maxFileBytes, locale),
          count: loaded.maxFilesPerEntry,
        })}
      </p>
      <RefusalList refused={refused} limits={loaded} />
    </div>
  )
}

export function RefusalList({
  refused,
  limits,
}: {
  refused: Refusal[]
  limits: AttachmentLimits | undefined
}) {
  const { t, i18n } = useTranslation()
  if (refused.length === 0) return null
  const locale = i18n.resolvedLanguage ?? 'en'
  return (
    <ul className="attachment-errors" role="alert">
      {refused.map((refusal) => (
        <li key={`${refusal.reason}-${refusal.name}`} className="field-error">
          {refusal.reason === 'upload'
            ? t('attachments.uploadError', { name: refusal.name })
            : t(`attachments.refused.${refusal.reason}`, {
                name: refusal.name,
                size: limits ? formatBytes(limits.maxFileBytes, locale) : '',
                count: limits?.maxFilesPerEntry,
              })}
        </li>
      ))}
    </ul>
  )
}
