package com.sealdice.dice


import android.content.SharedPreferences
import android.os.Bundle
import android.widget.SeekBar
import android.widget.SeekBar.OnSeekBarChangeListener
import androidx.preference.PreferenceFragmentCompat
import androidx.preference.SeekBarPreference


class SettingsFragment : PreferenceFragmentCompat() {
    private lateinit var sharedPreferences: SharedPreferences
    override fun onCreatePreferences(savedInstanceState: Bundle?, rootKey: String?) {
        setPreferencesFromResource(R.xml.prefrences, rootKey)
        sharedPreferences = preferenceScreen.sharedPreferences!!
        val mySeekBarPreference : SeekBarPreference? = findPreference<SeekBarPreference>("launch_waiting_time")
    }
}
