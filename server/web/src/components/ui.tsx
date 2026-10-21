// Composants de base du système de design AURORA.
import {
  createContext, forwardRef, useCallback, useContext, useEffect, useRef, useState,
  type ButtonHTMLAttributes, type InputHTMLAttributes, type ReactNode, type SelectHTMLAttributes,
} from 'react'
import { AlertTriangle, CheckCircle2, Info, Loader2, X } from 'lucide-react'

const cx = (...c: (string | false | null | undefined)[]) => c.filter(Boolean).join(' ')
export { cx }

type Variant = 'primary' | 'secondary' | 'ghost' | 'danger'

export const Button = forwardRef<HTMLButtonElement, ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: Variant; size?: 'sm' | 'md' | 'lg'; icon?: ReactNode; busy?: boolean
}>(function Button({ variant = 'secondary', size = 'md', icon, busy, className, children, disabled, ...rest }, ref) {
  const base = 'inline-flex items-center justify-center gap-2 rounded-full font-semibold transition-[background,color,transform,box-shadow] duration-200 active:scale-[0.97] disabled:opacity-45 disabled:pointer-events-none select-none whitespace-nowrap'
  const sizes = { sm: 'h-8 px-3.5 text-[13px]', md: 'h-10 px-5 text-sm', lg: 'h-12 px-7 text-[15px]' }
  const variants: Record<Variant, string> = {
    primary: 'bg-paper text-ink hover:bg-white shadow-[0_0_0_0_rgb(126_232_198/0)] hover:shadow-[0_0_32px_-4px_rgb(126_232_198/0.55)]',
    secondary: 'bg-white/[0.08] text-paper hover:bg-white/[0.14] ring-1 ring-inset ring-white/10',
    ghost: 'text-mist hover:text-paper hover:bg-white/[0.06]',
    danger: 'bg-danger/15 text-danger hover:bg-danger/25 ring-1 ring-inset ring-danger/30',
  }
  return (
    <button ref={ref} className={cx(base, sizes[size], variants[variant], className)} disabled={disabled || busy} {...rest}>
      {busy ? <Loader2 className="size-4 animate-spin" aria-hidden /> : icon}
      {children}
    </button>
  )
})

export function IconButton({ label, className, children, ...rest }: ButtonHTMLAttributes<HTMLButtonElement> & { label: string }) {
  return (
    <button aria-label={label} title={label}
      className={cx('inline-grid place-items-center size-10 rounded-full text-mist hover:text-paper hover:bg-white/10 transition-colors', className)} {...rest}>
      {children}
    </button>
  )
}

export function Spinner({ label = 'Chargement' }: { label?: string }) {
  return (
    <div role="status" className="flex items-center gap-3 text-ash text-sm">
      <Loader2 className="size-5 animate-spin text-aurora" aria-hidden />
      <span>{label}…</span>
    </div>
  )
}

export function PageLoader() {
  return <div className="min-h-[60vh] grid place-items-center"><Spinner /></div>
}

export function Skeleton({ className }: { className?: string }) {
  return <div className={cx('skeleton rounded-[var(--radius-card)]', className)} aria-hidden />
}

export function EmptyState({ icon, title, children, action }: { icon?: ReactNode; title: string; children?: ReactNode; action?: ReactNode }) {
  return (
    <div className="mx-auto max-w-md text-center py-20 px-6 rise">
      {icon && <div className="mx-auto mb-5 grid size-14 place-items-center rounded-full bg-white/[0.05] text-aurora">{icon}</div>}
      <h2 className="font-display text-2xl mb-2">{title}</h2>
      {children && <div className="text-ash leading-relaxed">{children}</div>}
      {action && <div className="mt-6">{action}</div>}
    </div>
  )
}

export function ErrorState({ error, retry }: { error: { message: string }; retry?: () => void }) {
  return (
    <div role="alert" className="mx-auto max-w-lg my-16 rounded-2xl bg-danger/[0.07] ring-1 ring-danger/25 p-6 flex gap-4">
      <AlertTriangle className="size-6 text-danger shrink-0 mt-0.5" aria-hidden />
      <div className="space-y-3">
        <p className="text-paper leading-relaxed">{error.message}</p>
        {retry && <Button size="sm" onClick={retry}>Réessayer</Button>}
      </div>
    </div>
  )
}

