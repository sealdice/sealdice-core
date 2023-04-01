package com.sealdice.dice

import android.content.DialogInterface
import android.content.Intent
import android.graphics.Color
import android.net.Uri
import android.os.Bundle
import android.provider.Settings
import android.util.Log
import android.view.Menu
import android.view.MenuItem
import android.view.View
import android.widget.TextView
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AlertDialog
import androidx.appcompat.app.AppCompatActivity
import androidx.core.view.MenuItemCompat
import androidx.core.view.forEach
import androidx.navigation.findNavController
import androidx.navigation.ui.AppBarConfiguration
import androidx.navigation.ui.navigateUp
import androidx.navigation.ui.setupActionBarWithNavController
import com.google.android.material.snackbar.Snackbar
import com.sealdice.dice.common.FileWrite
import com.sealdice.dice.databinding.ActivityMainBinding
import com.tencent.smtt.export.external.TbsCoreSettings
import com.tencent.smtt.sdk.QbSdk
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.GlobalScope
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import okhttp3.OkHttpClient
import okhttp3.Request
import org.json.JSONException
import org.json.JSONObject
import java.io.IOException


private class PreInitCallbackImpl: QbSdk.PreInitCallback {
    override fun onCoreInitFinished() {
    }

    override fun onViewInitFinished(p0: Boolean) {
    }
}

class MainActivity : AppCompatActivity() {
    private val requestIgnoreBatteryOptimizations = registerForActivityResult(
        ActivityResultContracts.StartActivityForResult()
    ) {}
    private lateinit var appBarConfiguration: AppBarConfiguration
    private lateinit var binding: ActivityMainBinding

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val intentUpdateService = Intent(this, UpdateService::class.java)
        startService(intentUpdateService)
        QbSdk.initX5Environment(this, PreInitCallbackImpl())
        val map = HashMap<String?, Any?>()
        map[TbsCoreSettings.TBS_SETTINGS_USE_SPEEDY_CLASSLOADER] = true
        map[TbsCoreSettings.TBS_SETTINGS_USE_DEXLOADER_SERVICE] = true
        QbSdk.initTbsSettings(map)
        binding = ActivityMainBinding.inflate(layoutInflater)
        setContentView(binding.root)
        setSupportActionBar(binding.toolbar)
        val packageManager = this.packageManager
        val packageName = this.packageName
        val packageInfo = packageManager.getPackageInfo(packageName, 0)
        val versionName = packageInfo.versionName
        val navController = findNavController(R.id.nav_host_fragment_content_main)
        appBarConfiguration = AppBarConfiguration(navController.graph)
        setupActionBarWithNavController(navController, appBarConfiguration)
//        binding.fab.setOnClickListener { view ->
//            Snackbar.make(view, "SealDice for Android $versionName\nSpecial Thanks: 木末君(logs404)", Snackbar.LENGTH_LONG)
//                .setAction("Action", null).show()
////            ExtractAssets(this).extractResources("sealdice")
//        }

    }

    override fun onCreateOptionsMenu(menu: Menu): Boolean {
        // Inflate the menu; this adds items to the action bar if it is present.
        menuInflater.inflate(R.menu.menu_main, menu)
        return true
    }

    override fun onOptionsItemSelected(item: MenuItem): Boolean {
        // Handle action bar item clicks here. The action bar will
        // automatically handle clicks on the Home/Up button, so long
        // as you specify a parent activity in AndroidManifest.xml.
        return when (item.itemId) {
            R.id.action_settings -> {
                val intent = Intent(this, SettingsActivity::class.java)
                startActivity(intent)
                true
            }
            R.id.action_official_website -> {
                val intent = Intent()
                intent.action = "android.intent.action.VIEW"
                intent.data = Uri.parse("https://sealdice.com")
                startActivity(intent)
                true
            }
            R.id.action_battery_setting -> {
                requestIgnoreBatteryOptimizations.launch(Intent(
                    Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS,
                    Uri.parse("package:${this.packageName}")
                ))
                true
            }
            R.id.action_check_update -> {
                val builder: AlertDialog.Builder = AlertDialog.Builder(this)
                builder.setCancelable(false) // if you want user to wait for some process to finish,
                builder.setView(R.layout.layout_loading_dialog)
                val dialog = builder.create()
                dialog.show()
                val self = this
                GlobalScope.launch(context = Dispatchers.IO) {
                    val UPDATE_URL = "https://get.sealdice.com/seal/version/android"
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
                            withContext(Dispatchers.Main) {
                                dialog.cancel()
                                val alertDialogBuilder = AlertDialog.Builder(self,R.style.Theme_Mshell_DialogOverlay)
                                alertDialogBuilder.setTitle("提示")
                                alertDialogBuilder.setMessage("发现更新，点击确定开始下载新版本\n线上版本:${latestVersion}\n本地版本:${currentVersion}")
                                alertDialogBuilder.setPositiveButton("确定") { _: DialogInterface, _: Int ->
                                    val uri = Uri.parse("https://d.catlevel.com/seal/android/latest")
                                    val intent = Intent()
                                    intent.action = "android.intent.action.VIEW"
                                    intent.data = uri
                                    startActivity(intent)
                                }
                                alertDialogBuilder.setNegativeButton("取消") {_: DialogInterface, _: Int ->}
                                alertDialogBuilder.create().show()
                            }
                        } else {
                        // Current version is up-to-date
                            withContext(Dispatchers.Main) {
                                dialog.cancel()
                                val alertDialogBuilder = AlertDialog.Builder(self,R.style.Theme_Mshell_DialogOverlay)
                                alertDialogBuilder.setTitle("提示")
                                alertDialogBuilder.setMessage("当前版本已是最新")
                                alertDialogBuilder.setPositiveButton("确定") { _: DialogInterface, _: Int ->}
                                alertDialogBuilder.create().show()
                            }
                        }
                    } catch (e: Exception) {
                        dialog.cancel()
                        e.printStackTrace()
                        withContext(Dispatchers.Main) {
                            dialog.cancel()
                            val alertDialogBuilder = AlertDialog.Builder(self,R.style.Theme_Mshell_DialogOverlay)
                            alertDialogBuilder.setTitle("提示")
                            alertDialogBuilder.setMessage("检查更新时出现错误，请稍后重试")
                            alertDialogBuilder.setPositiveButton("确定") { _: DialogInterface, _: Int ->}
                            alertDialogBuilder.create().show()
                        }
                    }
                }
                true
            }
            else -> super.onOptionsItemSelected(item)
        }
    }

    override fun onSupportNavigateUp(): Boolean {
        val navController = findNavController(R.id.nav_host_fragment_content_main)
        return navController.navigateUp(appBarConfiguration)
                || super.onSupportNavigateUp()
    }
}