param(
    [string]$DistDir = "dist"
)

$ErrorActionPreference = "Stop"

Add-Type -AssemblyName System.IO.Compression
Add-Type -AssemblyName System.IO.Compression.FileSystem

function Build-Lambda {
    param(
        [string]$PackagePath,
        [string]$OutputDir,
        [string]$ZipPath
    )

    New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null

    $previousGOOS = $env:GOOS
    $previousGOARCH = $env:GOARCH
    $previousCGO = $env:CGO_ENABLED

    try {
        $env:GOOS = "linux"
        $env:GOARCH = "amd64"
        $env:CGO_ENABLED = "0"
        go build -tags lambda.norpc -o (Join-Path $OutputDir "bootstrap") $PackagePath
    }
    finally {
        $env:GOOS = $previousGOOS
        $env:GOARCH = $previousGOARCH
        $env:CGO_ENABLED = $previousCGO
    }

    if (Test-Path $ZipPath) {
        Remove-Item -LiteralPath $ZipPath -Force
    }

    $zip = [System.IO.Compression.ZipFile]::Open($ZipPath, [System.IO.Compression.ZipArchiveMode]::Create)
    try {
        $entry = $zip.CreateEntry("bootstrap")
        $entry.ExternalAttributes = (0x81ED -shl 16)

        $entryStream = $entry.Open()
        try {
            $fileStream = [System.IO.File]::OpenRead((Join-Path $OutputDir "bootstrap"))
            try {
                $fileStream.CopyTo($entryStream)
            }
            finally {
                $fileStream.Dispose()
            }
        }
        finally {
            $entryStream.Dispose()
        }
    }
    finally {
        $zip.Dispose()
    }
}

$distPath = Join-Path (Get-Location) $DistDir
New-Item -ItemType Directory -Force -Path $distPath | Out-Null

Build-Lambda `
    -PackagePath "./card-onboarding-file-preprocessor" `
    -OutputDir (Join-Path $distPath "card-onboarding-file-preprocessor") `
    -ZipPath (Join-Path $distPath "card-onboarding-file-preprocessor.zip")

Build-Lambda `
    -PackagePath "./card-onboarding-worker" `
    -OutputDir (Join-Path $distPath "card-onboarding-worker") `
    -ZipPath (Join-Path $distPath "card-onboarding-worker.zip")
