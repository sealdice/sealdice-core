package com.logs404.walrus.utils

import androidx.lifecycle.MutableLiveData

object ViewModelMain {
    var isShowWindow = MutableLiveData<Boolean>()
    //悬浮窗口创建 移除

    var isShowSuspendWindow = MutableLiveData<Boolean>()

    //悬浮窗口显示 隐藏
    var isVisible = MutableLiveData<Boolean>()
}