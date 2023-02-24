package com.logs404.walrus

import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.Bundle
import android.view.KeyEvent
import android.view.KeyEvent.KEYCODE_BACK
import androidx.appcompat.app.AppCompatActivity
import com.tencent.smtt.export.external.interfaces.SslError
import com.tencent.smtt.export.external.interfaces.SslErrorHandler
import com.tencent.smtt.export.external.interfaces.WebResourceRequest
import com.tencent.smtt.sdk.*

private open class WVChromeClient(_context: Context, activity: WebViewActivity): WebChromeClient() {
    var _m: WebViewActivity? = activity
    private val TAG = "WebChromeClient："
    val CHOOSER_REQUEST = 0x33
    private var uploadFiles: ValueCallback<Array<Uri>>? = null
    var context: Context? = _context
    override fun onShowFileChooser(webView: WebView, filePathCallback: ValueCallback<Array<Uri>>?, fileChooserParams: WebChromeClient.FileChooserParams?): Boolean {
        uploadFiles = filePathCallback
        val i = fileChooserParams!!.createIntent()
        i.addCategory(Intent.CATEGORY_OPENABLE)
        i.putExtra(Intent.EXTRA_ALLOW_MULTIPLE, true) // 设置多选

        _m?.startActivityForResult(Intent.createChooser(i, "Image Chooser"), CHOOSER_REQUEST)
        return true
    }
    fun onActivityResultFileChooser(requestCode: Int, resultCode: Int, intent: Intent?) {
        if (requestCode != CHOOSER_REQUEST || uploadFiles == null) return
        var results: Array<Uri?>? = null
        if (resultCode == AppCompatActivity.RESULT_OK) {
            if (intent != null) {
                val dataString = intent.dataString
                val clipData = intent.clipData
                if (clipData != null) {
                    results = arrayOfNulls(clipData.itemCount)
                    for (i in 0 until clipData.itemCount) {
                        val item = clipData.getItemAt(i)
                        results[i] = item.uri
                    }
                }
                if (dataString != null) results = arrayOf(Uri.parse(dataString))
            }
        }
        uploadFiles!!.onReceiveValue(results)
        uploadFiles = null
    }
    private fun <T> ValueCallback<T>.onReceiveValue(results: Array<Uri?>?) {
        // Check if the results array is not null and has at least one item
        if (results != null && results.isNotEmpty()) {
            // Get the first Uri from the array
            val uri = results[0]
            // Call the original onReceiveValue method with the Uri as its argument
            this.onReceiveValue(uri as T)
        } else {
            // If the results array is null or empty, call the original onReceiveValue method with null as its argument
            this.onReceiveValue(null)
        }
    }

}
class WebViewActivity : AppCompatActivity() {
    private lateinit var mWebView: WebView
    private lateinit var mWebClient: WVChromeClient

    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        super.onActivityResult(requestCode, resultCode, data)
        if (requestCode == 0x33) { // 处理返回的文件
            mWebClient.onActivityResultFileChooser(
                requestCode,
                resultCode,
                data
            ) // 调用 WVChromeClient 类中的 回调方法
        }
    }
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_webview)

        val webView: WebView = findViewById(R.id.webview)

        val webSettings = webView.settings

        webSettings.javaScriptEnabled = true

        webSettings.useWideViewPort = true //将图片调整到适合webview的大小

        webSettings.loadWithOverviewMode = true // 缩放至屏幕的大小

        webSettings.setSupportZoom(true) //支持缩放，默认为true。是下面那个的前提。

        webSettings.builtInZoomControls = true //设置内置的缩放控件。若为false，则该WebView不可缩放

        webSettings.displayZoomControls = false //隐藏原生的缩放控件

        webSettings.cacheMode = WebSettings.LOAD_DEFAULT

        webSettings.allowFileAccess = true //设置可以访问文件

        webSettings.javaScriptCanOpenWindowsAutomatically = true //支持通过JS打开新窗口

        webSettings.loadsImagesAutomatically = true //支持自动加载图片

        webSettings.defaultTextEncodingName = "utf-8" //设置编码格式

        webSettings.domStorageEnabled = true
        val url = intent.getStringExtra("url")
        webView.webViewClient = object : WebViewClient() {
            override fun shouldOverrideUrlLoading(
                view: WebView?,
                request: WebResourceRequest?
            ): Boolean {
                view?.loadUrl(request?.url.toString())
                return true
            }
            override fun onReceivedSslError(view: WebView?, handler: SslErrorHandler?, error: SslError?) {
                handler?.proceed()
            }
        }
        mWebClient = WVChromeClient(applicationContext,this)
        webView.webChromeClient = mWebClient
        mWebView = webView
        CookieManager.getInstance().setAcceptThirdPartyCookies(webView, true)
        if (url != null) {
            webView.loadUrl(url)
        }
    }

    override fun onKeyDown(keyCode: Int, event: KeyEvent?): Boolean {
        if (keyCode == KEYCODE_BACK && mWebView.canGoBack()) {
            mWebView.goBack()
            return true
        }
        return super.onKeyDown(keyCode, event)
    }
}