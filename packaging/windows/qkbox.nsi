!ifndef ROOT_DIR
!error "ROOT_DIR is required"
!endif

!ifndef OUT_DIR
!error "OUT_DIR is required"
!endif

!ifndef APP_VERSION
!define APP_VERSION "0.1.0"
!endif

!define APP_NAME "qkbox"
!define PUBLISHER "qkbox contributors"

SetCompressor /SOLID lzma
Name "${APP_NAME}"
OutFile "${OUT_DIR}\qkbox-${APP_VERSION}-setup.exe"
InstallDir "$LOCALAPPDATA\Programs\qkbox"
InstallDirRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\qkbox" "InstallLocation"
RequestExecutionLevel highest
ShowInstDetails show
ShowUninstDetails show

!include "MUI2.nsh"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Section "qkbox"
  SetOutPath "$INSTDIR"
  File "${ROOT_DIR}\bin\qkbox.exe"
  File "${ROOT_DIR}\bin\qkbox-window.exe"
  File "${ROOT_DIR}\bin\qkbox-provider.exe"

  CreateDirectory "$SMPROGRAMS\qkbox"
  CreateShortcut "$SMPROGRAMS\qkbox\qkbox.lnk" "$INSTDIR\qkbox.exe"

  WriteUninstaller "$INSTDIR\Uninstall.exe"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\qkbox" "DisplayName" "qkbox"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\qkbox" "DisplayVersion" "${APP_VERSION}"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\qkbox" "Publisher" "${PUBLISHER}"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\qkbox" "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\qkbox" "UninstallString" "$INSTDIR\Uninstall.exe"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\qkbox" "DisplayIcon" "$INSTDIR\qkbox.exe"
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\qkbox" "NoModify" 1
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\qkbox" "NoRepair" 1
SectionEnd

Section "Uninstall"
  Delete "$INSTDIR\qkbox.exe"
  Delete "$INSTDIR\qkbox-window.exe"
  Delete "$INSTDIR\qkbox-provider.exe"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir "$INSTDIR"

  Delete "$SMPROGRAMS\qkbox\qkbox.lnk"
  RMDir "$SMPROGRAMS\qkbox"

  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\qkbox"
SectionEnd
