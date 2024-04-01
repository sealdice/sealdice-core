package com.sealdice.dice.utils;

import java.io.*;
import java.nio.file.*;
import java.util.zip.GZIPInputStream;
import org.apache.commons.compress.archivers.tar.TarArchiveInputStream;

public class DecompressUtil {


    public static void decompressTar(String tarFilePath, String outputDir) throws IOException {
        try (FileInputStream fis = new FileInputStream(tarFilePath);
             TarArchiveInputStream tarArchiveInputStream = new TarArchiveInputStream(fis)) {
            org.apache.commons.compress.archivers.tar.TarArchiveEntry entry;

            while ((entry = (org.apache.commons.compress.archivers.tar.TarArchiveEntry) tarArchiveInputStream.getNextEntry()) != null) {
                final String individualFile = outputDir + File.separator + entry.getName();
                final File file = new File(individualFile);

                if (entry.isDirectory()) {
                    if (!file.exists()) {
                        file.mkdirs();
                    }
                } else {
                    int count;
                    byte[] data = new byte[1024];
                    FileOutputStream fos = new FileOutputStream(individualFile, false);
                    try (BufferedOutputStream dest = new BufferedOutputStream(fos, 1024)) {
                        while ((count = tarArchiveInputStream.read(data, 0, 1024)) != -1) {
                            dest.write(data, 0, count);
                        }
                    }
                }
            }
        }
    }

    public static void decompressTarGz(String tarGzFilePath, String outputDir) throws IOException {
        try (GZIPInputStream gis = new GZIPInputStream(Files.newInputStream(Paths.get(tarGzFilePath)));
             TarArchiveInputStream tarArchiveInputStream = new TarArchiveInputStream(gis)) {
            org.apache.commons.compress.archivers.tar.TarArchiveEntry entry;

            while ((entry = (org.apache.commons.compress.archivers.tar.TarArchiveEntry) tarArchiveInputStream.getNextEntry()) != null) {
                final Path outputPath = Paths.get(outputDir, entry.getName());

                if (entry.isDirectory()) {
                    Files.createDirectories(outputPath);
                } else if (entry.isSymbolicLink()) {
                    // 对符号链接的处理
                    Path target = Paths.get(entry.getLinkName());
                    Files.createSymbolicLink(outputPath, target);
                } else {
                    // 处理常规文件
                    Files.createDirectories(outputPath.getParent());
                    Files.copy(tarArchiveInputStream, outputPath, StandardCopyOption.REPLACE_EXISTING);
                }
            }
        }
    }
}
