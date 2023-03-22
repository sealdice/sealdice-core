package com.sealdice.dice.utils;

import android.Manifest;
import android.content.Context;
import android.content.pm.PackageManager;
import android.database.Cursor;
import android.database.MatrixCursor;
import android.os.CancellationSignal;
import android.os.Environment;
import android.os.ParcelFileDescriptor;
import android.provider.DocumentsContract;
import android.provider.DocumentsContract.*;
import android.provider.DocumentsProvider;
import android.text.TextUtils;
import android.webkit.MimeTypeMap;

import androidx.annotation.Nullable;
import androidx.core.content.ContextCompat;

import com.sealdice.dice.BuildConfig;
import com.sealdice.dice.R;

import java.io.File;
import java.io.FileNotFoundException;

public class LocalStorageProvider extends DocumentsProvider {
    /**
     * 默认root需要查询的项
     */
    private final static String[] DEFAULT_ROOT_PROJECTION = new String[]{Root.COLUMN_ROOT_ID, Root.COLUMN_SUMMARY,
            Root.COLUMN_FLAGS, Root.COLUMN_TITLE, Root.COLUMN_DOCUMENT_ID, Root.COLUMN_ICON,
            Root.COLUMN_AVAILABLE_BYTES};
    /**
     * 默认Document需要查询的项
     */
    private final static String[] DEFAULT_DOCUMENT_PROJECTION = new String[]{Document.COLUMN_DOCUMENT_ID,
            Document.COLUMN_DISPLAY_NAME, Document.COLUMN_FLAGS, Document.COLUMN_MIME_TYPE, Document.COLUMN_SIZE,
            Document.COLUMN_LAST_MODIFIED};

    /**
     * 进行读写权限检查
     */
    static boolean isMissingPermission(@Nullable Context context) {
        if (context == null) {
            return true;
        }
        if (ContextCompat.checkSelfPermission(context,
                Manifest.permission.WRITE_EXTERNAL_STORAGE) != PackageManager.PERMISSION_GRANTED) {
            // 通知root的Uri失去权限, 禁止相关操作
            context.getContentResolver().notifyChange(
                    DocumentsContract.buildRootsUri(BuildConfig.DOCUMENTS_AUTHORITY), null);
            return true;
        }
        return false;
    }

    /*
     * 在此方法中组装一个cursor, 他的内容就是home与sd卡的路径信息,
     * 并将home与sd卡的信息存到数据库中
     */
    @Override
    public Cursor queryRoots(final String[] projection) throws FileNotFoundException {
        if (getContext() == null || ContextCompat.checkSelfPermission(getContext(),
                Manifest.permission.WRITE_EXTERNAL_STORAGE) != PackageManager.PERMISSION_GRANTED) {
            return null;
        }
        //创建一个查询cursor, 来设置需要查询的项, 如果"projection"为空, 那么使用默认项
        final MatrixCursor result = new MatrixCursor(projection != null ? projection : DEFAULT_ROOT_PROJECTION);
        // 添加home路径
        File homeDir = new File(getContext().getFilesDir().getAbsolutePath());
        if (TextUtils.equals(Environment.getExternalStorageState(), Environment.MEDIA_MOUNTED)) {
            final MatrixCursor.RowBuilder row = result.newRow();
            row.add(Root.COLUMN_ROOT_ID, homeDir.getAbsolutePath());
            row.add(Root.COLUMN_DOCUMENT_ID, homeDir.getAbsolutePath());
            row.add(Root.COLUMN_TITLE, getContext().getString(R.string.home));
            row.add(Root.COLUMN_FLAGS, Root.FLAG_LOCAL_ONLY | Root.FLAG_SUPPORTS_CREATE | Root.FLAG_SUPPORTS_IS_CHILD);
            row.add(Root.COLUMN_ICON, R.mipmap.ic_launcher);
        }
        // 添加SD卡路径
//        File sdCard = new File("/storage/extSdCard");
//        String storageState = EnvironmentCompat.getStorageState(sdCard);
//        if (TextUtils.equals(storageState, Environment.MEDIA_MOUNTED) ||
//                TextUtils.equals(storageState, Environment.MEDIA_MOUNTED_READ_ONLY)) {
//            final MatrixCursor.RowBuilder row = result.newRow();
//            row.add(Root.COLUMN_ROOT_ID, sdCard.getAbsolutePath());
//            row.add(Root.COLUMN_DOCUMENT_ID, sdCard.getAbsolutePath());
//            row.add(Root.COLUMN_TITLE, getContext().getString(R.string.sd_card));
//            row.add(Root.COLUMN_FLAGS, Root.FLAG_LOCAL_ONLY);
//            row.add(Root.COLUMN_ICON, R.drawable.ic_sd_card);
//            row.add(Root.COLUMN_SUMMARY, sdCard.getAbsolutePath());
//            row.add(Root.COLUMN_AVAILABLE_BYTES, new StatFs(sdCard.getAbsolutePath()).getAvailableBytes());
//        }
        return result;
    }

