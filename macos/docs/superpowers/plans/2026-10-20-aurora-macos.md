# AURORA macOS Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Construire un client macOS natif AURORA, à parité avec le client Android TV, utilisant libmpv intégré et livré sous forme d’application installée et de DMG.

**Architecture:** Application SwiftUI pilotée par des modèles `@Observable`, client REST `URLSession` et session par NAS dans le Trousseau. Le lecteur isole libmpv derrière un protocole testable ; un coordinateur conserve les règles direct/remux/HLS/conversion du client Android sans lecteur externe.

**Tech Stack:** Swift 6, SwiftUI/AppKit, Swift Testing, URLSession, Security.framework, XcodeGen 2.46, MPVKit 1.0.0 LGPL/libmpv 0.41, Xcode 26.5, macOS 15+.

**Spec:** `/Users/remyj/Documents/Codex/2026-10-20/tr/work/docs/superpowers/specs/2026-10-20-aurora-macos-design.md`

## Global Constraints

- Projet : `/Users/remyj/Documents/Xcode/AuroraMac`; produit et scheme : `AURORA`; bundle ID : `com.remyj.AURORA`.
- Cible macOS 15.0+, Swift 6, interface et messages en français.
- Parité fonctionnelle avec `/Users/remyj/Documents/AndroidStudio/AuroraTV`; API inchangée dans `/Users/remyj/Documents/Codex/2026-10-16/aurora`.
- libmpv est embarqué via le produit LGPL `MPVKit` 1.0.0 épinglé ; aucune dépendance Homebrew à l’exécution.
- Aucun VLC, lecteur externe, téléchargement local, mode hors-ligne ou administration serveur.
- Jamais de mot de passe, cookie ou secret de suivi dans une URL, `UserDefaults`, un log ou une erreur affichée.
- Toute requête modifiante porte `X-Aurora: 1`; le cookie s’appelle `aurora_session`.
- L’app n’écrit, ne déplace et ne supprime aucun fichier vidéo.
- Après ajout/retrait de Swift : `xcodegen generate`; avant livraison : tests + build Release.
- Après build final réussi : sauvegarder l’ancienne app, installer dans `/Users/remyj/Applications/MesApps/AURORA.app`, vérifier la copie et lancer.
- DMG final : `/Users/remyj/Documents/Codex/2026-10-20/tr/outputs/AURORA-0.1.0.dmg`.

## Review Focus

- Adresse IPv6 avec ou sans crochets/port : normalisation vers une origine HTTP valide sans doublon — test Task 1.
- Réponse HTML ou corps réseau coupé : erreur française affichable, jamais `DecodingError` non traité — test Task 2.
- Pagination dont l’ordre glisse et répète une œuvre : dédoublonnage stable sans carte SwiftUI dupliquée — test Task 4.
- Deux `/play` répondent dans le désordre : seule la génération récente est jouée, l’ancienne session HLS est arrêtée — test Task 7.
- Fermeture pendant `/play` ou progression : la session créée tardivement est arrêtée et la dernière position est envoyée sans bloquer l’UI — test Task 7.

---

## File Map

- `project.yml`: source XcodeGen, cible app/tests, réglages macOS et dépendance MPVKit.
- `AURORA/App/*`: entrée, état de session, navigation et composition des dépendances.
- `AURORA/Domain/*`: modèles API, adresses NAS et formats d’affichage purs.
- `AURORA/Networking/*`: transport HTTP, client AURORA, images authentifiées et erreurs.
- `AURORA/Security/*`: abstraction et implémentation Trousseau.
- `AURORA/Features/Setup/*`: NAS enregistrés, ajout et connexion code/mot de passe.
- `AURORA/Features/Catalog/*`: accueil, bibliothèques, recherche et mémoire de navigation.
- `AURORA/Features/Detail/*`: fiche, saisons, épisodes et actions d’état.
- `AURORA/Features/Profile/*`: profil, NAS courant, déconnexion.
- `AURORA/Playback/*`: pont libmpv, vue AppKit, capacités et coordinateur de lecture.
- `AURORA/Features/Player/*`: écran, contrôles, menus de pistes/chapitres et épisode suivant.
- `AURORA/DesignSystem/*`: thème, images, cartes, rangées et composants.
- `AURORATests/*`: tests purs, mocks réseau/API/mpv et fixtures JSON.
- `Scripts/*`: build Release, DMG, installation/sauvegarde/vérification/lancement.
- `Resources/ThirdPartyNotices.md`: licences MPVKit/libmpv/FFmpeg et liens sources.

