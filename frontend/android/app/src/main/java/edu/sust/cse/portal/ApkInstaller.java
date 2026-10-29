package edu.sust.cse.portal;

import android.content.Intent;
import android.net.Uri;
import android.os.Build;
import android.provider.Settings;
import android.text.TextUtils;

import androidx.core.content.FileProvider;

import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

import java.io.File;

@CapacitorPlugin(name = "ApkInstaller")
public class ApkInstaller extends Plugin {

    private static final String PACKAGE_MIME = "application/vnd.android.package-archive";

    private static final String PROVIDER_SUFFIX = ".fileprovider";

    private static final String FILE_PREFIX = "file://";

    @PluginMethod
    public void canInstall(PluginCall call) {
        JSObject result = new JSObject();
        result.put("allowed", allowed());
        call.resolve(result);
    }

    @PluginMethod
    public void openSettings(PluginCall call) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) {
            call.resolve();
            return;
        }
        Intent intent = new Intent(
                Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES,
                Uri.parse("package:" + getContext().getPackageName()));
        intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
        try {
            getContext().startActivity(intent);
        } catch (Exception failure) {
            call.reject("This phone has no screen for allowing installs from the app");
            return;
        }
        call.resolve();
    }

    @PluginMethod
    public void install(PluginCall call) {
        String path = call.getString("path", "");
        if (TextUtils.isEmpty(path)) {
            call.reject("The update needs the path of the downloaded file");
            return;
        }
        File apk = fileAt(path);
        if (!apk.exists()) {
            call.reject("The downloaded update is no longer on this phone");
            return;
        }
        Uri target;
        try {
            target = FileProvider.getUriForFile(
                    getContext(), getContext().getPackageName() + PROVIDER_SUFFIX, apk);
        } catch (IllegalArgumentException failure) {
            call.reject("Android refused to share the downloaded update");
            return;
        }
        Intent intent = new Intent(Intent.ACTION_VIEW);
        intent.setDataAndType(target, PACKAGE_MIME);
        intent.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION | Intent.FLAG_ACTIVITY_NEW_TASK);
        try {
            getContext().startActivity(intent);
        } catch (Exception failure) {
            call.reject("Android could not open the installer");
            return;
        }
        call.resolve();
    }

    private boolean allowed() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) {
            return true;
        }
        return getContext().getPackageManager().canRequestPackageInstalls();
    }

    private static File fileAt(String path) {
        if (!path.startsWith(FILE_PREFIX)) {
            return new File(path);
        }
        String plain = Uri.parse(path).getPath();
        return new File(TextUtils.isEmpty(plain) ? path : plain);
    }
}
