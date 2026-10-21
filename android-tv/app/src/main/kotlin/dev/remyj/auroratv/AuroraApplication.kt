package dev.remyj.auroratv

import android.app.Application
import coil3.ImageLoader
import coil3.PlatformContext
import coil3.SingletonImageLoader
import coil3.disk.DiskCache
import coil3.memory.MemoryCache
import coil3.network.okhttp.OkHttpNetworkFetcherFactory
import coil3.request.crossfade
import dev.remyj.auroratv.data.ServerStore
import dev.remyj.auroratv.net.Sessions
import okhttp3.OkHttpClient
import okio.Path.Companion.toOkioPath
import java.util.concurrent.TimeUnit

class AuroraApplication : Application(), SingletonImageLoader.Factory {

    lateinit var container: AppContainer
        private set

    override fun onCreate() {
        super.onCreate()
        container = AppContainer(this)
    }

    /** Affiches et fonds en cache disque : la navigation reste fluide et les images ne se rechargent pas. */
    override fun newImageLoader(context: PlatformContext): ImageLoader =
        ImageLoader.Builder(context)
            .components { add(OkHttpNetworkFetcherFactory(callFactory = { container.http })) }
            .memoryCache { MemoryCache.Builder().maxSizePercent(context, 0.25).build() }
            .diskCache {
                DiskCache.Builder()
                    .directory(cacheDir.resolve("images").toOkioPath())
                    .maxSizeBytes(400L * 1024 * 1024)
                    .build()
            }
            .crossfade(true)
            .build()
}

/** Dépendances partagées. */
class AppContainer(application: Application) {
    val http: OkHttpClient = OkHttpClient.Builder()
        // Un NAS éteint doit échouer vite plutôt que figer l'écran.
        .connectTimeout(6, TimeUnit.SECONDS)
        .readTimeout(60, TimeUnit.SECONDS)
        .addInterceptor(Sessions.interceptor())
        .build()
    val servers = ServerStore(application)
}
