Intermittent tests
===

These tests are known to fail intermittently.

```
--- FAIL: TestHashCacheAddContainsHashes (0.00s)
    hashcache_test.go:102: txid a323a17cc0b607ed83dcabf5e952aaa8b8a910d33c24d47606461eef9868c01e wasn't inserted into cache but was found
```

```
=== RUN TestRetryPermanent
--- FAIL: TestRetryPermanent (0.00s)
connmanager_test.go:273: retry: 127.0.0.1:18555 - want state 0, got state 3
```
