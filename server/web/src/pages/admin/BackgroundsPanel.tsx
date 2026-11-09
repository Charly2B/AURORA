import { useId, useState, type ChangeEvent } from 'react'
import { api } from '../../lib/api'
import { useApi } from '../../lib/hooks'
import type { BackgroundGallery, BackgroundImage } from '../../lib/types'
import { Button, Notice, useToast } from '../../components/ui'

/** Galerie partagée : le choix du fond appartient à chaque application cliente. */
export function BackgroundsPanel() {
  const gallery = useApi<BackgroundGallery>('/backgrounds')
  const inputId = useId()
  const toast = useToast()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()

  const upload = async (event: ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files ?? [])
    event.target.value = ''
    if (!files.length) return
    setBusy(true)
    setError(undefined)
    let added = 0
    try {
      for (const file of files) {
        if (file.size > 50 * 1024 * 1024) throw new Error(`« ${file.name} » dépasse la limite de 50 Mio.`)
        const form = new FormData()
        form.append('image', file)
        await api.upload<BackgroundImage>('/admin/backgrounds', form)
        added++
      }
      toast('ok', `${added} ${added > 1 ? 'images disponibles' : 'image disponible'} pour les applications`)
    } catch (e) {
      setError(`${e instanceof Error ? e.message : String(e)}${added ? ` (${added} image(s) déjà importée(s).)` : ''}`)
    } finally {
      gallery.reload()
      setBusy(false)
    }
  }

  return (
    <section className="rounded-2xl bg-night/70 ring-1 ring-white/[0.06] p-6 space-y-4" aria-labelledby={`${inputId}-title`}>
      <h2 id={`${inputId}-title`} className="font-display text-2xl">Fonds d’écran des applications</h2>
      <p className="text-sm text-ash">Ajoutez des images sur le serveur. Chaque application TV, Android ou autre peut ensuite choisir son propre fond dans cette galerie.</p>
      <p className="text-xs text-ash">JPEG ou PNG · 50 Mio par image. Les grandes images sont adaptées automatiquement jusqu’en 4K, en conservant leurs proportions. Elles sont incluses dans les sauvegardes AURORA.</p>
      <div className="space-y-2">
        <label htmlFor={inputId} className="block text-sm font-medium text-mist">Ajouter des fonds d’écran</label>
        <input id={inputId} type="file" accept="image/jpeg,image/png,.jpg,.jpeg,.png" multiple disabled={busy}
          onChange={upload} className="block w-full text-sm text-mist file:mr-4 file:rounded-full file:border-0 file:bg-aurora file:px-4 file:py-2 file:text-ink file:font-medium disabled:opacity-50" />
      </div>
      {busy && <p role="status" className="text-sm text-ash">Import des images…</p>}
      {error && <Notice tone="error">{error}</Notice>}
      {gallery.error && <div className="space-y-2"><Notice tone="error">{gallery.error.message}</Notice><Button variant="ghost" onClick={gallery.reload}>Réessayer</Button></div>}
      {!gallery.data && !gallery.error && <p className="text-sm text-ash">Chargement de la galerie…</p>}
      {gallery.data?.images.length === 0 && <p className="text-sm text-ash">Aucun fond importé.</p>}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        {gallery.data?.images.map(image => (
          <a key={image.id} href={image.url} target="_blank" rel="noreferrer" aria-label={`Ouvrir ${image.name}`}
            className="rounded-xl overflow-hidden bg-coal ring-1 ring-white/10 hover:ring-aurora/50 focus-visible:outline-2 focus-visible:outline-aurora">
            <img src={image.url} alt="" loading="lazy" className="w-full aspect-video object-cover" />
            <div className="p-3 space-y-1">
              <p className="text-sm text-paper truncate">{image.name}</p>
              <p className="text-xs text-ash">{image.width} × {image.height} · {(image.size / (1024 * 1024)).toFixed(1)} Mio</p>
            </div>
          </a>
        ))}
      </div>
    </section>
  )
}
