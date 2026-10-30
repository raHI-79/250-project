package edu.sust.cse.portal;

import android.Manifest;
import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.app.Service;
import android.bluetooth.BluetoothAdapter;
import android.bluetooth.BluetoothManager;
import android.bluetooth.le.AdvertiseCallback;
import android.bluetooth.le.AdvertiseData;
import android.bluetooth.le.AdvertiseSettings;
import android.bluetooth.le.BluetoothLeAdvertiser;
import android.content.Context;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.content.pm.ServiceInfo;
import android.location.Location;
import android.location.LocationListener;
import android.location.LocationManager;
import android.os.Build;
import android.os.Bundle;
import android.os.Handler;
import android.os.IBinder;
import android.os.Looper;
import android.os.ParcelUuid;
import android.text.TextUtils;
import android.util.Log;

import androidx.annotation.RequiresApi;
import androidx.core.app.NotificationCompat;
import androidx.core.content.ContextCompat;

import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.util.Locale;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

public class AttendanceService extends Service {

    public static final String ACTION_START = "edu.sust.cse.portal.ATTENDANCE_START";

    public static final String ACTION_UPDATE = "edu.sust.cse.portal.ATTENDANCE_UPDATE";

    public static final String ACTION_STOP = "edu.sust.cse.portal.ATTENDANCE_STOP";

    public static final String EXTRA_SESSION = "session_id";

    public static final String EXTRA_KEY = "token_key";

    public static final String EXTRA_COURSE = "course";

    public static final String EXTRA_PRESENT = "present";

    public static final String EXTRA_ROSTER = "roster";

    public static final String EXTRA_ENDPOINT = "endpoint";

    public static final String EXTRA_BEARER = "bearer";

    private static final String TAG = "AttendanceService";

    private static final String CHANNEL_ID = "sust_cse_attendance";

    private static final int NOTIFICATION_ID = 4221;

    private static final long LOCATION_INTERVAL = 15000L;

    private static final String[] PROVIDERS = {
            LocationManager.GPS_PROVIDER,
            LocationManager.NETWORK_PROVIDER
    };

    private static final long ROTATE_MILLIS = AttendanceTokens.TOKEN_SLICE_SECONDS * 1000L;

    private static final int CONNECT_TIMEOUT = 8000;

    private final Handler handler = new Handler(Looper.getMainLooper());

    private final ExecutorService uploads = Executors.newSingleThreadExecutor();

    private static volatile boolean advertisingNow;

    private static volatile String lastFailure = "";

    public static boolean broadcasting() {
        return advertisingNow;
    }

    public static String failure() {
        return lastFailure;
    }

    private static String advertiseReason(int errorCode) {
        switch (errorCode) {
            case AdvertiseCallback.ADVERTISE_FAILED_DATA_TOO_LARGE:
                return "The Bluetooth packet is too large for this phone";
            case AdvertiseCallback.ADVERTISE_FAILED_TOO_MANY_ADVERTISERS:
                return "This phone is already broadcasting too much over Bluetooth";
            case AdvertiseCallback.ADVERTISE_FAILED_INTERNAL_ERROR:
                return "Bluetooth hit an internal error, turn it off and on again";
            case AdvertiseCallback.ADVERTISE_FAILED_FEATURE_UNSUPPORTED:
                return "This phone cannot broadcast over Bluetooth, use the code on the board";
            default:
                return "Bluetooth refused to broadcast, code " + errorCode;
        }
    }

    private final AdvertiseCallback callback = new AdvertiseCallback() {
        @Override
        public void onStartSuccess(AdvertiseSettings settings) {
            advertisingNow = true;
            lastFailure = "";
        }

        @Override
        public void onStartFailure(int errorCode) {
            Log.w(TAG, "the advertisement did not start, code " + errorCode);
            if (errorCode == AdvertiseCallback.ADVERTISE_FAILED_ALREADY_STARTED) {
                advertisingNow = true;
                return;
            }
            advertisingNow = false;
            lastFailure = advertiseReason(errorCode);
        }
    };

