package com.sealdice.dice

import android.app.Service
import android.content.Intent
import android.os.IBinder
import android.util.Log
import androidx.preference.PreferenceManager
import okhttp3.OkHttpClient
import okhttp3.Request
import java.util.concurrent.Executors
import java.util.concurrent.ScheduledExecutorService
import java.util.concurrent.TimeUnit

class HeartbeatService : Service() {

    private val HEARTBEAT_INTERVAL_MS = 5000L // 心跳间隔时间，单位毫秒

    private var executorService: ScheduledExecutorService? = null
    private val client = OkHttpClient()

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val sharedPreferences = PreferenceManager.getDefaultSharedPreferences(applicationContext)
        val url = sharedPreferences.getString("ui_address", "http://127.0.0.1:3211")
        executorService = Executors.newSingleThreadScheduledExecutor()
        executorService?.scheduleAtFixedRate({
            // 在此处执行发送心跳请求的逻辑
            val request = Request.Builder()
                .url("$url/sd-api/signin/salt")
                .build()
            try {
                val response = client.newCall(request).execute()
                response.close()
                Log.d("--Service--","HeartBeat:${response.code}")
            } catch (e: Exception) {
                e.printStackTrace()
            }
        }, 11000, HEARTBEAT_INTERVAL_MS, TimeUnit.MILLISECONDS)
        return START_STICKY
    }

    override fun onBind(intent: Intent?): IBinder? {
        return null
    }

    override fun onDestroy() {
        executorService?.shutdown()
        super.onDestroy()
    }
}
