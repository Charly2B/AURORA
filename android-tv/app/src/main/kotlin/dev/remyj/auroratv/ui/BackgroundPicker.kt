package dev.remyj.auroratv.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.itemsIndexed
import androidx.compose.foundation.lazy.grid.rememberLazyGridState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.produceState
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusProperties
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.tv.material3.ClickableSurfaceDefaults
import androidx.tv.material3.Surface
import androidx.tv.material3.Text
import coil3.compose.AsyncImage
import coil3.compose.LocalPlatformContext
import coil3.network.HttpException
import coil3.request.ImageRequest
import dev.remyj.auroratv.net.ApiException
import dev.remyj.auroratv.net.AuroraClient
import dev.remyj.auroratv.net.BackgroundImage
import dev.remyj.auroratv.ui.components.ActionButton
import dev.remyj.auroratv.ui.components.Eyebrow
import dev.remyj.auroratv.ui.theme.Aurora
import kotlinx.coroutines.delay

/** Images Coil : même retour à la connexion que les appels JSON en cas de session révoquée. */
internal fun AuroraClient.backgroundImageFailed(error: Throwable) {
    if ((error as? HttpException)?.response?.code == 401) onUnauthorized?.invoke()
}

@Composable
fun BackgroundPicker(client: AuroraClient, appliedId: String?, onApply: (String?) -> Unit) {
    var attempt by remember { mutableStateOf(0) }
    var selectedId by rememberSaveable(client.baseUrl) { mutableStateOf(appliedId) }
    val state by produceState<Load<List<BackgroundImage>>>(Load.Loading, client, attempt) {
        value = Load.Loading
        value = try { Load.Ok(client.backgrounds().distinctBy { it.id }) }
        catch (e: ApiException) {
            Load.Failed(if (e.code == 404) "La galerie n’est pas disponible sur ce Synology." else e.message ?: "Galerie indisponible.")
        }
    }
    val images = (state as? Load.Ok)?.value.orEmpty()
    val selected = images.firstOrNull { it.id == selectedId } ?: images.firstOrNull()
    val requesters = remember(images) { images.associate { it.id to FocusRequester() } }
    val automatic = remember { FocusRequester() }
    val apply = remember { FocusRequester() }
    val grid = rememberLazyGridState()
    LaunchedEffect(state) {
        if (images.isNotEmpty()) {
            val index = images.indexOfFirst { it.id == selected?.id }.coerceAtLeast(0)
            grid.scrollToItem(index)
            delay(80)
            runCatching { requesters[images[index].id]?.requestFocus() }
        } else {
            delay(80)
            runCatching { automatic.requestFocus() }
        }
    }

    Column(Modifier.fillMaxSize().padding(horizontal = 30.dp, vertical = 24.dp)) {
        Eyebrow("Profil / Apparence", color = Aurora.Ash)
        Row(Modifier.fillMaxWidth().padding(top = 8.dp), verticalAlignment = Alignment.CenterVertically) {
            Text("Fond d’écran", color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 32.sp, modifier = Modifier.weight(1f))
            ActionButton("Automatique", onClick = { onApply(null) }, modifier = Modifier.focusRequester(automatic))
        }
        Text("Choisissez une image de votre Synology.", color = Aurora.Ash, fontSize = 13.sp, modifier = Modifier.padding(top = 10.dp, bottom = 20.dp))
        Row(Modifier.fillMaxWidth().weight(1f), horizontalArrangement = Arrangement.spacedBy(24.dp)) {
            Box(Modifier.weight(1f).fillMaxSize()) {
                when (val result = state) {
                    Load.Loading -> GalleryMessage("Chargement…")
                    is Load.Failed -> Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
                        Text(result.message, color = Aurora.Mist, fontSize = 15.sp)
                        ActionButton("Réessayer", onClick = { attempt++ })
                    }
                    is Load.Ok -> if (images.isEmpty()) GalleryMessage("Aucun fond sur ce Synology.") else LazyVerticalGrid(
                        columns = GridCells.Fixed(2), state = grid,
                        contentPadding = PaddingValues(4.dp),
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                        verticalArrangement = Arrangement.spacedBy(16.dp),
                    ) {
                        itemsIndexed(images, key = { _, image -> image.id }) { index, image ->
                            val chosen = image.id == selected?.id
                            Surface(
                                onClick = { selectedId = image.id; apply.requestFocus() },
                                shape = ClickableSurfaceDefaults.shape(RoundedCornerShape(8.dp)),
                                colors = ClickableSurfaceDefaults.colors(
                                    containerColor = Aurora.Coal, contentColor = Aurora.Paper,
                                    focusedContainerColor = Aurora.Accent, focusedContentColor = Aurora.Ink,
                                ),
                                scale = ClickableSurfaceDefaults.scale(focusedScale = 1.02f),
                                modifier = Modifier.fillMaxWidth()
                                    .focusRequester(requesters.getValue(image.id))
                                    .onFocusChanged { if (it.isFocused) selectedId = image.id }
                                    .focusProperties { if (index % 2 == 1) right = apply }
                                    .border(1.dp, if (chosen) Aurora.Accent else Aurora.Line, RoundedCornerShape(8.dp)),
                            ) {
                                Column(Modifier.padding(5.dp)) {
                                    AsyncImage(
                                        model = ImageRequest.Builder(LocalPlatformContext.current)
                                            .data(client.backgroundUrl(image.id)).size(320, 200).build(),
                                        contentDescription = null, contentScale = ContentScale.Crop,
                                        onError = { client.backgroundImageFailed(it.result.throwable) },
                                        modifier = Modifier.fillMaxWidth().aspectRatio(1.6f).clip(RoundedCornerShape(4.dp)),
                                    )
                                    Row(Modifier.padding(horizontal = 2.dp, vertical = 6.dp), verticalAlignment = Alignment.CenterVertically) {
                                        Text(image.name, fontSize = 11.sp, maxLines = 1, overflow = TextOverflow.Ellipsis, modifier = Modifier.weight(1f))
                                        if (chosen) Text(" ✓", fontSize = 12.sp)
                                    }
                                }
                            }
                        }
                    }
                }
            }
            Column(Modifier.weight(1.04f)) {
                var imageFailed by remember(selected?.id, attempt) { mutableStateOf(false) }
                Box(
                    Modifier.fillMaxWidth().aspectRatio(1.6f).clip(RoundedCornerShape(9.dp))
                        .background(Aurora.Coal).border(1.dp, Aurora.Line, RoundedCornerShape(9.dp)),
                    contentAlignment = Alignment.Center,
                ) {
                    if (selected != null) AsyncImage(
                        model = ImageRequest.Builder(LocalPlatformContext.current).data(client.backgroundUrl(selected.id)).size(720, 450).build(),
                        contentDescription = "Aperçu du fond choisi", contentScale = ContentScale.Crop,
                        onError = { imageFailed = true; client.backgroundImageFailed(it.result.throwable) },
                        onSuccess = { imageFailed = false }, modifier = Modifier.fillMaxSize(),
                    )
                    Box(Modifier.fillMaxSize().background(Aurora.Ink.copy(alpha = 0.25f)))
                    Column(
                        Modifier.fillMaxWidth(0.86f).background(Aurora.Paper.copy(alpha = 0.13f), RoundedCornerShape(8.dp))
                            .border(1.dp, Aurora.Paper.copy(alpha = 0.35f), RoundedCornerShape(8.dp)).padding(vertical = 28.dp),
                        horizontalAlignment = Alignment.CenterHorizontally,
                    ) {
                        Text("Bienvenue sur AURORA", color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 16.sp)
                        Box(Modifier.padding(top = 12.dp).width(25.dp).height(1.dp).background(Aurora.Accent))
                    }
                }
                Text(selected?.name ?: "Automatique", color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 18.sp, maxLines = 1, overflow = TextOverflow.Ellipsis, modifier = Modifier.padding(top = 14.dp))
                Text(if (imageFailed) "Image indisponible · repli automatique" else "Aperçu de l’accueil", color = Aurora.Ash, fontSize = 12.sp, modifier = Modifier.padding(top = 4.dp, bottom = 18.dp))
                if (selected != null) ActionButton("Utiliser ce fond", primary = true, onClick = { onApply(selected.id) }, modifier = Modifier
                    .focusRequester(apply).focusProperties { left = requesters.getValue(selected.id) })
            }
        }
        Row(Modifier.fillMaxWidth().padding(top = 16.dp)) {
            Text("Choix conservé pour ce Synology", color = Aurora.Ash, fontSize = 11.sp)
            Spacer(Modifier.weight(1f))
            Text("Retour · Profil", color = Aurora.Ash, fontSize = 11.sp)
        }
    }
}

@Composable
private fun GalleryMessage(message: String) {
    Text(message, color = Aurora.Mist, fontSize = 15.sp, modifier = Modifier.padding(top = 12.dp))
}
