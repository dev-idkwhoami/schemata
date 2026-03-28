!include "MUI2.nsh"

Name "Schemata"
OutFile "build\bin\schemata-setup.exe"
InstallDir "$LOCALAPPDATA\Schemata"
RequestExecutionLevel user

!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Section "Install"
  SetOutPath "$INSTDIR"
  File "build\bin\schemata.exe"
  File "build\bin\schemata-mcp.exe"

  ; Start Menu shortcut
  CreateDirectory "$SMPROGRAMS\Schemata"
  CreateShortcut "$SMPROGRAMS\Schemata\Schemata.lnk" "$INSTDIR\schemata.exe"
  CreateShortcut "$SMPROGRAMS\Schemata\Uninstall.lnk" "$INSTDIR\uninstall.exe"

  ; Uninstaller
  WriteUninstaller "$INSTDIR\uninstall.exe"
SectionEnd

Section "Uninstall"
  Delete "$INSTDIR\schemata.exe"
  Delete "$INSTDIR\schemata-mcp.exe"
  Delete "$INSTDIR\uninstall.exe"
  RMDir "$INSTDIR"
  Delete "$SMPROGRAMS\Schemata\Schemata.lnk"
  Delete "$SMPROGRAMS\Schemata\Uninstall.lnk"
  RMDir "$SMPROGRAMS\Schemata"
SectionEnd
