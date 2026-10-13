param([Parameter(Mandatory=$true)][string]$StaffToken)
$ErrorActionPreference = 'Stop'
$base = 'https://calbotbackend-production.up.railway.app'
$source = 'C:\Users\programador_cal\Desktop\scrapping-cal\menu_bot_cal'
$audit = Join-Path $PSScriptRoot '../import-audit'
$null = New-Item -ItemType Directory -Force -Path $audit
function Api([string]$method, [string]$route, $body = $null) {
    $argsList = @('--noproxy','*','-sS','--max-time','30','-X',$method,'-H',"Authorization: Bearer $StaffToken",'-w',"`n%{http_code}")
    $temp = $null
    try {
        if ($null -ne $body) {
            $temp = Join-Path $audit 'request.tmp.json'
            [IO.File]::WriteAllText($temp, ($body | ConvertTo-Json -Depth 30 -Compress), [Text.UTF8Encoding]::new($false))
            $argsList += @('-H','Content-Type: application/json','--data-binary',"@$temp")
        }
        $raw = & curl.exe @argsList "$base$route"
        if ($LASTEXITCODE -ne 0) { throw "Connection failed: $method $route. Inspect staff state before retrying." }
        $lines = @($raw)
        $code = [int]$lines[-1]
        $content = ($lines | Select-Object -SkipLast 1) -join "`n"
        if ($code -lt 200 -or $code -ge 300) { throw "HTTP $code $method ${route}: $content" }
        if ($content.Trim()) { return ($content | ConvertFrom-Json) }
    } finally { if ($temp -and (Test-Path $temp)) { Remove-Item -LiteralPath $temp } }
}
function ReadJson($name) { Get-Content -LiteralPath (Join-Path $source $name) -Raw -Encoding UTF8 | ConvertFrom-Json }
$roots = ReadJson 'menu_bot_propuesto.json'
$children = ReadJson 'menu_subopciones.json'
$groups = ReadJson 'response_groups.json'
$prompts = ((Get-Content -LiteralPath (Join-Path $source 'prompt_parent.json') -Encoding UTF8 | Where-Object { $_ -notmatch '^\s*//' }) -join "`n") | ConvertFrom-Json
$existing = @(Api 'GET' '/api/v1/staff/menu-options')
$existing | ConvertTo-Json -Depth 40 | Set-Content (Join-Path $audit ('snapshot-' + (Get-Date -Format 'yyyyMMdd-HHmmss') + '.json')) -Encoding UTF8
$created = [Collections.Generic.List[object]]::new()
if (Test-Path (Join-Path $audit 'created.json')) { foreach ($record in (Get-Content (Join-Path $audit 'created.json') -Raw -Encoding UTF8 | ConvertFrom-Json)) { $created.Add($record) } }
function Record($kind,$id,$title) {
    $created.Add([pscustomobject]@{kind=$kind;id=$id;title=$title})
    $created | ConvertTo-Json -Depth 10 | Set-Content (Join-Path $audit 'created.json') -Encoding UTF8
    Write-Output "Created $kind $id $title"
}
$rootIds = @{}
$leafIds = @{}
foreach ($root in $roots) {
    $match = @($existing | Where-Object { $null -eq $_.parent_option_id -and $_.title -eq $root.title })
    if ($match.Count -gt 1) { throw "Duplicate root: $($root.title)" }
    if ($match.Count) {
        if (!$match[0].is_active -or $match[0].response_groups.Count) { throw "Conflicting root: $($root.title)" }
        $id = $match[0].id
    } else {
        $new = Api 'POST' '/api/v1/staff/menu-options' $root
        $id = $new.id
        Record 'option' $id $root.title
    }
    $rootIds[$root.title] = $id
    foreach ($leaf in $children.($root.title)) {
        $match = @($existing | Where-Object { $_.parent_option_id -eq $id -and $_.title -eq $leaf.title })
        if ($match.Count -gt 1) { throw "Duplicate leaf: $($leaf.title)" }
        if ($match.Count) {
            if (!$match[0].is_active -or $match[0].prompts.Count) { throw "Conflicting leaf: $($leaf.title)" }
            $leafId = $match[0].id
        } else {
            $leaf.parent_option_id = $id
            $new = Api 'POST' '/api/v1/staff/menu-options' $leaf
            $leafId = $new.id
            Record 'option' $leafId $leaf.title
        }
        $leafIds[$leaf.title] = $leafId
    }
}
for ($i = 0; $i -lt $prompts.Count; $i++) {
    $prompt = $prompts[$i]
    $prompt.menu_option_id = $rootIds[$roots[[int][Math]::Floor($i / 3)].title]
    $match = @($existing | Where-Object { $_.id -eq $prompt.menu_option_id })
    $same = @($match | ForEach-Object { $_.prompts } | Where-Object { $_.message -eq $prompt.message -and $_.is_active })
    if (!$same.Count) { $new = Api 'POST' '/api/v1/staff/prompts' $prompt; Record 'prompt' $new.id $prompt.message }
}
foreach ($group in $groups) {
    $title = $group.name -replace ' \u2014 alternativa$', ''
    $group.menu_option_id = $leafIds[$title]
    if (!$group.menu_option_id) { throw "Unmapped group: $title" }
    foreach ($item in $group.items) {
        if ($item.caption) { $item.caption = $item.caption.Replace('publicaci?n','publicación').Replace('Publicaci?n','Publicación') }
    }
    # Complete the existing guidance using the archived official pages; keep menu and media selection intact.
    if ($title -eq 'Mesa de partes virtual') {
        $group.items[0].text = 'Puedes presentar hasta 5 documentos en PDF o JPG, de hasta 5 MB cada uno. La plataforma recibe envíos las 24 horas; la atención administrativa indicada es de lunes a viernes de 08:00 a 17:00. Los envíos fuera de horario se registran a las 08:00 del siguiente día hábil. Confirma estas indicaciones en el formulario oficial.'
    }
    if ($title -eq 'Devolución de pagos errados') {
        $group.items[0].text = 'Para solicitar una devolución por pago errado, prepara tu comprobante en PDF o JPG, DNI, registro CAL, teléfono, correo, motivo del reclamo, banco, número de cuenta y CCI de 20 dígitos. Para cambios de boleta o factura, el formulario también solicita RUC y número de factura. Revisa y completa el formulario oficial.'
    }
    if ($title -eq 'Correos electrónicos') {
        $group.items[0].text = 'Central: centraltelefonica@colegiodeabogadosdelima.pe\nIncorporaciones: sec_vicedecanato@colegiodeabogadosdelima.pe\nHabilitaciones: habilitaciones@colegiodeabogadosdelima.pe\nPago en línea: pagoenlinea@colegiodeabogadosdelima.pe\nCECAL: cecal@colegiodeabogadosdelima.pe\nConsulta el directorio oficial para otras oficinas y posibles actualizaciones.'
    }
    $match = @($existing | Where-Object { $_.id -eq $group.menu_option_id })
    $same = @($match | ForEach-Object { $_.response_groups } | Where-Object { $_.name -eq $group.name })
    if ($same.Count) {
        if ($same.Count -ne 1 -or !$same[0].is_active -or $same[0].items.Count -ne $group.items.Count) { throw "Conflicting response group: $($group.name)" }
        Write-Output "Already present: $($group.name)"
        continue
    }
    $new = Api 'POST' '/api/v1/staff/response-groups' $group
    Record 'response_group' $new.id $group.name
}
$after = @(Api 'GET' '/api/v1/staff/menu-options')
$after | ConvertTo-Json -Depth 40 | Set-Content (Join-Path $audit 'after.json') -Encoding UTF8
$publicRoots = @(Api 'GET' '/public/menu-options')
foreach ($root in $roots) {
    $id = $rootIds[$root.title]
    if (!@($publicRoots | Where-Object { $_.id -eq $id }).Count) { throw "Root not public: $($root.title)" }
    $branch = Api 'GET' "/public/menu-options/$id/sub-options"
    foreach ($leaf in $children.($root.title)) {
        $leafId = $leafIds[$leaf.title]
        if (!@($branch.options | Where-Object { $_.id -eq $leafId }).Count) { throw "Leaf not public: $($leaf.title)" }
        $response = Api 'GET' "/public/menu-options/$leafId/responses"
        if (!$response.items.Count) { throw "Empty response: $($leaf.title)" }
    }
}
Write-Output "VERIFIED: $($roots.Count) roots, $($leafIds.Count) leaves; $($created.Count) created records."
