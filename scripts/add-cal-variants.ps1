param([Parameter(Mandatory=$true)][string]$StaffToken)
$ErrorActionPreference = 'Stop'
$definitions = Get-Content (Join-Path $PSScriptRoot 'import-cal-menu.ps1') -Raw -Encoding UTF8
$prefix = $definitions.Substring(0, $definitions.IndexOf("`$roots = ReadJson"))
$callerScriptRoot = $PSScriptRoot
$prefix = $prefix.Replace('$PSScriptRoot', '$callerScriptRoot')
# Reuse the API helper without running the original import.
. ([scriptblock]::Create($prefix)) -StaffToken $StaffToken
$variants = Get-Content (Join-Path $PSScriptRoot 'cal-variants.json') -Raw -Encoding UTF8 | ConvertFrom-Json
$state = @(Api 'GET' '/api/v1/staff/menu-options')
$state | ConvertTo-Json -Depth 40 | Set-Content (Join-Path $audit 'before-variants.json') -Encoding UTF8
$log = [Collections.Generic.List[object]]::new()
function LogVariant($kind,$id,$optionId) {
    $log.Add([pscustomobject]@{kind=$kind;id=$id;menu_option_id=$optionId})
    $log | ConvertTo-Json -Depth 10 | Set-Content (Join-Path $audit 'variants-created.json') -Encoding UTF8
    Write-Output "Created $kind $id for $optionId"
}
foreach ($entry in $variants.prompts.PSObject.Properties) {
    $id = [int]$entry.Name
    $option = $state | Where-Object id -eq $id
    if (!$option -or !$option.is_active) { throw "Missing active option $id" }
    foreach ($message in $entry.Value) {
        if (@($option.prompts | Where-Object message -eq $message).Count) { continue }
        $new = Api 'POST' '/api/v1/staff/prompts' @{menu_option_id=$id;message=$message;weight=1}
        LogVariant 'prompt' $new.id $id
    }
}
foreach ($entry in $variants.responses.PSObject.Properties) {
    $id = [int]$entry.Name
    $option = $state | Where-Object id -eq $id
    if (!$option -or !$option.is_active) { throw "Missing active option $id" }
    $name = $option.title + ' - variante conversacional'
    if (@($option.response_groups | Where-Object name -eq $name).Count) { continue }
    $original = @($option.response_groups | Where-Object { $_.is_active } | Sort-Object id)[0]
    if (!$original) { throw "No original response for $id" }
    $items = @($original.items | Sort-Object sort_order | ForEach-Object {
        $item = @{type=$_.type;sort_order=$_.sort_order}
        foreach ($key in @('text','url','caption','metadata')) { if ($null -ne $_.$key) { $item[$key] = $_.$key } }
        if ($_.type -eq 'TEXT') { $item.text = $entry.Value }
        $item
    })
    $new = Api 'POST' '/api/v1/staff/response-groups' @{menu_option_id=$id;name=$name;weight=1;items=$items}
    LogVariant 'response_group' $new.id $id
}
$after = @(Api 'GET' '/api/v1/staff/menu-options')
$after | ConvertTo-Json -Depth 40 | Set-Content (Join-Path $audit 'after-variants.json') -Encoding UTF8
$observations = [Collections.Generic.List[object]]::new()
foreach ($entry in $variants.prompts.PSObject.Properties) {
    $id = [int]$entry.Name
    $seen = @{}
    for ($n=0; $n -lt 20; $n++) {
        $response = Api 'GET' "/public/menu-options/$id/sub-options"
        $key = [string]$response.prompt.id
        if (!$response.prompt.message) { throw "Missing prompt for $id" }
        $seen[$key]++
    }
    $observations.Add([pscustomobject]@{kind='prompt';menu_option_id=$id;requests=20;frequencies=$seen})
    Write-Output "Prompt $id distinct=$($seen.Count)/20"
}
foreach ($entry in $variants.responses.PSObject.Properties) {
    $id = [int]$entry.Name
    $seen = @{}
    for ($n=0; $n -lt 12; $n++) {
        $response = Api 'GET' "/public/menu-options/$id/responses"
        if (!$response.items.Count) { throw "Missing response for $id" }
        $key = [string]$response.id
        $seen[$key]++
    }
    $observations.Add([pscustomobject]@{kind='response';menu_option_id=$id;requests=12;frequencies=$seen})
    $observations | ConvertTo-Json -Depth 15 | Set-Content (Join-Path $audit 'randomness-observations.json') -Encoding UTF8
    Write-Output "Response $id distinct=$($seen.Count)/12"
}
Write-Output "Completed: $($log.Count) new variants."
