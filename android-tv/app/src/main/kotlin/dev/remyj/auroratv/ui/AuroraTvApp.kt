package dev.remyj.auroratv.ui

import androidx.activity.compose.BackHandler
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.runtime.key
import androidx.compose.runtime.saveable.rememberSaveableStateHolder
import dev.remyj.auroratv.ui.components.LocalCardFocus

/** Racine : affiche l'écran courant de la pile. Retour remonte d'un écran. */
@Composable
fun AuroraTvApp(vm: AppViewModel) {
    val screen = vm.screen
    // Actif seulement s'il reste où revenir : sinon Retour doit quitter l'app (comportement Android TV).
    val canGoBack = vm.canGoBack || (screen is Screen.Main && vm.tab != Tab.HOME)
    if (screen !is Screen.Player) {
        BackHandler(enabled = canGoBack) {
            if (!vm.back() && screen is Screen.Main && vm.tab != Tab.HOME) vm.tab = Tab.HOME
        }
    }
    // État « saveable » par écran (dossier, tri, défilement) conservé quand on revient d'une fiche.
    val saved = rememberSaveableStateHolder()
    // Clé = NAS + écran : un autre NAS ne restaure pas le défilement du précédent. Les états des écrans
    // sortis de la pile sont purgés (sinon le Bundle de sauvegarde grossit à chaque fiche ouverte).
    val serverId = vm.server?.id.orEmpty()
    fun keyOf(s: Screen) = "$serverId|$s"
    val inStack = vm.screens.map(::keyOf).toSet()
    val known = remember { mutableSetOf<String>() }
    LaunchedEffect(inStack) {
        (known - inStack).forEach { saved.removeState(it) }
        known.retainAll(inStack)
        known.addAll(inStack)
    }
    CompositionLocalProvider(LocalCardFocus provides vm.cardFocus) {
        // key(screen) : un écran remplacé par un autre du même type (épisode suivant, fiche → fiche)
        // doit repartir de zéro ; sans clé, Compose réutiliserait l'état — et le lecteur — du précédent.
        key(screen) {
            saved.SaveableStateProvider(keyOf(screen)) {
                when (screen) {
                    Screen.Loading -> Message("Connexion à AURORA…")
                    Screen.Servers -> ServersScreen(vm)
                    Screen.AddServer -> AddServerScreen(vm)
                    is Screen.Login -> LoginScreen(vm, screen)
                    Screen.Main -> MainScreen(vm)
                    is Screen.Detail -> DetailScreen(vm, screen)
                    is Screen.Player -> PlayerScreen(vm, screen)
                }
            }
        }
    }
}
