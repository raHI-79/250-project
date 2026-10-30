package edu.sust.cse.portal;

import android.Manifest;
import android.bluetooth.BluetoothAdapter;
import android.bluetooth.BluetoothManager;
import android.bluetooth.le.BluetoothLeScanner;
import android.bluetooth.le.ScanCallback;
import android.bluetooth.le.ScanFilter;
import android.bluetooth.le.ScanResult;
import android.bluetooth.le.ScanSettings;
import android.content.Context;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.location.Location;
import android.location.LocationListener;
import android.location.LocationManager;
import android.os.Build;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.os.ParcelUuid;
import android.provider.Settings;
import android.text.TextUtils;
import android.util.Log;

import androidx.biometric.BiometricManager;
import androidx.biometric.BiometricPrompt;
import androidx.core.content.ContextCompat;
import androidx.fragment.app.FragmentActivity;

import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;
import com.getcapacitor.annotation.Permission;
import com.getcapacitor.annotation.PermissionCallback;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.concurrent.Executor;
import java.util.concurrent.atomic.AtomicBoolean;

@CapacitorPlugin(
        name = "AttendanceMark",
        permissions = {
                @Permission(alias = AttendanceMark.NEARBY, strings = {
                        Manifest.permission.BLUETOOTH_SCAN,
                        Manifest.permission.BLUETOOTH_CONNECT
                }),
                @Permission(alias = AttendanceMark.PLACE, strings = {
                        Manifest.permission.ACCESS_FINE_LOCATION
                })
        }
)
public class AttendanceMark extends Plugin {

    public static final String NEARBY = "nearby";

    public static final String PLACE = "place";

    private static final String TAG = "AttendanceMark";

    private static final String ASKED = "asked";

    private static final long SCAN_MILLIS = 10000L;

    private static final long POSITION_MILLIS = 12000L;

    private static final long FRESH_POSITION_MILLIS = 10000L;

    private final Handler handler = new Handler(Looper.getMainLooper());

    private final AtomicBoolean scanning = new AtomicBoolean(false);

    private volatile String lastToken = "";

    private volatile long lastTokenAt;

    private volatile String lastError = "";

