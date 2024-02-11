package com.sealdice.dice


import android.content.SharedPreferences
import android.os.Bundle
import android.widget.Toast
import androidx.preference.Preference
import androidx.preference.PreferenceFragmentCompat
import androidx.preference.SeekBarPreference
import androidx.preference.SwitchPreferenceCompat
import com.sealdice.dice.utils.Utils


class SettingsFragment : PreferenceFragmentCompat() {
    private lateinit var sharedPreferences: SharedPreferences
    override fun onCreatePreferences(savedInstanceState: Bundle?, rootKey: String?) {
        setPreferencesFromResource(R.xml.preferences, rootKey)
        sharedPreferences = preferenceScreen.sharedPreferences!!
        // 查找后台删除对应的配置项，并为其设置监听器
        val mySwitchPreference: SwitchPreferenceCompat? = findPreference("alive_excluderecents")
        mySwitchPreference?.onPreferenceChangeListener =
            Preference.OnPreferenceChangeListener { preference, newValue ->
                // 在此处执行你想要的操作，例如根据新值来改变应用程序的行为
                val isChecked = newValue as Boolean
                if(!isChecked){
                    Toast.makeText(context,"豹豹已准备上浮",Toast.LENGTH_SHORT).show()
                }
                else{
                    Toast.makeText(context,"豹豹已准备下潜",Toast.LENGTH_SHORT).show()
                }
                Utils.setHideTaskStatus(isChecked)
                true // 返回 true 表示消费了设置项的变化事件
            }
        val mySeekBarPreference : SeekBarPreference? = findPreference<SeekBarPreference>("launch_waiting_time")
//        val switchPreference: SwitchPreferenceCompat = findPreference("my_switch_preference")
//        switchPreference.setTitleTextColor(resources.getColor(android.R.color.my_color))
    }
}
