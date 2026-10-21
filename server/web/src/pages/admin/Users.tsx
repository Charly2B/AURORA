import { useState } from 'react'
import { Baby, Pencil, Plus, Shield, Trash2, UserRound } from 'lucide-react'
import { api, ApiError } from '../../lib/api'
import { useApi } from '../../lib/hooks'
import { useSession } from '../../lib/session'
import { Button, ErrorState, Field, Modal, Notice, PageLoader, Select, Toggle, cx, useToast } from '../../components/ui'

interface U {
  id: number; username: string; role: 'admin' | 'user' | 'child'; maxAge: number | null; hideUnrated: boolean
  allLibraries: boolean; libraries: number[] | null; createdAt: number
}
interface Lib { id: number; name: string; kind: string }

const ROLE = { admin: 'Administrateur', user: 'Utilisateur', child: 'Enfant' }
const AGES = [['', 'Aucune limite'], ['0', 'Tous publics'], ['10', '10 ans'], ['12', '12 ans'], ['16', '16 ans']]

export function UsersPage() {
  const users = useApi<U[]>('/admin/users')
  const libs = useApi<Lib[]>('/admin/libraries')
  const { user: me } = useSession()
  const toast = useToast()
  const [edit, setEdit] = useState<Partial<U> & { password?: string }>()
  const [del, setDel] = useState<U>()
  if (users.error) return <ErrorState error={users.error} retry={users.reload} />
  if (!users.data || !libs.data) return <PageLoader />
  const remove = async () => {
    try { await api.del(`/admin/users/${del!.id}`); setDel(undefined); users.reload(); toast('ok', 'Profil supprimé') } catch (e) { toast('error', (e as Error).message) }
  }
  return (
    <div className="max-w-4xl space-y-6">
      <div className="flex justify-between items-center gap-4">
        <p className="text-sm text-ash">Chaque profil a son historique, sa progression, ses favoris et sa liste.</p>
        <Button variant="primary" icon={<Plus className="size-4" />} onClick={() => setEdit({ role: 'user', allLibraries: true, hideUnrated: false, maxAge: null })}>Nouveau profil</Button>
      </div>
      <ul className="space-y-2">
        {users.data.map((u) => (
          <li key={u.id} className="flex items-center gap-4 rounded-2xl bg-night/70 ring-1 ring-white/[0.06] px-5 py-4">
            <div className="grid size-10 place-items-center rounded-full bg-white/[0.06] text-aurora">
              {u.role === 'admin' ? <Shield className="size-5" /> : u.role === 'child' ? <Baby className="size-5" /> : <UserRound className="size-5" />}
            </div>
            <div className="flex-1 min-w-0">
              <div className="font-medium">{u.username} {u.id === me?.id && <span className="text-xs text-ash">(vous)</span>}</div>
              <div className="text-xs text-ash">
                {ROLE[u.role]}{u.maxAge != null ? ` · jusqu'à ${u.maxAge} ans` : ''}{u.hideUnrated ? ' · non classés masqués' : ''}
                {!u.allLibraries && u.role !== 'admin' ? ` · ${u.libraries?.length ?? 0} bibliothèque(s)` : ''}
              </div>
            </div>
            <Button size="sm" variant="ghost" aria-label="Modifier" onClick={() => setEdit(u)}><Pencil className="size-4" /></Button>
            {u.id !== me?.id && <Button size="sm" variant="ghost" aria-label="Supprimer" onClick={() => setDel(u)}><Trash2 className="size-4" /></Button>}
          </li>
        ))}
      </ul>
      {edit && <UserModal u={edit} libs={libs.data} onClose={() => setEdit(undefined)} onDone={() => { setEdit(undefined); users.reload() }} />}
      <Modal open={!!del} onClose={() => setDel(undefined)} title="Supprimer ce profil ?">
        <p className="text-mist mb-6">Le profil « {del?.username} », son historique et ses listes seront supprimés. Les vidéos ne sont pas concernées.</p>
        <div className="flex justify-end gap-2"><Button variant="ghost" onClick={() => setDel(undefined)}>Annuler</Button><Button variant="danger" onClick={remove}>Supprimer</Button></div>
      </Modal>
    </div>
  )
}

