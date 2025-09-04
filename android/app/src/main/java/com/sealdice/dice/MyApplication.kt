package com.sealdice.dice

import android.app.Application
import android.content.Context
import androidx.preference.PreferenceManager
import com.sealdice.dice.utils.Utils.setHideTaskStatus
import com.sealdice.dice.secrets.Auth.*
import org.acra.config.httpSender
import org.acra.data.StringFormat
import org.acra.ktx.initAcra
import org.acra.sender.HttpSender
import kotlin.properties.Delegates

class MyApplication : Application() {
    companion object {
        // Kotlin 中的对象表达式，它定义了一个伴生对象（companion object）
        // 表示在使用 appContext 属性之前，必须为它分配一个非空的值，否则会抛出异常。
        var appContext: Context by Delegates.notNull()
    }
    override fun onCreate() {
        super.onCreate()
        appContext = applicationContext
        // 读取用户配置，并根据配置处理是否在最近任务栏不显示
        val preferences = PreferenceManager.getDefaultSharedPreferences(this)
        val isEnabled = preferences.getBoolean("alive_excluderecents", false)
        // 根据启用状态，动态调整执行隐藏/不隐藏
        setHideTaskStatus(hide = isEnabled)
    }
    override fun attachBaseContext(base: Context) {
        super.attachBaseContext(base)

        initAcra {
            reportFormat = StringFormat.JSON
            httpSender {
                uri = ACRA_URL
                basicAuthLogin = ACRA_BASIC_AUTH
                basicAuthPassword = ACRA_LOGIN_PASS
                httpMethod = HttpSender.Method.POST
            }
        }

    }
}