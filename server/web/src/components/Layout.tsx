import { useEffect, useRef, useState } from 'react'
import { Link, NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { Bookmark, Clapperboard, House, LogOut, Search, Settings2, Shield, Tv, UserRound } from 'lucide-react'
import { useSession } from '../lib/session'
import { Logo, cx } from './ui'

const links = [
  { to: '/', label: 'Accueil', end: true },
  { to: '/movies', label: 'Films' },
  { to: '/series', label: 'Séries' },
  { to: '/collections', label: 'Collections' },
  { to: '/mylist', label: 'Ma liste' },
]

export function Layout() {
  const { user, logout } = useSession()
  const nav = useNavigate()
  const loc = useLocation()
  const [scrolled, setScrolled] = useState(false)
  const [menu, setMenu] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)
  useEffect(() => {
    const on = () => setScrolled(window.scrollY > 24)
    on()
    window.addEventListener('scroll', on, { passive: true })
    return () => window.removeEventListener('scroll', on)
  }, [])
  useEffect(() => setMenu(false), [loc.pathname])
  useEffect(() => {
    if (!menu) return
    const close = (e: MouseEvent) => !menuRef.current?.contains(e.target as Node) && setMenu(false)
    const esc = (e: KeyboardEvent) => e.key === 'Escape' && setMenu(false)
    document.addEventListener('mousedown', close)
    document.addEventListener('keydown', esc)
    return () => { document.removeEventListener('mousedown', close); document.removeEventListener('keydown', esc) }
  }, [menu])
  // Raccourci « / » : recherche.
  useEffect(() => {
    const on = (e: KeyboardEvent) => {
      const t = e.target as HTMLElement
      if (e.key === '/' && !['INPUT', 'TEXTAREA', 'SELECT'].includes(t.tagName)) { e.preventDefault(); nav('/search') }
    }
    window.addEventListener('keydown', on)
    return () => window.removeEventListener('keydown', on)
  }, [nav])

  return (
    <>
      <a href="#main" className="sr-only focus:not-sr-only focus:fixed focus:top-3 focus:left-3 focus:z-[95] focus:bg-paper focus:text-ink focus:px-4 focus:py-2 focus:rounded-full">Aller au contenu</a>
      <header className={cx('fixed top-0 inset-x-0 z-50 transition-[background,backdrop-filter,box-shadow] duration-300',
        scrolled ? 'bg-ink/85 backdrop-blur-md shadow-[0_1px_0_rgb(255_255_255/0.05)]' : 'bg-gradient-to-b from-ink/80 to-transparent')}>
        <div className="flex items-center gap-4 sm:gap-8 h-16 px-[var(--gutter)]">
          <Link to="/" aria-label="AURORA — accueil" className="shrink-0"><Logo /></Link>
          <nav className="hidden sm:block flex-1 min-w-0 overflow-x-auto [scrollbar-width:none]" aria-label="Navigation principale">
            <ul className="flex items-center gap-1 sm:gap-2">
              {links.map((l) => (
                <li key={l.to}>
                  <NavLink to={l.to} end={l.end}
                    className={({ isActive }) => cx('relative px-2.5 py-2 text-[14px] whitespace-nowrap transition-colors',
                      isActive ? 'text-paper after:absolute after:left-2.5 after:right-2.5 after:-bottom-0.5 after:h-px after:bg-aurora' : 'text-ash hover:text-paper')}>
                    {l.label}
                  </NavLink>
                </li>
              ))}
            </ul>
          </nav>
          <div className="flex-1 sm:hidden" />
          <Link to="/search" aria-label="Rechercher (raccourci /)" className="grid size-10 place-items-center rounded-full text-mist hover:text-paper hover:bg-white/10">
            <Search className="size-5" />
          </Link>
          <div className="relative" ref={menuRef}>
            <button aria-haspopup="menu" aria-expanded={menu} onClick={() => setMenu((m) => !m)}
              className="grid size-9 place-items-center rounded-full bg-gradient-to-br from-aurora/80 to-aurora-deep text-ink font-bold text-sm uppercase ring-2 ring-transparent hover:ring-aurora/40"
              aria-label={`Profil de ${user?.username}`}>
              {user?.username.slice(0, 1)}
            </button>
            {menu && (
              <div role="menu" className="rise absolute right-0 mt-2 w-60 rounded-xl bg-coal/98 backdrop-blur ring-1 ring-white/10 shadow-[var(--shadow-lift)] p-1.5 text-sm">
                <div className="px-3 py-2.5 border-b border-white/5 mb-1">
                  <div className="font-medium">{user?.username}</div>
                  <div className="text-xs text-ash">{user?.role === 'admin' ? 'Administrateur' : user?.role === 'child' ? 'Profil enfant' : 'Utilisateur'}</div>
                </div>
                <MenuLink to="/profile" icon={<UserRound className="size-4" />}>Mon profil</MenuLink>
                {user?.role === 'admin' && <MenuLink to="/admin" icon={<Shield className="size-4" />}>Administration</MenuLink>}
                <MenuLink to="/profile#home" icon={<Settings2 className="size-4" />}>Personnaliser l'accueil</MenuLink>
                <MenuLink to="/link" icon={<Tv className="size-4" />}>Associer une TV</MenuLink>
                <button role="menuitem" onClick={() => logout().then(() => nav('/'))}
                  className="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-mist hover:bg-white/[0.06] hover:text-paper">
                  <LogOut className="size-4" /> Se déconnecter
                </button>
              </div>
            )}
          </div>
        </div>
      </header>
      <main id="main" className="min-h-dvh pb-28 sm:pb-24">
        <Outlet />
      </main>
      {/* Mobile : onglets en bas d'écran, à portée de pouce. */}
      <nav className="sm:hidden fixed bottom-0 inset-x-0 z-50 bg-ink/92 backdrop-blur-md border-t border-white/[0.06] pb-[env(safe-area-inset-bottom)]" aria-label="Navigation">
        <ul className="grid grid-cols-4">
          {[
            { to: '/', label: 'Accueil', icon: House, end: true },
            { to: '/movies', label: 'Films', icon: Clapperboard },
            { to: '/series', label: 'Séries', icon: Tv },
            { to: '/mylist', label: 'Ma liste', icon: Bookmark },
          ].map((l) => (
            <li key={l.to}>
              <NavLink to={l.to} end={l.end}
                className={({ isActive }) => cx('flex flex-col items-center gap-1 py-2.5 text-[11px]', isActive ? 'text-aurora' : 'text-ash')}>
                <l.icon className="size-5" />{l.label}
              </NavLink>
            </li>
          ))}
        </ul>
      </nav>
    </>
  )
}

function MenuLink({ to, icon, children }: { to: string; icon: React.ReactNode; children: React.ReactNode }) {
  return (
    <Link role="menuitem" to={to} className="flex items-center gap-3 px-3 py-2.5 rounded-lg text-mist hover:bg-white/[0.06] hover:text-paper">
      {icon} {children}
    </Link>
  )
}

/** En-tête de page (pages sans bandeau). */
export function PageHeader({ eyebrow, title, children }: { eyebrow?: string; title: string; children?: React.ReactNode }) {
  return (
    <div className="px-[var(--gutter)] pt-28 pb-8 flex flex-wrap items-end justify-between gap-6">
      <div>
        {eyebrow && <p className="eyebrow mb-3">{eyebrow}</p>}
        <h1 className="title-xl text-[40px] sm:text-[56px]">{title}</h1>
      </div>
      {children}
    </div>
  )
}
