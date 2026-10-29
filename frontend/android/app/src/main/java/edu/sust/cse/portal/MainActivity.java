package edu.sust.cse.portal;

import android.content.Intent;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.text.TextUtils;

import com.getcapacitor.BridgeActivity;

import org.json.JSONObject;

public class MainActivity extends BridgeActivity {

    private static final String OPEN_EVENT = "sustcse:open";

    private static final long[] DELIVER_AFTER = {400L, 1800L};

    private final Handler handler = new Handler(Looper.getMainLooper());

    @Override
    public void onCreate(Bundle savedInstanceState) {
        registerPlugin(ApkInstaller.class);
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
