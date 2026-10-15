import { type AttachmentOwner, uploadAttachment } from '#/api/client'

// Uploads queued files one after another for an entry that was just
// created, reporting how many are done. It never throws: the entry is
// already saved, so the names of the files that failed are returned for the
// edit screen to report.
export async function uploadQueuedFiles(
  owner: AttachmentOwner,
  files: File[],
  onProgress: (done: number, total: number) => void = () => {},
): Promise<string[]> {
  const failed: string[] = []
  onProgress(0, files.length)
  for (const [index, file] of files.entries()) {
    try {
      await uploadAttachment(owner, file)
    } catch {
      failed.push(file.name)
    }
    onProgress(index + 1, files.length)
  }
  return failed
}
