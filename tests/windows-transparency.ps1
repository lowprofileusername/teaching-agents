$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
Add-Type @'
using System; using System.Text; using System.Runtime.InteropServices;
public class TeacherWindowProbe {
 public delegate bool EnumProc(IntPtr h,IntPtr p);
 [DllImport("user32.dll")] public static extern bool EnumWindows(EnumProc cb,IntPtr p);
 [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr h,out uint pid);
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] public static extern int GetWindowText(IntPtr h,StringBuilder s,int n);
 [DllImport("user32.dll")] public static extern bool SetWindowPos(IntPtr h,IntPtr after,int x,int y,int w,int hgt,uint flags);
 [DllImport("user32.dll")] public static extern bool SetCursorPos(int x,int y);
 [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h,out Rect r);
 public struct Rect { public int Left,Top,Right,Bottom; }
 public static IntPtr Find(uint pid) {IntPtr found=IntPtr.Zero;EnumWindows((h,p)=>{uint owner;GetWindowThreadProcessId(h,out owner);var s=new StringBuilder(256);GetWindowText(h,s,256);if(owner==pid&&s.ToString().StartsWith("桌面教师")){found=h;return false;}return true;},IntPtr.Zero);return found;}
}
'@
$qa=Join-Path $env:RUNNER_TEMP 'teacher-transparency'
New-Item -ItemType Directory -Force $qa | Out-Null
$fixture=Join-Path $qa 'fixture.sqlite3'
python tests/make_fixture.py $fixture
if ($LASTEXITCODE -ne 0) { throw 'Fixture creation failed' }
$env:WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS='--remote-debugging-port=9222'
$env:TEACHER_QA_DIR=$qa
$env:TEACHING_LIBRARY_DB=$fixture
$env:TEACHING_AGENT_PROFILES=Join-Path $qa 'profiles.json'
$checks=@()
foreach ($persona in @('original','male')) {
 $exe=(Resolve-Path "dist/teaching-agent-$persona-windows-amd64.exe").Path
 $log=Join-Path $qa "$persona.log";$err=Join-Path $qa "$persona.err"
 $env:TEACHER_QA_PERSONA=$persona
 $proc=Start-Process $exe -PassThru -RedirectStandardOutput $log -RedirectStandardError $err
 $backdrop=$null
 try {
  $link='';$hwnd=[IntPtr]::Zero
  for($i=0;$i -lt 100;$i++){
   Start-Sleep -Milliseconds 300
   if($proc.HasExited){throw "Native app exited: $(Get-Content $err -Raw)"}
   if(Test-Path $log){$raw=Get-Content $log -Raw;if(-not [string]::IsNullOrWhiteSpace($raw)){$m=[regex]::Match($raw,'http://127\.0\.0\.1:\d+/#([a-f0-9]+)');if($m.Success){$link=$m.Value}}}
   $hwnd=[TeacherWindowProbe]::Find([uint32]$proc.Id)
   if($link -and $hwnd -ne [IntPtr]::Zero){break}
  }
  if(-not $link -or $hwnd -eq [IntPtr]::Zero){throw 'Native teacher window did not appear'}
  $parts=$link.Split('#');$base=$parts[0].TrimEnd('/');$headers=@{'X-Session-Token'=$parts[1]}
  Invoke-RestMethod "$base/api/desktop" -Method Post -Headers $headers -ContentType 'application/json' -Body '{"action":"posture","posture":"sit"}' | Out-Null
  # Uniform coloured window directly beneath the teacher. An opaque WebView
  # cannot pass this test merely because its webpage/canvas alpha is zero.
  $colour=[Drawing.Color]::FromArgb(255,28,67,93)
  $backdrop=New-Object Windows.Forms.Form
  $backdrop.FormBorderStyle='None';$backdrop.StartPosition='Manual';$backdrop.Bounds=New-Object Drawing.Rectangle(40,30,320,600);$backdrop.BackColor=$colour;$backdrop.TopMost=$true;$backdrop.Show()
  [Windows.Forms.Application]::DoEvents()
  [TeacherWindowProbe]::SetWindowPos($hwnd,[IntPtr](-1),60,50,260,540,0x0040) | Out-Null
  [TeacherWindowProbe]::SetCursorPos(800,700) | Out-Null
  $matched=$false;$painted=0
  for($attempt=0;$attempt -lt 40;$attempt++){
   Start-Sleep -Milliseconds 250
   [Windows.Forms.Application]::DoEvents()
   [TeacherWindowProbe]::SetWindowPos($hwnd,[IntPtr](-1),60,50,260,540,0x0040) | Out-Null
   $rect=New-Object TeacherWindowProbe+Rect
   [TeacherWindowProbe]::GetWindowRect($hwnd,[ref]$rect) | Out-Null
   $bmp=New-Object Drawing.Bitmap(260,540);$g=[Drawing.Graphics]::FromImage($bmp)
   $g.CopyFromScreen($rect.Left,$rect.Top,0,0,$bmp.Size);$g.Dispose()
   $points=@(@(3,300),@(256,300),@(3,510),@(256,510),@(10,190),@(250,190))
   $matched=$true
   foreach($xy in $points){$c=$bmp.GetPixel($xy[0],$xy[1]);if([Math]::Abs([int]$c.R-$colour.R)+[Math]::Abs([int]$c.G-$colour.G)+[Math]::Abs([int]$c.B-$colour.B) -gt 15){$matched=$false}}
   $painted=0
   for($y=260;$y -lt 525;$y+=5){for($x=45;$x -lt 215;$x+=5){$c=$bmp.GetPixel($x,$y);if([Math]::Abs([int]$c.R-$colour.R)+[Math]::Abs([int]$c.G-$colour.G)+[Math]::Abs([int]$c.B-$colour.B) -gt 35){$painted++}}}
   $bmp.Save((Join-Path $qa "$persona-transparency.png"),[Drawing.Imaging.ImageFormat]::Png);$bmp.Dispose()
   if($matched -and $painted -gt 100){break}
  }
  node tests/windows-webview-probe.mjs
  $checks+=@{rect=@($rect.Left,$rect.Top,$rect.Right,$rect.Bottom);persona=$persona;corners_show_backdrop=$matched;person_pixels=$painted}
  if(-not $matched -or $painted -le 100){throw "Teacher alpha test failed: backdrop=$matched personSamples=$painted"}
  Write-Output "PASS ${persona}: real native window shows coloured background around visible teacher"
 } finally {if($backdrop){$backdrop.Close();$backdrop.Dispose()};if(-not $proc.HasExited){try{Invoke-RestMethod "$base/api/desktop" -Method Post -Headers $headers -ContentType 'application/json' -Body '{"action":"quit"}' | Out-Null;$proc.WaitForExit(5000) | Out-Null}catch{};if(-not $proc.HasExited){Stop-Process -Id $proc.Id -Force}};$checks | ConvertTo-Json | Set-Content -Encoding utf8 (Join-Path $qa 'checks.json')}
}
