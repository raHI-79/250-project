# Android build files

`google-services.json` belongs here. It is the public Firebase client config,
not a secret, so it is committed. The build copies it into
`android/app/google-services.json` before Gradle runs.

The signing keystore and the Firebase **service account** JSON never live here;
they are GitHub and Render secrets. See `docs/ANDROID.md`.