export function Notice({ tone = 'info', children }: { tone?: 'info' | 'warn' | 'error'; children: ReactNode }) {
  const styles = {
    info: 'bg-aurora/[0.06] ring-aurora/20 text-mist',
    warn: 'bg-ember/[0.08] ring-ember/25 text-mist',
    error: 'bg-danger/[0.08] ring-danger/25 text-paper',
  }
  const Icon = tone === 'info' ? Info : AlertTriangle
  const color = { info: 'text-aurora', warn: 'text-ember', error: 'text-danger' }[tone]
  return (
    <div className={cx('flex gap-3 rounded-xl p-4 ring-1 text-sm leading-relaxed', styles[tone])}>
      <Icon className={cx('size-5 shrink-0', color)} aria-hidden />
      <div>{children}</div>
    </div>
  )
}

export function ProgressBar({ value, className, label }: { value: number; className?: string; label?: string }) {
  const v = Math.max(0, Math.min(1, value))
  return (
    <div className={cx('h-1 rounded-full bg-white/15 overflow-hidden', className)} role="progressbar"
      aria-valuemin={0} aria-valuemax={100} aria-valuenow={Math.round(v * 100)} aria-label={label ?? 'Progression'}>
      <div className="h-full bg-aurora rounded-full transition-[width] duration-500" style={{ width: `${v * 100}%` }} />
    </div>
  )
}

export function Badge({ children, tone = 'plain' }: { children: ReactNode; tone?: 'plain' | 'accent' | 'warn' }) {
  const t = { plain: 'ring-white/20 text-mist', accent: 'ring-aurora/40 text-aurora', warn: 'ring-ember/40 text-ember' }[tone]
  return <span className={cx('inline-flex items-center h-5 px-1.5 rounded text-[10.5px] font-semibold tracking-wide ring-1 ring-inset', t)}>{children}</span>
}

export const Field = forwardRef<HTMLInputElement, InputHTMLAttributes<HTMLInputElement> & { label: string; hint?: ReactNode }>(
  function Field({ label, hint, className, id, ...rest }, ref) {
    const fid = id ?? `f-${label.replace(/\W+/g, '-').toLowerCase()}`
    return (
      <div className={cx('space-y-1.5', className)}>
        <label htmlFor={fid} className="block text-[13px] font-medium text-mist">{label}</label>
        <input ref={ref} id={fid}
          className="w-full h-11 rounded-lg bg-coal px-3.5 text-paper ring-1 ring-inset ring-graphite placeholder:text-ash/70 focus:ring-aurora/70 focus:outline-none transition-shadow"
          {...rest} />
        {hint && <p className="text-xs text-ash leading-relaxed">{hint}</p>}
      </div>
    )
  })

export function Select({ label, className, children, ...rest }: SelectHTMLAttributes<HTMLSelectElement> & { label?: string }) {
  return (
    <label className={cx('inline-flex flex-col gap-1.5', className)}>
      {label && <span className="text-[13px] font-medium text-mist">{label}</span>}
      <select className="h-10 rounded-lg bg-coal pl-3 pr-8 text-sm text-paper ring-1 ring-inset ring-graphite focus:ring-aurora/70 focus:outline-none" {...rest}>
        {children}
      </select>
    </label>
  )
}

export function Toggle({ checked, onChange, label, hint }: { checked: boolean; onChange: (v: boolean) => void; label: string; hint?: string }) {
  return (
    <label className="flex items-start gap-3 cursor-pointer select-none">
      <button type="button" role="switch" aria-checked={checked} onClick={() => onChange(!checked)}
        className={cx('mt-0.5 relative h-6 w-10 rounded-full transition-colors shrink-0', checked ? 'bg-aurora-deep' : 'bg-smoke')}>
        <span className={cx('absolute top-0.5 size-5 rounded-full bg-paper transition-transform', checked ? 'translate-x-[18px]' : 'translate-x-0.5')} />
      </button>
      <span>
        <span className="block text-sm text-paper">{label}</span>
        {hint && <span className="block text-xs text-ash mt-0.5 leading-relaxed">{hint}</span>}
      </span>
    </label>
  )
}

