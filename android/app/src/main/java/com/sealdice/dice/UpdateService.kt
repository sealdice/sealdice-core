package com.sealdice.dice

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Intent
import android.net.Uri
import android.os.IBinder
import android.util.Log
import androidx.core.app.NotificationCompat
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.GlobalScope
import kotlinx.coroutines.launch
import okhttp3.OkHttpClient
import okhttp3.Request
import org.json.JSONObject

class UpdateService : Service() {

    private val UPDATE_URL = "https://get.sealdice.com/seal/version/android"
    private val NOTIFICATION_ID = 1

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        GlobalScope.launch(context = Dispatchers.IO) {
            checkForUpdates()
        }
        return START_STICKY
    }

    override fun onBind(intent: Intent?): IBinder? {
        return null
    }

    private fun checkForUpdates() {
        val client = OkHttpClient()
        val request = Request.Builder()
            .url(UPDATE_URL)
            .build()
        try {
            val response = client.newCall(request).execute()
            val jsonData = response.body?.string()
            val jsonObject = jsonData?.let { JSONObject(it) }
            val latestVersion = jsonObject?.getString("version")
            Log.e("--Service--","已经获取了version${latestVersion}")
            // Check if current version is up-to-date
            val currentVersion = BuildConfig.VERSION_NAME
            if (latestVersion != currentVersion) {
                // Show update notification
                if (latestVersion != null) {
                    showUpdateNotification(latestVersion)
                }
            } else {
                // Current version is up-to-date
            }
        } catch (e: Exception) {
            e.printStackTrace()
        }
    }

    private fun showUpdateNotification(version: String) {
        // Create a notification channel for Android Oreo and higher
        val notificationManager = getSystemService(NotificationManager::class.java)
        val channel = NotificationChannel(
            "update_channel",
            "Update Channel",
            NotificationManager.IMPORTANCE_DEFAULT
        )
        notificationManager.createNotificationChannel(channel)
        // Create a notification builder
        val builder = NotificationCompat.Builder(this, "update_channel")
            .setSmallIcon(R.drawable.ic_launcher_foreground)
            .setContentTitle("SealDice 检测到更新")
            .setContentText("点击此处下载新版本 $version")
            .setAutoCancel(true)
            .setPriority(NotificationCompat.PRIORITY_DEFAULT)
        // Create a pending intent for the update dialog
        val uri = Uri.parse("https://d.catlevel.com/seal/android/latest")
        val intent = Intent()
        intent.action = "android.intent.action.VIEW"
        intent.data = uri
        val pendingIntent = PendingIntent.getActivity(this, 0, intent, PendingIntent.FLAG_UPDATE_CURRENT)
        builder.setContentIntent(pendingIntent)
        // Show the notification
        notificationManager.notify(NOTIFICATION_ID, builder.build())
    }

    // ...

}