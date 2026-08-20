import { useState } from 'react'

import type { PlayerProfile } from '@/lib/players'

/**
 * The read view of a profile. It is the same component for the owner and for a
 * visitor, because the payload is the same: a profile carries nothing an owner
 * should see and a stranger should not.
 */
export function ProfileCard({ profile }: { profile: PlayerProfile }) {
  return (
    <article>
      <header className="flex items-center gap-4">
        <Avatar name={profile.display_name} imageUrl={profile.image_url} />

        <div className="min-w-0">
          <h1 className="truncate text-xl font-semibold tracking-tight text-neutral-900 sm:text-2xl">
            {profile.display_name}
          </h1>
          {profile.location && (
            <p className="mt-0.5 flex items-center gap-1 text-sm text-neutral-500">
              <PinIcon />
              <span className="truncate">{profile.location}</span>
            </p>
          )}
        </div>
      </header>

      {profile.bio && (
        <p className="mt-5 whitespace-pre-line text-neutral-700">{profile.bio}</p>
      )}

      <section className="mt-6">
        <h2 className="text-sm font-medium text-neutral-800">Preferred sports</h2>

        {profile.sports.length === 0 ? (
          <p className="mt-2 text-sm text-neutral-500">No sports chosen yet.</p>
        ) : (
          <ul className="mt-3 space-y-2">
            {profile.sports.map((entry) => (
              <li
                key={entry.sport.id}
                className="flex items-center justify-between gap-3 rounded-lg border border-neutral-200 bg-white px-4 py-3"
              >
                <span className="font-medium text-neutral-900">{entry.sport.name}</span>
                {entry.position ? (
                  <span className="shrink-0 rounded-full bg-pitch-50 px-3 py-1 text-xs font-medium text-pitch-700">
                    {entry.position}
                  </span>
                ) : (
                  <span className="shrink-0 text-xs text-neutral-400">No position</span>
                )}
              </li>
            ))}
          </ul>
        )}
      </section>
    </article>
  )
}

/**
 * The profile image, falling back to initials.
 *
 * The image URL is arbitrary text a player typed, so a load failure is expected
 * rather than exceptional and swaps in the initials instead of leaving a broken
 * image icon. The server restricts the scheme to http and https, which is what
 * stops a javascript: URL reaching this attribute.
 */
function Avatar({ name, imageUrl }: { name: string; imageUrl: string | undefined }) {
  const [failed, setFailed] = useState(false)

  if (imageUrl && !failed) {
    return (
      <img
        src={imageUrl}
        alt=""
        onError={() => setFailed(true)}
        className="size-16 shrink-0 rounded-full border border-neutral-200 object-cover"
      />
    )
  }

  return (
    <span
      aria-hidden="true"
      className="grid size-16 shrink-0 place-items-center rounded-full bg-pitch-600 text-xl font-semibold text-white"
    >
      {initials(name)}
    </span>
  )
}

function initials(name: string): string {
  const parts = name.trim().split(/\s+/).slice(0, 2)
  return parts.map((part) => part.charAt(0).toUpperCase()).join('') || '?'
}

function PinIcon() {
  return (
    <svg viewBox="0 0 16 16" aria-hidden="true" className="size-3.5 shrink-0 fill-current">
      <path d="M8 1a5 5 0 0 0-5 5c0 3.5 5 9 5 9s5-5.5 5-9a5 5 0 0 0-5-5Zm0 7a2 2 0 1 1 0-4 2 2 0 0 1 0 4Z" />
    </svg>
  )
}
