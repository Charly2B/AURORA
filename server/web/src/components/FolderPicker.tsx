import { useState } from 'react'
import { ChevronRight, CornerLeftUp, Folder, FolderLock, HardDrive } from 'lucide-react'
import { useApi } from '../lib/hooks'
import { ErrorState, Spinner, cx } from './ui'

interface BrowseResp {
  path: string
  parent: string | null
  entries: { name: string; path: string; readable: boolean }[]
}

/** Explorateur de dossiers du NAS (lecture seule), limité aux volumes autorisés. */
export function FolderPicker({ value, onChange }: { value: string; onChange: (p: string) => void }) {
  const [cwd, setCwd] = useState('')
  const { data, error, loading, reload } = useApi<BrowseResp>(`/admin/browse?path=${encodeURIComponent(cwd)}`)
  return (
    <div className="rounded-xl ring-1 ring-graphite bg-night overflow-hidden">
      <div className="flex items-center gap-2 px-3 h-11 border-b border-white/5 text-sm">
        <HardDrive className="size-4 text-ash shrink-0" />
        <span className="truncate text-mist font-mono text-[12.5px]">{cwd || 'Volumes'}</span>
      </div>
      <div className="max-h-72 overflow-y-auto">
        {loading && <div className="p-4"><Spinner /></div>}
        {error && <ErrorState error={error} retry={reload} />}
        {data && !loading && (
          <ul>
            {cwd && (
              <li>
                <button onClick={() => setCwd(data.parent ?? '')} className="w-full flex items-center gap-3 px-3 py-2.5 text-sm text-ash hover:bg-white/[0.04]">
                  <CornerLeftUp className="size-4" /> Dossier parent
                </button>
              </li>
            )}
            {data.entries.length === 0 && <li className="px-3 py-4 text-sm text-ash">Aucun sous-dossier.</li>}
            {data.entries.map((e) => (
              <li key={e.path} className={cx('flex items-center', value === e.path && 'bg-aurora/[0.08]')}>
                <button onClick={() => onChange(e.path)} disabled={!e.readable}
                  className="flex-1 min-w-0 flex items-center gap-3 px-3 py-2.5 text-sm text-left hover:bg-white/[0.04] disabled:opacity-60"
                  title={e.readable ? 'Choisir ce dossier' : "AURORA n'a pas le droit de lire ce dossier"}>
                  {e.readable ? <Folder className={cx('size-4 shrink-0', value === e.path ? 'text-aurora' : 'text-mist')} /> : <FolderLock className="size-4 shrink-0 text-ember" />}
                  <span className="truncate">{e.name}</span>
                  {!e.readable && <span className="ml-auto text-[11px] text-ember shrink-0">accès refusé</span>}
                </button>
                {e.readable && (
                  <button onClick={() => setCwd(e.path)} aria-label={`Ouvrir ${e.name}`} className="px-3 py-2.5 text-ash hover:text-paper">
                    <ChevronRight className="size-4" />
                  </button>
                )}
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}
