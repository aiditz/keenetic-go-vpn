// src/utils.ts

// Base "time ago" formatter used across the app
export function formatTimeAgoFromDiffSeconds(diffSec: number): string {
  if (diffSec < 5) return 'just now'
  if (diffSec < 60) return `${diffSec} seconds ago`
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)} mins ago`
  return 'long time ago'
}

export function formatTimeAgoFromTimestamp(tsMs: number): string {
  const diffSec = Math.floor((Date.now() - tsMs) / 1000)
  return formatTimeAgoFromDiffSeconds(diffSec)
}

export function formatTimeAgoFromISO(iso?: string | null): string {
  if (!iso) return ''
  const t = Date.parse(iso)
  if (Number.isNaN(t)) return iso
  return formatTimeAgoFromTimestamp(t)
}