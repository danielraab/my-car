import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useLocation } from '@tanstack/react-router'
import { LoaderCircle, RefreshCw } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  ApiError,
  type Attachment,
  type AttachmentOwner,
  deleteAttachment,
  uploadAttachment,
} from '#/api/client'
import { TwoStepDeleteButton } from '#/components/two-step-delete-button'

import { FileList } from './attachment-list'
import { FilePicker, RefusalList } from './file-picker'
import type { Refusal, RefusalReason } from './files'
import {
  attachmentLimitsQueryOptions,
  attachmentListQueryOptions,
} from './queries'

const reasonsByCode: Record<string, RefusalReason> = {
  unsupported_media_type: 'type',
  file_too_large: 'size',
  attachment_limit_reached: 'count',
}

// Refreshes an entry's attachments and every query of its collection, whose
// representations carry attachmentCount.
export function useInvalidateAttachments(owner: AttachmentOwner) {
  const queryClient = useQueryClient()
  return () =>
    Promise.all([
      queryClient.invalidateQueries({
        queryKey: attachmentListQueryOptions(owner).queryKey,
      }),
      queryClient.invalidateQueries({ queryKey: [owner.kind] }),
    ])
}

// The attachment card of an edit screen. Unlike the form next to it, every
// change applies at once: picked files are uploaded and confirmed deletions
// take effect immediately.
export function AttachmentManager({ owner }: { owner: AttachmentOwner }) {
  const { t } = useTranslation()
  const failedAfterCreate = useLocation({
    select: (location) => location.state.failedUploads,
  })
  const attachments = useQuery(attachmentListQueryOptions(owner))
  const limits = useQuery(attachmentLimitsQueryOptions)
  const invalidate = useInvalidateAttachments(owner)
  const [refused, setRefused] = useState<Refusal[]>([])
  const upload = useMutation({
    mutationFn: async (files: File[]) => {
      const failed: Refusal[] = []
      for (const file of files) {
        try {
          await uploadAttachment(owner, file)
        } catch (error) {
          const reason =
            error instanceof ApiError ? reasonsByCode[error.code] : undefined
          failed.push({ name: file.name, reason: reason ?? 'upload' })
        }
      }
      return failed
    },
    onMutate: () => setRefused([]),
    onSettled: () => invalidate(),
    onSuccess: setRefused,
  })

  return (
    <section
      className="form-card attachment-section"
      aria-labelledby="attachments-title"
    >
      <h2 id="attachments-title">{t('attachments.title')}</h2>
      {failedAfterCreate && failedAfterCreate.length > 0 ? (
        <p className="field-error" role="alert">
          {t('attachments.failedAfterCreate', {
            names: failedAfterCreate.join(', '),
          })}
        </p>
      ) : null}
      {attachments.data === undefined ? (
        attachments.isError ? (
          <div className="attachment-status" role="alert">
            <p>{t('attachments.loadError')}</p>
            <button
              className="button button-secondary button-compact"
              type="button"
              onClick={() => attachments.refetch()}
            >
              <RefreshCw aria-hidden="true" size={16} />
              {t('attachments.retry')}
            </button>
          </div>
        ) : (
          <p className="field-hint">{t('attachments.loading')}</p>
        )
      ) : (
        <>
          {attachments.data.length === 0 ? (
            <p className="field-hint">{t('attachments.empty')}</p>
          ) : (
            <FileList
              label={t('attachments.listLabel')}
              items={attachments.data.map((attachment) => ({
                key: attachment.id,
                name: attachment.filename,
                contentType: attachment.contentType,
                size: attachment.size,
                attachmentId: attachment.id,
                actions: (
                  <DeleteAttachment
                    attachment={attachment}
                    onDeleted={invalidate}
                  />
                ),
              }))}
            />
          )}
          {upload.isPending ? (
            <output className="attachment-progress">
              <LoaderCircle className="spin" aria-hidden="true" size={16} />
              {t('attachments.uploading')}
            </output>
          ) : null}
          <FilePicker
            existingCount={attachments.data.length}
            disabled={upload.isPending}
            onFiles={(files) => upload.mutate(files)}
          />
          <RefusalList refused={refused} limits={limits.data} />
        </>
      )}
    </section>
  )
}

function DeleteAttachment({
  attachment,
  onDeleted,
}: {
  attachment: Attachment
  onDeleted: () => Promise<unknown>
}) {
  const { t } = useTranslation()
  const remove = useMutation({
    mutationFn: () => deleteAttachment(attachment.id),
    onSuccess: onDeleted,
  })
  const name = attachment.filename
  return (
    <span className="attachment-delete">
      <TwoStepDeleteButton
        label={t('attachments.delete.action')}
        prompt={t('attachments.delete.prompt', { name })}
        confirmLabel={t('attachments.delete.confirm')}
        cancelLabel={t('attachments.delete.cancel')}
        pendingLabel={t('attachments.delete.deleting')}
        pending={remove.isPending}
        onConfirm={() => remove.mutate()}
      />
      {remove.isError ? (
        <span className="field-error" role="alert">
          {t('attachments.delete.error')}
        </span>
      ) : null}
    </span>
  )
}
