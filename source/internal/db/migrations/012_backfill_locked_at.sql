UPDATE stages SET locked_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE locked = 1 AND locked_at IS NULL;
