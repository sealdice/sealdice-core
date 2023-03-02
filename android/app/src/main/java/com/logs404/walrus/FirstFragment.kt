package com.logs404.walrus

import android.content.Context
import android.content.DialogInterface
import android.content.Intent
import android.net.Uri
import android.os.Bundle
import android.util.Log
import android.view.*
import androidx.appcompat.app.AlertDialog
import androidx.core.app.ActivityCompat.finishAffinity
import androidx.fragment.app.Fragment
import androidx.preference.PreferenceManager
import com.logs404.walrus.common.ExtractAssets
import com.logs404.walrus.databinding.FragmentFirstBinding
import com.logs404.walrus.utils.Utils
import kotlinx.coroutines.*
import com.logs404.walrus.utils.ViewModelMain
import kotlin.system.exitProcess


/**
 * A simple [Fragment] subclass as the default destination in the navigation.
 */
class FirstFragment : Fragment() {

    private lateinit var _binding: FragmentFirstBinding

    // This property is only valid between onCreateView and
    // onDestroyView.
    private val binding get() = _binding
    private var shellLogs = ""

    override fun onCreateView(
        inflater: LayoutInflater, container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View {

        _binding = FragmentFirstBinding.inflate(inflater, container, false)
        return binding.root

    }

    @OptIn(DelicateCoroutinesApi::class)
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        var isrun = false
        val sharedPreferences = context?.let { PreferenceManager.getDefaultSharedPreferences(it) }
        binding.buttonThird.setOnClickListener {
            val address = sharedPreferences?.getString("ui_address", "http://127.0.0.1:3211")
            if (sharedPreferences?.getBoolean("use_internal_webview", true) == true) {
                val intent = Intent(context, WebViewActivity::class.java)
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
        binding.buttonExit.setOnClickListener {
            this.activity?.let { it1 -> finishAffinity(it1) } // Finishes all activities.
            exitProcess(0)
        }
        binding.buttonConsole.setOnClickListener {
            val alertDialogBuilder = context?.let { it1 ->
                AlertDialog.Builder(
                    it1
                )
            }
            alertDialogBuilder?.setTitle("控制台")
            alertDialogBuilder?.setMessage(shellLogs)
            alertDialogBuilder?.setPositiveButton("确定") { _: DialogInterface, _: Int ->
            }
            alertDialogBuilder?.create()?.show()
        }
        binding.buttonFirst.setOnClickListener {
//            findNavController().navigate(R.id.action_FirstFragment_to_SecondFragment)
            when (val text = binding.shellText.text.toString()) {
                "gin" -> {
                    ExtractAssets(context).extractResource("gin-arm")
//                    shellLogs += "gin"
                }
                "sealdice" -> {
                    if (isrun) {
                        shellLogs += "sealdice is running"
                        val alertDialogBuilder = context?.let { it1 ->
                            AlertDialog.Builder(
                                it1
                            )
                        }
                        alertDialogBuilder?.setTitle("提示")
                        alertDialogBuilder?.setMessage("请先重启APP后再启动海豹核心")
                        alertDialogBuilder?.setPositiveButton("确定") { _: DialogInterface, _: Int ->
                        }
                        alertDialogBuilder?.create()?.show()
                    } else {
                        isrun = true
                        ExtractAssets(context).extractResources("sealdice")
                        val args = sharedPreferences?.getString("launch_args", "")
                        execShell("cd sealdice&&./sealdice-core $args",true)
                        binding.buttonSecond.visibility = View.VISIBLE
                        binding.buttonFirst.visibility = View.GONE
                        if (!launchAliveService(context)) {
                            val alertDialogBuilder = context?.let { it1 ->
                                AlertDialog.Builder(
                                    it1
                                )
                            }
                            alertDialogBuilder?.setTitle("提示")
                            alertDialogBuilder?.setMessage("似乎并没有开启任何保活策略，这可能导致后台被清理")
                            alertDialogBuilder?.setPositiveButton("确定") { _: DialogInterface, _: Int ->
//                                                finish()
                            }
                            alertDialogBuilder?.create()?.show()
                        }
                        GlobalScope.launch(context = Dispatchers.IO) {
                            for (i in 0..10) {
                                withContext(Dispatchers.Main) {
                                    binding.textviewFirst.text = "正在启动...\n请等待${10 - i}s..."
                                }
                                Thread.sleep(1000)
                            }
                            withContext(Dispatchers.Main){
                            binding.textviewFirst.text = "启动完成（或者失败）"
                            }
                            val address = sharedPreferences?.getString("ui_address", "http://127.0.0.1:3211")

                            if (sharedPreferences?.getBoolean("use_internal_webview", true) == true) {
                                val intent = Intent(context, WebViewActivity::class.java)
                                intent.putExtra("url", address)
                                startActivity(intent)
                            } else {
                                val uri = Uri.parse(address)
                                val intent = Intent()
                                intent.action = "android.intent.action.VIEW"
                                intent.data = uri
                                startActivity(intent)
                            }
//                            binding.buttonFirst.text = "停止"
//                            binding.shellText.setText("stop")
                        }
                    }
                }
                "clear" -> {
                    clear()
                }
                "cls" -> {
                    clear()
                }
                else -> {
                    execShell(text,false)

                }
            }
            binding.buttonSecond.setOnClickListener {
                binding.buttonSecond.visibility = View.GONE
                execShell("pkill -SIGINT sealdice-core",false)
                binding.buttonFirst.visibility = View.VISIBLE
            }
        }

    }

    private fun clear() {
        shellLogs = ""
        binding.textviewFirst.text = shellLogs
    }

    private fun launchAliveService(context: Context?) : Boolean{
        val sharedPreferences = context?.let { PreferenceManager.getDefaultSharedPreferences(it) }
        var executed = false
        if (sharedPreferences != null) {
            if (sharedPreferences.getBoolean("alive_notification", true)) {
                val intentNoti = Intent(context, NotificationService::class.java)
                context.startService(intentNoti)
                executed = true
            }
            if (sharedPreferences.getBoolean("alive_media", false)) {
                val intentMedia = Intent(context, MediaService::class.java)
                context.startService(intentMedia)
                executed = true
            }
            if (sharedPreferences.getBoolean("alive_wakelock", true)) {
                val intentWakelock = Intent(context, WakeLockService::class.java)
                context.startService(intentWakelock)
                executed = true
            }
            if (sharedPreferences.getBoolean("alive_floatwindow", false)) {
                context.startService(Intent(context, FloatWindowService::class.java))
                this.activity?.let {
                    Utils.checkSuspendedWindowPermission(it) {
                        ViewModelMain.isShowSuspendWindow.postValue(true)
                        ViewModelMain.isVisible.postValue(true)
                    }
                }
                executed = true
            }
        }
        return executed
    }
    @OptIn(DelicateCoroutinesApi::class)
    private fun execShell(cmd: String, recordLog: Boolean) {
        GlobalScope.launch(context = Dispatchers.IO) {
            val process = Runtime.getRuntime().exec("sh")
            val os = process.outputStream
            os.write("cd ${context?.filesDir?.absolutePath}&&".toByteArray())
            os.write(cmd.toByteArray())
            os.flush()
            os.close()
//            Thread.sleep(3000)
//            val data = process.inputStream.readBytes()
//            val error = process.errorStream.readBytes()
//            if (data.isNotEmpty()) {
//                shellLogs += String(data)
//                shellLogs += "\n"
//            } else {
//                shellLogs += String(error)
//                shellLogs += "\n"
//            }
//            Log.i("ExecShell", shellLogs)
//            withContext(Dispatchers.Main) {
//                binding.textviewFirst.text = shellLogs
//            }
            while (recordLog) {
                val data = process.inputStream.readBytes()
                val error = process.errorStream.readBytes()
                if (data.isNotEmpty()) {
                    shellLogs += String(data)
                    shellLogs += "\n"
                } else {
                    shellLogs += String(error)
                    shellLogs += "\n"
                }
                Log.i("ExecShell", shellLogs)
                withContext(Dispatchers.Main) {
                binding.textviewFirst.text = shellLogs
                }
                Thread.sleep(1000)
            }
        }
    }
}