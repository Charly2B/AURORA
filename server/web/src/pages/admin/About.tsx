import { useSession } from '../../lib/session'
import { Logo } from '../../components/ui'

export function About() {
  const { status } = useSession()
  return (
    <div className="max-w-2xl space-y-6 text-mist leading-relaxed">
      <Logo />
      <p>AURORA {status?.version} — médiathèque personnelle installée nativement sur votre NAS Synology. Aucun compte en ligne, aucun abonnement, aucune publicité.</p>
      <section className="space-y-2">
        <h2 className="font-display text-2xl text-paper">Données de films et séries</h2>
        <p className="text-sm">Lorsque vous configurez une clé, affiches, résumés et distributions proviennent de The Movie Database (TMDB).</p>
        <p className="text-sm text-paper">This product uses the TMDB API but is not endorsed or certified by TMDB.</p>
      </section>
      <section className="space-y-2">
        <h2 className="font-display text-2xl text-paper">Logiciels tiers</h2>
        <ul className="text-sm space-y-1 list-disc pl-5">
          <li>FFmpeg (analyse, remux et conversion) — licence GPL v3 pour la version embarquée dans le paquet. Sources : ffmpeg.org.</li>
          <li>SQLite (via modernc.org/sqlite) — domaine public / BSD.</li>
          <li>React, React Router, hls.js, Lucide — licences MIT / Apache 2.0 / ISC.</li>
          <li>Polices Fraunces et Instrument Sans — SIL Open Font License.</li>
        </ul>
        <p className="text-xs text-ash">Le détail des licences est fourni dans le fichier THIRD_PARTY_NOTICES livré avec le paquet.</p>
      </section>
      <section className="space-y-2">
        <h2 className="font-display text-2xl text-paper">Raccourcis du lecteur</h2>
        <dl className="grid grid-cols-[auto_1fr] gap-x-6 gap-y-1 text-sm">
          <dt className="text-paper">Espace / K</dt><dd>Lecture / pause</dd>
          <dt className="text-paper">← / →</dt><dd>Reculer / avancer de 10 s</dd>
          <dt className="text-paper">↑ / ↓</dt><dd>Volume</dd>
          <dt className="text-paper">M</dt><dd>Couper le son</dd>
          <dt className="text-paper">F</dt><dd>Plein écran</dd>
          <dt className="text-paper">C</dt><dd>Changer de sous-titres</dd>
          <dt className="text-paper">/</dt><dd>Rechercher (hors lecteur)</dd>
        </dl>
      </section>
    </div>
  )
}
