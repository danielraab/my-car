import { queryOptions } from '@tanstack/react-query'

import {
  type AttachmentOwner,
  getAttachmentLimits,
  getAttachments,
} from '#/api/client'

export const attachmentsQueryKey = ['attachments'] as const

export const attachmentListQueryKey = (owner: AttachmentOwner) =>
  [...attachmentsQueryKey, owner.kind, owner.id] as const

export const attachmentListQueryOptions = (owner: AttachmentOwner) =>
  queryOptions({
    queryKey: attachmentListQueryKey(owner),
    queryFn: () => getAttachments(owner),
  })

// The limits only change with the server's configuration.
export const attachmentLimitsQueryOptions = queryOptions({
  queryKey: [...attachmentsQueryKey, 'limits'],
  queryFn: getAttachmentLimits,
  staleTime: Number.POSITIVE_INFINITY,
})
