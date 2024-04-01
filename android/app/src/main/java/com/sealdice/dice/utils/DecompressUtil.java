package com.sealdice.dice.utils;

import android.util.Log;

import java.io.*;

public class DecompressUtil {


    public static void decompressTarSys(String tarFilePath, String outputDir) throws IOException {
        // make sure the output directory exists
        ProcessBuilder pb = new ProcessBuilder("sh");
        pb.redirectErrorStream(true);
        Process p = pb.start();
        try (BufferedWriter bw = new BufferedWriter(new OutputStreamWriter(p.getOutputStream()));
             BufferedReader br = new BufferedReader(new InputStreamReader(p.getInputStream()))) {
            bw.write("mkdir "+ outputDir + "&&tar xf " + tarFilePath + " -C " + outputDir + "\n");
            Log.d("DecompressUtilSys", "tar xf " + tarFilePath + " -C " + outputDir);
            bw.write("exit\n");
            bw.flush();
            p.waitFor();
            String line;
            while ((line = br.readLine()) != null) {
                Log.d("DecompressUtilSys", line);
            }
        } catch (InterruptedException e) {
            e.printStackTrace();
        }
    }
}
