package dev.remyj.auroratv.ui

import android.provider.Settings
import android.os.Build
import android.content.Context
import androidx.compose.foundation.background
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusDirection
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusProperties
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.key.Key
import androidx.compose.ui.input.key.KeyEventType
import androidx.compose.ui.input.key.key
import androidx.compose.ui.input.key.onPreviewKeyEvent
import androidx.compose.ui.input.key.type
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.tv.material3.ClickableSurfaceDefaults
import androidx.tv.material3.Surface
import androidx.tv.material3.Text
import androidx.compose.ui.focus.onFocusChanged
import androidx.tv.material3.Border
import dev.remyj.auroratv.data.normalizeAddress
import dev.remyj.auroratv.ui.components.ActionButton
import dev.remyj.auroratv.ui.theme.Aurora
import androidx.compose.ui.platform.LocalContext
import dev.remyj.auroratv.net.ApiException
import dev.remyj.auroratv.net.AuroraClient
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/** Décor des écrans de configuration : fond de salle et léger halo d'aurore. */
@Composable
fun Stage(content: @Composable () -> Unit) {
    Box(
        Modifier
            .fillMaxSize()
            .background(Aurora.Ink)
            .background(Brush.radialGradient(listOf(Color(0x267EE8C6), Color.Transparent), radius = 1400f)),
    ) {
        Column(Modifier.fillMaxSize().padding(horizontal = 96.dp, vertical = 56.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text("AURORA", color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 26.sp, letterSpacing = 6.sp)
            }
            Spacer(Modifier.height(40.dp))
            content()
        }
    }
}

@Composable
fun ServersScreen(vm: AppViewModel) {
    val servers = remember { vm.servers }
    val first = remember { FocusRequester() }
    LaunchedEffect(Unit) { runCatching { first.requestFocus() } }
    Stage {
        Text("Vos Synology", color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 44.sp)
        Text("Choisissez le NAS à ouvrir.", color = Aurora.Ash, fontSize = 18.sp, modifier = Modifier.padding(top = 8.dp, bottom = 28.dp))
        vm.serverError?.let { Text(it, color = Aurora.Danger, fontSize = 16.sp, modifier = Modifier.padding(bottom = 16.dp)) }
        LazyColumn(verticalArrangement = Arrangement.spacedBy(12.dp), modifier = Modifier.width(720.dp)) {
            items(servers, key = { it.id }) { s ->
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Surface(
                        onClick = { vm.serverError = null; vm.openServer(s) },
                        shape = ClickableSurfaceDefaults.shape(RoundedCornerShape(14.dp)),
                        colors = ClickableSurfaceDefaults.colors(
                            containerColor = Aurora.Coal, focusedContainerColor = Aurora.Accent,
                            contentColor = Aurora.Paper, focusedContentColor = Aurora.Ink,
                        ),
                        scale = ClickableSurfaceDefaults.scale(focusedScale = 1.03f),
                        modifier = Modifier.weight(1f).then(if (s == servers.first()) Modifier.focusRequester(first) else Modifier),
                    ) {
                        Column(Modifier.padding(horizontal = 24.dp, vertical = 16.dp)) {
                            Text(s.name, fontSize = 20.sp, fontWeight = FontWeight.SemiBold)
                            Text(s.baseUrl + if (s.username.isNotEmpty()) "  ·  ${s.username}" else "", fontSize = 14.sp)
                        }
                    }
                    Spacer(Modifier.width(12.dp))
                    ActionButton("Retirer", onClick = { vm.removeServer(s) })
                }
            }
            item {
                Spacer(Modifier.height(16.dp))
                ActionButton(
                    "+  Ajouter un Synology", primary = true, onClick = { vm.push(Screen.AddServer) },
                    modifier = if (servers.isEmpty()) Modifier.focusRequester(first) else Modifier,
                )
            }
        }
    }
}

