DELETE FROM system_events
WHERE kind = 'http_5xx'
  AND (message LIKE 'This part of the portal is switched off%'
       OR message LIKE 'Attendance can only be marked from the Android app%');
