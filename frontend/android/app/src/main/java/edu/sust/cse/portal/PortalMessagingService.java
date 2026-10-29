package edu.sust.cse.portal;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.Context;
import android.content.Intent;
import android.media.AudioAttributes;
import android.media.RingtoneManager;
import android.net.Uri;
import android.os.Build;
import android.text.TextUtils;

import androidx.core.app.NotificationCompat;
import androidx.core.content.ContextCompat;

import com.google.firebase.messaging.FirebaseMessagingService;
import com.google.firebase.messaging.RemoteMessage;

import java.util.Map;

public class PortalMessagingService extends FirebaseMessagingService {

    public static final String EXTRA_URL = "sustcse.url";

    private static final int BIG_TEXT_AT = 60;

    @Override
    public void onMessageReceived(RemoteMessage message) {
        Map<String, String> data = message.getData();
        String title = pick(data.get("title"), notificationTitle(message));
        String body = pick(data.get("body"), notificationBody(message));
        if (TextUtils.isEmpty(title) && TextUtils.isEmpty(body)) {
            return;
        }
        if (TextUtils.isEmpty(title)) {
            title = getString(R.string.app_name);
        }
        String tag = pick(data.get("tag"), "sustcse");
        String url = pick(data.get("url"), "");
        String channelId = pick(data.get("channel"), getString(R.string.notification_channel_id));
        NotificationManager manager =
                (NotificationManager) getSystemService(Context.NOTIFICATION_SERVICE);
        if (manager == null) {
            return;
        }
        ensureChannel(manager, channelId);
        manager.notify(tag, tag.hashCode(), build(channelId, title, body, url));
    }

    private Notification build(String channelId, String title, String body, String url) {
        Uri sound = RingtoneManager.getDefaultUri(RingtoneManager.TYPE_NOTIFICATION);
        NotificationCompat.Builder builder = new NotificationCompat.Builder(this, channelId)
                .setSmallIcon(R.drawable.ic_stat_notify)
                .setColor(ContextCompat.getColor(this, R.color.notificationAccent))
                .setColorized(false)
                .setContentTitle(title)
                .setContentText(body)
                .setTicker(title)
                .setAutoCancel(true)
                .setPriority(NotificationCompat.PRIORITY_HIGH)
                .setCategory(NotificationCompat.CATEGORY_MESSAGE)
                .setDefaults(Notification.DEFAULT_ALL)
                .setSound(sound)
                .setVibrate(new long[] {0, 250, 150, 250})
                .setContentIntent(opener(url));
        if (!TextUtils.isEmpty(body) && body.length() > BIG_TEXT_AT) {
            builder.setStyle(new NotificationCompat.BigTextStyle().bigText(body));
        }
        return builder.build();
    }

    private PendingIntent opener(String url) {
        Intent intent = new Intent(this, MainActivity.class);
        intent.setAction(Intent.ACTION_MAIN);
        intent.addCategory(Intent.CATEGORY_LAUNCHER);
        intent.setFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP | Intent.FLAG_ACTIVITY_CLEAR_TOP);
        intent.putExtra(EXTRA_URL, url);
        int flags = PendingIntent.FLAG_UPDATE_CURRENT;
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
            flags |= PendingIntent.FLAG_IMMUTABLE;
        }
        return PendingIntent.getActivity(this, url.hashCode(), intent, flags);
    }

    private void ensureChannel(NotificationManager manager, String channelId) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) {
            return;
        }
        NotificationChannel channel = new NotificationChannel(
                channelId,
                getString(R.string.notification_channel_name),
                NotificationManager.IMPORTANCE_HIGH);
        channel.enableVibration(true);
        channel.setVibrationPattern(new long[] {0, 250, 150, 250});
        channel.enableLights(true);
        channel.setLightColor(ContextCompat.getColor(this, R.color.notificationAccent));
        channel.setLockscreenVisibility(Notification.VISIBILITY_PUBLIC);
        channel.setSound(
                RingtoneManager.getDefaultUri(RingtoneManager.TYPE_NOTIFICATION),
                new AudioAttributes.Builder()
                        .setContentType(AudioAttributes.CONTENT_TYPE_SONIFICATION)
                        .setUsage(AudioAttributes.USAGE_NOTIFICATION)
                        .build());
        manager.createNotificationChannel(channel);
    }

    private static String notificationTitle(RemoteMessage message) {
        RemoteMessage.Notification sent = message.getNotification();
        return sent == null ? "" : pick(sent.getTitle(), "");
    }

    private static String notificationBody(RemoteMessage message) {
        RemoteMessage.Notification sent = message.getNotification();
        return sent == null ? "" : pick(sent.getBody(), "");
    }

    private static String pick(String value, String fallback) {
        return TextUtils.isEmpty(value) ? fallback : value;
    }
}
