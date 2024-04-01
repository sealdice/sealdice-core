package com.sealdice.dice

import android.app.ActivityManager
import android.content.Context
import android.content.DialogInterface
import android.content.Intent
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.view.MenuItem
import android.widget.Button
import android.widget.EditText
import android.widget.ImageView
import androidx.appcompat.app.AlertDialog
import androidx.appcompat.app.AppCompatActivity
import androidx.preference.PreferenceManager
import com.sealdice.dice.common.FileWrite
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.GlobalScope
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.io.BufferedReader
import java.io.File
import java.io.InputStreamReader

class DebugActivity : AppCompatActivity() {
    private var processBuilder: ProcessBuilder = ProcessBuilder("sh").redirectErrorStream(true)
    private lateinit var process: Process
    private var shellLogs = ""
    private var isInit = false
    private lateinit var outputStream : java.io.OutputStream
    private var isRunning = false
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_debug)
        supportActionBar?.setDisplayHomeAsUpEnabled(true)
        findViewById<Button>(R.id.DEBUG_button_abis).setOnClickListener {
            val alertDialogBuilder = AlertDialog.Builder(
                this, R.style.Theme_Mshell_DialogOverlay
            )
            alertDialogBuilder.setTitle("DEBUG:ABIS")
            alertDialogBuilder.setMessage("Support 64 abis:"+ Build.SUPPORTED_64_BIT_ABIS.contentToString()+"\nSupport 32 abis:"+Build.SUPPORTED_32_BIT_ABIS.contentToString()+"\nSupport abis:"+Build.SUPPORTED_ABIS.contentToString())
            alertDialogBuilder.setPositiveButton("确定") { _: android.content.DialogInterface, _: Int ->
            }
            alertDialogBuilder.create().show()
        }
        findViewById<Button>(R.id.DEBUG_button_force_ui).setOnClickListener {
            val sharedPreferences = PreferenceManager.getDefaultSharedPreferences(this)
            val address = sharedPreferences?.getString("ui_address", "http://127.0.0.1:3211")
            if (sharedPreferences?.getBoolean("use_internal_webview", true) == true) {
                val intent = Intent(this, WebViewActivity::class.java)
                intent.putExtra("url", address)
                startActivity(intent)
            } else {
                val uri = Uri.parse(address)
                val intent = Intent()
                intent.action = "android.intent.action.VIEW"
                intent.data = uri
                startActivity(intent)
            }
        }
        findViewById<Button>(R.id.DEBUG_button_exec).setOnClickListener {
            val command = findViewById<EditText>(R.id.DEBUG_edittext_command).text.toString()
            if (!isInit) {
                processBuilder.directory(File(this.filesDir.absolutePath))
                processBuilder.environment()["LD_LIBRARY_PATH"] = this.filesDir.absolutePath + "/sealdice/lagrange/openssl-1.1"
                processBuilder.environment()["CLR_OPENSSL_VERSION_OVERRIDE"] = "1.1"

                GlobalScope.launch(context = Dispatchers.IO) {
                    isRunning = true
                    process = processBuilder.start()
                    outputStream = process.outputStream
                    outputStream.write(command.toByteArray())
                    outputStream.write("\n".toByteArray())
                    outputStream.flush()
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
            }  else {
                outputStream.write(command.toByteArray())
                outputStream.write("\n".toByteArray())
                outputStream.flush()
            }
        }
        findViewById<Button>(R.id.DEBUG_button_get_current_dir).setOnClickListener {
            val alertDialogBuilder = AlertDialog.Builder(
                this, R.style.Theme_Mshell_DialogOverlay
            )
            alertDialogBuilder.setTitle("当前目录")
            alertDialogBuilder.setMessage(this.filesDir.absolutePath)
            alertDialogBuilder.setPositiveButton("确定") { _: DialogInterface, _: Int ->
            }
            alertDialogBuilder.create().show()
        }
        findViewById<Button>(R.id.DEBUG_button_console).setOnClickListener {
            val alertDialogBuilder = AlertDialog.Builder(
                this, R.style.Theme_Mshell_DialogOverlay
            )
            alertDialogBuilder.setTitle("DEBUG:Console")
            alertDialogBuilder.setMessage(shellLogs)
            alertDialogBuilder.setPositiveButton("确定") { _: DialogInterface, _: Int ->
            }
            alertDialogBuilder.create().show()
        }
        findViewById<Button>(R.id.DEBUG_button_remove_log).setOnClickListener {
            shellLogs = ""
        }
        findViewById<Button>(R.id.DEBUG_button_force_output_data).setOnClickListener {
            val builder: AlertDialog.Builder = AlertDialog.Builder(this, R.style.Theme_Mshell_DialogOverlay)
            builder.setCancelable(false) // if you want user to wait for some process to finish,
            builder.setView(R.layout.layout_loading_dialog)
            val dialog = builder.create()
            dialog.show()
            GlobalScope.launch(context = Dispatchers.IO) {
                FileWrite.FileCount = 0
                FileWrite.copyFolder(
                    File(FileWrite.getPrivateFileDir(this@DebugActivity)+"sealdice/"),
                    File("${FileWrite.SDCardDir}/Documents/${packageName}/sealdice/")
                )
                dialog.dismiss()
                val alertDialogBuilder = AlertDialog.Builder(
                    this@DebugActivity, R.style.Theme_Mshell_DialogOverlay
                )
                alertDialogBuilder.setTitle("提示")
                alertDialogBuilder.setMessage("所有内部数据已经导出至\n"+"${FileWrite.SDCardDir}/Documents/${packageName}/sealdice/\n共${FileWrite.FileCount}个文件")
                alertDialogBuilder.setPositiveButton("确定") { _: DialogInterface, _: Int -> }
                withContext(Dispatchers.Main) {
                    alertDialogBuilder.create().show()
                }
            }
        }
        findViewById<Button>(R.id.DEBUG_button_force_stop_service).setOnClickListener {
            GlobalScope.launch {
                this@DebugActivity.stopService(Intent(this@DebugActivity, ProcessService::class.java))
                this@DebugActivity.stopService(Intent(this@DebugActivity, MediaService::class.java))
                this@DebugActivity.stopService(Intent(this@DebugActivity, WakeLockService::class.java))
                this@DebugActivity.stopService(Intent(this@DebugActivity, FloatWindowService::class.java))
                this@DebugActivity.stopService(Intent(this@DebugActivity, HeartbeatService::class.java))
                this@DebugActivity.stopService(Intent(this@DebugActivity, UpdateService::class.java))
            }
        }
        findViewById<Button>(R.id.DEBUG_button_show_running_services).setOnClickListener {
            var text = ""
            val manager: ActivityManager = getSystemService(Context.ACTIVITY_SERVICE) as ActivityManager
            for (service in manager.getRunningServices(Int.MAX_VALUE)) {
                text += service.service.className + "\n"
            }
            val alertDialogBuilder = AlertDialog.Builder(
                this, R.style.Theme_Mshell_DialogOverlay
            )
            alertDialogBuilder.setTitle("DEBUG:Running Services")
            alertDialogBuilder.setMessage(text)
            alertDialogBuilder.setPositiveButton("确定") { _: DialogInterface, _: Int ->
            }
            alertDialogBuilder.create().show()
        }
        findViewById<Button>(R.id.DEBUG_crash).setOnClickListener {
            throw Exception("DEBUG:Crash")
        }
        findViewById<Button>(R.id.DEBUG_button_info).setOnClickListener {
            val alertDialogBuilder = AlertDialog.Builder(
                this, R.style.Theme_Mshell_DialogOverlay
            )
            alertDialogBuilder.setTitle("DEBUG:Info")
            alertDialogBuilder.setMessage("Version:"+BuildConfig.VERSION_NAME+"\nBuild:"+BuildConfig.VERSION_CODE+"\nPackage:"+packageName)
            alertDialogBuilder.setPositiveButton("确定") { _: DialogInterface, _: Int ->
            }
            alertDialogBuilder.create().show()
        }
        findViewById<ImageView>(R.id.DEBUG_app_icon).setOnClickListener {
            it.animate().scaleX(1.2f).scaleY(1.2f).setDuration(100).withEndAction {
                it.animate().scaleX(1f).scaleY(1f).setDuration(100).start()
            }.start()
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