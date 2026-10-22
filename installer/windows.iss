; 甜橙短剧 Windows 一键安装包脚本（Inno Setup）
#define MyAppName "甜橙短剧"
#define MyAppVersion "0.2.76+84"
#define MyAppExeName "zhenguojian.exe"

[Setup]
AppId={{A1B2C3D4-1234-5678-9ABC-DEF012345678}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher=甜橙短剧
DefaultDirName={autopf}\甜橙短剧
DefaultGroupName=甜橙短剧
UninstallDisplayIcon={app}\{#MyAppExeName}
OutputDir=..\dist\windows\installer
OutputBaseFilename=甜橙短剧-安装版-{#MyAppVersion}
Compression=lzma2
SolidCompression=yes
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
WizardStyle=modern
DisableProgramGroupPage=yes

[Files]
Source: "..\dist\windows\app\*"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs createallsubdirs

[Icons]
Name: "{autoprograms}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"

[Run]
Filename: "{app}\{#MyAppExeName}"; Description: "立即运行 {#MyAppName}"; Flags: nowait postinstall skipifsilent
