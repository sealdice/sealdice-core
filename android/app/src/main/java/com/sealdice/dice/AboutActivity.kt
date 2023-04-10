package com.sealdice.dice

import android.content.Intent
import android.net.Uri
import android.os.Bundle
import android.view.MenuItem
import android.widget.Button
import android.widget.TextView
import androidx.appcompat.app.AppCompatActivity

class AboutActivity : AppCompatActivity(){
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        supportActionBar?.setDisplayHomeAsUpEnabled(true)
        setContentView(R.layout.activity_about)
        findViewById<TextView>(R.id.about_app_version).text = "App Version: " + BuildConfig.VERSION_NAME
        val buttonReport = findViewById<Button>(R.id.button_report)
        buttonReport.setOnClickListener {
            val intent = Intent("android.intent.action.VIEW")
            intent.data = Uri.parse("https://github.com/sealdice/sealdice-android/issues/new/choose")
            startActivity(intent)
        }
        val buttonRepo = findViewById<Button>(R.id.button_repo)
        buttonRepo.setOnClickListener {
            val intent = Intent("android.intent.action.VIEW")
            intent.data = Uri.parse("https://github.com/sealdice/sealdice-android")
            startActivity(intent)
        }
    }
    override fun onOptionsItemSelected(item: MenuItem): Boolean {
        return when (item.itemId) {
            android.R.id.home -> {
                onBackPressedDispatcher.onBackPressed()
                finish()
                true
            }
            else -> super.onOptionsItemSelected(item)
        }
    }
}