package dev.remyj.auroratv

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewmodel.compose.viewModel
import dev.remyj.auroratv.ui.AppViewModel
import dev.remyj.auroratv.ui.AuroraTvApp
import dev.remyj.auroratv.ui.theme.AuroraTheme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val container = (application as AuroraApplication).container
        setContent {
            AuroraTheme {
                val vm: AppViewModel = viewModel(
                    factory = object : ViewModelProvider.Factory {
                        @Suppress("UNCHECKED_CAST")
                        override fun <T : ViewModel> create(modelClass: Class<T>): T = AppViewModel(application, container) as T
                    },
                )
                AuroraTvApp(vm)
            }
        }
    }
}
