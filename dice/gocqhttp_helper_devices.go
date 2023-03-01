//go:build !prod
// +build !prod

package dice

import (
	"encoding/json"
	"strings"
)

var deviceData = `小米11:{"display": "SKQ1.211006.001 test-keys","product": "venus","device": "venus","board": "venus","model": "M2011K2C","finger_print": "Xiaomi\/venus\/venus:12\/SKQ1.211006.001\/V13.0.9.0.SKBCNXM:user\/release-keys","boot_id": "unknown","proc_version": "","imei": "","brand": "Xiaomi","bootloader": "unknown","base_band": "4.3CPL2-17.2-6980.37-0718_1742_ae22bde8045,4.3CPL2-17.2-6980.37-0718_1742_ae22bde8045","sim_info": "","os_type": "android","mac_address": "","ip_address": "","wifi_bssid": "02:00:00:00:00:00","wifi_ssid": "<unknown ssid>","imsi_md5": "","android_id": "","apn": "wifi","vendor_name": "Xiaomi","vendor_os_name": "Xiaomi","version": {"incremental": "V13.0.9.0.SKBCNXM","release": "12","codename": "REL","sdk": 31}}
荣耀x10:{"display": "TEL-AN10 2.0.0.270(C00E230R7P5)","product": "TEL-AN10","device": "HWTEL-H","board": "TEL","model": "TEL-AN10","finger_print": "HONOR\/TEL-AN10\/HWTEL-H:10\/HONORTEL-AN10\/102.0.0.270C00:user\/release-keys","boot_id": "unknown","proc_version": "","imei": "","brand": "HONOR","bootloader": "unknown","base_band": "21C93B377S000C000,21C93B377S000C000","sim_info": "46003","os_type": "android","mac_address": "","ip_address": "","wifi_bssid": "02:00:00:00:00:00","wifi_ssid": "<unknown ssid>","imsi_md5": "","android_id": "","apn": "ctnet","vendor_name": "HUAWEI","vendor_os_name": "HUAWEI","version": {"incremental": "102.0.0.270C00","release": "10","codename": "REL","sdk": 29}}`

func gocqhttpRandomDeviceAndroidInit() []struct {
	name string
	data *deviceFile
} {
	var pool []struct {
		name string
		data *deviceFile
	}
	allItems := strings.Split(deviceData, "\n")
	for _, i := range allItems {
		n := strings.SplitN(i, ":", 2)
		if len(n) >= 2 {
			name := n[0]
			data := strings.ReplaceAll(n[1], `"ip_address": ""`, `"ip_address": []`)
			v := deviceFile{}
			err := json.Unmarshal([]byte(data), &v)
			if err != nil {
				continue
			}
			if name == "" {
				name = v.Model
			}

			pool = append(pool, struct {
				name string
				data *deviceFile
			}{name: name, data: &v})
		}
	}
	return pool
}

var androidDevicePool = gocqhttpRandomDeviceAndroidInit()