### Task 1: Projet, modèles API et adresses NAS

**Files:**
- Create: `AGENTS.md`, `README.md`, `project.yml`, `.gitignore`
- Create: `AURORA/App/AURORAApp.swift`, `AURORA/App/RootView.swift`, `AURORA/Info.plist`
- Create: `AURORA/Domain/APIModels.swift`, `AURORA/Domain/ServerAddress.swift`
- Create: `AURORATests/Fixtures/status.json`, `AURORATests/Fixtures/home.json`, `AURORATests/DomainModelsTests.swift`, `AURORATests/ServerAddressTests.swift`
- Create: `docs/superpowers/specs/2026-10-20-aurora-macos-design.md`, `docs/superpowers/plans/2026-10-20-aurora-macos.md`

**Interfaces:**
- Produces: `struct ServerAddress: Codable, Hashable, Identifiable { let url: URL; var id: String { origin }; var origin: String { get }; static func normalize(_ input: String) throws -> ServerAddress }`
- Produces: API `Codable, Sendable, Identifiable` types mirroring Android: `Card`, `Section`, `Home`, `LibraryRef`, `ListPage`, `AudioTrack`, `SubtitleTrack`, `MediaFile`, `Person`, `ItemState`, `Season`, `Detail`, `Chapter`, `PlaySubtitle`, `PlayResponse`, `SearchResults`, `Preferences`, `UserProfile`, `Status`, `Capabilities`, `LinkStart`, `PlayRequest`.

- [ ] **Step 1: Scaffold the XcodeGen project and repository instructions**

Set macOS 15.0, Swift 6, French development language, app + Swift Testing targets, `ENABLE_APP_SANDBOX: NO`, generated Info.plist keys for display name `AURORA`, category `public.app-category.entertainment`, and ATS local HTTP. Copy the approved spec/plan into `docs/superpowers`, then initialize Git.

- [ ] **Step 2: Write failing address and decoding tests**

Assert `nas`, `192.168.1.37`, `https://nas.local:9443`, `fe80::1`, and `[fe80::1]:9000` normalize respectively with default `http` and port `8890`; whitespace/trailing slash disappears; duplicates share `id`; empty/path-bearing inputs throw. Decode fixtures with omitted optional fields and unknown fields.

- [ ] **Step 3: Generate the project and verify RED**

Run: `xcodegen generate && xcodebuild -project AURORA.xcodeproj -scheme AURORA -destination 'platform=macOS' test`

Expected: tests fail because `ServerAddress` and API models are missing.

- [ ] **Step 4: Implement models and normalization**

Use `URLComponents`; bracket raw IPv6 before applying the default port; accept only `http`/`https`; reject credentials, query, fragment and non-empty path. Keep JSON keys camelCase via `JSONDecoder` defaults and give server-omitted fields Android-equivalent defaults.

- [ ] **Step 5: Verify GREEN and commit**

Run the Task 1 test command; expected all tests pass and Debug app launches to a placeholder root. Commit: `feat: scaffold Aurora macOS domain`.

### Task 2: Transport, sessions, Trousseau et API AURORA

**Files:**
- Create: `AURORA/Networking/APIError.swift`, `AURORA/Networking/HTTPTransport.swift`, `AURORA/Networking/AuroraAPI.swift`, `AURORA/Networking/ImageRepository.swift`
- Create: `AURORA/Security/SessionStore.swift`, `AURORA/Security/KeychainSessionStore.swift`
- Create: `AURORATests/MockURLProtocol.swift`, `AURORATests/NetworkingTests.swift`, `AURORATests/KeychainSessionStoreTests.swift`

