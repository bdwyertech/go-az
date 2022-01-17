# cookiejar

```
import "github.com/orirawlings/persistent-cookiejar"
```

[Package cookiejar](https://pkg.go.dev/github.com/orirawlings/persistent-cookiejar) implements an in-memory RFC 6265-compliant http.CookieJar.

This implementation is a fork of net/http/cookiejar which also implements
methods for dumping the cookies to persistent storage and retrieving them.