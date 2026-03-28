param(
  [Parameter(Mandatory = $true)]
  [string[]]$Paths,
  [string[]]$Include = @('*.vue','*.js','*.ts','*.json','*.css','*.scss','*.html','*.go')
)

$utf8Strict = [System.Text.UTF8Encoding]::new($false, $true)
$replacement = [char]0xFFFD

$invalid = New-Object System.Collections.Generic.List[string]
$replacementFiles = New-Object System.Collections.Generic.List[string]

foreach ($path in $Paths) {
  if (-not (Test-Path -Path $path)) {
    Write-Warning "Path not found: $path"
    continue
  }
  $files = Get-ChildItem -Path $path -Recurse -File -Include $Include
  foreach ($file in $files) {
    $bytes = [System.IO.File]::ReadAllBytes($file.FullName)
    try {
      $text = $utf8Strict.GetString($bytes)
    } catch {
      $invalid.Add($file.FullName)
      continue
    }
    if ($text.IndexOf($replacement) -ge 0) {
      $replacementFiles.Add($file.FullName)
    }
  }
}

if ($invalid.Count -eq 0 -and $replacementFiles.Count -eq 0) {
  Write-Output 'OK: All checked files are valid UTF-8 and contain no replacement characters.'
  exit 0
}

if ($invalid.Count -gt 0) {
  Write-Output 'Invalid UTF-8 files:'
  $invalid | Sort-Object -Unique | ForEach-Object { Write-Output "- $_" }
}

if ($replacementFiles.Count -gt 0) {
  Write-Output 'Files containing replacement character (U+FFFD):'
  $replacementFiles | Sort-Object -Unique | ForEach-Object { Write-Output "- $_" }
}

exit 1