@Composable
fun AddServerScreen(vm: AppViewModel) {
    var address by remember { mutableStateOf("") }
    var error by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val field = remember { FocusRequester() }
    val go = remember { FocusRequester() }
    LaunchedEffect(Unit) { runCatching { field.requestFocus() } }
    Stage {
        Text("Ajouter un Synology", color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 44.sp)
        Text(
            "Adresse IP du NAS sur votre réseau (ex. 192.168.1.20). Le port 8890 d'AURORA est ajouté automatiquement.",
            color = Aurora.Ash, fontSize = 18.sp, modifier = Modifier.padding(top = 8.dp, bottom = 28.dp).width(820.dp),
        )
        Column(verticalArrangement = Arrangement.spacedBy(18.dp), modifier = Modifier.width(560.dp)) {
            Field(
                address, { address = it; error = null }, "Adresse IP du Synology",
                Modifier.focusRequester(field).focusProperties { down = go }, KeyboardType.Uri,
            )
            error?.let { Text(it, color = Aurora.Danger, fontSize = 16.sp) }
            Row(horizontalArrangement = Arrangement.spacedBy(14.dp)) {
                ActionButton(if (busy) "Vérification…" else "Continuer", primary = true, onClick = {
                    if (busy) return@ActionButton
                    val url = normalizeAddress(address)
                    if (url == null) {
                        error = "Adresse invalide."
                        return@ActionButton
                    }
                    busy = true
                    scope.launch {
                        error = vm.addServer(url)
                        busy = false
                    }
                }, modifier = Modifier.focusRequester(go))
                if (vm.servers.isNotEmpty()) ActionButton("Annuler", onClick = { vm.back() })
            }
        }
    }
}

/**
 * Connexion. Par défaut, **code d'association** (comme plex.tv/link) : la TV affiche un code, on le valide
 * sur /link depuis un navigateur déjà connecté, et la TV reçoit sa session — rien à taper à la télécommande.
 * Le mot de passe reste possible en secours.
 */
@Composable
fun LoginScreen(vm: AppViewModel, screen: Screen.Login) {
    var withPassword by remember { mutableStateOf(false) }
    if (withPassword) PasswordLogin(vm, screen, onUseCode = { withPassword = false })
    else CodeLogin(vm, screen, onUsePassword = { withPassword = true })
}

@Composable
private fun CodeLogin(vm: AppViewModel, screen: Screen.Login, onUsePassword: () -> Unit) {
    val context = LocalContext.current
    var code by remember { mutableStateOf<String?>(null) }
    var message by remember { mutableStateOf<String?>(null) }
    val first = remember { FocusRequester() }
    LaunchedEffect(Unit) { delay(100); runCatching { first.requestFocus() } }
    LaunchedEffect(Unit) {
        val client = AuroraClient(vm.http, screen.server.baseUrl)
        val name = deviceName(context)
        val id = screen.server.id
        // Boucle : nouveau code seulement à l'expiration (410) ; une coupure réseau ne fait que retarder le suivi
        // du même code (sinon le code saisi entre-temps sur /link serait perdu).
        while (true) {
            val link = vm.cachedLink(id) ?: try {
                client.linkStart(name).also { vm.rememberLink(id, it) }
            } catch (e: ApiException) {
                code = null
                message = e.message
                delay(5_000)
                continue
            }
            code = link.code // message conservé : il s'efface au premier suivi réussi
            while (true) {
                delay(2_000)
                val token = try {
                    client.linkPoll(link.poll)
                } catch (e: ApiException) {
                    if (e.code == 410) {
                        vm.forgetLink(id)
                        break
                    }
                    message = e.message
                    delay(3_000)
                    continue
                }
                message = null
                if (token != null) {
                    vm.forgetLink(id)
                    message = vm.loginWithToken(screen.server, token)
                    if (message == null) return@LaunchedEffect // connecté : l'écran change
                    break
                }
            }
        }
    }
    Stage {
        Text("Connexion", color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 44.sp)
        Text(
            "Sur un ordinateur ou un téléphone déjà connecté à AURORA, ouvrez",
            color = Aurora.Mist, fontSize = 19.sp, modifier = Modifier.padding(top = 10.dp),
        )
        Text(
            screen.server.baseUrl + "/link",
            color = Aurora.Accent, fontSize = 26.sp, fontWeight = FontWeight.SemiBold, modifier = Modifier.padding(top = 4.dp),
        )
        Text("puis saisissez ce code :", color = Aurora.Mist, fontSize = 19.sp, modifier = Modifier.padding(top = 4.dp))
        Text(
            code ?: "…",
            color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 64.sp, letterSpacing = 8.sp,
            modifier = Modifier.padding(top = 4.dp, bottom = 4.dp),
        )
        Text(
            message ?: "La TV se connectera toute seule dès que le code sera validé.",
            color = if (message != null) Aurora.Danger else Aurora.Ash, fontSize = 16.sp,
        )
        Row(Modifier.padding(top = 18.dp), horizontalArrangement = Arrangement.spacedBy(14.dp)) {
            ActionButton("Utiliser un mot de passe", onClick = onUsePassword, modifier = Modifier.focusRequester(first))
            ActionButton("Retour", onClick = { if (!vm.back()) vm.switchServer() })
        }
    }
}

