package edu.sust.cse.portal;

import java.nio.charset.StandardCharsets;

import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;

public final class AttendanceTokens {

    public static final String SERVICE_UUID = "0000a7a7-0000-1000-8000-00805f9b34fb";

    public static final int TOKEN_BYTES = 8;

    public static final long TOKEN_SLICE_SECONDS = 10L;

    public static final long CODE_SLICE_SECONDS = 20L;

    private static final String ALGORITHM = "HmacSHA256";

    private static final char[] HEX = "0123456789abcdef".toCharArray();

    private AttendanceTokens() {
    }

    public static long slice(long epochSeconds, long seconds) {
        return seconds <= 0 ? 0 : epochSeconds / seconds;
    }

    public static byte[] token(String key, long sessionId, long slice) {
        byte[] sum = digest(key, sessionId, slice);
        byte[] token = new byte[TOKEN_BYTES];
        System.arraycopy(sum, 0, token, 0, TOKEN_BYTES);
        return token;
    }

    public static String hex(byte[] raw) {
        char[] out = new char[raw.length * 2];
        for (int index = 0; index < raw.length; index++) {
            int value = raw[index] & 0xFF;
            out[index * 2] = HEX[value >>> 4];
            out[index * 2 + 1] = HEX[value & 0x0F];
        }
        return new String(out);
    }

    private static byte[] digest(String key, long sessionId, long slice) {
        try {
            Mac mac = Mac.getInstance(ALGORITHM);
            mac.init(new SecretKeySpec(key.getBytes(StandardCharsets.UTF_8), ALGORITHM));
            String message = sessionId + ":" + slice;
            return mac.doFinal(message.getBytes(StandardCharsets.UTF_8));
        } catch (Exception failure) {
            return new byte[32];
        }
    }
}
