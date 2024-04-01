package com.sealdice.dice

import android.app.*
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.graphics.PixelFormat
import android.media.MediaPlayer
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import android.net.Uri
import android.os.Binder
import android.os.Build
import android.os.IBinder
import android.os.PowerManager
import android.util.Base64
import android.util.Log
import android.view.Gravity
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.view.WindowManager
import androidx.lifecycle.LifecycleService
import androidx.preference.PreferenceManager
import com.sealdice.dice.utils.Utils
import com.sealdice.dice.utils.ViewModelMain
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.GlobalScope
import kotlinx.coroutines.launch
import java.io.BufferedReader
import java.io.File
import java.io.FileOutputStream
import java.io.InputStreamReader
import java.util.concurrent.TimeUnit


class ProcessService : LifecycleService(){
    private val binder = MyBinder()
    private var processBuilder: ProcessBuilder = ProcessBuilder("sh").redirectErrorStream(true)
    private lateinit var process: Process
    private var isRunning = false
    private var shellLogs = ""
    private lateinit var player: MediaPlayer

    private lateinit var windowManager: WindowManager
    private var floatRootView: View? = null//悬浮窗View
    inner class MyBinder : Binder() {
        fun getService(): ProcessService = this@ProcessService
    }
    fun getShellLogs(): String {
        return shellLogs
    }
    fun stopProcess() {
        player.stop()
        isRunning = false
        if (process.isAlive) {
            process.destroy()
            process.waitFor(10, TimeUnit.SECONDS)
//            Log.e("ProcessService", "stopProcess")
            process.destroyForcibly()
        }
    }
    override fun onCreate() {
        super.onCreate()
        initObserve()
        val base64 =
            "AAAAGGZ0eXBtcDQyAAAAAG1wNDFpc29tAAAAKHV1aWRcpwj7Mo5CBahhZQ7KCpWWAAAADDEwLjAuMTgzNjMuMAAAAG5tZGF0AAAAAAAAABAnDEMgBAIBAIBAIBAIBAIBAIBAIBAIBAIBAIBAIBAIBAIBAIBAIBAIBAIBAIBAIBAIBAIBAIBAIBAIBDSX5AAAAAAAAB9Pp9Pp9Pp9Pp9Pp9Pp9Pp9Pp9Pp9Pp9Pp9AAAC/m1vb3YAAABsbXZoZAAAAADeilCc3opQnAAAu4AAAAIRAAEAAAEAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAMAAAHBdHJhawAAAFx0a2hkAAAAAd6KUJzeilCcAAAAAgAAAAAAAAIRAAAAAAAAAAAAAAAAAQAAAAABAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAABXW1kaWEAAAAgbWRoZAAAAADeilCc3opQnAAAu4AAAAIRVcQAAAAAAC1oZGxyAAAAAAAAAABzb3VuAAAAAAAAAAAAAAAAU291bmRIYW5kbGVyAAAAAQhtaW5mAAAAEHNtaGQAAAAAAAAAAAAAACRkaW5mAAAAHGRyZWYAAAAAAAAAAQAAAAx1cmwgAAAAAQAAAMxzdGJsAAAAZHN0c2QAAAAAAAAAAQAAAFRtcDRhAAAAAAAAAAEAAAAAAAAAAAACABAAAAAAu4AAAAAAADBlc2RzAAAAAAOAgIAfAAAABICAgBRAFQAGAAACM2gAAjNoBYCAgAIRkAYBAgAAABhzdHRzAAAAAAAAAAEAAAABAAACEQAAABxzdHNjAAAAAAAAAAEAAAABAAAAAQAAAAEAAAAYc3RzegAAAAAAAAAAAAAAAQAAAF4AAAAUc3RjbwAAAAAAAAABAAAAUAAAAMl1ZHRhAAAAkG1ldGEAAAAAAAAAIWhkbHIAAAAAAAAAAG1kaXIAAAAAAAAAAAAAAAAAAAAAY2lsc3QAAAAeqW5hbQAAABZkYXRhAAAAAQAAAADlvZXpn7MAAAAcqWRheQAAABRkYXRhAAAAAQAAAAAyMDIyAAAAIWFBUlQAAAAZZGF0YQAAAAEAAAAA5b2V6Z+z5py6AAAAMVh0cmEAAAApAAAAD1dNL0VuY29kaW5nVGltZQAAAAEAAAAOABUA2rD/dVfYAQ=="

        // creating a media player which
        // will play the audio of Default
        // ringtone in android device

        val mp3SoundByteArray = Base64.decode(base64, Base64.DEFAULT)

        val tempMp3: File = File.createTempFile("silent", ".mp3")
        tempMp3.deleteOnExit()
        val fos = FileOutputStream(tempMp3)
        fos.write(mp3SoundByteArray)
        fos.close()
        player = MediaPlayer.create(this, Uri.fromFile(tempMp3))
//        player.setDataSource(fis.getFD())
        // providing the boolean
        // value as true to play
        // the audio on loop
        player.isLooping = true
    }

