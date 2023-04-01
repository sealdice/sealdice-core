package com.sealdice.dice

import android.app.Service
import android.content.Context
import android.content.Intent
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import android.os.Handler
import android.os.IBinder
import android.os.Looper
import android.os.PowerManager

class WakeLockService : Service(){
    private lateinit var wakeLockHelper: WakeLockHelper
    private val handler = Handler(Looper.getMainLooper())

    private val task = object : Runnable {
        override fun run() {
            wakeLockHelper.restartWakeLock(5000, 5000) // 每隔5秒释放Wakelock并获取新的Wakelock
            handler.postDelayed(this, 300000) // 每隔5分钟执行一次
        }
    }

    override fun onCreate() {
        super.onCreate()
        wakeLockHelper = WakeLockHelper(this)
        wakeLockHelper.acquireWakeLock()
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        handler.postDelayed(task, 300000) // 开始执行任务
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
        return START_STICKY
    }

    override fun onDestroy() {
        super.onDestroy()
        handler.removeCallbacks(task)
        wakeLockHelper.releaseWakeLock()
    }

    override fun onBind(intent: Intent?): IBinder? {
        return null
    }

    class WakeLockHelper(private val context: Context) {
        private var wakeLock: PowerManager.WakeLock? = null

        fun acquireWakeLock() {
            val powerManager = context.getSystemService(Context.POWER_SERVICE) as PowerManager
            wakeLock = powerManager.newWakeLock(PowerManager.PARTIAL_WAKE_LOCK, "SealDice::SealDiceWakelockTag")
            wakeLock?.acquire(10*60*1000L /*10 minutes*/)
        }

        fun releaseWakeLock() {
            wakeLock?.let {
                if (it.isHeld) {
                    it.release()
                }
            }
            wakeLock = null
        }

        fun restartWakeLock(releaseDelayMillis: Long, acquireDelayMillis: Long) {
            releaseWakeLock()
            Thread.sleep(releaseDelayMillis)
            acquireWakeLock()
            Thread.sleep(acquireDelayMillis)
        }
    }
}