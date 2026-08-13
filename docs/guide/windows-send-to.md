# Windows "Send to Plik"

Upload files to Plik directly from the Windows Explorer right-click menu.

## 1. Install the CLI

Download `plik-<VERSION>-windows-amd64.exe` from the [releases page](https://github.com/root-gg/plik/releases) (or from your Plik server's web UI), rename it to `plik.exe`, and place it in a permanent location:

**CMD (Command Prompt):**
```cmd
mkdir "%LOCALAPPDATA%\Plik"
move plik.exe "%LOCALAPPDATA%\Plik\plik.exe"
```

**PowerShell:**
```powershell
mkdir "$env:LOCALAPPDATA\Plik" -Force
Move-Item .\plik.exe "$env:LOCALAPPDATA\Plik\plik.exe"
```

This puts it under `C:\Users\<you>\AppData\Local\Plik\` — no admin rights required.

::: warning
Don't mix syntaxes! CMD uses `%LOCALAPPDATA%`, PowerShell uses `$env:LOCALAPPDATA`. Using the wrong one creates a literal folder named `%LOCALAPPDATA%`.
:::

Optionally, add `%LOCALAPPDATA%\Plik` to your user `PATH` so you can call `plik` from any terminal:

**Via Settings (GUI):** Open **Settings → System → About → Advanced system settings** (or press Win+R, type `sysdm.cpl`), click **Environment Variables…**, under **User variables** select `Path`, click **Edit…**, click **New**, add `%LOCALAPPDATA%\Plik`, and click **OK** on all dialogs.

**Via PowerShell:**
```powershell
[Environment]::SetEnvironmentVariable("Path", "$env:LOCALAPPDATA\Plik;" + [Environment]::GetEnvironmentVariable("Path", "User"), "User")
```

Open a **new terminal** for the change to take effect.

## 2. First run

Just run `plik` — it will guide you through the initial setup:

```
PS> plik
Please enter your plik domain [default:http://127.0.0.1:8080] :
https://plik.example.com

Authentication is required on this server.
Would you like to authenticate with your browser? [Y/n]
  Open this URL in your browser to authenticate:

    https://plik.example.com/#/cli-auth?code=XXXX-XXXX&hostname=my-pc

  Your one-time code: XXXX-XXXX

  Waiting for authentication...

  ✓ Authenticated! Token saved to ~/.plikrc
  Token: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx

Do you want to enable client auto update ? [Y/n]

Plik client settings successfully saved to C:\Users\<you>/.plikrc
```

The CLI will:
1. Prompt for the **server URL** and save it to `%USERPROFILE%\.plikrc`
2. **Automatically trigger login** if authentication is forced on the server
3. Ask about **auto-update** preference

::: tip
If the server has authentication **enabled** (but not forced), you can authenticate later with `plik --login`.
:::

## 3. Create the upload script

Save the following as **`plik-upload.cmd`**:

```cmd
@echo off
REM ── Send to Plik ──
REM Uploads the selected file(s) via the Plik CLI.
REM Adjust the path below if plik.exe is in a different location.

set PLIK=%LOCALAPPDATA%\Plik\plik.exe

if "%~1"=="" (
    echo No files selected.
    pause
    exit /b 1
)

echo Uploading to Plik...
echo.

"%PLIK%" %*

echo.
echo Press any key to close...
pause >nul
```

This passes all selected files (`%*`) to `plik.exe`, displays the resulting download links, then waits for a keypress so you can copy the URLs before the window closes.

You can customize the plik invocation with flags, for example:
```cmd
"%PLIK%" --oneshot --ttl 24h %*
```

## 4. Install into the Send To folder

1. Press **Win + R**, type `shell:sendto`, press Enter.
2. Copy (or move) `plik-upload.cmd` into that folder.

You can also place a **shortcut** to the script there instead of the script itself — both work.

## 5. Use it!

1. In Windows Explorer, select one or more files.
2. Right-click → **Send to** → **plik-upload**.
3. A console window opens, uploads the files, and prints the download links.
4. Copy the links before pressing any key to close.

::: tip Windows 11 Note
Windows 11 introduced a simplified right-click menu that hides **Send to** behind **Show more options**. This adds an extra click every time you want to upload.

To restore the classic full context menu (with Send to directly visible), run in an elevated terminal:
```cmd
reg add "HKCU\Software\Classes\CLSID\{86ca1aa0-34aa-4e8b-a509-50c905bae2a2}\InprocServer32" /f /ve
taskkill /f /im explorer.exe & start explorer.exe
```

To revert back to the Windows 11 modern menu:
```cmd
reg delete "HKCU\Software\Classes\CLSID\{86ca1aa0-34aa-4e8b-a509-50c905bae2a2}\InprocServer32" /f
taskkill /f /im explorer.exe & start explorer.exe
```
:::
