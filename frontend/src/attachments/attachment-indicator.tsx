import { Popover, PopoverButton, PopoverPanel } from '@headlessui/react'
import { useQuery } from '@tanstack/react-query'
import { Paperclip } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import type { AttachmentOwner } from '#/api/client'

import { FileList } from './attachment-list'
import { attachmentListQueryOptions } from './queries'

// The paperclip of a list row with attachments. Its popover loads the
// entry's files only when opened; each opens in a new tab.
export function AttachmentIndicator({
  owner,
  count,
}: {
  owner: AttachmentOwner
  count: number
}) {
  const { t } = useTranslation()
  if (count === 0) return null
  return (
    <Popover className="attachment-popover">
      <PopoverButton
        className="attachment-indicator"
        aria-label={t('attachments.indicator', { count })}
      >
        <Paperclip aria-hidden="true" size={15} />
        <span aria-hidden="true">{count}</span>
      </PopoverButton>
      <PopoverPanel anchor="bottom end" className="attachment-panel">
        <PopoverFiles owner={owner} />
      </PopoverPanel>
    </Popover>
  )
}

function PopoverFiles({ owner }: { owner: AttachmentOwner }) {
  const { t } = useTranslation()
  const attachments = useQuery(attachmentListQueryOptions(owner))
  if (attachments.data === undefined) {
    return (
      <p className={attachments.isError ? 'field-error' : 'field-hint'}>
        {t(
          attachments.isError ? 'attachments.loadError' : 'attachments.loading',
        )}
      </p>
    )
  }
  if (attachments.data.length === 0) {
    return <p className="field-hint">{t('attachments.empty')}</p>
  }
  return (
    <FileList
      label={t('attachments.listLabel')}
      items={attachments.data.map((attachment) => ({
        key: attachment.id,
        name: attachment.filename,
        contentType: attachment.contentType,
        size: attachment.size,
        attachmentId: attachment.id,
      }))}
    />
  )
}

// A read-only list of an entry's attachments, as on the car Details tab.
export function AttachmentOverview({ owner }: { owner: AttachmentOwner }) {
  const { t } = useTranslation()
  return (
    <section className="attachment-overview" aria-labelledby="overview-title">
      <h2 id="overview-title">{t('attachments.title')}</h2>
      <PopoverFiles owner={owner} />
    </section>
  )
}
