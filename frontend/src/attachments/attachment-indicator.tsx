import { Popover, PopoverButton, PopoverPanel } from '@headlessui/react'
import { useQuery } from '@tanstack/react-query'
import { Paperclip } from 'lucide-react'
import { useRef } from 'react'
import { useTranslation } from 'react-i18next'

import type { Attachment, AttachmentOwner } from '#/api/client'

import { FileList } from './attachment-list'
import { useAttachmentViewer } from './attachment-viewer'
import { attachmentListQueryOptions } from './queries'

// The paperclip of a list row with attachments. Its popover loads the
// entry's files only when opened; opening a file closes the popover and shows
// the file in the viewer, which returns focus to the paperclip.
export function AttachmentIndicator({
  owner,
  count,
}: {
  owner: AttachmentOwner
  count: number
}) {
  const { t } = useTranslation()
  const button = useRef<HTMLButtonElement>(null)
  const { open, viewer } = useAttachmentViewer(button)
  if (count === 0) return null
  return (
    <>
      <Popover className="attachment-popover">
        <PopoverButton
          ref={button}
          className="attachment-indicator"
          aria-label={t('attachments.indicator', { count })}
        >
          <Paperclip aria-hidden="true" size={15} />
          <span aria-hidden="true">{count}</span>
        </PopoverButton>
        <PopoverPanel anchor="bottom end" className="attachment-panel">
          {({ close }) => (
            <AttachmentFiles
              owner={owner}
              onOpen={(attachments, index) => {
                close()
                open(attachments, index)
              }}
            />
          )}
        </PopoverPanel>
      </Popover>
      {viewer}
    </>
  )
}

function AttachmentFiles({
  owner,
  onOpen,
}: {
  owner: AttachmentOwner
  onOpen: (attachments: Attachment[], index: number) => void
}) {
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
  const data = attachments.data
  return (
    <FileList
      label={t('attachments.listLabel')}
      items={data.map((attachment) => ({
        key: attachment.id,
        name: attachment.filename,
        contentType: attachment.contentType,
        size: attachment.size,
      }))}
      onOpen={(index) => onOpen(data, index)}
    />
  )
}

// A read-only list of an entry's attachments, as on the car Details tab.
export function AttachmentOverview({ owner }: { owner: AttachmentOwner }) {
  const { t } = useTranslation()
  const { open, viewer } = useAttachmentViewer()
  return (
    <section className="attachment-overview" aria-labelledby="overview-title">
      <h2 id="overview-title">{t('attachments.title')}</h2>
      <AttachmentFiles owner={owner} onOpen={open} />
      {viewer}
    </section>
  )
}
