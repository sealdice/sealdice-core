package com.logs404.walrus

import android.content.Intent
import android.net.Uri
import android.os.Bundle
import android.support.design.widget.Snackbar
import android.support.v7.app.AppCompatActivity
import android.view.Menu
import android.view.MenuItem
import androidx.navigation.findNavController
import androidx.navigation.ui.AppBarConfiguration
import androidx.navigation.ui.navigateUp
import androidx.navigation.ui.setupActionBarWithNavController
import com.logs404.walrus.databinding.ActivityMainBinding

class MainActivity : AppCompatActivity() {

    private lateinit var appBarConfiguration: AppBarConfiguration
    private lateinit var binding: ActivityMainBinding

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        startService(Intent(this, NewService::class.java))
        binding = ActivityMainBinding.inflate(layoutInflater)
        setContentView(binding.root)

        setSupportActionBar(binding.toolbar)

        val navController = findNavController(R.id.nav_host_fragment_content_main)
        appBarConfiguration = AppBarConfiguration(navController.graph)
        setupActionBarWithNavController(navController, appBarConfiguration)
//        binding.
        binding.fab.setOnClickListener { view ->
            Snackbar.make(view, "SealDice For Android\nSpecial Thanks: 木末君(logs404)", Snackbar.LENGTH_LONG)
                .setAction("Action", null).show()
//            ExtractAssets(this).extractResources("sealdice")
        }

//        val alertDialogBuilder = AlertDialog.Builder(this)
//        alertDialogBuilder.setTitle("提示")
//        alertDialogBuilder.setMessage("此app非海豹官方app，随时可能停止维护，并且不兼容后续海豹官方app")
//        alertDialogBuilder.setPositiveButton("确定") { _: DialogInterface, _: Int ->
////            finish()
//        }
//        val alertDialog = alertDialogBuilder.create().show()

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
                val intent = Intent()
                intent.action = "android.intent.action.VIEW"
                intent.data = Uri.parse("https://sealdice.com")
                startActivity(intent)
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