    @PluginMethod
    public void deviceId(PluginCall call) {
        String id = Settings.Secure.getString(getContext().getContentResolver(),
                Settings.Secure.ANDROID_ID);
        JSObject result = new JSObject();
        result.put("device_id", id == null ? "" : id);
        call.resolve(result);
    }

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
                && !granted(Manifest.permission.BLUETOOTH_SCAN)) {
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
    public void scan(final PluginCall call) {
        BluetoothAdapter adapter = adapter();
        if (adapter == null || !adapter.isEnabled()) {
            call.resolve(refused("Bluetooth is off"));
            return;
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S && !granted(Manifest.permission.BLUETOOTH_SCAN)) {
            call.resolve(refused("Nearby devices is not allowed"));
            return;
        }
        if (!granted(Manifest.permission.ACCESS_FINE_LOCATION)) {
            call.resolve(refused("Precise location is not allowed"));
            return;
        }
        final BluetoothLeScanner scanner = adapter.getBluetoothLeScanner();
        if (scanner == null) {
            call.resolve(refused("This phone cannot scan for Bluetooth"));
            return;
        }
        final ParcelUuid uuid = ParcelUuid.fromString(AttendanceTokens.SERVICE_UUID);
        final AtomicBoolean done = new AtomicBoolean(false);
        final ScanCallback[] holder = new ScanCallback[1];
        final ScanCallback callback = new ScanCallback() {
            @Override
            public void onScanResult(int callbackType, ScanResult result) {
                byte[] data = result.getScanRecord() == null
                        ? null
                        : result.getScanRecord().getServiceData(uuid);
                if (data == null || data.length < AttendanceTokens.TOKEN_BYTES) {
                    return;
                }
                if (!done.compareAndSet(false, true)) {
                    return;
                }
                stopScan(scanner, holder[0]);
                byte[] token = new byte[AttendanceTokens.TOKEN_BYTES];
                System.arraycopy(data, 0, token, 0, AttendanceTokens.TOKEN_BYTES);
                call.resolve(heard(AttendanceTokens.hex(token)));
            }

            @Override
            public void onScanFailed(int errorCode) {
                Log.w(TAG, "the scan did not start, code " + errorCode);
                if (done.compareAndSet(false, true)) {
                    call.resolve(refused("Bluetooth refused the scan, code " + errorCode));
                }
            }
        };
        holder[0] = callback;
        ScanSettings settings = new ScanSettings.Builder()
                .setScanMode(ScanSettings.SCAN_MODE_LOW_LATENCY)
                .build();
        List<ScanFilter> filters = Collections.singletonList(
                new ScanFilter.Builder().setServiceUuid(uuid).build());
        lastError = "";
        scanning.set(true);
        try {
            scanner.startScan(filters, settings, callback);
        } catch (Exception failure) {
            Log.w(TAG, "the scan could not start", failure);
            call.resolve(refused("Bluetooth could not start the scan"));
            return;
        }
        long seconds = call.getLong("seconds", 0L);
        long window = seconds > 0 ? seconds * 1000L : SCAN_MILLIS;
        handler.postDelayed(new Runnable() {
            @Override
            public void run() {
                if (done.compareAndSet(false, true)) {
                    stopScan(scanner, callback);
                    call.resolve(heard(""));
                }
            }
        }, window);
    }

    private int authenticators() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            return BiometricManager.Authenticators.BIOMETRIC_WEAK
                    | BiometricManager.Authenticators.DEVICE_CREDENTIAL;
        }
        return BiometricManager.Authenticators.BIOMETRIC_WEAK;
    }

    @PluginMethod
    public void position(final PluginCall call) {
        if (!granted(Manifest.permission.ACCESS_FINE_LOCATION)) {
            call.resolve(placed(null));
            return;
        }
        final LocationManager locations = (LocationManager) getContext()
                .getSystemService(Context.LOCATION_SERVICE);
        if (locations == null) {
            call.resolve(placed(null));
            return;
        }
        Location last = null;
        try {
            last = locations.getLastKnownLocation(LocationManager.GPS_PROVIDER);
        } catch (Exception failure) {
            Log.w(TAG, "the last position could not be read", failure);
        }
        if (last != null && System.currentTimeMillis() - last.getTime() < FRESH_POSITION_MILLIS) {
            call.resolve(placed(last));
            return;
        }
        final AtomicBoolean done = new AtomicBoolean(false);
        final LocationListener[] holder = new LocationListener[1];
        final LocationListener listener = new LocationListener() {
            @Override
            public void onLocationChanged(Location location) {
                if (done.compareAndSet(false, true)) {
                    stopUpdates(locations, holder[0]);
                    call.resolve(placed(location));
                }
            }

            @Override
            public void onStatusChanged(String provider, int status, Bundle extras) {
            }

            @Override
            public void onProviderEnabled(String provider) {
            }

            @Override
            public void onProviderDisabled(String provider) {
            }
        };
        holder[0] = listener;
        try {
            locations.requestLocationUpdates(LocationManager.GPS_PROVIDER, 0L, 0f, listener,
                    Looper.getMainLooper());
        } catch (Exception failure) {
            call.resolve(placed(last));
            return;
        }
        final Location fallback = last;
        handler.postDelayed(new Runnable() {
            @Override
            public void run() {
                if (done.compareAndSet(false, true)) {
                    stopUpdates(locations, listener);
                    call.resolve(placed(fallback));
                }
            }
        }, POSITION_MILLIS);
    }

    @PluginMethod
    public void confirm(final PluginCall call) {
        final FragmentActivity activity = getActivity();
        if (activity == null) {
            call.reject("This screen cannot ask for your fingerprint");
            return;
        }
        BiometricManager manager = BiometricManager.from(getContext());
        final int allowed = authenticators();
        if (manager.canAuthenticate(allowed) != BiometricManager.BIOMETRIC_SUCCESS) {
            JSObject result = new JSObject();
            result.put("confirmed", false);
            result.put("available", false);
            call.resolve(result);
            return;
        }
        final Executor executor = ContextCompat.getMainExecutor(getContext());
        activity.runOnUiThread(new Runnable() {
            @Override
            public void run() {
                BiometricPrompt prompt = new BiometricPrompt(activity, executor,
                        new BiometricPrompt.AuthenticationCallback() {
                            @Override
                            public void onAuthenticationSucceeded(BiometricPrompt.AuthenticationResult result) {
                                JSObject answer = new JSObject();
                                answer.put("confirmed", true);
                                answer.put("available", true);
                                call.resolve(answer);
                            }

                            @Override
                            public void onAuthenticationError(int code, CharSequence message) {
                                JSObject answer = new JSObject();
                                answer.put("confirmed", false);
                                answer.put("available", true);
                                answer.put("message", message == null ? "" : message.toString());
                                call.resolve(answer);
                            }
                        });
                BiometricPrompt.PromptInfo.Builder builder = new BiometricPrompt.PromptInfo.Builder()
                        .setTitle(getContext().getString(R.string.attendance_confirm_title))
                        .setSubtitle(getContext().getString(R.string.attendance_confirm_subtitle))
                        .setAllowedAuthenticators(allowed);
                if ((allowed & BiometricManager.Authenticators.DEVICE_CREDENTIAL) == 0) {
                    builder.setNegativeButtonText(
                            getContext().getString(R.string.attendance_confirm_cancel));
                }
                BiometricPrompt.PromptInfo info = builder.build();
                try {
                    prompt.authenticate(info);
                } catch (Exception failure) {
                    JSObject answer = new JSObject();
                    answer.put("confirmed", false);
                    answer.put("available", false);
                    call.resolve(answer);
                }
            }
        });
    }

    private void stopScan(BluetoothLeScanner scanner, ScanCallback callback) {
        try {
            scanner.stopScan(callback);
        } catch (Exception failure) {
            Log.w(TAG, "the scan could not be stopped", failure);
        }
    }

    private void stopUpdates(LocationManager locations, LocationListener listener) {
        try {
            locations.removeUpdates(listener);
        } catch (Exception failure) {
            Log.w(TAG, "the position updates could not stop", failure);
        }
    }

    private JSObject snapshot() {
        BluetoothAdapter adapter = adapter();
        boolean feature = getContext().getPackageManager()
                .hasSystemFeature(PackageManager.FEATURE_BLUETOOTH_LE);
        JSObject result = new JSObject();
        result.put("supported", adapter != null && feature);
        result.put("bluetooth_on", adapter != null && adapter.isEnabled());
        result.put("nearby", Build.VERSION.SDK_INT < Build.VERSION_CODES.S
                || granted(Manifest.permission.BLUETOOTH_SCAN));
        result.put("place", granted(Manifest.permission.ACCESS_FINE_LOCATION));
        result.put("scanning", scanning.get());
        result.put("token", lastToken);
        result.put("token_age_ms", lastTokenAt == 0L
                ? 0L
                : System.currentTimeMillis() - lastTokenAt);
        result.put("error", lastError);
        return result;
    }

    private JSObject heard(String token) {
        scanning.set(false);
        JSObject result = new JSObject();
        if (TextUtils.isEmpty(token)) {
            result.put("token", "");
            result.put("age_ms", 0L);
            result.put("error", lastError);
            return result;
        }
        lastToken = token;
        lastTokenAt = System.currentTimeMillis();
        lastError = "";
        result.put("token", token);
        result.put("age_ms", 0L);
        result.put("error", "");
        return result;
    }

    private JSObject refused(String reason) {
        lastError = reason == null ? "" : reason;
        return heard("");
    }

    private JSObject placed(Location location) {
        JSObject result = new JSObject();
        if (location == null) {
            result.put("found", false);
            result.put("lat", 0);
            result.put("lng", 0);
            result.put("accuracy_m", 0);
            result.put("is_mock", false);
            return result;
        }
        result.put("found", true);
        result.put("lat", location.getLatitude());
        result.put("lng", location.getLongitude());
        result.put("accuracy_m", location.getAccuracy());
        result.put("is_mock", mocked(location));
        return result;
    }

    private boolean mocked(Location location) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
            return location.isMock();
        }
        return location.isFromMockProvider();
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