/** Nom affiché sur /link quand on valide le code (« Xiaomi-Chambre »…), sinon le modèle. */
private fun deviceName(context: Context): String =
    runCatching { Settings.Global.getString(context.contentResolver, "device_name") }.getOrNull()
        ?.takeIf { it.isNotBlank() } ?: Build.MODEL

@Composable
private fun PasswordLogin(vm: AppViewModel, screen: Screen.Login, onUseCode: () -> Unit) {
    var username by remember { mutableStateOf(screen.server.username) }
    var password by remember { mutableStateOf("") }
    var error by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val first = remember { FocusRequester() }
    val secret = remember { FocusRequester() }
    val enter = remember { FocusRequester() }
    // Nom déjà connu (reconnexion) : on part directement sur le mot de passe.
    LaunchedEffect(Unit) { runCatching { (if (username.isEmpty()) first else secret).requestFocus() } }
    Stage {
        Text("Connexion", color = Aurora.Paper, fontFamily = Aurora.Display, fontSize = 44.sp)
        Text(
            "${screen.server.name} — ${screen.server.baseUrl}. Utilisez votre compte AURORA (pas celui du Synology).",
            color = Aurora.Ash, fontSize = 18.sp, modifier = Modifier.padding(top = 8.dp, bottom = 28.dp),
        )
        // Champs empilés verticalement : un champ dans une rangée capterait ◀/▶ (piège connu sur la box).
        Column(verticalArrangement = Arrangement.spacedBy(18.dp), modifier = Modifier.width(560.dp)) {
            Field(username, { username = it; error = null }, "Nom d'utilisateur", Modifier.focusRequester(first))
            Field(
                password, { password = it; error = null }, "Mot de passe",
                Modifier.focusRequester(secret).focusProperties { down = enter }, KeyboardType.Password, secret = true,
            )
            error?.let { Text(it, color = Aurora.Danger, fontSize = 16.sp) }
            Row(horizontalArrangement = Arrangement.spacedBy(14.dp)) {
                ActionButton(if (busy) "Connexion…" else "Entrer", primary = true, onClick = {
                    if (busy || username.isBlank() || password.isEmpty()) return@ActionButton
                    busy = true
                    scope.launch {
                        error = vm.login(screen.server, username.trim(), password)
                        busy = false
                    }
                }, modifier = Modifier.focusRequester(enter))
                ActionButton("Avec un code", onClick = onUseCode)
            }
        }
    }
}