/** Fenêtre modale basée sur <dialog> (piège du focus et touche Échap natifs). */
export function Modal({ open, onClose, title, children, wide }: { open: boolean; onClose: () => void; title: string; children: ReactNode; wide?: boolean }) {
  const ref = useRef<HTMLDialogElement>(null)
  useEffect(() => {
    const d = ref.current
    if (!d) return
    if (open && !d.open) d.showModal()
    if (!open && d.open) d.close()
  }, [open])
  return (
    <dialog ref={ref} onClose={onClose} onClick={(e) => e.target === ref.current && onClose()}
      className={cx('m-auto w-[calc(100%-32px)] rounded-2xl bg-night text-paper ring-1 ring-white/10 shadow-[var(--shadow-lift)] p-0', wide ? 'max-w-3xl' : 'max-w-lg')}>
      {open && (
        <div className="p-6 sm:p-7">
          <div className="flex items-start justify-between gap-4 mb-5">
            <h2 className="font-display text-2xl leading-tight">{title}</h2>
            <IconButton label="Fermer" onClick={onClose} className="-mr-2 -mt-1"><X className="size-5" /></IconButton>
          </div>
          {children}
        </div>
      )}
    </dialog>
  )
}

// --- Notifications -------------------------------------------------------------------------------

interface Toast { id: number; tone: 'ok' | 'error' | 'info'; text: string }
const ToastCtx = createContext<(tone: Toast['tone'], text: string) => void>(() => {})

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])
  const push = useCallback((tone: Toast['tone'], text: string) => {
    const id = Date.now() + Math.random()
    setToasts((t) => [...t.slice(-2), { id, tone, text }])
    setTimeout(() => setToasts((t) => t.filter((x) => x.id !== id)), tone === 'error' ? 7000 : 3500)
  }, [])
  return (
    <ToastCtx.Provider value={push}>
      {children}
      <div aria-live="polite" className="fixed bottom-5 left-1/2 -translate-x-1/2 z-[90] flex flex-col items-center gap-2 w-[min(92vw,460px)]">
        {toasts.map((t) => (
          <div key={t.id} className="rise flex items-center gap-3 rounded-full bg-coal/95 backdrop-blur px-5 py-3 text-sm ring-1 ring-white/10 shadow-[var(--shadow-lift)]">
            {t.tone === 'ok' && <CheckCircle2 className="size-4 text-aurora shrink-0" />}
            {t.tone === 'error' && <AlertTriangle className="size-4 text-danger shrink-0" />}
            {t.tone === 'info' && <Info className="size-4 text-mist shrink-0" />}
            <span>{t.text}</span>
          </div>
        ))}
      </div>
    </ToastCtx.Provider>
  )
}

export const useToast = () => useContext(ToastCtx)

export function Logo({ className }: { className?: string }) {
  return (
    <span className={cx('inline-flex items-center gap-2.5', className)}>
      <svg viewBox="0 0 64 64" className="size-7" aria-hidden>
        <defs>
          <linearGradient id="lg" x1="0" y1="1" x2="1" y2="0"><stop offset="0" stopColor="#2fbf98" /><stop offset="1" stopColor="#d4faec" /></linearGradient>
        </defs>
        <path d="M12 48 L32 12 L52 48" fill="none" stroke="url(#lg)" strokeWidth="5.5" strokeLinecap="round" strokeLinejoin="round" />
        <path d="M20 37 Q32 30 44 37" fill="none" stroke="url(#lg)" strokeWidth="4" strokeLinecap="round" opacity=".7" />
      </svg>
      <span className="font-display text-[19px] tracking-[0.2em] font-semibold">AURORA</span>
    </span>
  )
}