    private final LocationListener listener = new LocationListener() {
        @Override
        public void onLocationChanged(Location location) {
            send(location);
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

    private final Runnable rotate = new Runnable() {
        @Override
        public void run() {
            advertise();
            handler.postDelayed(this, ROTATE_MILLIS);
        }
    };

    private BluetoothLeAdvertiser advertiser;

    private LocationManager locations;

    private long sessionId;

    private String tokenKey = "";

    private String course = "";

    private String endpoint = "";

    private String bearer = "";

    private int present;

    private int roster;

    private boolean advertising;

    private boolean started;

    @Override
    public IBinder onBind(Intent intent) {
        return null;
    }

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        String action = intent == null ? ACTION_STOP : intent.getAction();
        if (ACTION_STOP.equals(action)) {
            stopEverything();
            return START_NOT_STICKY;
        }
        if (ACTION_UPDATE.equals(action)) {
            if (!started) {
                stopSelf();
                return START_NOT_STICKY;
            }
            present = intent.getIntExtra(EXTRA_PRESENT, present);
            roster = intent.getIntExtra(EXTRA_ROSTER, roster);
            notifyOngoing();
            return START_STICKY;
        }
        sessionId = intent.getLongExtra(EXTRA_SESSION, 0L);
        tokenKey = text(intent.getStringExtra(EXTRA_KEY));
        course = text(intent.getStringExtra(EXTRA_COURSE));
        endpoint = text(intent.getStringExtra(EXTRA_ENDPOINT));
        bearer = text(intent.getStringExtra(EXTRA_BEARER));
        present = intent.getIntExtra(EXTRA_PRESENT, 0);
        roster = intent.getIntExtra(EXTRA_ROSTER, 0);
        if (sessionId <= 0 || TextUtils.isEmpty(tokenKey)) {
            stopEverything();
            return START_NOT_STICKY;
        }
        if (!startForegroundNotice()) {
            stopEverything();
            return START_NOT_STICKY;
        }
        startAdvertising();
        startLocations();
        return START_STICKY;
    }

    @Override
    public void onDestroy() {
        stopAdvertising();
        stopLocations();
        handler.removeCallbacks(rotate);
        uploads.shutdown();
        super.onDestroy();
    }

    private void stopEverything() {
        stopAdvertising();
        stopLocations();
        handler.removeCallbacks(rotate);
        started = false;
        stopForeground(true);
        stopSelf();
    }

    @RequiresApi(Build.VERSION_CODES.Q)
    private int serviceTypes() {
        int types = 0;
        if (allowed(Manifest.permission.ACCESS_FINE_LOCATION)
                || allowed(Manifest.permission.ACCESS_COARSE_LOCATION)) {
            types |= ServiceInfo.FOREGROUND_SERVICE_TYPE_LOCATION;
        }
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.S
                || allowed(Manifest.permission.BLUETOOTH_ADVERTISE)) {
            types |= ServiceInfo.FOREGROUND_SERVICE_TYPE_CONNECTED_DEVICE;
        }
        return types;
    }

