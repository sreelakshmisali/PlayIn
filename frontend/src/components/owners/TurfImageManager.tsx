import { useState, type FormEvent } from 'react'

import type { TurfImage } from '@/lib/owners'

interface TurfImageManagerProps {
  images: TurfImage[]
  pending: boolean
  removingId: string | null
  error: string
  onAdd: (url: string) => void
  onRemove: (imageId: string) => void
}

/**
 * URL-only image management: add a link, see it appear, remove it. There is
 * no upload and no storage this phase, so the row is exactly the URL an owner
 * typed.
 */
export function TurfImageManager({
  images,
  pending,
  removingId,
  error,
  onAdd,
  onRemove,
}: TurfImageManagerProps) {
  const [url, setUrl] = useState('')

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!url.trim()) return
    onAdd(url.trim())
    setUrl('')
  }

  return (
    <div>
      {images.length > 0 && (
        <ul className="grid grid-cols-3 gap-2 sm:grid-cols-4">
          {images.map((image) => (
            <li key={image.id} className="relative">
              <ImageThumb url={image.image_url} />
              <button
                type="button"
                onClick={() => onRemove(image.id)}
                disabled={removingId === image.id}
                aria-label="Remove image"
                className="absolute -right-1.5 -top-1.5 grid size-6 place-items-center rounded-full bg-neutral-900 text-white shadow-sm disabled:opacity-50"
              >
                ×
              </button>
            </li>
          ))}
        </ul>
      )}

      <form onSubmit={handleSubmit} className="mt-3 flex gap-2">
        <input
          type="url"
          value={url}
          onChange={(event) => setUrl(event.target.value)}
          placeholder="https://example.com/photo.jpg"
          inputMode="url"
          autoCapitalize="none"
          spellCheck={false}
          className="min-w-0 flex-1 rounded-lg border border-neutral-300 bg-white px-3.5 py-2.5 text-base text-neutral-900 placeholder:text-neutral-400 focus:outline-2 focus:outline-offset-1 focus:outline-pitch-600"
        />
        <button
          type="submit"
          disabled={pending || !url.trim()}
          className="shrink-0 rounded-lg bg-pitch-600 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-pitch-700 disabled:cursor-not-allowed disabled:bg-neutral-300"
        >
          Add
        </button>
      </form>

      {error && <p className="mt-2 text-sm text-red-700">{error}</p>}
    </div>
  )
}

function ImageThumb({ url }: { url: string }) {
  const [failed, setFailed] = useState(false)

  if (failed) {
    return (
      <span className="grid aspect-square place-items-center rounded-lg border border-neutral-200 bg-neutral-100 text-xs text-neutral-400">
        Broken
      </span>
    )
  }

  return (
    <img
      src={url}
      alt=""
      onError={() => setFailed(true)}
      className="aspect-square w-full rounded-lg border border-neutral-200 object-cover"
    />
  )
}