**Interfaces:**
- Consumes: `ServerAddress` and API types from Task 1.
- Produces: `protocol SessionStoring: Sendable { func token(for origin: String) async throws -> String?; func setToken(_ token: String?, for origin: String) async throws }`.
- Produces: `protocol HTTPTransporting: Sendable { func send(_ request: URLRequest) async throws -> (Data, HTTPURLResponse) }`.
- Produces: `protocol AuroraServing: Sendable` with `status`, `login`, `linkStart`, `linkPoll`, `me`, `logout`, `home`, `libraries`, `list`, `detail`, `episodes`, `next`, `search`, `play`, `progress`, `setState`, `hideContinue`, `stopSession`, `imageData`.
- Produces: `actor AuroraAPI` implementing `AuroraServing`; `func absoluteURL(_ path: String?) -> URL?`; unauthorized callback as an injected `@Sendable () async -> Void`.

- [ ] **Step 1: Write failing transport/session tests**

Cover cookie extraction from multiple `Set-Cookie`, cookie + `X-Aurora` injection, percent-encoded search, POST bodies, 401 callback, server JSON error capitalization, HTML body, connection failure and body-read failure. Keychain test uses a unique test origin and verifies save/load/delete without printing the token.

- [ ] **Step 2: Verify RED**

Run Networking and Keychain suites; expected missing interfaces/types.

- [ ] **Step 3: Implement session storage and transport**

Keychain service is `com.remyj.AURORA.session`, account is normalized origin, accessibility `kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly`. `HTTPTransport` maps `URLError` and unreadable bodies to concise French `APIError` values.

- [ ] **Step 4: Implement the complete API surface**

Prefix paths with `/api/v1`; parse only the `aurora_session` cookie; never request `external: true`; use limit 500. Images use the same authenticated request path and `URLCache`, not `AsyncImage` unauthenticated fetches.

- [ ] **Step 5: Verify GREEN and commit**

Run all tests and `xcodebuild ... build`; expected pass. Commit: `feat: add secure Aurora API client`.

### Task 3: Multi-NAS et connexion à parité Android

**Files:**
- Create: `AURORA/App/AppSession.swift`, `AURORA/App/AppDependencies.swift`
- Create: `AURORA/Features/Setup/ServerStore.swift`, `AURORA/Features/Setup/SetupModel.swift`
- Create: `AURORA/Features/Setup/ServerListView.swift`, `AddServerView.swift`, `LoginView.swift`, `LinkCodeView.swift`
- Create: `AURORATests/ServerStoreTests.swift`, `AURORATests/SetupModelTests.swift`, `AURORATests/Fakes/FakeAuroraAPI.swift`, `AURORATests/Fakes/ImmediateClock.swift`

**Interfaces:**
- Consumes: `AuroraServing`, `SessionStoring`, `ServerAddress`.
- Produces: `@MainActor @Observable final class AppSession` with `servers`, `selectedServer`, `profile`, `phase: AppPhase`, `select`, `authenticated`, `unauthorized`, `logout`.
- Produces: `@MainActor @Observable final class SetupModel` with `addServer`, `beginLink(deviceName:)`, `pollLink()`, `login(username:password:)`, `cancel`; link state preserves the same code across transient network failures until expiry.

- [ ] **Step 1: Write failing store/setup tests**

Assert stable server order, no duplicate normalized origin, delete removes its Keychain token, remembered current server, automatic authenticated status restoration, link polling every 2 seconds, no regenerated code after transient error, 410 expiration, password fallback, and 401 returning only that NAS to login.

- [ ] **Step 2: Verify RED**

Run Setup suites; expected missing state models.

- [ ] **Step 3: Implement stores and session state**

Persist only `[ServerAddress]` and selected origin in `UserDefaults`; token remains exclusively in `SessionStoring`. Use `Host.current().localizedName ?? "iMac"` for link device name.

