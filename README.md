# admin

```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o bin/admin ./cmd/ `
&& ssh root@192.168.71.206 "supervisorctl stop all" `
&& scp ./bin/admin root@192.168.71.206:/opt/admin/admin `
&& scp ./settings.prod.toml root@192.168.71.206:/opt/admin/settings.toml `
&& ssh root@192.168.71.206 "supervisorctl start all"
```
