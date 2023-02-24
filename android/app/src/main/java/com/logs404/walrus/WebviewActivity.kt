package com.logs404.walrus

import android.net.http.SslError
import android.os.Bundle
import android.view.KeyEvent
import android.view.KeyEvent.KEYCODE_BACK
import android.webkit.*
import androidx.appcompat.app.AppCompatActivity


class WebViewActivity : AppCompatActivity() {
    private lateinit var mWebView: WebView
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

        webSettings.cacheMode = WebSettings.LOAD_CACHE_ELSE_NETWORK //关闭webview中缓存

        webSettings.allowFileAccess = true //设置可以访问文件

        webSettings.javaScriptCanOpenWindowsAutomatically = true //支持通过JS打开新窗口

        webSettings.loadsImagesAutomatically = true //支持自动加载图片

        webSettings.defaultTextEncodingName = "utf-8" //设置编码格式

        webSettings.mixedContentMode = WebSettings.MIXED_CONTENT_ALWAYS_ALLOW

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
        webView.webChromeClient = object : WebChromeClient() {
        }
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