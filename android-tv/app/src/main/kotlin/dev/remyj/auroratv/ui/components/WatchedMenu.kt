package dev.remyj.auroratv.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.MutableState
import androidx.compose.runtime.saveable.listSaver
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.Dialog
import androidx.tv.material3.ClickableSurfaceDefaults
import androidx.tv.material3.Surface
import androidx.tv.material3.Text
import dev.remyj.auroratv.net.ApiException
import dev.remyj.auroratv.net.AuroraClient
import dev.remyj.auroratv.net.Card
import dev.remyj.auroratv.ui.theme.Aurora
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/** Conserve l'état confirmé lors du recyclage, en invalidant une réponse source devenue différente. */
@Composable
fun rememberWatchedCard(source: Card): MutableState<Card> = rememberSaveable(
    source,
    stateSaver = listSaver<Card, Any?>(
        save = { listOf(source.watched, source.unwatched, source.total, source.position, it.watched, it.unwatched) },
        restore = { saved ->
            if (saved.take(4) == listOf(source.watched, source.unwatched, source.total, source.position)) {
                source.copy(watched = saved[4] as Boolean, unwatched = saved[5] as Int?)
            } else source
        },
    ),
) { mutableStateOf(source) }

/** Diamètre du « ⋯ » : les écrans qui le posent sur une vignette s'en servent pour l'aligner. */
val MoreButtonSize = 26.dp

@Composable
fun MoreButton(onClick: () -> Unit, modifier: Modifier = Modifier) {
    Surface(
        onClick = onClick,
        shape = ClickableSurfaceDefaults.shape(CircleShape),
        colors = ClickableSurfaceDefaults.colors(
            containerColor = Aurora.Night.copy(alpha = 0.8f), contentColor = Aurora.Paper,
            focusedContainerColor = Aurora.Accent, focusedContentColor = Aurora.Ink,
        ),
        scale = ClickableSurfaceDefaults.scale(focusedScale = 1.1f),
        modifier = modifier.size(MoreButtonSize).semantics { contentDescription = "Vu / Pas vu" },
    ) {
        Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { Text("⋯", fontSize = 16.sp, fontWeight = FontWeight.Bold) }
    }
}

/** État d'une série dérivé des épisodes ; un état partiel ne coche aucune option. */
@Composable
fun WatchedMenu(card: Card, client: AuroraClient, onDismiss: () -> Unit, onChanged: (Card) -> Unit) {
    val watched = if (card.kind == "series") when {
        (card.total ?: 0) <= 0 -> null
        card.unwatched == 0 -> true
        card.unwatched == card.total -> false
        else -> null
    } else card.watched
    val currentOption = remember { FocusRequester() }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    LaunchedEffect(Unit) { delay(80); runCatching { currentOption.requestFocus() } }
    Dialog(onDismissRequest = { if (!busy) onDismiss() }) {
        Column(
            Modifier.width(300.dp).background(Aurora.Coal, RoundedCornerShape(12.dp))
                .border(1.dp, Aurora.Line, RoundedCornerShape(12.dp)).padding(10.dp),
            verticalArrangement = Arrangement.spacedBy(2.dp),
        ) {
            Text(if (card.kind == "episode") "Épisode ${card.episode ?: 0}" else card.title,
                color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 20.sp,
                modifier = Modifier.padding(start = 10.dp, top = 4.dp, bottom = 8.dp))
            listOf(true to "Vu", false to "Pas vu").forEach { (value, label) ->
                val selected = watched == value
                Surface(
                    onClick = {
                        if (!busy) {
                            if (selected) onDismiss() else {
                                busy = true
                                error = null
                                scope.launch {
                                    try {
                                        client.setState(card.id, watched = value)
                                        onChanged(card.copy(watched = value,
                                            unwatched = if (card.kind == "series") { if (value) 0 else card.total } else card.unwatched))
                                        onDismiss()
                                    } catch (e: ApiException) {
                                        error = e.message ?: "Modification impossible"
                                    } finally { busy = false }
                                }
                            }
                        }
                    },
                    shape = ClickableSurfaceDefaults.shape(RoundedCornerShape(8.dp)),
                    colors = ClickableSurfaceDefaults.colors(
                        containerColor = if (selected) Aurora.Accent.copy(alpha = 0.14f) else Color.Transparent,
                        contentColor = if (selected) Aurora.Paper else Aurora.Mist,
                        focusedContainerColor = Aurora.Accent, focusedContentColor = Aurora.Ink,
                    ),
                    scale = ClickableSurfaceDefaults.scale(focusedScale = 1f),
                    modifier = Modifier.fillMaxWidth().then(if (value == (watched ?: false)) Modifier.focusRequester(currentOption) else Modifier),
                ) {
                    Row(Modifier.padding(horizontal = 12.dp, vertical = 10.dp), verticalAlignment = Alignment.CenterVertically) {
                        Text(if (selected) "✓" else "", fontSize = 15.sp, fontWeight = FontWeight.Bold, modifier = Modifier.width(24.dp))
                        Text(label, fontSize = 17.sp)
                    }
                }
            }
            if (busy) Text("Enregistrement…", color = Aurora.Ash, modifier = Modifier.padding(10.dp))
            error?.let { Text(it, color = Aurora.Danger, modifier = Modifier.padding(10.dp)) }
        }
    }
}