function UserModal({ u, libs, onClose, onDone }: { u: Partial<U> & { password?: string }; libs: Lib[]; onClose: () => void; onDone: () => void }) {
  const [v, setV] = useState({ ...u, password: '', libraries: u.libraries ?? [] })
  const [error, setError] = useState<string>()
  const [busy, setBusy] = useState(false)
  const isNew = !u.id
  const save = async () => {
    setBusy(true)
    setError(undefined)
    const body = { username: v.username, password: v.password, role: v.role, maxAge: v.maxAge, hideUnrated: v.hideUnrated, allLibraries: v.allLibraries, libraries: v.libraries }
    try {
      if (isNew) await api.post('/admin/users', body)
      else await api.put(`/admin/users/${u.id}`, body)
      onDone()
    } catch (e) {
      setError(e instanceof ApiError ? e.message : String(e))
    }
    setBusy(false)
  }
  return (
    <Modal open onClose={onClose} title={isNew ? 'Nouveau profil' : `Modifier ${u.username}`}>
      <div className="space-y-4">
        {isNew && <Field label="Nom d'utilisateur" value={v.username ?? ''} onChange={(e) => setV({ ...v, username: e.target.value })} />}
        <Field label={isNew ? 'Mot de passe' : 'Nouveau mot de passe (laisser vide pour ne pas changer)'} type="password" autoComplete="new-password"
          value={v.password} onChange={(e) => setV({ ...v, password: e.target.value })} />
        <Select label="Type de profil" value={v.role} className="w-full" onChange={(e) => {
          const role = e.target.value as U['role']
          setV({ ...v, role, ...(role === 'child' && v.maxAge == null ? { maxAge: 10, hideUnrated: true } : {}) })
        }}>
          <option value="user">Utilisateur</option><option value="child">Enfant</option><option value="admin">Administrateur</option>
        </Select>
        {v.role !== 'admin' && (
          <>
            <Select label="Classification maximale" className="w-full" value={v.maxAge == null ? '' : String(v.maxAge)}
              onChange={(e) => setV({ ...v, maxAge: e.target.value === '' ? null : Number(e.target.value) })}>
              {AGES.map(([val, l]) => <option key={val} value={val}>{l}</option>)}
            </Select>
            {v.maxAge != null && (
              <Toggle checked={!!v.hideUnrated} onChange={(x) => setV({ ...v, hideUnrated: x })} label="Masquer les contenus sans classification"
                hint="Recommandé pour un enfant : sans métadonnées, AURORA ne peut pas connaître l'âge conseillé." />
            )}
            <Toggle checked={!!v.allLibraries} onChange={(x) => setV({ ...v, allLibraries: x })} label="Accès à toutes les bibliothèques" />
            {!v.allLibraries && (
              <div className="space-y-1.5">
                {libs.map((l) => {
                  const on = v.libraries.includes(l.id)
                  return (
                    <label key={l.id} className={cx('flex items-center gap-3 rounded-lg px-3 py-2 ring-1 cursor-pointer', on ? 'ring-aurora/40' : 'ring-graphite')}>
                      <input type="checkbox" checked={on} onChange={() => setV({ ...v, libraries: on ? v.libraries.filter((x) => x !== l.id) : [...v.libraries, l.id] })} />
                      <span className="text-sm">{l.name}</span>
                    </label>
                  )
                })}
              </div>
            )}
            <Notice tone="warn">Le contrôle parental s'appuie sur les classifications TMDB : il n'est pas infaillible si les métadonnées sont absentes ou erronées.</Notice>
          </>
        )}
        {error && <Notice tone="error">{error}</Notice>}
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={onClose}>Annuler</Button>
          <Button variant="primary" busy={busy} onClick={save}>{isNew ? 'Créer' : 'Enregistrer'}</Button>
        </div>
      </div>
    </Modal>
  )
}
