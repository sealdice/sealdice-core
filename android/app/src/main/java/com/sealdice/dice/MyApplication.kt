package com.sealdice.dice

import android.app.Application
import android.content.Context
import com.sealdice.dice.secrets.Auth.*
import org.acra.config.httpSender
import org.acra.data.StringFormat
import org.acra.ktx.initAcra
import org.acra.sender.HttpSender

class MyApplication : Application() {
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