- [ ] **Step 4: Implement setup screens**

Match Android order: NAS list → add → connect; Back from connection returns to NAS list. Link code is primary, opens `baseURL/link?code=...` in the user’s browser only when explicitly clicked, and password is secondary.

- [ ] **Step 5: Verify GREEN, accessibility labels and commit**

Run all tests/build and inspect setup at 100%/200% scaling. Commit: `feat: add multi-server authentication flow`.

### Task 4: Accueil, bibliothèques, recherche et design system

**Files:**
- Create: `AURORA/DesignSystem/AuroraTheme.swift`, `AuthenticatedImage.swift`, `MediaCardView.swift`, `MediaRowView.swift`, `LoadingStateView.swift`, `FolderPanelView.swift`
- Create: `AURORA/Features/Catalog/CatalogModels.swift`, `HomeModel.swift`, `HomeView.swift`, `LibraryModel.swift`, `LibraryView.swift`, `SearchModel.swift`, `SearchView.swift`, `BrowseMemory.swift`
- Create: `AURORATests/HomeModelTests.swift`, `LibraryModelTests.swift`, `SearchModelTests.swift`, `BrowseMemoryTests.swift`

**Interfaces:**
- Consumes: authenticated `AppSession`, `AuroraServing`, `ImageRepository`.
- Produces: `func loadAll(kind: MediaKind, libraryID: Int64?, sort: LibrarySort, descending: Bool) async throws -> [Card]` using pages of 500 and stable ID deduplication.
- Produces: `@MainActor @Observable` `HomeModel`, `LibraryModel`, `SearchModel`; `BrowseMemory` keyed by `origin + tab + libraryID` storing sort, scroll target and opened item ID.

- [ ] **Step 1: Write failing catalog tests**

Test Android section order, folders never merged, pages 0/500/1000, duplicated boundary ID kept once in first-seen order, empty search, fuzzy label, cancelled old-server response ignored, and browse memory restored then purged when a screen leaves navigation.

- [ ] **Step 2: Verify RED**

Run Catalog suites; expected missing models.

- [ ] **Step 3: Implement catalog state and authenticated images**

Model explicit loading/content/empty/error states. Cancel work on query/server changes. Cache decoded images in `NSCache` and bytes in configured `URLCache`; never include the session token in cache keys.

- [ ] **Step 4: Implement Android-parity browsing UI**

Use dark cinematic theme, hero, horizontal rows, adaptive grids, watched/progress/unwatched badges, folder tabs and macOS-window-style grouped folder panels. Add hover and keyboard focus without changing Android labels/order.

- [ ] **Step 5: Verify GREEN and commit**

Run tests/build; manually resize from 960×640 to full screen and verify no mixed folders. Commit: `feat: add Aurora catalog browsing`.

### Task 5: Fiches, saisons, épisodes et actions

**Files:**
- Create: `AURORA/Features/Detail/DetailModel.swift`, `DetailView.swift`, `SeasonEpisodesView.swift`, `PeopleRowView.swift`, `MediaActionsView.swift`
- Modify: `AURORA/Features/Catalog/HomeView.swift`, `LibraryView.swift`, `SearchView.swift`
- Create: `AURORATests/DetailModelTests.swift`

**Interfaces:**
- Consumes: `AuroraServing.detail/episodes/setState/hideContinue`, `BrowseMemory`.
- Produces: `@MainActor @Observable final class DetailModel` with `load`, `selectSeason`, `toggleFavorite`, `toggleWatchlist`, `setWatched`, and `playIntent(restart:fileID:) -> PlayIntent`.

- [ ] **Step 1: Write failing detail tests**

Test preferred file is `lastFileId` then first file, resume only when position > 0 and not watched, restart at zero, season episode reload, optimistic state rollback on API error, and hide-continue leaves progress unchanged while removing the row after confirmation.

- [ ] **Step 2: Verify RED**

Run Detail suite; expected missing model.