    private fun initObserve() {
        ViewModelMain.apply {
            /**
             * 悬浮窗按钮的显示和隐藏
             */
            isVisible.observe(this@ProcessService) {
                floatRootView?.visibility = if (it) View.VISIBLE else View.GONE
            }
            /**
             * 悬浮窗按钮的创建和移除
             */
            isShowSuspendWindow.observe(this@ProcessService) {
                if (it) {
                    showWindow()
                } else {
                    if (!Utils.isNull(floatRootView)) {
                        if (!Utils.isNull(floatRootView?.windowToken)) {
                            if (!Utils.isNull(windowManager)) {
                                windowManager.removeView(floatRootView)
                            }
                        }
                    }
                }
            }
        }
    }

    private fun showWindow() {
        //获取WindowManager
        Log.d("--FloatWindow--","show window")
        windowManager = getSystemService(WINDOW_SERVICE) as WindowManager
        val outMetrics = applicationContext.resources.displayMetrics
        val layoutParam = WindowManager.LayoutParams().apply {
            /**
             * 设置type 这里进行了兼容
             */
            type = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                WindowManager.LayoutParams.TYPE_APPLICATION_OVERLAY
            } else {
                WindowManager.LayoutParams.TYPE_PHONE
            }
            format = PixelFormat.RGBA_8888
            flags = WindowManager.LayoutParams.FLAG_NOT_TOUCH_MODAL or WindowManager.LayoutParams.FLAG_NOT_FOCUSABLE
            //位置大小设置
            width = ViewGroup.LayoutParams.WRAP_CONTENT
            height = ViewGroup.LayoutParams.WRAP_CONTENT
            gravity = Gravity.START or Gravity.TOP
            //设置剧中屏幕显示
            x = outMetrics.widthPixels / 2 - width / 2
            y = outMetrics.heightPixels / 2 - height / 2
        }
        // 新建悬浮窗控件
        floatRootView = LayoutInflater.from(this).inflate(R.layout.activity_float_item, null)
        floatRootView?.setOnTouchListener(ItemViewTouchListener(layoutParam, windowManager))
        // 将悬浮窗控件添加到WindowManager
        windowManager.addView(floatRootView, layoutParam)
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        super.onStartCommand(intent, flags, startId)
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

        if (intent != null) {
            if (intent.getBooleanExtra("alive_media", false)) {
                player.start()
            }
            if (intent.getBooleanExtra("alive_wakelock", false)) {
                val powerManager = getSystemService(Context.POWER_SERVICE) as PowerManager
                val wakeLock = powerManager.newWakeLock(PowerManager.PARTIAL_WAKE_LOCK, "sealdice:LocationManagerService")
                wakeLock.acquire()
                val connectivityManager = getSystemService(Context.CONNECTIVITY_SERVICE) as ConnectivityManager
                val networkRequest = NetworkRequest.Builder()
                    .addCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
                    .build()

                val networkCallback = object : ConnectivityManager.NetworkCallback() {
                    override fun onAvailable(network: Network) {
                        // Network is now available
                    }
                }
                connectivityManager.registerNetworkCallback(networkRequest, networkCallback)
            }
        }

        if (!isRunning) {
            isRunning = true
            if (processBuilder.directory(File(this.filesDir.absolutePath)).environment().containsKey("PATH")) {
                Log.e("ProcessService", "PATH: " + processBuilder.directory(File(this.filesDir.absolutePath)).environment()["PATH"])
                processBuilder.environment()["PATH"] = processBuilder.directory(File(this.filesDir.absolutePath)).environment()["PATH"] + ":${this.filesDir.absolutePath}/sealdice"
            } else {
                processBuilder.environment()["PATH"] = "${this.filesDir.absolutePath}/sealdice"
            }
            val sharedPreferences = PreferenceManager.getDefaultSharedPreferences(this)
            processBuilder.environment()["LD_LIBRARY_PATH"] = this.filesDir.absolutePath + sharedPreferences.getString("ld_library_path", "/sealdice/lagrange/openssl-1.1")
            processBuilder.environment()["CLR_OPENSSL_VERSION_OVERRIDE"] = "1.1"
            processBuilder.environment()["RUNNER_PATH"] = this.filesDir.absolutePath + "/runner"
//            var env = arrayOf("LD_LIBRARY_PATH=${this.filesDir.absolutePath + sharedPreferences.getString("ld_library_path", "/sealdice/lagrange/openssl-1.1")}", "CLR_OPENSSL_VERSION_OVERRIDE=1.1")
            process = processBuilder.start()
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
    override fun onBind(intent: Intent): IBinder {
        super.onBind(intent)
        return binder
    }
    override fun onDestroy() {
        super.onDestroy()
        stopProcess()
    }
}