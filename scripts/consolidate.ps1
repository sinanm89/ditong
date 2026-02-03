# Consolidate dictionary output to single files
# Usage: .\scripts\consolidate.ps1 -InputDir test_output_full -OutputDir consolidated

param(
    [string]$InputDir = "test_output_full",
    [string]$OutputDir = "consolidated"
)

# Create output directory
New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null

# Process each word length
foreach ($length in 3..10) {
    $type = "$length-c"
    $words = @{}
    
    # Collect from all language directories
    Get-ChildItem -Path $InputDir -Recurse -Filter "$type.json" | ForEach-Object {
        $json = Get-Content $_.FullName | ConvertFrom-Json
        if ($json.words) {
            $json.words.PSObject.Properties | ForEach-Object {
                if (-not $words.ContainsKey($_.Name)) {
                    $words[$_.Name] = $_.Value
                }
            }
        }
    }
    
    if ($words.Count -gt 0) {
        # JSON output
        $output = @{
            type = $type
            count = $words.Count
            words = $words.Keys | Sort-Object
        }
        $output | ConvertTo-Json -Depth 1 | Out-File "$OutputDir/$type.json" -Encoding UTF8
        
        # CSV output (just words)
        $words.Keys | Sort-Object | Out-File "$OutputDir/$type.csv" -Encoding UTF8
        
        Write-Host "$type : $($words.Count) unique words"
    }
}

# Create master file with all words
$allWords = @()
Get-ChildItem -Path $OutputDir -Filter "*.csv" | ForEach-Object {
    $allWords += Get-Content $_.FullName
}
$allWords | Sort-Object -Unique | Out-File "$OutputDir/all_words.csv" -Encoding UTF8

$total = (Get-Content "$OutputDir/all_words.csv").Count
Write-Host "`nTotal unique words: $total"
Write-Host "Output: $OutputDir/"