/**
 * Champ texte pour télécommande. Hors édition, c'est un bouton à l'image du champ : le traverser avec
 * ▲/▼ n'ouvre pas le clavier (l'IME Android TV l'ouvre dès qu'un éditeur prend le focus, et il capte
 * alors toutes les touches). OK passe en édition ; dès que le focus part, retour au bouton.
 */
@Composable
fun Field(
    value: String,
    onValueChange: (String) -> Unit,
    label: String,
    modifier: Modifier = Modifier,
    keyboard: KeyboardType = KeyboardType.Text,
    secret: Boolean = false,
) {
    val focus = LocalFocusManager.current
    var editing by remember { mutableStateOf(false) }
    val editor = remember { FocusRequester() }
    // focusRequester / focusProperties de l'appelant s'appliquent à l'enfant focalisable, quel que soit l'état.
    Box(modifier.fillMaxWidth()) {
        if (!editing) {
            Surface(
                onClick = { editing = true },
                shape = ClickableSurfaceDefaults.shape(RoundedCornerShape(10.dp)),
                colors = ClickableSurfaceDefaults.colors(
                    containerColor = Color.Transparent, focusedContainerColor = Color(0x14FFFFFF),
                    contentColor = Aurora.Mist, focusedContentColor = Aurora.Paper,
                ),
                border = ClickableSurfaceDefaults.border(
                    border = Border(BorderStroke(1.dp, Aurora.Graphite), shape = RoundedCornerShape(10.dp)),
                    focusedBorder = Border(BorderStroke(2.dp, Aurora.Accent), shape = RoundedCornerShape(10.dp)),
                ),
                scale = ClickableSurfaceDefaults.scale(focusedScale = 1f),
                modifier = Modifier.fillMaxWidth(),
            ) {
                Column(Modifier.padding(horizontal = 16.dp, vertical = 9.dp)) {
                    Text(label, color = Aurora.Ash, fontSize = 12.sp)
                    Text(
                        if (secret) "•".repeat(value.length) else value,
                        fontSize = 16.sp, maxLines = 1, modifier = Modifier.padding(top = 2.dp).height(22.dp),
                    )
                }
            }
        } else {
            var hadFocus by remember { mutableStateOf(false) }
            LaunchedEffect(Unit) { runCatching { editor.requestFocus() } }
            OutlinedTextField(
                value = value,
                onValueChange = onValueChange,
                singleLine = true,
                label = { androidx.compose.material3.Text(label, color = Aurora.Ash) },
                keyboardOptions = KeyboardOptions(keyboardType = keyboard, imeAction = ImeAction.Next),
                keyboardActions = KeyboardActions(onNext = { focus.moveFocus(FocusDirection.Down) }),
                visualTransformation = if (secret) PasswordVisualTransformation() else VisualTransformation.None,
                shape = RoundedCornerShape(10.dp),
                colors = OutlinedTextFieldDefaults.colors(
                    focusedTextColor = Aurora.Paper,
                    unfocusedTextColor = Aurora.Mist,
                    focusedBorderColor = Aurora.Accent,
                    unfocusedBorderColor = Aurora.Graphite,
                    cursorColor = Aurora.Accent,
                ),
                modifier = Modifier
                    .fillMaxWidth()
                    .focusRequester(editor)
                    .onFocusChanged {
                        if (it.isFocused) hadFocus = true else if (hadFocus) editing = false
                    }
                    // Un champ texte garde ▼/▲ pour lui : sans ceci, la télécommande ne peut plus en sortir.
                    .onPreviewKeyEvent { e ->
                        if (e.type != KeyEventType.KeyDown) return@onPreviewKeyEvent false
                        when (e.key) {
                            Key.DirectionDown -> focus.moveFocus(FocusDirection.Down)
                            Key.DirectionUp -> focus.moveFocus(FocusDirection.Up)
                            else -> false
                        }
                    },
            )
        }
    }
}