- [ ] **Step 3: Implement detail state and actions**

Refresh affected catalog/detail models after successful mutations. Keep file/version selection explicit when multiple files exist.

- [ ] **Step 4: Implement immersive detail UI**

Backdrop/poster, metadata, overview, cast, play/resume/restart, watched, list, favorite, seasons, episode progress and similar/collection rows. Long-click/right-click Continue card opens the same removal confirmation as Android.

- [ ] **Step 5: Verify GREEN and commit**

Run all tests/build and verify Back restores folder/sort/scroll/card. Commit: `feat: add media detail and state actions`.

### Task 6: libmpv intégré et rendu vidéo

**Files:**
- Modify: `project.yml`
- Create: `AURORA/Playback/MPVControlling.swift`, `LibMPVController.swift`, `MPVRenderView.swift`, `MPVModels.swift`, `PlaybackCapabilities.swift`
- Create: `AURORA/Resources/ThirdPartyNotices.md`
- Create: `AURORATests/PlaybackCapabilitiesTests.swift`, `MPVCommandTests.swift`, `Fakes/FakeMPVController.swift`

**Interfaces:**
- Consumes: MPVKit exact version `1.0.0`, product `MPVKit` (LGPL, never `MPVKit-GPL`).
- Produces: `protocol MPVControlling: AnyObject` with `attach(to:)`, `load(url:headers:start:)`, `setPaused`, `seek(to:)`, `selectAudio(id:)`, `selectSubtitle(id:)`, `addSubtitle(url:)`, `setRate`, `stop`; events as `AsyncStream<MPVEvent>`.
- Produces: `struct MPVTrack: Sendable, Identifiable`; `enum MPVEvent: Sendable { case ready, time(Double), duration(Double), paused(Bool), buffering(Bool), tracks([MPVTrack]), ended, failed(MPVFailure) }`.
- Produces: `PlaybackCapabilities.detect(hardwareDecoder:) -> Capabilities` with containers `mp4,mov,mkv,webm,ts,avi`, HLS `native`, audio codecs bundled by MPVKit, and video codecs gated by VideoToolbox except H.264 fallback.

- [ ] **Step 1: Add MPVKit and write failing adapter/capability tests**

Pin the tag/checksum resolution. Assert headers are passed via mpv `http-header-fields`, cookie values never appear in `description`/logs, H.264 is present, HEVC/AV1 depend on injected hardware support, no unsupported height above 2160, and command mapping uses mpv track IDs.

- [ ] **Step 2: Generate project and verify RED**

Run `xcodegen generate`, resolve packages, then MPV tests; expected missing adapter/capabilities.

- [ ] **Step 3: Implement the C bridge and AppKit view**

Initialize mpv before load; attach native view through `wid`; enable VideoToolbox hardware decoding with safe fallback; observe `time-pos`, `duration`, `pause`, `track-list`, `eof-reached` and error events through a single serialized event loop; destroy handle after the loop stops.

- [ ] **Step 4: Add LGPL compliance resources and smoke fixture**

Bundle MPVKit/libmpv/FFmpeg notices, versions, source URLs and relinking terms. Generate a two-second local H.264/AAC fixture with AVFoundation during tests only (no Homebrew tool) and assert mpv reaches `ready` then `ended` without network.

- [ ] **Step 5: Verify GREEN, embedded binaries and commit**

Run tests/build; inspect `otool -L` and app Frameworks so nothing points to `/usr/local`. Commit: `feat: integrate embedded libmpv playback`.

### Task 7: Coordinateur direct/remux/HLS/conversion

**Files:**
- Create: `AURORA/Playback/PlaybackCoordinator.swift`, `PlaybackState.swift`, `PlaybackClock.swift`, `TrackSelection.swift`
- Create: `AURORATests/PlaybackCoordinatorTests.swift`, `TrackSelectionTests.swift`, `Fakes/ManualPlaybackClock.swift`