    @Override
    public boolean isChildDocument(final String parentDocumentId, final String documentId) {
        return documentId.startsWith(parentDocumentId);
    }

    @Override
    public Cursor queryChildDocuments(final String parentDocumentId, final String[] projection,
                                      final String sortOrder) throws FileNotFoundException {
        // 判断是否缺少权限
        if (LocalStorageProvider.isMissingPermission(getContext())) {
            return null;
        }
        // 创建一个查询cursor, 来设置需要查询的项, 如果"projection"为空, 那么使用默认项
        final MatrixCursor result = new MatrixCursor(projection != null ? projection : DEFAULT_DOCUMENT_PROJECTION);
        final File parent = new File(parentDocumentId);
        for (File file : parent.listFiles()) {
            // 不显示隐藏的文件或文件夹
            if (!file.getName().startsWith(".")) {
                // 添加文件的名字, 类型, 大小等属性
                includeFile(result, file);
            }
        }
        return result;
    }

    @Override
    public Cursor queryDocument(final String documentId, final String[] projection) throws FileNotFoundException {
        if (LocalStorageProvider.isMissingPermission(getContext())) {
            return null;
        }
        // 创建一个查询cursor, 来设置需要查询的项, 如果"projection"为空, 那么使用默认项
        final MatrixCursor result = new MatrixCursor(projection != null ? projection : DEFAULT_DOCUMENT_PROJECTION);
        includeFile(result, new File(documentId));
        return result;
    }

    private void includeFile(final MatrixCursor result, final File file) throws FileNotFoundException {
        final MatrixCursor.RowBuilder row = result.newRow();
        row.add(Document.COLUMN_DOCUMENT_ID, file.getAbsolutePath());
        row.add(Document.COLUMN_DISPLAY_NAME, file.getName());
        String mimeType = getDocumentType(file.getAbsolutePath());
        row.add(Document.COLUMN_MIME_TYPE, mimeType);
        int flags = file.canWrite()
                ? Document.FLAG_SUPPORTS_DELETE | Document.FLAG_SUPPORTS_WRITE | Document.FLAG_SUPPORTS_RENAME
                | (mimeType.equals(Document.MIME_TYPE_DIR) ? Document.FLAG_DIR_SUPPORTS_CREATE : 0) : 0;
        if (mimeType.startsWith("image/"))
            flags |= Document.FLAG_SUPPORTS_THUMBNAIL;
        row.add(Document.COLUMN_FLAGS, flags);
        row.add(Document.COLUMN_SIZE, file.length());
        row.add(Document.COLUMN_LAST_MODIFIED, file.lastModified());
    }

    @Override
    public String getDocumentType(final String documentId) throws FileNotFoundException {
        if (LocalStorageProvider.isMissingPermission(getContext())) {
            return null;
        }
        File file = new File(documentId);
        if (file.isDirectory())
            return Document.MIME_TYPE_DIR;
        final int lastDot = file.getName().lastIndexOf('.');
        if (lastDot >= 0) {
            final String extension = file.getName().substring(lastDot + 1);
            final String mime = MimeTypeMap.getSingleton().getMimeTypeFromExtension(extension);
            if (mime != null) {
                return mime;
            }
        }
        return "application/octet-stream";
    }

    @Override
    public ParcelFileDescriptor openDocument(String documentId, String mode, @Nullable CancellationSignal signal) throws FileNotFoundException {
        if ("data_file".equals(documentId)) {
            // Open the data file and return a ParcelFileDescriptor
            File file = new File(getContext().getFilesDir().getAbsolutePath());
            int fileMode = modeToMode(mode);
            return ParcelFileDescriptor.open(file, fileMode);
        } else {
            throw new FileNotFoundException("Invalid document ID");
        }
    }
    private int modeToMode(String mode) {
        if ("r".equals(mode)) {
            return ParcelFileDescriptor.MODE_READ_ONLY;
        } else if ("w".equals(mode) || "wt".equals(mode)) {
            return ParcelFileDescriptor.MODE_WRITE_ONLY
                    | ParcelFileDescriptor.MODE_CREATE
                    | ParcelFileDescriptor.MODE_TRUNCATE;
        } else if ("wa".equals(mode)) {
            return ParcelFileDescriptor.MODE_WRITE_ONLY
                    | ParcelFileDescriptor.MODE_CREATE
                    | ParcelFileDescriptor.MODE_APPEND;
        } else if ("rw".equals(mode)) {
            return ParcelFileDescriptor.MODE_READ_WRITE
                    | ParcelFileDescriptor.MODE_CREATE
                    | ParcelFileDescriptor.MODE_TRUNCATE;
        } else if ("rwt".equals(mode)) {
            return ParcelFileDescriptor.MODE_READ_WRITE
                    | ParcelFileDescriptor.MODE_CREATE
                    | ParcelFileDescriptor.MODE_TRUNCATE;
        } else {
            throw new IllegalArgumentException("Invalid mode: " + mode);
        }
    }
    @Override
    public boolean onCreate() {
        return true;  // 这里需要返回true
    }
}