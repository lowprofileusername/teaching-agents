$ErrorActionPreference = 'Stop'
$fixture = Join-Path $env:RUNNER_TEMP 'teaching-fixture.sqlite3'
python tests/make_fixture.py $fixture
if ($LASTEXITCODE -ne 0) { throw 'Fixture creation failed' }
$env:TEACHING_LIBRARY_DB = $fixture
$index = 0
foreach ($exe in Get-ChildItem dist/*.exe) {
  $index++
  $out = Join-Path $env:RUNNER_TEMP "agent-$index.out"
  $err = Join-Path $env:RUNNER_TEMP "agent-$index.err"
  $process = Start-Process -FilePath $exe.FullName -ArgumentList '-no-browser' -PassThru -RedirectStandardOutput $out -RedirectStandardError $err
  try {
    $link = ''
    for ($i=0; $i -lt 40; $i++) {
      Start-Sleep -Milliseconds 500
      if (Test-Path $out) { $match = [regex]::Match([string](Get-Content $out -Raw), 'http://127\.0\.0\.1:\d+/#([a-f0-9]+)'); if ($match.Success) { $link=$match.Value; break } }
      if ($process.HasExited) { throw (Get-Content $err -Raw) }
    }
    if (-not $link) { throw 'Executable startup timed out' }
    $parts=$link.Split('#');$base=$parts[0].TrimEnd('/');$headers=@{'X-Session-Token'=$parts[1]}
    $lib=Invoke-RestMethod "$base/api/library" -Method Post -Headers $headers -ContentType 'application/json' -Body '{}'
    if ($lib.counts.video -ne 1 -or -not $lib.read_only) { throw 'Read-only SQLite connection failed' }
    $res=Invoke-RestMethod "$base/api/search" -Method Post -Headers $headers -ContentType 'application/json' -Body '{"tag":"synthetic","page_no":1}'
    if ($res.pages[0].id -ne 'synthetic:R01') { throw 'Page retrieval failed' }
    $ui=Invoke-WebRequest "$base/"
    if ($ui.StatusCode -ne 200 -or -not $ui.Content.Contains('scope')) { throw 'Embedded UI missing' }
    foreach ($module in @('app.mjs','pet.mjs','teacher-animation.mjs','walk-calibration.mjs','lib/pdf.mjs')) {
      $script=Invoke-WebRequest "$base/$module"
      if ($script.StatusCode -ne 200 -or $script.Headers['Content-Type'] -notmatch '^(text|application)/javascript') { throw "Module MIME incorrect: $module" }
    }
    Write-Output "PASS $($exe.Name): launch, private session, read-only SQLite, page retrieval, embedded UI"
  } finally { if (-not $process.HasExited) { Stop-Process -Id $process.Id -Force } }
}