**Interfaces:**
- Consumes: `AuroraServing`, `MPVControlling`, `PlayIntent`, token header from `SessionStoring`.
- Produces: `@MainActor @Observable final class PlaybackCoordinator` with `start(intent:)`, `togglePause`, `seek(by:)`, `seek(to:)`, `selectAudio`, `selectSubtitle`, `setRate`, `stop`, `playNext`; observable `state`, `position`, `duration`, `buffered`, `tracks`, `chapters`, `error`, `nextEpisode`, `countdown`.

- [ ] **Step 1: Write the playback state-machine tests**

Cover resume/restart, direct absolute seek, HLS local seek versus new session outside produced range or >60 s backward, 500 ms seek debounce, 10 s progress, pause/close progress, audio change creating HLS session, text subtitle `sub-add`, image subtitle server burn-in, French preference once, next episode countdown, network HLS one retry, direct failure→remux, video decode failure→transcode, and final explicit error with no VLC action.

- [ ] **Step 2: Add stale/concurrent response tests from Review Focus**

Use controllable continuations: resolve generation 2 before generation 1; assert only 2 loads and generation 1’s session is deleted. Stop while `/play` is suspended; after it resolves assert its session is deleted and mpv never loads it.

- [ ] **Step 3: Verify RED**

Run PlaybackCoordinator suite; expected missing coordinator.

- [ ] **Step 4: Implement coordinator**

Absolute position is `PlayResponse.offset + mpvTime`. For direct load at absolute start; for HLS load at local zero. Preserve `forcedMode` and chosen audio across restarts. Do not stop the previous HLS session before the replacement `/play` response because the NAS replaces it atomically; if the replacement is direct, stop the old HLS session after the direct source is ready. Use structured tasks plus a monotonically increasing generation; detached cleanup is forbidden, use an injected cleanup actor whose lifetime exceeds the view.

- [ ] **Step 5: Verify GREEN and commit**

Run all tests/build. Commit: `feat: implement resilient Aurora playback`.

### Task 8: Écran lecteur macOS

**Files:**
- Create: `AURORA/Features/Player/PlayerView.swift`, `PlayerControlsView.swift`, `TrackMenuView.swift`, `ChapterMenuView.swift`, `NextEpisodeView.swift`, `PlayerKeyCommand.swift`
- Modify: `AURORA/App/RootView.swift`
- Create: `AURORATests/PlayerKeyCommandTests.swift`, `NextEpisodeTests.swift`

**Interfaces:**
- Consumes: `PlaybackCoordinator`, `MPVRenderView`.
- Produces: `enum PlayerKeyCommand { case togglePause, seek(Double), fullscreen, closePanelOrPlayer, cycleSubtitles, cycleAudio }` and `static func map(_ event: NSEvent) -> PlayerKeyCommand?`.

- [ ] **Step 1: Write failing keyboard/countdown tests**

Assert space, left -10, right +30, F, Escape, C and A mappings; text fields ignore playback keys; controls remain visible while paused/menu open; countdown 8→0 can cancel or replace current episode screen.

- [ ] **Step 2: Verify RED**

Run Player UI state tests; expected missing mapper/view state.

- [ ] **Step 3: Implement player presentation**

Video fills black content area with aspect fit. Overlay shows title/chapter, seek bar with chapter marks, buffer, elapsed/remaining, back/play/forward, audio, subtitle, chapters, rate and stop. Auto-hide after six seconds only while playing and no menu.

- [ ] **Step 4: Wire lifecycle and full screen**

Closing view/window and app termination call `stop`; system media keys route through coordinator; full-screen uses the current app window, never a second external player.

- [ ] **Step 5: Verify GREEN and commit**

Run tests/build; play local fixture, seek, pause, change speed, enter/leave full screen. Commit: `feat: add native macOS player controls`.

### Task 9: Shell complet, profil, iconographie et intégration

