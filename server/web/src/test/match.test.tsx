import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { FixMatch } from '../components/detail'
import type { Detail } from '../lib/types'

// jsdom ne fournit pas les méthodes natives de <dialog>.
beforeEach(() => {
  Object.defineProperty(HTMLDialogElement.prototype, 'showModal', { configurable: true, value() { this.open = true } })
  Object.defineProperty(HTMLDialogElement.prototype, 'close', { configurable: true, value() { this.open = false } })
})
afterEach(() => { cleanup(); vi.unstubAllGlobals() })

it('associe le candidat IMDb choisi même quand plusieurs résultats ont un id TMDB nul', async () => {
  const fetch = vi.fn(async (_path: string, init: RequestInit) => new Response(JSON.stringify(init.method === 'GET' ? [
    { id: 0, imdbId: 'tt1234567', title: 'First Film', originalTitle: 'First Film', year: 2023, overview: '' },
    { id: 0, imdbId: 'tt7654321', title: 'Second Film', originalTitle: 'Second Film', year: 2024, overview: '' },
  ] : { itemId: 77 }), { status: 200 }))
  vi.stubGlobal('fetch', fetch)
  const onDone = vi.fn()
  render(<FixMatch item={{ id: 77, title: 'Local Film', kind: 'movie', matchState: 'failed' } as Detail} onDone={onDone} />)
  fireEvent.click(screen.getByRole('button', { name: "Corriger l'identification" }))
  fireEvent.click(await screen.findByRole('button', { name: 'Second Film 2024' }))
  await waitFor(() => expect(onDone).toHaveBeenCalledWith(77))
  expect(JSON.parse(String(fetch.mock.calls[1][1].body))).toEqual({ imdbId: 'tt7654321' })
})

it('conserve le protocole existant pour un résultat TMDB', async () => {
  const fetch = vi.fn(async (_path: string, init: RequestInit) => new Response(JSON.stringify(init.method === 'GET' ? [
    { id: 603, title: 'Matrix', originalTitle: 'Matrix', year: 1999, overview: '' },
  ] : { itemId: 77 }), { status: 200 }))
  vi.stubGlobal('fetch', fetch)
  render(<FixMatch item={{ id: 77, title: 'Local Film', kind: 'movie', matchState: 'failed' } as Detail} onDone={() => {}} />)
  fireEvent.click(screen.getByRole('button', { name: "Corriger l'identification" }))
  fireEvent.click(await screen.findByRole('button', { name: 'Matrix 1999' }))
  await waitFor(() => expect(fetch).toHaveBeenCalledTimes(2))
  expect(JSON.parse(String(fetch.mock.calls[1][1].body))).toEqual({ tmdbId: 603 })
})
