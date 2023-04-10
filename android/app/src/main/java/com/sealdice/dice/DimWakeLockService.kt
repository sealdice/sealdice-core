package com.sealdice.dice

import android.app.Service
import android.content.Context
import android.content.Intent
import android.os.IBinder
import android.os.PowerManager

class DimWakeLockService : Service(){
    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val powerManager = getSystemService(Context.POWER_SERVICE) as PowerManager
        val wakeLock = powerManager.newWakeLock(PowerManager.SCREEN_DIM_WAKE_LOCK, "DIM:LocationManagerService")
        wakeLock.acquire()
        return START_STICKY
    }
    override fun onBind(intent: Intent?): IBinder? {
        return null
    }
}