**Files:**
- Create: `AURORA/App/MainShellView.swift`, `AppRoute.swift`
- Create: `AURORA/Features/Profile/ProfileView.swift`, `ProfileModel.swift`
- Modify: `AURORA/App/AURORAApp.swift`, `RootView.swift`, `AppSession.swift`
- Create: `AURORA/Assets.xcassets/AppIcon.appiconset/*`, `AURORA/Assets.xcassets/Contents.json`
- Create: `AURORATests/AppSessionIntegrationTests.swift`, `ProfileModelTests.swift`
- Modify: `README.md`, `AGENTS.md`

**Interfaces:**
- Consumes: every feature model/view.
- Produces: single navigation shell with Android order `Accueil / Films / Séries / Recherche / Profil`, navigation stack to detail/player, server switch and logout.

- [ ] **Step 1: Write failing integration tests**

Test authenticated launch→home, 401 from any feature→current NAS login, changing NAS cancels old loads/clears navigation, logout deletes only current token, and same screen type with a different episode gets a fresh playback coordinator.

- [ ] **Step 2: Verify RED**

Run integration tests; expected missing shell wiring.

- [ ] **Step 3: Implement app shell and profile**

Keep Android information architecture, restore browse memory, expose server/user/version, switch NAS and log out. Use `.id(route.identity)` for player/detail identity so adjacent episodes never reuse state.

- [ ] **Step 4: Add icon and finish accessibility/documentation**

Derive all icon sizes from AURORA’s existing 256 px source without changing that source. Add VoiceOver labels/values, keyboard focus order, reduced-motion handling, README commands and honest verified limitations.

- [ ] **Step 5: Verify GREEN and commit**

Run full tests/build, launch, exercise every tab with fixtures or reachable server, run Accessibility Inspector spot check. Commit: `feat: complete Aurora macOS experience`.

### Task 10: Release, DMG, installation et lancement

**Files:**
- Create: `Scripts/build-release.sh`, `Scripts/create-dmg.sh`, `Scripts/install-and-launch.sh`, `Scripts/verify-release.sh`
- Modify: `README.md`, `AGENTS.md`
- Output: `/Users/remyj/Documents/Codex/2026-10-20/tr/outputs/AURORA-0.1.0.dmg`

**Interfaces:**
- Consumes: scheme `AURORA`, app Release product and notices.
- Produces: repeatable commands `Scripts/build-release.sh`, `Scripts/create-dmg.sh <app> <dmg>`, `Scripts/install-and-launch.sh <app>`.

- [ ] **Step 1: Write release verification script first**

`verify-release.sh` must fail unless app executable exists, `codesign --verify --deep --strict` passes, MPV libraries/resources/notices exist, `otool -L` contains no `/usr/local` or `/opt/homebrew`, DMG mounts, contains `AURORA.app` plus `Applications` link, and mounted app matches source hashes.

- [ ] **Step 2: Verify the checker fails before artifacts exist**

Run checker with nonexistent app/DMG; expected nonzero exit and concise missing-artifact message.

- [ ] **Step 3: Implement deterministic build and DMG scripts**

Build Release into project-local `Build`, sign ad hoc because no Developer ID identity is installed, create a compressed read-only DMG with `hdiutil`, and write directly to the exact output path. Never depend on installed Homebrew mpv.

- [ ] **Step 4: Implement recoverable installation**

Before replacement, move existing `/Users/remyj/Applications/MesApps/AURORA.app` to `/Users/remyj/Applications/MesApps/.backups/AURORA-YYYYMMDD-HHMMSS.app`; copy with `ditto`; compare complete file manifests + SHA-256; run `codesign --verify`; launch with `open` and verify the process/window starts. Never delete a backup.

- [ ] **Step 5: Execute final verification**

Run, in order: full tests; clean Release build; release checker; DMG creation; DMG checker; installation; installed-copy checker; launch smoke. Expected zero failures. Test against the real NAS/video only if credentials/session are available; document exactly what was and was not exercised.

- [ ] **Step 6: Final independent review and commit**

Review API parity, secrets, playback cleanup, licensing, UI and packaging; fix all P1/P2 findings; rerun Step 5. Commit: `release: package Aurora 0.1.0 for macOS`.
