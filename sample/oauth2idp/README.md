# OAuth2 provider with ucon

```
$ ./run.sh
open http://localhost:8080/
```

try it on swagger-ui!

This example uses [github.com/RangelReale/osin](https://github.com/RangelReale/osin).
However, anything can be used as long as the following requirements are satisfied.

* It can handle oauth2 authentication.
* It can get scopes from Authorization header with swagger.CheckSecurityRequirements middleware.
