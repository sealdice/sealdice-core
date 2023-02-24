package com.logs404.walrus

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.Service
import android.content.Context
import android.content.Intent
import android.media.MediaPlayer
import android.net.Uri
import android.os.IBinder
import android.util.*
import java.io.File
import java.io.FileOutputStream

class MediaService : Service() {

    // declaring object of MediaPlayer
    private lateinit var player: MediaPlayer

    // execution of service will start
    // on calling this method
    override fun onStartCommand(intent: Intent, flags: Int, startId: Int): Int {
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
        // starting the process
        player.start()

        // returns the status
        // of the program
        return START_STICKY
    }

    // execution of the service will
    // stop on calling this method
    override fun onDestroy() {
        super.onDestroy()

        // stopping the process
        player.stop()
//        val intent = Intent(applicationContext, MediaService::class.java)
//        startService(intent)
    }

    override fun onBind(intent: Intent): IBinder? {
        return null
    }
}
