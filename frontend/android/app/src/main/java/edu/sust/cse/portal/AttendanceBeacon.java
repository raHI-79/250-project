package edu.sust.cse.portal;

import android.Manifest;
import android.bluetooth.BluetoothAdapter;
import android.bluetooth.BluetoothManager;
import android.content.Context;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.os.Build;
import android.text.TextUtils;

import androidx.core.content.ContextCompat;

import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;
import com.getcapacitor.annotation.Permission;
import com.getcapacitor.annotation.PermissionCallback;

import java.util.ArrayList;
import java.util.List;

@CapacitorPlugin(
        name = "AttendanceBeacon",
        permissions = {
                @Permission(alias = AttendanceBeacon.NEARBY, strings = {
                        Manifest.permission.BLUETOOTH_ADVERTISE,
                        Manifest.permission.BLUETOOTH_SCAN,
                        Manifest.permission.BLUETOOTH_CONNECT
                }),
                @Permission(alias = AttendanceBeacon.PLACE, strings = {
                        Manifest.permission.ACCESS_FINE_LOCATION
                })
        }
)
public class AttendanceBeacon extends Plugin {

    public static final String NEARBY = "nearby";

    public static final String PLACE = "place";

    private static final String ASKED = "asked";

    @PluginMethod
    public void status(PluginCall call) {
        call.resolve(snapshot());
    }

    @PluginMethod
    public void ensure(PluginCall call) {
        List<String> missing = new ArrayList<>();
        if (!granted(Manifest.permission.ACCESS_FINE_LOCATION)) {
            missing.add(PLACE);
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S
                && !granted(Manifest.permission.BLUETOOTH_ADVERTISE)) {
            missing.add(NEARBY);
        }
        if (missing.isEmpty()) {
            call.resolve(snapshot());
            return;
        }
        requestPermissionForAliases(missing.toArray(new String[0]), call, ASKED);
    }

    @PermissionCallback
    private void asked(PluginCall call) {
        call.resolve(snapshot());
    }

    @PluginMethod
    public void enableBluetooth(PluginCall call) {
        BluetoothAdapter adapter = adapter();
        if (adapter == null) {
            call.reject("This phone has no Bluetooth");
            return;
        }
        if (adapter.isEnabled()) {
            call.resolve(snapshot());
            return;
        }
        Intent intent = new Intent(BluetoothAdapter.ACTION_REQUEST_ENABLE);
        intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
        try {
            getContext().startActivity(intent);
        } catch (Exception failure) {
            call.reject("Turn Bluetooth on from the settings and try again");
            return;
        }
        call.resolve(snapshot());
    }

    @PluginMethod
    public void start(PluginCall call) {
        String tokenKey = call.getString("token_key", "");
        String endpoint = call.getString("endpoint", "");
        String bearer = call.getString("bearer", "");
        long sessionId = call.getLong("session_id", 0L);
        if (sessionId <= 0 || TextUtils.isEmpty(tokenKey)) {
            call.reject("The session is missing its key");
            return;
        }
        Intent intent = new Intent(getContext(), AttendanceService.class);
        intent.setAction(AttendanceService.ACTION_START);
        intent.putExtra(AttendanceService.EXTRA_SESSION, sessionId);
        intent.putExtra(AttendanceService.EXTRA_KEY, tokenKey);
        intent.putExtra(AttendanceService.EXTRA_COURSE, call.getString("course", ""));
        intent.putExtra(AttendanceService.EXTRA_PRESENT, call.getInt("present", 0));
        intent.putExtra(AttendanceService.EXTRA_ROSTER, call.getInt("roster", 0));
        intent.putExtra(AttendanceService.EXTRA_ENDPOINT, endpoint);
        intent.putExtra(AttendanceService.EXTRA_BEARER, bearer);
        try {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                getContext().startForegroundService(intent);
            } else {
                getContext().startService(intent);
            }
        } catch (Exception failure) {
            call.reject("The attendance service could not start");
            return;
        }
        call.resolve(snapshot());
    }

    @PluginMethod
    public void update(PluginCall call) {
        Intent intent = new Intent(getContext(), AttendanceService.class);
        intent.setAction(AttendanceService.ACTION_UPDATE);
        intent.putExtra(AttendanceService.EXTRA_PRESENT, call.getInt("present", 0));
        intent.putExtra(AttendanceService.EXTRA_ROSTER, call.getInt("roster", 0));
        try {
            getContext().startService(intent);
        } catch (Exception failure) {
            call.resolve();
            return;
        }
        call.resolve();
    }

    @PluginMethod
    public void stop(PluginCall call) {
        Intent intent = new Intent(getContext(), AttendanceService.class);
        intent.setAction(AttendanceService.ACTION_STOP);
        try {
            getContext().startService(intent);
        } catch (Exception failure) {
            call.resolve();
            return;
        }
        call.resolve();
    }

    private JSObject snapshot() {
        BluetoothAdapter adapter = adapter();
        boolean advertiser = adapter != null
                && getContext().getPackageManager()
                .hasSystemFeature(PackageManager.FEATURE_BLUETOOTH_LE);
        JSObject result = new JSObject();
        result.put("supported", advertiser);
        result.put("bluetooth_on", adapter != null && adapter.isEnabled());
        result.put("nearby", Build.VERSION.SDK_INT < Build.VERSION_CODES.S
                || granted(Manifest.permission.BLUETOOTH_ADVERTISE));
        result.put("place", granted(Manifest.permission.ACCESS_FINE_LOCATION));
        return result;
    }

    private boolean granted(String permission) {
        return ContextCompat.checkSelfPermission(getContext(), permission)
                == PackageManager.PERMISSION_GRANTED;
    }

    private BluetoothAdapter adapter() {
        BluetoothManager manager = (BluetoothManager) getContext()
                .getSystemService(Context.BLUETOOTH_SERVICE);
        return manager == null ? null : manager.getAdapter();
    }
}
