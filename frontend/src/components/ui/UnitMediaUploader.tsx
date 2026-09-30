import { useRef, useState } from 'react'
import { Image, Upload, X } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { mediaURL } from '@/lib/apiClient'
import { useDeleteUnitCover, useDeleteUnitLogo, useUploadUnitCover, useUploadUnitLogo } from '@/hooks/useUnits'

interface Props {
  unitId: string
  brandLogoUrl?: string | null
  brandCoverUrl?: string | null
  editable?: boolean
}

export function UnitMediaUploader({ unitId, brandLogoUrl, brandCoverUrl, editable = true }: Props) {
  const logoInput = useRef<HTMLInputElement>(null)
  const coverInput = useRef<HTMLInputElement>(null)
  const [err, setErr] = useState('')

  const uploadLogo = useUploadUnitLogo()
  const uploadCover = useUploadUnitCover()
  const deleteLogo = useDeleteUnitLogo()
  const deleteCover = useDeleteUnitCover()

  const logo = mediaURL(brandLogoUrl) ?? null
  const cover = mediaURL(brandCoverUrl) ?? null

  function pickLogo() {
    setErr('')
    logoInput.current?.click()
  }
  function pickCover() {
    setErr('')
    coverInput.current?.click()
  }

  function onLogo(e: React.ChangeEvent<HTMLInputElement>) {
    const f = e.target.files?.[0]
    if (!f) return
    if (f.size > 5 * 1024 * 1024) { setErr('Logo must be 5 MB or smaller.'); return }
    uploadLogo.mutate({ unitId, file: f }, { onError: () => setErr('Could not upload logo.') })
    e.target.value = ''
  }
  function onCover(e: React.ChangeEvent<HTMLInputElement>) {
    const f = e.target.files?.[0]
    if (!f) return
    if (f.size > 5 * 1024 * 1024) { setErr('Cover must be 5 MB or smaller.'); return }
    uploadCover.mutate({ unitId, file: f }, { onError: () => setErr('Could not upload cover.') })
    e.target.value = ''
  }

  return (
    <div className="space-y-4">
      {/* Cover */}
      <div className="relative overflow-hidden rounded-panel border border-border-hi bg-surface-hi">
        <div className="relative h-32 sm:h-40">
          {cover ? (
            <img src={cover} alt="" className="h-full w-full object-cover" />
          ) : (
            <div className="grid h-full w-full place-items-center text-ink-faint">
              <Image className="size-8" aria-hidden />
            </div>
          )}
          {editable ? (
            <div className="absolute inset-0 flex items-center justify-center gap-2 bg-void/40 opacity-0 transition-opacity hover:opacity-100">
              <Button size="sm" icon={<Upload className="size-3.5" />} loading={uploadCover.isPending} onClick={pickCover}>
                {cover ? 'Replace cover' : 'Upload cover'}
              </Button>
              {cover ? (
                <Button size="sm" variant="ghost" icon={<X className="size-3.5" />} loading={deleteCover.isPending} onClick={() => deleteCover.mutate(unitId)}>
                  Remove
                </Button>
              ) : null}
            </div>
          ) : null}
        </div>

        {/* Logo overlay */}
        <div className="absolute -bottom-6 left-4 size-16 overflow-hidden rounded-full border-2 border-border bg-void">
          {logo ? (
            <img src={logo} alt="" className="h-full w-full object-cover" />
          ) : (
            <div className="grid h-full w-full place-items-center text-ink-faint text-[10px]">LOGO</div>
          )}
        </div>
      </div>

      {editable ? (
        <div className="flex flex-wrap items-center gap-2 pt-4">
          <Button size="sm" variant="ghost" icon={<Upload className="size-3.5" />} loading={uploadLogo.isPending} onClick={pickLogo}>
            {logo ? 'Replace logo' : 'Upload logo'}
          </Button>
          {logo ? (
            <Button size="sm" variant="ghost" icon={<X className="size-3.5" />} loading={deleteLogo.isPending} onClick={() => deleteLogo.mutate(unitId)}>
              Remove logo
            </Button>
          ) : null}
        </div>
      ) : null}

      <input ref={logoInput} type="file" accept="image/*" className="hidden" onChange={onLogo} />
      <input ref={coverInput} type="file" accept="image/*" className="hidden" onChange={onCover} />

      {err ? <p role="alert" className="text-xs text-warn">{err}</p> : null}
    </div>
  )
}