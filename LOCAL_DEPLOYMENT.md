# Windows local deployment

CoAI is built from the current source tree by `docker-compose.local.yaml`. A
Sub2API process running on Windows must be configured in the CoAI channel as:

```text
http://host.docker.internal:8080
```

Do not use `localhost` from inside the CoAI container and do not append `/v1`.

Before starting, create a local secret of at least 32 characters (do not commit
it):

```powershell
$env:SECRET = -join ((1..48) | ForEach-Object { [char](Get-Random -Minimum 33 -Maximum 127) })
.\start-local.ps1
```

The compose configuration makes new MySQL databases use `utf8mb4` and
`utf8mb4_unicode_ci`. To migrate an existing local database without deleting
data, back it up and convert it in place:

```powershell
docker exec db mysqldump -uroot -proot --single-transaction --routines --triggers chatnio | Set-Content -Encoding utf8 chatnio-before-utf8mb4.sql
docker exec db mysql -uroot -proot -e "ALTER DATABASE chatnio CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
docker exec db mysql -uroot -proot -D chatnio -e "ALTER TABLE conversation CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
```

Verify Sub2API directly from Windows without exposing the API key:

```powershell
$env:SUB2API_KEY = Read-Host "Sub2API key"
curl.exe -X POST "http://localhost:8080/v1/images/generations" `
  -H "Authorization: Bearer $env:SUB2API_KEY" `
  -H "Content-Type: application/json" `
  -d '{"model":"gpt-image-2","prompt":"generate a cute cat","size":"1024x1024","n":1}'
Remove-Item Env:SUB2API_KEY
```
