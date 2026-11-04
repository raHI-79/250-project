package edu.sust.cse.portal;

import android.content.Intent;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.text.TextUtils;

import com.getcapacitor.BridgeActivity;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginHandle;

import ee.forgr.capacitor.social.login.GoogleProvider;
import ee.forgr.capacitor.social.login.ModifiedMainActivityForSocialLoginPlugin;
import ee.forgr.capacitor.social.login.SocialLoginPlugin;

import org.json.JSONObject;

public class MainActivity extends BridgeActivity implements ModifiedMainActivityForSocialLoginPlugin {

    private static final String OPEN_EVENT = "sustcse:open";

    private static final String SOCIAL_LOGIN_PLUGIN = "SocialLogin";

    private static final long[] DELIVER_AFTER = {400L, 1800L};

    private final Handler handler = new Handler(Looper.getMainLooper());

    @Override
    public void onCreate(Bundle savedInstanceState) {
        registerPlugin(ApkInstaller.class);
        registerPlugin(AttendanceBeacon.class);
        registerPlugin(AttendanceMark.class);
        super.onCreate(savedInstanceState);
    }

    @Override
    protected void onNewIntent(Intent intent) {
        super.onNewIntent(intent);
        setIntent(intent);
        deliver(intent);
    }

    @Override
    public void onResume() {
        super.onResume();
        deliver(getIntent());
    }

    @Override
    public void onActivityResult(int requestCode, int resultCode, Intent data) {
        super.onActivityResult(requestCode, resultCode, data);
        if (requestCode < GoogleProvider.REQUEST_AUTHORIZE_GOOGLE_MIN
                || requestCode >= GoogleProvider.REQUEST_AUTHORIZE_GOOGLE_MAX) {
            return;
        }
        if (getBridge() == null) {
            return;
        }
        PluginHandle handle = getBridge().getPlugin(SOCIAL_LOGIN_PLUGIN);
        if (handle == null) {
            return;
        }
        Plugin plugin = handle.getInstance();
        if (plugin instanceof SocialLoginPlugin) {
            ((SocialLoginPlugin) plugin).handleGoogleLoginIntent(requestCode, data);
        }
    }

    @Override
    public void IHaveModifiedTheMainActivityForTheUseWithSocialLoginPlugin() {
    }

    private void deliver(Intent intent) {
        if (intent == null) {
            return;
        }
        final String url = intent.getStringExtra(PortalMessagingService.EXTRA_URL);
        if (TextUtils.isEmpty(url)) {
            return;
        }
        intent.removeExtra(PortalMessagingService.EXTRA_URL);
        for (long delay : DELIVER_AFTER) {
            handler.postDelayed(() -> announce(url), delay);
        }
    }

    private void announce(String url) {
        if (getBridge() == null) {
            return;
        }
        JSONObject payload = new JSONObject();
        try {
            payload.put("url", url);
        } catch (Exception failure) {
            return;
        }
        getBridge().triggerWindowJSEvent(OPEN_EVENT, payload.toString());
    }
}