    private boolean startForegroundNotice() {
        createChannel();
        try {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
                int types = serviceTypes();
                if (types == 0) {
                    Log.w(TAG, "no permission left for the attendance service to run in the foreground");
                    return false;
                }
                startForeground(NOTIFICATION_ID, notification(), types);
            } else {
                startForeground(NOTIFICATION_ID, notification());
            }
        } catch (Exception failure) {
            Log.w(TAG, "the attendance service could not go to the foreground", failure);
            return false;
        }
        started = true;
        return true;
    }

    private void notifyOngoing() {
        NotificationManager manager = (NotificationManager) getSystemService(Context.NOTIFICATION_SERVICE);
        if (manager != null) {
            manager.notify(NOTIFICATION_ID, notification());
        }
    }

    private Notification notification() {
        Intent open = new Intent(this, MainActivity.class);
        open.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_CLEAR_TOP);
        int flags = PendingIntent.FLAG_UPDATE_CURRENT;
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
            flags |= PendingIntent.FLAG_IMMUTABLE;
        }
        PendingIntent pending = PendingIntent.getActivity(this, 0, open, flags);
        String title = getString(R.string.attendance_notification_title);
        String body = String.format(Locale.US, "%s · %d/%d", course, present, roster);
        return new NotificationCompat.Builder(this, CHANNEL_ID)
                .setContentTitle(title)
                .setContentText(body)
                .setSmallIcon(R.drawable.ic_stat_notify)
                .setColor(ContextCompat.getColor(this, R.color.notificationAccent))
                .setOngoing(true)
                .setOnlyAlertOnce(true)
                .setPriority(NotificationCompat.PRIORITY_LOW)
                .setContentIntent(pending)
                .build();
    }

    private void createChannel() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) {
            return;
        }
        NotificationManager manager = (NotificationManager) getSystemService(Context.NOTIFICATION_SERVICE);
        if (manager == null || manager.getNotificationChannel(CHANNEL_ID) != null) {
            return;
        }
        NotificationChannel channel = new NotificationChannel(CHANNEL_ID,
                getString(R.string.attendance_channel_name), NotificationManager.IMPORTANCE_LOW);
        channel.setShowBadge(false);
        manager.createNotificationChannel(channel);
    }

    private void startAdvertising() {
        BluetoothManager manager = (BluetoothManager) getSystemService(Context.BLUETOOTH_SERVICE);
        BluetoothAdapter adapter = manager == null ? null : manager.getAdapter();
        if (adapter == null) {
            lastFailure = "This phone has no Bluetooth, use the code on the board";
            return;
        }
        if (!adapter.isEnabled()) {
            lastFailure = "Bluetooth is off";
            return;
        }
        if (!getPackageManager().hasSystemFeature(PackageManager.FEATURE_BLUETOOTH_LE)) {
            lastFailure = "This phone has no Bluetooth Low Energy, use the code on the board";
            return;
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S && !allowed(Manifest.permission.BLUETOOTH_ADVERTISE)) {
            lastFailure = "Nearby devices is not allowed";
            return;
        }
        advertiser = adapter.getBluetoothLeAdvertiser();
        if (advertiser == null) {
            lastFailure = "This phone cannot broadcast over Bluetooth, use the code on the board";
            return;
        }
        lastFailure = "";
        handler.removeCallbacks(rotate);
        advertise();
        long delay = ROTATE_MILLIS - System.currentTimeMillis() % ROTATE_MILLIS;
        handler.postDelayed(rotate, delay);
    }

    private void advertise() {
        if (advertiser == null) {
            return;
        }
        long slice = AttendanceTokens.slice(System.currentTimeMillis() / 1000L,
                AttendanceTokens.TOKEN_SLICE_SECONDS);
        byte[] token = AttendanceTokens.token(tokenKey, sessionId, slice);
        ParcelUuid uuid = ParcelUuid.fromString(AttendanceTokens.SERVICE_UUID);
        AdvertiseSettings settings = new AdvertiseSettings.Builder()
                .setAdvertiseMode(AdvertiseSettings.ADVERTISE_MODE_LOW_POWER)
                .setTxPowerLevel(AdvertiseSettings.ADVERTISE_TX_POWER_LOW)
                .setConnectable(false)
                .build();
        AdvertiseData data = new AdvertiseData.Builder()
                .setIncludeDeviceName(false)
                .setIncludeTxPowerLevel(false)
                .addServiceUuid(uuid)
                .addServiceData(uuid, token)
                .build();
        try {
            if (advertising) {
                advertiser.stopAdvertising(callback);
            }
            advertiser.startAdvertising(settings, data, callback);
            advertising = true;
        } catch (Exception failure) {
            Log.w(TAG, "the advertisement could not be refreshed", failure);
            advertisingNow = false;
            lastFailure = "Bluetooth stopped broadcasting, turn it off and on again";
        }
    }

    private void stopAdvertising() {
        if (advertiser == null || !advertising) {
            return;
        }
        try {
            advertiser.stopAdvertising(callback);
        } catch (Exception failure) {
            Log.w(TAG, "the advertisement could not be stopped", failure);
        }
        advertising = false;
        advertisingNow = false;
    }

    private void startLocations() {
        if (!allowed(Manifest.permission.ACCESS_FINE_LOCATION)) {
            Log.w(TAG, "precise location is not allowed, this session has no position");
            return;
        }
        locations = (LocationManager) getSystemService(Context.LOCATION_SERVICE);
        if (locations == null) {
            return;
        }
        try {
            for (String provider : PROVIDERS) {
                if (locations.isProviderEnabled(provider)) {
                    locations.requestLocationUpdates(provider, LOCATION_INTERVAL, 0f,
                            listener, Looper.getMainLooper());
                }
            }
            send(freshest());
        } catch (Exception failure) {
            Log.w(TAG, "the position updates could not start", failure);
        }
    }

    private Location freshest() {
        Location best = null;
        for (String provider : PROVIDERS) {
            Location found = null;
            try {
                found = locations.getLastKnownLocation(provider);
            } catch (Exception failure) {
                Log.w(TAG, "a last position could not be read", failure);
            }
            if (found != null && (best == null || found.getTime() > best.getTime())) {
                best = found;
            }
        }
        return best;
    }

    private void stopLocations() {
        if (locations == null) {
            return;
        }
        try {
            locations.removeUpdates(listener);
        } catch (Exception failure) {
            Log.w(TAG, "the position updates could not stop", failure);
        }
        locations = null;
    }

    private void send(final Location location) {
        if (TextUtils.isEmpty(endpoint) || location == null || uploads.isShutdown()) {
            return;
        }
        final String body = String.format(Locale.US,
                "{\"lat\":%.7f,\"lng\":%.7f,\"accuracy_m\":%.1f}",
                location.getLatitude(), location.getLongitude(), location.getAccuracy());
        uploads.submit(new Runnable() {
            @Override
            public void run() {
                post(body);
            }
        });
    }

    private void post(String body) {
        HttpURLConnection connection = null;
        try {
            connection = (HttpURLConnection) new URL(endpoint).openConnection();
            connection.setRequestMethod("POST");
            connection.setConnectTimeout(CONNECT_TIMEOUT);
            connection.setReadTimeout(CONNECT_TIMEOUT);
            connection.setDoOutput(true);
            connection.setRequestProperty("Content-Type", "application/json");
            connection.setRequestProperty("X-Client", "android");
            if (!TextUtils.isEmpty(bearer)) {
                connection.setRequestProperty("Authorization", "Bearer " + bearer);
            }
            OutputStream stream = connection.getOutputStream();
            stream.write(body.getBytes("UTF-8"));
            stream.flush();
            stream.close();
            int status = connection.getResponseCode();
            if (status >= 400) {
                Log.w(TAG, "the server refused a position with status " + status);
            }
        } catch (Exception failure) {
            Log.w(TAG, "the position could not be sent", failure);
        } finally {
            if (connection != null) {
                connection.disconnect();
            }
        }
    }

    private boolean allowed(String permission) {
        return ContextCompat.checkSelfPermission(this, permission) == PackageManager.PERMISSION_GRANTED;
    }

    private static String text(String value) {
        return value == null ? "" : value;
    }
}
