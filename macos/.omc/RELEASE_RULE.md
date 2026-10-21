# Release Rules
<!-- last-analyzed: 2026-10-20T17:48:07Z -->

## Version Sources

- `project.yml`: `CFBundleShortVersionString` (`0.1.0`) et `CFBundleVersion`.
- `AURORA/Info.plist`: généré/maintenu avec les mêmes valeurs.
- `docs/superpowers/plans/2026-10-20-aurora-macos.md`: nom du livrable DMG.

## Release Trigger

Release locale manuelle via les scripts `Scripts/`; aucun push ni tag requis pour le DMG demandé.

## Test Gate

`xcodegen generate && xcodebuild -project AURORA.xcodeproj -scheme AURORA -destination 'platform=macOS' test`

## Registry / Distribution

Aucun registre et aucune publication automatique. Distribution locale par DMG compressé, signature ad hoc.

## Release Notes Strategy

README utilisateur ; aucun changelog historique pour cette première version.

## CI Workflow Files

Aucun.

## First-Time Setup Gaps

- Aucun workflow CI/release.
- Aucun tag Git existant.
- `Build/` et `.superpowers/` sont ignorés.
- Pas de certificat Developer ID : DMG local non notarisé, explicitement documenté.
