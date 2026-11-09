import { afterEach, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { SettingsPage } from '../pages/admin/Settings'

afterEach(() => { cleanup(); vi.unstubAllGlobals() })

it('importe plusieurs fichiers multipart dans la galerie disponible aux applications', async () => {
  const images = [{ id: 'night', name: 'Nuit.png', url: '/api/v1/backgrounds/night/image', width: 1920, height: 1080, size: 42, createdAt: 0 }]
  const uploaded: File[] = []
  vi.stubGlobal('fetch', async (url: string, init: RequestInit) => {
    if (url.endsWith('/admin/settings')) return new Response(JSON.stringify({ language: 'fr-FR', scanIntervalHours: 6, probeJobs: 2, maxTranscodes: 1, watchEnabled: true }))
    if (init.method === 'POST') {
      expect(init.headers).not.toHaveProperty('Content-Type')
      expect(init.headers).toHaveProperty('X-Aurora', '1')
      expect(init.body).toBeInstanceOf(FormData)
      uploaded.push((init.body as FormData).get('image') as File)
      return new Response(JSON.stringify(images[0]), { status: 201 })
    }
    return new Response(JSON.stringify({ images }))
  })
  render(<SettingsPage />)
  expect(await screen.findByText('Nuit.png')).toBeTruthy()
  const files = [new File(['png'], 'premier.png', { type: 'image/png' }), new File(['jpg'], 'second.jpg', { type: 'image/jpeg' })]
  Object.defineProperty(files[1], 'size', { value: 11 * 1024 * 1024 })
  fireEvent.change(screen.getByLabelText('Ajouter des fonds d’écran'), { target: { files } })
  await waitFor(() => expect(uploaded.map(f => f.name)).toEqual(['premier.png', 'second.jpg']))
})

it('affiche une erreur d’import et garde les images déjà présentes', async () => {
  vi.stubGlobal('fetch', async (url: string, init: RequestInit) => {
    if (url.endsWith('/admin/settings')) return new Response('{}')
    if (init.method === 'POST') return new Response(JSON.stringify({ error: 'image endommagée' }), { status: 400 })
    return new Response(JSON.stringify({ images: [] }))
  })
  render(<SettingsPage />)
  const input = await screen.findByLabelText('Ajouter des fonds d’écran')
  fireEvent.change(input, { target: { files: [new File(['x'], 'cassée.png')] } })
  expect(await screen.findByText('Image endommagée')).toBeTruthy()
  expect(screen.getByText('Aucun fond importé.')).toBeTruthy()
})
