package com.sealdice.dice

import android.app.*
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.os.Binder
import android.os.Build
import android.os.IBinder
import androidx.preference.PreferenceManager
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.GlobalScope
import kotlinx.coroutines.launch
import java.io.BufferedReader
import java.io.File
import java.io.InputStreamReader


class ProcessService : Service(){
    private val binder = MyBinder()
    private var processBuilder: ProcessBuilder = ProcessBuilder("sh").redirectErrorStream(true)
    private lateinit var process: Process
    private var isRunning = false
    private var shellLogs = ""
    inner class MyBinder : Binder() {
        fun getService(): ProcessService = this@ProcessService
    }
    fun getShellLogs(): String {
        return shellLogs
    }
    fun stopProcess() {
        isRunning = false
        Thread.sleep(5)
        process.destroy()
    }
    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        if (Build.VERSION.SDK_INT >= 26) {
            val ns: String = Context.NOTIFICATION_SERVICE
            val mNotificationManager = getSystemService(ns) as NotificationManager
            val notificationChannel = NotificationChannel("sealdice","SealDice", NotificationManager.IMPORTANCE_HIGH)
            mNotificationManager.createNotificationChannel(notificationChannel)
            val pendingIntent = PendingIntent.getActivity(applicationContext, 0, Intent(applicationContext, NotificationActivity::class.java), PendingIntent.FLAG_MUTABLE)
            val notification: Notification = Notification.Builder(this,"sealdice")
                .setContentTitle("SealDice is running")
                .setSmallIcon(R.drawable.ic_launcher_foreground)
                .setContentIntent(pendingIntent)
                .build()
            startForeground(1, notification)
        }
        else {
            val pendingIntent = PendingIntent.getActivity(applicationContext, 0, Intent(applicationContext, NotificationActivity::class.java), PendingIntent.FLAG_MUTABLE)
            val notification: Notification = Notification.Builder(this)
                .setContentTitle("SealDice is running")
                .setSmallIcon(R.drawable.ic_launcher_foreground)
                .setContentIntent(pendingIntent)
                .build()
            startForeground(1, notification)
        }
        if (!isRunning) {
            isRunning = true
            process = processBuilder.directory(File(this.filesDir.absolutePath)).start()
            val sharedPreferences = PreferenceManager.getDefaultSharedPreferences(this)
            val args = sharedPreferences.getString("launch_args", "")
            val cmd = "cd sealdice&&./sealdice-core $args"
            GlobalScope.launch(context = Dispatchers.IO) {
                val os = process.outputStream
                os.write(cmd.toByteArray())
                os.flush()
                os.close()
                val data = process.inputStream
                val ir = BufferedReader(InputStreamReader(data))
                while (isRunning) {
                    var line: String?
                    try {
                        line = ir.readLine()
                    } catch (e: Exception) {
                        break
                    }
                    while (line != null && isRunning) {
                        shellLogs += line
                        shellLogs += "\n"
                        try {
                            line = ir.readLine()
                        } catch (e: Exception) {
                            break
                        }
                    }
                    Thread.sleep(1000)
                }
            }
        }
        return START_STICKY
    }
    fun isRunning(): Boolean {
        return isRunning
    }
    override fun onBind(p0: Intent?): IBinder {
        return binder
    }
    override fun onDestroy() {
        super.onDestroy()
        stopProcess()
    }
}