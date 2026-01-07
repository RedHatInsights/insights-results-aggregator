---
layout: page
nav_order: 9
---
# pprof

In debug mode, standard Golang pprof interface is available at `/debug/pprof/`

Common usage (for using pprof against local instance):

```shell
go tool pprof localhost:8080/debug/pprof/profile
```

A practical example is available here:
<https://medium.com/@paulborile/profiling-a-golang-rest-api-server-635fa0ed45f3>
