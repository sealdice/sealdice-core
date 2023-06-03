package com.sealdice.dice.common;

import android.Manifest;
import android.annotation.SuppressLint;
import android.app.Activity;
import android.content.Context;
import android.content.pm.PackageManager;
import android.net.ConnectivityManager;
import android.net.NetworkInfo;
import android.net.wifi.WifiInfo;
import android.net.wifi.WifiManager;
import android.os.Build;
import android.provider.Settings;
import android.telephony.SubscriptionInfo;
import android.telephony.SubscriptionManager;
import android.telephony.TelephonyManager;

import androidx.core.app.ActivityCompat;

import org.json.JSONException;
import org.json.JSONObject;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.util.List;
import java.util.Locale;

public class DeviceInfo {
    private Context context;
    private WifiManager wifiManager;

    public DeviceInfo(Context context) {
        this.context = context;
        wifiManager = (WifiManager) context.getApplicationContext().getSystemService(Context.WIFI_SERVICE);
    }

    public JSONObject getDeviceInfo() {
        JSONObject json = new JSONObject();
        try {
            json.put("display", Build.DISPLAY);
            json.put("product", Build.PRODUCT);
            json.put("device", Build.DEVICE);
            json.put("board", Build.BOARD);
            json.put("model", Build.MODEL);
            json.put("finger_print", Build.FINGERPRINT);
            json.put("boot_id", Build.BOOTLOADER);
            json.put("proc_version", getProcVersion());
            json.put("imei", getIMEI());
            json.put("brand", Build.BRAND);
            json.put("bootloader", Build.BOOTLOADER);
            json.put("base_band", getBaseBandVersion());
            json.put("sim_info", getSimInfo());
            json.put("os_type", "android");
            json.put("mac_address", getMacAddress());
            json.put("ip_address", getIpAddress());
            json.put("wifi_bssid", getWifiBssid());
            json.put("wifi_ssid", getWifiSsid());
            json.put("imsi_md5", getIMSIMd5());
            json.put("android_id", getAndroidID());
            json.put("apn", getAPN());
            json.put("vendor_name", Build.MANUFACTURER);
            json.put("vendor_os_name", Build.MANUFACTURER);
            json.put("version", getVersion());
        } catch (JSONException e) {
            e.printStackTrace();
        }
        return json;
    }

    public JSONObject getVersion() {
        JSONObject json = new JSONObject();
        try {
            json.put("incremental", Build.VERSION.INCREMENTAL);
            json.put("release", Build.VERSION.RELEASE);
            json.put("codename", Build.VERSION.CODENAME);
            json.put("sdk", Build.VERSION.SDK_INT);
            return json;
        } catch (JSONException e) {
            e.printStackTrace();
            return null;
        }
    }

    private String getImsiFromSubscriptionManager() {
        String simSerialNo = "";
        try {
            SubscriptionManager subsManager = null;
            if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.LOLLIPOP_MR1) {
                subsManager = (SubscriptionManager) context.getSystemService(Context.TELEPHONY_SUBSCRIPTION_SERVICE);
            }

            if (ActivityCompat.checkSelfPermission(context, Manifest.permission.READ_PHONE_STATE) == PackageManager.PERMISSION_GRANTED) {
                List<SubscriptionInfo> subsList = null;
                if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.LOLLIPOP_MR1) {
                    subsList = subsManager.getActiveSubscriptionInfoList();
                }

                if (subsList != null) {
                    for (SubscriptionInfo subsInfo : subsList) {
                        if (subsInfo != null) {
                            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
                                simSerialNo = subsInfo.getMccString() + subsInfo.getMncString() + subsInfo.getIccId().substring(8, 18);
                            }
                        }
                    }
                } else {
                    simSerialNo = "WiFi";
                }
            }
        } catch (Exception e) {
            e.printStackTrace();
        }

        if(simSerialNo.isEmpty())
            simSerialNo = "N/A";

        return simSerialNo;
    }

    public String getIMSIMd5() {
        return getImsiFromSubscriptionManager();
//        return "";
    }

    // 获取Android ID
    public String getAndroidID() {
        @SuppressLint("HardwareIds") String androidId = Settings.Secure.getString(context.getContentResolver(), Settings.Secure.ANDROID_ID);
        return androidId != null ? androidId : "";
    }

    public String getAPN() {
        String apn = "";
        ConnectivityManager connectivityManager = (ConnectivityManager) context.getSystemService(Context.CONNECTIVITY_SERVICE);
        if (connectivityManager != null) {
            NetworkInfo networkInfo = connectivityManager.getActiveNetworkInfo();
            if (networkInfo != null && networkInfo.isConnected()) {
                if (networkInfo.getType() == ConnectivityManager.TYPE_WIFI) {
                    apn = "wifi";
                } else if (networkInfo.getType() == ConnectivityManager.TYPE_MOBILE) {
                    String extraInfo = networkInfo.getExtraInfo();
                    if (extraInfo != null) {
                        apn = extraInfo.toLowerCase(Locale.getDefault());
                    }
                }
            }
        }
        return apn;
    }

    public String getMacAddress() {
        WifiInfo wifiInfo = wifiManager.getConnectionInfo();
        if (ActivityCompat.checkSelfPermission(context, Manifest.permission.ACCESS_FINE_LOCATION) != PackageManager.PERMISSION_GRANTED) {
            ActivityCompat.requestPermissions((Activity) context, new String[]{Manifest.permission.ACCESS_FINE_LOCATION}, 1);
        }
        return wifiInfo.getMacAddress();
    }

    public String getIpAddress() {
        return "";
    }

    public String getWifiBssid() {
        WifiInfo wifiInfo = wifiManager.getConnectionInfo();
        return wifiInfo.getBSSID();
    }

    public String getWifiSsid() {
        WifiInfo wifiInfo = wifiManager.getConnectionInfo();
        return wifiInfo.getSSID();
    }

    private String getProcVersion() {
        String line;
        StringBuilder sb = new StringBuilder();
        try {
            BufferedReader br = new BufferedReader(new InputStreamReader(Runtime.getRuntime().exec("cat /proc/version").getInputStream()));
            while ((line = br.readLine()) != null) {
                sb.append(line);
            }
            br.close();
        } catch (IOException e) {
            e.printStackTrace();
        }
        return sb.toString();
    }

    private String getIMEI() {
        return "";
    }

//    private String getIMEI2() {
//        String deviceId;
//
//        if (android.os.Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
//            deviceId = Settings.Secure.getString(
//                    context.getContentResolver(),
//                    Settings.Secure.ANDROID_ID);
//        } else {
//            final TelephonyManager mTelephony = (TelephonyManager) context.getSystemService(Context.TELEPHONY_SERVICE);
//            if (mTelephony.getDeviceId() != null) {
//                deviceId = mTelephony.getDeviceId();
//            } else {
//                deviceId = Settings.Secure.getString(
//                        context.getContentResolver(),
//                        Settings.Secure.ANDROID_ID);
//            }
//        }
//        return deviceId;
//    }

    private String getBaseBandVersion() {
        String baseBandVersion = Build.getRadioVersion();
        if (baseBandVersion == null) {
            baseBandVersion = "";
        }
        return baseBandVersion;
    }

    private String getSimInfo() {
        TelephonyManager tm = (TelephonyManager) context.getSystemService(Context.TELEPHONY_SERVICE);
        String simInfo = tm.getSimOperator();
        if (simInfo == null) {
            simInfo = "";
        }
        return simInfo;
    }
}