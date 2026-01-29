# verify_flush.ps1
Write-Host "Starting Data Population..."

# Write 120 keys to trigger flush (Threshold is 100)
for ($i=1; $i -le 120; $i++) {
    ./bin/client.exe -addr localhost:50051 -op put -key "key$i" -val "val$i" | Out-Null
    if ($i % 10 -eq 0) { Write-Host "Written $i keys..." }
}

Write-Host "Writes Complete. Waiting for SSTable Flush..."
Start-Sleep -Seconds 2

# Check if SST files exist
$sstFiles = Get-ChildItem -Filter "*.sst"
if ($sstFiles.Count -gt 0) {
    Write-Host "SUCCESS: Found $($sstFiles.Count) SSTable files on disk."
} else {
    Write-Host "FAILURE: No SSTable files found."
}

# Verify Read from SSTable (key 5 should be in SSTable now, key 115 in MemTable)
Write-Host "Verifying Read of key5..."
./bin/client.exe -addr localhost:50051 -op get -key "key5"

Write-Host "Waiting 65 seconds for Compaction (Interval 1m)..."
Start-Sleep -Seconds 65

# Check if files merged
$newSstFiles = Get-ChildItem -Filter "*.sst"
Write-Host "Current SSTables: $($newSstFiles.Count)"
if ($newSstFiles.Count -lt $sstFiles.Count -or $newSstFiles.Count -eq 1) {
     Write-Host "SUCCESS: Compaction likely occurred!"
}
