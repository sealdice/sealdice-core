package com.sealdice.dice

import androidx.lifecycle.LifecycleService
import android.content.Intent
import android.graphics.PixelFormat
import android.os.Build
import android.util.DisplayMetrics
import android.util.Log
import android.view.*
import com.sealdice.dice.utils.Utils
import com.sealdice.dice.utils.ViewModelMain

class FloatWindowService : LifecycleService(){
    private lateinit var windowManager: WindowManager
    private var floatRootView: View? = null//悬浮窗View
    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        super.onStartCommand(intent, flags, startId)
        return START_STICKY
    }
    override fun onCreate() {
        super.onCreate()
        initObserve()
    }
    private fun initObserve() {
        ViewModelMain.apply {
            /**
             * 悬浮窗按钮的显示和隐藏
             */
            isVisible.observe(this@FloatWindowService) {
                floatRootView?.visibility = if (it) View.VISIBLE else View.GONE
            }
            /**
             * 悬浮窗按钮的创建和移除
             */
            isShowSuspendWindow.observe(this@FloatWindowService) {
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
        val outMetrics = DisplayMetrics()
        windowManager.defaultDisplay.getMetrics(outMetrics)
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
}